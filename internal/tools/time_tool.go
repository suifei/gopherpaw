package tools

import (
	"context"
	"fmt"
	"time"

	"github.com/suifei/gopherpaw/internal/agent"
)

// TimeTool returns the current system time with timezone info.
type TimeTool struct{}

// Name returns the tool identifier.
func (t *TimeTool) Name() string { return "get_current_time" }

// Description returns a human-readable description.
func (t *TimeTool) Description() string {
	return "Get the current system time with timezone information. Useful for time-sensitive tasks such as scheduling."
}

// Parameters returns the JSON Schema for tool parameters (empty for this tool).
func (t *TimeTool) Parameters() any {
	return map[string]any{
		"type":       "object",
		"properties": map[string]any{},
	}
}

// Execute runs the tool.
func (t *TimeTool) Execute(ctx context.Context, arguments string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	_ = arguments // no args
	now := time.Now()
	zone, _ := now.Zone()
	offset := now.Format("-0700")
	return fmt.Sprintf("%s %s (UTC%s)", now.Format("2006-01-02 15:04:05"), zone, offset), nil
}

// Ensure TimeTool implements agent.Tool.
var _ agent.Tool = (*TimeTool)(nil)
