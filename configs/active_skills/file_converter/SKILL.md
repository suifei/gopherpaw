---
name: file_converter
description: "文件格式转换工具。支持 JSON ↔ YAML ↔ TOML ↔ CSV 互转。当用户需要转换配置文件、数据格式时使用。"
metadata:
  {
    "copaw":
      {
        "emoji": "🔄",
        "requires": {"runtime": "bun", "packages": ["yaml", "toml", "csv-parse", "csv-stringify"]}
      }
  }
---

# 文件格式转换 Skill

在不同文件格式之间进行转换。

## 支持的格式

- **JSON**：标准 JSON 格式
- **YAML**：YAML 配置文件
- **TOML**：TOML 配置文件
- **CSV**：逗号分隔值

## 转换矩阵

| 从 \ 到 | JSON | YAML | TOML | CSV |
|---------|------|------|------|-----|
| JSON    | -    | ✅   | ✅   | ✅  |
| YAML    | ✅   | -    | ✅   | ✅  |
| TOML    | ✅   | ✅   | -    | ✅  |
| CSV     | ✅   | ✅   | ✅   | -   |

## 安装依赖

```bash
cd configs/active_skills/file_converter
bun install yaml toml csv-parse csv-stringify
```

## 使用方法

### JSON → YAML
```bash
bun scripts/json2yaml.js <input.json> <output.yaml>
```

### YAML → JSON
```bash
bun scripts/yaml2json.js <input.yaml> <output.json>
```

### JSON → CSV
```bash
bun scripts/json2csv.js <input.json> <output.csv>
```

### CSV → JSON
```bash
bun scripts/csv2json.js <input.csv> <output.json>
```

### 自动检测格式
```bash
bun scripts/convert.js <input> <output>
```

## 示例场景

- 用户说："把这个 JSON 转成 YAML"
- 用户说："转换配置文件格式"
- 用户说："把 CSV 导出为 JSON"
- 用户说："这个 TOML 文件转成 JSON"

## 特性

- ✅ 自动格式检测
- ✅ 保持数据结构
- ✅ 支持嵌套对象
- ✅ 批量转换
