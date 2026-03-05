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
