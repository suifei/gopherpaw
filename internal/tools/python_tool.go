package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/suifei/gopherpaw/internal/agent"
)

// PythonCodeTool executes Python code with optional packages.
type PythonCodeTool struct {
	WorkingDir    string
	PythonPath    string
	AutoInstall   bool
	VenvPath      string
	TempDirPrefix string
}

// Name returns the tool identifier.
func (t *PythonCodeTool) Name() string { return "execute_python_code" }

// Description returns a human-readable description.
func (t *PythonCodeTool) Description() string {
	return "Execute Python code and return the output. Optionally install required packages. Use timeout to limit execution time (default 60 seconds)."
}

// Parameters returns the JSON Schema for tool parameters.
func (t *PythonCodeTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"code": map[string]any{
				"type":        "string",
				"description": "The Python code to execute",
			},
			"packages": map[string]any{
				"type":        "array",
				"items":       map[string]string{"type": "string"},
				"description": "Optional list of Python packages to install before execution",
			},
			"timeout": map[string]any{
				"type":        "integer",
				"description": "Maximum seconds to run (default 60)",
			},
		},
		"required": []string{"code"},
	}
}

type pythonCodeArgs struct {
	Code     string   `json:"code"`
	Packages []string `json:"packages"`
	Timeout  int      `json:"timeout"`
}

// PythonResult represents the detailed result of Python code execution.
type PythonResult struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	TimedOut bool   `json:"timed_out,omitempty"`
	Error    string `json:"error,omitempty"`
}

// Execute runs the tool and returns combined output for LLM compatibility.
func (t *PythonCodeTool) Execute(ctx context.Context, arguments string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	var args pythonCodeArgs
	if arguments != "" {
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
	}
	code := strings.TrimSpace(args.Code)
	if code == "" {
		return "Error: No code provided.", nil
	}
	timeout := args.Timeout
	if timeout <= 0 {
		timeout = 60
	}

	result := t.executePython(ctx, code, args.Packages, timeout)
	return formatPythonResult(result), nil
}

// ExecuteRich runs the tool and returns structured PythonResult.
func (t *PythonCodeTool) ExecuteRich(ctx context.Context, arguments string) (*PythonResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var args pythonCodeArgs
	if arguments != "" {
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return nil, fmt.Errorf("invalid arguments: %w", err)
		}
	}
	code := strings.TrimSpace(args.Code)
	if code == "" {
		return &PythonResult{Error: "No code provided."}, nil
	}
	timeout := args.Timeout
	if timeout <= 0 {
		timeout = 60
	}

	return t.executePython(ctx, code, args.Packages, timeout), nil
}

func (t *PythonCodeTool) executePython(ctx context.Context, code string, packages []string, timeout int) *PythonResult {
	// Determine Python interpreter
	pythonExe := t.resolvePythonPath()

	// Install packages if needed
	if len(packages) > 0 && t.AutoInstall {
		installCmd := fmt.Sprintf("%s -m pip install --quiet %s", pythonExe, strings.Join(packages, " "))
		installResult := t.runCommand(ctx, installCmd, t.getWorkingDir(), 300) // 5 minutes for install
		if installResult.ExitCode != 0 {
			return &PythonResult{
				ExitCode: installResult.ExitCode,
				Stderr:   fmt.Sprintf("Failed to install packages: %s", installResult.Stderr),
				Error:    "Package installation failed",
			}
		}
	}

	// Create temporary Python file
	tmpFile, err := os.CreateTemp("", t.TempDirPrefix+"*.py")
	if err != nil {
		return &PythonResult{Error: fmt.Sprintf("Failed to create temp file: %v", err)}
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(code); err != nil {
		return &PythonResult{Error: fmt.Sprintf("Failed to write code: %v", err)}
	}
	if err := tmpFile.Close(); err != nil {
		return &PythonResult{Error: fmt.Sprintf("Failed to close file: %v", err)}
	}

	// Execute Python file
	cmd := fmt.Sprintf("%s %s", pythonExe, tmpFile.Name())
	return t.runCommand(ctx, cmd, t.getWorkingDir(), timeout)
}

func (t *PythonCodeTool) resolvePythonPath() string {
	if t.PythonPath != "" {
		return t.PythonPath
	}
	if t.VenvPath != "" {
		venvPython := filepath.Join(t.VenvPath, "bin", "python")
		if _, err := os.Stat(venvPython); err == nil {
			return venvPython
		}
		venvPython = filepath.Join(t.VenvPath, "Scripts", "python.exe")
		if _, err := os.Stat(venvPython); err == nil {
			return venvPython
		}
	}
	return "python3"
}

func (t *PythonCodeTool) getWorkingDir() string {
	if t.WorkingDir != "" {
		return t.WorkingDir
	}
	wd, _ := os.Getwd()
	return wd
}

func (t *PythonCodeTool) runCommand(ctx context.Context, command string, cwd string, timeout int) *PythonResult {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	shell := "/bin/sh"
	if strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") {
		shell = "cmd"
		cmd = exec.CommandContext(ctx, shell, "/c", command)
	} else {
		cmd = exec.CommandContext(ctx, shell, "-c", command)
	}

	cmd.Dir = cwd
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := &PythonResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if ctx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
		result.Error = fmt.Sprintf("Command timed out after %d seconds", timeout)
	} else if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
			result.Error = err.Error()
		}
	}

	return result
}

func formatPythonResult(r *PythonResult) string {
	var parts []string
	if r.Stdout != "" {
		parts = append(parts, fmt.Sprintf("Output:\n%s", r.Stdout))
	}
	if r.Stderr != "" {
		parts = append(parts, fmt.Sprintf("Error:\n%s", r.Stderr))
	}
	if r.TimedOut {
		parts = append(parts, fmt.Sprintf("Timed out after execution limit"))
	}
	if r.Error != "" {
		parts = append(parts, fmt.Sprintf("Error: %s", r.Error))
	}
	if r.ExitCode != 0 {
		parts = append(parts, fmt.Sprintf("Exit code: %d", r.ExitCode))
	}
	if len(parts) == 0 {
		return "Command executed successfully with no output."
	}
	return strings.Join(parts, "\n")
}

// Ensure PythonCodeTool implements agent.Tool interface
var _ agent.Tool = (*PythonCodeTool)(nil)
