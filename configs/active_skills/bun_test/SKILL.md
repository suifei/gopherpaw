---
name: bun_test
description: "测试 Bun 运行时功能。当用户要求测试 Bun、验证 Bun 安装、或执行 JavaScript 代码时使用。"
metadata:
  {
    "copaw":
      {
        "emoji": "🧪",
        "requires": {"runtime": "bun"}
      }
  }
---

# Bun 运行时测试 Skill

这个 Skill 用于测试 Bun 运行时的各种功能。

## 功能测试

### 1. 基础测试
执行 `bun --version` 验证 Bun 安装。

### 2. JavaScript 执行
使用 `execute_shell_command` 执行 JS 脚本：
```bash
bun configs/active_skills/bun_test/scripts/hello.js
```

### 3. Bun 特有 API
测试 Bun 特有功能（如 Bun.version、Bun.file 等）。

### 4. Node.js 兼容性
测试 Node.js 兼容 API（如 fs、path 等）。

## 使用场景

- 用户说："测试 Bun 运行时"
- 用户说："验证 Bun 是否正常工作"
- 用户说："执行 JavaScript 代码"
- 用户说："运行 JS 脚本"

## 注意事项

- Bun 路径：`/home/suifei/.bun/bin/bun`
- 脚本目录：`configs/active_skills/bun_test/scripts/`
- 使用绝对路径避免路径问题
