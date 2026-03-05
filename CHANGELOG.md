# Changelog

All notable changes to GopherPaw will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.1.0] - 2026-03-05

### Added

- **ReAct Agent** — Thought → Action → Observation 循环，支持多轮工具调用和自动上下文压缩
- **会话管理** — 基于 chatID 的多会话隔离，支持 `/new`、`/history`、`/status` 等内置命令
- **配置系统** — 基于 Viper 的 YAML 配置，支持 `GOPHERPAW_*` 环境变量覆盖
- **日志系统** — 基于 zap 的结构化日志，支持 JSON / Console 输出格式
- **LLM 接口层**
  - OpenAI 兼容 Provider（支持 GPT / DashScope / ModelScope 等）
  - Ollama 本地模型 Provider
  - Provider 注册表和动态切换
- **记忆系统**
  - 内存后端（sync.RWMutex + map）
  - 文件持久化后端（JSON 文件存储）
  - BM25 关键词搜索
  - 可选 Embedding 语义搜索（OpenAI Embedding API）
  - 混合搜索（BM25 + Embedding 加权融合）
  - LLM 驱动的自动对话压缩
- **内置工具**（11 个）
  - `get_current_time` — 当前时间
  - `web_search` — DuckDuckGo 网页搜索
  - `http_request` — HTTP 请求
  - `read_file` / `write_file` / `edit_file` / `append_file` — 文件操作
  - `execute_shell_command` — Shell 命令执行
  - `grep_search` / `glob_search` — 文件搜索
  - `memory_search` — 对话记忆搜索
- **消息渠道**
  - Console（stdin/stdout 交互）
  - Telegram（telebot）
  - Discord（discordgo）
  - 钉钉（HTTP API）
  - 飞书（HTTP API）
  - QQ（HTTP API）
  - Webhook 服务器（统一接收回调）
- **任务调度** — 基于 cron/v3 的定时任务 + 心跳机制
- **技能系统** — Markdown 文件定义技能，自动注入 System Prompt
- **MCP 客户端** — 支持 stdio 传输的 Model Context Protocol 工具服务器
- **CLI** — 基于 cobra 的子命令体系（app / init / models / channels / cron / chats / skills / env / clean / daemon）
- **Webhook 服务器** — 统一 HTTP 端点接收钉钉/飞书/QQ 回调
- **热重载** — 支持配置重载和服务重启

[0.1.0]: https://github.com/suifei/gopherpaw/releases/tag/v0.1.0
