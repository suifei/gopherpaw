---
name: file_reader
description: 文件读取说明
---

# 文件读取技能

Agent 可使用此技能读取和处理本地文件。

## 能力说明

- **read_file**：读取指定路径的文件内容
- **grep_search**：按模式搜索文件内容
- **glob_search**：按 glob 模式查找文件

## 使用规范

1. 读取前确认文件路径在工作目录内
2. 大文件建议分段读取
3. 二进制文件不可直接读取

## 工具说明

- `read_file`: 读取文件，参数 `path` 为文件路径
- `grep_search`: 正则搜索，参数 `pattern` 为搜索模式，`path` 为目录
- `glob_search`: 通配符查找，参数 `pattern` 如 `*.go`
