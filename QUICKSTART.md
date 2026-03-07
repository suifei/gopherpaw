# GopherPaw 快速开始指南

**5 分钟快速上手 GopherPaw**

---

## 📋 前置要求

- Go 1.21+ 
- Git
- （可选）Python 3.8+ - 用于 execute_python_code 工具
- （可选）Ollama - 用于本地 LLM

---

## 🚀 安装

### 方式 1: 从源码编译

```bash
# 克隆仓库
git clone https://github.com/suifei/gopherpaw.git
cd gopherpaw

# 编译
go build -o gopherpaw ./cmd/gopherpaw/

# 运行
./gopherpaw --help
```

### 方式 2: 使用安装脚本

```bash
# Linux/macOS
curl -fsSL https://raw.githubusercontent.com/suifei/gopherpaw/main/install.sh | bash

# Windows (PowerShell)
iwr https://raw.githubusercontent.com/suifei/gopherpaw/main/install.ps1 -useb | iex
```

### 方式 3: 使用 Docker

```bash
# 拉取镜像
docker pull suifei/gopherpaw:latest

# 运行
docker run -it --rm suifei/gopherpaw:latest --help
```

---

## ⚙️ 配置

### 1. 创建配置文件

```bash
# 创建配置目录
mkdir -p ~/.gopherpaw

# 复制配置示例
cp configs/config.yaml.example ~/.gopherpaw/config.yaml

# 编辑配置
vim ~/.gopherpaw/config.yaml
```

### 2. 最小配置

编辑 `~/.gopherpaw/config.yaml`：

```yaml
# LLM 配置
llm:
  default_provider: "openai"
  providers:
    openai:
      api_key: "${GOPHERPAW_LLM_API_KEY}"  # 从环境变量读取
      base_url: "https://api.openai.com/v1"
      model: "gpt-4"

# Agent 配置
agent:
  language: "zh-CN"
  running:
    max_turns: 10

# 通道配置
channels:
  console:
    enabled: true
```

### 3. 设置环境变量

```bash
# 设置 API Key
export GOPHERPAW_LLM_API_KEY="your-api-key-here"

# 或使用测试端点（仅用于测试）
export GOPHERPAW_LLM_API_KEY="c9186fd1c99fb2e3cbe0cdbe9709fe746a4823d0b9b4322d"
export GOPHERPAW_LLM_BASE_URL="https://llm.meta2cs.cn/v1"
```

---

## 💬 基本使用

### Console 模式（交互式聊天）

```bash
# 启动 Console 通道
./gopherpaw app

# 或指定工作目录
./gopherpaw app --workdir /path/to/your/project
```

**示例对话**：

```
You: 你好！请介绍一下你自己
AI: 你好！我是 GopherPaw，一个基于 ReAct 架构的 AI 助手...

You: 帮我读取 README.md 文件
AI: [调用 read_file 工具]
这是 README.md 的内容...

You: 现在几点了？
AI: [调用 get_current_time 工具]
当前时间是 2026-03-07 14:30:00 CST
```

### 魔法命令

```bash
# 查看历史记录
/history

# 开始新会话
/new

# 清除历史
/clear

# 压缩记忆
/compact

# 查看状态
/status

# 后台任务
/daemon
```

---

## 🛠️ 工具使用示例

### 文件操作

```
You: 读取 main.go 文件的第 10-20 行
AI: [调用 read_file 工具]
main.go (lines 10-20 of 100):
func main() {
    ...
}

You: 在 test.txt 中写入 "Hello, GopherPaw!"
AI: [调用 write_file 工具]
Wrote 20 bytes to test.txt.

You: 追加一行 "This is a test."
AI: [调用 append_file 工具]
Appended 16 bytes to test.txt.
```

### Shell 命令

```
You: 执行 ls -la 命令
AI: [调用 execute_shell_command 工具]
total 48
drwxr-xr-x  12 user  staff   384 Mar  7 14:30 .
...
```

### Python 代码执行

```
You: 用 Python 计算斐波那契数列的前 10 项
AI: [调用 execute_python_code 工具]
[1, 1, 2, 3, 5, 8, 13, 21, 34, 55]
```

### 网页搜索

```
You: 搜索 Go 语言的最新特性
AI: [调用 web_search 工具]
1. Go 1.22 发布，新增功能包括...
2. Go 语言性能优化技巧...
```

---

## 📚 技能系统

### 使用内置技能

```bash
# 列出所有技能
./gopherpaw skills list

# 启用技能
./gopherpaw skills enable browser_visible

# 禁用技能
./gopherpaw skills disable pdf
```

### 创建自定义技能

```bash
# 创建技能目录
mkdir -p ~/.gopherpaw/skills/my-skill

# 创建技能文件
cat > ~/.gopherpaw/skills/my-skill/SKILL.md << 'EOF'
---
name: my-skill
description: 我的自定义技能
enabled: true
---

# My Skill

这是一个自定义技能的说明。

## 使用场景

- 场景 1
- 场景 2

## 注意事项

- 注意点 1
- 注意点 2
EOF

# 重新加载技能
./gopherpaw skills list
```

---

## 🌐 多通道使用

### Telegram

1. **创建 Bot**
   - 在 Telegram 中找到 @BotFather
   - 发送 `/newbot` 创建新 Bot
   - 获取 API Token

2. **配置**

```yaml
channels:
  telegram:
    enabled: true
    token: "your-telegram-bot-token"
```

3. **运行**

```bash
./gopherpaw app --channel telegram
```

### Discord

1. **创建 Bot**
   - 在 Discord Developer Portal 创建应用
   - 创建 Bot 并获取 Token

2. **配置**

```yaml
channels:
  discord:
    enabled: true
    token: "your-discord-bot-token"
```

3. **运行**

```bash
./gopherpaw app --channel discord
```

---

## 🔄 后台运行

### 使用 Daemon 模式

```bash
# 启动守护进程
./gopherpaw daemon

# 查看状态
./gopherpaw daemon status

# 停止
./gopherpaw daemon stop
```

### 使用 Supervisor

```ini
[program:gopherpaw]
command=/usr/local/bin/gopherpaw daemon
directory=/home/user/.gopherpaw
user=user
autostart=true
autorestart=true
stderr_logfile=/var/log/gopherpaw/err.log
stdout_logfile=/var/log/gopherpaw/out.log
```

```bash
# 启动
supervisorctl start gopherpaw

# 停止
supervisorctl stop gopherpaw

# 重启
supervisorctl restart gopherpaw
```

---

## 🐛 故障排查

### 1. LLM 连接失败

```bash
# 检查 API Key
echo $GOPHERPAW_LLM_API_KEY

# 测试连接
curl -H "Authorization: Bearer $GOPHERPAW_LLM_API_KEY" \
     https://api.openai.com/v1/models
```

### 2. 配置文件未找到

```bash
# 检查配置文件位置
ls -la ~/.gopherpaw/config.yaml

# 或指定配置文件
./gopherpaw app --config /path/to/config.yaml
```

### 3. 权限问题

```bash
# 检查文件权限
ls -la ~/.gopherpaw/

# 修复权限
chmod 755 ~/.gopherpaw/
chmod 644 ~/.gopherpaw/config.yaml
```

### 4. 日志查看

```bash
# 查看日志
tail -f ~/.gopherpaw/logs/gopherpaw.log

# 或使用 journalctl（systemd）
journalctl -u gopherpaw -f
```

---

## 📖 更多资源

- **完整文档**: [docs/](./docs/)
- **API 规范**: [docs/api_spec.md](./docs/api_spec.md)
- **架构说明**: [docs/architecture_spec.md](./docs/architecture_spec.md)
- **部署指南**: [docs/deployment-guide.md](./docs/deployment-guide.md)
- **更新日志**: [CHANGELOG.md](./CHANGELOG.md)
- **发布说明**: [RELEASE_NOTES.md](./RELEASE_NOTES.md)

---

## 💡 提示

1. **首次使用**：建议先用 Console 模式测试，确认配置正确
2. **生产环境**：使用 Supervisor 或 Docker 进行部署
3. **性能优化**：根据实际需求调整 max_turns 和内存配置
4. **安全性**：不要在配置文件中硬编码 API Key，使用环境变量

---

## 🎯 下一步

- ✅ 完成基本配置和测试
- 📚 阅读完整文档了解高级功能
- 🛠️ 根据需求定制技能和工具
- 🚀 部署到生产环境

---

**祝你使用愉快！如有问题，请访问 [GitHub Issues](https://github.com/suifei/gopherpaw/issues) 反馈。**
