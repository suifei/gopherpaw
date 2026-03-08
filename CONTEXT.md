# GopherPaw - 项目上下文说明

> 这是给 AI 协作的上下文文件，每次开始开发前请让 AI 先阅读此文件。

## 项目概述

- **项目名称**: GopherPaw
- **项目定位**: CoPaw 的 Go 语言复刻版
- **目标**: 用 Go 语言实现与 CoPaw 相同的功能，获得更好的性能和部署体验
- **GitHub**: [https://github.com/suifei/gopherpaw](https://github.com/suifei/gopherpaw)

## 原始项目信息

- **源项目**: CoPaw (阿里巴巴开源)
- **源语言**: Python
- **源码位置**: `./copaw-source/`
- **GitHub**: [https://github.com/agentscope-ai/CoPaw](https://github.com/agentscope-ai/CoPaw)

### CoPaw 核心架构

```
┌─────────────────────────────────────────────────┐
│  消息渠道层  │ DingTalk │ Feishu │ QQ │ Discord │
├─────────────────────────────────────────────────┤
│  Agent 运行时 │ AgentScope Runtime              │
├─────────────────────────────────────────────────┤
│  记忆系统     │ ReMe (短期/长期记忆)             │
├─────────────────────────────────────────────────┤
│  工具层       │ 文件操作 │ 搜索 │ API调用        │
├─────────────────────────────────────────────────┤
│  LLM 层       │ Qwen │ GPT │ Claude │ 本地模型  │
└─────────────────────────────────────────────────┘
```

## Go 技术选型


| 模块     | Go 方案                    | 说明                |
| ------ | ------------------------ | ----------------- |
| LLM 接口 | `sashabaranov/go-openai` | OpenAI 兼容客户端      |
| 配置管理   | `spf13/viper`            | YAML/JSON 配置      |
| 数据库    | `sqlite3` + `badger`     | 关系型 + KV 存储       |
| 日志     | `uber-go/zap`            | 高性能日志             |
| 定时任务   | `robfig/cron`            | Cron 调度           |
| 消息平台   | `discordgo`, `telebot`   | Discord, Telegram |


## 目标目录结构

```
gopherpaw/
├── cmd/
│   └── gopherpaw/
│       └── main.go           # 入口
├── internal/
│   ├── agent/                # Agent 核心
│   ├── llm/                  # LLM 接口
│   ├── memory/               # 记忆系统
│   ├── tools/                # 工具集
│   ├── channels/             # 消息渠道
│   ├── scheduler/            # 任务调度
│   ├── runtime/              # Python/Bun 运行时管理
│   ├── app/                  # 应用生命周期管理
│   └── config/               # 配置管理
├── docker/                   # Docker 桌面容器
│   ├── Dockerfile.desktop    # 桌面容器镜像
│   ├── docker-compose.yml    # 容器编排
│   └── scripts/              # 启动脚本
├── pkg/                      # 公共库
├── configs/
│   └── config.yaml           # 配置文件
├── copaw-source/             # Python 源码参考
├── docs/                     # 契约文档
│   ├── architecture_spec.md  # 系统架构契约
│   └── api_spec.md           # 接口定义契约
├── go.mod
├── CONTEXT.md                # 本文件
└── README.md
```

## 运行时环境

GopherPaw 支持 Python 和 Bun/Node.js 运行时，用于 Skills 和 MCP 工具扩展：

### Python 环境

- **用途**：文档处理（docx/xlsx/pdf）、浏览器自动化等 Skills
- **推荐**：使用虚拟环境隔离依赖
- **配置**：`runtime.python.venv_path`
- **设置命令**：`gopherpaw env setup`

### Bun 环境

- **用途**：JavaScript/TypeScript Skills 和 MCP 服务器
- **自动下载**：首次运行时自动下载到 `~/.gopherpaw/bin/bun`
- **配置**：`runtime.node.runtime = "bun"`

## 开发阶段

### 阶段一：基础设施

- 项目骨架搭建
- AI 协作基础设施（Rules/Skills/契约文档）
- 配置系统 (`internal/config/`)：已与 copaw-source `config/config.py` 对齐，统一 YAML 配置并扩展 server/llm/memory/log/skills/runtime；差异与扩展见 `docs/architecture_spec.md`「internal/config 与 CoPaw 对齐说明」
- 日志系统 (`pkg/logger/`)
- LLM 接口层 (`internal/llm/`)：已与 `providers/*`、`local_models/*` 对齐（注册、OpenAI/Ollama、ModelRouter、Formatter、Downloader）；未复刻 providers.json 持久化、自定义 Provider CRUD、发现/测通 API、本地 llamacpp/mlx 推理，见 `docs/architecture_spec.md`「internal/llm 与 CoPaw providers/*、local_models/* 对齐说明」
- 核心类型定义 (`internal/agent/types.go`)
- 项目入口 (`cmd/gopherpaw/main.go`)

### 阶段二：核心功能

- Agent 核心逻辑 (`internal/agent/`)：已与 `agents/react_agent`、`app/runner`（runner、session）、`agents/hooks`、`agents/utils`、`agents/command_handler` 对齐；ReAct、Hooks、Utils、魔法命令、会话管理均已覆盖，差异见 `docs/architecture_spec.md`「internal/agent 与 CoPaw agents/react_agent、runner、session、hooks、utils 对齐说明」
- 工具系统 (`internal/tools/`)
- 记忆系统 (`internal/memory/`)：已与 `agents/memory`（MemoryManager + AgentMdManager）对齐，MemoryStore 覆盖短期/长期/压缩/混合检索/文件监控/Embedding；差异见 `docs/architecture_spec.md`「internal/memory 与 CoPaw agents/memory 对齐说明」
- 工具系统 (`internal/tools/`)：已与 `agents/tools` 全量对齐（含 browser_use、desktop_screenshot、send_file_to_user），扩展 web_search、http_request、switch_model；差异见 `docs/architecture_spec.md`「internal/tools 与 CoPaw agents/tools 全量对齐说明」

### 阶段三：消息渠道

- Telegram 接入
- Discord 接入
- DingTalk/Feishu 接入

### 阶段四：高级功能

- 定时任务 (`internal/scheduler/`)
- 多 Agent 协作
- Web UI (可选)

## 开发规范

1. **契约先行**: 所有核心变动先更新 `docs/` 下契约文档（军规 1）
2. **资产脱水**: 通用模式固化为 `.cursor/skills/` 资产（军规 2）
3. **测试底线**: 核心模块必须有极端边界测试（军规 3）
4. **代码风格**: 遵循 Go 标准规范，使用 `gofmt`, `golint`
5. **注释**: 所有导出函数必须有 godoc 注释
6. **错误处理**: 不忽略错误，使用 `fmt.Errorf` 包装
7. **接口驱动**: 先定义接口再实现，依赖注入优先

## 测试用 OPENAI 端点

- APIKEY:  c9186fd1c99fb2e3cbe0cdbe9709fe746a4823d0b9b4322d
- BASEURL: https://llm.meta2cs.cn/v1
- MODEL:   gopherpaw

## AI 协作基础设施

### Cursor Rules (`.cursor/rules/`)


| 规则                        | 作用域               | 用途            |
| ------------------------- | ----------------- | ------------- |
| `project-context.mdc`     | 全局                | 项目上下文自动注入     |
| `contract-first.mdc`      | 全局                | 契约驱动开发流程      |
| `go-standards.mdc`        | `**/*.go`         | Go 编码规范       |
| `architecture-layers.mdc` | `internal/`**     | 四层架构约束        |
| `copaw-reference.mdc`     | `copaw-source/**` | Python 源码参考指南 |
| `testing-tdd.mdc`         | `**/*_test.go`    | TDD 测试规范      |
| `spec-documents.mdc`      | `docs/**/*.md`    | 契约文档维护规范      |


### Cursor Skills (`.cursor/skills/`)


| 技能                    | 触发场景                      |
| --------------------- | ------------------------- |
| `python-to-go`        | 翻译/转换 CoPaw Python 模块为 Go |
| `add-go-module`       | 新增 Go 功能模块                |
| `asset-dehydration`   | 提炼/脱水通用模式为永久资产            |
| `copaw-analysis`      | 分析 CoPaw 源码模块             |
| `spec-driven-dev`     | 四阶段确认流启动新功能开发             |
| `architecture-review` | 架构合规性审查                   |


## 当前状态

- **版本**: v0.3.0
- **阶段**: 规划-执行分离模式完成 ✅
- **进度**: config、llm、memory、tools、agent、channels、scheduler、mcp、CLI、app 均已对齐并通过生产环境验证；Docker 桌面容器化模块完成（internal/app/ + docker/）；规划-执行分离模式实现（internal/agent/planner.go、executor.go、context_manager.go、capability_extractor.go、skill_hook.go、缓存系统）；CLI 任务参数支持（启动时直接传入任务、--once 标志）；HTML 内容自动提取（HTTP 工具自动检测 Content-Type）；Bun 1.3.10 已安装并配置；共 23 个 Skills；测试覆盖率：agent 74.7%、config 79.2%、llm 62.3%、mcp 72.1%、memory 59.4%、runtime 82.4%、scheduler 96.2%、skills 62.3%、tools 51.3%、app 100%；`go vet ./...` 与 `go test -short ./...` 通过；测试和监控文档已完善
- **功能对标**: 详见 [docs/feature_matrix.md](docs/feature_matrix.md)（有效完成度 100%，排除跳过项）
- **新增模块**:
  - `internal/agent/planner.go` - TaskPlanner 任务规划器
  - `internal/agent/executor.go` - Executor 执行器
  - `internal/agent/context_manager.go` - ContextManager 上下文管理器
  - `internal/agent/capability_extractor.go` - CapabilityExtractor 能力提取器
  - `internal/agent/skill_hook.go` - SkillHook Skill 钩子
  - `internal/agent/capability_registry.go` - 能力注册表缓存
  - `internal/agent/cache_file.go` - 文件持久化缓存
  - `internal/app/` - 应用生命周期管理（Start/Stop/RestartServices/HealthCheck），6/6 测试通过
  - `docker/` - 桌面容器化（Ubuntu + XFCE + TigerVNC + noVNC），浏览器访问 http://localhost:6080/vnc.html
- **验证报告**:
  - [docs/production_verification_report.md](docs/production_verification_report.md) - 生产环境验证 ✅
  - [docs/bun_nodejs_installation_guide.md](docs/bun_nodejs_installation_guide.md) - Bun/Node.js 安装指南 ✅
  - [docs/js_skills_test_report.md](docs/js_skills_test_report.md) - JS Skills 测试 ✅
  - [docs/new_js_skills_report.md](docs/new_js_skills_report.md) - 新增 Skills 报告 ✅
  - [docs/alignment/ALIGNMENT_VERIFICATION_REPORT.md](docs/alignment/ALIGNMENT_VERIFICATION_REPORT.md) - 对齐验证报告 ✅
- **测试指南**:
  - [docs/channel_testing_guide.md](docs/channel_testing_guide.md) - 渠道测试指南 ✅
  - [docs/mcp_transport_testing_guide.md](docs/mcp_transport_testing_guide.md) - MCP 传输测试指南 ✅
  - [docs/performance_benchmark_guide.md](docs/performance_benchmark_guide.md) - 性能基准测试指南 ✅
- **下一步**: 生产环境部署；真实渠道测试（Telegram/Discord/钉钉/飞书/QQ）；性能优化；用户文档完善；Docker 容器生产化（SSL、多用户、持久化）

## 更新日志

- 2026-03-09: **v0.3.0 规划-执行分离模式完成**：TaskPlanner（任务规划器）、Executor（执行器）、ContextManager（上下文管理器）、CapabilityExtractor（能力提取器）、SkillHook（Skill 钩子）、缓存系统（capability_registry.go、cache_file.go）；CLI 任务参数支持（启动时直接传入任务、--once 标志）；HTML 内容自动提取（HTTP 工具自动检测 Content-Type）；LLM 400 错误修复（compressMessages 消息结构修复）；提示词加载系统改进（对齐 CoPaw、移除 YAML 前言）；文档更新（RELEASE_NOTES.md、CHANGELOG.md、README.md、QUICKSTART.md、CONTEXT.md）
- 2026-03-08: **Docker 桌面容器化模块完成**：internal/app/ 模块（App 生命周期管理，6/6 测试通过）；Docker Desktop Container（Ubuntu + XFCE + TigerVNC + noVNC）；VNC 配置修复（tigervnc-tools、密码文件、语法错误）；容器验证通过（端口 6080/5901 可访问，浏览器访问 http://localhost:6080/vnc.html）；契约文档更新（architecture_spec.md、api_spec.md、CONTEXT.md）
- 2026-03-07: **测试和监控体系完善**：创建渠道测试指南（channel_testing_guide.md）涵盖 6 个渠道的单元测试、集成测试、端到端测试；创建 MCP 传输测试指南（mcp_transport_testing_guide.md）涵盖 Stdio/HTTP/SSE 三种传输；创建性能基准测试指南（performance_benchmark_guide.md）包含性能指标、优化建议、持续监控；创建基准测试脚本（scripts/benchmark.sh）支持完整测试、快速测试、内存分析、CPU 分析、竞态检测
- 2026-03-07: **测试稳定性修复**：修复 browser_e2e_test.go 构建标签（改为 chrome 标签，默认不运行）；增强 web_and_browser_test.go 网络超时处理（添加 -short 跳过和超时跳过机制）；所有测试在 -short 模式下通过（覆盖率：agent 74.7%、config 79.2%、llm 62.3%、mcp 72.1%、memory 59.4%、runtime 82.4%、scheduler 96.2%、skills 62.3%、tools 51.3%）
- 2026-03-07: **密钥管理增强**：新增 security_best_practices skill；增强 config.yaml.example 安全说明；添加 pre-commit hook 检测敏感信息；Agent 新增工具重名策略、环境上下文构建；Config 新增密钥目录管理、环境变量辅助函数；Skills 添加 enable/disable/create/delete 命令；MCP 添加自动重连机制；新增 6 个 active_skills 和 8 个团队共享技能
- 2026-03-07: **CH-002 完成**：实现 media_dir 支持；为 Config 结构体添加 MediaDir 字段；实现 ResolveMediaDir 函数（支持环境变量覆盖）；构建成功
- 2026-03-07: **CH-001 完成**：实现 filter_tool_messages 配置支持；为所有渠道配置（Console/Telegram/Discord/DingTalk/Feishu/QQ）添加 FilterToolMessages 字段；构建成功
- 2026-03-07: **CFG-003 完成**：实现所有环境变量（21 个），对齐 CoPaw COPAW_* → GOPHERPAW_*；新增辅助函数（GetEnvString/Bool/Int/Float/Slice）；新增专用函数（IsRunningInContainer/GetEnabledChannels/GetCORSOrigins/GetConfigFile/GetJobsFile/GetChatsFile/GetHeartbeatFile/GetModelProviderCheckTimeout）；更新 api_spec.md；添加 13 个测试用例（共 31 个子测试）；go vet 与测试通过
- 2026-03-07: **CFG-002 完成**：实现密钥目录支持（GetSecretDir/GetEnvsJSONPath/GetProvidersJSONPath/EnsureSecretDir），对齐 CoPaw SECRET_DIR；更新 architecture_spec.md、api_spec.md；添加 4 个测试用例；go vet 与测试通过
- 2026-03-07: **CFG-001 完成**：AgentConfig 重构为嵌套结构（defaults/running），对齐 CoPaw AgentsConfig；更新 architecture_spec.md、api_spec.md；修复所有引用（agent、hooks、测试）；go vet 与核心测试通过
- 2026-03-06: 按 docs/copaw_to_gopherpaw_plan.md 计划题词 1–11 顺序执行：步骤 1–9 对齐验证（config/llm/memory/tools/agent/channels/scheduler/mcp/CLI），步骤 10 全量自检 feature_matrix 与当前状态，步骤 11 go vet + go test 通过并回填 CONTEXT
- 2026-03-06: internal/tools 与 CoPaw agents/tools 全量对齐（含 browser、screenshot、send_file），更新 architecture_spec、feature_matrix、api_spec、CONTEXT
- 2026-03-06: cmd/gopherpaw 与 CoPaw cli/* 子命令对齐，更新契约与 CONTEXT
- 2026-03-06: internal/mcp 与 app/mcp、app/routers/mcp 对齐，更新契约与 CONTEXT
- 2026-03-06: internal/scheduler 与 app/crons 对齐，更新契约与 CONTEXT
- 2026-03-06: internal/channels 与 app/channels 各渠道对齐，更新契约与 CONTEXT
- 2026-03-06: internal/agent 与 CoPaw react_agent、runner、session、hooks、utils 对齐，更新契约与 CONTEXT
- 2026-03-06: internal/tools 与 agents/tools 全量对齐（含 browser、screenshot、send_file），更新契约与 CONTEXT
- 2026-03-06: internal/memory 与 CoPaw agents/memory 对齐，更新 feature_matrix（记忆系统对齐说明与 CoPaw 来源标注）与 CONTEXT（阶段二、当前进度）
- 2026-03-06: internal/llm 与 CoPaw providers/*、local_models/* 对齐并更新契约与 CONTEXT（架构对齐说明、api_spec Registry/Downloader/对齐小节）
- 2026-03-06: internal/config 与 copaw-source config 对齐：阅读 CONTEXT/architecture_spec/api_spec 与 CoPaw config 模块，更新契约（对齐说明、扩展项、差异项）与 CONTEXT 阶段一/当前状态
- 2026-03-06: MCP Transport 抽象层：Transport 接口（Start/Stop/Call/WriteNotification/IsRunning）；三种实现（StdioTransport/HTTPTransport/SSETransport）；MCPServerConfig 扩展（Name/Description/Transport/URL/Headers/Cwd）；MCPClient 重构使用 Transport 组合；对齐 CoPaw mcp.py 三种传输支持；向后兼容现有 stdio 配置
- 2026-03-05: 项目初始化，创建 CONTEXT.md
- 2026-03-05: AI 协作基础设施就绪（7 Rules + 6 Skills + 契约文档）
- 2026-03-05: 阶段一完成：types、config、logger、llm、main.go
- 2026-03-05: 阶段二完成：memory、tools、ReAct Agent、session
- 2026-03-05: 阶段三/四完成：channels(console)、scheduler、main.go 完整装配
- 2026-03-05: 新增 web_search（DuckDuckGo）、http_request 工具；优化 system_prompt 引导 Agent 使用工具
- 2026-03-05: 增强运行时日志（agent.go ReAct 循环、openai.go LLM 通讯含计时）；Console ProgressReporter 实时反馈；新增 edit_file、append_file 工具；system_prompt 增加中文支持提示
- 2026-03-05: 新增 docs/feature_matrix.md 功能对标矩阵，整理 CoPaw 83 项功能与 GopherPaw 实现状态
- 2026-03-05: 完成 12 项收尾功能：/daemon restart、reload-config；Skills ImportFromURL；MCP AddClient/RemoveClient/Reload、ParseMCPConfig；LLM downloader、SwitchProvider、providers.json；钉钉/飞书/QQ WebhookServer；configs/active_skills 内置 Skills；/switch-model
- 2026-03-05: CoPaw Agents 全量复刻：skills/hub.go、agent/hooks.go、agent/utils.go、llm/formatter.go、skills/manager.go 增强、prompt_loader.go 增强、commands.go 增强、configs/md_files/ 和 configs/active_skills/ 内置资源（11 个 Skills + 12 个 MD 模板）

---

> 提示：每次开发会话开始时，请让 AI 执行：
>
> ```
> 请阅读 ./CONTEXT.md 文件，了解项目背景后再继续。
> ```

