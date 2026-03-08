# 浏览器环境配置指南

本文档详细说明如何为 GopherPaw 配置浏览器环境。

## 目录

- [快速开始](#快速开始)
- [检测系统浏览器](#检测系统浏览器)
- [安装浏览器](#安装浏览器)
- [配置路径](#配置路径)
- [容器环境](#容器环境)
- [故障排除](#故障排除)

## 快速开始

### 一键检测和配置

```bash
# 检测系统中的浏览器
bash configs/active_skills/browser_setup/scripts/detect.sh

# 验证浏览器配置
bash configs/active_skills/browser_setup/scripts/verify.sh
```

## 检测系统浏览器

### 使用 detect.sh 脚本

```bash
bash configs/active_skills/browser_setup/scripts/detect.sh
```

输出示例：
```
[INFO] 开始检测浏览器环境...
[INFO] 操作系统: linux
[INFO] 架构: amd64
[INFO] 找到浏览器: /usr/bin/google-chrome-stable
[INFO] 版本: Google Chrome 120.0.6099.109
/usr/bin/google-chrome-stable
```

### 手动检测

#### Linux

```bash
# 检查常见路径
ls -la /usr/bin/google-chrome*
ls -la /usr/bin/chromium*

# 使用 which 命令
which google-chrome google-chrome-stable chromium

# 检查 snap 安装的版本
ls -la /snap/bin/chromium
```

#### macOS

```bash
# 检查 Applications 目录
ls -la "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
ls -la "/Applications/Chromium.app/Contents/MacOS/Chromium"
```

#### Windows

```cmd
REM 使用 where 命令
where chrome.exe

REM 检查常见路径
dir "C:\Program Files\Google\Chrome\Application\chrome.exe"
dir "C:\Program Files (x86)\Google\Chrome\Application\chrome.exe"
```

## 安装浏览器

### Linux

#### Ubuntu/Debian

```bash
# 安装 Chromium
sudo apt-get update
sudo apt-get install -y chromium-browser

# 或安装 Google Chrome
wget https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb
sudo dpkg -i google-chrome-stable_current_amd64.deb
sudo apt-get install -f -y
```

#### Alpine Linux (Docker)

```bash
apk add --no-cache chromium
```

#### CentOS/RHEL/Fedora

```bash
# CentOS/RHEL
sudo yum install -y chromium

# Fedora
sudo dnf install -y chromium
```

#### Arch Linux

```bash
sudo pacman -S chromium
```

### macOS

```bash
# 使用 Homebrew
brew install --cask chromium

# 或
brew install --cask google-chrome
```

### Windows

从官网下载安装：
- Chrome: https://www.google.com/chrome/
- Chromium: https://chromium.woolyss.com/

## 配置路径

### 方法 1: 环境变量（推荐）

#### Bash

```bash
# 临时设置（当前会话）
export CHROME_BIN="/usr/bin/google-chrome-stable"

# 永久设置（添加到 ~/.bashrc）
echo 'export CHROME_BIN="/usr/bin/google-chrome-stable"' >> ~/.bashrc
source ~/.bashrc
```

#### Zsh

```bash
# 永久设置（添加到 ~/.zshrc）
echo 'export CHROME_BIN="/usr/bin/google-chrome-stable"' >> ~/.zshrc
source ~/.zshrc
```

#### Windows PowerShell

```powershell
# 临时设置
$env:CHROME_BIN = "C:\Program Files\Google\Chrome\Application\chrome.exe"

# 永久设置（系统环境变量）
[System.Environment]::SetEnvironmentVariable('CHROME_BIN', 'C:\Program Files\Google\Chrome\Application\chrome.exe', 'User')
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

### 方法 3: 使用 install.sh 脚本

```bash
# 自动下载并安装 Chromium 到本地缓存
bash configs/active_skills/browser_setup/scripts/install.sh

# 指定缓存目录
bash configs/active_skills/browser_setup/scripts/install.sh --cache-dir /opt/chromium
```

## 容器环境

### Docker 容器配置

在 Docker 容器中运行浏览器需要特殊处理：

#### 1. 添加 no-sandbox 参数

GopherPaw 会自动检测容器环境并添加 `--no-sandbox` 参数。

#### 2. 显式指定容器环境

```bash
export GOPHERPAW_RUNNING_IN_CONTAINER=1
```

#### 3. Dockerfile 示例

```dockerfile
FROM ubuntu:22.04

# 安装 Chromium
RUN apt-get update && \
    apt-get install -y chromium-browser && \
    rm -rf /var/lib/apt/lists/*

# 设置环境变量
ENV CHROME_BIN=/usr/bin/chromium-browser
ENV GOPHERPAW_RUNNING_IN_CONTAINER=1

# ... 其他配置
```

#### 4. docker run 示例

```bash
docker run -it \
  -e GOPHERPAW_RUNNING_IN_CONTAINER=1 \
  -e CHROME_BIN=/usr/bin/chromium-browser \
  --cap-add=SYS_ADMIN \
  gopherpaw:latest
```

### Docker Compose 示例

```yaml
version: '3.8'
services:
  gopherpaw:
    image: gopherpaw:latest
    environment:
      - GOPHERPAW_RUNNING_IN_CONTAINER=1
      - CHROME_BIN=/usr/bin/chromium-browser
    cap_add:
      - SYS_ADMIN
```

## 验证配置

### 使用 verify.sh 脚本

```bash
# 验证浏览器配置
bash configs/active_skills/browser_setup/scripts/verify.sh

# 验证指定浏览器
bash configs/active_skills/browser_setup/scripts/verify.sh --browser /usr/bin/chromium
```

输出示例：
```
==========================================
  浏览器环境验证
==========================================

[INFO] 使用浏览器: /usr/bin/google-chrome-stable

[✓] Testing 浏览器文件存在... PASS
[✓] Testing 浏览器可执行... PASS
[✓] Testing 版本信息... PASS
  Google Chrome 120.0.6099.109

检查依赖库:
[✓] 所有依赖库满足

[✓] Testing 容器环境检测... PASS (非容器环境)

环境变量:
[✓] CHROME_BIN=/usr/bin/google-chrome-stable

==========================================
  验证结果
==========================================

[INFO] 通过: 7
[✓] 所有检查通过!
```

### 手动验证

```bash
# 检查环境变量
echo $CHROME_BIN

# 检查版本
$CHROME_BIN --version

# 测试启动（headless 模式）
$CHROME_BIN --headless --disable-gpu --no-sandbox --dump-dom about:blank
```

## 测试浏览器工具

配置完成后，在 GopherPaw 中测试：

```
# 启动浏览器
browser_use action=start headed=true

# 打开网页
browser_use action=open url=https://example.com

# 截图
browser_use action=screenshot path=/tmp/test.png

# 获取页面快照
browser_use action=snapshot

# 关闭浏览器
browser_use action=stop
```

## 故障排除

### 问题: "executable file not found"

**原因**: 浏览器未安装或路径未配置。

**解决方案**:
1. 安装浏览器（见[安装浏览器](#安装浏览器)）
2. 设置 `CHROME_BIN` 环境变量
3. 或创建 `~/.gopherpaw/browser.json` 配置文件

### 问题: "no sandbox" 错误

**原因**: 容器环境需要特殊参数。

**解决方案**:
```bash
export GOPHERPAW_RUNNING_IN_CONTAINER=1
```

或手动添加参数：
```bash
chromium --no-sandbox --disable-setuid-sandbox
```

### 问题: 依赖库缺失

**原因**: 某些系统缺少浏览器所需的共享库。

**解决方案**:

#### Ubuntu/Debian
```bash
sudo apt-get install -y \
    libnss3 \
    libatk-bridge2.0-0 \
    libdrm2 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxrandr2 \
    libgbm1 \
    libasound2
```

#### Alpine
```bash
apk add --no-cache nss at-spi2-atk cups-libs gtk+3.0 libXcomposite \
    libXdamage libXfixes libXrandr libgbm libatspi2 pango \
    libxshmfence
```

### 问题: 权限错误

**原因**: 浏览器文件没有执行权限。

**解决方案**:
```bash
chmod +x /path/to/browser
```

### 问题: 内存不足

**原因**: headless 模式仍需要一定内存。

**解决方案**:
```bash
# 添加内存限制参数
export CHROME_FLAGS="--disable-dev-shm-usage --disable-gpu"
```

### 问题: 显示相关错误（headed 模式）

**原因**: 无图形界面环境。

**解决方案**:
使用 headless 模式：
```
browser_use action=start headed=false
```

## 高级配置

### 自定义 Chromium 版本

下载特定版本的 Chromium：

```bash
# 获取版本列表
# https://www.googleapis.com/download/storage/v1/b/chromium-browser-snapshots/o

# 下载指定版本（示例: Linux amd64 版本 1234567)
BASE_URL="https://www.googleapis.com/download/storage/v1/b/chromium-browser-snapshots"
wget "$BASE_URL/Linux_x64/1234567/chrome-linux.zip"
```

### 使用 Puppeteer 的 Chromium

```bash
# 安装 Node.js
sudo apt-get install -y nodejs npm

# 安装 Puppeteer（会自动下载 Chromium）
npm install puppeteer

# Chromium 路径
CHROME_BIN=$(npm root -g)/puppeteer/.local-chromium/linux-*/chrome-linux/chrome
```

### 使用 Playwright 的 Chromium

```bash
# 安装 Playwright
npm install -D playwright

# 安装浏览器
npx playwright install chromium

# Chromium 路径
CHROME_BIN=$(npm root -g)/playwright-core/.local-browsers/chromium-*/chrome-linux/chrome
```

## 参考资料

- [chromedp 文档](https://github.com/chromedp/chromedp)
- [Chrome 启动参数](https://peter.sh/experiments/chromium-command-line-switches/)
- [Chromium 下载](https://www.chromium.org/getting-involved/download-chromium)
