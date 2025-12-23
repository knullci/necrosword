# Build stage
FROM golang:1.22-alpine AS builder

# Install git and ca-certificates (needed for fetching dependencies)
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=$(git describe --tags --always) -X main.commit=$(git rev-parse --short HEAD)" \
    -o /necrosword ./cmd/necrosword

# Runtime stage
FROM alpine:3.19

# Install necessary tools for build execution
RUN apk add --no-cache \
    git \
    openssh-client \
    ca-certificates \
    curl \
    bash \
    docker-cli \
    && rm -rf /var/cache/apk/*

# Create non-root user
RUN addgroup -g 1000 necrosword && \
    adduser -u 1000 -G necrosword -s /bin/sh -D necrosword

# Create workspace directory
RUN mkdir -p /workspace && chown necrosword:necrosword /workspace

# Copy binary from builder
COPY --from=builder /necrosword /usr/local/bin/necrosword

# Copy default config
COPY --from=builder /app/config/config.yaml /etc/necrosword/config.yaml

# Set working directory
WORKDIR /workspace

# Switch to non-root user
USER necrosword

# Expose port
EXPOSE 8081

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8081/health || exit 1

# Run the server
ENTRYPOINT ["necrosword"]
CMD ["server"]
