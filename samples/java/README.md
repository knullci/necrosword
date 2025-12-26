# Java Integration Guide

Necrosword is designed to be easily called from Java applications (like KnullCI) using gRPC. Below are examples of how to integrate it.

## Setup

1. Copy `api/proto/executor/v1/executor.proto` from this repository to your Java project's `src/main/proto` directory.
2. Use the Maven or Gradle protobuf plugin to generate the Java stubs.

## Examples

### 1. Basic Command Execution (Unary)

Use a blocking stub for simple commands that return a single result.

```java
// Create channel
ManagedChannel channel = ManagedChannelBuilder
    .forAddress("localhost", 8081)
    .usePlaintext()
    .build();

// Create blocking stub
ExecutorServiceGrpc.ExecutorServiceBlockingStub stub = 
    ExecutorServiceGrpc.newBlockingStub(channel);

// Execute a command
ExecuteRequest request = ExecuteRequest.newBuilder()
    .setTool("npm")
    .addArgs("install")
    .setWorkDir("/path/to/workspace")
    .setTimeoutSeconds(300)
    .build();

ExecuteResponse response = stub.execute(request);
System.out.println("Success: " + response.getSuccess());
System.out.println("Output: " + response.getStdout());
```

### 2. Real-time Streaming Output

Use an async stub to receive stdout/stderr lines as they occur.

```java
// Create async stub
ExecutorServiceGrpc.ExecutorServiceStub asyncStub = 
    ExecutorServiceGrpc.newStub(channel);

ExecuteRequest request = ExecuteRequest.newBuilder()
    .setTool("npm")
    .addArgs("install")
    .setWorkDir("/path/to/workspace")
    .build();

// Stream output
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

### 3. Pipeline Execution

Execute a complex multi-step pipeline.

```java
PipelineRequest pipeline = PipelineRequest.newBuilder()
    .setId("build-123")
    .setName("CI Build")
    .setWorkspaceDir("/path/to/workspace")
    .addSteps(BuildStep.newBuilder()
        .setName("Clone Repository")
        .setTool("git")
        .addArgs("clone")
        .addArgs("https://github.com/user/repo.git")
        .addArgs(".")
        .build())
    .addSteps(BuildStep.newBuilder()
        .setName("Install Dependencies")
        .setTool("npm")
        .addArgs("install")
        .build())
    .addSteps(BuildStep.newBuilder()
        .setName("Run Tests")
        .setTool("npm")
        .addArgs("test")
        .setContinueOnError(true)
        .build())
    .setTimeoutSeconds(1800)
    .build();

PipelineResponse result = stub.executePipeline(pipeline);
System.out.println("Pipeline success: " + result.getSuccess());
```
