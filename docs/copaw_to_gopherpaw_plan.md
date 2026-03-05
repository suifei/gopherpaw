# CoPaw → GopherPaw 长期迭代计划

> 本文档由 `docs/copaw_python_files.txt` 与项目契约驱动生成，供 `.cursor/agents/gopherpaw-autopilot.md` 调度执行，使 GopherPaw 实现与 CoPaw 源码功能对齐（或超越）。

## 使用方式

- **Autopilot 调度**：每次启动时阅读 `CONTEXT.md`、`docs/architecture_spec.md`、`docs/api_spec.md`，再按下方「计划题词」顺序执行；或根据「按模块的执行顺序」从第一个未完成项继续。
- **人类**：可按阶段/模块勾选进度，或指定「从某模块开始」让 Autopilot 执行。
- **文件列表**：完整 Python 文件列表见 `docs/copaw_python_files.txt`（由 `scripts/list_copaw_python_files.sh` 生成）。

## 按模块的执行顺序（与 Module Build Order 一致）

依赖顺序（仅列出需 Go 实现或深度对齐的模块，Skills 脚本、tests、setup 等见后表）：

| 序号 | Go 目标 | CoPaw 源码（主要） | 状态 |
|------|---------|---------------------|------|
| 1 | `internal/config/` | config/config.py, config/utils.py, config/watcher.py | 已实现 |
| 2 | `internal/llm/` | providers/registry.py, store.py, openai_chat_model_compat.py, models.py; local_models/* | 已实现 |
| 3 | `internal/memory/` | agents/memory/memory_manager.py, agent_md_manager.py | 已实现 |
| 4 | `internal/tools/` | agents/tools/*.py (file_io, shell, file_search, memory_search, get_current_time, browser_*, desktop_screenshot, send_file) | 已实现 |
| 5 | `internal/agent/` | agents/react_agent.py, prompt.py, schema.py, command_handler.py, model_factory.py, skills_hub.py, skills_manager.py; agents/hooks/*; agents/utils/*; app/runner/runner.py, session.py, command_dispatch.py, daemon_commands.py, manager.py, models.py, utils.py, api.py; app/runner/repo/* | 已实现 |
| 6 | `internal/channels/` | app/channels/base.py, manager.py, registry.py, renderer.py, schema.py, utils.py; console/, dingtalk/, feishu/, qq/, telegram/, discord_/ | 已实现 |
| 7 | `internal/scheduler/` | app/crons/manager.py, executor.py, heartbeat.py, models.py, api.py; app/crons/repo/* | 已实现 |
| 8 | `internal/mcp/` | app/mcp/manager.py, app/mcp/watcher.py; app/routers/mcp.py | 已实现 |
| 9 | `cmd/gopherpaw/` | cli/main.py, app_cmd.py, channels_cmd.py, chats_cmd.py, cron_cmd.py, daemon_cmd.py, env_cmd.py, init_cmd.py, providers_cmd.py, skills_cmd.py, uninstall_cmd.py, clean_cmd.py; cli/utils.py, http.py | 已实现 |
| 10 | 运行时/配置 | config/__init__.py; envs/store.py; app/_app.py, console_push_store.py, download_task_store.py | 已实现/部分 |
| 11 | 契约与文档 | - | 持续：每次实现后更新 architecture_spec.md、api_spec.md、CONTEXT.md |

## 按 CoPaw 文件的细粒度映射

- **已实现**：GopherPaw 已有对应实现，可做「验证/增强」。
- **Go 实现**：需在 internal/ 或 cmd/ 中实现或对齐。
- **运行时调用**：通过 Python/Bun 运行时调用，不要求 Go 重写。
- **跳过/参考**：tests、setup、可选功能（如 iMessage/Voice/Tunnel），仅作参考或后续按需。

| CoPaw 源码路径 | Go 目标 / 策略 | 说明 |
|----------------|----------------|------|
| config/config.py, utils.py, watcher.py | internal/config/ | 已实现，验证热重载与契约 |
| providers/registry.py, store.py, openai_chat_model_compat.py, models.py | internal/llm/ | 已实现，验证多 Provider、流式 |
| local_models/* | internal/llm/ 或扩展 | 本地模型（Ollama/llama.cpp/MLX）可做可选增强 |
| agents/memory/memory_manager.py, agent_md_manager.py | internal/memory/ | 已实现 |
| agents/tools/file_io.py, shell.py, file_search.py, memory_search.py, get_current_time.py, browser_*.py, desktop_screenshot.py, send_file.py | internal/tools/ | 已实现 |
| agents/react_agent.py, prompt.py, schema.py, command_handler.py, model_factory.py, skills_hub.py, skills_manager.py | internal/agent/ | 已实现 |
| agents/hooks/bootstrap.py, memory_compaction.py | internal/agent/ | 已实现 |
| agents/utils/* (file_handling, message_processing, token_counting, tool_message_utils, setup_utils) | internal/agent/ | 已实现或分散在 agent/tools/memory |
| app/runner/runner.py, session.py, command_dispatch.py, daemon_commands.py, manager.py, models.py, utils.py, api.py, query_error_dump.py | internal/agent/, internal/channels/ | 已实现 |
| app/runner/repo/base.py, json_repo.py | internal/agent/ 或 internal/scheduler/ | 已实现（内存/文件持久化） |
| app/channels/base.py, manager.py, registry.py, renderer.py, schema.py, utils.py | internal/channels/ | 已实现 |
| app/channels/console/, dingtalk/, feishu/, qq/, telegram/, discord_/ | internal/channels/ | 已实现（iMessage/Voice 跳过） |
| app/crons/manager.py, executor.py, heartbeat.py, models.py, api.py, repo/* | internal/scheduler/ | 已实现 |
| app/mcp/manager.py, watcher.py | internal/mcp/ | 已实现 |
| app/routers/* (agent, config, console, envs, local_models, mcp, ollama_models, providers, skills, voice, workspace) | 参考/Web 可选 | Web 控制台跳过；逻辑已落在 CLI/channels |
| cli/main.py, app_cmd.py, channels_cmd.py, chats_cmd.py, cron_cmd.py, daemon_cmd.py, env_cmd.py, init_cmd.py, providers_cmd.py, skills_cmd.py, uninstall_cmd.py, clean_cmd.py, utils.py, http.py | cmd/gopherpaw/ | 已实现 |
| agents/skills/docx|pptx|xlsx|pdf/scripts/** | 运行时调用 | Python 脚本由 GopherPaw 通过 runtime 调用，不 Go 重写 |
| tunnel/* | 可选/跳过 | 内网穿透，可后续按需 |
| utils/logging.py | pkg/logger/ | 已实现 |
| constant.py, __init__.py, __main__.py, __version__.py | cmd/gopherpaw/, 根版本 | 已实现 |
| setup.py, tests/* | 参考 | 不要求 Go 复刻，测试可作行为参考 |

## 计划题词（供 Autopilot 一次执行或按条调度）

以下每行是一条可独立发给 gopherpaw-autopilot 的题词，按顺序执行即可长期迭代对齐 CoPaw。

```
1. 阅读 CONTEXT.md、docs/architecture_spec.md、docs/api_spec.md，确认 internal/config 与 copaw-source config 对齐并更新契约与 CONTEXT。
2. 阅读 CONTEXT.md 与契约，确认 internal/llm 与 providers/*、local_models/* 对齐，补充缺失 API 或文档。
3. 确认 internal/memory 与 agents/memory 对齐，更新 feature_matrix 与 CONTEXT。
4. 确认 internal/tools 与 agents/tools 全量工具对齐（含 browser、screenshot、send_file），更新契约。
5. 确认 internal/agent 与 agents/react_agent、runner、session、hooks、utils 对齐，更新契约与 CONTEXT。
6. 确认 internal/channels 与 app/channels 各渠道对齐，更新契约。
7. 确认 internal/scheduler 与 app/crons 对齐，更新契约。
8. 确认 internal/mcp 与 app/mcp、app/routers/mcp 对齐，更新契约。
9. 确认 cmd/gopherpaw 与 cli/* 所有子命令对齐，更新契约与 CONTEXT。
10. 按 docs/feature_matrix.md 做一次全量自检，更新「当前状态」与「更新日志」。
11. 运行 go vet ./... 与 go test ./...，修复所有问题并回填 docs 与 CONTEXT。
```

## 单条「从某模块开始」的题词模板

- 「从 config 开始，按 copaw_to_gopherpaw_plan 执行到通过测试并更新契约。」
- 「从 llm 开始，按 copaw_to_gopherpaw_plan 执行到通过测试并更新契约。」
- 「从 memory 开始，按 copaw_to_gopherpaw_plan 执行到通过测试并更新契约。」
- 「从 tools 开始，按 copaw_to_gopherpaw_plan 执行到通过测试并更新契约。」
- 「从 agent 开始，按 copaw_to_gopherpaw_plan 执行到通过测试并更新契约。」
- 「从 channels 开始，按 copaw_to_gopherpaw_plan 执行到通过测试并更新契约。」
- 「从 scheduler 开始，按 copaw_to_gopherpaw_plan 执行到通过测试并更新契约。」
- 「从 mcp 开始，按 copaw_to_gopherpaw_plan 执行到通过测试并更新契约。」
- 「从 cmd/gopherpaw 开始，按 copaw_to_gopherpaw_plan 执行到通过测试并更新契约。」
- 「按 docs/copaw_to_gopherpaw_plan.md 的计划题词 1–11 顺序执行一遍，并输出每步状态。」

## 更新本计划

- 新增 CoPaw 文件时：运行 `scripts/list_copaw_python_files.sh` 更新 `docs/copaw_python_files.txt`，再在本文档中补充「按 CoPaw 文件的细粒度映射」与所需计划题词。
- 完成某模块后：在「按模块的执行顺序」表中将对应行标为已实现，并在 CONTEXT.md 更新日志中记录。
