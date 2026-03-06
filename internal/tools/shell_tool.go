package tools

import (
	"bytes"
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

// ShellTool executes shell commands with separate stdout/stderr capture.
type ShellTool struct {
	WorkingDir string
}

// Name returns the tool identifier.
func (t *ShellTool) Name() string { return "execute_shell_command" }

// Description returns a human-readable description.
func (t *ShellTool) Description() string {
	return "Execute a shell command and return the exit code, stdout and stderr separately. Use timeout to limit execution time (default 60 seconds)."
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

// ShellResult represents the detailed result of a shell command execution.
type ShellResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	TimedOut bool   `json:"timed_out,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Execute runs the tool and returns combined output for LLM compatibility.
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

	result := t.executeCommand(ctx, cmd, absCwd, timeout)
	return formatShellResult(result), nil
}

// ExecuteRich runs the tool and returns structured ShellResult.
// This implements agent.RichExecutor for enhanced output.
func (t *ShellTool) ExecuteRich(ctx context.Context, arguments string) (any, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var args shellArgs
	if arguments != "" {
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
	}
	cmd := strings.TrimSpace(args.Command)
	if cmd == "" {
		return &ShellResult{ExitCode: -1, Error: "No command provided"}, nil
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

	return t.executeCommand(ctx, cmd, absCwd, timeout), nil
}

// executeCommand runs the command with separate stdout/stderr capture.
func (t *ShellTool) executeCommand(ctx context.Context, cmd, cwd string, timeout int) *ShellResult {
	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	var execCmd *exec.Cmd
	if runtime.GOOS == "windows" {
		execCmd = exec.CommandContext(execCtx, "cmd", "/C", cmd)
	} else {
		execCmd = exec.CommandContext(execCtx, "sh", "-c", cmd)
	}
	execCmd.Dir = cwd

	var stdout, stderr bytes.Buffer
	execCmd.Stdout = &stdout
	execCmd.Stderr = &stderr

	err := execCmd.Run()

	result := &ShellResult{
		Stdout: strings.TrimSpace(stdout.String()),
		Stderr: strings.TrimSpace(stderr.String()),
	}

	if execCtx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
		result.ExitCode = -1
		result.Error = fmt.Sprintf("Command timed out after %d seconds", timeout)
		return result
	}

	if err != nil {
		if execCmd.ProcessState != nil {
			result.ExitCode = execCmd.ProcessState.ExitCode()
		} else {
			result.ExitCode = -1
			result.Error = err.Error()
		}
	} else {
		result.ExitCode = 0
	}

	return result
}

// formatShellResult formats the result for LLM consumption.
func formatShellResult(r *ShellResult) string {
	var sb strings.Builder

	// Handle timeout
	if r.TimedOut {
		sb.WriteString(fmt.Sprintf("Error: %s\n", r.Error))
		if r.Stdout != "" {
			sb.WriteString(fmt.Sprintf("\n[stdout (partial)]:\n%s\n", r.Stdout))
		}
		if r.Stderr != "" {
			sb.WriteString(fmt.Sprintf("\n[stderr (partial)]:\n%s\n", r.Stderr))
		}
		return sb.String()
	}

	// Handle execution error
	if r.Error != "" && r.ExitCode == -1 {
		sb.WriteString(fmt.Sprintf("Error: %s\n", r.Error))
		return sb.String()
	}

	// Normal execution
	sb.WriteString(fmt.Sprintf("Exit code: %d\n", r.ExitCode))

	if r.Stdout != "" {
		sb.WriteString(fmt.Sprintf("\n[stdout]:\n%s\n", r.Stdout))
	}

	if r.Stderr != "" {
		sb.WriteString(fmt.Sprintf("\n[stderr]:\n%s\n", r.Stderr))
	}

	if r.Stdout == "" && r.Stderr == "" {
		if r.ExitCode == 0 {
			sb.WriteString("Command executed successfully (no output).\n")
		}
	}

	return strings.TrimSpace(sb.String())
}

// Ensure ShellTool implements agent.Tool.
var _ agent.Tool = (*ShellTool)(nil)
