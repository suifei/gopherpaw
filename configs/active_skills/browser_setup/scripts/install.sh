#!/bin/bash
# Chromium 下载安装脚本
# 用法: bash install.sh [--cache-dir DIR] [--platform linux/darwin] [--arch amd64/arm64]

set -euo pipefail

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# 默认参数
CACHE_DIR="${HOME}/.gopherpaw/cache/chromium"
PLATFORM=""
ARCH_TYPE=""

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --cache-dir)
            CACHE_DIR="$2"
            shift 2
            ;;
        --platform)
            PLATFORM="$2"
            shift 2
            ;;
        --arch)
            ARCH_TYPE="$2"
            shift 2
            ;;
        -h|--help)
            echo "用法: $0 [选项]"
            echo ""
            echo "选项:"
            echo "  --cache-dir DIR    缓存目录 (默认: ~/.gopherpaw/cache/chromium)"
            echo "  --platform PLATFORM 平台 (linux/darwin，默认自动检测)"
            echo "  --arch ARCH        架构 (amd64/arm64，默认自动检测)"
            echo "  -h, --help         显示帮助"
            exit 0
            ;;
        *)
            log_error "未知参数: $1"
            exit 1
            ;;
    esac
done

# 检测系统信息
detect_system() {
    if [[ -z "$PLATFORM" ]]; then
        case "$(uname -s)" in
            Linux*)     PLATFORM="linux" ;;
            Darwin*)    PLATFORM="darwin" ;;
            *)          log_error "不支持的操作系统"; exit 1 ;;
        esac
    fi

    if [[ -z "$ARCH_TYPE" ]]; then
        case "$(uname -m)" in
            x86_64|amd64)   ARCH_TYPE="amd64" ;;
            aarch64|arm64)  ARCH_TYPE="arm64" ;;
            *)              log_error "不支持的架构: $(uname -m)"; exit 1 ;;
        esac
    fi

    log_info "平台: $PLATFORM"
    log_info "架构: $ARCH_TYPE"
}

# 获取下载 URL
get_download_url() {
    local base_url="https://www.googleapis.com/download/storage/v1/b/chromium-browser-snapshots"

    case "$PLATFORM" in
        linux)
            if [[ "$ARCH_TYPE" == "arm64" ]]; then
                echo "$base_url/Linux_arm64/1000000/chrome-linux.zip"
            else
                echo "$base_url/Linux_x64/1000000/chrome-linux.zip"
            fi
            ;;
        darwin)
            echo "$base_url/Mac/1000000/chrome-mac.zip"
            ;;
        *)
            log_error "不支持的平台: $PLATFORM"
            exit 1
            ;;
    esac
}

# 下载 Chromium
download_chromium() {
    local url="$1"
    local zip_file="$CACHE_DIR/chromium.zip"

    log_step "创建缓存目录: $CACHE_DIR"
    mkdir -p "$CACHE_DIR"

    log_step "下载 Chromium..."
    log_info "URL: $url"

    # 检查是否有 wget 或 curl
    if command -v wget &> /dev/null; then
        wget -O "$zip_file" "$url" || {
            log_error "下载失败"
            rm -f "$zip_file"
            exit 1
        }
    elif command -v curl &> /dev/null; then
        curl -L -o "$zip_file" "$url" || {
            log_error "下载失败"
            rm -f "$zip_file"
            exit 1
        }
    else
        log_error "需要 wget 或 curl 来下载 Chromium"
        exit 1
    fi

    log_info "下载完成: $zip_file"
}

# 解压 Chromium
extract_chromium() {
    local zip_file="$CACHE_DIR/chromium.zip"

    log_step "解压 Chromium..."

    # 检查是否有 unzip 命令
    if ! command -v unzip &> /dev/null; then
        log_error "需要 unzip 命令来解压文件"
        log_info "安装方法: sudo apt-get install unzip (Debian/Ubuntu)"
        exit 1
    fi

    # 清理旧文件
    rm -rf "$CACHE_DIR/chrome-linux"
    rm -rf "$CACHE_DIR/chrome-mac"

    # 解压
    unzip -q "$zip_file" -d "$CACHE_DIR" || {
        log_error "解压失败"
        exit 1
    }

    log_info "解压完成"
}

# 获取可执行文件路径
get_executable_path() {
    if [[ "$PLATFORM" == "linux" ]]; then
        echo "$CACHE_DIR/chrome-linux/chrome"
    elif [[ "$PLATFORM" == "darwin" ]]; then
        echo "$CACHE_DIR/chrome-mac/Chromium.app/Contents/MacOS/Chromium"
    fi
}

# 设置可执行权限
set_permissions() {
    local exec_path="$1"

    log_step "设置可执行权限..."
    chmod +x "$exec_path"
}

# 验证安装
verify_installation() {
    local exec_path="$1"

    log_step "验证安装..."

    if [[ ! -x "$exec_path" ]]; then
        log_error "验证失败: $exec_path 不可执行"
        exit 1
    fi

    local version
    version=$("$exec_path" --version 2>/dev/null || true)
    log_info "版本: $version"
}

# 创建配置文件
create_config() {
    local exec_path="$1"
    local config_file="${HOME}/.gopherpaw/browser.json"
    local config_dir

    config_dir=$(dirname "$config_file")
    mkdir -p "$config_dir"

    log_step "创建配置文件: $config_file"

    cat > "$config_file" << EOF
{
  "chrome_path": "$exec_path",
  "last_updated": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")",
  "auto_installed": true
}
EOF

    log_info "配置文件已创建"
}

# 打印环境变量设置指引
print_env_setup() {
    local exec_path="$1"

    echo ""
    log_step "环境变量设置"
    echo ""
    echo "临时设置（当前会话）:"
    echo "  export CHROME_BIN=\"$exec_path\""
    echo ""
    echo "永久设置（添加到 ~/.bashrc）:"
    echo "  echo 'export CHROME_BIN=\"$exec_path\"' >> ~/.bashrc"
    echo "  source ~/.bashrc"
    echo ""
    echo "永久设置（添加到 ~/.zshrc）:"
    echo "  echo 'export CHROME_BIN=\"$exec_path\"' >> ~/.zshrc"
    echo "  source ~/.zshrc"
    echo ""
}

# 打印容器环境指引
print_container_note() {
    echo ""
    log_step "容器环境注意事项"
    echo ""
    echo "如果在 Docker 容器中运行，需要设置:"
    echo "  export GOPHERPAW_RUNNING_IN_CONTAINER=1"
    echo ""
    echo "或确保启动时添加 --no-sandbox 参数"
    echo ""
}

# 主函数
main() {
    log_info "开始安装 Chromium..."

    detect_system
    local url
    url=$(get_download_url)
    download_chromium "$url"
    extract_chromium

    local exec_path
    exec_path=$(get_executable_path)
    set_permissions "$exec_path"
    verify_installation "$exec_path"
    create_config "$exec_path"

    echo ""
    log_info "安装完成!"
    echo ""
    log_info "可执行文件: $exec_path"
    print_env_setup "$exec_path"

    # 检查是否在容器中
    if [[ -f "/.dockerenv" ]] || grep -qE "(docker|kubepods)" /proc/1/cgroup 2>/dev/null; then
        print_container_note
    fi

    log_info "现在可以测试浏览器工具:"
    echo "  browser_use action=start headed=true"
}

# 执行主函数
main "$@"
