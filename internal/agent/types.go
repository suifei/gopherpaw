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
