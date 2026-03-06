---
name: data_analyzer
description: "使用 Bun + SQL.js 进行数据分析。支持 SQL 查询、数据统计、聚合分析、数据可视化准备。当用户需要分析数据、执行 SQL 查询、生成统计报告时使用。"
metadata:
  {
    "copaw":
      {
        "emoji": "📊",
        "requires": {"runtime": "bun", "packages": ["sql.js"]}
      }
  }
---

# 数据分析 Skill

使用 Bun + SQL.js 进行高性能数据分析。

## 功能

### 1. SQL 查询
- 标准 SQL 支持
- JOIN 查询
- 子查询
- 聚合函数

### 2. 数据统计
- 基础统计（COUNT/SUM/AVG/MIN/MAX）
- 分组统计
- 排序和筛选

### 3. 数据处理
- CSV 导入
- JSON 导入
- 结果导出

## 安装依赖

```bash
cd configs/active_skills/data_analyzer
bun install sql.js
```

## 使用方法

### 从 CSV 创建数据库
```bash
bun scripts/csv_to_db.js <data.csv> <output.db>
```

### 执行 SQL 查询
```bash
bun scripts/query.js <database.db> <sql_query>
```

### 数据统计
```bash
bun scripts/stats.js <database.db> <table_name>
```

### 生成报告
```bash
bun scripts/report.js <database.db> <output.json>
```

## 示例场景

- 用户说："分析这份数据"
- 用户说："执行 SQL 查询"
- 用户说："生成统计报告"
- 用户说："数据聚合分析"

## 支持的 SQL 功能

- ✅ SELECT / INSERT / UPDATE / DELETE
- ✅ JOIN / LEFT JOIN / RIGHT JOIN
- ✅ GROUP BY / HAVING
- ✅ ORDER BY / LIMIT
- ✅ 聚合函数（COUNT/SUM/AVG/MIN/MAX）
- ✅ 子查询

## 性能优势

- **内存数据库**：SQL.js 在内存中运行
- **快速查询**：比传统数据库快 10-100x（适合中小数据集）
- **零配置**：无需安装数据库服务器

## 限制

- 数据量：< 100MB（内存限制）
- 并发：单线程（适合分析任务）
- 持久化：需手动保存到文件
