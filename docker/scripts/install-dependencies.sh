#!/bin/bash
# GopherPaw 依赖安装脚本
# 根据 internal/runtime/detector.go 中的依赖列表安装

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_info "========================================="
log_info "GopherPaw Dependencies Installer"
log_info "========================================="
log_info ""

# 检测包管理器
detect_package_manager() {
    if command -v apt-get &> /dev/null; then
        echo "apt"
    elif command -v yum &> /dev/null; then
        echo "yum"
    elif command -v dnf &> /dev/null; then
        echo "dnf"
    elif command -v pacman &> /dev/null; then
        echo "pacman"
    else
        echo "unknown"
    fi
}

PKG_MGR=$(detect_package_manager)

if [ "$PKG_MGR" = "unknown" ]; then
    log_error "Unsupported package manager"
    exit 1
fi

log_info "Detected package manager: $PKG_MGR"
log_info ""

# 依赖列表（对应 detector.go 中的 skillBinaries）
declare -A DEPENDENCIES=(
    ["soffice"]="libreoffice"
    ["pdftoppm"]="poppler-utils"
    ["himalaya"]="himalaya"
    ["pandoc"]="pandoc"
    ["ffmpeg"]="ffmpeg"
    ["git"]="git"
)

# 检查并安装依赖
install_dependency() {
    local binary=$1
    local package=$2
    
    log_info "Checking $binary..."
    
    if command -v $binary &> /dev/null; then
        log_info "  ✓ $binary already installed"
        return 0
    fi
    
    log_info "  ✗ $binary not found, installing $package..."
    
    case $PKG_MGR in
        apt)
            sudo apt-get update -qq
            sudo apt-get install -y -qq $package
            ;;
        yum)
            sudo yum install -y -q $package
            ;;
        dnf)
            sudo dnf install -y -q $package
            ;;
        pacman)
            sudo pacman -S --noconfirm --quiet $package
            ;;
    esac
    
    if [ $? -eq 0 ]; then
        log_info "  ✓ $package installed successfully"
    else
        log_error "  ✗ Failed to install $package"
        return 1
    fi
}

# 特殊处理：himalaya (需要 cargo)
install_himalaya() {
    log_info "Installing himalaya (requires Rust/cargo)..."
    
    # 检查 cargo
    if ! command -v cargo &> /dev/null; then
        log_info "  Installing Rust..."
        curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
        source ~/.cargo/env
    fi
    
    # 安装 himalaya
    cargo install himalaya
    
    if [ $? -eq 0 ]; then
        log_info "  ✓ himalaya installed successfully"
    else
        log_error "  ✗ Failed to install himalaya"
        return 1
    fi
}

# 安装所有依赖
log_info "Installing dependencies..."
log_info ""

for binary in "${!DEPENDENCIES[@]}"; do
    package="${DEPENDENCIES[$binary]}"
    
    if [ "$binary" = "himalaya" ]; then
        install_himalaya || log_warn "Failed to install himalaya (optional)"
    else
        install_dependency "$binary" "$package" || log_warn "Failed to install $package"
    fi
    
    log_info ""
done

# 验证安装
log_info "========================================="
log_info "Verification"
log_info "========================================="
log_info ""

FAILED=0

for binary in "${!DEPENDENCIES[@]}"; do
    if command -v $binary &> /dev/null; then
        version=$($binary --version 2>&1 | head -n 1 || echo "unknown")
        log_info "✓ $binary: $version"
    else
        log_error "✗ $binary: NOT INSTALLED"
        FAILED=$((FAILED + 1))
    fi
done

log_info ""

if [ $FAILED -eq 0 ]; then
    log_info "========================================="
    log_info "All dependencies installed successfully!"
    log_info "========================================="
    exit 0
else
    log_warn "========================================="
    log_warn "$FAILED dependencies failed to install"
    log_warn "========================================="
    exit 1
fi
