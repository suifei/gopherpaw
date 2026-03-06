package agent

import (
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

func TestRegisterTool_NoDuplicate(t *testing.T) {
	toolMap := make(map[string]Tool)
	tool := &mockTool{name: "test_tool_unique"}
	
	err := registerTool(toolMap, tool, NamesakeSkip)
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if toolMap["test_tool_unique"] != tool {
		t.Error("tool not registered")
	}
}

func TestRegisterTool_Override(t *testing.T) {
	toolMap := make(map[string]Tool)
	tool1 := &mockTool{name: "test_override"}
	tool2 := &mockTool{name: "test_override"}
	
	_ = registerTool(toolMap, tool1, NamesakeOverride)
	err := registerTool(toolMap, tool2, NamesakeOverride)
	
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if toolMap["test_override"] != tool2 {
		t.Error("tool not overridden")
	}
}

func TestRegisterTool_Skip(t *testing.T) {
	toolMap := make(map[string]Tool)
	tool1 := &mockTool{name: "test_skip"}
	tool2 := &mockTool{name: "test_skip"}
	
	_ = registerTool(toolMap, tool1, NamesakeSkip)
	err := registerTool(toolMap, tool2, NamesakeSkip)
	
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
	if toolMap["test_skip"] != tool1 {
		t.Error("original tool should be kept")
	}
}

func TestRegisterTool_Raise(t *testing.T) {
	toolMap := make(map[string]Tool)
	tool1 := &mockTool{name: "test_raise"}
	tool2 := &mockTool{name: "test_raise"}
	
	_ = registerTool(toolMap, tool1, NamesakeRaise)
	err := registerTool(toolMap, tool2, NamesakeRaise)
	
	if err == nil {
		t.Error("expected error on duplicate with raise strategy")
	}
}

func TestNewReactWithPrompt_NamesakeStrategy(t *testing.T) {
	tool1 := &mockTool{name: "test_namesake"}
	tool2 := &mockTool{name: "test_namesake"}
	tools := []Tool{tool1, tool2}
	
	cfg := config.AgentConfig{
		Running: config.AgentRunningConfig{
			MaxTurns:         5,
			NamesakeStrategy: "skip",
		},
	}
	
	agent := NewReactWithPrompt(nil, nil, tools, cfg, nil, "")
	
	// With skip strategy, should have only 1 tool
	if len(agent.tools) != 1 {
		t.Errorf("expected 1 tool with skip strategy, got %d", len(agent.tools))
	}
}
