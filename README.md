# GopherPaw 🐹

**用 Go 养一只属于你的 AI 小助手**

[![Go Report Card](https://goreportcard.com/badge/github.com/suifei/gopherpaw)](https://goreportcard.com/report/github.com/suifei/gopherpaw)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.25+-00ADD8.svg)](https://golang.org)

GopherPaw 是 [CoPaw](https://github.com/agentscope-ai/CoPaw) 的 Go 语言复刻版 —— 一个自托管的 AI Agent，单二进制、低内存、快启动。支持 Telegram、Discord、钉钉、飞书、QQ 等多平台接入。

## 特性

- **规划-执行分离模式** — 支持复杂任务的自动规划和执行
- **ReAct Agent** — Thought → Action → Observation 循环，支持多轮工具调用和并行执行
- **CLI 任务参数** — 启动时直接传入任务: `./gopherpaw app "任务"`
- **桌面容器化** — Docker XFCE + noVNC 支持
- **多渠道接入** — Console / Telegram / Discord / 钉钉 / 飞书 / QQ
- **内置工具** — 15+ 个内置工具（Web 搜索、HTTP 请求、文件操作、Shell 命令、Grep/Glob 搜索、浏览器自动化等）
- **记忆系统** — 短期对话历史 + BM25 搜索 + 可选 Embedding 语义搜索 + 自动压缩
- **技能系统** — 通过 Markdown 文件定义可扩展技能，注入 System Prompt
- **任务调度** — 基于 Cron 的定时任务和心跳机制
- **多 LLM 后端** — OpenAI 兼容 API（GPT / Claude / Qwen / DashScope / ModelScope）+ Ollama 本地模型
- **MCP 客户端** — 完整支持 Model Context Protocol（stdio/HTTP/SSE 传输）
- **单二进制部署** — `go build` 即可，无运行时依赖
- **高测试覆盖率** — 核心模块测试覆盖率 75%+

## 快速开始

### 前置要求

- Go 1.25+
- 一个 OpenAI 兼容的 API Key
- **可选**：Python 3.9+（用于文档处理等 Skills）
- **可选**：Bun 或 Node.js（自动下载，用于部分 Skills/MCP）

### 环境配置

首次使用建议运行环境检查：

```bash
# 检查运行时环境
./gopherpaw env check

# 自动设置 Python 虚拟环境（推荐）
./gopherpaw env setup
```

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

### 运行时环境配置

GopherPaw 支持 Python 和 Bun/Node.js 运行时，用于扩展 Skills 和 MCP 工具。

**Python 配置**（推荐使用虚拟环境）：

```yaml
runtime:
  python:
    venv_path: "~/.gopherpaw/venv"
    auto_install: true
```

**快速设置**：

```bash
# 自动创建虚拟环境并安装依赖
./gopherpaw env setup

# 或手动设置
python -m venv ~/.gopherpaw/venv
source ~/.gopherpaw/venv/bin/activate  # Linux/Mac
# 或 ~/.gopherpaw/venv/Scripts/activate  # Windows
pip install -r internal/runtime/requirements.txt
```

**Bun 配置**（自动下载）：

```yaml
runtime:
  node:
    runtime: "bun"
    auto_install: true
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
./gopherpaw env          # 环境变量和运行时检查
./gopherpaw clean        # 清理数据

# 新增：直接执行任务（非交互模式）
./gopherpaw app "搜索最新资讯" --once
./gopherpaw app "帮我分析数据"
```

启动后在终端输入消息即可与 Agent 对话。输入 `/help` 查看内置命令。

## 魔法命令

GopherPaw 提供了一系列内置命令，方便管理：

| 命令 | 说明 |
|------|------|
| `/help` | 显示帮助信息 |
| `/compact` | 压缩对话历史 |
| `/clear` | 清空上下文 |
| `/history` | 查看对话历史 |
| `/new` | 保存到长期记忆并清空上下文 |
| `/compact_str` | 查看压缩摘要 |
| `/switch-model <provider> <model>` | 切换 LLM 模型 |
| `/daemon [status\|version\|logs\|reload-config\|restart]` | 守护进程管理 |

## 高级功能

### 记忆系统

GopherPaw 的记忆系统支持：

- **短期记忆**: 最近 N 轮对话（可配置）
- **长期记忆**: 持久化存储重要信息
- **BM25 搜索**: 基于关键词的记忆搜索
- **Embedding 搜索**: 可选的语义搜索（需配置向量数据库）
- **自动压缩**: 当对话历史超过阈值时自动压缩

### 技能系统

通过 Markdown 文件定义技能：

```
configs/active_skills/
├── python_expert.md
├── code_reviewer.md
└── translator.md
```

技能会自动注入到 System Prompt 中。

### 多渠道支持

| 渠道 | 状态 | 特性 |
|------|------|------|
| Console | ✅ | 终端交互，默认启用 |
| Telegram | ✅ | Bot API，支持 Markdown |
| Discord | ✅ | Bot，支持 Markdown |
| 钉钉 | ✅ | Stream 模式 |
| 飞书 | ✅ | Stream 模式 |
| QQ | ✅ | go-cqhttp |

### 性能优化

- **单二进制**: 无运行时依赖，启动快
- **低内存**: 典型使用 < 100MB
- **并发处理**: 支持多会话并发
- **流式响应**: 支持 LLM 流式输出

`用浏览器开同程的网站看看最新的旅游产品是什么,对比下携程，同样的旅游产品，哪边优势更大？最后把选好的链接地址发给我`

## 项目结构

```
gopherpaw/
├── cmd/gopherpaw/          # CLI 入口 (cobra)
├── internal/
│   ├── agent/              # ReAct Agent 核心、会话管理、命令处理
│   ├── llm/                # LLM Provider (OpenAI / Ollama)
│   ├── memory/             # 记忆系统 (内存/文件/BM25/Embedding)
│   ├── tools/              # 内置工具 (15+ 个)
│   ├── channels/           # 消息渠道 (Console/Telegram/Discord/...)
│   ├── scheduler/          # Cron 调度 + 心跳
│   ├── skills/             # 技能管理器
│   ├── mcp/                # MCP 客户端
│   └── config/             # 配置加载与验证
├── pkg/logger/             # zap 日志封装
├── configs/
│   ├── config.yaml.example # 配置模板
│   └── active_skills/      # 内置技能
├── docs/                   # 文档
│   ├── architecture_spec.md
│   ├── api_spec.md
│   └── alignment/          # 对齐验证文档
└── go.mod
```

## 内置工具

GopherPaw 内置 15+ 个实用工具：

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
| `list_directory` | 列出目录内容 |
| `copy_file` | 复制文件 |
| `move_file` | 移动/重命名文件 |
| `delete_file` | 删除文件 |

## 支持的 LLM

通过 OpenAI 兼容 API 支持：

- OpenAI (GPT-4o / GPT-4o-mini / ...)
- Anthropic Claude (通过兼容代理)
- 阿里云 DashScope / ModelScope
- 本地 Ollama

### 多模型切换

支持在运行时动态切换模型：

```bash
/switch-model <provider> <model>
```

## MCP 支持

GopherPaw 支持 Model Context Protocol (MCP)，可以通过 MCP 扩展工具能力：

**支持的传输协议**：
- `stdio` - 标准输入输出
- `streamable_http` - HTTP 流式传输
- `sse` - Server-Sent Events

**配置示例**：

```yaml
mcp:
  servers:
    filesystem:
      transport: stdio
      command: mcp-filesystem
      args: ["--root", "/path/to/files"]
      enabled: true
```

## 测试与质量

- **单元测试覆盖率**: 70%+ (核心模块 75%+)
- **集成测试**: Agent + LLM + Memory + Tools
- **代码质量**: go vet + gofmt 通过
- **持续集成**: GitHub Actions (计划中)

运行测试：

```bash
# 运行所有测试
go test ./...

# 查看覆盖率
go test ./internal/agent/ -cover

# 运行竞态检测
go test -race ./...
```

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

## 开发指南

### 环境要求

- Go 1.25+
- Make (可选)

### 构建

```bash
# 构建
go build -o gopherpaw ./cmd/gopherpaw/

# 运行测试
go test ./...

# 代码检查
go vet ./...
gofmt -w .

# 运行测试并查看覆盖率
go test ./internal/agent/ -cover
```

### 架构

GopherPaw 采用分层架构：

```
cmd/                    # 应用入口
  └── gopherpaw/
internal/
  channels/             # 接口层 → agent/
  agent/                # 领域核心 → llm/, memory/, tools/ (接口)
  llm/, memory/, tools/ # 基础设施 → config/
  scheduler/            # 调度器 → agent/ (接口)
  mcp/                  # MCP 客户端 → agent/ (接口)
  config/               # 配置加载与验证
```

**关键原则**：
- 下层不依赖上层
- 基础设施不依赖领域核心
- 通过接口解耦

详见 [架构规范](docs/architecture_spec.md)。

### 贡献代码

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'Add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

**代码规范**：
- 遵循 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- 使用 `gofmt` 格式化代码
- 确保测试通过且覆盖率不低于当前水平
- 添加必要的注释和文档

## 路线图

### v1.0.0 (当前)
- ✅ 核心 ReAct Agent
- ✅ 多渠道支持
- ✅ 记忆系统
- ✅ 技能系统
- ✅ MCP 客户端
- ✅ 15+ 内置工具

### v1.1.0 (计划中)
- 🔄 Web UI 界面
- 🔄 更多 LLM 提供商
- 🔄 插件系统
- 🔄 性能优化

### v2.0.0 (未来)
- 📋 多 Agent 协作
- 📋 可视化工作流
- 📋 企业级特性

## 常见问题

**Q: 如何切换不同的 LLM 提供商？**

A: 使用 `/switch-model` 命令或修改配置文件：

```bash
/switch-model openai gpt-4o
```

**Q: 记忆系统如何工作？**

A: GopherPaw 使用三层记忆：
1. 短期记忆（最近 N 轮对话）
2. 长期记忆（持久化存储）
3. 搜索记忆（BM25 + 可选 Embedding）

**Q: 如何添加自定义工具？**

A: 通过 MCP 协议或直接在代码中实现 Tool 接口。详见文档。

**Q: 支持哪些语言？**

A: 目前支持中文和英文，可通过配置切换。

## 社区

- **问题反馈**: [GitHub Issues](https://github.com/suifei/gopherpaw/issues)
- **功能建议**: [GitHub Discussions](https://github.com/suifei/gopherpaw/discussions)
- **代码贡献**: [Pull Requests](https://github.com/suifei/gopherpaw/pulls)

## 更新日志

### 2026-03-07
- ✅ 完成核心模块测试覆盖率提升（70%+）
- ✅ 新增 50+ 测试用例
- ✅ 完善 MCP 配置验证
- ✅ 实现 RebuildInfo 方法
- ✅ 文档更新

### 2026-03-06
- ✅ 初始版本发布
- ✅ 核心功能实现
- ✅ 多渠道支持
