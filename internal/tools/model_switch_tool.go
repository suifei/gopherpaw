package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/suifei/gopherpaw/internal/agent"
)

// ModelSwitchTool allows the agent to switch the active model slot at runtime.
// Requires a ModelSwitcher to be injected via context (agent.WithModelSwitcher).
type ModelSwitchTool struct{}

func (t *ModelSwitchTool) Name() string { return "switch_model" }

func (t *ModelSwitchTool) Description() string {
	return "Switch the active LLM model slot. Use this when you need a different model capability (e.g. 'vision' for image understanding, 'code' for coding tasks). Call with slot name or capability name. Use action 'list' to see available slots."
}

func (t *ModelSwitchTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"enum":        []string{"switch", "list", "status"},
				"description": "Action to perform: 'switch' to change model, 'list' to show available slots, 'status' to show current slot",
			},
			"slot": map[string]any{
				"type":        "string",
				"description": "Target slot name (e.g. 'default', 'vision', 'code') or capability name (e.g. 'vision'). Required for 'switch' action.",
			},
		},
		"required": []string{"action"},
	}
}

type modelSwitchArgs struct {
	Action string `json:"action"`
	Slot   string `json:"slot"`
}

func (t *ModelSwitchTool) Execute(ctx context.Context, arguments string) (string, error) {
	var args modelSwitchArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("parse arguments: %w", err)
	}

	ms := agent.GetModelSwitcher(ctx)
	if ms == nil {
		return "Model routing is not configured. Only a single model is available.", nil
	}

	switch args.Action {
	case "list":
		names := ms.SlotNames()
		active := ms.ActiveSlot()
		var sb strings.Builder
		sb.WriteString("Available model slots:\n")
		for _, name := range names {
			marker := "  "
			if name == active {
				marker = "* "
			}
			sb.WriteString(fmt.Sprintf("%s%s\n", marker, name))
		}
		return sb.String(), nil

	case "status":
		return fmt.Sprintf("Current active slot: %s", ms.ActiveSlot()), nil

	case "switch":
		if args.Slot == "" {
			return "Error: 'slot' parameter is required for switch action.", nil
		}
		if err := ms.Switch(args.Slot); err != nil {
			return fmt.Sprintf("Failed to switch model: %s", err.Error()), nil
		}
		return fmt.Sprintf("Switched to model slot: %s", args.Slot), nil

	default:
		return fmt.Sprintf("Unknown action %q. Use 'switch', 'list', or 'status'.", args.Action), nil
	}
}
