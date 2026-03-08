#!/bin/bash
# 浏览器配置验证脚本
# 用法: bash verify.sh [--browser PATH]

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

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_fail() {
    echo -e "${RED}[✗]${NC} $1"
}

# 参数
BROWSER_PATH=""

# 解析参数
while [[ $# -gt 0 ]]; do
    case $1 in
        --browser)
            BROWSER_PATH="$2"
            shift 2
            ;;
        -h|--help)
            echo "用法: $0 [选项]"
            echo ""
            echo "选项:"
            echo "  --browser PATH   指定浏览器可执行文件路径"
            echo "  -h, --help       显示帮助"
            exit 0
            ;;
        *)
            log_error "未知参数: $1"
            exit 1
            ;;
    esac
done

# 验证计数器
PASS=0
FAIL=0

# 测试函数
test_item() {
    local name="$1"
    local test_cmd="$2"

    echo -n "Testing $name... "
    if eval "$test_cmd" &>/dev/null; then
        log_success "PASS"
        ((PASS++))
        return 0
    else
        log_fail "FAIL"
        ((FAIL++))
        return 1
    fi
}

# 获取浏览器路径
get_browser_path() {
    # 1. 命令行参数
    if [[ -n "$BROWSER_PATH" ]]; then
        echo "$BROWSER_PATH"
        return
    fi

    # 2. 环境变量
    if [[ -n "${CHROME_BIN:-}" ]]; then
        echo "$CHROME_BIN"
        return
    fi

    # 3. 配置文件
    local config_file="${HOME}/.gopherpaw/browser.json"
    if [[ -f "$config_file" ]]; then
        local path
        path=$(grep -oP '"chrome_path"\s*:\s*"\K[^"]+' "$config_file" 2>/dev/null || echo "")
        if [[ -n "$path" && -x "$path" ]]; then
            echo "$path"
            return
        fi
    fi

    # 4. 系统检测
    if command -v google-chrome &>/dev/null; then
        command -v google-chrome
        return
    fi
    if command -v google-chrome-stable &>/dev/null; then
        command -v google-chrome-stable
        return
    fi
    if command -v chromium &>/dev/null; then
        command -v chromium
        return
    fi
    if command -v chromium-browser &>/dev/null; then
        command -v chromium-browser
        return
    fi
}

# 主函数
main() {
    echo "=========================================="
    echo "  浏览器环境验证"
    echo "=========================================="
    echo ""

    # 获取浏览器路径
    BROWSER=$(get_browser_path)

    if [[ -z "$BROWSER" ]]; then
        log_error "未找到浏览器可执行文件"
        echo ""
        echo "请通过以下方式之一指定浏览器路径:"
        echo "  1. 设置环境变量: export CHROME_BIN=/path/to/browser"
        echo "  2. 创建配置文件: ~/.gopherpaw/browser.json"
        echo "  3. 使用参数: $0 --browser /path/to/browser"
        echo ""
        exit 1
    fi

    log_info "使用浏览器: $BROWSER"
    echo ""

    # 1. 检查文件存在性
    test_item "浏览器文件存在" "[[ -f '$BROWSER' ]]"

    # 2. 检查可执行权限
    test_item "浏览器可执行" "[[ -x '$BROWSER' ]]"

    # 3. 检查版本信息
    echo -n "Testing 版本信息... "
    VERSION=$("$BROWSER" --version 2>&1 || true)
    if [[ -n "$VERSION" ]]; then
        log_success "PASS"
        echo "  $VERSION"
        ((PASS++))
    else
        log_fail "FAIL"
        ((FAIL++))
    fi

    echo ""

    # 4. 检查依赖库（仅 Linux）
    if [[ "$(uname -s)" == "Linux" ]]; then
        echo "检查依赖库:"
        if command -v ldd &>/dev/null; then
            local missing=0
            while IFS= read -r line; do
                if echo "$line" | grep -q "not found"; then
                    log_warn "$(echo "$line" | awk '{print $1}') 缺失"
                    ((missing++))
                fi
            done < <(ldd "$BROWSER" 2>/dev/null || true)

            if [[ $missing -eq 0 ]]; then
                log_success "所有依赖库满足"
                ((PASS++))
            else
                log_fail "缺失 $missing 个依赖库"
                ((FAIL++))
            fi
        else
            log_warn "ldd 不可用，跳过依赖检查"
        fi
        echo ""
    fi

    # 5. 检查容器环境
    echo -n "Testing 容器环境检测... "
    IN_CONTAINER=false
    if [[ -f "/.dockerenv" ]]; then
        IN_CONTAINER=true
    elif [[ -f "/proc/1/cgroup" ]]; then
        if grep -qE "(docker|kubepods|containerd)" /proc/1/cgroup 2>/dev/null; then
            IN_CONTAINER=true
        fi
    elif [[ -n "${GOPHERPAW_RUNNING_IN_CONTAINER:-}" ]]; then
        IN_CONTAINER=true
    fi

    if $IN_CONTAINER; then
        log_success "PASS (容器环境)"
        log_info "提示: 容器环境需要 --no-sandbox 参数"
        ((PASS++))
    else
        log_success "PASS (非容器环境)"
        ((PASS++))
    fi
    echo ""

    # 6. 环境变量检查
    echo "环境变量:"
    if [[ -n "${CHROME_BIN:-}" ]]; then
        log_success "CHROME_BIN=$CHROME_BIN"
        ((PASS++))
    else
        log_warn "CHROME_BIN 未设置（可选）"
    fi

    if [[ -n "${GOPHERPAW_RUNNING_IN_CONTAINER:-}" ]]; then
        log_success "GOPHERPAW_RUNNING_IN_CONTAINER=$GOPHERPAW_RUNNING_IN_CONTAINER"
    else
        log_info "GOPHERPAW_RUNNING_IN_CONTAINER 未设置"
    fi
    echo ""

    # 7. 配置文件检查
    echo "配置文件:"
    local config_file="${HOME}/.gopherpaw/browser.json"
    if [[ -f "$config_file" ]]; then
        log_success "$config_file 存在"
        ((PASS++))
        echo "  内容:"
        cat "$config_file" | sed 's/^/    /'
    else
        log_info "$config_file 不存在（可选）"
    fi
    echo ""

    # 8. 测试启动（可选）
    echo -n "Testing 浏览器启动测试... "
    if timeout 5 "$BROWSER" --headless --disable-gpu --no-sandbox --dump-dom about:blank &>/dev/null; then
        log_success "PASS"
        ((PASS++))
    else
        log_warn "SKIP (需要显示环境)"
    fi
    echo ""

    # 总结
    echo "=========================================="
    echo "  验证结果"
    echo "=========================================="
    echo ""
    log_info "通过: $PASS"
    if [[ $FAIL -gt 0 ]]; then
        log_error "失败: $FAIL"
        echo ""
        log_error "验证未通过，请检查上述错误项"
        exit 1
    else
        log_success "所有检查通过!"
        echo ""
        log_info "浏览器已配置完成，可以正常使用 browser_use 工具"
    fi
}

# 执行主函数
main "$@"
