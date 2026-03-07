package agent

import (
	"context"
	"testing"
)

func TestWithProgressReporter(t *testing.T) {
	ctx := context.Background()
	reporter := &mockProgressReporter{}

	ctx = WithProgressReporter(ctx, reporter)

	got := getProgressReporter(ctx)
	if got == nil {
		t.Error("ProgressReporter not set correctly")
	}
}

func TestGetProgressReporter_Nil(t *testing.T) {
	ctx := context.Background()

	got := getProgressReporter(ctx)
	if got != nil {
		t.Error("expected nil ProgressReporter")
	}
}

func TestWithDaemonInfo(t *testing.T) {
	ctx := context.Background()
	daemonInfo := &DaemonInfo{
		Status:  "running",
		Version: "1.0.0",
	}

	ctx = WithDaemonInfo(ctx, daemonInfo)

	got := getDaemonInfo(ctx)
	if got != daemonInfo {
		t.Error("DaemonInfo not set correctly")
	}
}

func TestGetDaemonInfo_Nil(t *testing.T) {
	ctx := context.Background()

	got := getDaemonInfo(ctx)
	if got != nil {
		t.Error("expected nil DaemonInfo")
	}
}

func TestGetMemoryStore(t *testing.T) {
	ctx := context.Background()
	mem := &mockMemory{}

	ctx = WithMemoryStore(ctx, mem)

	got := GetMemoryStore(ctx)
	if got != mem {
		t.Error("MemoryStore not set correctly")
	}
}

func TestGetMemoryStore_Nil(t *testing.T) {
	ctx := context.Background()

	got := GetMemoryStore(ctx)
	if got != nil {
		t.Error("expected nil MemoryStore")
	}
}

func TestGetChatID(t *testing.T) {
	ctx := context.Background()
	chatID := "test_chat"

	ctx = WithChatID(ctx, chatID)

	got := GetChatID(ctx)
	if got != chatID {
		t.Errorf("ChatID = %q, want %q", got, chatID)
	}
}

func TestGetChatID_Empty(t *testing.T) {
	ctx := context.Background()

	got := GetChatID(ctx)
	if got != "" {
		t.Errorf("ChatID = %q, want empty", got)
	}
}

func TestWithFileSender(t *testing.T) {
	ctx := context.Background()
	sender := func(ctx context.Context, att Attachment) error {
		return nil
	}

	ctx = WithFileSender(ctx, sender)

	got := GetFileSender(ctx)
	if got == nil {
		t.Error("FileSender not set correctly")
	}
}

func TestGetFileSender_Nil(t *testing.T) {
	ctx := context.Background()

	got := GetFileSender(ctx)
	if got != nil {
		t.Error("expected nil FileSender")
	}
}

type mockProgressReporter struct{}

func (m *mockProgressReporter) OnThinking()                                 {}
func (m *mockProgressReporter) OnToolCall(toolName string, args string)     {}
func (m *mockProgressReporter) OnToolResult(toolName string, result string) {}
func (m *mockProgressReporter) OnFinalReply(content string)                 {}
