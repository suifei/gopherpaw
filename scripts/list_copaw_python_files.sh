#!/usr/bin/env bash
# 列出 copaw-source/ 下所有 Python 源码文件，写入 docs/copaw_python_files.txt
# 用法: 在项目根目录执行 ./scripts/list_copaw_python_files.sh 或 bash scripts/list_copaw_python_files.sh

set -e
# 项目根目录：以脚本所在目录的上一级为准，或传入第一个参数
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
ROOT="${1:-$(cd "$SCRIPT_DIR/.." && pwd)}"
if [[ "$ROOT" != /* ]]; then
  ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
fi
OUT="${ROOT}/docs/copaw_python_files.txt"
SRC="${ROOT}/copaw-source"

if [[ ! -d "$SRC" ]]; then
  echo "错误: 未找到目录 $SRC" >&2
  exit 1
fi

echo "# CoPaw Python 源码文件列表 (自动生成于 $(date -u +%Y-%m-%dT%H:%M:%SZ))" > "$OUT"
echo "# 生成命令: $0" >> "$OUT"
echo "" >> "$OUT"

# 递归查找 .py，排除 __pycache__，按路径排序
find "$SRC" -type f -name "*.py" | grep -v __pycache__ | sort >> "$OUT"

COUNT=$(grep -c '\.py$' "$OUT" || true)
echo "已写入 $COUNT 个 .py 文件到 $OUT"
