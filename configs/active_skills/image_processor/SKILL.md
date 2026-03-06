---
name: image_processor
description: "使用 Bun + Sharp 处理图片。支持调整大小、裁剪、格式转换、添加水印等操作。当用户需要处理图片、转换图片格式、压缩图片时使用。"
metadata:
  {
    "copaw":
      {
        "emoji": "🖼️",
        "requires": {"runtime": "bun", "packages": ["sharp"]}
      }
  }
---

# 图片处理 Skill

使用 Bun + Sharp 进行高性能图片处理。

## 功能

### 1. 图片转换
- 调整大小（resize）
- 裁剪（crop）
- 旋转（rotate）
- 格式转换（JPEG/PNG/WebP/AVIF）

### 2. 图片优化
- 压缩图片
- 质量调整
- 元数据移除

### 3. 图片编辑
- 添加水印
- 模糊效果
- 灰度转换

## 安装依赖

```bash
cd configs/active_skills/image_processor
bun install sharp
```

## 使用方法

### 调整图片大小
```bash
bun scripts/resize.js <input.jpg> <width> <height> <output.jpg>
```

### 转换图片格式
```bash
bun scripts/convert.js <input.png> <output.webp>
```

### 添加水印
```bash
bun scripts/watermark.js <input.jpg> <watermark.png> <output.jpg>
```

### 压缩图片
```bash
bun scripts/compress.js <input.jpg> <output.jpg> [quality]
```

## 示例场景

- 用户说："把这张图片缩小到 800x600"
- 用户说："把这个 PNG 转成 WebP"
- 用户说："压缩这张图片"
- 用户说："给这张图片加水印"

## 支持的格式

- **输入**：JPEG, PNG, WebP, AVIF, GIF, SVG, TIFF
- **输出**：JPEG, PNG, WebP, AVIF, GIF

## 性能优势

- Bun + Sharp 比 Node.js + Sharp 快 2-3x
- 内存占用更低
- 支持流式处理大图片
