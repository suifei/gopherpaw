package llm

import (
	"context"
	"testing"

	"github.com/sashabaranov/go-openai"
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

func TestStreamToolCallAccumulator_Accumulate(t *testing.T) {
	acc := newStreamToolCallAccumulator()

	// Simulate first chunk with ID and Name
	idx0 := 0
	chunk1 := []openai.ToolCall{
		{
			Index: &idx0,
			ID:    "call_123",
			Function: openai.FunctionCall{
				Name:      "read_file",
				Arguments: `{"path":`,
			},
		},
	}
	result := acc.accumulate(chunk1)
	if len(result) != 1 {
		t.Fatalf("accumulate chunk1: got %d tool_calls, want 1", len(result))
	}
	if result[0].ID != "call_123" {
		t.Errorf("ID = %q, want call_123", result[0].ID)
	}
	if result[0].Name != "read_file" {
		t.Errorf("Name = %q, want read_file", result[0].Name)
	}
	if result[0].Arguments != `{"path":` {
		t.Errorf("Arguments = %q, want {\"path\":", result[0].Arguments)
	}

	// Simulate subsequent chunk with Arguments fragment
	chunk2 := []openai.ToolCall{
		{
			Index: &idx0,
			Function: openai.FunctionCall{
				Arguments: `"test.txt"}`,
			},
		},
	}
	result = acc.accumulate(chunk2)
	if len(result) != 1 {
		t.Fatalf("accumulate chunk2: got %d tool_calls, want 1", len(result))
	}
	want := `{"path":"test.txt"}`
	if result[0].Arguments != want {
		t.Errorf("Arguments = %q, want %q", result[0].Arguments, want)
	}
}

func TestStreamToolCallAccumulator_MultipleToolCalls(t *testing.T) {
	acc := newStreamToolCallAccumulator()

	idx0 := 0
	idx1 := 1

	// First tool_call
	acc.accumulate([]openai.ToolCall{
		{
			Index:    &idx0,
			ID:       "call_A",
			Function: openai.FunctionCall{Name: "tool_a", Arguments: `{"a":1}`},
		},
	})

	// Second tool_call
	acc.accumulate([]openai.ToolCall{
		{
			Index:    &idx1,
			ID:       "call_B",
			Function: openai.FunctionCall{Name: "tool_b", Arguments: `{"b":2}`},
		},
	})

	result := acc.snapshot()
	if len(result) != 2 {
		t.Fatalf("snapshot: got %d tool_calls, want 2", len(result))
	}
	if result[0].ID != "call_A" || result[1].ID != "call_B" {
		t.Errorf("IDs = [%s, %s], want [call_A, call_B]", result[0].ID, result[1].ID)
	}
}

func TestStreamToolCallAccumulator_Sanitize(t *testing.T) {
	acc := newStreamToolCallAccumulator()

	idx0 := 0
	idx1 := 1
	idx2 := 2

	// Valid tool_call
	acc.accumulate([]openai.ToolCall{
		{
			Index:    &idx0,
			ID:       "call_valid",
			Function: openai.FunctionCall{Name: "valid_tool", Arguments: `{"key":"value"}`},
		},
	})

	// Invalid: missing ID
	acc.accumulate([]openai.ToolCall{
		{
			Index:    &idx1,
			ID:       "",
			Function: openai.FunctionCall{Name: "no_id", Arguments: `{}`},
		},
	})

	// Invalid: missing Name
	acc.accumulate([]openai.ToolCall{
		{
			Index:    &idx2,
			ID:       "call_no_name",
			Function: openai.FunctionCall{Name: "", Arguments: `{}`},
		},
	})

	result := acc.sanitize()
	if len(result) != 1 {
		t.Fatalf("sanitize: got %d tool_calls, want 1 (valid only)", len(result))
	}
	if result[0].ID != "call_valid" {
		t.Errorf("ID = %q, want call_valid", result[0].ID)
	}
}

func TestSanitizeJSONArguments(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "valid", input: `{"key":"value"}`, want: `{"key":"value"}`},
		{name: "empty", input: "", want: "{}"},
		{name: "whitespace", input: "  ", want: "{}"},
		{name: "unclosed_brace", input: `{"key":"value"`, want: `{"key":"value"}`},
		{name: "unclosed_array", input: `{"arr":[1,2`, want: `{"arr":[1,2]}`},
		{name: "invalid_unfixable", input: `{{{`, want: "{}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sanitizeJSONArguments(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeJSONArguments(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTryFixJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "valid json", input: `{"a":1}`, want: ""},
		{name: "unclosed brace", input: `{"a":1`, want: `{"a":1}`},
		{name: "unclosed array", input: `{"a":[1,2`, want: `{"a":[1,2]}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tryFixJSON(tt.input)
			if got != tt.want {
				t.Errorf("tryFixJSON(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestTruncateStr(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{
			name:   "empty string",
			input:  "",
			maxLen: 10,
			want:   "",
		},
		{
			name:   "shorter than max",
			input:  "hello",
			maxLen: 10,
			want:   "hello",
		},
		{
			name:   "equal to max",
			input:  "hello",
			maxLen: 5,
			want:   "hello",
		},
		{
			name:   "longer than max",
			input:  "hello world",
			maxLen: 5,
			want:   "hello...",
		},
		{
			name:   "max length 0 or negative",
			input:  "hello",
			maxLen: 0,
			want:   "hello",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateStr(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateStr(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}
