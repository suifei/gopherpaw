---
name: data_processor
description: "使用 Bun 处理 JSON 数据、文件转换、数据清洗等任务。当用户需要处理 JSON、CSV、数据转换时使用。"
metadata:
  {
    "copaw":
      {
        "emoji": "📊",
        "requires": {"runtime": "bun"}
      }
  }
---

# 数据处理 Skill

使用 Bun 的高性能 JavaScript 运行时处理各种数据任务。

## 功能

### 1. JSON 数据处理
- 读取、解析、修改 JSON 文件
- JSON 格式化和验证
- JSON 转换（如 JSON → CSV）

### 2. 文件操作
- 读取文本文件
- 写入文件
- 文件内容处理

### 3. 数据转换
- CSV ↔ JSON 转换
- 数据清洗和过滤
- 数据聚合和统计

## 使用方法

### 处理 JSON 文件
```bash
bun configs/active_skills/data_processor/scripts/process_json.js <input.json> <output.json>
```

### 格式化 JSON
```bash
bun configs/active_skills/data_processor/scripts/format_json.js <input.json>
```

## 优势

- **高性能**：Bun 的执行速度比 Node.js 快 3-4 倍
- **内置 API**：Bun.file()、Bun.write() 等特有 API
- **TypeScript 支持**：可以直接运行 .ts 文件
- **Node.js 兼容**：完全兼容 Node.js API

## 示例场景

- 用户说："处理这个 JSON 文件"
- 用户说："把这个 CSV 转成 JSON"
- 用户说："格式化这个 JSON"
- 用户说："清洗这批数据"
