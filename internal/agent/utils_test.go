package agent

import (
	"testing"
)

func TestCountStringTokens(t *testing.T) {
	if CountStringTokens("") != 0 {
		t.Error("empty string should be 0 tokens")
	}
	n := CountStringTokens("hello world this is a test")
	if n <= 0 {
		t.Errorf("expected positive count, got %d", n)
	}
}

// TestCountStringTokens_Tiktoken verifies tiktoken provides accurate token counts.
// These expected values are based on cl100k_base encoding (GPT-4, GPT-3.5-turbo).
func TestCountStringTokens_Tiktoken(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantMin int // minimum expected tokens
		wantMax int // maximum expected tokens
	}{
		{
			name:    "simple english",
			input:   "Hello, world!",
			wantMin: 3,
			wantMax: 5,
		},
		{
			name:    "code snippet",
			input:   "func main() { fmt.Println(\"hello\") }",
			wantMin: 10,
			wantMax: 20,
		},
		{
			name:    "chinese text",
			input:   "你好世界",
			wantMin: 2,
			wantMax: 8,
		},
		{
			name:    "mixed cjk and english",
			input:   "Hello 你好 World 世界",
			wantMin: 4,
			wantMax: 12,
		},
		{
			name:    "json data",
			input:   `{"name": "test", "value": 123, "nested": {"key": "value"}}`,
			wantMin: 15,
			wantMax: 30,
		},
		{
			name:    "long text",
			input:   "The quick brown fox jumps over the lazy dog. This is a common pangram used in typography.",
			wantMin: 15,
			wantMax: 25,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountStringTokens(tt.input)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("CountStringTokens(%q) = %d, want between %d and %d",
					tt.input, got, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// TestTokenCounter_ModelSpecific tests model-specific token counting.
func TestTokenCounter_ModelSpecific(t *testing.T) {
	tests := []struct {
		model string
		input string
	}{
		{"gpt-4", "Hello, world!"},
		{"gpt-3.5-turbo", "Hello, world!"},
		{"text-embedding-ada-002", "Hello, world!"},
		{"unknown-model", "Hello, world!"}, // should fall back to cl100k_base
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			tc := NewTokenCounter(tt.model)
			n := tc.Count(tt.input)
			if n <= 0 {
				t.Errorf("TokenCounter(%s).Count(%q) = %d, want > 0", tt.model, tt.input, n)
			}
		})
	}
}

// TestCountStringTokensForModel tests model-specific counting via helper function.
func TestCountStringTokensForModel(t *testing.T) {
	input := "Hello, this is a test message for token counting."

	gpt4Count := CountStringTokensForModel(input, "gpt-4")
	gpt35Count := CountStringTokensForModel(input, "gpt-3.5-turbo")
	unknownCount := CountStringTokensForModel(input, "unknown-model")

	// All should return positive counts
	if gpt4Count <= 0 {
		t.Errorf("gpt-4 count should be positive, got %d", gpt4Count)
	}
	if gpt35Count <= 0 {
		t.Errorf("gpt-3.5-turbo count should be positive, got %d", gpt35Count)
	}
	if unknownCount <= 0 {
		t.Errorf("unknown model count should be positive, got %d", unknownCount)
	}

	// GPT-4 and GPT-3.5-turbo use the same encoding (cl100k_base), so counts should match
	if gpt4Count != gpt35Count {
		t.Errorf("gpt-4 (%d) and gpt-3.5-turbo (%d) should have same token count", gpt4Count, gpt35Count)
	}
}

func TestCountMessageTokens(t *testing.T) {
	msgs := []Message{
		{Role: "user", Content: "hello world"},
		{Role: "assistant", Content: "hi there"},
	}
	n := CountMessageTokens(msgs)
	if n <= 0 {
		t.Errorf("expected positive count, got %d", n)
	}
}

func TestSafeCountMessageTokens(t *testing.T) {
	n := SafeCountMessageTokens(nil)
	if n != 0 {
		t.Errorf("expected 0 for nil, got %d", n)
	}
}

func TestCheckValidMessages_Valid(t *testing.T) {
	msgs := []Message{
		{Role: "assistant", ToolCalls: []ToolCall{{ID: "tc1", Name: "test"}}},
		{Role: "tool", ToolCallID: "tc1", Content: "result"},
	}
	if !CheckValidMessages(msgs) {
		t.Error("expected valid")
	}
}

func TestCheckValidMessages_Unpaired(t *testing.T) {
	msgs := []Message{
		{Role: "assistant", ToolCalls: []ToolCall{{ID: "tc1", Name: "test"}}},
	}
	if CheckValidMessages(msgs) {
		t.Error("expected invalid: tool call without result")
	}
}

func TestCheckValidMessages_OrphanResult(t *testing.T) {
	msgs := []Message{
		{Role: "tool", ToolCallID: "tc1", Content: "result"},
	}
	if CheckValidMessages(msgs) {
		t.Error("expected invalid: result without tool call")
	}
}

func TestSanitizeToolMessages_RemovesInvalid(t *testing.T) {
	msgs := []Message{
		{Role: "assistant", ToolCalls: []ToolCall{
			{ID: "tc1", Name: "test"},
			{ID: "", Name: ""},
		}},
		{Role: "tool", ToolCallID: "tc1", Content: "ok"},
		{Role: "tool", ToolCallID: "", Content: "orphan"},
	}
	result := SanitizeToolMessages(msgs)
	for _, m := range result {
		if m.Role == "tool" && m.ToolCallID == "" {
			t.Error("should have removed tool message with empty ToolCallID")
		}
		for _, tc := range m.ToolCalls {
			if tc.ID == "" {
				t.Error("should have removed tool call with empty ID")
			}
		}
	}
}

func TestSanitizeToolMessages_DedupToolCalls(t *testing.T) {
	msgs := []Message{
		{Role: "assistant", ToolCalls: []ToolCall{
			{ID: "tc1", Name: "test", Arguments: "a"},
			{ID: "tc1", Name: "test", Arguments: "b"},
		}},
		{Role: "tool", ToolCallID: "tc1", Content: "ok"},
	}
	result := SanitizeToolMessages(msgs)
	for _, m := range result {
		if len(m.ToolCalls) > 1 {
			t.Errorf("expected dedup to 1 tool call, got %d", len(m.ToolCalls))
		}
	}
}

func TestSanitizeToolMessages_ReordersResults(t *testing.T) {
	msgs := []Message{
		{Role: "tool", ToolCallID: "tc1", Content: "result1"},
		{Role: "assistant", ToolCalls: []ToolCall{{ID: "tc1", Name: "test"}}},
	}
	result := SanitizeToolMessages(msgs)
	foundAssistant := false
	for _, m := range result {
		if m.Role == "assistant" {
			foundAssistant = true
		}
		if m.Role == "tool" && !foundAssistant {
			t.Error("tool result should come after assistant message")
		}
	}
}

func TestTruncateText_Short(t *testing.T) {
	s := "hello"
	if TruncateText(s, 100) != s {
		t.Error("short text should not be truncated")
	}
}

func TestTruncateText_Long(t *testing.T) {
	s := "abcdefghijklmnopqrstuvwxyz0123456789abcdefghijklmnopqrstuvwxyz"
	result := TruncateText(s, 30)
	if len(result) > 50 {
		t.Errorf("expected truncated text, got len %d", len(result))
	}
	if result == s {
		t.Error("expected text to be truncated")
	}
}

func TestRepairEmptyToolInputs(t *testing.T) {
	msgs := []Message{
		{Role: "assistant", ToolCalls: []ToolCall{
			{ID: "tc1", Name: "test", Arguments: `{"key":"value"}`},
			{ID: "tc2", Name: "test2", Arguments: "not json"},
		}},
	}
	result := RepairEmptyToolInputs(msgs)
	if result[0].ToolCalls[0].Arguments != `{"key":"value"}` {
		t.Error("valid JSON should be preserved")
	}
	if result[0].ToolCalls[1].Arguments != "{}" {
		t.Errorf("invalid JSON should be replaced with {}, got %q", result[0].ToolCalls[1].Arguments)
	}
}
