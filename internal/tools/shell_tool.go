package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/suifei/gopherpaw/internal/agent"
)

// ShellTool executes shell commands.
type ShellTool struct {
	WorkingDir string
}

// Name returns the tool identifier.
func (t *ShellTool) Name() string { return "execute_shell_command" }

// Description returns a human-readable description.
func (t *ShellTool) Description() string {
	return "Execute a shell command and return the return code, stdout and stderr. Use timeout to limit execution time (default 60 seconds)."
}

// Parameters returns the JSON Schema for tool parameters.
func (t *ShellTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{
				"type":        "string",
				"description": "The shell command to execute",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Maximum seconds to run (default 60)",
			},
			"cwd": map[string]any{
				"type":        "string",
				"description": "Working directory (default: current)",
			},
		},
		"required": []string{"command"},
	}
}

type shellArgs struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
	CWD     string `json:"cwd"`
}

// Execute runs the tool.
func (t *ShellTool) Execute(ctx context.Context, arguments string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var args shellArgs
	if arguments != "" {
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
	}
	cmd := strings.TrimSpace(args.Command)
	if cmd == "" {
		return "Error: No command provided.", nil
	}
	timeout := args.Timeout
	if timeout <= 0 {
		timeout = 60
	}
	cwd := args.CWD
	if cwd == "" && t.WorkingDir != "" {
		cwd = t.WorkingDir
	}
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	absCwd, _ := filepath.Abs(cwd)

	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	var execCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		execCmd = exec.CommandContext(execCtx, "cmd", "/C", cmd)
	} else {
		execCmd = exec.CommandContext(execCtx, "sh", "-c", cmd)
	}
	execCmd.Dir = absCwd
	out, err := execCmd.CombinedOutput()
	result := strings.TrimSpace(string(out))
	if execCtx.Err() == context.DeadlineExceeded {
		return fmt.Sprintf("Error: Command timed out after %d seconds.", timeout), nil
	}
	if err != nil {
		if execCmd.ProcessState != nil {
			return fmt.Sprintf("Exit code: %v\n%s", execCmd.ProcessState.ExitCode(), result), nil
		}
		return fmt.Sprintf("Error: %v\n%s", err, result), nil
	}
	if result == "" {
		return "Command executed successfully (no output).", nil
	}
	return result, nil
}

// Ensure ShellTool implements agent.Tool.
var _ agent.Tool = (*ShellTool)(nil)
