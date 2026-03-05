package tools

import (
	"context"
	"testing"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/internal/memory"
)

func TestMemorySearchTool_NoStore(t *testing.T) {
	tool := &MemorySearchTool{}
	ctx := context.Background()
	result, err := tool.Execute(ctx, `{"query":"test"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "Error: Memory store is not available." {
		t.Errorf("expected store error, got %q", result)
	}
}

func TestMemorySearchTool_NoChatID(t *testing.T) {
	store := memory.New(config.MemoryConfig{})
	ctx := agent.WithMemoryStore(context.Background(), store)
	tool := &MemorySearchTool{}
	result, err := tool.Execute(ctx, `{"query":"test"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result != "Error: Chat context is not available." {
		t.Errorf("expected chatID error, got %q", result)
	}
}

func TestMemorySearchTool_Search(t *testing.T) {
	cfg := config.MemoryConfig{MaxHistory: 50}
	store := memory.New(cfg)
	ctx := agent.WithMemoryStore(agent.WithChatID(context.Background(), "c1"), store)
	tool := &MemorySearchTool{}
	_ = store.Save(ctx, "c1", agent.Message{Role: "user", Content: "deployment process"})
	_ = store.Save(ctx, "c1", agent.Message{Role: "assistant", Content: "discussed deployment"})
	result, err := tool.Execute(ctx, `{"query":"deployment","top_k":5}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result == "" {
		t.Error("expected results")
	}
	if result == "" {
		t.Error("expected non-empty result")
	}
	if result == "Error: Memory store is not available." || result == "Error: Chat context is not available." {
		t.Errorf("unexpected error result: %q", result)
	}
}
