// Package agent provides the core Agent runtime, ReAct loop, and domain types.
package agent

import (
	"context"
	"testing"

	"github.com/suifei/gopherpaw/internal/agent/cache"
	"github.com/suifei/gopherpaw/internal/config"
)

// mockLLMForExtractor 是用于能力提取器测试的 LLM 模拟实现。
type mockLLMForExtractor struct{}

func (m *mockLLMForExtractor) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// 返回一个简单的模拟响应
	return &ChatResponse{
		Content: "# 能力总结\n\n## 内置工具\n- tool1: 描述1\n- tool2: 描述2\n\n## 技能\n- skill1: 技能描述",
		Usage: Usage{
			PromptTokens:     100,
			CompletionTokens: 50,
			TotalTokens:      150,
		},
	}, nil
}

func (m *mockLLMForExtractor) ChatStream(ctx context.Context, req *ChatRequest) (ChatStream, error) {
	return nil, nil
}

func (m *mockLLMForExtractor) Name() string {
	return "mock"
}

// mockToolForExtractor 是用于能力提取器测试的工具模拟实现。
type mockToolForExtractor struct {
	name        string
	description string
}

func (m *mockToolForExtractor) Name() string {
	return m.name
}

func (m *mockToolForExtractor) Description() string {
	return m.description
}

func (m *mockToolForExtractor) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input": map[string]any{
				"type":        "string",
				"description": "输入参数",
			},
		},
	}
}

func (m *mockToolForExtractor) Execute(ctx context.Context, arguments string) (string, error) {
	return "mock result", nil
}

func TestNewExtractor(t *testing.T) {
	llm := &mockLLMForExtractor{}
	tools := []Tool{
		&mockToolForExtractor{name: "tool1", description: "描述1"},
		&mockToolForExtractor{name: "tool2", description: "描述2"},
	}
	cfg := config.AgentConfig{}

	extractor := NewExtractor(llm, tools, nil, nil, cfg, "/tmp", "/tmp/skills", 24)

	if extractor == nil {
		t.Fatal("NewExtractor returned nil")
	}

	if extractor.llm != llm {
		t.Error("LLM not set correctly")
	}

	if len(extractor.tools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(extractor.tools))
	}
}

func TestExtractCapabilities(t *testing.T) {
	llm := &mockLLMForExtractor{}
	tools := []Tool{
		&mockToolForExtractor{name: "web_search", description: "搜索网络信息"},
		&mockToolForExtractor{name: "read_file", description: "读取文件内容"},
	}
	cfg := config.AgentConfig{}

	extractor := NewExtractor(llm, tools, nil, nil, cfg, "/tmp", "/tmp/skills", 1)

	ctx := context.Background()
	registry, err := extractor.ExtractCapabilities(ctx)

	if err != nil {
		t.Fatalf("ExtractCapabilities failed: %v", err)
	}

	if registry == nil {
		t.Fatal("Registry is nil")
	}

	if len(registry.Capabilities) != 2 {
		t.Errorf("Expected 2 capabilities, got %d", len(registry.Capabilities))
	}

	// 检查能力类型
	for _, cap := range registry.Capabilities {
		if cap.Type != "tool" {
			t.Errorf("Expected type 'tool', got '%s'", cap.Type)
		}
		if !stringSliceContains([]string{"tool:web_search", "tool:read_file"}, cap.ID) {
			t.Errorf("Unexpected capability ID: %s", cap.ID)
		}
	}

	// 检查总结
	if registry.Summary == "" {
		t.Error("Summary is empty")
	}
}

func TestGenerateFallbackSummary(t *testing.T) {
	llm := &mockLLMForExtractor{}
	extractor := NewExtractor(llm, nil, nil, nil, config.AgentConfig{}, "", "", 1)

	caps := []cache.Capability{
		{ID: "tool:search", Type: "tool", Name: "search", Description: "搜索功能"},
		{ID: "skill:docx", Type: "skill", Name: "docx", Description: "文档处理"},
	}

	summary := extractor.generateFallbackSummary(caps)

	if summary == "" {
		t.Fatal("Fallback summary is empty")
	}

	// 检查是否包含关键信息
	if !containsString(summary, "search") {
		t.Error("Summary should contain 'search'")
	}
	if !containsString(summary, "docx") {
		t.Error("Summary should contain 'docx'")
	}
}

func TestCapabilityExtractionWithSkills(t *testing.T) {
	llm := &mockLLMForExtractor{}
	tools := []Tool{
		&mockToolForExtractor{name: "tool1", description: "工具1"},
	}
	cfg := config.AgentConfig{}

	// 使用 nil 技能管理器，应该不会崩溃
	extractor := NewExtractor(llm, tools, nil, nil, cfg, "/tmp", "/tmp/skills", 1)

	ctx := context.Background()
	registry, err := extractor.ExtractCapabilities(ctx)

	if err != nil {
		t.Fatalf("ExtractCapabilities failed: %v", err)
	}

	if len(registry.Capabilities) != 1 {
		t.Errorf("Expected 1 capability (tools only), got %d", len(registry.Capabilities))
	}
}

// Helper functions

func stringSliceContains(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func containsString(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	if s == substr {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
