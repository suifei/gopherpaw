# CoPaw Module Map & Key File Index

All paths relative to `copaw-source/src/copaw/`.

## 1. Agent Module

Core ReAct agent implementation.

| File | Class/Function | Responsibility |
|------|---------------|----------------|
| `agents/react_agent.py` | `CoPawAgent` | Main agent, extends ReActAgent. Builds toolkit, system prompt, hooks |
| `agents/command_handler.py` | `CommandHandler` | System commands (`/compact`, `/new`, etc.) |
| `agents/model_factory.py` | `create_model()` | LLM and formatter creation from config |
| `agents/prompt.py` | `build_system_prompt_from_working_dir()` | System prompt from AGENTS.md, SOUL.md, PROFILE.md |
| `agents/hooks.py` | `BootstrapHook`, `MemoryCompactionHook` | Lifecycle hooks |

**Key flow**: User message -> `AgentRunner.query_handler` -> `CoPawAgent.__call__` -> ReAct loop (Think -> Act -> Observe) -> Response

## 2. Runner Module

Application runtime and session management.

| File | Class/Function | Responsibility |
|------|---------------|----------------|
| `app/runner/runner.py` | `AgentRunner` | Main runtime, init/query/shutdown handlers |
| `app/runner/session.py` | `SafeJSONSession` | JSON session with safe filenames |
| `app/runner/manager.py` | `ChatManager` | Chat CRUD, auto-registration from channels |
| `app/runner/api.py` | REST endpoints | `/chats` API |
| `app/runner/command_dispatch.py` | `run_command_path()` | Command routing without creating agent |

**Key flow**: HTTP request -> FastAPI -> `AgentRunner.query_handler` -> command dispatch OR agent creation -> session save

## 3. Channels Module

Messaging platform adapters.

| File | Class/Function | Responsibility |
|------|---------------|----------------|
| `app/channels/base.py` | `BaseChannel` | Abstract channel with process handler, renderer, debounce |
| `app/channels/manager.py` | `ChannelManager` | Starts channels, message queues, consumers |
| `app/channels/registry.py` | `BUILTIN_CHANNEL_TYPES` | Channel discovery and registration |
| `app/channels/schema.py` | `ChannelAddress`, `ChannelMessageConverter` | Protocol types |
| `app/channels/renderer.py` | Message rendering | Format responses for channels |

**Supported channels**: iMessage, Discord, DingTalk, Feishu, QQ, Telegram, Console, Voice (Twilio)

Each channel is in its own subdirectory: `app/channels/telegram/`, `app/channels/discord_/`, etc.

## 4. Tools Module

Built-in agent tools.

| File | Tool Function(s) | Responsibility |
|------|------------------|----------------|
| `agents/tools/file_io.py` | `read_file`, `write_file`, `edit_file`, `append_file` | File operations |
| `agents/tools/file_search.py` | `grep_search`, `glob_search` | File search |
| `agents/tools/shell.py` | `execute_shell_command` | Shell execution |
| `agents/tools/send_file.py` | `send_file_to_user` | File delivery |
| `agents/tools/browser_control.py` | `browser_use` | Browser automation |
| `agents/tools/memory_search.py` | `create_memory_search_tool` | Memory recall |
| `agents/tools/get_current_time.py` | `get_current_time` | Time utility |
| `agents/tools/desktop_screenshot.py` | `desktop_screenshot` | Screen capture |

Tools are registered in `CoPawAgent._create_toolkit()`.

## 5. Memory Module

ReMe-based memory management.

| File | Class/Function | Responsibility |
|------|---------------|----------------|
| `agents/memory/memory_manager.py` | `MemoryManager` | Extends ReMe, handles compaction/search |
| `agents/memory/agent_md_manager.py` | `AgentMdManager` | Agent markdown file management |

Features: semantic search (vector + full-text), tool result compaction, conversation summarization.

## 6. Providers Module

LLM backend management.

| File | Class/Function | Responsibility |
|------|---------------|----------------|
| `providers/registry.py` | Provider definitions | Built-in provider registry |
| `providers/store.py` | `ProviderStore` | Load/save provider settings (JSON) |
| `providers/models.py` | `ProviderDefinition`, `ModelInfo` | Data models |
| `providers/openai_chat_model_compat.py` | `OpenAIChatModelCompat` | OpenAI-compatible wrapper |
| `providers/ollama_manager.py` | `OllamaManager` | Local Ollama model listing |

**Built-in providers**: ModelScope, DashScope, OpenAI, Azure OpenAI, Anthropic, Ollama, llama.cpp, MLX

## 7. Config Module

Configuration management.

| File | Class/Function | Responsibility |
|------|---------------|----------------|
| `config/config.py` | `Config`, `ChannelConfig`, `MCPConfig`, `AgentsConfig` | Pydantic config models |
| `config/utils.py` | `load_config()`, `read_last_api()` | Config loading and path helpers |
| `config/watcher.py` | `ConfigWatcher` | Hot-reload on config file change |

## 8. Skills Module

Skill management and hub.

| File | Class/Function | Responsibility |
|------|---------------|----------------|
| `agents/skills_manager.py` | `SkillService` | Skill sync: builtin -> customized -> active |
| `agents/skills_hub.py` | `search_hub_skills()`, `install_skill_from_hub()` | Hub client |

Skill layout: `SKILL.md` (YAML front matter) + optional `references/` and `scripts/`.

Built-in skills: cron, docx, pdf, pptx, xlsx, file_reader, news, himalaya, browser_visible, dingtalk_channel.

## 9. CLI Module

Command-line interface.

| File | Command | Responsibility |
|------|---------|----------------|
| `cli/main.py` | `copaw` | Click CLI group |
| `cli/app_cmd.py` | `copaw app` | Run FastAPI via uvicorn |
| `cli/init_cmd.py` | `copaw init` | Interactive setup |
| `cli/skills_cmd.py` | `copaw skills` | Skill management |
| `cli/channels_cmd.py` | `copaw channels` | Channel management |
| `cli/providers_cmd.py` | `copaw models` | Provider/model selection |

## Module Dependency Graph

```
CLI (cli/)
 └── App (app/_app.py)
      ├── Runner (app/runner/)
      │    ├── Agent (agents/react_agent.py)
      │    │    ├── Tools (agents/tools/)
      │    │    ├── Memory (agents/memory/)
      │    │    ├── Skills (agents/skills_manager.py)
      │    │    └── Providers (providers/) via model_factory
      │    └── Session (app/runner/session.py)
      ├── Channels (app/channels/)
      │    └── Runner (via message queue)
      ├── Crons (app/crons/)
      │    └── Runner (via heartbeat)
      └── Config (config/)
```
