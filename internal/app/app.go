package app

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	executorv1 "github.com/knullci/necrosword/gen/executor/v1"
	"github.com/knullci/necrosword/internal/config"
	grpcserver "github.com/knullci/necrosword/internal/grpc"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// App is the main application struct
type App struct {
	config     *config.Config
	logger     *zap.Logger
	grpcServer *grpc.Server
	execServer *grpcserver.ExecutorServer
}

// New creates a new application instance
func New(cfg *config.Config) (*App, error) {
	// Initialize logger
	logger, err := initLogger(cfg.Logging)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	// Create executor server
	execServer := grpcserver.NewExecutorServer(&cfg.Executor, logger)

	return &App{
		config:     cfg,
		logger:     logger,
		execServer: execServer,
	}, nil
}

// Run starts the gRPC server
func (a *App) Run() error {
	// Create listener
	address := a.config.Server.Address()
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
	}

	// Create gRPC server
	a.grpcServer = grpc.NewServer(
		grpc.MaxRecvMsgSize(50*1024*1024), // 50MB max message size
		grpc.MaxSendMsgSize(50*1024*1024),
	)

	// Register executor service
	executorv1.RegisterExecutorServiceServer(a.grpcServer, a.execServer)

	// Enable reflection for debugging (grpcurl, etc.)
	reflection.Register(a.grpcServer)

	// Graceful shutdown
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
		<-quit

		a.logger.Info("shutting down gRPC server...")

		// Graceful stop with timeout
		done := make(chan struct{})
		go func() {
			a.grpcServer.GracefulStop()
			close(done)
		}()

		select {
		case <-done:
			a.logger.Info("server stopped gracefully")
		case <-time.After(30 * time.Second):
			a.logger.Warn("forcing server stop after timeout")
			a.grpcServer.Stop()
		}
	}()

	a.logger.Info("starting gRPC server",
		zap.String("address", address),
		zap.Strings("allowed_tools", a.config.Executor.AllowedTools),
	)

	if err := a.grpcServer.Serve(listener); err != nil {
		return fmt.Errorf("gRPC server error: %w", err)
	}

	return nil
}

// ExecuteCommand runs a single command (CLI mode)
func (a *App) ExecuteCommand(tool, args, workdir string) error {
	// Parse comma-separated args
	var argList []string
	if args != "" {
		argList = strings.Split(args, ",")
		for i := range argList {
			argList[i] = strings.TrimSpace(argList[i])
		}
	}

	req := &executorv1.ExecuteRequest{
		Tool:    tool,
		Args:    argList,
		WorkDir: workdir,
	}

	result, err := a.execServer.Execute(context.Background(), req)
	if err != nil {
		a.logger.Error("command execution failed", zap.Error(err))
		return err
	}

	// Print results
	fmt.Printf("\n=== Execution Result ===\n")
	fmt.Printf("Tool: %s\n", result.Tool)
	fmt.Printf("Args: %v\n", result.Args)
	fmt.Printf("Exit Code: %d\n", result.ExitCode)
	fmt.Printf("Duration: %dms\n", result.DurationMs)
	fmt.Printf("Success: %v\n", result.Success)

	if result.Stdout != "" {
		fmt.Printf("\n--- STDOUT ---\n%s\n", result.Stdout)
	}

	if result.Stderr != "" {
		fmt.Printf("\n--- STDERR ---\n%s\n", result.Stderr)
	}

	if !result.Success {
		return fmt.Errorf("command failed with exit code %d", result.ExitCode)
	}

	return nil
}

// initLogger initializes the Zap logger
func initLogger(cfg config.LoggingConfig) (*zap.Logger, error) {
	var zapCfg zap.Config

	if cfg.Format == "console" {
		zapCfg = zap.NewDevelopmentConfig()
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		zapCfg = zap.NewProductionConfig()
	}

	// Set log level
	switch strings.ToLower(cfg.Level) {
	case "debug":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		zapCfg.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		zapCfg.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	return zapCfg.Build()
}
