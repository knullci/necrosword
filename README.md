# 🗡️ Necrosword

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go](https://img.shields.io/badge/Go-1.22-cyan.svg)
![gRPC](https://img.shields.io/badge/gRPC-High%20Performance-green.svg)

> **The blazing-fast, open-source process executor for modern CI/CD pipelines.**

Necrosword is a high-performance command execution engine written in **Go**. Originally built as the powerhouse behind **KnullCI**, it is designed to be a standalone component for any system requiring secure, isolated, and streamed process execution.

---

## ✨ Features

- **Extreme Performance**: Built with Go 1.22 for minimal overhead and massive concurrency.
- **Real-Time Streaming**: Native gRPC streaming for stdout/stderr—see logs as they happen.
- **Pipeline Orchestration**: Execute complex, multi-step build pipelines with a single request.
- **Secure Sandbox**: White-listed tool execution to prevent unauthorized commands.
- **Resource Efficient**: Tiny memory footprint compared to JVM or Node.js executors.

---

## 🛠️ Technology Stack

- **Language**: Go 1.22
- **Protocol**: gRPC & Protocol Buffers (v3)
- **Architecture**: Hexagonal / Clean Architecture
- **Dependencies**: Cobra (CLI), Viper (Config), Zap (Logging)

---

## 🚀 Quick Start

### Prerequisites

- **Go 1.22+**
- **Make** (optional)

### Installation

```bash
# Clone the repository
git clone https://github.com/knullci/necrosword.git
cd necrosword

# Build the binary
make build
```

### Running the Server

Start the gRPC server to listen for execution requests:

```bash
./bin/necrosword server
```
*Listens on port `8081` by default.*

### CLI Mode

You can also use Necrosword as a direct CLI tool:

```bash
./bin/necrosword execute --tool git --args "status"
```

---

## 🔌 API Integration (gRPC)

Necrosword exposes a simple yet powerful Protobuf API.

**Execute a Command:**
```protobuf
service ExecutorService {
  rpc ExecuteStream(ExecuteRequest) returns (stream ExecuteStreamResponse);
}
```

Check `api/proto/executor/v1/executor.proto` for the full definition.

> **Java Developers:** Check out the [Java Integration Guide](samples/java/README.md) for code examples.

---

## 🤝 Contributing

Necrosword is open source and community-driven. We want to make it the fastest executor on the planet.

1.  **Fork** the project.
2.  **Clone** your fork.
3.  **Hack** away! (Check `Makefile` for dev commands like `make proto` to regen GRPC code).
4.  **Submit** a Pull Request.

## 📄 License

This software is released under the **MIT License**. See the [LICENSE](LICENSE) file for more details.

---

<p align="center">
  <b>Fast. Secure. Unrelenting.</b><br>
  <i>Part of the Knull Open Source Ecosystem.</i>
</p>
