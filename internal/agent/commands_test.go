package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestHandleMagicCommand_NotAMagicCommand(t *testing.T) {
	mem := &mockMemory{}
	result, handled, err := HandleMagicCommand(context.Background(), mem, "chat1", "hello", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if handled {
		t.Error("should not handle non-magic command")
	}
	if result != "" {
		t.Errorf("result should be empty, got %q", result)
	}
}

func TestHandleMagicCommand_Compact(t *testing.T) {
	compacted := false
	mem := &mockMemory{
		compactFunc: func(ctx context.Context, chatID string) error {
			compacted = true
			return nil
		},
	}

	result, handled, err := HandleMagicCommand(context.Background(), mem, "chat1", "/compact", nil)
	if err != nil {
		t.Fatalf("HandleMagicCommand failed: %v", err)
	}
	if !handled {
		t.Error("should handle /compact command")
	}
	if !compacted {
		t.Error("compact was not called")
	}
	if !strings.Contains(result, "压缩") {
		t.Errorf("result = %q, should contain '压缩'", result)
	}
}

func TestHandleMagicCommand_Clear(t *testing.T) {
	compacted := false
	mem := &mockMemory{
		compactFunc: func(ctx context.Context, chatID string) error {
			compacted = true
			return nil
		},
	}

	result, handled, err := HandleMagicCommand(context.Background(), mem, "chat1", "/clear", nil)
	if err != nil {
		t.Fatalf("HandleMagicCommand failed: %v", err)
	}
	if !handled {
		t.Error("should handle /clear command")
	}
	if !compacted {
		t.Error("compact was not called")
	}
	if !strings.Contains(result, "清空") {
		t.Errorf("result = %q, should contain '清空'", result)
	}
}

func TestHandleMagicCommand_History(t *testing.T) {
	mem := &mockMemory{
		loadFunc: func(ctx context.Context, chatID string, limit int) ([]Message, error) {
			return []Message{
				{Role: "user", Content: "hello"},
				{Role: "assistant", Content: "hi"},
			}, nil
		},
	}

	result, handled, err := HandleMagicCommand(context.Background(), mem, "chat1", "/history", nil)
	if err != nil {
		t.Fatalf("HandleMagicCommand failed: %v", err)
	}
	if !handled {
		t.Error("should handle /history command")
	}
	if !strings.Contains(result, "2") {
		t.Errorf("result = %q, should contain message count", result)
	}
}

func TestHandleMagicCommand_New(t *testing.T) {
	mem := &mockMemory{
		loadFunc: func(ctx context.Context, chatID string, limit int) ([]Message, error) {
			return []Message{
				{Role: "user", Content: "test"},
			}, nil
		},
		saveLongTermFunc: func(ctx context.Context, chatID string, content string, category string) error {
			return nil
		},
		compactFunc: func(ctx context.Context, chatID string) error {
			return nil
		},
	}

	result, handled, err := HandleMagicCommand(context.Background(), mem, "chat1", "/new", nil)
	if err != nil {
		t.Fatalf("HandleMagicCommand failed: %v", err)
	}
	if !handled {
		t.Error("should handle /new command")
	}
	if !strings.Contains(result, "保存") {
		t.Errorf("result = %q, should contain '保存'", result)
	}
}

func TestHandleMagicCommand_SwitchModel_WithArgs(t *testing.T) {
	mem := &mockMemory{}
	switched := false
	daemonInfo := &DaemonInfo{
		SwitchLLM: func(provider, model string) error {
			switched = true
			return nil
		},
	}

	result, handled, err := HandleMagicCommand(context.Background(), mem, "chat1", "/switch-model openai gpt-4", daemonInfo)
	if err != nil {
		t.Fatalf("HandleMagicCommand failed: %v", err)
	}
	if !handled {
		t.Error("should handle /switch-model command")
	}
	if !switched {
		t.Error("SwitchLLM was not called")
	}
	if !strings.Contains(result, "切换") {
		t.Errorf("result = %q, should contain '切换'", result)
	}
}

func TestHandleMagicCommand_SwitchModel_Error(t *testing.T) {
	mem := &mockMemory{}
	daemonInfo := &DaemonInfo{
		SwitchLLM: func(provider, model string) error {
			return errors.New("switch failed")
		},
	}

	_, _, err := HandleMagicCommand(context.Background(), mem, "chat1", "/switch-model openai gpt-4", daemonInfo)
	if err == nil {
		t.Error("expected error for switch failure")
	}
}

func TestHandleMagicCommand_Daemon_NoArgs(t *testing.T) {
	mem := &mockMemory{}
	daemonInfo := &DaemonInfo{
		Status:  "running",
		Version: "1.0.0",
	}

	result, handled, err := HandleMagicCommand(context.Background(), mem, "chat1", "/daemon", daemonInfo)
	if err != nil {
		t.Fatalf("HandleMagicCommand failed: %v", err)
	}
	if !handled {
		t.Error("should handle /daemon command")
	}
	if !strings.Contains(result, "running") || !strings.Contains(result, "1.0.0") {
		t.Errorf("result = %q, should contain status and version", result)
	}
}

func TestHandleMagicCommand_Daemon_Logs(t *testing.T) {
	mem := &mockMemory{}
	daemonInfo := &DaemonInfo{
		Logs: func(n int) string {
			if n != 10 {
				t.Errorf("expected n=10, got %d", n)
			}
			return "log output"
		},
	}

	result, handled, err := HandleMagicCommand(context.Background(), mem, "chat1", "/daemon logs 10", daemonInfo)
	if err != nil {
		t.Fatalf("HandleMagicCommand failed: %v", err)
	}
	if !handled {
		t.Error("should handle /daemon logs command")
	}
	if result != "log output" {
		t.Errorf("result = %q, want %q", result, "log output")
	}
}

func TestHandleMagicCommand_Daemon_LogsNoFunc(t *testing.T) {
	mem := &mockMemory{}
	daemonInfo := &DaemonInfo{}

	result, handled, err := HandleMagicCommand(context.Background(), mem, "chat1", "/daemon logs", daemonInfo)
	if err != nil {
		t.Fatalf("HandleMagicCommand failed: %v", err)
	}
	if !handled {
		t.Error("should handle /daemon logs command")
	}
	if !strings.Contains(result, "未配置") {
		t.Errorf("result = %q, should contain '未配置'", result)
	}
}

func TestHandleMagicCommand_Daemon_ReloadConfig(t *testing.T) {
	mem := &mockMemory{}
	reloaded := false
	daemonInfo := &DaemonInfo{
		ReloadConfig: func() error {
			reloaded = true
			return nil
		},
	}

	result, handled, err := HandleMagicCommand(context.Background(), mem, "chat1", "/daemon reload-config", daemonInfo)
	if err != nil {
		t.Fatalf("HandleMagicCommand failed: %v", err)
	}
	if !handled {
		t.Error("should handle /daemon reload-config command")
	}
	if !reloaded {
		t.Error("ReloadConfig was not called")
	}
	if !strings.Contains(result, "已重新加载") {
		t.Errorf("result = %q, should contain '已重新加载'", result)
	}
}

func TestHandleMagicCommand_Daemon_Restart(t *testing.T) {
	mem := &mockMemory{}
	restarted := false
	daemonInfo := &DaemonInfo{
		Restart: func() error {
			restarted = true
			return nil
		},
	}

	result, handled, err := HandleMagicCommand(context.Background(), mem, "chat1", "/daemon restart", daemonInfo)
	if err != nil {
		t.Fatalf("HandleMagicCommand failed: %v", err)
	}
	if !handled {
		t.Error("should handle /daemon restart command")
	}
	if !restarted {
		t.Error("Restart was not called")
	}
	if !strings.Contains(result, "重启") {
		t.Errorf("result = %q, should contain '重启'", result)
	}
}

func TestHandleMagicCommand_UnknownCommand(t *testing.T) {
	mem := &mockMemory{}

	result, handled, err := HandleMagicCommand(context.Background(), mem, "chat1", "/unknown", nil)
	if err != nil {
		t.Fatalf("HandleMagicCommand failed: %v", err)
	}
	if handled {
		t.Error("should not handle unknown command")
	}
	if result != "" {
		t.Errorf("result should be empty, got %q", result)
	}
}

func TestHandleMagicCommand_EmptyCommand(t *testing.T) {
	mem := &mockMemory{}

	result, handled, err := HandleMagicCommand(context.Background(), mem, "chat1", "/", nil)
	if err != nil {
		t.Fatalf("HandleMagicCommand failed: %v", err)
	}
	if handled {
		t.Error("should not handle empty command")
	}
	if result != "" {
		t.Errorf("result should be empty, got %q", result)
	}
}
