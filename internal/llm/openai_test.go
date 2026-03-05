package llm

import (
	"context"
	"testing"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
)

func TestOpenAIProvider_Name(t *testing.T) {
	p := NewOpenAI(config.LLMConfig{})
	if p.Name() != "openai" {
		t.Errorf("Name() = %q, want openai", p.Name())
	}
}

func TestOpenAIProvider_Chat_NoAPIKey(t *testing.T) {
	p := NewOpenAI(config.LLMConfig{
		Model:  "gpt-4o-mini",
		APIKey: "",
	})
	ctx := context.Background()
	req := &agent.ChatRequest{
		Messages: []agent.Message{
			{Role: "user", Content: "hi"},
		},
	}
	_, err := p.Chat(ctx, req)
	if err == nil {
		t.Error("Chat with empty API key should fail")
	}
}

func TestOpenAIProvider_ChatStream_NoAPIKey(t *testing.T) {
	p := NewOpenAI(config.LLMConfig{
		Model:  "gpt-4o-mini",
		APIKey: "",
	})
	ctx := context.Background()
	req := &agent.ChatRequest{
		Messages: []agent.Message{
			{Role: "user", Content: "hi"},
		},
	}
	_, err := p.ChatStream(ctx, req)
	if err == nil {
		t.Error("ChatStream with empty API key should fail")
	}
}

func TestRegistry_Create(t *testing.T) {
	cfg := config.LLMConfig{Provider: "openai"}
	p, err := Create(cfg)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if p == nil {
		t.Fatal("Create returned nil provider")
	}
	if p.Name() != "openai" {
		t.Errorf("provider name = %q, want openai", p.Name())
	}
}

func TestRegistry_Create_Unknown(t *testing.T) {
	cfg := config.LLMConfig{Provider: "unknown"}
	_, err := Create(cfg)
	if err == nil {
		t.Error("Create with unknown provider should fail")
	}
}
