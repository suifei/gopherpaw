package memory

import (
	"context"
	"strings"
	"testing"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
)

func TestFileStore_SaveLoadLongTerm(t *testing.T) {
	dir := t.TempDir()
	cfg := config.MemoryConfig{WorkingDir: dir}
	fs := NewFileStore(cfg)
	ctx := context.Background()

	if err := fs.SaveLongTerm(ctx, "chat1", "key decision: use Go", "memory"); err != nil {
		t.Fatalf("SaveLongTerm: %v", err)
	}
	content, err := fs.LoadLongTerm(ctx, "chat1")
	if err != nil {
		t.Fatalf("LoadLongTerm: %v", err)
	}
	if content == "" {
		t.Error("expected content")
	}
	if !strings.Contains(content, "key decision") {
		t.Errorf("content missing key: %s", content)
	}
}

func TestFileStore_EmptyChatID(t *testing.T) {
	dir := t.TempDir()
	fs := NewFileStore(config.MemoryConfig{WorkingDir: dir})
	ctx := context.Background()
	if err := fs.SaveLongTerm(ctx, "", "x", "memory"); err == nil {
		t.Error("SaveLongTerm empty chatID should fail")
	}
	if _, err := fs.LoadLongTerm(ctx, ""); err == nil {
		t.Error("LoadLongTerm empty chatID should fail")
	}
}

func TestFileStore_HistoryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := config.MemoryConfig{WorkingDir: dir}
	fs := NewFileStore(cfg)
	ctx := context.Background()

	msgs := []storedMessage{
		{Msg: agent.Message{Role: "user", Content: "hi"}, Timestamp: 1},
		{Msg: agent.Message{Role: "assistant", Content: "hello"}, Timestamp: 2},
	}
	if err := fs.SaveHistory(ctx, "chat1", msgs); err != nil {
		t.Fatalf("SaveHistory: %v", err)
	}
	loaded, err := fs.LoadHistory(ctx, "chat1")
	if err != nil {
		t.Fatalf("LoadHistory: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(loaded))
	}
	if loaded[0].Msg.Content != "hi" || loaded[1].Msg.Content != "hello" {
		t.Errorf("content mismatch: %v", loaded)
	}
}

