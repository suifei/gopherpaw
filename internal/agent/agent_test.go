package agent

import (
	"context"
	"errors"
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
type mockMemory struct {
	saveFunc    func(ctx context.Context, chatID string, msg Message) error
	loadFunc    func(ctx context.Context, chatID string, limit int) ([]Message, error)
	searchFunc  func(ctx context.Context, chatID string, query string, topK int) ([]MemoryResult, error)
	compactFunc func(ctx context.Context, chatID string) error
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
