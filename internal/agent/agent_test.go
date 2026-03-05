package agent

import (
	"context"
	"errors"
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
	saveFunc  func(ctx context.Context, chatID string, msg Message) error
	loadFunc  func(ctx context.Context, chatID string, limit int) ([]Message, error)
	searchFunc func(ctx context.Context, chatID string, query string, topK int) ([]MemoryResult, error)
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
	agent := NewReact(llm, mem, nil, config.AgentConfig{SystemPrompt: "You are helpful.", MaxTurns: 5})
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
	agent := NewReact(llm, &mockMemory{}, nil, config.AgentConfig{MaxTurns: 5})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := agent.Run(ctx, "c1", "hi")
	if err != context.Canceled {
		t.Errorf("Run: got %v", err)
	}
}
