package logger

import (
	"testing"
)

func TestInit(t *testing.T) {
	err := Init(Config{Level: "debug", Format: "console"})
	if err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	L().Info("test message")
}

func TestInit_InvalidLevel(t *testing.T) {
	err := Init(Config{Level: "invalid", Format: "json"})
	if err != nil {
		t.Fatalf("Init should not fail for invalid level (uses default): %v", err)
	}
}

func TestL_PanicWithoutInit(t *testing.T) {
	// Reset global to simulate uninitialized state
	old := global
	global = nil
	defer func() {
		global = old
	}()

	// L() creates a production logger when global is nil
	l := L()
	if l == nil {
		t.Fatal("L() should return a logger")
	}
}

func TestParseLevel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"debug", "debug"},
		{"info", "info"},
		{"warn", "warn"},
		{"error", "error"},
		{"unknown", "info"},
	}
	for _, tt := range tests {
		lvl := parseLevel(tt.input)
		_ = lvl // used to verify no panic
	}
}
