package channels

import (
	"testing"
)

func TestChunkText(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		maxLen int
		want   []string
	}{
		{"empty", "", 100, nil},
		{"short", "hello", 100, []string{"hello"}},
		{"exact", "12345", 5, []string{"12345"}},
		{"split at space", "hello world", 11, []string{"hello world"}},
		{"split long", "aaaaaaaaaa", 3, []string{"aaa", "aaa", "aaa", "a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chunkText(tt.s, tt.maxLen)
			if len(got) != len(tt.want) {
				t.Errorf("chunkText() len = %v, want %v", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("chunkText()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestIncomingMessage(t *testing.T) {
	msg := IncomingMessage{
		ChatID:    "123",
		UserID:    "u1",
		UserName:  "alice",
		Content:   "hi",
		Channel:   "telegram",
		Timestamp: 12345,
		Metadata:  map[string]string{"chat_id": "123"},
	}
	if msg.Channel != "telegram" {
		t.Errorf("Channel = %q, want telegram", msg.Channel)
	}
	if msg.Metadata["chat_id"] != "123" {
		t.Errorf("Metadata[chat_id] = %q, want 123", msg.Metadata["chat_id"])
	}
}
