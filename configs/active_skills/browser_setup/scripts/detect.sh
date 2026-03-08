#!/bin/bash
# 跨平台浏览器检测脚本
# 用法: bash detect.sh
# 输出: 找到的浏览器路径或空行

set -euo pipefail

# 颜色输出
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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

# 获取操作系统类型
get_os_type() {
    case "$(uname -s)" in
        Linux*)     echo "linux" ;;
        Darwin*)    echo "darwin" ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *)          echo "unknown" ;;
    esac
}

# 获取架构类型
get_arch_type() {
    case "$(uname -m)" in
        x86_64|amd64)   echo "amd64" ;;
        i386|i686)      echo "386" ;;
        aarch64|arm64)  echo "arm64" ;;
        armv7l)         echo "arm" ;;
        *)              echo "unknown" ;;
    esac
}

# 检测 Linux 浏览器
detect_linux_browsers() {
    local candidates=(
        "/usr/bin/google-chrome"
        "/usr/bin/google-chrome-stable"
        "/usr/bin/google-chrome-beta"
        "/usr/bin/google-chrome-dev"
        "/usr/bin/chromium"
        "/usr/bin/chromium-browser"
        "/usr/lib/chromium/chromium"
        "/snap/bin/chromium"
        "/opt/google/chrome/google-chrome"
        "/opt/chromium.org/chromium/chrome"
    )

    for path in "${candidates[@]}"; do
        if [[ -x "$path" ]]; then
            echo "$path"
            return 0
        fi
    done

    # 尝试 which 命令
    local browser_names=("google-chrome" "google-chrome-stable" "chromium" "chromium-browser")
    for name in "${browser_names[@]}"; do
        local found
        found=$(command -v "$name" 2>/dev/null || true)
        if [[ -n "$found" && -x "$found" ]]; then
            echo "$found"
            return 0
        fi
    done

    return 1
}

# 检测 macOS 浏览器
detect_darwin_browsers() {
    local candidates=(
        "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
        "/Applications/Chromium.app/Contents/MacOS/Chromium"
        "/Applications/Google Chrome Beta.app/Contents/MacOS/Google Chrome Beta"
        "/Applications/Google Chrome Dev.app/Contents/MacOS/Google Chrome Dev"
        "/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge"
    )

    for path in "${candidates[@]}"; do
        if [[ -x "$path" ]]; then
            echo "$path"
            return 0
        fi
    done

    return 1
}

# 检测 Windows 浏览器
detect_windows_browsers() {
    local candidates=(
        "$LOCALAPPDATA\\Google\\Chrome\\Application\\chrome.exe"
        "$PROGRAMFILES\\Google\\Chrome\\Application\\chrome.exe"
        "$PROGRAMFILES(X86)\\Google\\Chrome\\Application\\chrome.exe"
        "$PROGRAMFILES\\Microsoft\\Edge\\Application\\msedge.exe"
    )

    for path in "${candidates[@]}"; do
        # 展开环境变量
        local expanded="${path//\%/%%}"
        if [[ -f "$expanded" ]]; then
            echo "$expanded"
            return 0
        fi
    done

    # 尝试 where 命令
    local found
    found=$(where chrome.exe 2>/dev/null | head -n1 || true)
    if [[ -n "$found" && -f "$found" ]]; then
        echo "$found"
        return 0
    fi

    return 1
}

# 检测容器环境
detect_container() {
    if [[ -f "/.dockerenv" ]]; then
        echo "docker"
        return 0
    fi

    if [[ -f "/proc/1/cgroup" ]]; then
        local cgroup
        cgroup=$(cat /proc/1/cgroup 2>/dev/null || true)
        if echo "$cgroup" | grep -qE "(docker|kubepods|containerd)"; then
            echo "docker"
            return 0
        fi
    fi

    if [[ -n "${GOPHERPAW_RUNNING_IN_CONTAINER:-}" ]]; then
        echo "explicit"
        return 0
    fi

    echo "none"
    return 1
}

# 主函数
main() {
    log_info "开始检测浏览器环境..."

    # 获取系统信息
    local os_type arch_type
    os_type=$(get_os_type)
    arch_type=$(get_arch_type)

    log_info "操作系统: $os_type"
    log_info "架构: $arch_type"

    # 检测容器环境
    if [[ "$os_type" == "linux" ]]; then
        local container
        container=$(detect_container)
        if [[ "$container" != "none" ]]; then
            log_warn "检测到容器环境 ($container)"
        fi
    fi

    # 检查环境变量
    if [[ -n "${CHROME_BIN:-}" ]]; then
        if [[ -x "$CHROME_BIN" ]]; then
            log_info "使用环境变量 CHROME_BIN: $CHROME_BIN"
            echo "$CHROME_BIN"
            exit 0
        else
            log_warn "CHROME_BIN 设置为 $CHROME_BIN，但文件不存在或不可执行"
        fi
    fi

    # 根据操作系统检测
    local browser_path=""
    case "$os_type" in
        linux)
            browser_path=$(detect_linux_browsers)
            ;;
        darwin)
            browser_path=$(detect_darwin_browsers)
            ;;
        windows)
            browser_path=$(detect_windows_browsers)
            ;;
        *)
            log_error "不支持的操作系统: $os_type"
            exit 1
            ;;
    esac

    if [[ -n "$browser_path" && -x "$browser_path" ]]; then
        log_info "找到浏览器: $browser_path"

        # 尝试获取版本信息
        local version
        version=$("$browser_path" --version 2>/dev/null || true)
        if [[ -n "$version" ]]; then
            log_info "版本: $version"
        fi

        echo "$browser_path"
        exit 0
    else
        log_error "未找到可用的浏览器"

        # 提供安装建议
        echo ""
        log_info "安装建议:"
        case "$os_type" in
            linux)
                echo "  Ubuntu/Debian: sudo apt-get install -y chromium-browser"
                echo "  Alpine: apk add --no-cache chromium"
                echo "  CentOS/RHEL: sudo yum install -y chromium"
                ;;
            darwin)
                echo "  使用 Homebrew: brew install --cask chromium"
                ;;
            windows)
                echo "  从 https://www.google.com/chrome/ 下载安装"
                ;;
        esac

        exit 1
    fi
}

# 执行主函数
main "$@"
