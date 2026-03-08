---
name: browser_setup
description: "浏览器环境配置技能 - 自动检测系统已安装的 Chrome/Chromium，或提供下载安装指引。支持 Linux/macOS/Windows 跨平台检测。"
version: "1.0"
author: "GopherPaw"
---

# 浏览器环境配置

当用户需要**配置浏览器环境**、**检测系统浏览器**、**解决浏览器找不到的问题**时，使用本技能。

## 功能概述

本技能帮助用户解决 `browser_use` 工具报告 "executable file not found" 的问题：

1. **自动检测**: 扫描系统常见浏览器路径（Chrome/Chromium/Edge）
2. **环境变量配置**: 指导设置 `CHROME_BIN` 环境变量
3. **容器环境支持**: Docker 环境特殊配置（--no-sandbox）
4. **安装指引**: 提供各平台安装 Chromium 的指引

## 检测系统浏览器

用户询问类似问题时进行检测：

- "检测系统中的 Chrome 或 Chromium"
- "系统有浏览器吗？"
- "找不到浏览器"
- "browser_use 报错找不到浏览器"

### 检测方法

使用以下命令检测系统浏览器：

```bash
# Linux/macOS
which google-chrome google-chrome-stable chromium chromium-browser

# 或使用 ls 检查常见路径
ls /usr/bin/google-chrome* /usr/bin/chromium* /snap/bin/chromium

# macOS
ls "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"

# Windows
where chrome.exe
```

### 检测脚本

可调用 shell_tool 运行检测脚本：

```bash
# 运行检测脚本
bash configs/active_skills/browser_setup/scripts/detect.sh
```

## 常见浏览器路径

### Linux
- `/usr/bin/google-chrome`
- `/usr/bin/google-chrome-stable`
- `/usr/bin/chromium`
- `/usr/bin/chromium-browser`
- `/usr/lib/chromium/chromium`
- `/snap/bin/chromium`

### macOS
- `/Applications/Google Chrome.app/Contents/MacOS/Google Chrome`
- `/Applications/Chromium.app/Contents/MacOS/Chromium`

### Windows
- `C:\Program Files\Google\Chrome\Application\chrome.exe`
- `C:\Program Files (x86)\Google\Chrome\Application\chrome.exe`

## 配置浏览器路径

### 方法 1: 环境变量（推荐）

```bash
# 临时设置（当前会话）
export CHROME_BIN="/usr/bin/google-chrome-stable"

# 永久设置（添加到 ~/.bashrc 或 ~/.zshrc）
echo 'export CHROME_BIN="/usr/bin/google-chrome-stable"' >> ~/.bashrc
source ~/.bashrc
```

### 方法 2: 配置文件

创建 `~/.gopherpaw/browser.json`：

```json
{
  "chrome_path": "/usr/bin/google-chrome-stable",
  "last_updated": "2025-03-08T10:30:00Z",
  "auto_installed": false
}
```

## 容器环境（Docker）

在 Docker 容器中运行时，需要特殊处理：

1. **检测容器环境**: 检查 `/.dockerenv` 文件或 `/proc/1/cgroup`
2. **添加启动参数**: `--no-sandbox` 和 `--disable-setuid-sandbox`

可通过环境变量显式指定：

```bash
export GOPHERPAW_RUNNING_IN_CONTAINER=1
```

## 安装 Chromium

### Linux (Debian/Ubuntu)

```bash
# Ubuntu/Debian
sudo apt-get update
sudo apt-get install -y chromium-browser

# 或安装 Chrome
wget https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb
sudo dpkg -i google-chrome-stable_current_amd64.deb
sudo apt-get install -f -y
```

### Linux (CentOS/RHEL)

```bash
# CentOS/RHEL
sudo yum install -y chromium
```

### macOS

```bash
# 使用 Homebrew
brew install --cask chromium
```

### Alpine Linux (Docker)

```bash
apk add --no-cache chromium
```

### Windows

从官网下载安装：
- Chrome: https://www.google.com/chrome/
- Chromium: https://www.chromium.org/getting-involved/download-chromium

## 验证配置

运行验证脚本检查浏览器是否可用：

```bash
bash configs/active_skills/browser_setup/scripts/verify.sh
```

或手动验证：

```bash
# 检查环境变量
echo $CHROME_BIN

# 检查可执行文件
$CHROME_BIN --version

# 或直接调用
google-chrome --version
chromium --version
```

## 测试浏览器工具

配置完成后，测试 browser_use 工具：

```json
// 启动浏览器
{"action": "start", "headed": true}

// 打开网页
{"action": "open", "url": "https://example.com"}

// 截图测试
{"action": "screenshot", "path": "/tmp/test.png"}
```

## 常见问题

### Q: 找不到浏览器怎么办？

A: 按优先级尝试：
1. 设置 `CHROME_BIN` 环境变量指向浏览器路径
2. 安装系统浏览器包
3. 使用便携版 Chromium

### Q: Docker 环境中浏览器启动失败？

A: 确保设置了：
- 环境变量 `GOPHERPAW_RUNNING_IN_CONTAINER=1`
- 或确保 `/.dockerenv` 文件存在
- 浏览器会自动添加 `--no-sandbox` 参数

### Q: 权限问题？

A: 检查浏览器可执行文件权限：
```bash
chmod +x /usr/bin/chromium
```
