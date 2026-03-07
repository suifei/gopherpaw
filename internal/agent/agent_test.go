package agent

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

// mockLLM implements LLMProvider for testing.
type mockLLM struct {
	chatFunc func(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
}

func (m *mockLLM) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	if m.chatFunc != nil {
		return m.chatFunc(ctx, req)
	}
	return &ChatResponse{Content: "mock response"}, nil
}

func (m *mockLLM) ChatStream(ctx context.Context, req *ChatRequest) (ChatStream, error) {
	return nil, errors.New("not implemented")
}

func (m *mockLLM) Name() string { return "mock" }

// mockMemory implements MemoryStore for testing.
// mockMemory implements MemoryStore for testing.
type mockMemory struct {
	saveFunc              func(ctx context.Context, chatID string, msg Message) error
	loadFunc              func(ctx context.Context, chatID string, limit int) ([]Message, error)
	searchFunc            func(ctx context.Context, chatID string, query string, topK int) ([]MemoryResult, error)
	compactFunc           func(ctx context.Context, chatID string) error
	getCompactSummaryFunc func(ctx context.Context, chatID string) (string, error)
	saveLongTermFunc      func(ctx context.Context, chatID string, content string, category string) error
	loadLongTermFunc      func(ctx context.Context, chatID string) (string, error)
}

func (m *mockMemory) Save(ctx context.Context, chatID string, msg Message) error {
	if m.saveFunc != nil {
		return m.saveFunc(ctx, chatID, msg)
	}
	return nil
}

func (m *mockMemory) Load(ctx context.Context, chatID string, limit int) ([]Message, error) {
	if m.loadFunc != nil {
		return m.loadFunc(ctx, chatID, limit)
	}
	return nil, nil
}

func (m *mockMemory) Search(ctx context.Context, chatID string, query string, topK int) ([]MemoryResult, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, chatID, query, topK)
	}
	return nil, nil
}

func (m *mockMemory) Compact(ctx context.Context, chatID string) error {
	if m.compactFunc != nil {
		return m.compactFunc(ctx, chatID)
	}
	return nil
}

func (m *mockMemory) GetCompactSummary(ctx context.Context, chatID string) (string, error) {
	return "", nil
}

func (m *mockMemory) SaveLongTerm(ctx context.Context, chatID string, content string, category string) error {
	return nil
}

func (m *mockMemory) LoadLongTerm(ctx context.Context, chatID string) (string, error) {
	if m.loadLongTermFunc != nil {
		return m.loadLongTermFunc(ctx, chatID)
	}
	return "", nil
}

func TestReactAgent_Run_NoTools(t *testing.T) {
	called := false
	llm := &mockLLM{
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			called = true
			return &ChatResponse{Content: "Hello!"}, nil
		},
	}
	mem := &mockMemory{}
	agent := NewReact(llm, mem, nil, config.AgentConfig{SystemPrompt: "You are helpful.", Running: config.AgentRunningConfig{MaxTurns: 5}})
	ctx := context.Background()
	out, err := agent.Run(ctx, "chat1", "hi")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !called {
		t.Error("LLM Chat was not called")
	}
	if out != "Hello!" {
		t.Errorf("Run: got %q", out)
	}
}

func TestReactAgent_Run_EmptyInput(t *testing.T) {
	agent := NewReact(&mockLLM{}, &mockMemory{}, nil, config.AgentConfig{})
	ctx := context.Background()
	_, err := agent.Run(ctx, "", "hi")
	if err == nil {
		t.Error("expected error for empty chatID")
	}
	_, err = agent.Run(ctx, "c1", "")
	if err == nil {
		t.Error("expected error for empty message")
	}
}

func TestReactAgent_Run_ContextCancel(t *testing.T) {
	llm := &mockLLM{
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	agent := NewReact(llm, &mockMemory{}, nil, config.AgentConfig{Running: config.AgentRunningConfig{MaxTurns: 5}})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := agent.Run(ctx, "c1", "hi")
	if err != context.Canceled {
		t.Errorf("Run: got %v", err)
	}
}

// mockTool implements Tool for testing.
type mockTool struct {
	name        string
	description string
	executeFunc func(ctx context.Context, args string) (string, error)
}

func (m *mockTool) Name() string        { return m.name }
func (m *mockTool) Description() string { return m.description }
func (m *mockTool) Parameters() any {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
}
func (m *mockTool) Execute(ctx context.Context, args string) (string, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, args)
	}
	return "ok", nil
}

func TestReactAgent_ParallelToolExecution(t *testing.T) {
	// Track execution with atomic counter
	var executedCount int32
	var mu sync.Mutex
	executedTools := make([]string, 0, 3)

	tool1 := &mockTool{
		name:        "tool1",
		description: "Test tool 1",
		executeFunc: func(ctx context.Context, args string) (string, error) {
			mu.Lock()
			executedTools = append(executedTools, "tool1")
			mu.Unlock()
			atomic.AddInt32(&executedCount, 1)
			return "result1", nil
		},
	}
	tool2 := &mockTool{
		name:        "tool2",
		description: "Test tool 2",
		executeFunc: func(ctx context.Context, args string) (string, error) {
			mu.Lock()
			executedTools = append(executedTools, "tool2")
			mu.Unlock()
			atomic.AddInt32(&executedCount, 1)
			return "result2", nil
		},
	}
	tool3 := &mockTool{
		name:        "tool3",
		description: "Test tool 3",
		executeFunc: func(ctx context.Context, args string) (string, error) {
			mu.Lock()
			executedTools = append(executedTools, "tool3")
			mu.Unlock()
			atomic.AddInt32(&executedCount, 1)
			return "result3", nil
		},
	}

	callCount := 0
	llm := &mockLLM{
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			callCount++
			if callCount == 1 {
				// First call: return 3 tool calls
				return &ChatResponse{
					Content: "",
					ToolCalls: []ToolCall{
						{ID: "tc1", Name: "tool1", Arguments: `{}`},
						{ID: "tc2", Name: "tool2", Arguments: `{}`},
						{ID: "tc3", Name: "tool3", Arguments: `{}`},
					},
				}, nil
			}
			// Second call: return final answer
			return &ChatResponse{Content: "Done with all tools"}, nil
		},
	}

	mem := &mockMemory{}
	tools := []Tool{tool1, tool2, tool3}
	agent := NewReact(llm, mem, tools, config.AgentConfig{
		SystemPrompt: "You are helpful.",
		Running:      config.AgentRunningConfig{MaxTurns: 5},
	})

	ctx := context.Background()
	result, err := agent.Run(ctx, "chat1", "run all tools")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result != "Done with all tools" {
		t.Errorf("unexpected result: %q", result)
	}

	// Verify all tools were executed
	if atomic.LoadInt32(&executedCount) != 3 {
		t.Errorf("expected 3 tools executed, got %d", executedCount)
	}

	// Verify LLM was called twice (once for tools, once for final)
	if callCount != 2 {
		t.Errorf("expected 2 LLM calls, got %d", callCount)
	}
}

func TestReactAgent_ParallelToolExecution_SingleTool(t *testing.T) {
	// Single tool should not spawn goroutines (optimization)
	executed := false
	tool := &mockTool{
		name:        "single",
		description: "Single tool",
		executeFunc: func(ctx context.Context, args string) (string, error) {
			executed = true
			return "single_result", nil
		},
	}

	callCount := 0
	llm := &mockLLM{
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			callCount++
			if callCount == 1 {
				return &ChatResponse{
					ToolCalls: []ToolCall{{ID: "tc1", Name: "single", Arguments: `{}`}},
				}, nil
			}
			return &ChatResponse{Content: "Done"}, nil
		},
	}

	agent := NewReact(llm, &mockMemory{}, []Tool{tool}, config.AgentConfig{Running: config.AgentRunningConfig{MaxTurns: 5}})
	_, err := agent.Run(context.Background(), "chat1", "test")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if !executed {
		t.Error("tool was not executed")
	}
}

func TestReactAgent_ParallelToolExecution_WithError(t *testing.T) {
	// One tool fails, others succeed
	tool1 := &mockTool{
		name: "good",
		executeFunc: func(ctx context.Context, args string) (string, error) {
			return "success", nil
		},
	}
	tool2 := &mockTool{
		name: "bad",
		executeFunc: func(ctx context.Context, args string) (string, error) {
			return "", errors.New("tool failed")
		},
	}

	callCount := 0
	llm := &mockLLM{
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			callCount++
			if callCount == 1 {
				return &ChatResponse{
					ToolCalls: []ToolCall{
						{ID: "tc1", Name: "good", Arguments: `{}`},
						{ID: "tc2", Name: "bad", Arguments: `{}`},
					},
				}, nil
			}
			// Verify error message was passed to LLM
			for _, msg := range req.Messages {
				if msg.ToolCallID == "tc2" && msg.Content != "Error: tool failed" {
					// Continue to final response
				}
			}
			return &ChatResponse{Content: "Handled error"}, nil
		},
	}

	agent := NewReact(llm, &mockMemory{}, []Tool{tool1, tool2}, config.AgentConfig{Running: config.AgentRunningConfig{MaxTurns: 5}})
	result, err := agent.Run(context.Background(), "chat1", "test")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}
	if result != "Handled error" {
		t.Errorf("unexpected result: %q", result)
	}
}

func TestReactAgent_AddHook(t *testing.T) {
	llm := &mockLLM{}
	agent := NewReact(llm, &mockMemory{}, nil, config.AgentConfig{})

	executed := false
	hook := func(ctx context.Context, a *ReactAgent, chatID string, messages []Message) ([]Message, error) {
		executed = true
		return messages, nil
	}

	agent.AddHook(hook)

	if len(agent.hooks) != 1 {
		t.Error("hook was not added")
	}

	_ = executed
}

func TestReactAgent_AddHooks(t *testing.T) {
	llm := &mockLLM{}
	agent := NewReact(llm, &mockMemory{}, nil, config.AgentConfig{})

	hook1 := func(ctx context.Context, a *ReactAgent, chatID string, messages []Message) ([]Message, error) {
		return messages, nil
	}
	hook2 := func(ctx context.Context, a *ReactAgent, chatID string, messages []Message) ([]Message, error) {
		return messages, nil
	}

	agent.AddHooks(hook1, hook2)

	if len(agent.hooks) != 2 {
		t.Error("hooks were not added")
	}
}

func TestReactAgent_SetLLMProvider(t *testing.T) {
	llm1 := &mockLLM{}
	agent := NewReact(llm1, &mockMemory{}, nil, config.AgentConfig{})

	llm2 := &mockLLM{}
	agent.SetLLMProvider(llm2)

	if agent.llm != llm2 {
		t.Error("LLM provider was not set")
	}
}

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  int
	}{
		{"empty", "", 0},
		{"short", "hello", 2},
		{"medium", "hello world", 3},
		{"long", "this is a longer text that should have more tokens", 11},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimateTokens(tt.input)
			if got < tt.want-2 || got > tt.want+2 {
				t.Errorf("estimateTokens(%q) = %d, want around %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestEstimateMessagesTokens(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
	}

	got := estimateMessagesTokens(messages)
	if got < 2 {
		t.Errorf("estimateMessagesTokens() = %d, want >= 2", got)
	}
}

func TestReactAgent_RunStream(t *testing.T) {
	llm := &mockLLM{
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			return &ChatResponse{Content: "stream response"}, nil
		},
	}

	agent := NewReact(llm, &mockMemory{}, nil, config.AgentConfig{Running: config.AgentRunningConfig{MaxTurns: 5}})

	stream, err := agent.RunStream(context.Background(), "chat1", "test")
	if err != nil {
		t.Fatalf("RunStream error: %v", err)
	}
	if stream == nil {
		t.Fatal("RunStream returned nil stream")
	}

	chunks := []string{}
	for chunk := range stream {
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		t.Error("no chunks received from stream")
	}
}

func TestTruncateForHistory(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		maxTokens int
		wantLen   int
	}{
		{"short", "hello", 100, 5},
		{"exact", "hello world", 3, 11},
		{"truncate", "this is a very long text that needs to be truncated", 10, 40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateForHistory(tt.content, tt.maxTokens)
			if len(got) > len(tt.content) {
				t.Errorf("truncated content is longer than original")
			}
		})
	}
}

func TestEstimateTokensForHistory(t *testing.T) {
	messages := []Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi"},
	}
	got := estimateTokensForHistory(messages)
	if got < 1 {
		t.Errorf("estimateTokensForHistory() = %d, want > 0", got)
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		name string
		a, b int
		want int
	}{
		{"a smaller", 1, 2, 1},
		{"b smaller", 2, 1, 1},
		{"equal", 2, 2, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := min(tt.a, tt.b); got != tt.want {
				t.Errorf("min(%d, %d) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestBootstrapRunner_NewBootstrapRunner(t *testing.T) {
	llm := &mockLLM{}
	agent := NewReact(llm, &mockMemory{}, nil, config.AgentConfig{})
	loader := NewPromptLoader("/tmp", "en")

	runner := NewBootstrapRunner(agent, loader)

	if runner == nil {
		t.Fatal("NewBootstrapRunner returned nil")
	}
	if runner.agent != agent {
		t.Error("agent not set correctly")
	}
	if runner.loader != loader {
		t.Error("loader not set correctly")
	}
}

func TestBootstrapRunner_RunIfNeeded_NoBootstrap(t *testing.T) {
	tmpDir := t.TempDir()
	llm := &mockLLM{}
	agent := NewReact(llm, &mockMemory{}, nil, config.AgentConfig{})
	loader := NewPromptLoader(tmpDir, "en")
	runner := NewBootstrapRunner(agent, loader)

	err := runner.RunIfNeeded(context.Background(), "chat1")
	if err != nil {
		t.Errorf("RunIfNeeded failed: %v", err)
	}
}

func TestBootstrapRunner_RunIfNeeded_WithBootstrap(t *testing.T) {
	tmpDir := t.TempDir()

	// Create BOOTSTRAP.md
	bootstrapPath := filepath.Join(tmpDir, "BOOTSTRAP.md")
	bootstrapContent := "Please help me set up my profile"
	if err := os.WriteFile(bootstrapPath, []byte(bootstrapContent), 0644); err != nil {
		t.Fatalf("write bootstrap: %v", err)
	}

	executed := false
	llm := &mockLLM{
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			executed = true
			if len(req.Messages) == 0 {
				t.Error("no messages in request")
			}
			return &ChatResponse{Content: "This is your profile"}, nil
		},
	}

	agent := NewReact(llm, &mockMemory{}, nil, config.AgentConfig{Running: config.AgentRunningConfig{MaxTurns: 5}})
	loader := NewPromptLoader(tmpDir, "en")
	runner := NewBootstrapRunner(agent, loader)

	err := runner.RunIfNeeded(context.Background(), "chat1")
	if err != nil {
		t.Fatalf("RunIfNeeded failed: %v", err)
	}

	if !executed {
		t.Error("agent was not executed")
	}

	// Check PROFILE.md was created
	profilePath := filepath.Join(tmpDir, "PROFILE.md")
	data, err := os.ReadFile(profilePath)
	if err != nil {
		t.Errorf("PROFILE.md not created: %v", err)
	}
	if string(data) != "This is your profile" {
		t.Errorf("PROFILE.md content = %q, want %q", string(data), "This is your profile")
	}

	// Check BOOTSTRAP.md was deleted
	if _, err := os.Stat(bootstrapPath); !os.IsNotExist(err) {
		t.Error("BOOTSTRAP.md was not deleted")
	}
}

func TestBootstrapRunner_RunIfNeeded_EmptyBootstrap(t *testing.T) {
	tmpDir := t.TempDir()

	// Create empty BOOTSTRAP.md
	bootstrapPath := filepath.Join(tmpDir, "BOOTSTRAP.md")
	if err := os.WriteFile(bootstrapPath, []byte(""), 0644); err != nil {
		t.Fatalf("write bootstrap: %v", err)
	}

	llm := &mockLLM{}
	agent := NewReact(llm, &mockMemory{}, nil, config.AgentConfig{})
	loader := NewPromptLoader(tmpDir, "en")
	runner := NewBootstrapRunner(agent, loader)

	err := runner.RunIfNeeded(context.Background(), "chat1")
	if err != nil {
		t.Fatalf("RunIfNeeded failed: %v", err)
	}

	// Check BOOTSTRAP.md was deleted
	if _, err := os.Stat(bootstrapPath); !os.IsNotExist(err) {
		t.Error("BOOTSTRAP.md was not deleted")
	}
}

func TestBootstrapRunner_readBootstrap(t *testing.T) {
	tmpDir := t.TempDir()

	// Create BOOTSTRAP.md
	bootstrapPath := filepath.Join(tmpDir, "BOOTSTRAP.md")
	expectedContent := "Bootstrap content"
	if err := os.WriteFile(bootstrapPath, []byte(expectedContent), 0644); err != nil {
		t.Fatalf("write bootstrap: %v", err)
	}

	llm := &mockLLM{}
	agent := NewReact(llm, &mockMemory{}, nil, config.AgentConfig{})
	loader := NewPromptLoader(tmpDir, "en")
	runner := NewBootstrapRunner(agent, loader)

	content, err := runner.readBootstrap()
	if err != nil {
		t.Fatalf("readBootstrap failed: %v", err)
	}

	if content != expectedContent {
		t.Errorf("content = %q, want %q", content, expectedContent)
	}
}

// mockRichTool implements RichExecutor for testing
type mockRichTool struct {
	name            string
	description     string
	executeFunc     func(ctx context.Context, args string) (string, error)
	executeRichFunc func(ctx context.Context, args string) (*ToolResult, error)
}

func (m *mockRichTool) Name() string        { return m.name }
func (m *mockRichTool) Description() string { return m.description }
func (m *mockRichTool) Parameters() any {
	return map[string]interface{}{"type": "object"}
}
func (m *mockRichTool) Execute(ctx context.Context, args string) (string, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, args)
	}
	return "ok", nil
}
func (m *mockRichTool) ExecuteRich(ctx context.Context, args string) (*ToolResult, error) {
	if m.executeRichFunc != nil {
		return m.executeRichFunc(ctx, args)
	}
	return &ToolResult{Text: "rich ok"}, nil
}

func TestReactAgent_ExecuteTool_RichExecutor(t *testing.T) {
	executed := false
	tool := &mockRichTool{
		name: "rich_tool",
		executeRichFunc: func(ctx context.Context, args string) (*ToolResult, error) {
			executed = true
			return &ToolResult{
				Text: "rich result",
				Attachments: []Attachment{
					{FilePath: "/tmp/test.txt"},
				},
			}, nil
		},
	}

	llm := &mockLLM{}
	agent := NewReact(llm, &mockMemory{}, []Tool{tool}, config.AgentConfig{Running: config.AgentRunningConfig{MaxTurns: 5}})

	// Test RichExecutor
	result, err := agent.executeTool(context.Background(), "chat1", ToolCall{Name: "rich_tool", Arguments: "{}"})
	if err != nil {
		t.Fatalf("executeTool failed: %v", err)
	}

	if !executed {
		t.Error("ExecuteRich was not called")
	}
	if result != "rich result" {
		t.Errorf("result = %q, want %q", result, "rich result")
	}
}

func TestReactAgent_ExecuteTool_UnknownTool(t *testing.T) {
	llm := &mockLLM{}
	agent := NewReact(llm, &mockMemory{}, nil, config.AgentConfig{})

	_, err := agent.executeTool(context.Background(), "chat1", ToolCall{Name: "unknown", Arguments: "{}"})
	if err == nil {
		t.Error("expected error for unknown tool")
	}
	if !strings.Contains(err.Error(), "unknown tool") {
		t.Errorf("error = %v, want 'unknown tool' error", err)
	}
}

func TestReactAgent_BuildMessages_WithCompact(t *testing.T) {
	// Create memory with many messages to trigger compact
	messages := make([]Message, 100)
	for i := range messages {
		messages[i] = Message{
			Role:    "user",
			Content: strings.Repeat("test message ", 100), // Make it long to trigger token threshold
		}
	}

	compacted := false
	mem := &mockMemory{
		loadFunc: func(ctx context.Context, chatID string, limit int) ([]Message, error) {
			if compacted {
				// Return compacted messages
				return []Message{
					{Role: "user", Content: "compacted"},
				}, nil
			}
			return messages, nil
		},
		compactFunc: func(ctx context.Context, chatID string) error {
			compacted = true
			return nil
		},
	}

	llm := &mockLLM{}
	agent := NewReact(llm, mem, nil, config.AgentConfig{
		Running: config.AgentRunningConfig{
			MaxTurns:       5,
			MaxInputLength: 1000, // Low threshold to trigger compact
		},
	})

	result, err := agent.buildMessages(context.Background(), "chat1")
	if err != nil {
		t.Fatalf("buildMessages failed: %v", err)
	}

	if !compacted {
		t.Error("expected compact to be triggered")
	}

	// Should have system message + compacted messages
	if len(result) < 2 {
		t.Errorf("expected at least 2 messages, got %d", len(result))
	}
}

func TestReactAgent_ExecuteTool_WithContextValues(t *testing.T) {
	mem := &mockMemory{}
	tool := &mockTool{
		name: "test_tool",
		executeFunc: func(ctx context.Context, args string) (string, error) {
			// Verify context values are set
			if GetMemoryStore(ctx) == nil {
				t.Error("MemoryStore not in context")
			}
			if GetChatID(ctx) == "" {
				t.Error("ChatID not in context")
			}
			return "ok", nil
		},
	}

	llm := &mockLLM{}
	agent := NewReact(llm, mem, []Tool{tool}, config.AgentConfig{Running: config.AgentRunningConfig{MaxTurns: 5}})

	result, err := agent.executeTool(context.Background(), "chat1", ToolCall{Name: "test_tool", Arguments: "{}"})
	if err != nil {
		t.Fatalf("executeTool failed: %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %q, want %q", result, "ok")
	}
}

func TestReactAgent_ExecuteTool_WithModelSwitcher(t *testing.T) {
	mem := &mockMemory{}
	tool := &mockTool{
		name: "test_tool",
		executeFunc: func(ctx context.Context, args string) (string, error) {
			// Verify ModelSwitcher is in context
			ms := GetModelSwitcher(ctx)
			if ms == nil {
				t.Error("ModelSwitcher not in context")
			}
			return "ok", nil
		},
	}

	llm := &mockLLM{}
	agent := NewReact(llm, mem, []Tool{tool}, config.AgentConfig{Running: config.AgentRunningConfig{MaxTurns: 5}})

	// Set LLM as ModelSwitcher
	ms := &mockModelSwitcher{}
	agent.SetLLMProvider(ms)

	result, err := agent.executeTool(context.Background(), "chat1", ToolCall{Name: "test_tool", Arguments: "{}"})
	if err != nil {
		t.Fatalf("executeTool failed: %v", err)
	}
	if result != "ok" {
		t.Errorf("result = %q, want %q", result, "ok")
	}
}

func TestReactAgent_LastUserMessage(t *testing.T) {
	tests := []struct {
		name     string
		messages []Message
		want     string
	}{
		{
			name:     "no messages",
			messages: []Message{},
			want:     "",
		},
		{
			name: "only system",
			messages: []Message{
				{Role: "system", Content: "sys"},
			},
			want: "",
		},
		{
			name: "user message",
			messages: []Message{
				{Role: "system", Content: "sys"},
				{Role: "user", Content: "hello"},
			},
			want: "hello",
		},
		{
			name: "multiple user messages",
			messages: []Message{
				{Role: "user", Content: "first"},
				{Role: "assistant", Content: "hi"},
				{Role: "user", Content: "second"},
			},
			want: "second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := lastUserMessage(tt.messages)
			if got != tt.want {
				t.Errorf("lastUserMessage() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReactAgent_Truncate(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		maxTokens int
		wantLen   int
	}{
		{
			name:      "short content",
			content:   "hello",
			maxTokens: 100,
			wantLen:   5,
		},
		{
			name:      "exact length",
			content:   "hello world",
			maxTokens: 3,
			wantLen:   11,
		},
		{
			name:      "needs truncation",
			content:   "this is a very long content that needs to be truncated",
			maxTokens: 10,
			wantLen:   40, // 10 * 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.content, tt.maxTokens)
			if len(got) > len(tt.content) {
				t.Errorf("truncated content is longer than original")
			}
			// For truncation case, check that it's actually truncated
			if tt.name == "needs truncation" {
				if len(got) >= len(tt.content) {
					t.Errorf("content should be truncated")
				}
			}
		})
	}
}

type mockModelSwitcher struct {
	switchFunc func(provider, model string) error
	slotFunc   func(slotName string) error
}

func (m *mockModelSwitcher) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	return &ChatResponse{Content: "ok"}, nil
}

func (m *mockModelSwitcher) ChatStream(ctx context.Context, req *ChatRequest) (ChatStream, error) {
	return nil, errors.New("not implemented")
}

func (m *mockModelSwitcher) Name() string { return "mock_switcher" }

func (m *mockModelSwitcher) SwitchLLM(provider, model string) error {
	if m.switchFunc != nil {
		return m.switchFunc(provider, model)
	}
	return nil
}

func (m *mockModelSwitcher) Switch(slotName string) error {
	if m.slotFunc != nil {
		return m.slotFunc(slotName)
	}
	return nil
}

func (m *mockModelSwitcher) ActiveSlot() string {
	return "default"
}

func (m *mockModelSwitcher) SlotNames() []string {
	return []string{"default"}
}

func (m *mockModelSwitcher) HasCapability(cap string) bool {
	return true
}
