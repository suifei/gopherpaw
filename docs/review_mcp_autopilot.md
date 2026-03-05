# MCP 实现审查报告（gopherpaw-autopilot 工作）

**审查日期**: 2026-03-06  
**范围**: `internal/mcp/` 对齐 CoPaw `copaw-source/src/copaw/app/routers/mcp.py`

---

## Summary

**结论: PASS（已修复 2 项问题后通过）**

- 与 CoPaw 的**客户端配置与三种传输**已对齐。
- 架构依赖、接口设计、并发与契约文档均符合规范。
- 测试文件存在语法错误与缺失实现，已修复；`envToSlice` 未定义问题已修复。

---

## 1. 与 CoPaw 对齐情况

| 维度 | CoPaw (mcp.py) | GopherPaw (internal/mcp) | 状态 |
|------|----------------|--------------------------|------|
| 传输类型 | stdio, streamable_http, sse | StdioTransport, HTTPTransport, SSETransport | 一致 |
| 配置字段 | key, name, description, enabled, transport, url, headers, command, args, env, cwd | MCPServerConfig 含 Name, Description, Enabled, Transport, URL, Headers, Command, Args, Env, Cwd | 一致 |
| 客户端 CRUD | REST API (list/get/create/update/toggle/delete) | 无 HTTP 路由，仅 LoadConfig/AddClient/RemoveClient/Reload | 预期差异（Go 侧为库，API 由上层提供） |
| 环境变量脱敏 | _mask_env_value 用于 API 响应 | 未实现 | 可选增强 |

**说明**: CoPaw 的 `mcp.py` 是 **HTTP 路由层**（FastAPI），负责配置的 CRUD 与持久化；GopherPaw 的 `internal/mcp` 是 **客户端库**，负责连接 MCP 服务器并暴露 `agent.Tool`。配置的读写、持久化及 REST 暴露应由 `cmd/` 或未来的 HTTP 层使用本包 + config 实现，当前设计合理。

---

## 2. 架构合规性 (architecture-review)

### Check 1: Layer dependency

- `internal/mcp` 依赖: `internal/agent`（Tool 接口）、`internal/config`、`pkg/logger`。
- 等价于 tools 层：依赖 agent 接口与 config，符合「tools → config、agent 仅接口」的规则。
- **PASS**

### Check 2: Interface-first

- `MCPManager.GetTools()` 返回 `[]agent.Tool`，不暴露具体 Transport。
- 外部仅通过 `NewMCPClient(name, cfg)` 和 `Transport` 接口使用，具体实现为 `*StdioTransport` 等。
- `Reload` 内对 `*StdioTransport`/`*HTTPTransport`/`*SSETransport` 的类型断言仅用于从已存在客户端反推配置做 diff，属工厂/编排层合理用法。
- **PASS**

### Check 3: Error handling

- 错误均带上下文（`fmt.Errorf("...: %w", err)`）。
- 未发现忽略返回值（如 `_ = ...`）在关键路径上。
- **PASS**

### Check 4: Test coverage

- 已修复并保留：ParseMCPConfig 多格式、NewMCPClient 校验、Manager Load/Start/GetTools、无效 JSON、single 格式 Description。
- 表驱动、边界用例（空输入、缺字段、非法 transport）已覆盖。
- **PASS**

### Check 5: Concurrency

- `MCPManager` 使用 `sync.RWMutex` 保护 `clients`/`tools`。
- 各 Transport 实现内部使用 `sync.Mutex` 保护状态与 IO。
- **PASS**

### Check 6: Contract alignment

- `docs/api_spec.md` 已包含 MCPConfig、MCPServerConfig、Transport、MCPClient、MCPManager 及三种 Transport 说明。
- **PASS**

---

## 3. 已修复问题

### 3.1 client_test.go 语法与结构错误（Critical）

- **现象**: 缺少 `package mcp_test`、import 块断裂、表驱动用例缺少 `{}`、误用 `err` 字段（ParseMCPConfig 不返回校验错误）。
- **修复**: 重写测试文件，统一 `package mcp_test` 与 import；ParseMCPConfig 只测解析与多格式；NewMCPClient 校验单独为 `TestNewMCPClient_Validation`；默认 stdio 时接受 `Transport == ""`。

### 3.2 envToSlice 未定义（Critical）

- **现象**: `transport_stdio.go` 调用 `envToSlice(t.env)`，但该函数在重构时从 client 中移除未迁移。
- **修复**: 在 `transport_stdio.go` 中新增 `envToSlice` 实现。

### 3.3 Single 格式缺少 Description（Info）

- **现象**: ParseMCPConfig 的 single 格式未解析 `description`，与 CoPaw 及 api_spec 不一致。
- **修复**: 在 single 格式的临时 struct 和返回的 `config.MCPServerConfig` 中增加 `Description` 字段。

---

## 4. 建议（非必须）

1. **REST API**: 若需与 CoPaw 行为完全一致，可在 `cmd/` 或单独 router 包中增加 `/mcp` 的 list/get/create/update/toggle/delete，内部调用 `config.Load/Save` 与 `MCPManager.AddClient/RemoveClient/Reload`。
2. **环境变量脱敏**: 若对外暴露 MCP 配置（如调试接口），可增加与 CoPaw 类似的 `_mask_env_value` 用于 env/headers 展示。
3. **E2E 测试**: 对 stdio 传输可使用真实 MCP 子进程或 mock 进程，避免依赖系统 `echo`（Windows 上为 shell 内置，当前测试已容忍 Start 失败）。

---

## 5. 变更文件清单

| 文件 | 变更类型 |
|------|----------|
| `internal/mcp/client_test.go` | 重写（语法修复、用例拆分、单格式 Description 测试） |
| `internal/mcp/client.go` | 单格式解析增加 Description |
| `internal/mcp/transport_stdio.go` | 新增 `envToSlice` |
| `docs/review_mcp_autopilot.md` | 新增（本审查报告） |

---

*审查依据: .cursor/rules/contract-first.mdc, project-context.mdc, .cursor/skills/architecture-review/SKILL.md*
