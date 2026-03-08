package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

func TestMemoryCompactionHook_BelowThreshold(t *testing.T) {
	hook := MemoryCompactionHook(100000, 10)
	agent := NewReact(&mockLLM{}, &mockMemory{}, nil, config.AgentConfig{Running: config.AgentRunningConfig{MaxTurns: 5}})
	msgs := []Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hello"},
	}
	result, err := hook(context.Background(), agent, "chat1", msgs)
	if err != nil {
		t.Fatalf("hook: %v", err)
	}
	if len(result) != len(msgs) {
		t.Errorf("expected %d messages unchanged, got %d", len(msgs), len(result))
	}
}

func TestMemoryCompactionHook_FewMessages(t *testing.T) {
	hook := MemoryCompactionHook(10, 10)
	agent := NewReact(&mockLLM{}, &mockMemory{}, nil, config.AgentConfig{Running: config.AgentRunningConfig{MaxTurns: 5}})
	msgs := []Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "hi"},
	}
	result, err := hook(context.Background(), agent, "chat1", msgs)
	if err != nil {
		t.Fatalf("hook: %v", err)
	}
	if len(result) != len(msgs) {
		t.Errorf("expected messages unchanged when few messages")
	}
}

func TestMemoryCompactionHook_TriggersCompact(t *testing.T) {
	compacted := false
	mem := &mockMemory{
		compactFunc: func(ctx context.Context, chatID string) error {
			compacted = true
			return nil
		},
		loadFunc: func(ctx context.Context, chatID string, limit int) ([]Message, error) {
			return []Message{{Role: "user", Content: "compacted"}}, nil
		},
	}
	hook := MemoryCompactionHook(1, 2)
	agent := NewReact(&mockLLM{}, mem, nil, config.AgentConfig{Running: config.AgentRunningConfig{MaxTurns: 5}})

	msgs := make([]Message, 0, 15)
	msgs = append(msgs, Message{Role: "system", Content: "sys"})
	for i := 0; i < 12; i++ {
		msgs = append(msgs, Message{Role: "user", Content: "message with enough content to exceed threshold"})
	}

	result, err := hook(context.Background(), agent, "chat1", msgs)
	if err != nil {
		t.Fatalf("hook: %v", err)
	}
	if !compacted {
		t.Error("expected Compact to be called")
	}
	if len(result) == len(msgs) {
		t.Error("expected messages to be replaced after compaction")
	}
}

func TestBootstrapHook_NoBootstrapFile(t *testing.T) {
	dir := t.TempDir()
	hook := BootstrapHook(dir, "zh")
	agent := NewReact(&mockLLM{}, &mockMemory{}, nil, config.AgentConfig{Running: config.AgentRunningConfig{MaxTurns: 5}})
	msgs := []Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "hi"},
	}
	result, err := hook(context.Background(), agent, "chat1", msgs)
	if err != nil {
		t.Fatalf("hook: %v", err)
	}
	if result[1].Content != "hi" {
		t.Errorf("expected unchanged message, got %q", result[1].Content)
	}
}

func TestBootstrapHook_WithBootstrapFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "BOOTSTRAP.md"), []byte("bootstrap content"), 0644); err != nil {
		t.Fatal(err)
	}
	hook := BootstrapHook(dir, "zh")
	agent := NewReact(&mockLLM{}, &mockMemory{}, nil, config.AgentConfig{Running: config.AgentRunningConfig{MaxTurns: 5}})
	msgs := []Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "hi"},
	}
	result, err := hook(context.Background(), agent, "chat1", msgs)
	if err != nil {
		t.Fatalf("hook: %v", err)
	}
	if result[1].Content == "hi" {
		t.Error("expected user message to be prepended with bootstrap guidance")
	}
	if _, err := os.Stat(filepath.Join(dir, ".bootstrap_completed")); os.IsNotExist(err) {
		t.Error("expected .bootstrap_completed to be created")
	}
}

func TestBootstrapHook_AlreadyCompleted(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "BOOTSTRAP.md"), []byte("bootstrap"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".bootstrap_completed"), []byte("done"), 0644); err != nil {
		t.Fatal(err)
	}
	hook := BootstrapHook(dir, "zh")
	agent := NewReact(&mockLLM{}, &mockMemory{}, nil, config.AgentConfig{Running: config.AgentRunningConfig{MaxTurns: 5}})
	msgs := []Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "hi"},
	}
	result, err := hook(context.Background(), agent, "chat1", msgs)
	if err != nil {
		t.Fatalf("hook: %v", err)
	}
	if result[1].Content != "hi" {
		t.Error("expected message unchanged when already completed")
	}
}

func TestEstimateMessageTokens(t *testing.T) {
	m := Message{Content: "hello world", ToolCalls: []ToolCall{{Arguments: "args"}}}
	tokens := EstimateMessageTokens(m)
	if tokens <= 0 {
		t.Errorf("expected positive token count, got %d", tokens)
	}
}

func TestBuildBootstrapGuidance_Zh(t *testing.T) {
	guidance := BuildBootstrapGuidance("zh")

	// 验证关键元素存在
	if guidance == "" {
		t.Error("expected non-empty zh guidance")
	}
	// 检查引导模式激活标识
	if !contains(guidance, "引导模式已激活") {
		t.Error("missing bootstrap mode activation")
	}
	// 检查 BOOTSTRAP.md 引用
	if !contains(guidance, "BOOTSTRAP.md") {
		t.Error("missing BOOTSTRAP.md reference")
	}
	// 检查 PROFILE.md 引用
	if !contains(guidance, "PROFILE.md") {
		t.Error("missing PROFILE.md reference")
	}
	// 检查 MEMORY.md 引用
	if !contains(guidance, "MEMORY.md") {
		t.Error("missing MEMORY.md reference")
	}
	// 检查跳过机制说明
	if !contains(guidance, "跳过") {
		t.Error("missing skip instruction")
	}
	// 检查原始消息占位符
	if !contains(guidance, "用户的原始消息") {
		t.Error("missing original message placeholder")
	}
}

func TestBuildBootstrapGuidance_En(t *testing.T) {
	guidance := BuildBootstrapGuidance("en")

	if guidance == "" {
		t.Error("expected non-empty en guidance")
	}
	// 检查引导模式激活标识
	if !contains(guidance, "BOOTSTRAP MODE ACTIVATED") {
		t.Error("missing bootstrap mode activation")
	}
	// 检查跳过机制说明
	if !contains(guidance, "skip") {
		t.Error("missing skip instruction")
	}
	// 检查原始消息占位符
	if !contains(guidance, "Original user message") {
		t.Error("missing original message placeholder")
	}
}

func TestBuildBootstrapGuidance_Different(t *testing.T) {
	zh := BuildBootstrapGuidance("zh")
	en := BuildBootstrapGuidance("en")
	if zh == en {
		t.Error("expected different guidance for zh and en")
	}
}

// contains 是简单的字符串包含检查
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstr(s, substr))
}

func findSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestIsFirstUserInteraction(t *testing.T) {
	if !isFirstUserInteraction([]Message{{Role: "system"}, {Role: "user", Content: "hi"}}) {
		t.Error("expected true for single user message")
	}
	if isFirstUserInteraction([]Message{{Role: "user", Content: "a"}, {Role: "user", Content: "b"}}) {
		t.Error("expected false for multiple user messages")
	}
	if isFirstUserInteraction([]Message{{Role: "system"}}) {
		t.Error("expected false for no user messages")
	}
}
