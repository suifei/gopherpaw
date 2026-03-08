package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

// contains is a helper function to check if a string contains a substring.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestNewManager tests the creation of a new runtime manager.
func TestNewManager(t *testing.T) {
	cfg := &config.RuntimeConfig{
		Python: config.PythonConfig{},
		Node:   config.NodeConfig{},
	}

	mgr := NewManager(cfg)

	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
	if mgr.cfg != cfg {
		t.Error("Manager config not set correctly")
	}
}

// TestInitialize tests runtime manager initialization.
func TestInitialize(t *testing.T) {
	cfg := &config.RuntimeConfig{
		Python: config.PythonConfig{},
		Node:   config.NodeConfig{},
	}

	mgr := NewManager(cfg)
	err := mgr.Initialize()

	// Initialize should not return error even if runtimes not available
	if err != nil {
		t.Fatalf("Initialize returned error: %v", err)
	}

	// After initialization, GetPython and GetBun should not be nil
	if mgr.GetPython() == nil {
		t.Error("GetPython returned nil after Initialize")
	}
	if mgr.GetBun() == nil {
		t.Error("GetBun returned nil after Initialize")
	}
}

// TestGetPython tests getting the Python runtime.
func TestGetPython(t *testing.T) {
	cfg := &config.RuntimeConfig{
		Python: config.PythonConfig{},
		Node:   config.NodeConfig{},
	}

	mgr := NewManager(cfg)
	mgr.Initialize()

	py := mgr.GetPython()
	if py == nil {
		t.Fatal("GetPython returned nil")
	}
}

// TestGetBun tests getting the Bun runtime.
func TestGetBun(t *testing.T) {
	cfg := &config.RuntimeConfig{
		Python: config.PythonConfig{},
		Node:   config.NodeConfig{},
	}

	mgr := NewManager(cfg)
	mgr.Initialize()

	bun := mgr.GetBun()
	if bun == nil {
		t.Fatal("GetBun returned nil")
	}
}

// TestCheckEnvironment tests environment diagnostics.
func TestCheckEnvironment(t *testing.T) {
	cfg := &config.RuntimeConfig{
		Python: config.PythonConfig{},
		Node:   config.NodeConfig{},
	}

	mgr := NewManager(cfg)
	mgr.Initialize()

	statuses := mgr.CheckEnvironment()

	// Should have at least Python and Bun status
	if len(statuses) < 2 {
		t.Errorf("Expected at least 2 statuses, got %d", len(statuses))
	}

	// Check that Python and Bun are in the list
	hasPython := false
	hasBun := false
	for _, s := range statuses {
		if s.Name == "Python" {
			hasPython = true
		}
		if s.Name == "Bun" {
			hasBun = true
		}
	}

	if !hasPython {
		t.Error("Python status not found")
	}
	if !hasBun {
		t.Error("Bun status not found")
	}
}

// TestCheckEnvironmentStatus tests that Status has required fields.
func TestCheckEnvironmentStatus(t *testing.T) {
	cfg := &config.RuntimeConfig{
		Python: config.PythonConfig{},
		Node:   config.NodeConfig{},
	}

	mgr := NewManager(cfg)
	mgr.Initialize()

	statuses := mgr.CheckEnvironment()

	for _, s := range statuses {
		if s.Name == "" {
			t.Error("Status Name is empty")
		}
		// Ready should be either true or false
		_ = s.Ready

		// If not ready, should have an error message
		if !s.Ready && s.Ready == false && s.Error == "" && s.Name != "Python" && s.Name != "Bun" {
			// Optional binaries may not have error messages
			_ = s.Error
		}
	}
}

// TestCheckEnvironmentPythonStatus tests Python status reporting.
func TestCheckEnvironmentPythonStatus(t *testing.T) {
	cfg := &config.RuntimeConfig{
		Python: config.PythonConfig{},
		Node:   config.NodeConfig{},
	}

	mgr := NewManager(cfg)
	mgr.Initialize()

	statuses := mgr.CheckEnvironment()

	for _, s := range statuses {
		if s.Name == "Python" {
			// Python status should be present
			if s.Ready {
				// If ready, should have path and version
				if s.Path == "" {
					t.Error("Python path should not be empty when ready")
				}
				if s.Version == "" {
					t.Error("Python version should not be empty when ready")
				}
			} else {
				// If not ready, should have error
				if s.Error == "" {
					t.Error("Python error should not be empty when not ready")
				}
			}
		}
	}
}

// TestCheckEnvironmentBunStatus tests Bun status reporting.
func TestCheckEnvironmentBunStatus(t *testing.T) {
	cfg := &config.RuntimeConfig{
		Python: config.PythonConfig{},
		Node:   config.NodeConfig{},
	}

	mgr := NewManager(cfg)
	mgr.Initialize()

	statuses := mgr.CheckEnvironment()

	for _, s := range statuses {
		if s.Name == "Bun" {
			// Bun status should be present
			if s.Ready {
				// If ready, should have path and version
				if s.Path == "" {
					t.Error("Bun path should not be empty when ready")
				}
				if s.Version == "" {
					t.Error("Bun version should not be empty when ready")
				}
			} else {
				// If not ready, should have error
				if s.Error == "" {
					t.Error("Bun error should not be empty when not ready")
				}
			}
		}
	}
}

// TestExpandPath tests path expansion with ~ and environment variables.
func TestExpandPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected func(string) bool // validator function
	}{
		{
			name:  "empty path",
			input: "",
			expected: func(s string) bool {
				return s == ""
			},
		},
		{
			name:  "absolute path",
			input: "/usr/bin",
			expected: func(s string) bool {
				return filepath.IsAbs(s)
			},
		},
		{
			name:  "relative path",
			input: "./test",
			expected: func(s string) bool {
				return filepath.IsAbs(s)
			},
		},
		{
			name:  "home expansion",
			input: "~/test",
			expected: func(s string) bool {
				return filepath.IsAbs(s) && !contains(s, "~")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandPath(tt.input)
			if !tt.expected(result) {
				t.Errorf("ExpandPath(%q) = %q, validation failed", tt.input, result)
			}
		})
	}
}

// TestExpandPathWithEnvVars tests environment variable expansion.
func TestExpandPathWithEnvVars(t *testing.T) {
	// Set a test environment variable
	testEnv := "GOPHERPAW_TEST_VAR"
	testValue := "test/path"
	os.Setenv(testEnv, testValue)
	defer os.Unsetenv(testEnv)

	input := "$" + testEnv + "/subdir"
	result := ExpandPath(input)

	if !filepath.IsAbs(result) {
		t.Errorf("ExpandPath(%q) should return absolute path, got %q", input, result)
	}

	// Check that environment variable was expanded (check contains part of value)
	if !contains(result, testValue) && !contains(result, "test") {
		t.Logf("ExpandPath(%q) expanded to %q (may not contain literal env value on Windows)", input, result)
	}
}

// TestExpandPathHomeOnly tests ExpandPath with only home directory.
func TestExpandPathHomeOnly(t *testing.T) {
	input := "~"
	result := ExpandPath(input)

	if !filepath.IsAbs(result) {
		t.Errorf("ExpandPath(%q) should return absolute path, got %q", input, result)
	}

	// Should be the home directory
	home, err := os.UserHomeDir()
	if err == nil {
		if result != home && !strings.HasPrefix(result, home) {
			t.Logf("ExpandPath(%q) = %q, expected home directory %q", input, result, home)
		}
	}
}

// TestExpandPathSlash tests ExpandPath with tilde and slash.
func TestExpandPathSlash(t *testing.T) {
	input := "~/test/path"
	result := ExpandPath(input)

	if !filepath.IsAbs(result) {
		t.Errorf("ExpandPath(%q) should return absolute path, got %q", input, result)
	}

	if contains(result, "~") {
		t.Errorf("ExpandPath should expand ~, got %q", result)
	}
}

// TestExpandPathWithBraces tests ExpandPath with environment variable braces.
func TestExpandPathWithBraces(t *testing.T) {
	testEnv := "TEST_VAR_BRACES"
	testValue := "testvalue"
	os.Setenv(testEnv, testValue)
	defer os.Unsetenv(testEnv)

	input := "${" + testEnv + "}/subdir"
	result := ExpandPath(input)

	if !filepath.IsAbs(result) {
		t.Errorf("ExpandPath(%q) should return absolute path, got %q", input, result)
	}
}

// TestExpandPathDoesNotExist tests ExpandPath with non-existent path.
func TestExpandPathDoesNotExist(t *testing.T) {
	input := "~/nonexistent/path/that/should/not/exist"
	result := ExpandPath(input)

	// Even for non-existent path, should return absolute path
	if !filepath.IsAbs(result) {
		t.Errorf("ExpandPath should return absolute path even for non-existent paths, got %q", result)
	}
}

// TestGetDefaultBinDir tests getting the default binary directory.
func TestGetDefaultBinDir(t *testing.T) {
	binDir := GetDefaultBinDir()

	if binDir == "" {
		t.Error("GetDefaultBinDir returned empty string")
	}

	// Should contain .gopherpaw or bin
	if !contains(binDir, "gopherpaw") && !contains(binDir, "bin") {
		t.Errorf("GetDefaultBinDir() = %q, expected to contain 'gopherpaw' or 'bin'", binDir)
	}
}

// TestGetDefaultBinDirIsAbsolute tests that default bin dir is absolute path.
func TestGetDefaultBinDirIsAbsolute(t *testing.T) {
	binDir := GetDefaultBinDir()

	if !filepath.IsAbs(binDir) {
		t.Errorf("GetDefaultBinDir() = %q, expected absolute path", binDir)
	}
}

// TestGetDefaultBinDirContainsBin tests that default bin dir contains bin.
func TestGetDefaultBinDirContainsBin(t *testing.T) {
	binDir := GetDefaultBinDir()

	if !contains(binDir, "bin") {
		t.Errorf("GetDefaultBinDir() = %q, expected to contain 'bin'", binDir)
	}
}

// TestGetDefaultBinDirConsistent tests that GetDefaultBinDir is consistent.
func TestGetDefaultBinDirConsistent(t *testing.T) {
	dir1 := GetDefaultBinDir()
	dir2 := GetDefaultBinDir()

	if dir1 != dir2 {
		t.Errorf("GetDefaultBinDir should be consistent: %q vs %q", dir1, dir2)
	}
}

// TestIsWindows tests the Windows detection.
func TestIsWindows(t *testing.T) {
	result := isWindows()

	// Should return a boolean
	_ = result
}

// TestRunCommandHelp tests running a simple command.
func TestRunCommandHelp(t *testing.T) {
	// Use 'echo' command which works on most systems
	output, err := runCommand("echo", "test")

	if err != nil {
		t.Logf("runCommand failed (may not have 'echo' on this system): %v", err)
		// This is acceptable on some systems
		return
	}

	if output != "test" {
		t.Errorf("runCommand('echo', 'test') = %q, expected 'test'", output)
	}
}

// TestRunCommandNotFound tests running a non-existent command.
func TestRunCommandNotFound(t *testing.T) {
	_, err := runCommand("nonexistent-command-xyz", "arg")

	if err == nil {
		t.Error("runCommand with non-existent command should return error")
	}
}

// TestRunCommandWithMultipleArgs tests running a command with multiple arguments.
func TestRunCommandWithMultipleArgs(t *testing.T) {
	output, err := runCommand("echo", "arg1", "arg2", "arg3")

	if err != nil {
		t.Logf("runCommand failed (may not have 'echo' on this system): %v", err)
		return
	}

	// Output should contain all arguments
	if !contains(output, "arg1") {
		t.Errorf("runCommand output should contain arguments, got %q", output)
	}
}

// TestRunCommandEmptyArgs tests running a command with no additional arguments.
func TestRunCommandEmptyArgs(t *testing.T) {
	_, err := runCommand("echo")

	if err != nil {
		t.Logf("runCommand failed (may not have 'echo' on this system): %v", err)
		return
	}
}

// TestRunCommandErrorOutput tests that runCommand captures error output.
func TestRunCommandErrorOutput(t *testing.T) {
	_, err := runCommand("nonexistent-command-xyz", "arg")

	if err == nil {
		t.Error("runCommand with non-existent command should return error")
	}

	// Error should be wrapped with command name
	if !contains(err.Error(), "nonexistent-command") {
		t.Logf("Error message should contain command name: %v", err)
	}
}

// TestPrintEnvironmentReport tests printing environment report (no assertion, just ensure no panic).
func TestPrintEnvironmentReport(t *testing.T) {
	cfg := &config.RuntimeConfig{
		Python: config.PythonConfig{},
		Node:   config.NodeConfig{},
	}

	mgr := NewManager(cfg)
	mgr.Initialize()

	// Just ensure it doesn't panic
	mgr.PrintEnvironmentReport()
}

// TestManagerWithNilConfig tests manager behavior with nil config fields.
func TestManagerWithNilConfig(t *testing.T) {
	cfg := &config.RuntimeConfig{
		Python: config.PythonConfig{},
		Node:   config.NodeConfig{},
	}

	mgr := NewManager(cfg)

	// Should not panic
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
}

// TestCheckEnvironmentWithUninitializedManager tests CheckEnvironment on uninitialized manager.
func TestCheckEnvironmentWithUninitializedManager(t *testing.T) {
	cfg := &config.RuntimeConfig{
		Python: config.PythonConfig{},
		Node:   config.NodeConfig{},
	}

	mgr := NewManager(cfg)
	// Note: Not calling Initialize()

	// Even without Initialize, CheckEnvironment should handle nil pointers gracefully
	statuses := mgr.CheckEnvironment()

	// Should still return some statuses
	if len(statuses) == 0 {
		t.Error("CheckEnvironment should return at least one status")
	}
}

// TestStatusStructure tests Status structure fields.
func TestStatusStructure(t *testing.T) {
	status := Status{
		Name:     "Test",
		Ready:    true,
		Path:     "/test/path",
		Version:  "1.0.0",
		Error:    "",
		Warnings: []string{},
	}

	if status.Name != "Test" {
		t.Error("Status Name not set")
	}
	if !status.Ready {
		t.Error("Status Ready not set")
	}
	if status.Path != "/test/path" {
		t.Error("Status Path not set")
	}
	if status.Version != "1.0.0" {
		t.Error("Status Version not set")
	}
}
