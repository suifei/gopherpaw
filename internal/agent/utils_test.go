package agent

import (
	"testing"
)

func TestCountStringTokensForModel(t *testing.T) {
	tests := []struct {
		name    string
		content string
		model   string
		want    int
	}{
		{"empty", "", "gpt-4", 0},
		{"short", "hello", "gpt-4", 1},
		{"medium", "hello world", "gpt-3.5-turbo", 2},
		{"long", "this is a longer text", "gpt-4", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CountStringTokensForModel(tt.content, tt.model)
			if got < 0 {
				t.Errorf("CountStringTokensForModel(%q, %q) = %d, want >= 0", tt.content, tt.model, got)
			}
		})
	}
}

func TestSafeCountMessageTokens(t *testing.T) {
	tests := []struct {
		name     string
		messages []Message
		want     int
	}{
		{
			name:     "empty",
			messages: []Message{},
			want:     0,
		},
		{
			name: "valid messages",
			messages: []Message{
				{Role: "user", Content: "hello"},
				{Role: "assistant", Content: "hi"},
			},
			want: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SafeCountMessageTokens(tt.messages)
			if got < tt.want {
				t.Errorf("SafeCountMessageTokens() = %d, want >= %d", got, tt.want)
			}
		})
	}
}

func TestCheckValidMessages(t *testing.T) {
	tests := []struct {
		name     string
		messages []Message
		want     bool
	}{
		{
			name:     "empty",
			messages: []Message{},
			want:     true,
		},
		{
			name: "valid messages",
			messages: []Message{
				{Role: "user", Content: "hello"},
				{Role: "assistant", Content: "hi"},
			},
			want: true,
		},
		{
			name: "with matched tool calls",
			messages: []Message{
				{Role: "assistant", ToolCalls: []ToolCall{{ID: "1", Name: "test"}}},
				{Role: "tool", ToolCallID: "1"},
			},
			want: true,
		},
		{
			name: "with unmatched tool calls",
			messages: []Message{
				{Role: "assistant", ToolCalls: []ToolCall{{ID: "1", Name: "test"}}},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CheckValidMessages(tt.messages)
			if got != tt.want {
				t.Errorf("CheckValidMessages() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDedupToolCalls(t *testing.T) {
	tests := []struct {
		name     string
		messages []Message
		wantLen  int
	}{
		{
			name:     "empty",
			messages: []Message{},
			wantLen:  0,
		},
		{
			name: "no duplicates",
			messages: []Message{
				{Role: "assistant", ToolCalls: []ToolCall{
					{ID: "1", Name: "tool1"},
					{ID: "2", Name: "tool2"},
				}},
			},
			wantLen: 1,
		},
		{
			name: "with duplicates",
			messages: []Message{
				{Role: "assistant", ToolCalls: []ToolCall{
					{ID: "1", Name: "tool1"},
					{ID: "1", Name: "tool1"},
				}},
			},
			wantLen: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := dedupToolCalls(tt.messages)
			if len(got) != tt.wantLen {
				t.Errorf("dedupToolCalls() returned %d messages, want %d", len(got), tt.wantLen)
			}
		})
	}
}
