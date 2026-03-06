package llm

import (
	"context"
	"testing"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
)

func TestNewAnthropic(t *testing.T) {
	cfg := config.LLMConfig{
		Provider: "anthropic",
		Model:    "claude-3-opus-20240229",
		APIKey:   "test-api-key",
		BaseURL:  "https://api.anthropic.com",
	}

	provider, err := NewAnthropic(cfg)
	if err != nil {
		t.Fatalf("NewAnthropic failed: %v", err)
	}

	if provider.Name() != "anthropic" {
		t.Errorf("expected provider name 'anthropic', got %q", provider.Name())
	}
}

func TestAnthropicProvider_toAnthropicMessages(t *testing.T) {
	cfg := config.LLMConfig{Model: "claude-3-opus-20240229"}
	provider, _ := NewAnthropic(cfg)

	tests := []struct {
		name          string
		messages      []agent.Message
		wantMsgCount  int
		wantSystemLen int // 0 means empty
		wantHasSystem bool
	}{
		{
			name: "simple user message",
			messages: []agent.Message{
				{Role: "user", Content: "Hello"},
			},
			wantMsgCount:  1,
			wantHasSystem: false,
		},
		{
			name: "system + user messages",
			messages: []agent.Message{
				{Role: "system", Content: "You are helpful."},
				{Role: "user", Content: "Hi"},
			},
			wantMsgCount:  1, // system is extracted, only user remains
			wantHasSystem: true,
		},
		{
			name: "assistant with tool calls",
			messages: []agent.Message{
				{Role: "user", Content: "Use a tool"},
				{
					Role:    "assistant",
					Content: "",
					ToolCalls: []agent.ToolCall{
						{ID: "tc1", Name: "test_tool", Arguments: `{"arg": "value"}`},
					},
				},
			},
			wantMsgCount:  2,
			wantHasSystem: false,
		},
		{
			name: "tool result",
			messages: []agent.Message{
				{Role: "user", Content: "Use a tool"},
				{
					Role:    "assistant",
					Content: "",
					ToolCalls: []agent.ToolCall{
						{ID: "tc1", Name: "test_tool", Arguments: `{}`},
					},
				},
				{Role: "tool", ToolCallID: "tc1", Content: "tool result"},
			},
			wantMsgCount:  3, // user, assistant, user (with tool result)
			wantHasSystem: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &agent.ChatRequest{Messages: tt.messages}
			msgs, systemPrompt := provider.toAnthropicMessages(req)

			if len(msgs) != tt.wantMsgCount {
				t.Errorf("got %d messages, want %d", len(msgs), tt.wantMsgCount)
			}

			if tt.wantHasSystem && systemPrompt == "" {
				t.Error("expected system prompt to be set")
			}
			if !tt.wantHasSystem && systemPrompt != "" {
				t.Errorf("unexpected system prompt: %q", systemPrompt)
			}
		})
	}
}

func TestAnthropicProvider_toAnthropicTools(t *testing.T) {
	cfg := config.LLMConfig{Model: "claude-3-opus-20240229"}
	provider, _ := NewAnthropic(cfg)

	tools := []agent.ToolDef{
		{
			Name:        "test_tool",
			Description: "A test tool",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"arg1": map[string]interface{}{
						"type":        "string",
						"description": "First argument",
					},
				},
			},
		},
	}

	result := provider.toAnthropicTools(tools)
	if len(result) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result))
	}

	if result[0].OfTool == nil {
		t.Fatal("expected OfTool to be set")
	}

	if result[0].OfTool.Name != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got %q", result[0].OfTool.Name)
	}
}

func TestAnthropicProvider_toAnthropicToolChoice(t *testing.T) {
	cfg := config.LLMConfig{Model: "claude-3-opus-20240229"}
	provider, _ := NewAnthropic(cfg)

	tools := []agent.ToolDef{
		{Name: "tool1", Description: "Tool 1"},
		{Name: "tool2", Description: "Tool 2"},
	}

	tests := []struct {
		name       string
		toolChoice *agent.ToolChoice
		wantAuto   bool
		wantAny    bool
		wantNone   bool
		wantTool   bool
	}{
		{
			name:       "nil defaults to empty",
			toolChoice: nil,
		},
		{
			name:       "auto mode",
			toolChoice: &agent.ToolChoice{Mode: agent.ToolChoiceAuto},
			wantAuto:   true,
		},
		{
			name:       "required mode",
			toolChoice: &agent.ToolChoice{Mode: agent.ToolChoiceRequired},
			wantAny:    true,
		},
		{
			name:       "none mode",
			toolChoice: &agent.ToolChoice{Mode: agent.ToolChoiceNone},
			wantNone:   true,
		},
		{
			name:       "force specific tool",
			toolChoice: &agent.ToolChoice{ForceTool: "tool1"},
			wantTool:   true,
		},
		{
			name:       "force non-existent tool returns empty",
			toolChoice: &agent.ToolChoice{ForceTool: "nonexistent"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := provider.toAnthropicToolChoice(tt.toolChoice, tools)

			if tt.wantAuto && result.OfAuto == nil {
				t.Error("expected OfAuto to be set")
			}
			if tt.wantAny && result.OfAny == nil {
				t.Error("expected OfAny to be set")
			}
			if tt.wantNone && result.OfNone == nil {
				t.Error("expected OfNone to be set")
			}
			if tt.wantTool && result.OfTool == nil {
				t.Error("expected OfTool to be set")
			}
		})
	}
}

func TestAnthropicProvider_isTransientError(t *testing.T) {
	cfg := config.LLMConfig{}
	provider, _ := NewAnthropic(cfg)

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "rate limit error",
			err:  context.DeadlineExceeded, // not a transient in our definition
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := provider.isTransientError(tt.err)
			if got != tt.want {
				t.Errorf("isTransientError() = %v, want %v", got, tt.want)
			}
		})
	}
}
