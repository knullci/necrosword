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

### Quick Install (Recommended)

One-line install for Ubuntu/Debian with automatic systemd service setup:

```bash
curl -fsSL https://raw.githubusercontent.com/knullci/necrosword/main/install.sh | sudo bash
```

This will:
- Download the latest binary for your platform
- Install to `/usr/local/bin/necrosword`
- Create systemd service
- Start Necrosword on port 8081
- Enable auto-start on boot

#### Custom Port

```bash
curl -fsSL https://raw.githubusercontent.com/knullci/necrosword/main/install.sh | sudo bash -s -- --port 9091
```

#### Specific Version (Including Beta/RC)

```bash
curl -fsSL https://raw.githubusercontent.com/knullci/necrosword/main/install.sh | sudo bash -s -- --version 0.0.1-beta
```

#### Install Without Service (Manual Control)

```bash
curl -fsSL https://raw.githubusercontent.com/knullci/necrosword/main/install.sh | sudo bash -s -- --no-service
```

#### All Options Combined

```bash
curl -fsSL https://raw.githubusercontent.com/knullci/necrosword/main/install.sh | sudo bash -s -- --version 1.0.0 --port 9091
```

### Service Management

```bash
# Check status
sudo systemctl status necrosword

# Stop/Start/Restart
sudo systemctl stop necrosword
sudo systemctl start necrosword
sudo systemctl restart necrosword

# View logs
sudo journalctl -u necrosword -f

# Change port after installation
sudo nano /etc/necrosword/necrosword.conf   # Edit NECROSWORD_PORT=9091
sudo systemctl restart necrosword
```

### Manual Installation

#### Prerequisites

- **Go 1.22+** (for building from source)
- **Make** (optional)

#### From Source

```bash
git clone https://github.com/knullci/necrosword.git
cd necrosword
make build
./bin/necrosword server --port 8081
```

#### Download Binary

```bash
# Download for your platform (linux-amd64, linux-arm64, darwin-amd64, darwin-arm64)
curl -L -o necrosword https://github.com/knullci/necrosword/releases/latest/download/necrosword-linux-amd64
chmod +x necrosword
sudo mv necrosword /usr/local/bin/

# Run
necrosword server --port 8081
```

### CLI Mode

You can also use Necrosword as a direct CLI tool:

```bash
necrosword execute --tool git --args "status"
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
