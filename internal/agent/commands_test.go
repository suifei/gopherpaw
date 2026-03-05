package agent

import (
	"context"
	"strings"
	"testing"
)

func TestHandleMagicCommand_NotMagic(t *testing.T) {
	mem := &mockMemory{}
	result, handled, err := HandleMagicCommand(context.Background(), mem, "chat1", "hello world", nil)
	if err != nil {
		t.Fatalf("HandleMagicCommand: %v", err)
	}
	if handled {
		t.Error("expected not handled")
	}
	if result != "" {
		t.Errorf("expected empty result, got %q", result)
	}
}

func TestHandleMagicCommand_Compact(t *testing.T) {
	mem := &mockMemory{}
	result, handled, err := HandleMagicCommand(context.Background(), mem, "chat1", "/compact", nil)
	if err != nil {
		t.Fatalf("HandleMagicCommand: %v", err)
	}
	if !handled {
		t.Error("expected handled")
	}
	if !strings.Contains(result, "压缩") {
		t.Errorf("expected 压缩 in result, got %q", result)
	}
}

func TestHandleMagicCommand_DaemonVersion(t *testing.T) {
	info := &DaemonInfo{Version: "0.1.0", Status: "running"}
	result, handled, err := HandleMagicCommand(context.Background(), nil, "chat1", "/daemon version", info)
	if err != nil {
		t.Fatalf("HandleMagicCommand: %v", err)
	}
	if !handled {
		t.Error("expected handled")
	}
	if result != "0.1.0" {
		t.Errorf("expected version 0.1.0, got %q", result)
	}
}

func TestHandleMagicCommand_DaemonReloadConfig(t *testing.T) {
	called := false
	info := &DaemonInfo{
		ReloadConfig: func() error { called = true; return nil },
	}
	result, handled, err := HandleMagicCommand(context.Background(), nil, "chat1", "/daemon reload-config", info)
	if err != nil {
		t.Fatalf("HandleMagicCommand: %v", err)
	}
	if !handled {
		t.Error("expected handled")
	}
	if !strings.Contains(result, "配置已重新加载") {
		t.Errorf("expected 配置已重新加载, got %q", result)
	}
	if !called {
		t.Error("ReloadConfig was not called")
	}
}

func TestHandleMagicCommand_DaemonRestart(t *testing.T) {
	called := false
	info := &DaemonInfo{
		Restart: func() error { called = true; return nil },
	}
	result, handled, err := HandleMagicCommand(context.Background(), nil, "chat1", "/daemon restart", info)
	if err != nil {
		t.Fatalf("HandleMagicCommand: %v", err)
	}
	if !handled {
		t.Error("expected handled")
	}
	if !strings.Contains(result, "重启") {
		t.Errorf("expected 重启 in result, got %q", result)
	}
	if !called {
		t.Error("Restart was not called")
	}
}
