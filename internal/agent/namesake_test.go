package agent

import (
	"strings"
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

func TestRegisterTool_AllStrategies(t *testing.T) {
	tests := []struct {
		name      string
		strategy  NamesakeStrategy
		setup     func(map[string]Tool)
		wantErr   bool
		errMsg    string
		checkFunc func(t *testing.T, toolMap map[string]Tool)
	}{
		{
			name:     "override strategy replaces existing",
			strategy: NamesakeOverride,
			setup: func(toolMap map[string]Tool) {
				toolMap["test"] = &mockTool{name: "test", description: "old"}
			},
			wantErr: false,
			checkFunc: func(t *testing.T, toolMap map[string]Tool) {
				if toolMap["test"].Description() != "new" {
					t.Error("tool should be overridden")
				}
			},
		},
		{
			name:     "skip strategy keeps existing",
			strategy: NamesakeSkip,
			setup: func(toolMap map[string]Tool) {
				toolMap["test"] = &mockTool{name: "test", description: "old"}
			},
			wantErr: false,
			checkFunc: func(t *testing.T, toolMap map[string]Tool) {
				if toolMap["test"].Description() != "old" {
					t.Error("tool should not be overridden with skip")
				}
			},
		},
		{
			name:     "raise strategy returns error",
			strategy: NamesakeRaise,
			setup: func(toolMap map[string]Tool) {
				toolMap["test"] = &mockTool{name: "test"}
			},
			wantErr: true,
			errMsg:  "duplicate tool name",
		},
		{
			name:     "rename strategy returns error",
			strategy: NamesakeRename,
			setup: func(toolMap map[string]Tool) {
				toolMap["test"] = &mockTool{name: "test"}
			},
			wantErr: true,
			errMsg:  "rename strategy not fully implemented",
		},
		{
			name:     "unknown strategy returns error",
			strategy: NamesakeStrategy("unknown"),
			setup: func(toolMap map[string]Tool) {
				toolMap["test"] = &mockTool{name: "test"}
			},
			wantErr: true,
			errMsg:  "unknown namesake strategy",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			toolMap := make(map[string]Tool)
			if tt.setup != nil {
				tt.setup(toolMap)
			}

			tool := &mockTool{name: "test", description: "new"}
			err := registerTool(toolMap, tool, tt.strategy)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}

			if tt.checkFunc != nil {
				tt.checkFunc(t, toolMap)
			}
		})
	}
}
