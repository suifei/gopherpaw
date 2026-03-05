# GopherPaw 🐹

**用 Go 养一只属于你的 AI 小助手**

GopherPaw 是 [CoPaw](https://github.com/agentscope-ai/CoPaw) 的 Go 语言复刻版 —— 一个自托管的 AI Agent，单二进制、低内存、快启动。支持 Telegram、Discord、钉钉、飞书、QQ 等多平台接入。

## 特性

- **ReAct Agent** — Thought → Action → Observation 循环，支持多轮工具调用
- **多渠道接入** — Console / Telegram / Discord / 钉钉 / 飞书 / QQ
- **内置工具** — Web 搜索、HTTP 请求、文件操作、Shell 命令、Grep/Glob 搜索、时间查询、记忆搜索
- **记忆系统** — 短期对话历史 + BM25 搜索 + 可选 Embedding 语义搜索 + 自动压缩
- **技能系统** — 通过 Markdown 文件定义可扩展技能，注入 System Prompt
- **任务调度** — 基于 Cron 的定时任务和心跳机制
- **多 LLM 后端** — OpenAI 兼容 API（GPT / Claude / Qwen / DashScope / ModelScope）+ Ollama 本地模型
- **MCP 客户端** — 支持 Model Context Protocol stdio 工具服务器
- **单二进制部署** — `go build` 即可，无运行时依赖

## 快速开始

### 前置要求

- Go 1.25+
- 一个 OpenAI 兼容的 API Key

### 安装

```bash
git clone https://github.com/suifei/gopherpaw.git
cd gopherpaw
go build -o gopherpaw ./cmd/gopherpaw
```

### 配置

```bash
cp configs/config.yaml.example configs/config.yaml
```

编辑 `configs/config.yaml`，填入你的 LLM 配置：

```yaml
llm:
  provider: openai
  model: gpt-4o-mini
  api_key: "your-api-key"
  base_url: "https://api.openai.com/v1"
```

或通过环境变量设置（推荐）：

```bash
export GOPHERPAW_LLM_API_KEY=your-api-key
export GOPHERPAW_LLM_BASE_URL=https://api.openai.com/v1
export GOPHERPAW_LLM_MODEL=gpt-4o-mini
```

### 运行

```bash
# 直接启动（默认开启 Console 渠道）
./gopherpaw

# 或指定配置文件
./gopherpaw -c /path/to/config.yaml

# 使用子命令
./gopherpaw app          # 启动服务
./gopherpaw init         # 交互式初始化
./gopherpaw models       # 管理 LLM 模型
./gopherpaw channels     # 管理消息渠道
./gopherpaw skills       # 管理技能
./gopherpaw cron         # 管理定时任务
./gopherpaw chats        # 管理对话
./gopherpaw env          # 查看环境变量
./gopherpaw clean        # 清理数据
```

启动后在终端输入消息即可与 Agent 对话。输入 `/help` 查看内置命令。

## 项目结构

```
gopherpaw/
├── cmd/gopherpaw/          # CLI 入口 (cobra)
├── internal/
│   ├── agent/              # ReAct Agent 核心、会话管理、命令处理
│   ├── llm/                # LLM Provider (OpenAI / Ollama)
│   ├── memory/             # 记忆系统 (内存/文件/BM25/Embedding)
│   ├── tools/              # 内置工具 (11 个)
│   ├── channels/           # 消息渠道 (Console/Telegram/Discord/...)
│   ├── scheduler/          # Cron 调度 + 心跳
│   ├── skills/             # 技能管理器
│   ├── mcp/                # MCP 客户端
│   └── config/             # 配置加载与验证
├── pkg/logger/             # zap 日志封装
├── configs/
│   ├── config.yaml.example # 配置模板
│   └── active_skills/      # 内置技能
└── go.mod
```

## 内置工具

| 工具 | 说明 |
|------|------|
| `get_current_time` | 获取当前日期和时间 |
| `web_search` | DuckDuckGo 网页搜索（无需 API Key） |
| `http_request` | 发送 HTTP 请求（GET/POST/PUT/DELETE） |
| `read_file` | 读取文件内容 |
| `write_file` | 写入文件 |
| `edit_file` | 查找替换编辑文件 |
| `append_file` | 追加内容到文件末尾 |
| `execute_shell_command` | 执行 Shell 命令 |
| `grep_search` | 正则搜索文件内容 |
| `glob_search` | 按模式搜索文件名 |
| `memory_search` | 搜索对话记忆 |

## 支持的 LLM

通过 OpenAI 兼容 API 支持：

- OpenAI (GPT-4o / GPT-4o-mini / ...)
- Anthropic Claude (通过兼容代理)
- 阿里云 DashScope / ModelScope
- 本地 Ollama

## 技术栈

| 组件 | 库 |
|------|-----|
| CLI | [cobra](https://github.com/spf13/cobra) |
| 配置 | [viper](https://github.com/spf13/viper) |
| LLM | [go-openai](https://github.com/sashabaranov/go-openai) |
| 日志 | [zap](https://go.uber.org/zap) |
| 调度 | [cron](https://github.com/robfig/cron) |
| Telegram | [telebot](https://gopkg.in/telebot.v4) |
| Discord | [discordgo](https://github.com/bwmarrin/discordgo) |
| Web 搜索 | [ddgsearch](https://github.com/kuhahalong/ddgsearch) |

## 致谢

- [CoPaw](https://github.com/agentscope-ai/CoPaw) — 原始 Python 实现
- [AgentScope](https://github.com/modelscope/agentscope) — 阿里巴巴 AI Agent 框架

## License

MIT
