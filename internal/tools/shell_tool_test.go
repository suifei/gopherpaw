package tools

import (
	"context"
	"encoding/json"
	"runtime"
	"strings"
	"testing"
)

func TestShellTool_Execute_Basic(t *testing.T) {
	tool := &ShellTool{}
	ctx := context.Background()

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "echo hello"
	} else {
		cmd = "echo hello"
	}

	args, _ := json.Marshal(map[string]any{"command": cmd})
	result, err := tool.Execute(ctx, string(args))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "Exit code: 0") {
		t.Errorf("expected exit code 0 in result: %s", result)
	}
	if !strings.Contains(result, "hello") {
		t.Errorf("expected 'hello' in result: %s", result)
	}
}

func TestShellTool_Execute_StdoutStderr(t *testing.T) {
	tool := &ShellTool{}
	ctx := context.Background()

	var cmd string
	if runtime.GOOS == "windows" {
		// Windows cmd doesn't have easy stderr redirect, so just test stdout
		cmd = "echo stdout_output"
	} else {
		cmd = "echo stdout_output && echo stderr_output >&2"
	}

	args, _ := json.Marshal(map[string]any{"command": cmd})
	result, err := tool.Execute(ctx, string(args))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "stdout_output") {
		t.Errorf("expected stdout_output in result: %s", result)
	}
	if runtime.GOOS != "windows" {
		if !strings.Contains(result, "[stderr]") {
			t.Errorf("expected [stderr] section in result: %s", result)
		}
		if !strings.Contains(result, "stderr_output") {
			t.Errorf("expected stderr_output in result: %s", result)
		}
	}
}

func TestShellTool_Execute_ExitCode(t *testing.T) {
	tool := &ShellTool{}
	ctx := context.Background()

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "exit 42"
	} else {
		cmd = "exit 42"
	}

	args, _ := json.Marshal(map[string]any{"command": cmd})
	result, err := tool.Execute(ctx, string(args))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "Exit code: 42") {
		t.Errorf("expected exit code 42 in result: %s", result)
	}
}

func TestShellTool_Execute_EmptyCommand(t *testing.T) {
	tool := &ShellTool{}
	ctx := context.Background()

	args, _ := json.Marshal(map[string]any{"command": ""})
	result, err := tool.Execute(ctx, string(args))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "Error") {
		t.Errorf("expected error for empty command: %s", result)
	}
}

func TestShellTool_Execute_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test in short mode")
	}

	tool := &ShellTool{}
	ctx := context.Background()

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "ping -n 10 127.0.0.1"
	} else {
		cmd = "sleep 10"
	}

	args, _ := json.Marshal(map[string]any{"command": cmd, "timeout": 1})
	result, err := tool.Execute(ctx, string(args))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(result, "timed out") {
		t.Errorf("expected timeout message in result: %s", result)
	}
}

func TestShellTool_ExecuteRich(t *testing.T) {
	tool := &ShellTool{}
	ctx := context.Background()

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "echo hello"
	} else {
		cmd = "echo hello"
	}

	args, _ := json.Marshal(map[string]any{"command": cmd})
	result, err := tool.ExecuteRich(ctx, string(args))
	if err != nil {
		t.Fatal(err)
	}

	shellResult, ok := result.(*ShellResult)
	if !ok {
		t.Fatalf("expected *ShellResult, got %T", result)
	}

	if shellResult.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", shellResult.ExitCode)
	}
	if !strings.Contains(shellResult.Stdout, "hello") {
		t.Errorf("expected 'hello' in stdout: %s", shellResult.Stdout)
	}
	if shellResult.TimedOut {
		t.Error("expected not timed out")
	}
}

func TestShellTool_ExecuteRich_Stderr(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping stderr test on Windows")
	}

	tool := &ShellTool{}
	ctx := context.Background()

	args, _ := json.Marshal(map[string]any{"command": "echo error >&2"})
	result, err := tool.ExecuteRich(ctx, string(args))
	if err != nil {
		t.Fatal(err)
	}

	shellResult, ok := result.(*ShellResult)
	if !ok {
		t.Fatalf("expected *ShellResult, got %T", result)
	}

	if shellResult.Stderr != "error" {
		t.Errorf("expected 'error' in stderr, got %q", shellResult.Stderr)
	}
}

func TestShellTool_WorkingDir(t *testing.T) {
	tmpDir := t.TempDir()
	tool := &ShellTool{WorkingDir: tmpDir}
	ctx := context.Background()

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "cd"
	} else {
		cmd = "pwd"
	}

	args, _ := json.Marshal(map[string]any{"command": cmd})
	result, err := tool.Execute(ctx, string(args))
	if err != nil {
		t.Fatal(err)
	}

	// The output should contain the temp dir path
	if !strings.Contains(result, tmpDir) {
		t.Errorf("expected working dir %s in result: %s", tmpDir, result)
	}
}

func TestShellTool_CWDParameter(t *testing.T) {
	tmpDir := t.TempDir()
	tool := &ShellTool{} // No default working dir
	ctx := context.Background()

	var cmd string
	if runtime.GOOS == "windows" {
		cmd = "cd"
	} else {
		cmd = "pwd"
	}

	args, _ := json.Marshal(map[string]any{"command": cmd, "cwd": tmpDir})
	result, err := tool.Execute(ctx, string(args))
	if err != nil {
		t.Fatal(err)
	}

	// The output should contain the temp dir path
	if !strings.Contains(result, tmpDir) {
		t.Errorf("expected cwd %s in result: %s", tmpDir, result)
	}
}
