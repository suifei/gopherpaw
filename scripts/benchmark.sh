#!/bin/bash
# GopherPaw 性能基准测试脚本
# 用途：运行全面的性能基准测试并生成报告

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
REPORT_DIR="benchmark_reports"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
REPORT_FILE="${REPORT_DIR}/benchmark_${TIMESTAMP}.txt"
JSON_REPORT="${REPORT_DIR}/benchmark_${TIMESTAMP}.json"

# 创建报告目录
mkdir -p "$REPORT_DIR"

echo -e "${BLUE}======================================${NC}"
echo -e "${BLUE}GopherPaw 性能基准测试${NC}"
echo -e "${BLUE}======================================${NC}"
echo ""
echo "开始时间: $(date)"
echo "报告文件: $REPORT_FILE"
echo ""

# 函数：运行基准测试并记录结果
run_benchmark() {
    local package=$1
    local name=$2
    local pattern=$3
    
    echo -e "${YELLOW}[测试] $name${NC}"
    echo "========================================" >> "$REPORT_FILE"
    echo "测试: $name" >> "$REPORT_FILE"
    echo "时间: $(date)" >> "$REPORT_FILE"
    echo "========================================" >> "$REPORT_FILE"
    
    # 运行基准测试
    go test -bench="$pattern" -benchmem -benchtime=3s "$package" 2>&1 | tee -a "$REPORT_FILE"
    
    echo "" >> "$REPORT_FILE"
    echo -e "${GREEN}[完成] $name${NC}"
    echo ""
}

# 函数：运行所有基准测试
run_all_benchmarks() {
    echo -e "${BLUE}开始运行所有基准测试...${NC}"
    echo ""
    
    # 1. Agent 包基准测试
    if [ -d "internal/agent" ]; then
        run_benchmark "./internal/agent" "Agent 包基准测试" "."
    fi
    
    # 2. Channels 包基准测试
    if [ -d "internal/channels" ]; then
        run_benchmark "./internal/channels" "Channels 包基准测试" "."
    fi
    
    # 3. Config 包基准测试
    if [ -d "internal/config" ]; then
        run_benchmark "./internal/config" "Config 包基准测试" "."
    fi
    
    # 4. LLM 包基准测试
    if [ -d "internal/llm" ]; then
        run_benchmark "./internal/llm" "LLM 包基准测试" "."
    fi
    
    # 5. MCP 包基准测试
    if [ -d "internal/mcp" ]; then
        run_benchmark "./internal/mcp" "MCP 包基准测试" "."
    fi
    
    # 6. Memory 包基准测试
    if [ -d "internal/memory" ]; then
        run_benchmark "./internal/memory" "Memory 包基准测试" "."
    fi
    
    # 7. Runtime 包基准测试
    if [ -d "internal/runtime" ]; then
        run_benchmark "./internal/runtime" "Runtime 包基准测试" "."
    fi
    
    # 8. Scheduler 包基准测试
    if [ -d "internal/scheduler" ]; then
        run_benchmark "./internal/scheduler" "Scheduler 包基准测试" "."
    fi
    
    # 9. Skills 包基准测试
    if [ -d "internal/skills" ]; then
        run_benchmark "./internal/skills" "Skills 包基准测试" "."
    fi
    
    # 10. Tools 包基准测试
    if [ -d "internal/tools" ]; then
        run_benchmark "./internal/tools" "Tools 包基准测试" "."
    fi
    
    # 11. Logger 包基准测试
    if [ -d "pkg/logger" ]; then
        run_benchmark "./pkg/logger" "Logger 包基准测试" "."
    fi
}

# 函数：生成 JSON 报告
generate_json_report() {
    echo -e "${YELLOW}生成 JSON 报告...${NC}"
    
    # 运行基准测试并输出为 JSON 格式
    go test -bench=. -benchmem -json ./... > "$JSON_REPORT" 2>&1 || true
    
    echo -e "${GREEN}JSON 报告已生成: $JSON_REPORT${NC}"
    echo ""
}

# 函数：分析内存使用
analyze_memory() {
    echo -e "${YELLOW}分析内存使用...${NC}"
    echo "========================================" >> "$REPORT_FILE"
    echo "内存分析" >> "$REPORT_FILE"
    echo "========================================" >> "$REPORT_FILE"
    
    # 运行内存分析
    for pkg in internal/agent internal/channels internal/config internal/llm internal/mcp internal/memory internal/runtime internal/scheduler internal/skills internal/tools pkg/logger; do
        if [ -d "$pkg" ]; then
            echo "分析 $pkg..." >> "$REPORT_FILE"
            go test -bench=. -benchmem -memprofile=/tmp/mem.prof "$pkg" 2>&1 >> "$REPORT_FILE" || true
            if [ -f /tmp/mem.prof ]; then
                echo "Top 10 内存分配:" >> "$REPORT_FILE"
                go tool pprof -top -lines /tmp/mem.prof 2>&1 | head -20 >> "$REPORT_FILE" || true
                echo "" >> "$REPORT_FILE"
                rm /tmp/mem.prof
            fi
        fi
    done
    
    echo -e "${GREEN}内存分析完成${NC}"
    echo ""
}

# 函数：分析 CPU 使用
analyze_cpu() {
    echo -e "${YELLOW}分析 CPU 使用...${NC}"
    echo "========================================" >> "$REPORT_FILE"
    echo "CPU 分析" >> "$REPORT_FILE"
    echo "========================================" >> "$REPORT_FILE"
    
    # 运行 CPU 分析
    for pkg in internal/agent internal/channels internal/config internal/llm internal/mcp internal/memory internal/runtime internal/scheduler internal/skills internal/tools pkg/logger; do
        if [ -d "$pkg" ]; then
            echo "分析 $pkg..." >> "$REPORT_FILE"
            go test -bench=. -cpuprofile=/tmp/cpu.prof "$pkg" 2>&1 >> "$REPORT_FILE" || true
            if [ -f /tmp/cpu.prof ]; then
                echo "Top 10 CPU 消耗:" >> "$REPORT_FILE"
                go tool pprof -top -lines /tmp/cpu.prof 2>&1 | head -20 >> "$REPORT_FILE" || true
                echo "" >> "$REPORT_FILE"
                rm /tmp/cpu.prof
            fi
        fi
    done
    
    echo -e "${GREEN}CPU 分析完成${NC}"
    echo ""
}

# 函数：运行竞态检测
check_race() {
    echo -e "${YELLOW}运行竞态检测...${NC}"
    echo "========================================" >> "$REPORT_FILE"
    echo "竞态检测" >> "$REPORT_FILE"
    echo "========================================" >> "$REPORT_FILE"
    
    go test -race ./... 2>&1 | tee -a "$REPORT_FILE" || true
    
    echo "" >> "$REPORT_FILE"
    echo -e "${GREEN}竞态检测完成${NC}"
    echo ""
}

# 函数：生成摘要
generate_summary() {
    echo -e "${YELLOW}生成测试摘要...${NC}"
    echo "" >> "$REPORT_FILE"
    echo "========================================" >> "$REPORT_FILE"
    echo "测试摘要" >> "$REPORT_FILE"
    echo "========================================" >> "$REPORT_FILE"
    echo "完成时间: $(date)" >> "$REPORT_FILE"
    echo "" >> "$REPORT_FILE"
    
    # 统计测试数量
    local total_tests=$(grep -c "Benchmark" "$REPORT_FILE" || echo "0")
    echo "总基准测试数: $total_tests" >> "$REPORT_FILE"
    
    # 统计覆盖率
    echo "" >> "$REPORT_FILE"
    echo "测试覆盖率:" >> "$REPORT_FILE"
    go test -short -cover ./... 2>&1 | grep "coverage:" >> "$REPORT_FILE" || true
    
    echo "" >> "$REPORT_FILE"
    echo -e "${GREEN}摘要已生成${NC}"
    echo ""
}

# 主函数
main() {
    # 检查参数
    case "${1:-all}" in
        "all")
            run_all_benchmarks
            generate_json_report
            ;;
        "quick")
            # 快速测试（只运行主要包）
            run_benchmark "./internal/agent" "Agent 包" "."
            run_benchmark "./internal/channels" "Channels 包" "."
            run_benchmark "./internal/tools" "Tools 包" "."
            ;;
        "memory")
            analyze_memory
            ;;
        "cpu")
            analyze_cpu
            ;;
        "race")
            check_race
            ;;
        "json")
            generate_json_report
            ;;
        *)
            echo "用法: $0 [all|quick|memory|cpu|race|json]"
            echo ""
            echo "选项:"
            echo "  all     - 运行所有基准测试（默认）"
            echo "  quick   - 快速测试（仅主要包）"
            echo "  memory  - 内存分析"
            echo "  cpu     - CPU 分析"
            echo "  race    - 竞态检测"
            echo "  json    - 生成 JSON 报告"
            exit 1
            ;;
    esac
    
    # 生成摘要
    generate_summary
    
    echo -e "${GREEN}======================================${NC}"
    echo -e "${GREEN}性能基准测试完成${NC}"
    echo -e "${GREEN}======================================${NC}"
    echo ""
    echo "报告文件:"
    echo "  - 文本报告: $REPORT_FILE"
    if [ -f "$JSON_REPORT" ]; then
        echo "  - JSON 报告: $JSON_REPORT"
    fi
    echo ""
    echo "查看报告:"
    echo "  cat $REPORT_FILE"
    echo ""
}

# 运行主函数
main "$@"
