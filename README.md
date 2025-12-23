# 🗡️ Necrosword

**High-performance gRPC process executor for Knull CI/CD**

Necrosword is a blazing-fast, lightweight process executor written in Go. It handles build pipeline execution, command running, and real-time output streaming for the Knull CI/CD platform via gRPC.

## ✨ Features

| Feature | Description |
|---------|-------------|
| ⚡ **Fast Execution** | Native Go performance for minimal overhead |
| 📡 **gRPC API** | High-performance RPC with streaming support |
| 🔄 **Real-time Streaming** | Stream command output as it happens |
| 🔀 **Pipeline Support** | Execute multi-step build pipelines |
| 🔒 **Tool Whitelisting** | Only allow approved tools to execute |
| ⏱️ **Timeout Handling** | Configurable timeouts with graceful termination |
| 🏃 **Process Management** | Track, monitor, and cancel running processes |

---

## 📁 Project Structure

```
necrosword/
├── api/
│   └── proto/
│       └── executor/
│           └── v1/
│               └── executor.proto       # Protocol buffer definitions
├── cmd/
│   └── necrosword/
│       └── main.go                      # CLI entry point
├── gen/
│   └── executor/
│       └── v1/
│           ├── executor.pb.go           # Generated protobuf
│           └── executor_grpc.pb.go      # Generated gRPC stubs
├── internal/
│   ├── app/
│   │   └── app.go                       # gRPC server setup
│   ├── config/
│   │   └── config.go                    # Configuration management
│   └── grpc/
│       └── server.go                    # gRPC service implementation
├── config/
│   └── config.yaml                      # Default configuration
├── buf.yaml                             # Buf module config
├── buf.gen.yaml                         # Buf codegen config
├── Dockerfile                           # Container build
├── Makefile                             # Build automation
├── go.mod / go.sum                      # Go modules
└── README.md                            # This file
```

---

## 🔌 gRPC Service (ExecutorService)

| RPC | Type | Description |
|-----|------|-------------|
| `Execute` | Unary | Run command, return result |
| `ExecuteStream` | Server streaming | Run command, stream stdout/stderr lines |
| `ExecutePipeline` | Unary | Run multi-step pipeline, return results |
| `ExecutePipelineStream` | Server streaming | Run pipeline, stream step progress & output |
| `CancelProcess` | Unary | Cancel running process by ID |
| `GetRunningProcesses` | Unary | List running processes |
| `Health` | Unary | Health check |

### Proto Location

Protocol buffer definitions are in `api/proto/executor/v1/executor.proto`

---

## 🚀 Quick Start

### Prerequisites

- Go 1.22 or higher
- [Buf](https://buf.build/docs/installation) (for proto generation)
- Make (optional)

### Build

```bash
# Download dependencies
go mod tidy

# Build the binary
make build

# Or directly with Go
go build -o bin/necrosword ./cmd/necrosword
```

### Run Server

```bash
# Start the gRPC server (default: localhost:8081)
./bin/necrosword server

# Or with Make
make run
```

### CLI Execution

```bash
# Execute a git command
./bin/necrosword execute --tool git --args "version"

# Execute npm install in a directory
./bin/necrosword execute --tool npm --args "install" --workdir /path/to/project
```

---

## 🧪 Test with grpcurl

```bash
# Install grpcurl if needed
# brew install grpcurl (macOS)
# go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest

# List available services
grpcurl -plaintext localhost:8081 list

# Health check
grpcurl -plaintext localhost:8081 executor.v1.ExecutorService/Health

# Execute a command
grpcurl -plaintext -d '{
  "tool": "git",
  "args": ["version"]
}' localhost:8081 executor.v1.ExecutorService/Execute

# Execute with streaming output
grpcurl -plaintext -d '{
  "tool": "npm",
  "args": ["install"],
  "work_dir": "/path/to/project"
}' localhost:8081 executor.v1.ExecutorService/ExecuteStream

# Execute a pipeline
grpcurl -plaintext -d '{
  "id": "build-123",
  "name": "My Build",
  "workspace_dir": "/workspace/build-123",
  "steps": [
    {"name": "Install", "tool": "npm", "args": ["install"]},
    {"name": "Build", "tool": "npm", "args": ["run", "build"]},
    {"name": "Test", "tool": "npm", "args": ["test"]}
  ]
}' localhost:8081 executor.v1.ExecutorService/ExecutePipeline

# Stream pipeline execution
grpcurl -plaintext -d '{
  "id": "build-456",
  "name": "Streaming Build",
  "workspace_dir": "/workspace/build-456",
  "steps": [
    {"name": "Install", "tool": "npm", "args": ["install"]},
    {"name": "Build", "tool": "npm", "args": ["run", "build"]}
  ]
}' localhost:8081 executor.v1.ExecutorService/ExecutePipelineStream

# Get running processes
grpcurl -plaintext localhost:8081 executor.v1.ExecutorService/GetRunningProcesses

# Cancel a process
grpcurl -plaintext -d '{"process_id": "uuid-here"}' \
  localhost:8081 executor.v1.ExecutorService/CancelProcess
```

---

## ⚙️ Configuration

Configuration can be provided via:
1. YAML file (`config/config.yaml`, `./config.yaml`, or `/etc/necrosword/config.yaml`)
2. Environment variables (prefixed with `NECROSWORD_`)

### Example Configuration

```yaml
server:
  host: "0.0.0.0"
  port: 8081

executor:
  allowed_tools:
    - git
    - npm
    - mvn
    - docker
    - kubectl
    - go
    - make
    - yarn
    - pnpm
    - gradle
    - python
    - pip
  default_timeout: 3600    # 1 hour in seconds
  max_concurrent: 10
  workspace_base: "workspace"

logging:
  level: "info"            # debug, info, warn, error
  format: "json"           # json or console
```

### Environment Variables

```bash
export NECROSWORD_SERVER_PORT=8081
export NECROSWORD_EXECUTOR_DEFAULT_TIMEOUT=3600
export NECROSWORD_LOGGING_LEVEL=debug
export NECROSWORD_LOGGING_FORMAT=console
```

---

## 🛠️ Development

### Regenerate Proto Code

```bash
# Using buf (recommended)
buf generate

# Or via Makefile
make proto
```

### Run Tests

```bash
make test
```

### Build for All Platforms

```bash
make build-all  # Builds for Linux and macOS
```

---

## 🐳 Docker

```bash
# Build image
make docker

# Or directly
docker build -t necrosword .

# Run container
docker run -p 8081:8081 necrosword
```

---

## 🔗 Integration with Knull CI/CD (Java)

Necrosword is designed to be called from the Knull Java backend via gRPC. The Java project will manage the overall CI/CD workflow while Necrosword handles the actual command execution.

### Setup for Java

1. Copy `api/proto/executor/v1/executor.proto` to the Java project
2. Use gradle/maven protobuf plugin to generate Java stubs
3. Create a gRPC client to call Necrosword

### Example Java Integration

```java
// Create channel
ManagedChannel channel = ManagedChannelBuilder
    .forAddress("localhost", 8081)
    .usePlaintext()
    .build();

// Create blocking stub (for unary calls)
ExecutorServiceGrpc.ExecutorServiceBlockingStub stub = 
    ExecutorServiceGrpc.newBlockingStub(channel);

// Execute a command
ExecuteRequest request = ExecuteRequest.newBuilder()
    .setTool("npm")
    .addArgs("install")
    .setWorkDir(workspaceDir)
    .setTimeoutSeconds(300)
    .build();

ExecuteResponse response = stub.execute(request);
System.out.println("Success: " + response.getSuccess());
System.out.println("Output: " + response.getStdout());
```

### Streaming Example (Java)

```java
// Create async stub (for streaming)
ExecutorServiceGrpc.ExecutorServiceStub asyncStub = 
    ExecutorServiceGrpc.newStub(channel);

ExecuteRequest request = ExecuteRequest.newBuilder()
    .setTool("npm")
    .addArgs("install")
    .setWorkDir(workspaceDir)
    .build();

// Stream output in real-time
asyncStub.executeStream(request, new StreamObserver<ExecuteStreamResponse>() {
    @Override
    public void onNext(ExecuteStreamResponse response) {
        if (response.hasStdoutLine()) {
            System.out.println("[stdout] " + response.getStdoutLine());
        } else if (response.hasStderrLine()) {
            System.err.println("[stderr] " + response.getStderrLine());
        } else if (response.hasResult()) {
            ExecuteResponse result = response.getResult();
            System.out.println("Completed: " + result.getSuccess());
        }
    }
    
    @Override
    public void onError(Throwable t) {
        t.printStackTrace();
    }
    
    @Override
    public void onCompleted() {
        System.out.println("Stream completed");
    }
});
```

### Pipeline Execution (Java)

```java
PipelineRequest pipeline = PipelineRequest.newBuilder()
    .setId("build-" + buildId)
    .setName("CI Build")
    .setWorkspaceDir(workspaceDir)
    .addSteps(BuildStep.newBuilder()
        .setName("Clone Repository")
        .setTool("git")
        .addArgs("clone")
        .addArgs(repoUrl)
        .addArgs(".")
        .build())
    .addSteps(BuildStep.newBuilder()
        .setName("Install Dependencies")
        .setTool("npm")
        .addArgs("install")
        .build())
    .addSteps(BuildStep.newBuilder()
        .setName("Run Build")
        .setTool("npm")
        .addArgs("run")
        .addArgs("build")
        .build())
    .addSteps(BuildStep.newBuilder()
        .setName("Run Tests")
        .setTool("npm")
        .addArgs("test")
        .setContinueOnError(true)  // Continue even if tests fail
        .build())
    .setTimeoutSeconds(1800)  // 30 minutes
    .build();

PipelineResponse result = stub.executePipeline(pipeline);
System.out.println("Pipeline success: " + result.getSuccess());
System.out.println("Completed steps: " + result.getCompletedSteps() + "/" + result.getTotalSteps());
```

---

## 📋 Makefile Targets

| Target | Description |
|--------|-------------|
| `make build` | Build the binary |
| `make run` | Build and run the server |
| `make test` | Run tests |
| `make proto` | Regenerate protobuf code |
| `make build-all` | Build for Linux and macOS |
| `make docker` | Build Docker image |
| `make clean` | Clean build artifacts |
| `make help` | Show all targets |

---

## 📝 License

Part of the Knull CI/CD project.
