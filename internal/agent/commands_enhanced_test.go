package agent

import (
	"context"
	"strings"
	"testing"
)

func TestHandleMagicCommand_AwaitSummary_Empty(t *testing.T) {
	mem := &mockMemory{}
	result, handled, err := HandleMagicCommand(context.Background(), mem, "chat1", "/await_summary", nil)
	if err != nil {
		t.Fatalf("HandleMagicCommand: %v", err)
	}
	if !handled {
		t.Error("expected handled")
	}
	if !strings.Contains(result, "无待处理") && !strings.Contains(result, "正在生成") {
		t.Errorf("expected empty summary message, got %q", result)
	}
}

func TestHandleMagicCommand_AwaitSummary_WithContent(t *testing.T) {
	mem := &mockMemory{}
	mem.compactFunc = nil
	origGetSummary := mem.GetCompactSummary
	_ = origGetSummary

	summaryMem := &mockMemoryWithSummary{summary: "This is a summary of the conversation."}
	result, handled, err := HandleMagicCommand(context.Background(), summaryMem, "chat1", "/await_summary", nil)
	if err != nil {
		t.Fatalf("HandleMagicCommand: %v", err)
	}
	if !handled {
		t.Error("expected handled")
	}
	if !strings.Contains(result, "This is a summary") {
		t.Errorf("expected summary in result, got %q", result)
	}
}

func TestHandleMagicCommand_SwitchModel(t *testing.T) {
	called := false
	var gotProvider, gotModel string
	info := &DaemonInfo{
		SwitchLLM: func(provider, model string) error {
			called = true
			gotProvider = provider
			gotModel = model
			return nil
		},
	}
	result, handled, err := HandleMagicCommand(context.Background(), nil, "chat1", "/switch-model openai gpt-4", info)
	if err != nil {
		t.Fatalf("HandleMagicCommand: %v", err)
	}
	if !handled {
		t.Error("expected handled")
	}
	if !called {
		t.Error("SwitchLLM was not called")
	}
	if gotProvider != "openai" || gotModel != "gpt-4" {
		t.Errorf("expected openai/gpt-4, got %s/%s", gotProvider, gotModel)
	}
	if !strings.Contains(result, "openai") {
		t.Errorf("expected provider in result, got %q", result)
	}
}

func TestHandleMagicCommand_SwitchModel_NotEnoughArgs(t *testing.T) {
	info := &DaemonInfo{}
	result, handled, err := HandleMagicCommand(context.Background(), nil, "chat1", "/switch-model openai", info)
	if err != nil {
		t.Fatalf("HandleMagicCommand: %v", err)
	}
	if !handled {
		t.Error("expected handled")
	}
	if !strings.Contains(result, "用法") {
		t.Errorf("expected usage hint, got %q", result)
	}
}

// mockMemoryWithSummary extends mockMemory with a non-empty compact summary.
type mockMemoryWithSummary struct {
	mockMemory
	summary string
}

func (m *mockMemoryWithSummary) GetCompactSummary(ctx context.Context, chatID string) (string, error) {
	return m.summary, nil
}
