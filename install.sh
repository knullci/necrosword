#!/bin/bash
# Necrosword Easy Install Script with systemd
# Usage: curl -fsSL https://raw.githubusercontent.com/knullci/necrosword/main/install.sh | bash
# With custom port: curl -fsSL ... | bash -s -- --port 9091

set -e

REPO="knullci/necrosword"
BINARY_NAME="necrosword"
INSTALL_DIR="/usr/local/bin"
CONFIG_DIR="/etc/necrosword"
SERVICE_USER="necrosword"
DEFAULT_PORT=8081
IS_UPGRADE=false
BACKUP_CONFIG=""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Parse arguments
PORT=$DEFAULT_PORT
SKIP_SERVICE=false
VERSION=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --port)
            PORT="$2"
            shift 2
            ;;
        --port=*)
            PORT="${1#*=}"
            shift
            ;;
        --version)
            VERSION="$2"
            shift 2
            ;;
        --version=*)
            VERSION="${1#*=}"
            shift
            ;;
        --no-service)
            SKIP_SERVICE=true
            shift
            ;;
        *)
            shift
            ;;
    esac
done

echo -e "${GREEN}╔══════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║     Necrosword Installer                     ║${NC}"
echo -e "${GREEN}║     gRPC Process Executor for Knull CI       ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════╝${NC}"
echo ""

# Check if running as root or with sudo
if [ "$EUID" -ne 0 ]; then
    echo -e "${YELLOW}Note: This script needs sudo access to install system services${NC}"
    SUDO="sudo"
else
    SUDO=""
fi

# Detect OS and Architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

case "$OS" in
    linux)
        PLATFORM="linux-${ARCH}"
        ;;
    darwin)
        PLATFORM="darwin-${ARCH}"
        echo -e "${YELLOW}Note: systemd is not available on macOS. Skipping service setup.${NC}"
        SKIP_SERVICE=true
        ;;
    *)
        echo -e "${RED}Error: Unsupported OS: $OS${NC}"
        exit 1
        ;;
esac

echo -e "Platform: ${BLUE}${PLATFORM}${NC}"
echo -e "Port: ${BLUE}${PORT}${NC}"

# Check for existing installation (upgrade detection)
if [ -f "${CONFIG_DIR}/necrosword.conf" ]; then
    IS_UPGRADE=true
    echo -e "${YELLOW}Existing installation detected - running in upgrade mode${NC}"
    echo -e "${GREEN}Your existing configuration will be preserved${NC}"
fi

# Get version (use provided or fetch latest)
echo ""
if [ -n "$VERSION" ]; then
    # Remove 'v' prefix if provided
    VERSION="${VERSION#v}"
    echo -e "Using specified version: ${BLUE}v${VERSION}${NC}"
else
    echo "Fetching latest version..."
    VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"v?([^"]+)".*/\1/')
    
    if [ -z "$VERSION" ]; then
        # Fallback: try to get the first release (including prereleases)
        echo -e "${YELLOW}No stable release found, checking for prereleases...${NC}"
        VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases" | grep '"tag_name":' | head -1 | sed -E 's/.*"v?([^"]+)".*/\1/')
    fi
    
    if [ -z "$VERSION" ]; then
        echo -e "${RED}Error: No releases found. Please specify a version with --version${NC}"
        exit 1
    fi
    
    echo -e "Version: ${BLUE}v${VERSION}${NC}"
fi

# Download
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/v${VERSION}/${BINARY_NAME}-${PLATFORM}"
echo ""
echo "Downloading Necrosword..."

TMP_DIR=$(mktemp -d)
TMP_FILE="${TMP_DIR}/${BINARY_NAME}"

if ! curl -fsSL -o "$TMP_FILE" "$DOWNLOAD_URL"; then
    echo -e "${RED}Error: Download failed${NC}"
    rm -rf "$TMP_DIR"
    exit 1
fi

chmod +x "$TMP_FILE"

# Install binary
echo "Installing to ${INSTALL_DIR}..."
$SUDO mv "$TMP_FILE" "${INSTALL_DIR}/${BINARY_NAME}"

# Create config directory
$SUDO mkdir -p "$CONFIG_DIR"

# Backup existing configuration before upgrade
if [ "$IS_UPGRADE" = true ]; then
    echo "Backing up existing configuration..."
    BACKUP_CONFIG="/tmp/necrosword.conf.backup.$$"
    $SUDO cp "${CONFIG_DIR}/necrosword.conf" "$BACKUP_CONFIG"
    
    # Stop the service before upgrading
    if [ "$SKIP_SERVICE" = false ] && command -v systemctl &> /dev/null; then
        echo "Stopping Necrosword service for upgrade..."
        $SUDO systemctl stop necrosword 2>/dev/null || true
    fi
fi

# Handle configuration
if [ "$IS_UPGRADE" = true ] && [ -n "$BACKUP_CONFIG" ] && [ -f "$BACKUP_CONFIG" ]; then
    echo "Restoring existing configuration..."
    $SUDO cp "$BACKUP_CONFIG" "${CONFIG_DIR}/necrosword.conf"
    rm -f "$BACKUP_CONFIG"
    echo -e "${GREEN}Configuration preserved from previous installation${NC}"
else
    # Create new config file (fresh install only)
    echo "Creating configuration..."
    $SUDO tee "$CONFIG_DIR/necrosword.conf" > /dev/null << EOF
# Necrosword Configuration
# Generated by install script

# gRPC server port (NECROSWORD_SERVER_PORT)
NECROSWORD_SERVER_PORT=${PORT}

# Workspace directory for build execution
# This MUST match Knull's KNULL_WORKSPACE_BASE_PATH setting
# Using /var/lib for shared access between Knull and Necrosword
NECROSWORD_EXECUTOR_WORKSPACE_BASE=/var/lib/knull-workspace

# Maximum concurrent processes
# NECROSWORD_EXECUTOR_MAX_CONCURRENT=10
EOF
fi

# Setup systemd service (Linux only)
if [ "$SKIP_SERVICE" = false ] && [ "$OS" = "linux" ]; then
    echo ""
    echo "Setting up systemd service..."
    
    # Create service user if not exists
    if ! id "$SERVICE_USER" &>/dev/null; then
        $SUDO useradd --system --no-create-home --shell /bin/false "$SERVICE_USER" 2>/dev/null || true
    fi
    
    # Create service working directory
    $SUDO mkdir -p /var/lib/necrosword
    $SUDO chown -R "$SERVICE_USER:$SERVICE_USER" /var/lib/necrosword 2>/dev/null || true
    
    # Create shared workspace directory (accessible by both Knull and Necrosword)
    $SUDO mkdir -p /var/lib/knull-workspace
    $SUDO chmod 1777 /var/lib/knull-workspace  # World-writable for both services
    
    # Create systemd service
    $SUDO tee /etc/systemd/system/necrosword.service > /dev/null << EOF
[Unit]
Description=Necrosword gRPC Process Executor
Documentation=https://github.com/knullci/necrosword
After=network.target

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_USER}
EnvironmentFile=${CONFIG_DIR}/necrosword.conf
ExecStart=${INSTALL_DIR}/${BINARY_NAME} server
Restart=on-failure
RestartSec=10
WorkingDirectory=/var/lib/necrosword

# Security
NoNewPrivileges=true
PrivateTmp=true

# Logging
StandardOutput=journal
StandardError=journal
SyslogIdentifier=necrosword

[Install]
WantedBy=multi-user.target
EOF

    # Reload systemd
    $SUDO systemctl daemon-reload
    
    # Enable service
    $SUDO systemctl enable necrosword
    
    # Start service
    echo ""
    echo "Starting Necrosword..."
    $SUDO systemctl start necrosword
    
    sleep 2
    
    # Check status
    if $SUDO systemctl is-active --quiet necrosword; then
        echo -e "${GREEN}✓ Necrosword is running!${NC}"
    else
        echo -e "${YELLOW}Warning: Service may not have started correctly${NC}"
        echo "Check logs with: sudo journalctl -u necrosword -f"
    fi
fi

rm -rf "$TMP_DIR"

echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════╗${NC}"
if [ "$IS_UPGRADE" = true ]; then
echo -e "${GREEN}║     Upgrade Complete!                        ║${NC}"
else
echo -e "${GREEN}║     Installation Complete!                   ║${NC}"
fi
echo -e "${GREEN}╚══════════════════════════════════════════════╝${NC}"
echo ""
echo -e "Necrosword gRPC server listening on: ${BLUE}localhost:${PORT}${NC}"
echo ""
echo "Useful commands:"
echo -e "  ${YELLOW}sudo systemctl status necrosword${NC}    - Check status"
echo -e "  ${YELLOW}sudo systemctl restart necrosword${NC}   - Restart"
echo -e "  ${YELLOW}sudo systemctl stop necrosword${NC}      - Stop"
echo -e "  ${YELLOW}sudo journalctl -u necrosword -f${NC}    - View logs"
echo ""
echo "Configuration: ${CONFIG_DIR}/necrosword.conf"
echo ""
echo "To change the port, edit ${CONFIG_DIR}/necrosword.conf and restart:"
echo "  sudo nano ${CONFIG_DIR}/necrosword.conf"
echo "  sudo systemctl restart necrosword"
echo ""
echo -e "${BLUE}Note: Configure Knull CI to connect to this executor at localhost:${PORT}${NC}"
