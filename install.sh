#!/bin/bash
set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Print functions
print_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
if [ "$EUID" -eq 0 ]; then
    print_warn "Running as root is not recommended. Please run as a regular user."
    exit 1
fi

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64|amd64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    *)
        print_error "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

print_info "Detected OS: $OS, Architecture: $ARCH"

# Set installation directory
INSTALL_DIR="${HOME}/.local/bin"
CONFIG_DIR="${HOME}/.gopherpaw"
DATA_DIR="${HOME}/.gopherpaw"

# Create directories
print_info "Creating directories..."
mkdir -p "$INSTALL_DIR"
mkdir -p "$CONFIG_DIR"
mkdir -p "$DATA_DIR"

# Check for Go
if ! command -v go &> /dev/null; then
    print_error "Go is not installed. Please install Go 1.23 or later."
    print_info "Visit: https://golang.org/doc/install"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
print_info "Go version: $GO_VERSION"

# Build from source
print_info "Building GopherPaw from source..."
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

go build -o gopherpaw ./cmd/gopherpaw/

# Install binary
print_info "Installing GopherPaw to $INSTALL_DIR..."
cp gopherpaw "$INSTALL_DIR/gopherpaw"
chmod +x "$INSTALL_DIR/gopherpaw"

# Create default config if not exists
if [ ! -f "$CONFIG_DIR/config.yaml" ]; then
    print_info "Creating default configuration..."
    if [ -f "configs/config.yaml.example" ]; then
        cp configs/config.yaml.example "$CONFIG_DIR/config.yaml"
    fi
fi

# Create active_skills and customized_skills directories
mkdir -p "$CONFIG_DIR/active_skills"
mkdir -p "$CONFIG_DIR/customized_skills"

# Check if INSTALL_DIR is in PATH
if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    print_warn "$INSTALL_DIR is not in your PATH."
    print_info "Add the following to your shell profile (~/.bashrc or ~/.zshrc):"
    echo ""
    echo "    export PATH=\"\$PATH:$INSTALL_DIR\""
    echo ""
fi

# Print success message
print_info "GopherPaw installed successfully!"
print_info "Binary location: $INSTALL_DIR/gopherpaw"
print_info "Config directory: $CONFIG_DIR"
print_info "Data directory: $DATA_DIR"
echo ""
print_info "To get started, run:"
echo "    gopherpaw --help"
