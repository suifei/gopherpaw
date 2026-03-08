// Package agent provides the core Agent runtime, ReAct loop, and domain types.
package agent

import (
	"context"
)

// ProgressReporter receives progress events during Agent.Run.
// Pass via context using WithProgressReporter(ctx, reporter).
type ProgressReporter interface {
	OnThinking()
	OnToolCall(toolName string, args string)
	OnToolResult(toolName string, result string)
	OnFinalReply(content string)
}

type progressReporterKey struct{}

// WithProgressReporter attaches a ProgressReporter to the context.
func WithProgressReporter(ctx context.Context, r ProgressReporter) context.Context {
	return context.WithValue(ctx, progressReporterKey{}, r)
}

func getProgressReporter(ctx context.Context) ProgressReporter {
	if r, ok := ctx.Value(progressReporterKey{}).(ProgressReporter); ok {
		return r
	}
	return nil
}

// DaemonInfo provides system info for /daemon magic commands.
type DaemonInfo struct {
	Version      string
	Status       string
	Logs         func(n int) string
	ReloadConfig func() error                       // 热重载配置，/daemon reload-config 调用
	Restart      func() error                       // 进程内重启，/daemon restart 调用
	SwitchLLM    func(provider, model string) error // 运行时切换 LLM，/switch-model 调用
}

type daemonInfoKey struct{}

// WithDaemonInfo attaches DaemonInfo to context for /daemon magic commands.
func WithDaemonInfo(ctx context.Context, info *DaemonInfo) context.Context {
	return context.WithValue(ctx, daemonInfoKey{}, info)
}

func getDaemonInfo(ctx context.Context) *DaemonInfo {
	if info, ok := ctx.Value(daemonInfoKey{}).(*DaemonInfo); ok {
		return info
	}
	return nil
}

// Agent processes user messages through a ReAct loop.
type Agent interface {
	// Run processes a message and returns the agent's response.
	Run(ctx context.Context, chatID string, message string) (string, error)

	// RunStream processes a message and streams response chunks.
	RunStream(ctx context.Context, chatID string, message string) (<-chan string, error)
}

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

// ToolChoiceMode specifies how the model should select tools.
type ToolChoiceMode string

const (
	// ToolChoiceAuto lets the model decide whether to call tools (default).
	ToolChoiceAuto ToolChoiceMode = "auto"
	// ToolChoiceNone prevents the model from calling any tools.
	ToolChoiceNone ToolChoiceMode = "none"
	// ToolChoiceRequired forces the model to call at least one tool.
	ToolChoiceRequired ToolChoiceMode = "required"
)

// NamesakeStrategy defines how to handle duplicate tool names.
type NamesakeStrategy string

const (
	// NamesakeOverride replaces existing tool with same name.
	NamesakeOverride NamesakeStrategy = "override"
	// NamesakeSkip keeps existing tool, ignores new one (CoPaw default).
	NamesakeSkip NamesakeStrategy = "skip"
	// NamesakeRaise returns error on duplicate tool name.
	NamesakeRaise NamesakeStrategy = "raise"
	// NamesakeRename auto-renames new tool (tool_2, tool_3, ...).
	NamesakeRename NamesakeStrategy = "rename"
)

// ToolChoice specifies tool selection behavior for chat completion.
// Use mode for auto/none/required, or set ForceTool for a specific tool.
type ToolChoice struct {
	Mode      ToolChoiceMode `json:"mode,omitempty"`       // auto, none, required
	ForceTool string         `json:"force_tool,omitempty"` // specific tool name to force
}

// ChatRequest holds the input for a chat completion.
type ChatRequest struct {
	Messages    []Message   `json:"messages"`
	Tools       []ToolDef   `json:"tools,omitempty"`
	ToolChoice  *ToolChoice `json:"tool_choice,omitempty"` // nil defaults to auto
	Temperature float64     `json:"temperature,omitempty"`
	MaxTokens   int         `json:"max_tokens,omitempty"`
}

// ChatResponse holds the output of a chat completion.
type ChatResponse struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Usage     Usage      `json:"usage"`
}

// Message represents a single chat message.
type Message struct {
	Role       string     `json:"role"` // system, user, assistant, tool
	Content    string     `json:"content"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

// ToolCall represents a tool invocation request.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON string
}

// ToolDef describes a tool for LLM tool selection.
type ToolDef struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  any    `json:"parameters"` // JSON Schema
}

// Usage holds token usage statistics.
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatChunk represents a streaming response chunk.
type ChatChunk struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
}

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

// MemorySummarizer is an optional extension for generating standalone summaries.
// MemoryStore implementations may implement this interface for /compact_str and similar features.
type MemorySummarizer interface {
	// SummaryMemory generates a summary of the given messages without modifying history.
	SummaryMemory(ctx context.Context, messages []Message) (string, error)
}

// MemoryResult represents a search hit from memory.
type MemoryResult struct {
	Content   string  `json:"content"`
	Score     float64 `json:"score"`
	Timestamp int64   `json:"timestamp"`
	ChunkID   string  `json:"chunk_id,omitempty"` // Unique chunk identifier for dedup
}

type memoryStoreKey struct{}
type chatIDKey struct{}

// WithMemoryStore attaches MemoryStore to context for tools (e.g. memory_search).
func WithMemoryStore(ctx context.Context, store MemoryStore) context.Context {
	return context.WithValue(ctx, memoryStoreKey{}, store)
}

// GetMemoryStore retrieves MemoryStore from context, or nil if not set.
func GetMemoryStore(ctx context.Context) MemoryStore {
	if s, ok := ctx.Value(memoryStoreKey{}).(MemoryStore); ok {
		return s
	}
	return nil
}

// WithChatID attaches chatID to context for tools that need it.
func WithChatID(ctx context.Context, chatID string) context.Context {
	return context.WithValue(ctx, chatIDKey{}, chatID)
}

// GetChatID retrieves chatID from context, or empty string if not set.
func GetChatID(ctx context.Context) string {
	if s, ok := ctx.Value(chatIDKey{}).(string); ok {
		return s
	}
	return ""
}

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

// RichExecutor is an optional Tool extension for multimodal results.
// Agent runtime checks for this interface; falls back to Tool.Execute if absent.
type RichExecutor interface {
	ExecuteRich(ctx context.Context, arguments string) (*ToolResult, error)
}

// FileSenderFunc delivers an attachment to the user via the current channel.
type FileSenderFunc func(ctx context.Context, att Attachment) error

type fileSenderKey struct{}

// WithFileSender attaches a FileSenderFunc to context.
func WithFileSender(ctx context.Context, fn FileSenderFunc) context.Context {
	return context.WithValue(ctx, fileSenderKey{}, fn)
}

// GetFileSender retrieves the FileSenderFunc from context, or nil.
func GetFileSender(ctx context.Context) FileSenderFunc {
	if fn, ok := ctx.Value(fileSenderKey{}).(FileSenderFunc); ok {
		return fn
	}
	return nil
}

// ModelSwitcher allows tools to switch the active model slot at runtime.
type ModelSwitcher interface {
	Switch(slotName string) error
	ActiveSlot() string
	SlotNames() []string
	HasCapability(cap string) bool
}

type modelSwitcherKey struct{}

// WithModelSwitcher attaches a ModelSwitcher to context.
func WithModelSwitcher(ctx context.Context, ms ModelSwitcher) context.Context {
	return context.WithValue(ctx, modelSwitcherKey{}, ms)
}

// GetModelSwitcher retrieves the ModelSwitcher from context, or nil.
func GetModelSwitcher(ctx context.Context) ModelSwitcher {
	if ms, ok := ctx.Value(modelSwitcherKey{}).(ModelSwitcher); ok {
		return ms
	}
	return nil
}

// Channel represents a messaging platform adapter.
type Channel interface {
	// Start begins listening for messages.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the channel.
	Stop(ctx context.Context) error

	// Name returns the channel identifier.
	Name() string
}

// MessageHandler is called when a channel receives a message.
type MessageHandler func(ctx context.Context, msg IncomingMessage) (string, error)

// IncomingMessage represents a message received from a channel.
type IncomingMessage struct {
	ChatID    string `json:"chat_id"`
	UserID    string `json:"user_id"`
	UserName  string `json:"user_name"`
	Content   string `json:"content"`
	Channel   string `json:"channel"`
	Timestamp int64  `json:"timestamp"`
}

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

// ============================================================================
// Structured Response Types - AI 协作约定
// ============================================================================

// StructuredResponse 表示 AI 的结构化响应约定。
// 这允许 AI 向框架暴露其内部状态和需求，实现智能协作。
type StructuredResponse struct {
	Thought            string           `json:"thought"`             // 当前思考过程
	CurrentFocus       string           `json:"current_focus"`       // 当前关注的部分
	NextStep           string           `json:"next_step"`           // 下一步计划
	CapabilitiesNeeded []string         `json:"capabilities_needed"` // 需要的能力（技能/MCP/工具）
	ProgressUpdate     string           `json:"progress_update"`     // 进度更新
	StorageRequests    []StorageRequest `json:"storage_requests"`    // 存储请求
	RetrievalRequests  []string         `json:"retrieval_requests"`  // 检索请求
	FinalAnswer        string           `json:"final_answer"`        // 给用户的最终回答
}

// StorageRequest 表示 AI 的存储请求。
// AI 可以将有价值的信息存储起来，供后续使用。
type StorageRequest struct {
	Name        string `json:"name"`        // 唯一名称
	Description string `json:"description"` // 简短描述
	Content     string `json:"content"`     // 要存储的内容
}

// StoredContent 表示已存储的内容。
type StoredContent struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Content     string    `json:"content"`
	Timestamp   int64     `json:"timestamp"`
	Tags        []string  `json:"tags,omitempty"` // 标签，用于检索
}

// Milestone 表示任务进度的里程碑。
type Milestone struct {
	Name      string `json:"name"`
	Status    string `json:"status"` // "pending", "in_progress", "completed"
	Timestamp int64  `json:"timestamp"`
}

// ============================================================================
// Context Manager Types
// ============================================================================

// contextManagerKey 是 context.Context 中 ContextManager 的键。
type contextManagerKey struct{}

// ContextManager 管理上下文和存储，作为 AI 的智能助手。
type ContextManager interface {
	// Store 存储内容供后续使用。
	Store(ctx context.Context, chatID string, requests []StorageRequest) error

	// Retrieve 检索已存储的内容。
	Retrieve(ctx context.Context, chatID string, names []string) ([]StoredContent, error)

	// ListAll 列出所有已存储的内容名称。
	ListAll(ctx context.Context, chatID string) ([]string, error)

	// SetGoal 设置当前目标。
	SetGoal(ctx context.Context, chatID string, goal string) error

	// GetGoal 获取当前目标。
	GetGoal(ctx context.Context, chatID string) (string, error)

	// AddMilestone 添加进度里程碑。
	AddMilestone(ctx context.Context, chatID string, milestone Milestone) error

	// GetMilestones 获取进度里程碑。
	GetMilestones(ctx context.Context, chatID string) ([]Milestone, error)

	// InjectContext 注入上下文到消息中（存储内容、目标、进展等）。
	InjectContext(ctx context.Context, chatID string, messages []Message, retrievalRequests []string) ([]Message, error)

	// BuildCapabilityReminder 构建能力提醒（智能提点）。
	BuildCapabilityReminder(ctx context.Context, chatID string, capabilitiesNeeded []string) string
}

// WithContextManager 将 ContextManager 附加到 context。
func WithContextManager(ctx context.Context, cm ContextManager) context.Context {
	return context.WithValue(ctx, contextManagerKey{}, cm)
}

// GetContextManager 从 context 获取 ContextManager，如果没有则返回 nil。
func GetContextManager(ctx context.Context) ContextManager {
	if cm, ok := ctx.Value(contextManagerKey{}).(ContextManager); ok {
		return cm
	}
	return nil
}

// ============================================================================
// Planning-Execution Architecture Types
// ============================================================================

// PlanningRequest 规划请求，包含用户消息和上下文。
type PlanningRequest struct {
	UserMessage       string `json:"user_message"`        // 用户原始消息
	CapabilitySummary string `json:"capability_summary"`  // AI 生成的能力总结
	Context           string `json:"context,omitempty"`   // 额外上下文 (如之前的对话摘要)
	ConversationID    string `json:"conversation_id,omitempty"` // 对话 ID
}

// Task 执行任务，描述计划中的一个具体任务。
type Task struct {
	ID            string         `json:"id"`                      // 任务唯一 ID
	Description   string         `json:"description"`             // 任务描述
	Capability    string         `json:"capability"`              // 需要使用的能力 ID
	Input         map[string]any `json:"input,omitempty"`        // 输入参数
	DependsOn     []string       `json:"depends_on,omitempty"`   // 依赖的任务 ID
	ExpectedOutput string        `json:"expected_output,omitempty"` // 预期输出描述
	Priority      int            `json:"priority,omitempty"`     // 优先级 (数字越大越优先)
}

// Plan 执行计划，包含完整的任务列表。
type Plan struct {
	Tasks          []Task `json:"tasks"`                    // 任务列表 (按依赖排序)
	Summary        string `json:"summary"`                  // 计划摘要
	EstimatedSteps int    `json:"estimated_steps,omitempty"` // 预估步骤数
	Reasoning      string `json:"reasoning,omitempty"`      // 规划推理过程
}

// TaskResult 任务执行结果，保存任务的执行输出和摘要。
type TaskResult struct {
	TaskID    string    `json:"task_id"`     // 任务 ID
	Output    string    `json:"output"`      // 完整输出
	Summary   string    `json:"summary"`     // AI 精简的摘要 (供下个任务使用)
	Status    string    `json:"status"`      // "success" | "failed" | "skipped"
	Error     string    `json:"error,omitempty"` // 错误信息
	Timestamp int64     `json:"timestamp"`   // 执行时间戳
	Metadata  map[string]any `json:"metadata,omitempty"` // 额外元数据
}

// TaskExecutor 任务执行器接口，负责执行任务并管理结果。
type TaskExecutor interface {
	// Execute 执行计划并返回最终结果。
	Execute(ctx context.Context, plan *Plan) (string, error)
	// GetResult 获取特定任务的执行结果。
	GetResult(taskID string) (*TaskResult, bool)
}

// Planner 任务规划器接口，负责生成执行计划。
type Planner interface {
	// Plan 根据用户请求生成执行计划。
	Plan(ctx context.Context, req *PlanningRequest) (*Plan, error)
}

// CapabilityExtractor 能力提取器接口，负责从系统中提取所有能力。
// 使用 interface{} 作为返回类型避免循环导入。
type CapabilityExtractor interface {
	// ExtractCapabilities 提取所有系统能力并生成总结。
	ExtractCapabilities(ctx context.Context) (interface{}, error)
	// GetRegistry 获取缓存的能力注册表。
	GetRegistry() (interface{}, error)
	// Refresh 强制刷新能力注册表。
	Refresh(ctx context.Context) error
	// GetSummary 获取能力总结。
	GetSummary() (string, error)
}
