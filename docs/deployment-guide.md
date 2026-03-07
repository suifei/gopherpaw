# GopherPaw 部署指南

> **版本**: v1.0
> **最后更新**: 2026-03-07
> **适用环境**: Linux, macOS, Windows, Docker

## 目录

- [快速开始](#快速开始)
- [安装方式](#安装方式)
  - [从源码构建](#从源码构建)
  - [Docker 部署](#docker-部署)
  - [二进制部署](#二进制部署)
- [配置说明](#配置说明)
- [运行方式](#运行方式)
- [生产环境部署](#生产环境部署)
- [监控和日志](#监控和日志)
- [故障排查](#故障排查)

---

## 快速开始

### 前置要求

- **Go**: 1.23 或更高版本（从源码构建需要）
- **Python**: 3.8+（可选，用于 Python 代码执行工具）
- **Node.js/Bun**: 用于运行 JS/TS 脚本（可选）
- **Docker**: 20.10+（Docker 部署需要）

### 最快安装方式

**Linux/macOS:**
```bash
git clone https://github.com/suifei/gopherpaw.git
cd gopherpaw
./install.sh
```

**Windows:**
```powershell
git clone https://github.com/suifei/gopherpaw.git
cd gopherpaw
.\install.ps1
```

---

## 安装方式

### 1. 从源码构建

#### Linux / macOS

```bash
# 克隆仓库
git clone https://github.com/suifei/gopherpaw.git
cd gopherpaw

# 运行安装脚本
chmod +x install.sh
./install.sh

# 或者手动构建
go build -o gopherpaw ./cmd/gopherpaw/
```

安装脚本会：
- 检测系统架构（amd64/arm64）
- 编译二进制文件
- 安装到 `~/.local/bin/gopherpaw`
- 创建默认配置文件 `~/.gopherpaw/config.yaml`
- 创建技能目录 `~/.gopherpaw/active_skills` 和 `~/.gopherpaw/customized_skills`

#### Windows

**使用 PowerShell:**
```powershell
# 克隆仓库
git clone https://github.com/suifei/gopherpaw.git
cd gopherpaw

# 运行安装脚本
.\install.ps1
```

**使用 CMD:**
```cmd
REM 克隆仓库
git clone https://github.com/suifei/gopherpaw.git
cd gopherpaw

REM 运行安装脚本
install.bat
```

安装脚本会：
- 编译二进制文件
- 安装到 `%USERPROFILE%\.local\bin\gopherpaw.exe`
- 创建默认配置文件 `%USERPROFILE%\.gopherpaw\config.yaml`
- 创建技能目录

### 2. Docker 部署

#### 使用轻量级镜像（推荐）

```bash
# 构建镜像
docker build -t gopherpaw:latest -f Dockerfile .

# 运行容器
docker run -d \
  --name gopherpaw \
  -p 8080:8080 \
  -v ~/.gopherpaw:/app/data \
  -e GOPHERPAW_LLM_API_KEY=your_api_key \
  -e GOPHERPAW_LLM_BASE_URL=https://api.openai.com/v1 \
  gopherpaw:latest \
  gopherpaw start
```

#### 使用完整镜像（包含 Python 和 Node.js）

```bash
# 构建镜像
docker build -t gopherpaw:full -f Dockerfile.full .

# 运行容器
docker run -d \
  --name gopherpaw-full \
  -p 8080:8080 \
  -v ~/.gopherpaw:/app/data \
  -e GOPHERPAW_LLM_API_KEY=your_api_key \
  -e GOPHERPAW_LLM_BASE_URL=https://api.openai.com/v1 \
  gopherpaw:full \
  gopherpaw start
```

#### Docker Compose

创建 `docker-compose.yml`:

```yaml
version: '3.8'

services:
  gopherpaw:
    build:
      context: .
      dockerfile: Dockerfile.full
    container_name: gopherpaw
    restart: unless-stopped
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
      - ./configs:/app/configs
    environment:
      - GOPHERPAW_WORKING_DIR=/app/data
      - GOPHERPAW_LLM_API_KEY=${GOPHERPAW_LLM_API_KEY}
      - GOPHERPAW_LLM_BASE_URL=${GOPHERPAW_LLM_BASE_URL}
      - GOPHERPAW_LOG_LEVEL=info
    healthcheck:
      test: ["CMD", "wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 3s
      retries: 3
      start_period: 5s
```

运行：
```bash
docker-compose up -d
```

### 3. 二进制部署

直接下载预编译的二进制文件（如果有发布）：

```bash
# Linux/macOS
wget https://github.com/suifei/gopherpaw/releases/download/v1.0.0/gopherpaw-linux-amd64
chmod +x gopherpaw-linux-amd64
sudo mv gopherpaw-linux-amd64 /usr/local/bin/gopherpaw

# Windows
# 下载 gopherpaw-windows-amd64.exe 并添加到 PATH
```

---

## 配置说明

### 配置文件位置

GopherPaw 按以下顺序查找配置文件：

1. 命令行参数 `--config`
2. 环境变量 `GOPHERPAW_CONFIG_FILE`
3. 工作目录下的 `config.yaml`
4. `~/.gopherpaw/config.yaml`（默认）

### 配置文件结构

```yaml
# 服务器配置
server:
  host: 0.0.0.0
  port: 8080

# Agent 配置
agent:
  system_prompt: "You are a helpful AI assistant."
  working_dir: ""
  defaults:
    heartbeat:
      every: 30m
      target: main
  running:
    max_turns: 20
    max_input_length: 131072
    namesake_strategy: override
  language: zh

# LLM 配置
llm:
  provider: openai
  model: gpt-4o-mini
  api_key: ${GOPHERPAW_LLM_API_KEY}  # 从环境变量读取
  base_url: ${GOPHERPAW_LLM_BASE_URL}
  ollama_url: http://localhost:11434

# 记忆配置
memory:
  backend: sqlite
  db_path: ./data/gopherpaw.db
  max_history: 50
  compact_threshold: 100000
  compact_keep_recent: 3
  compact_ratio: 0.7

# 通道配置
channels:
  console:
    enabled: true
    filter_tool_messages: false
  telegram:
    enabled: false
    bot_token: ""
    filter_tool_messages: true
  # ... 其他通道

# 调度器配置
scheduler:
  enabled: false
  heartbeat:
    every: 30m
    target: main

# 日志配置
log:
  level: info
  format: json

# 技能配置
skills:
  active_dir: ./active_skills
  customized_dir: ./customized_skills

# MCP 配置
mcp:
  servers:
    example-server:
      name: "Example MCP Server"
      transport: stdio
      command: "node"
      args: ["server.js"]
      enabled: true

# UI 配置
show_tool_details: true
```

### 环境变量

所有配置项都可以通过环境变量覆盖，格式为 `GOPHERPAW_<SECTION>_<KEY>`：

```bash
# 示例
export GOPHERPAW_LLM_API_KEY="sk-..."
export GOPHERPAW_LLM_BASE_URL="https://api.openai.com/v1"
export GOPHERPAW_SERVER_PORT="9090"
export GOPHERPAW_LOG_LEVEL="debug"
export GOPHERPAW_WORKING_DIR="/var/lib/gopherpaw"
```

---

## 运行方式

### 开发模式

```bash
# 使用默认配置
gopherpaw start

# 指定配置文件
gopherpaw start --config /path/to/config.yaml

# 启用调试日志
gopherpaw start --log-level debug

# 启用特定通道
gopherpaw start --channels console,telegram
```

### 生产模式

#### 使用 Systemd（Linux）

创建服务文件 `/etc/systemd/system/gopherpaw.service`:

```ini
[Unit]
Description=GopherPaw AI Assistant
After=network.target

[Service]
Type=simple
User=gopherpaw
Group=gopherpaw
WorkingDirectory=/var/lib/gopherpaw
ExecStart=/usr/local/bin/gopherpaw start --config /etc/gopherpaw/config.yaml
Restart=on-failure
RestartSec=5s

# 环境变量
Environment="GOPHERPAW_LLM_API_KEY=your_api_key"
Environment="GOPHERPAW_LOG_LEVEL=info"

# 安全限制
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=/var/lib/gopherpaw

[Install]
WantedBy=multi-user.target
```

启动服务：
```bash
sudo systemctl daemon-reload
sudo systemctl enable gopherpaw
sudo systemctl start gopherpaw
sudo systemctl status gopherpaw
```

#### 使用 Supervisor

创建配置文件 `/etc/supervisor/conf.d/gopherpaw.conf`:

```ini
[program:gopherpaw]
command=/usr/local/bin/gopherpaw start --config /etc/gopherpaw/config.yaml
directory=/var/lib/gopherpaw
user=gopherpaw
autostart=true
autorestart=true
startsecs=3
stderr_logfile=/var/log/gopherpaw/err.log
stdout_logfile=/var/log/gopherpaw/out.log
environment=GOPHERPAW_LLM_API_KEY="your_api_key",GOPHERPAW_LOG_LEVEL="info"
```

启动服务：
```bash
sudo supervisorctl reread
sudo supervisorctl update
sudo supervisorctl start gopherpaw
```

---

## 生产环境部署

### 性能优化

#### 1. 数据库优化

```yaml
memory:
  backend: sqlite
  db_path: /var/lib/gopherpaw/gopherpaw.db
  max_history: 100
  compact_threshold: 200000
  compact_keep_recent: 5
  fts_enabled: true
  embedding_dimensions: 1024
  embedding_max_cache: 5000
```

#### 2. 并发配置

```yaml
agent:
  running:
    max_turns: 30
    max_input_length: 200000
```

#### 3. 资源限制

Docker 部署时设置资源限制：
```bash
docker run -d \
  --name gopherpaw \
  --memory="2g" \
  --cpus="2.0" \
  -p 8080:8080 \
  gopherpaw:latest \
  gopherpaw start
```

### 高可用部署

#### 负载均衡

使用 Nginx 作为反向代理：

```nginx
upstream gopherpaw {
    server 10.0.1.10:8080;
    server 10.0.1.11:8080;
    server 10.0.1.12:8080;
}

server {
    listen 80;
    server_name gopherpaw.example.com;

    location / {
        proxy_pass http://gopherpaw;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

---

## 监控和日志

### 日志管理

GopherPaw 使用结构化日志（JSON 格式）：

```bash
# 查看日志
tail -f /var/log/gopherpaw/out.log

# 过滤特定级别的日志
cat /var/log/gopherpaw/out.log | jq 'select(.level=="error")'

# 导出日志到 ELK
# 配置 filebeat 或 fluentd 收集日志
```

### 健康检查

```bash
# HTTP 健康检查
curl http://localhost:8080/health

# 命令行健康检查
gopherpaw status
```

### Prometheus 监控（计划中）

```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'gopherpaw'
    static_configs:
      - targets: ['localhost:9090']
```

---

## 故障排查

### 常见问题

#### 1. 配置文件找不到

**症状**: 启动失败，提示配置文件不存在

**解决**:
```bash
# 检查配置文件位置
ls -la ~/.gopherpaw/config.yaml

# 或者使用环境变量指定
export GOPHERPAW_WORKING_DIR=/path/to/config
```

#### 2. API Key 无效

**症状**: LLM 调用失败

**解决**:
```bash
# 检查环境变量
echo $GOPHERPAW_LLM_API_KEY

# 测试 API 连接
curl -H "Authorization: Bearer $GOPHERPAW_LLM_API_KEY" \
     https://api.openai.com/v1/models
```

#### 3. 权限问题

**症状**: 无法写入数据库或日志

**解决**:
```bash
# 检查目录权限
ls -la /var/lib/gopherpaw
sudo chown -R gopherpaw:gopherpaw /var/lib/gopherpaw
```

#### 4. 端口占用

**症状**: 启动失败，端口已被占用

**解决**:
```bash
# 检查端口占用
lsof -i :8080

# 使用其他端口
gopherpaw start --port 9090
```

### 调试模式

```bash
# 启用详细日志
gopherpaw start --log-level debug

# 查看实时日志
tail -f /var/log/gopherpaw/out.log | grep DEBUG
```

---

## 升级和维护

### 升级步骤

```bash
# 1. 备份配置和数据
cp -r ~/.gopherpaw ~/.gopherpaw.backup

# 2. 停止服务
sudo systemctl stop gopherpaw

# 3. 下载新版本
git pull origin main

# 4. 重新构建
go build -o gopherpaw ./cmd/gopherpaw/

# 5. 安装
./install.sh

# 6. 启动服务
sudo systemctl start gopherpaw

# 7. 验证
gopherpaw version
```

### 数据备份

```bash
# 备份数据库
sqlite3 ~/.gopherpaw/gopherpaw.db ".backup ~/.gopherpaw/backup.db"

# 备份整个目录
tar -czf gopherpaw-backup-$(date +%Y%m%d).tar.gz ~/.gopherpaw
```

---

## 安全建议

1. **使用非 root 用户运行**
2. **设置文件权限**: 配置文件 600，数据目录 700
3. **使用环境变量存储敏感信息**（API Key 等）
4. **定期更新依赖**：`go get -u && go mod tidy`
5. **启用 HTTPS**（生产环境）
6. **配置防火墙规则**，限制访问端口

---

## 技术支持

- **GitHub Issues**: https://github.com/suifei/gopherpaw/issues
- **文档**: https://github.com/suifei/gopherpaw/docs
- **社区**: GitHub Discussions

---

**最后更新**: 2026-03-07
