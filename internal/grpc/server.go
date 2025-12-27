package grpc

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	executorv1 "github.com/knullci/necrosword/gen/executor/v1"
	"github.com/knullci/necrosword/internal/config"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var startTime = time.Now()

// ExecutorServer implements the gRPC ExecutorService
type ExecutorServer struct {
	executorv1.UnimplementedExecutorServiceServer
	config  *config.ExecutorConfig
	logger  *zap.Logger
	running map[string]*RunningProcess
	mu      sync.RWMutex
}

// RunningProcess tracks a running process
type RunningProcess struct {
	ID        string
	Tool      string
	Args      []string
	Command   *exec.Cmd
	Cancel    context.CancelFunc
	StartedAt time.Time
}

// NewExecutorServer creates a new gRPC executor server
func NewExecutorServer(cfg *config.ExecutorConfig, logger *zap.Logger) *ExecutorServer {
	return &ExecutorServer{
		config:  cfg,
		logger:  logger,
		running: make(map[string]*RunningProcess),
	}
}

// Execute runs a single command and returns the result
func (s *ExecutorServer) Execute(ctx context.Context, req *executorv1.ExecuteRequest) (*executorv1.ExecuteResponse, error) {
	// Validate tool
	if !s.config.IsToolAllowed(req.Tool) {
		return nil, fmt.Errorf("tool '%s' is not allowed. Allowed tools: %v", req.Tool, s.config.AllowedTools)
	}

	// Create timeout context
	timeout := time.Duration(s.config.DefaultTimeout) * time.Second
	if req.TimeoutSeconds > 0 {
		timeout = time.Duration(req.TimeoutSeconds) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build command
	cmd := exec.CommandContext(ctx, req.Tool, req.Args...)

	if req.WorkDir != "" {
		cmd.Dir = req.WorkDir
	}

	if len(req.Env) > 0 {
		cmd.Env = append(cmd.Environ(), req.Env...)
	}

	// Capture output
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	startTime := time.Now()
	processID := uuid.New().String()

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	// Track running process
	runningProc := &RunningProcess{
		ID:        processID,
		Tool:      req.Tool,
		Args:      req.Args,
		Command:   cmd,
		Cancel:    cancel,
		StartedAt: startTime,
	}

	s.mu.Lock()
	s.running[processID] = runningProc
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.running, processID)
		s.mu.Unlock()
	}()

	// Read output
	var wg sync.WaitGroup
	var stdoutBuf, stderrBuf strings.Builder

	wg.Add(2)
	go func() {
		defer wg.Done()
		s.readOutput(stdout, &stdoutBuf)
	}()
	go func() {
		defer wg.Done()
		s.readOutput(stderr, &stderrBuf)
	}()

	wg.Wait()

	// Wait for command
	err = cmd.Wait()
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	response := &executorv1.ExecuteResponse{
		ProcessId:  processID,
		Tool:       req.Tool,
		Args:       req.Args,
		Stdout:     stdoutBuf.String(),
		Stderr:     stderrBuf.String(),
		DurationMs: duration.Milliseconds(),
		StartedAt:  timestamppb.New(startTime),
		EndedAt:    timestamppb.New(endTime),
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			response.ExitCode = -1
			response.Error = "command timed out"
			response.TimedOut = true
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			response.ExitCode = int32(exitErr.ExitCode())
			response.Error = exitErr.Error()
		} else {
			response.ExitCode = -1
			response.Error = err.Error()
		}
		response.Success = false
	} else {
		response.ExitCode = 0
		response.Success = true
	}

	s.logger.Info("command executed",
		zap.String("process_id", processID),
		zap.String("tool", req.Tool),
		zap.Strings("args", req.Args),
		zap.Int32("exit_code", response.ExitCode),
		zap.Duration("duration", duration),
		zap.Bool("success", response.Success),
	)

	return response, nil
}

// ExecuteStream runs a command and streams output in real-time
func (s *ExecutorServer) ExecuteStream(req *executorv1.ExecuteRequest, stream executorv1.ExecutorService_ExecuteStreamServer) error {
	// Validate tool
	if !s.config.IsToolAllowed(req.Tool) {
		return fmt.Errorf("tool '%s' is not allowed. Allowed tools: %v", req.Tool, s.config.AllowedTools)
	}

	ctx := stream.Context()

	// Create timeout context
	timeout := time.Duration(s.config.DefaultTimeout) * time.Second
	if req.TimeoutSeconds > 0 {
		timeout = time.Duration(req.TimeoutSeconds) * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Build command
	cmd := exec.CommandContext(ctx, req.Tool, req.Args...)

	if req.WorkDir != "" {
		cmd.Dir = req.WorkDir
	}

	if len(req.Env) > 0 {
		cmd.Env = append(cmd.Environ(), req.Env...)
	}

	// Create pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start command
	startTime := time.Now()
	processID := uuid.New().String()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %w", err)
	}

	// Track running process
	runningProc := &RunningProcess{
		ID:        processID,
		Tool:      req.Tool,
		Args:      req.Args,
		Command:   cmd,
		Cancel:    cancel,
		StartedAt: startTime,
	}

	s.mu.Lock()
	s.running[processID] = runningProc
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.running, processID)
		s.mu.Unlock()
	}()

	// Stream output
	var wg sync.WaitGroup
	var stdoutBuf, stderrBuf strings.Builder

	wg.Add(2)
	go func() {
		defer wg.Done()
		s.streamOutput(stdout, &stdoutBuf, stream, true)
	}()
	go func() {
		defer wg.Done()
		s.streamOutput(stderr, &stderrBuf, stream, false)
	}()

	wg.Wait()

	// Wait for command
	err = cmd.Wait()
	endTime := time.Now()
	duration := endTime.Sub(startTime)

	response := &executorv1.ExecuteResponse{
		ProcessId:  processID,
		Tool:       req.Tool,
		Args:       req.Args,
		Stdout:     stdoutBuf.String(),
		Stderr:     stderrBuf.String(),
		DurationMs: duration.Milliseconds(),
		StartedAt:  timestamppb.New(startTime),
		EndedAt:    timestamppb.New(endTime),
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			response.ExitCode = -1
			response.Error = "command timed out"
			response.TimedOut = true
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			response.ExitCode = int32(exitErr.ExitCode())
			response.Error = exitErr.Error()
		} else {
			response.ExitCode = -1
			response.Error = err.Error()
		}
		response.Success = false
	} else {
		response.ExitCode = 0
		response.Success = true
	}

	// Send final result
	return stream.Send(&executorv1.ExecuteStreamResponse{
		Output: &executorv1.ExecuteStreamResponse_Result{
			Result: response,
		},
	})
}

// ExecutePipeline runs a multi-step pipeline
func (s *ExecutorServer) ExecutePipeline(ctx context.Context, req *executorv1.PipelineRequest) (*executorv1.PipelineResponse, error) {
	startTime := time.Now()

	pipelineID := req.Id
	if pipelineID == "" {
		pipelineID = uuid.New().String()
	}

	s.logger.Info("starting pipeline",
		zap.String("pipeline_id", pipelineID),
		zap.String("name", req.Name),
		zap.Int("steps", len(req.Steps)),
	)

	// Set overall timeout
	if req.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	response := &executorv1.PipelineResponse{
		PipelineId:  pipelineID,
		Name:        req.Name,
		StartedAt:   timestamppb.New(startTime),
		Success:     true,
		TotalSteps:  int32(len(req.Steps)),
		StepResults: make([]*executorv1.StepResult, 0, len(req.Steps)),
	}

	// Execute each step
	for i, step := range req.Steps {
		// Check context cancellation
		if ctx.Err() != nil {
			response.Success = false
			response.FailedStep = step.Name
			break
		}

		// Build execute request for step
		execReq := &executorv1.ExecuteRequest{
			Tool:           step.Tool,
			Args:           step.Args,
			Env:            append(req.Env, step.Env...),
			TimeoutSeconds: step.TimeoutSeconds,
		}

		// Set working directory
		if step.WorkDir != "" && req.WorkspaceDir != "" {
			execReq.WorkDir = req.WorkspaceDir + "/" + step.WorkDir
		} else if step.WorkDir != "" {
			execReq.WorkDir = step.WorkDir
		} else if req.WorkspaceDir != "" {
			execReq.WorkDir = req.WorkspaceDir
		}

		s.logger.Info("executing pipeline step",
			zap.String("pipeline_id", pipelineID),
			zap.Int("step_index", i),
			zap.String("step_name", step.Name),
		)

		execResult, err := s.Execute(ctx, execReq)

		stepResult := &executorv1.StepResult{
			Name:      step.Name,
			StepIndex: int32(i),
		}

		if err != nil {
			stepResult.ExecuteResult = &executorv1.ExecuteResponse{
				Success: false,
				Error:   err.Error(),
			}
			response.Success = false
			response.FailedStep = step.Name
		} else {
			stepResult.ExecuteResult = execResult
			if !execResult.Success && !step.ContinueOnError {
				response.Success = false
				response.FailedStep = step.Name
			}
		}

		response.StepResults = append(response.StepResults, stepResult)
		response.CompletedSteps = int32(i + 1)

		// Stop if step failed and not configured to continue
		if !response.Success && !step.ContinueOnError {
			break
		}
	}

	endTime := time.Now()
	response.EndedAt = timestamppb.New(endTime)
	response.TotalDurationMs = endTime.Sub(startTime).Milliseconds()

	s.logger.Info("pipeline completed",
		zap.String("pipeline_id", pipelineID),
		zap.Bool("success", response.Success),
		zap.Int64("duration_ms", response.TotalDurationMs),
	)

	return response, nil
}

// ExecutePipelineStream runs a pipeline and streams step outputs
func (s *ExecutorServer) ExecutePipelineStream(req *executorv1.PipelineRequest, stream executorv1.ExecutorService_ExecutePipelineStreamServer) error {
	startTime := time.Now()
	ctx := stream.Context()

	pipelineID := req.Id
	if pipelineID == "" {
		pipelineID = uuid.New().String()
	}

	s.logger.Info("starting pipeline stream",
		zap.String("pipeline_id", pipelineID),
		zap.String("name", req.Name),
		zap.Int("steps", len(req.Steps)),
	)

	// Set overall timeout
	if req.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(req.TimeoutSeconds)*time.Second)
		defer cancel()
	}

	response := &executorv1.PipelineResponse{
		PipelineId:  pipelineID,
		Name:        req.Name,
		StartedAt:   timestamppb.New(startTime),
		Success:     true,
		TotalSteps:  int32(len(req.Steps)),
		StepResults: make([]*executorv1.StepResult, 0, len(req.Steps)),
	}

	// Execute each step
	for i, step := range req.Steps {
		if ctx.Err() != nil {
			response.Success = false
			response.FailedStep = step.Name
			break
		}

		// Send step started event
		stepStarted := &executorv1.StepStartedEvent{
			StepName:   step.Name,
			StepIndex:  int32(i),
			TotalSteps: int32(len(req.Steps)),
			StartedAt:  timestamppb.Now(),
		}

		if err := stream.Send(&executorv1.PipelineStreamResponse{
			Event: &executorv1.PipelineStreamResponse_StepStarted{
				StepStarted: stepStarted,
			},
		}); err != nil {
			return err
		}

		// Execute step with output streaming
		stepResult := s.executeStepWithStreaming(ctx, req, step, i, stream)
		response.StepResults = append(response.StepResults, stepResult)
		response.CompletedSteps = int32(i + 1)

		// Send step completed event
		if err := stream.Send(&executorv1.PipelineStreamResponse{
			Event: &executorv1.PipelineStreamResponse_StepCompleted{
				StepCompleted: stepResult,
			},
		}); err != nil {
			return err
		}

		// Check if we should stop
		if stepResult.ExecuteResult != nil && !stepResult.ExecuteResult.Success {
			if !step.ContinueOnError {
				response.Success = false
				response.FailedStep = step.Name
				break
			}
		}
	}

	endTime := time.Now()
	response.EndedAt = timestamppb.New(endTime)
	response.TotalDurationMs = endTime.Sub(startTime).Milliseconds()

	// Send pipeline completed event
	return stream.Send(&executorv1.PipelineStreamResponse{
		Event: &executorv1.PipelineStreamResponse_PipelineCompleted{
			PipelineCompleted: response,
		},
	})
}

// executeStepWithStreaming executes a step and streams its output
func (s *ExecutorServer) executeStepWithStreaming(
	ctx context.Context,
	pipelineReq *executorv1.PipelineRequest,
	step *executorv1.BuildStep,
	stepIndex int,
	stream executorv1.ExecutorService_ExecutePipelineStreamServer,
) *executorv1.StepResult {
	result := &executorv1.StepResult{
		Name:      step.Name,
		StepIndex: int32(stepIndex),
	}

	// Build command
	cmd := exec.CommandContext(ctx, step.Tool, step.Args...)

	// Set working directory
	if step.WorkDir != "" && pipelineReq.WorkspaceDir != "" {
		cmd.Dir = pipelineReq.WorkspaceDir + "/" + step.WorkDir
	} else if step.WorkDir != "" {
		cmd.Dir = step.WorkDir
	} else if pipelineReq.WorkspaceDir != "" {
		cmd.Dir = pipelineReq.WorkspaceDir
	}

	// Set environment
	if len(pipelineReq.Env) > 0 || len(step.Env) > 0 {
		cmd.Env = append(cmd.Environ(), pipelineReq.Env...)
		cmd.Env = append(cmd.Env, step.Env...)
	}

	// Create pipes
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		result.ExecuteResult = &executorv1.ExecuteResponse{Success: false, Error: err.Error()}
		return result
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		result.ExecuteResult = &executorv1.ExecuteResponse{Success: false, Error: err.Error()}
		return result
	}

	// Start command
	stepStartTime := time.Now()
	processID := uuid.New().String()

	if err := cmd.Start(); err != nil {
		result.ExecuteResult = &executorv1.ExecuteResponse{Success: false, Error: err.Error()}
		return result
	}

	// Track running process
	ctx, cancel := context.WithCancel(ctx)
	runningProc := &RunningProcess{
		ID:        processID,
		Tool:      step.Tool,
		Args:      step.Args,
		Command:   cmd,
		Cancel:    cancel,
		StartedAt: stepStartTime,
	}

	s.mu.Lock()
	s.running[processID] = runningProc
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.running, processID)
		s.mu.Unlock()
	}()

	// Stream output
	var wg sync.WaitGroup
	var stdoutBuf, stderrBuf strings.Builder

	wg.Add(2)
	go func() {
		defer wg.Done()
		s.streamPipelineOutput(stdout, &stdoutBuf, stream, step.Name, int32(stepIndex), true)
	}()
	go func() {
		defer wg.Done()
		s.streamPipelineOutput(stderr, &stderrBuf, stream, step.Name, int32(stepIndex), false)
	}()

	wg.Wait()

	// Wait for command
	err = cmd.Wait()
	endTime := time.Now()
	duration := endTime.Sub(stepStartTime)

	execResult := &executorv1.ExecuteResponse{
		ProcessId:  processID,
		Tool:       step.Tool,
		Args:       step.Args,
		Stdout:     stdoutBuf.String(),
		Stderr:     stderrBuf.String(),
		DurationMs: duration.Milliseconds(),
		StartedAt:  timestamppb.New(stepStartTime),
		EndedAt:    timestamppb.New(endTime),
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			execResult.ExitCode = -1
			execResult.Error = "command timed out"
			execResult.TimedOut = true
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			execResult.ExitCode = int32(exitErr.ExitCode())
			execResult.Error = exitErr.Error()
		} else {
			execResult.ExitCode = -1
			execResult.Error = err.Error()
		}
		execResult.Success = false
	} else {
		execResult.ExitCode = 0
		execResult.Success = true
	}

	result.ExecuteResult = execResult
	return result
}

// CancelProcess cancels a running process by ID
func (s *ExecutorServer) CancelProcess(ctx context.Context, req *executorv1.CancelRequest) (*executorv1.CancelResponse, error) {
	s.mu.RLock()
	proc, exists := s.running[req.ProcessId]
	s.mu.RUnlock()

	if !exists {
		return &executorv1.CancelResponse{
			Success: false,
			Message: fmt.Sprintf("process %s not found", req.ProcessId),
		}, nil
	}

	proc.Cancel()

	return &executorv1.CancelResponse{
		Success: true,
		Message: fmt.Sprintf("process %s cancelled", req.ProcessId),
	}, nil
}

// GetRunningProcesses returns all currently running processes
func (s *ExecutorServer) GetRunningProcesses(ctx context.Context, req *executorv1.GetProcessesRequest) (*executorv1.GetProcessesResponse, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	processes := make([]*executorv1.ProcessInfo, 0, len(s.running))
	for _, proc := range s.running {
		processes = append(processes, &executorv1.ProcessInfo{
			ProcessId:  proc.ID,
			Tool:       proc.Tool,
			Args:       proc.Args,
			StartedAt:  timestamppb.New(proc.StartedAt),
			DurationMs: time.Since(proc.StartedAt).Milliseconds(),
		})
	}

	return &executorv1.GetProcessesResponse{
		Processes: processes,
		Count:     int32(len(processes)),
	}, nil
}

// Health returns the service health status
func (s *ExecutorServer) Health(ctx context.Context, req *executorv1.HealthRequest) (*executorv1.HealthResponse, error) {
	s.mu.RLock()
	runningCount := len(s.running)
	s.mu.RUnlock()

	return &executorv1.HealthResponse{
		Status:        "healthy",
		Version:       "0.1.0",
		RunningCount:  int32(runningCount),
		MaxConcurrent: int32(s.config.MaxConcurrent),
		UptimeSeconds: time.Since(startTime).Seconds(),
		CheckedAt:     timestamppb.Now(),
	}, nil
}

// Helper functions

func (s *ExecutorServer) readOutput(r io.Reader, buf *strings.Builder) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		buf.WriteString(scanner.Text())
		buf.WriteString("\n")
	}
}

func (s *ExecutorServer) streamOutput(r io.Reader, buf *strings.Builder, stream executorv1.ExecutorService_ExecuteStreamServer, isStdout bool) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		buf.WriteString(line)
		buf.WriteString("\n")

		var msg *executorv1.ExecuteStreamResponse
		if isStdout {
			msg = &executorv1.ExecuteStreamResponse{
				Output: &executorv1.ExecuteStreamResponse_StdoutLine{
					StdoutLine: line,
				},
			}
		} else {
			msg = &executorv1.ExecuteStreamResponse{
				Output: &executorv1.ExecuteStreamResponse_StderrLine{
					StderrLine: line,
				},
			}
		}

		if err := stream.Send(msg); err != nil {
			s.logger.Warn("failed to stream output", zap.Error(err))
			return
		}
	}
}

func (s *ExecutorServer) streamPipelineOutput(r io.Reader, buf *strings.Builder, stream executorv1.ExecutorService_ExecutePipelineStreamServer, stepName string, stepIndex int32, isStdout bool) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		buf.WriteString(line)
		buf.WriteString("\n")

		event := &executorv1.StepOutputEvent{
			StepName:  stepName,
			StepIndex: stepIndex,
		}

		if isStdout {
			event.Output = &executorv1.StepOutputEvent_StdoutLine{StdoutLine: line}
		} else {
			event.Output = &executorv1.StepOutputEvent_StderrLine{StderrLine: line}
		}

		msg := &executorv1.PipelineStreamResponse{
			Event: &executorv1.PipelineStreamResponse_StepOutput{
				StepOutput: event,
			},
		}

		if err := stream.Send(msg); err != nil {
			s.logger.Warn("failed to stream pipeline output", zap.Error(err))
			return
		}
	}
}
