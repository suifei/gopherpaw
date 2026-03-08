# GopherPaw API Specification

> 契约文档：内部接口定义的唯一真理来源。所有接口变更必须先更新此文档。

### CLI 入口与子命令 (cmd/gopherpaw)

入口为 `cmd/gopherpaw/main.go`，根命令使用 cobra，默认执行 `app`。子命令与 CoPaw `cli/*` 对齐：app、channels、chats、clean、cron、daemon、env、init、models、skills；差异与子子命令范围见 `docs/architecture_spec.md`「cmd/gopherpaw 与 CoPaw cli/* 子命令对齐说明」。

#### CLI 任务参数支持（v0.3.0 新增）

```bash
# 启动时直接传入任务
./gopherpaw app "任务内容"

# --once 标志：执行后自动退出
./gopherpaw app "任务内容" --once

# 指定工作目录
./gopherpaw app "任务内容" --workdir /path/to/project
```

## 核心接口定义

### Agent 接口 (internal/agent/)

```go
// Agent processes user messages through a ReAct loop.
type Agent interface {
    // Run processes a message and returns the agent's response.
    Run(ctx context.Context, chatID string, message string) (string, error)

    // RunStream processes a message and streams response chunks.
    RunStream(ctx context.Context, chatID string, message string) (<-chan string, error)
}
```

### LLM Provider 接口 (defined in internal/agent/, implemented in internal/llm/)

```go
// LLMProvider abstracts LLM backend communication.
type LLMProvider interface {
    // Chat sends a chat completion request and returns the response.
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)

    // ChatStream sends a streaming chat completion request.
    ChatStream(ctx context.Context, req *ChatRequest) (ChatStream, error)

    // Name returns the provider identifier.
    Name() string
}

// ChatStream represents a streaming response.
type ChatStream interface {
    // Recv receives the next chunk. Returns io.EOF when done.
    Recv() (*ChatChunk, error)

    // Close releases stream resources.
    Close() error
}
```

### ChatRequest / ChatResponse 数据结构

```go
type ChatRequest struct {
    Messages    []Message      `json:"messages"`
    Tools       []ToolDef      `json:"tools,omitempty"`
    Temperature float64        `json:"temperature,omitempty"`
    MaxTokens   int            `json:"max_tokens,omitempty"`
}

type ChatResponse struct {
    Content   string     `json:"content"`
    ToolCalls []ToolCall `json:"tool_calls,omitempty"`
    Usage     Usage      `json:"usage"`
}

type Message struct {
    Role       string     `json:"role"`       // system, user, assistant, tool
    Content    string     `json:"content"`
    ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
    ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
    ID       string `json:"id"`
    Name     string `json:"name"`
    Arguments string `json:"arguments"` // JSON string
}

type ToolDef struct {
    Name        string `json:"name"`
    Description string `json:"description"`
    Parameters  any    `json:"parameters"` // JSON Schema
}

type Usage struct {
    PromptTokens     int `json:"prompt_tokens"`
    CompletionTokens int `json:"completion_tokens"`
    TotalTokens      int `json:"total_tokens"`
}

type ChatChunk struct {
    Content   string     `json:"content"`
    ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}
```

### Memory Store 接口 (defined in internal/agent/, implemented in internal/memory/)

```go
// MemoryStore manages conversation memory and long-term recall.
type MemoryStore interface {
    // Save stores a message in the conversation history.
    Save(ctx context.Context, chatID string, msg Message) error

    // Load retrieves recent conversation history.
    Load(ctx context.Context, chatID string, limit int) ([]Message, error)

    // Search performs semantic search across memory (hybrid: vector + BM25).
    Search(ctx context.Context, chatID string, query string, topK int) ([]MemoryResult, error)

    // Compact compresses old conversation history (with LLM summary when available).
    Compact(ctx context.Context, chatID string) error

    // GetCompactSummary returns the current compact summary for a chat (empty if none).
    GetCompactSummary(ctx context.Context, chatID string) (string, error)

    // SaveLongTerm persists content to long-term memory (MEMORY.md or memory/YYYY-MM-DD.md).
    SaveLongTerm(ctx context.Context, chatID string, content string, category string) error

    // LoadLongTerm loads all long-term memory content for a chat.
    LoadLongTerm(ctx context.Context, chatID string) (string, error)
}

type MemoryResult struct {
    Content   string  `json:"content"`
    Score     float64 `json:"score"`
    Timestamp int64   `json:"timestamp"`
    ChunkID   string  `json:"chunk_id,omitempty"` // Unique chunk identifier for dedup
}

// WithMemoryStore attaches MemoryStore to context for tools (e.g. memory_search).
func WithMemoryStore(ctx context.Context, store MemoryStore) context.Context
```

### 记忆系统扩展类型 (internal/memory/)

```go
// CompactConfig holds compaction parameters.
type CompactConfig struct {
    Threshold   int     // Token threshold to trigger auto-compact (default 100000)
    KeepRecent  int     // Messages to keep after compact (default 3)
    Ratio       float64 // Target compression ratio 0.0-1.0 (default 0.7)
}

// EmbeddingConfig holds embedding API settings.
type EmbeddingConfig struct {
    APIKey     string
    BaseURL    string
    Model      string
    Dimensions int
    MaxCache   int // LRU cache size (default 2000)
}

// Chunk represents an indexed memory chunk for hybrid search.
type Chunk struct {
    ID        string
    Content   string
    Timestamp int64
    Vector    []float32 // Embedding vector (nil if not embedded)
}
```

### Tool 接口 (defined in internal/agent/, implemented in internal/tools/)

```go
// Tool represents a callable tool that the agent can invoke.
type Tool interface {
    // Name returns the tool identifier.
    Name() string

    // Description returns a human-readable description for LLM tool selection.
    Description() string

    // Parameters returns the JSON Schema for tool parameters.
    Parameters() any

    // Execute runs the tool with the given JSON arguments.
    Execute(ctx context.Context, arguments string) (string, error)
}
```

### 多模态工具扩展 (internal/agent/)

工具可通过可选的 `RichExecutor` 接口返回附件（图片、文件等），不影响现有 Tool 接口。

```go
// Attachment represents a file attachment in a tool result.
type Attachment struct {
    FilePath string `json:"file_path"`
    MimeType string `json:"mime_type"`
    FileName string `json:"file_name"`
}

// ToolResult holds text and optional file attachments from tool execution.
type ToolResult struct {
    Text        string       `json:"text"`
    Attachments []Attachment `json:"attachments,omitempty"`
}

// RichExecutor is an optional Tool extension that returns multimodal results.
// Agent runtime checks for this interface; falls back to Tool.Execute if absent.
type RichExecutor interface {
    ExecuteRich(ctx context.Context, arguments string) (*ToolResult, error)
}

// FileSenderFunc delivers an attachment to the user via the current channel.
// Set by Manager in context before calling Agent.Run.
type FileSenderFunc func(ctx context.Context, att Attachment) error

func WithFileSender(ctx context.Context, fn FileSenderFunc) context.Context
func GetFileSender(ctx context.Context) FileSenderFunc
```

### 规划-执行接口（v0.3.0 新增，internal/agent/）

```go
// TaskPlanner 生成结构化执行计划
type TaskPlanner interface {
    // Plan 根据用户请求生成执行计划
    Plan(ctx context.Context, task string) (*ExecutionPlan, error)
}

// Executor 按照计划执行具体步骤
type Executor interface {
    // Execute 执行计划中的步骤
    Execute(ctx context.Context, plan *ExecutionPlan) (*ExecutionResult, error)
}

// ContextManager 管理执行上下文和状态
type ContextManager interface {
    // GetContext 获取执行上下文
    GetContext(chatID string) (*ExecutionContext, error)
    // UpdateContext 更新执行上下文
    UpdateContext(chatID string, ctx *ExecutionContext) error
}

// CapabilityExtractor 从系统提取可用能力
type CapabilityExtractor interface {
    // ExtractCapabilities 提取可用工具和技能
    ExtractCapabilities() ([]Capability, error)
}

// SkillHook 集成技能系统到执行流程
type SkillHook interface {
    // BeforeStep 在步骤执行前调用
    BeforeStep(ctx context.Context, step *ExecutionStep) error
    // AfterStep 在步骤执行后调用
    AfterStep(ctx context.Context, step *ExecutionStep, result *StepResult) error
}

// ExecutionPlan 执行计划
type ExecutionPlan struct {
    ID          string           `json:"id"`
    Task        string           `json:"task"`
    Steps       []*ExecutionStep `json:"steps"`
    Context     *ExecutionContext `json:"context,omitempty"`
    CreatedAt   time.Time        `json:"created_at"`
}

// ExecutionStep 执行步骤
type ExecutionStep struct {
    ID          int             `json:"id"`
    Type        string          `json:"type"` // tool, skill, thought
    Description string          `json:"description"`
    Tool        string          `json:"tool,omitempty"`
    Arguments   string          `json:"arguments,omitempty"`
    Status      string          `json:"status"` // pending, running, completed, failed
    Result      *StepResult     `json:"result,omitempty"`
}

// ExecutionContext 执行上下文
type ExecutionContext struct {
    ChatID       string                 `json:"chat_id"`
    Variables    map[string]interface{} `json:"variables"`
    History      []*ExecutionStep       `json:"history"`
    CurrentStep  int                    `json:"current_step"`
}

// Capability 能力描述
type Capability struct {
    Type        string `json:"type"` // tool, skill
    Name        string `json:"name"`
    Description string `json:"description"`
    Parameters  any    `json:"parameters,omitempty"`
}

// ExecutionResult 执行结果
type ExecutionResult struct {
    Success      bool             `json:"success"`
    StepsTotal   int              `json:"steps_total"`
    StepsDone    int              `json:"steps_done"`
    Output       string           `json:"output"`
    Error        string           `json:"error,omitempty"`
    Attachments  []Attachment     `json:"attachments,omitempty"`
}

// StepResult 步骤结果
type StepResult struct {
    Success     bool       `json:"success"`
    Output      string     `json:"output"`
    Error       string     `json:"error,omitempty"`
    Duration    time.Duration `json:"duration"`
    Attachments []Attachment `json:"attachments,omitempty"`
}
```

### 内置工具列表 (internal/tools/)

与 CoPaw agents/tools 全量对齐（含 browser_use、desktop_screenshot、send_file_to_user）；扩展 web_search、http_request、switch_model。差异见 `docs/architecture_spec.md`「internal/tools 与 CoPaw agents/tools 全量对齐说明」。

| 工具名 | 实现 | 说明 |
|--------|------|------|
| `get_current_time` | TimeTool | 获取当前系统时间 |
| `execute_shell_command` | ShellTool | 执行 Shell 命令 |
| `read_file` | ReadFileTool | 读取文件内容 |
| `write_file` | WriteFileTool | 写入文件 |
| `edit_file` | EditFileTool | 查找替换文件内容（old_text -> new_text） |
| `append_file` | AppendFileTool | 在文件末尾追加内容 |
| `grep_search` | GrepSearchTool | 按模式搜索文件内容 |
| `glob_search` | GlobSearchTool | 按 glob 模式查找文件 |
| `web_search` | WebSearchTool | 网络搜索（DuckDuckGo，无需 API Key） |
| `http_request` | HTTPTool | 通用 HTTP 请求（GET/POST 等） |
| `memory_search` | MemorySearchTool | 语义搜索记忆（混合检索：向量+BM25） |
| `browser_use` | BrowserTool | 浏览器自动化（go-rod/rod，CDP 协议） |
| `desktop_screenshot` | ScreenshotTool | 跨平台桌面截屏（kbinani/screenshot） |
| `send_file_to_user` | SendFileTool | 发送本地文件给用户（RichExecutor） |
| `switch_model` | ModelSwitchTool | 切换活跃模型槽位（需 ModelRouter 支持） |

### Channel 接口 (internal/channels/)

与 CoPaw app/channels（BaseChannel、ChannelManager、各渠道）对齐；六渠道已实现，imessage/voice 跳过。详见 `docs/architecture_spec.md`「internal/channels 与 CoPaw app/channels 各渠道对齐说明」。

```go
// Channel represents a messaging platform adapter.
type Channel interface {
    // Start begins listening for messages.
    Start(ctx context.Context) error

    // Stop gracefully shuts down the channel.
    Stop(ctx context.Context) error

    // Name returns the channel identifier.
    Name() string

    // Send sends a text message to the given target.
    Send(ctx context.Context, to string, text string, meta map[string]string) error

    // IsEnabled returns true if the channel is enabled in config.
    IsEnabled() bool
}

// FileSender is an optional Channel extension for delivering file attachments.
// Channels that support file upload implement this interface.
// Agent runtime falls back to sending file path as text if not implemented.
type FileSender interface {
    SendFile(ctx context.Context, to string, filePath string, mime string, meta map[string]string) error
}

// MessageHandler is called when a channel receives a message.
type MessageHandler func(ctx context.Context, msg IncomingMessage) (string, error)

// IncomingMessage represents a message received from a channel.
type IncomingMessage struct {
    ChatID    string            `json:"chat_id"`
    UserID    string            `json:"user_id"`
    UserName  string            `json:"user_name"`
    Content   string            `json:"content"`
    Channel   string            `json:"channel"`
    Timestamp int64             `json:"timestamp"`
    Metadata  map[string]string  `json:"metadata,omitempty"`
}
```

### 渠道配置结构 (ChannelsConfig)

```go
type ChannelsConfig struct {
    Console  ConsoleConfig  `mapstructure:"console" yaml:"console"`
    Telegram TelegramConfig `mapstructure:"telegram" yaml:"telegram"`
    Discord  DiscordConfig  `mapstructure:"discord" yaml:"discord"`
    DingTalk DingTalkConfig `mapstructure:"dingtalk" yaml:"dingtalk"`
    Feishu   FeishuConfig   `mapstructure:"feishu" yaml:"feishu"`
    QQ       QQConfig       `mapstructure:"qq" yaml:"qq"`
}

type ConsoleConfig struct {
    Enabled bool `mapstructure:"enabled" yaml:"enabled"`
}

type TelegramConfig struct {
    Enabled       bool   `mapstructure:"enabled" yaml:"enabled"`
    BotPrefix     string `mapstructure:"bot_prefix" yaml:"bot_prefix"`
    BotToken      string `mapstructure:"bot_token" yaml:"bot_token"`
    HTTPProxy     string `mapstructure:"http_proxy" yaml:"http_proxy"`
    HTTPProxyAuth string `mapstructure:"http_proxy_auth" yaml:"http_proxy_auth"`
}

type DiscordConfig struct {
    Enabled       bool   `mapstructure:"enabled" yaml:"enabled"`
    BotPrefix     string `mapstructure:"bot_prefix" yaml:"bot_prefix"`
    BotToken      string `mapstructure:"bot_token" yaml:"bot_token"`
    HTTPProxy     string `mapstructure:"http_proxy" yaml:"http_proxy"`
    HTTPProxyAuth string `mapstructure:"http_proxy_auth" yaml:"http_proxy_auth"`
}

type DingTalkConfig struct {
    Enabled      bool   `mapstructure:"enabled" yaml:"enabled"`
    BotPrefix    string `mapstructure:"bot_prefix" yaml:"bot_prefix"`
    ClientID     string `mapstructure:"client_id" yaml:"client_id"`
    ClientSecret string `mapstructure:"client_secret" yaml:"client_secret"`
}

type FeishuConfig struct {
    Enabled            bool   `mapstructure:"enabled" yaml:"enabled"`
    BotPrefix          string `mapstructure:"bot_prefix" yaml:"bot_prefix"`
    AppID              string `mapstructure:"app_id" yaml:"app_id"`
    AppSecret          string `mapstructure:"app_secret" yaml:"app_secret"`
    EncryptKey         string `mapstructure:"encrypt_key" yaml:"encrypt_key"`
    VerificationToken  string `mapstructure:"verification_token" yaml:"verification_token"`
}

type QQConfig struct {
    Enabled      bool   `mapstructure:"enabled" yaml:"enabled"`
    BotPrefix    string `mapstructure:"bot_prefix" yaml:"bot_prefix"`
    AppID        string `mapstructure:"app_id" yaml:"app_id"`
    ClientSecret string `mapstructure:"client_secret" yaml:"client_secret"`
}
```

### Scheduler 接口 (internal/scheduler/)

与 CoPaw app/crons（CronManager、heartbeat）对齐；心跳与 CronScheduler 启停/AddJob 已实现，无 job repo/CRUD、无通用 CronExecutor。详见 `docs/architecture_spec.md`「internal/scheduler 与 CoPaw app/crons 对齐说明」。

```go
// Scheduler manages cron jobs and periodic tasks.
type Scheduler interface {
    // Start begins the scheduler loop.
    Start(ctx context.Context) error

    // Stop gracefully shuts down the scheduler.
    Stop(ctx context.Context) error

    // AddJob registers a new cron job.
    AddJob(spec string, job Job) error
}

// Job represents a schedulable task.
type Job interface {
    // Run executes the job.
    Run(ctx context.Context) error

    // Name returns the job identifier.
    Name() string
}
```

### Config 结构 (internal/config/)

```go
type Config struct {
    Server    ServerConfig    `mapstructure:"server" yaml:"server"`
    Agent     AgentConfig     `mapstructure:"agent" yaml:"agent"`
    LLM       LLMConfig       `mapstructure:"llm" yaml:"llm"`
    Memory    MemoryConfig    `mapstructure:"memory" yaml:"memory"`
    Channels  ChannelsConfig  `mapstructure:"channels" yaml:"channels"`
    Scheduler SchedulerConfig `mapstructure:"scheduler" yaml:"scheduler"`
    Log       LogConfig       `mapstructure:"log" yaml:"log"`
}

type ServerConfig struct {
    Host string `mapstructure:"host" yaml:"host"`
    Port int    `mapstructure:"port" yaml:"port"`
}

type AgentConfig struct {
    SystemPrompt   string             `mapstructure:"system_prompt" yaml:"system_prompt"`
    WorkingDir     string             `mapstructure:"working_dir" yaml:"working_dir"`
    Defaults       AgentDefaultsConfig `mapstructure:"defaults" yaml:"defaults"`
    Running        AgentRunningConfig  `mapstructure:"running" yaml:"running"`
    Language       string             `mapstructure:"language" yaml:"language"`
}

type AgentDefaultsConfig struct {
    Heartbeat      *HeartbeatConfig   `mapstructure:"heartbeat" yaml:"heartbeat"`
}

type AgentRunningConfig struct {
    MaxTurns         int                `mapstructure:"max_turns" yaml:"max_turns"`
    MaxInputLength   int                `mapstructure:"max_input_length" yaml:"max_input_length"`
    NamesakeStrategy string             `mapstructure:"namesake_strategy" yaml:"namesake_strategy"`
}

type LLMConfig struct {
    Provider   string                `mapstructure:"provider" yaml:"provider"`
    Model      string                `mapstructure:"model" yaml:"model"`
    APIKey     string                `mapstructure:"api_key" yaml:"api_key"`
    BaseURL    string                `mapstructure:"base_url" yaml:"base_url"`
    OllamaURL  string                `mapstructure:"ollama_url" yaml:"ollama_url"`
    Models     map[string]ModelSlot  `mapstructure:"models" yaml:"models"`   // multi-model routing slots
}

type MemoryConfig struct {
    Backend           string  `mapstructure:"backend" yaml:"backend"`
    DBPath            string  `mapstructure:"db_path" yaml:"db_path"`
    MaxHistory        int     `mapstructure:"max_history" yaml:"max_history"`
    WorkingDir        string  `mapstructure:"working_dir" yaml:"working_dir"`
    CompactThreshold  int     `mapstructure:"compact_threshold" yaml:"compact_threshold"`
    CompactKeepRecent int     `mapstructure:"compact_keep_recent" yaml:"compact_keep_recent"`
    CompactRatio      float64 `mapstructure:"compact_ratio" yaml:"compact_ratio"`
    EmbeddingAPIKey   string  `mapstructure:"embedding_api_key" yaml:"embedding_api_key"`
    EmbeddingBaseURL  string  `mapstructure:"embedding_base_url" yaml:"embedding_base_url"`
    EmbeddingModel    string  `mapstructure:"embedding_model" yaml:"embedding_model"`
    EmbeddingDimensions int   `mapstructure:"embedding_dimensions" yaml:"embedding_dimensions"`
    EmbeddingMaxCache int     `mapstructure:"embedding_max_cache" yaml:"embedding_max_cache"`
    FTSEnabled        bool    `mapstructure:"fts_enabled" yaml:"fts_enabled"`
}

type ChannelsConfig struct {
    Console  ConsoleConfig  `mapstructure:"console" yaml:"console"`
    Telegram TelegramConfig `mapstructure:"telegram" yaml:"telegram"`
    Discord  DiscordConfig  `mapstructure:"discord" yaml:"discord"`
    DingTalk DingTalkConfig `mapstructure:"dingtalk" yaml:"dingtalk"`
    Feishu   FeishuConfig   `mapstructure:"feishu" yaml:"feishu"`
    QQ       QQConfig       `mapstructure:"qq" yaml:"qq"`
}

type SchedulerConfig struct {
    Enabled   bool             `mapstructure:"enabled" yaml:"enabled"`
    Heartbeat HeartbeatConfig  `mapstructure:"heartbeat" yaml:"heartbeat"`
}

type HeartbeatConfig struct {
    Every       string       `mapstructure:"every" yaml:"every"`
    Target      string       `mapstructure:"target" yaml:"target"`
    ActiveHours *ActiveHours `mapstructure:"active_hours" yaml:"active_hours"`
}

type ActiveHours struct {
    Start string `mapstructure:"start" yaml:"start"`
    End   string `mapstructure:"end" yaml:"end"`
}

type MCPConfig struct {
    Servers map[string]MCPServerConfig `mapstructure:"servers" yaml:"servers"`
}

// MCPServerConfig holds configuration for a single MCP server.
// Supports three transport types: stdio, streamable_http, sse.
type MCPServerConfig struct {
    Name        string            `mapstructure:"name" yaml:"name"`
    Description string            `mapstructure:"description" yaml:"description"`
    Enabled     *bool             `mapstructure:"enabled" yaml:"enabled"`             // nil = true (default)
    Transport   string            `mapstructure:"transport" yaml:"transport"`         // "stdio", "streamable_http", "sse" (default: "stdio")
    URL         string            `mapstructure:"url" yaml:"url"`                     // HTTP/SSE endpoint URL
    Headers     map[string]string `mapstructure:"headers" yaml:"headers"`             // HTTP headers for remote transports
    Command     string            `mapstructure:"command" yaml:"command"`             // Executable command for stdio transport
    Args        []string          `mapstructure:"args" yaml:"args"`                   // Command-line arguments
    Env         map[string]string `mapstructure:"env" yaml:"env"`                     // Environment variables
    Cwd         string            `mapstructure:"cwd" yaml:"cwd"`                     // Working directory for stdio transport
}

type LogConfig struct {
    Level  string `mapstructure:"level" yaml:"level"`
    Format string `mapstructure:"format" yaml:"format"`
}
```

### Config 与 CoPaw config 对齐

- **对应关系**：CoPaw `config/config.py` 的 `Config` 含 channels、mcp、agents、last_api 等；GopherPaw `Config` 覆盖上述语义并扩展 server/llm/memory/log/skills/runtime。渠道名与字段名与 CoPaw 一致（如 telegram/discord/dingtalk/feishu/qq/console）；MCP 使用 `mcp.servers`（CoPaw 为 `mcp.clients`），单条配置与 CoPaw `MCPClientConfig` 对齐（transport/url/headers/command/args/env/cwd）。
- **扩展**：server、llm（含 models 多槽位）、memory、log、skills、runtime 为 GopherPaw 独有，便于单文件部署。
- **差异**：CoPaw 有 imessage/voice 渠道及 filter_tool_messages、show_typing、media_dir；GopherPaw 暂无。配置格式为 YAML+热重载，无 save_config 写回。详见 `docs/architecture_spec.md` 中「internal/config 与 CoPaw 对齐说明」。

### 环境变量覆盖 (internal/config/)

所有环境变量使用 `GOPHERPAW_` 前缀（CoPaw 为 `COPAW_`）：

| 环境变量 | 对应配置字段 | 默认值 | 说明 |
|---------|-------------|-------|------|
| `GOPHERPAW_WORKING_DIR` | `working_dir` | `~/.gopherpaw` | 工作目录 |
| `GOPHERPAW_SECRET_DIR` | - | `{working_dir}.secret` | 密钥目录 |
| `GOPHERPAW_CONFIG_FILE` | - | `config.yaml` | 配置文件名 |
| `GOPHERPAW_JOBS_FILE` | - | `jobs.json` | Cron 任务文件 |
| `GOPHERPAW_CHATS_FILE` | - | `chats.json` | 聊天记录文件 |
| `GOPHERPAW_HEARTBEAT_FILE` | - | `HEARTBEAT.md` | 心跳查询文件 |
| `GOPHERPAW_LOG_LEVEL` | `log.level` | `info` | 日志级别 |
| `GOPHERPAW_LOG_FORMAT` | `log.format` | `json` | 日志格式 |
| `GOPHERPAW_RUNNING_IN_CONTAINER` | - | `false` | 容器环境标识 |
| `GOPHERPAW_LLM_API_KEY` | `llm.api_key` | - | LLM API Key |
| `GOPHERPAW_LLM_BASE_URL` | `llm.base_url` | - | LLM Base URL |
| `GOPHERPAW_LLM_MODEL` | `llm.model` | - | LLM 模型名 |
| `GOPHERPAW_MEMORY_WORKING_DIR` | `memory.working_dir` | - | 记忆工作目录 |
| `GOPHERPAW_MEMORY_COMPACT_KEEP_RECENT` | `memory.compact_keep_recent` | `3` | 压缩保留消息数 |
| `GOPHERPAW_MEMORY_COMPACT_RATIO` | `memory.compact_ratio` | `0.7` | 压缩目标比例 |
| `GOPHERPAW_EMBEDDING_API_KEY` | `memory.embedding_api_key` | - | Embedding API Key |
| `GOPHERPAW_EMBEDDING_BASE_URL` | `memory.embedding_base_url` | - | Embedding Base URL |
| `GOPHERPAW_EMBEDDING_MODEL` | `memory.embedding_model` | - | Embedding 模型名 |
| `GOPHERPAW_ENABLED_CHANNELS` | - | - | 启用的渠道列表（逗号分隔） |
| `GOPHERPAW_CORS_ORIGINS` | - | - | CORS 允许的源（逗号分隔） |
| `GOPHERPAW_MODEL_PROVIDER_CHECK_TIMEOUT` | - | `5.0` | Provider 检查超时（秒） |

```go
// GetEnvString returns the value of an environment variable or the default.
func GetEnvString(key, defaultValue string) string

// GetEnvBool returns a boolean environment variable.
func GetEnvBool(key string, defaultValue bool) bool

// GetEnvInt returns an integer environment variable.
func GetEnvInt(key string, defaultValue int) int

// GetEnvFloat returns a float environment variable.
func GetEnvFloat(key string, defaultValue float64) float64

// GetEnvSlice returns a comma-separated environment variable as a slice.
func GetEnvSlice(key string, defaultValue []string) []string

// IsRunningInContainer returns true if running inside a container.
func IsRunningInContainer() bool

// GetEnabledChannels returns the list of enabled channels from env or config.
func GetEnabledChannels() []string

// GetCORSOrigins returns the list of CORS origins from env.
func GetCORSOrigins() []string
```


### PromptLoader (internal/agent/)

```go
// PromptFileEntry defines a file to load and whether it is required.
type PromptFileEntry struct {
    Filename string
    Required bool
}

// PromptConfig holds the prompt loading configuration.
type PromptConfig struct {
    FileOrder []PromptFileEntry
    Language  string
}

func DefaultPromptConfig() PromptConfig

// PromptLoader loads the six-file prompt system from working directory.
type PromptLoader struct { ... }

func NewPromptLoader(workingDir, fallback string) *PromptLoader
func NewPromptLoaderWithConfig(workingDir, fallback string, cfg PromptConfig) *PromptLoader
func (p *PromptLoader) LoadSystemPrompt() (string, error)
func (p *PromptLoader) LoadSOUL() (string, error)
func (p *PromptLoader) LoadAGENTS() (string, error)
func (p *PromptLoader) LoadPROFILE() (string, error)
func (p *PromptLoader) LoadMEMORY() (string, error)
func (p *PromptLoader) LoadHEARTBEAT() (string, error)
func (p *PromptLoader) HasBootstrap() bool
func (p *PromptLoader) DeleteBootstrap() error
func (p *PromptLoader) BuildSystemPrompt(skillsContent string) string
func (p *PromptLoader) CopyMDFiles(srcDir string) error
func (p *PromptLoader) Language() string
func (p *PromptLoader) Config() PromptConfig
func BuildBootstrapGuidance(language string) string
```

### Hooks (internal/agent/)

与 CoPaw agents/hooks（BootstrapHook、MemoryCompactionHook）对齐；Runner/会话/Utils 对齐见 `docs/architecture_spec.md`「internal/agent 与 CoPaw agents/react_agent、runner、session、hooks、utils 对齐说明」。

```go
// Hook is invoked before each ReAct reasoning step.
type Hook func(ctx context.Context, agent *ReactAgent, chatID string, messages []Message) ([]Message, error)

func MemoryCompactionHook(threshold, keepRecent int) Hook
func BootstrapHook(workingDir, language string) Hook
func EstimateMessageTokens(m Message) int
```

### NamesakeStrategy (internal/agent/)

与 CoPaw `NamesakeStrategy = Literal["override", "skip", "raise", "rename"]` 对齐。

```go
// NamesakeStrategy defines how to handle duplicate tool names.
// Options: "override", "skip", "raise", "rename"
// Default: "skip"
type NamesakeStrategy string

const (
    NamesakeOverride NamesakeStrategy = "override" // Replace existing tool with same name
    NamesakeSkip     NamesakeStrategy = "skip"     // Keep existing tool, ignore new one
    NamesakeRaise    NamesakeStrategy = "raise"    // Return error on duplicate
    NamesakeRename   NamesakeStrategy = "rename"   // Auto-rename new tool (tool_2, tool_3, ...)
)

// RegisterTool registers a tool with namesake strategy handling.
func RegisterTool(tools []Tool, toolMap map[string]Tool, tool Tool, strategy NamesakeStrategy) error
```

- **override**: 新工具覆盖同名旧工具（默认 Go 行为）
- **skip**: 保留旧工具，跳过新工具（CoPaw 默认）
- **raise**: 遇到同名工具时返回错误
- **rename**: 自动重命名新工具（添加后缀 _2, _3, ...）


### Message Utilities (internal/agent/)

```go
func CountMessageTokens(messages []Message) int
func CountStringTokens(text string) int
func SafeCountMessageTokens(messages []Message) int
func SanitizeToolMessages(messages []Message) []Message
func CheckValidMessages(messages []Message) bool
func TruncateText(text string, maxLength int) string
func RepairEmptyToolInputs(messages []Message) []Message
func BuildEnvContext(sessionID, userID, channel, workingDir string) string
```

### EnvContext (internal/agent/)

与 CoPaw `build_env_context(session_id, user_id, channel, working_dir)` 对齐。

```go
// BuildEnvContext builds environment context string with session/user/channel info.
// Returns formatted string like:
// ## 环境上下文
//
// - 当前的session_id: xxx
// - 当前的user_id: xxx
// - 当前的channel: xxx
// - 工作目录: /path/to/work
func BuildEnvContext(sessionID, userID, channel, workingDir string) string
```

- **用途**：构建环境上下文字符串，可添加到 system prompt 前
- **对齐**：与 CoPaw `build_env_context` 格式一致
- **集成**：可在 PromptLoader 或 Agent.buildMessages 中调用


### ModelSlot 配置 (internal/config/)

```go
// ModelSlot defines a named model with optional provider/credential overrides and capability tags.
type ModelSlot struct {
    Model        string   `mapstructure:"model" yaml:"model"`
    Provider     string   `mapstructure:"provider" yaml:"provider"`         // optional override
    BaseURL      string   `mapstructure:"base_url" yaml:"base_url"`         // optional override
    APIKey       string   `mapstructure:"api_key" yaml:"api_key"`           // optional override
    Capabilities []string `mapstructure:"capabilities" yaml:"capabilities"` // e.g. text, vision, code, tools
}
```

`LLMConfig.Models` (`map[string]ModelSlot`) 允许在 `llm` 配置下声明多个命名模型槽位。
未配置时行为与单模型完全一致（向后兼容）。

### ModelRouter (internal/llm/)

```go
// ModelRouter implements agent.LLMProvider with multi-model routing.
// It wraps multiple providers (one per slot) and selects the appropriate
// one based on message content analysis or explicit switching.
type ModelRouter struct { ... }

func NewModelRouter(baseCfg config.LLMConfig) (*ModelRouter, error)
func (r *ModelRouter) Chat(ctx context.Context, req *agent.ChatRequest) (*agent.ChatResponse, error)
func (r *ModelRouter) ChatStream(ctx context.Context, req *agent.ChatRequest) (agent.ChatStream, error)
func (r *ModelRouter) Name() string

// Switch changes the active model slot by name. Returns error if slot not found.
func (r *ModelRouter) Switch(slotName string) error

// ActiveSlot returns the name of the currently active model slot.
func (r *ModelRouter) ActiveSlot() string

// SlotNames returns all configured slot names.
func (r *ModelRouter) SlotNames() []string

// HasCapability checks if any configured slot has the given capability.
func (r *ModelRouter) HasCapability(cap string) bool
```

**能力检测规则**：`Chat` 方法扫描最近消息，若检测到图片引用（`.png`, `.jpg`, `.jpeg`, `.gif`, `.webp`, `.bmp`, `base64,`）且当前 slot 无 `vision` 能力，临时切换到具有 `vision` 能力的 slot 执行请求，完成后恢复。

### LLM Registry (internal/llm/)

```go
// Factory creates an LLMProvider from config.
type Factory func(cfg config.LLMConfig) (agent.LLMProvider, error)

func Register(name string, f Factory)                    // 注册 provider，如 "openai"、"ollama"
func Create(cfg config.LLMConfig) (agent.LLMProvider, error)  // 按 cfg.Provider 创建实例
func SwitchProvider(providerName, model string, baseCfg config.LLMConfig) (agent.LLMProvider, error)  // 运行时切换 provider+model
```

内置注册：`openai` → NewOpenAI；`ollama` → NewOllama。

### LLM Provider 实现 (internal/llm/)

| 名称 | 实现 | 说明 |
|------|------|------|
| OpenAI 兼容 | `OpenAIProvider`、`NewOpenAI(cfg)` | go-openai，支持 BaseURL/APIKey，Chat/ChatStream |
| Ollama | `OllamaProvider`、`NewOllama(cfg)` | HTTP /api/chat，cfg.OllamaURL、cfg.Model |

### LLM Downloader (internal/llm/)

用于将模型文件下载到 `~/.gopherpaw/models/`，与 CoPaw local_models 下载入口语义对齐（实现为简化版，仅 URL 直下）。

```go
const (
    SourceHuggingFace = "huggingface"
    SourceModelScope  = "modelscope"
    SourceURL         = "url"
)

// DownloadModel 下载到 ~/.gopherpaw/models/。source=url 时 repoID 为直接文件 URL，backend 为可选文件名。
func DownloadModel(ctx context.Context, repoID, source, backend string) (string, error)

// DownloadFromURL 从给定 URL 下载到 ~/.gopherpaw/models/，返回本地路径。
func DownloadFromURL(ctx context.Context, url string) (string, error)
```

当前仅 `source=url` 支持完整下载；HuggingFace/ModelScope 需传直接文件 URL 使用。

### internal/llm 与 CoPaw providers/local_models 对齐

- **已对齐**：OpenAI 兼容 Chat/ChatStream（openai_chat_model_compat）、Ollama 聊天（ollama_manager 的调用面）、Provider 注册与按配置创建（registry）、多槽位与切换（store active_llm → ModelRouter + Models）、消息格式化（Formatter）、模型下载入口（local_models 下载 → DownloadModel/DownloadFromURL）。
- **未实现或简化**：providers.json 持久化与 set_active_llm/update_provider_settings、自定义 Provider CRUD、discover_provider_models/test_provider_connection/test_model_connection、Ollama list_models/pull_model API、local_models 的 manifest、llamacpp/mlx 后端与 create_local_chat_model。详见 `docs/architecture_spec.md`「internal/llm 与 CoPaw providers/*、local_models/* 对齐说明」。

### LLM Formatter (internal/llm/)

```go
// Formatter processes messages before sending to the LLM API.
type Formatter interface {
    Format(messages []agent.Message) []agent.Message
}

// FileBlockSupportFormatter converts file blocks in tool results to text.
type FileBlockSupportFormatter struct {
    StripMessageName bool
}

func NewFileBlockFormatter() *FileBlockSupportFormatter
func (f *FileBlockSupportFormatter) Format(messages []agent.Message) []agent.Message
```


### Skills (internal/skills/)

```go
type Skill struct {
    Name        string
    Description string
    Content     string
    Enabled     bool
    Path        string
    Scripts     map[string]string    // Script files from scripts/ directory
    References  map[string]string    // Reference docs from references/ directory
    ExtraFiles  map[string][]byte    // Extra files (binary or text)
}

type Manager struct { ... }

func NewManager() *Manager
func (m *Manager) LoadSkills(workingDir, configDir string, cfg config.SkillsConfig) error
func (m *Manager) GetEnabledSkills() []Skill
func (m *Manager) EnableSkill(name string) error
func (m *Manager) DisableSkill(name string) error
func (m *Manager) ImportFromURL(ctx context.Context, url, workingDir string, cfg config.SkillsConfig) (string, error)
```

- **三目录系统**：active_dir（激活技能）、customized_dir（自定义技能）
- **目录结构**：
  - `SKILL.md`：技能主文件（YAML front matter + Markdown 内容）
  - `scripts/`：脚本文件（文本格式，如 .sh、.py）
  - `references/`：参考文档（文本格式，如 .md、.txt）
  - `extra_files/`：额外文件（二进制或文本）
- **对齐**：CoPaw agents/skills 三目录系统
### MCP Manager 接口 (internal/mcp/)

与 CoPaw app/mcp（MCPClientManager）、app/routers/mcp 对齐；Manager/Client/Transport、三种传输已实现，无 MCP ConfigWatcher、无 MCP HTTP API。详见 `docs/architecture_spec.md`「internal/mcp 与 CoPaw app/mcp、app/routers/mcp 对齐说明」。

```go
// Transport abstracts MCP communication layer.
type Transport interface {
    // Start initializes the transport connection.
    Start(ctx context.Context) error
    
    // Stop closes the transport connection.
    Stop() error
    
    // Call sends a JSON-RPC request and returns the response.
    Call(ctx context.Context, req jsonRPCRequest, result interface{}) error
    
    // WriteNotification sends a notification (no response expected).
    WriteNotification(msg map[string]any) error
    
    // IsRunning returns true if transport is active.
    IsRunning() bool
}

// MCPClient represents a connection to a single MCP server.
type MCPClient struct {
    Name        string
    Description string
    Transport   Transport
    Enabled     bool
}

func NewMCPClient(name string, cfg MCPServerConfig) (*MCPClient, error)

// MCPManager manages multiple MCP clients and provides tools.
type MCPManager struct {
    clients map[string]*MCPClient
}

func NewManager() *MCPManager
func (m *MCPManager) LoadConfig(cfg map[string]MCPServerConfig) error
func (m *MCPManager) AddClient(ctx context.Context, name string, cfg MCPServerConfig) error
func (m *MCPManager) RemoveClient(ctx context.Context, name string) error
func (m *MCPManager) Reload(ctx context.Context, newConfigs map[string]MCPServerConfig) error
func (m *MCPManager) GetTools() []agent.Tool
func (m *MCPManager) Start(ctx context.Context) error
func (m *MCPManager) Stop() error
```

### Transport 实现 (internal/mcp/)

| Transport | 实现文件 | 说明 |
|-----------|---------|------|
| stdio | `transport_stdio.go` | 本地子进程 stdin/stdout 通信 |
| streamable_http | `transport_http.go` | HTTP POST JSON-RPC 请求 |
| sse | `transport_sse.go` | HTTP POST + SSE 响应流 |

**Transport 选择逻辑：**
- `transport` 为空或 `"stdio"`：使用 StdioTransport
- `transport` 为 `"streamable_http"`：使用 HTTPTransport
- `transport` 为 `"sse"`：使用 SSETransport

### HeartbeatRunner 接口 (internal/scheduler/)

```go
type HeartbeatRunner struct {
    agent      agent.Agent
    loader     agent.PromptLoader
    config     HeartbeatConfig
    channelMgr HeartbeatChannelMgr
}

type HeartbeatChannelMgr interface {
    Send(channel, to, text string) error
}

func (h *HeartbeatRunner) Start(ctx context.Context) error
func (h *HeartbeatRunner) Stop() error
```

### ProgressReporter 接口（可选，用于 Console 等渠道的实时反馈）

```go
// ProgressReporter receives progress events during Agent.Run.
// Pass via context using agent.WithProgressReporter(ctx, reporter).
type ProgressReporter interface {
    OnThinking()
    OnToolCall(toolName string, args string)
    OnToolResult(toolName string, result string)
    OnFinalReply(content string)
}

// WithProgressReporter attaches a ProgressReporter to the context.
func WithProgressReporter(ctx context.Context, r ProgressReporter) context.Context
```

### 魔法命令处理器 (internal/agent/)

以 `/` 开头的用户消息在 Agent.Run 入口被拦截，路由到对应处理函数：

| 命令 | 处理逻辑 |
|------|----------|
| /compact | 调用 memory.Compact |
| /new | 保存到长期记忆 + 清空短期 |
| /clear | 直接清空短期上下文 |
| /history | 返回消息列表和 token 估算 |
| /compact_str | 返回压缩摘要 |
| /await_summary | 获取当前压缩摘要 |
| /switch-model \<provider\> \<model\> | 运行时切换 LLM |
| /daemon status | 运行状态 |
| /daemon version | 版本信息 |
| /daemon reload-config | 热重载配置 |
| /daemon restart | 进程内重启 |
| /daemon logs [N] | 最近 N 行日志 |

```go
// HandleMagicCommand checks if message is a magic command and returns (result, true) if handled.
// Returns ("", false) if not a magic command.
func HandleMagicCommand(ctx context.Context, memory MemoryStore, chatID string, message string, daemonInfo *DaemonInfo) (string, bool, error)

type DaemonInfo struct {
    Version      string
    Status       string
    Logs         func(n int) string
    ReloadConfig func() error
    Restart      func() error
    SwitchLLM    func(provider, model string) error
}
```

### SkillManager 接口 (internal/skills/)

```go
// Skill represents a loaded SKILL.md with YAML front matter.
type Skill struct {
    Name        string
    Description string
    Content     string
    Enabled     bool
    Path        string
}

// Manager loads and manages skills from directories.
type Manager struct { ... }

func NewManager() *Manager
func (m *Manager) LoadSkills(workingDir, configDir string, cfg SkillsConfig) error
func (m *Manager) GetEnabledSkills() []Skill
func (m *Manager) ListAllSkills() []Skill
func (m *Manager) GetSkill(name string) *Skill
func (m *Manager) EnableSkill(name string) error
func (m *Manager) DisableSkill(name string) error
func (m *Manager) CreateSkill(workingDir string, cfg SkillsConfig, name, description, content string) error
func (m *Manager) DeleteSkill(name string, removeFromDisk bool) error
func (m *Manager) SyncSkillsToWorkingDir(workingDir, configDir string, cfg SkillsConfig) error
func (m *Manager) ListAvailableSkills(workingDir, configDir string, cfg SkillsConfig) []string
func (m *Manager) GetSystemPromptAddition() string
func (m *Manager) ImportFromURL(ctx context.Context, url, workingDir string, cfg SkillsConfig) (string, error)
func (m *Manager) InstallSkillFromHub(ctx context.Context, bundleURL, workingDir string, cfg SkillInstallConfig) (*HubInstallResult, error)
```

### Skills Hub (internal/skills/)

```go
// HubSkillResult represents a skill found via hub search.
type HubSkillResult struct {
    Slug, Name, Description, Version, SourceURL string
}

// HubInstallResult represents the result of installing a skill from hub.
type HubInstallResult struct {
    Name      string
    Enabled   bool
    SourceURL string
}

func SearchHubSkills(ctx context.Context, query string, limit int) ([]HubSkillResult, error)
```

### Config 扩展 (internal/config/)

```go
type Config struct {
    // ... existing ...
    WorkingDir string       `mapstructure:"working_dir" yaml:"working_dir"`
    Skills     SkillsConfig `mapstructure:"skills" yaml:"skills"`
}

type SkillsConfig struct {
    ActiveDir     string `mapstructure:"active_dir" yaml:"active_dir"`
    CustomizedDir string `mapstructure:"customized_dir" yaml:"customized_dir"`
}
```

- 工作目录：默认 `~/.gopherpaw/`，`GOPHERPAW_WORKING_DIR` 环境变量覆盖
- 热重载：`viper.WatchConfig()` + fsnotify 回调

### 密钥目录支持 (internal/config/)

```go
// GetSecretDir returns the secret directory path for storing sensitive data.
// Priority: GOPHERPAW_SECRET_DIR env > {working_dir}.secret
func GetSecretDir() string

// GetEnvsJSONPath returns the path to envs.json under the secret directory.
func GetEnvsJSONPath() string

// GetProvidersJSONPath returns the path to providers.json under the secret directory.
func GetProvidersJSONPath() string

// EnsureSecretDir creates the secret directory with proper permissions (0700).
func EnsureSecretDir() error
```

- 密钥目录：默认 `~/.gopherpaw.secret/`，`GOPHERPAW_SECRET_DIR` 环境变量覆盖
- 用途：存储敏感配置（envs.json、providers.json）
- 权限：目录 `0700`，文件 `0600`（仅所有者可访问）
- 对齐：CoPaw `SECRET_DIR` 与 `envs.json`/`providers.json` 持久化

### Runtime 配置 (internal/config/)

```go
type RuntimeConfig struct {
    Python PythonConfig `mapstructure:"python" yaml:"python"`
    Node   NodeConfig   `mapstructure:"node" yaml:"node"`
}

type PythonConfig struct {
    VenvPath    string `mapstructure:"venv_path" yaml:"venv_path"`       // 虚拟环境路径
    Interpreter string `mapstructure:"interpreter" yaml:"interpreter"`   // 显式指定 Python 路径
    AutoInstall bool   `mapstructure:"auto_install" yaml:"auto_install"` // 自动安装依赖
}

type NodeConfig struct {
    Runtime     string `mapstructure:"runtime" yaml:"runtime"`       // "bun" 或 "node"
    BunPath     string `mapstructure:"bun_path" yaml:"bun_path"`     // Bun 可执行文件路径
    NodePath    string `mapstructure:"node_path" yaml:"node_path"`   // Node 可执行文件路径
    AutoInstall bool   `mapstructure:"auto_install" yaml:"auto_install"` // 自动安装依赖
}
```

### Runtime Manager (internal/runtime/)

```go
// Status 表示运行时环境状态
type Status struct {
    Name     string   `json:"name"`
    Ready    bool     `json:"ready"`
    Path     string   `json:"path"`
    Version  string   `json:"version"`
    Error    string   `json:"error,omitempty"`
    Warnings []string `json:"warnings,omitempty"`
}

// Manager 管理运行时环境（Python, Bun/Node）
type Manager struct { ... }

func NewManager(cfg *config.RuntimeConfig) *Manager
func (m *Manager) Initialize() error
func (m *Manager) GetPython() *PythonRuntime
func (m *Manager) GetBun() *BunRuntime
func (m *Manager) CheckEnvironment() []Status
func (m *Manager) PrintEnvironmentReport()

// PythonRuntime 管理 Python 解释器和虚拟环境
type PythonRuntime struct { ... }

func NewPythonRuntime(cfg *config.PythonConfig) *PythonRuntime
func (p *PythonRuntime) Detect() error
func (p *PythonRuntime) IsReady() bool
func (p *PythonRuntime) GetInterpreter() string
func (p *PythonRuntime) GetVersion() string
func (p *PythonRuntime) GetError() string
func (p *PythonRuntime) RunScript(scriptPath string, args ...string) (string, error)
func (p *PythonRuntime) RunModule(module string, args ...string) (string, error)
func (p *PythonRuntime) InstallPackage(pkg string) error
func (p *PythonRuntime) InstallPackages(packages []string) error
func (p *PythonRuntime) CheckPackage(pkg string) (bool, error)

// BunRuntime 管理 Bun JavaScript 运行时
type BunRuntime struct { ... }

func NewBunRuntime(cfg *config.NodeConfig) *BunRuntime
func (b *BunRuntime) Detect() error
func (b *BunRuntime) IsReady() bool
func (b *BunRuntime) GetPath() string
func (b *BunRuntime) GetVersion() string
func (b *BunRuntime) GetError() string
func (b *BunRuntime) RunScript(scriptPath string, args ...string) (string, error)
func (b *BunRuntime) InstallPackage(pkg string) error
func (b *BunRuntime) RunCommand(args ...string) (string, error)

// Bun下载和版本检查
// 优先使用官方安装脚本：curl -fsSL https://bun.sh/install | bash
// 检测系统架构和版本兼容性，自动选择合适的安装方式

// 环境检测工具函数
func CheckSkillBinaries() []Status
func IsCommandAvailable(name string) bool
func FindBinary(name string) string
func GetInstallHint(name string) string
```

## 更新日志

- 2026-03-06: 计划题词 1–11 执行验证：各小节与 architecture_spec 对齐说明一致，接口与实现相符
- 2026-03-06: cmd/gopherpaw 与 CoPaw cli/* 子命令对齐（CLI 入口与子命令小节引用 architecture_spec）
- 2026-03-06: internal/mcp 与 CoPaw app/mcp、app/routers/mcp 对齐（MCP Manager 小节引用 architecture_spec）
- 2026-03-06: internal/scheduler 与 CoPaw app/crons 对齐（Scheduler 小节引用 architecture_spec）
- 2026-03-06: internal/channels 与 CoPaw app/channels 各渠道对齐（Channel 小节引用 architecture_spec）
- 2026-03-06: internal/agent 与 CoPaw react_agent、runner、session、hooks、utils 对齐说明（Hooks 小节引用 architecture_spec）
- 2026-03-06: internal/llm 与 CoPaw providers/local_models 对齐：LLM Registry、Provider 实现表、LLM Downloader、对齐与缺失说明
- 2026-03-06: Config 与 CoPaw config 对齐说明（对应关系、扩展、差异），见「Config 与 CoPaw config 对齐」小节
- 2026-03-06: MCP Transport 抽象层：Transport 接口（Start/Stop/Call/WriteNotification/IsRunning）；三种实现（StdioTransport/HTTPTransport/SSETransport）；MCPServerConfig 扩展（Name/Description/Transport/URL/Headers/Cwd）；MCPClient 重构使用 Transport 组合；对齐 CoPaw mcp.py 三种传输支持
- 2026-03-06: Runtime 环境管理：RuntimeConfig/PythonConfig/NodeConfig 配置；Manager/PythonRuntime/BunRuntime 运行时管理器；环境检测工具（CheckSkillBinaries/IsCommandAvailable/FindBinary）；`gopherpaw env check/setup` 子命令
- 2026-03-05: CoPaw Agents 全量复刻：SkillsHub 搜索安装；Hook 系统（MemoryCompactionHook/BootstrapHook）；消息工具函数（SanitizeToolMessages/CheckValidMessages/TokenCounting）；PromptConfig/CopyMDFiles/BuildBootstrapGuidance；/await_summary 命令；FileBlockSupportFormatter；Skills CRUD（Create/Delete/Sync/ListAvailable）
- 2026-03-05: 多模型路由：ModelSlot/ModelRouter 接口；LLMConfig.Models 字段；switch_model 工具
- 2026-03-05: 多模态工具扩展：Attachment/ToolResult/RichExecutor/FileSenderFunc/FileSender 接口；新增 browser_use/desktop_screenshot/send_file_to_user 工具
- 2026-03-05: 12 项功能完成：/daemon restart、/daemon reload-config；Skills ImportFromURL；MCP AddClient/RemoveClient/Reload、ParseMCPConfig 多格式；LLM downloader、SwitchProvider、providers.json；钉钉/飞书/QQ WebhookServer；内置 Skills 文件；/switch-model 命令
- 2026-03-05: 六文件提示词体系（PromptLoader、BootstrapRunner）；AgentConfig 新增 MaxInputLength、WorkingDir；LLMConfig 新增 OllamaURL；SchedulerConfig 新增 Heartbeat；MCPConfig、MCPServerConfig；Ollama 提供商；心跳系统；MCP 客户端
- 2026-03-05: CLI 框架（cobra）、魔法命令、Skills 系统、配置增强（热重载、工作目录、skills 配置）
- 2026-03-05: 记忆系统全面增强：扩展 MemoryStore 接口（GetCompactSummary/SaveLongTerm/LoadLongTerm）；新增 CompactConfig/EmbeddingConfig/Chunk 类型；MemoryConfig 新增 WorkingDir/Compact*/Embedding*/FTSEnabled；memory_search 工具；WithMemoryStore 上下文
- 2026-03-05: 初始化接口定义
- 2026-03-05: 补充 Channel 子配置结构（TelegramConfig 等）
- 2026-03-05: 新增 ConsoleConfig、ChannelsConfig.Console
- 2026-03-05: 新增 web_search、http_request 工具定义
- 2026-03-05: 新增 edit_file、append_file 工具；新增 ProgressReporter 接口

## 补充接口定义 (2026-03-07)

### ModelSwitcher 接口

```go
// ModelSwitcher is an optional LLM extension for runtime model switching.
type ModelSwitcher interface {
    // Switch changes the active model slot.
    Switch(slotName string) error
    // ActiveSlot returns the current active slot name.
    ActiveSlot() string
    // SlotNames returns all available slot names.
    SlotNames() []string
    // HasCapability checks if a capability is supported.
    HasCapability(cap string) bool
}
```

### Context 工具函数

```go
// Context key types for type-safe context values
type progressReporterKey struct{}
type daemonInfoKey struct{}
type memoryStoreKey struct{}
type chatIDKey struct{}
type fileSenderKey struct{}
type modelSwitcherKey struct{}

// Context helper functions
func WithProgressReporter(ctx context.Context, r ProgressReporter) context.Context
func getProgressReporter(ctx context.Context) ProgressReporter
func WithDaemonInfo(ctx context.Context, info *DaemonInfo) context.Context
func getDaemonInfo(ctx context.Context) *DaemonInfo
func WithMemoryStore(ctx context.Context, store MemoryStore) context.Context
func GetMemoryStore(ctx context.Context) MemoryStore
func WithChatID(ctx context.Context, chatID string) context.Context
func GetChatID(ctx context.Context) string
func WithFileSender(ctx context.Context, fn FileSenderFunc) context.Context
func GetFileSender(ctx context.Context) FileSenderFunc
func WithModelSwitcher(ctx context.Context, ms ModelSwitcher) context.Context
func GetModelSwitcher(ctx context.Context) ModelSwitcher
```

### Session 管理

```go
// Session holds per-chat conversation state.
type Session struct {
    ChatID string
    mu     sync.RWMutex
}

// SessionManager manages sessions by chatID.
type SessionManager struct {
    mu       sync.RWMutex
    sessions map[string]*Session
}

func NewSessionManager() *SessionManager
func (m *SessionManager) GetOrCreate(chatID string) *Session
func (m *SessionManager) Remove(chatID string)
```

### App Manager 接口 (internal/app/)

```go
// State represents the application state.
type State string

const (
    StateRunning    State = "running"
    StateStopped    State = "stopped"
    StateRestarting State = "restarting"
)

// App represents the application lifecycle.
type App struct {
    Name        string
    Version     string
    State       State
    StartedAt   time.Time
    Config      *config.Config
    Scheduler   *scheduler.CronScheduler
    Agent       agent.Agent
    ChannelMgr  *channels.Manager
    mu          sync.RWMutex
}

// Manager manages application lifecycle.
type Manager interface {
    // Start starts the application and all services.
    Start(ctx context.Context) error
    
    // Stop stops the application gracefully.
    Stop(ctx context.Context) error
    
    // RestartServices restarts specific services (channels, scheduler).
    RestartServices(ctx context.Context, services []string) error
    
    // HealthCheck returns the current health status.
    HealthCheck(ctx context.Context) (map[string]interface{}, error)
    
    // State returns the current application state.
    State() State
}

// NewManager creates a new app manager.
func NewManager(cfg *config.Config) (Manager, error)
```

**与 CoPaw 对齐**：
- CoPaw: `app/_app.py` - App 类（lifespan 上下文管理器）
- GopherPaw: `internal/app/` - Manager 接口 + App 实现
- 功能对齐：启动/停止服务、健康检查、状态管理
- 差异：GopherPaw 与 Scheduler 集成，无 FastAPI/uvicorn 依赖

**使用示例**：
```go
// In cmd/gopherpaw/app.go
mgr, err := app.NewManager(cfg)
if err != nil {
    log.Fatal(err)
}

if err := mgr.Start(ctx); err != nil {
    log.Fatal(err)
}

// Handle shutdown
sigCh := make(chan os.Signal, 1)
signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
<-sigCh

mgr.Stop(ctx)
```

### Desktop Manager 接口 (internal/desktop/) - 规划中

> **状态**: 🟡 规划中（Phase 2）
>
> 该模块提供 GopherPaw 远程桌面环境的管理能力，包括 VNC 服务器、noVNC Web 代理、会话录制和控制切换功能。

#### 核心接口

```go
// Manager manages remote desktop environment.
type Manager interface {
    // Start starts the desktop environment (VNC + noVNC).
    Start(ctx context.Context) error
    
    // Stop stops the desktop environment gracefully.
    Stop(ctx context.Context) error
    
    // IsRunning returns whether the desktop is active.
    IsRunning() bool
    
    // GetAccessURL returns the web access URL (http://host:port/vnc.html).
    GetAccessURL() string
    
    // HealthCheck returns health status of VNC and noVNC components.
    HealthCheck() map[string]bool
}

// VNCServer manages TigerVNC server process.
type VNCServer interface {
    // Start starts the VNC server.
    Start(ctx context.Context) error
    
    // Stop stops the VNC server.
    Stop(ctx context.Context) error
    
    // IsRunning returns whether VNC server is active.
    IsRunning() bool
    
    // GetDisplay returns the X11 display (e.g., ":1").
    GetDisplay() string
    
    // HealthCheck verifies X11 socket is accessible.
    HealthCheck() error
}

// NoVNCProxy manages noVNC WebSocket proxy.
type NoVNCProxy interface {
    // Start starts the noVNC proxy.
    Start(ctx context.Context) error
    
    // Stop stops the noVNC proxy.
    Stop(ctx context.Context) error
    
    // IsRunning returns whether proxy is active.
    IsRunning() bool
    
    // GetURL returns the web access URL.
    GetURL() string
    
    // HealthCheck verifies websockify is listening.
    HealthCheck() error
}
```

#### 配置结构

```go
// Config defines desktop environment configuration.
type Config struct {
    Display    string // X11 display number (e.g., ":1")
    Password   string // VNC password (from environment variable)
    Geometry   string // Screen resolution (e.g., "1920x1080")
    Depth      int    // Color depth (default: 24)
    VNCPort    int    // VNC server port (default: 5901)
    NoVNCPort  int    // noVNC proxy port (default: 6080)
}

// RecordingConfig defines session recording settings (optional).
type RecordingConfig struct {
    Enabled   bool   // Enable session recording
    OutputDir string // Directory for recording files
    Format    string // Output format: "webm", "mp4", "json"
}

// ControlConfig defines control mode settings (optional).
type ControlConfig struct {
    Mode              string // Control mode: "agent", "user", "cooperative"
    AllowUserOverride bool   // Allow users to switch mode
}
```

#### 使用示例

```go
// In cmd/gopherpaw/app.go
func startDesktop(cfg *config.Config) error {
    desktopCfg := &desktop.Config{
        Display:   cfg.Desktop.Display,
        Password:  os.Getenv("VNC_PASSWORD"),
        Geometry:  cfg.Desktop.Geometry,
        Depth:     cfg.Desktop.Depth,
        VNCPort:   cfg.Desktop.VNCPort,
        NoVNCPort: cfg.Desktop.NoVNCPort,
    }
    
    mgr, err := desktop.NewManager(desktopCfg)
    if err != nil {
        return fmt.Errorf("create desktop manager: %w", err)
    }
    
    ctx := context.Background()
    if err := mgr.Start(ctx); err != nil {
        return fmt.Errorf("start desktop: %w", err)
    }
    
    zap.L().Info("Desktop started",
        zap.String("url", mgr.GetAccessURL()))
    
    return nil
}

// Health check
func checkDesktopHealth(mgr desktop.Manager) {
    status := mgr.HealthCheck()
    for component, healthy := range status {
        if !healthy {
            zap.L().Warn("Component unhealthy",
                zap.String("component", component))
        }
    }
}
```

#### 错误处理

```go
// Common errors
var (
    ErrVNCAlreadyRunning = errors.New("VNC server already running")
    ErrVNCPortInUse      = errors.New("VNC port already in use")
    ErrInvalidPassword   = errors.New("invalid VNC password")
    ErrDisplayNotFound   = errors.New("X11 display not found")
    ErrNoVNCFailed       = errors.New("noVNC proxy failed to start")
)
```

#### 与 App 模块集成

```go
// In internal/app/app.go
type App struct {
    Config         *config.Config
    CronScheduler  *scheduler.CronScheduler
    DesktopManager desktop.Manager  // 新增
}

func (a *App) startDesktop(ctx context.Context) error {
    if a.Config.Desktop == nil || !a.Config.Desktop.Enabled {
        zap.L().Info("Desktop disabled, skipping")
        return nil
    }
    
    desktopMgr, err := desktop.NewManager(a.Config.Desktop)
    if err != nil {
        return fmt.Errorf("create desktop manager: %w", err)
    }
    
    if err := desktopMgr.Start(ctx); err != nil {
        return fmt.Errorf("start desktop: %w", err)
    }
    
    a.DesktopManager = desktopMgr
    zap.L().Info("Desktop started",
        zap.String("url", desktopMgr.GetAccessURL()))
    
    return nil
}

func (a *App) stopDesktop(ctx context.Context) error {
    if a.DesktopManager == nil {
        return nil
    }
    return a.DesktopManager.Stop(ctx)
}

func (a *App) HealthCheck() map[string]bool {
    status := map[string]bool{
        "cron": a.CronScheduler != nil,
    }
    
    if a.DesktopManager != nil {
        desktopStatus := a.DesktopManager.HealthCheck()
        for k, v := range desktopStatus {
            status["desktop_"+k] = v
        }
    }
    
    return status
}
```

#### 与 CoPaw 对齐

- **CoPaw**: 无对应远程桌面功能
- **GopherPaw 扩展**: 提供可观测的桌面运行环境
- **用途**: 开发调试、演示、远程访问
- **技术栈**: TigerVNC + noVNC + XFCE4

#### 实现计划

**Phase 2.1 - 核心功能**（预计 2 天）：
- [ ] 实现 `vnc_server.go`（VNC 启动/停止/健康检查）
- [ ] 实现 `novnc_proxy.go`（noVNC 代理管理）
- [ ] 实现 `manager.go`（统一管理接口）
- [ ] 编写单元测试（覆盖率 > 80%）

**Phase 2.2 - 集成与测试**（预计 1 天）：
- [ ] 集成到 `internal/app/`
- [ ] 容器内集成测试
- [ ] 性能测试（并发、内存、延迟）

**Phase 2.3 - CLI 集成**（预计 1 天）：
- [ ] 添加 `gopherpaw desktop` 子命令
- [ ] 添加 `gopherpaw desktop status` 命令
- [ ] 添加 `gopherpaw desktop health` 命令

**Phase 2.5 - 增强功能**（可选，预计 2 天）：
- [ ] 实现 `session_recorder.go`（会话录制）
- [ ] 实现 `control_switcher.go`（控制切换）
- [ ] 支持配置热重载

---

### Token 计数工具

```go
// Token counting utilities
func CountMessageTokens(messages []Message) int
func CountStringTokens(text string) int
func CountStringTokensForModel(text string, model string) int
func SafeCountMessageTokens(messages []Message) int
func CheckValidMessages(messages []Message) int
```

### 消息处理工具

```go
// Message processing utilities
func SanitizeToolMessages(messages []Message) []Message
func removeInvalidToolBlocks(messages []Message) []Message
func dedupToolCalls(calls []ToolCall) []ToolCall
func extractMessageText(msg Message) string
```

## 更新日志 (续)

- 2026-03-08: **补充 Desktop Manager 接口定义**（规划中）：VNC/noVNC 管理接口、配置结构、使用示例、错误处理、与 App 模块集成方案、实现计划
- 2026-03-08: 补充 App Manager 接口定义（应用生命周期管理，对齐 CoPaw _app.py）
- 2026-03-07: 补充 ModelSwitcher 接口定义（用于运行时模型切换）
- 2026-03-07: 补充 Context 工具函数（类型安全的上下文值管理）
- 2026-03-07: 补充 Session 管理接口（会话状态管理）
- 2026-03-07: 补充 Token 计数工具函数（Token 估算）
- 2026-03-07: 补充消息处理工具函数（消息清理和验证）
- 2026-03-07: 测试覆盖率提升：Agent 模块达到 74.7%，Config 模块 79.2%，MCP 模块 75.8%
- 2026-03-07: 文档更新：README.md 补充开发指南、高级功能说明
