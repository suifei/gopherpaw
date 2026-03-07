# Changelog

All notable changes to GopherPaw will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.2.0] - 2026-03-07

### 🐛 Bug Fixes
- **Runtime 包优化** — 修复 Bun 下载和安装问题，优先使用官方安装脚本
- **测试稳定性提升** — 修复 channels 包中的并发测试问题，解决 stdout/stderr 资源泄漏
- **依赖管理** — 添加 node_modules/ 到 .gitignore，防止测试生成的依赖被提交

### ⚡ 性能优化
- **Bun 下载策略** — 添加 GitHub API 版本查询，改进 Windows 平台兼容性
- **并发安全测试** — 添加资源管理 helper 函数，避免测试中的资源竞争

### 🔧 开发体验
- **Git 忽略更新** — 防止 node_modules 被意外提交到版本控制
- **文档完善** — 更新架构文档，明确 Bun 安装方式

## [1.0.0] - 2026-03-07

### 🎉 Production Release

这是 GopherPaw 的首个生产就绪版本，完成了与 CoPaw 的 95% 功能对齐。

### Added

#### 核心功能增强
- **view_text_file 工具** — 安全查看文本文件，自动检测并拒绝二进制文件
- **execute_python_code 工具** — 执行 Python 代码并返回结果
- **完整的工具元数据** — 所有 17 个内置工具都有完整的 JSON Schema 参数定义
- **LastDispatch 配置** — 支持保存和加载最后调度状态
- **ShowToolDetails 配置** — 控制工具详细信息的显示
- **filter_tool_messages 配置** — 所有通道支持工具消息过滤

#### MCP 系统增强
- **熔断器模式** — 防止频繁重连导致的系统过载，支持 closed/open/half-open 状态
- **健康检查机制** — 定期检查客户端健康状态，自动触发重连
- **错误恢复策略** — 智能重连结合熔断器状态，详细的错误记录和追踪
- **MCPManager** — 完整的客户端管理器，支持动态添加/删除/重载

#### Agent 系统增强
- **Bootstrap Hook** — 首次交互引导机制，支持 BOOTSTRAP.md
- **Memory Compaction Hook** — 自动内存压缩，支持 LLM 摘要
- **NamesakeStrategy** — 重名工具处理策略（override/skip/raise/rename）
- **HEARTBEAT.md 支持** — 心跳查询文件支持
- **完整命令系统** — 所有魔法命令已实现

#### Skills 系统增强
- **scripts/ 目录支持** — 技能脚本目录
- **references/ 目录支持** — 技能参考文档目录
- **18 个内置技能** — 包括 docx、pdf、xlsx、pptx、browser_visible 等
- **完整 CLI 命令** — list/enable/disable/create/delete/import

#### 部署支持
- **Dockerfile** — 轻量版和完整版多阶段构建
- **安装脚本** — Linux/macOS/Windows 多平台支持
- **Supervisor 配置** — 进程管理配置
- **Docker Compose** — 容器编排配置
- **完整部署文档** — docs/deployment-guide.md

#### 测试和质量
- **性能测试套件** — 内存使用、响应时间、并发、内存泄漏测试
- **基准测试** — Agent 运行、并行执行、工具执行基准
- **竞态检测** — 所有核心模块通过 -race 检测
- **测试覆盖率** — 总体 69.4%，核心模块 > 70%
- **熔断器测试** — 完整的状态转换测试

### Changed

#### 性能优化
- **内存使用** — 优化后 < 1MB（空闲状态）
- **响应时间** — 平均 2.759µs（极快）
- **并发能力** — 支持 100+ 并发会话
- **零内存泄漏** — 长时间运行测试通过

#### 代码质量
- **MIME 类型检测** — 优先使用自定义映射，提升准确性
- **内存统计** — 修复竞态检测模式下的统计问题
- **错误处理** — 更完善的错误包装和上下文

### Fixed

- **TestDetectMIME** — 修复 MIME 类型检测优先级错误
- **TestMemoryUsage** — 修复竞态检测模式下的内存统计异常
- **配置兼容性** — 完全兼容 CoPaw 配置格式
- **通道过滤** — 所有通道正确支持 filter_tool_messages

### Documentation

- **API 规范** — docs/api_spec.md（1200+ 行）
- **架构规范** — docs/architecture_spec.md
- **部署指南** — docs/deployment-guide.md
- **用户指南** — docs/user-guide.md（创建中）
- **对齐文档** — docs/alignment/ 目录下的完整对齐验证文档
- **测试报告** — FINAL_TEST_REPORT.md

### Metrics

- **代码行数** — 15,000+ 行 Go 代码
- **测试用例** — 120+ 个测试用例
- **测试覆盖率** — 69.4%（核心模块 > 70%）
- **对齐度** — 95%（从 75% 提升）
- **性能** — 所有指标达标

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

[1.0.0]: https://github.com/suifei/gopherpaw/releases/tag/v1.0.0
[0.1.0]: https://github.com/suifei/gopherpaw/releases/tag/v0.1.0
