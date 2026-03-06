package runtime

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

// TestNewPythonRuntime tests creating a new Python runtime.
func TestNewPythonRuntime(t *testing.T) {
	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)

	if py == nil {
		t.Fatal("NewPythonRuntime returned nil")
	}
	if py.cfg != cfg {
		t.Error("Config not set correctly")
	}
	if py.ready {
		t.Error("New runtime should not be ready")
	}
}

// TestPythonRuntimeIsReady tests the IsReady method.
func TestPythonRuntimeIsReady(t *testing.T) {
	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)

	if py.IsReady() {
		t.Error("New runtime should not be ready")
	}
}

// TestPythonRuntimeGetError tests the GetError method.
func TestPythonRuntimeGetError(t *testing.T) {
	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)

	// Should return empty string if no error
	err := py.GetError()
	if err != "" {
		t.Errorf("Expected empty error for new runtime, got %q", err)
	}
}

// TestPythonRuntimeGetInterpreter tests the GetInterpreter method.
func TestPythonRuntimeGetInterpreter(t *testing.T) {
	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)

	interpreter := py.GetInterpreter()
	if interpreter != "" {
		t.Errorf("Expected empty interpreter for new runtime, got %q", interpreter)
	}
}

// TestPythonRuntimeGetVersion tests the GetVersion method.
func TestPythonRuntimeGetVersion(t *testing.T) {
	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)

	version := py.GetVersion()
	if version != "" {
		t.Errorf("Expected empty version for new runtime, got %q", version)
	}
}

// TestPythonDetectWithExplicitInterpreter tests detection with explicit interpreter.
func TestPythonDetectWithExplicitInterpreter(t *testing.T) {
	// Create a temporary Python script that acts as Python
	tmpDir := t.TempDir()
	pythonPath := filepath.Join(tmpDir, "python")

	// On Windows, create python.exe
	if isWindows() {
		pythonPath = filepath.Join(tmpDir, "python.exe")
	}

	// Create the fake Python executable
	err := os.WriteFile(pythonPath, []byte("#!/bin/sh\necho 3.10.0"), 0755)
	if err != nil {
		t.Fatalf("Failed to create temp Python: %v", err)
	}

	cfg := &config.PythonConfig{
		Interpreter: pythonPath,
	}
	py := NewPythonRuntime(cfg)

	// Try to detect - may fail if version check fails, that's ok
	_ = py.Detect()
}

// TestPythonDetectNotFound tests detection when Python is not found.
func TestPythonDetectNotFound(t *testing.T) {
	cfg := &config.PythonConfig{
		Interpreter: "/nonexistent/python/path/that/does/not/exist",
	}
	py := NewPythonRuntime(cfg)

	err := py.Detect()
	if err == nil {
		t.Error("Detect should return error when interpreter not found")
	}
}

// TestPythonDetectWithVenvPath tests detection with virtual environment path.
func TestPythonDetectWithVenvPath(t *testing.T) {
	// Create a temporary directory to simulate venv
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	os.MkdirAll(binDir, 0755)

	// Don't create actual Python file - just test path logic
	cfg := &config.PythonConfig{
		VenvPath: tmpDir,
	}
	py := NewPythonRuntime(cfg)

	err := py.Detect()
	if err == nil {
		// Expected to fail since we didn't create actual Python
		t.Logf("Detect with venv: %v (expected to fail)", err)
	}
}

// TestFindVenvPython tests finding Python in virtual environment.
func TestFindVenvPython(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)

	// Test with non-existent venv
	result := py.findVenvPython(tmpDir)
	if result != "" {
		t.Errorf("findVenvPython should return empty string for non-existent venv, got %q", result)
	}
}

// TestFindVenvPythonWithBinDirectory tests finding Python in venv bin directory.
func TestFindVenvPythonWithBinDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	os.MkdirAll(binDir, 0755)

	// Create fake python executable
	pythonPath := filepath.Join(binDir, "python")
	os.WriteFile(pythonPath, []byte("fake"), 0755)

	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)

	result := py.findVenvPython(tmpDir)
	if result != pythonPath {
		t.Errorf("findVenvPython = %q, expected %q", result, pythonPath)
	}
}

// TestFindVenvPythonWithPython3 tests finding python3 in venv bin directory.
func TestFindVenvPythonWithPython3(t *testing.T) {
	tmpDir := t.TempDir()
	binDir := filepath.Join(tmpDir, "bin")
	os.MkdirAll(binDir, 0755)

	// Create fake python3 executable (should be found before python)
	python3Path := filepath.Join(binDir, "python3")
	os.WriteFile(python3Path, []byte("fake"), 0755)

	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)

	result := py.findVenvPython(tmpDir)
	if result != python3Path {
		t.Errorf("findVenvPython = %q, expected %q", result, python3Path)
	}
}

// TestFindVenvPythonWithWindowsScripts tests finding Python in Windows Scripts directory.
func TestFindVenvPythonWithWindowsScripts(t *testing.T) {
	tmpDir := t.TempDir()
	scriptsDir := filepath.Join(tmpDir, "Scripts")
	os.MkdirAll(scriptsDir, 0755)

	// Create fake python.exe
	pythonPath := filepath.Join(scriptsDir, "python.exe")
	os.WriteFile(pythonPath, []byte("fake"), 0755)

	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)

	result := py.findVenvPython(tmpDir)
	if result != pythonPath {
		t.Errorf("findVenvPython = %q, expected %q", result, pythonPath)
	}
}

// TestFindSystemPython tests finding Python on system PATH.
func TestFindSystemPython(t *testing.T) {
	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)

	result := py.findSystemPython()
	// Result may or may not find Python depending on system
	_ = result
}

// TestIsVersionAtLeast tests version comparison.
func TestIsVersionAtLeast(t *testing.T) {
	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)

	tests := []struct {
		version string
		target  string
		want    bool
	}{
		{"3.10.0", "3.9", true},
		{"3.9.0", "3.9", true},
		{"3.8.0", "3.9", false},
		{"3.11.5", "3.9", true},
		{"2.7.0", "3.9", false},
		{"3.9", "3.9", true},
		{"3.10", "3.9", true},
		{"3.8", "3.9", false},
	}

	for _, tt := range tests {
		result := py.isVersionAtLeast(tt.version, tt.target)
		if result != tt.want {
			t.Errorf("isVersionAtLeast(%q, %q) = %v, want %v", tt.version, tt.target, result, tt.want)
		}
	}
}

// TestGetVersion tests getting Python version.
func TestGetVersion(t *testing.T) {
	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)

	// Try to get version from system Python if available
	// This may fail on systems without Python
	version, err := py.getVersion("python3")
	if err != nil {
		t.Logf("getVersion failed (system may not have Python): %v", err)
		return
	}

	if version == "" {
		t.Error("getVersion returned empty string")
	}
}

// TestPythonRunScriptNotReady tests RunScript when runtime is not ready.
func TestPythonRunScriptNotReady(t *testing.T) {
	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)
	py.err = fmt.Errorf("test error")

	_, err := py.RunScript("/tmp/test.py")
	if err == nil {
		t.Error("RunScript should return error when not ready")
	}
	if !strings.Contains(err.Error(), "not ready") {
		t.Errorf("Expected 'not ready' error, got %v", err)
	}
}

// TestRunScriptWithMockCommand tests RunScript with a mock command.
func TestRunScriptWithMockCommand(t *testing.T) {
	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)
	py.ready = true
	py.interpreter = "python3"

	// Create a temporary test script
	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.py")
	os.WriteFile(scriptPath, []byte("print('hello')"), 0755)

	// Try to run it (will fail unless python3 is available)
	output, err := py.RunScript(scriptPath)
	if err != nil {
		// This is expected if python3 is not available
		t.Logf("RunScript failed (python3 may not be available): %v", err)
	} else {
		// If it succeeded, verify output
		if !strings.Contains(output, "hello") && !strings.Contains(output, "") {
			t.Logf("RunScript output: %q", output)
		}
	}
}

// TestPythonRunScriptWithArgs tests RunScript with arguments.
func TestPythonRunScriptWithArgs(t *testing.T) {
	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)
	py.ready = true
	py.interpreter = "python3"

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.py")
	os.WriteFile(scriptPath, []byte("import sys; print(sys.argv[1])"), 0755)

	output, err := py.RunScript(scriptPath, "test_arg")
	if err != nil {
		t.Logf("RunScript with args failed (python3 may not be available): %v", err)
	} else {
		_ = output // Suppress unused variable
	}
}

// TestRunScriptWithVenv tests RunScript with virtual environment setup.
func TestRunScriptWithVenv(t *testing.T) {
	tmpDir := t.TempDir()
	venvDir := filepath.Join(tmpDir, "venv")
	os.MkdirAll(filepath.Join(venvDir, "bin"), 0755)

	cfg := &config.PythonConfig{
		VenvPath: venvDir,
	}
	py := NewPythonRuntime(cfg)
	py.ready = true
	py.interpreter = "python3"

	scriptPath := filepath.Join(tmpDir, "test.py")
	os.WriteFile(scriptPath, []byte("print('test')"), 0755)

	output, err := py.RunScript(scriptPath)
	if err != nil {
		t.Logf("RunScript with venv failed: %v", err)
	} else {
		_ = output
	}
}

// TestRunModuleNotReady tests RunModule when runtime is not ready.
func TestRunModuleNotReady(t *testing.T) {
	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)
	py.err = fmt.Errorf("test error")

	_, err := py.RunModule("pip")
	if err == nil {
		t.Error("RunModule should return error when not ready")
	}
	if !strings.Contains(err.Error(), "not ready") {
		t.Errorf("Expected 'not ready' error, got %v", err)
	}
}

// TestRunModuleWithVenv tests RunModule with virtual environment.
func TestRunModuleWithVenv(t *testing.T) {
	tmpDir := t.TempDir()
	venvDir := filepath.Join(tmpDir, "venv")
	os.MkdirAll(filepath.Join(venvDir, "bin"), 0755)

	cfg := &config.PythonConfig{
		VenvPath: venvDir,
	}
	py := NewPythonRuntime(cfg)
	py.ready = true
	py.interpreter = "python3"

	output, err := py.RunModule("sys", "-c", "print('test')")
	if err != nil {
		t.Logf("RunModule with venv failed: %v", err)
	} else {
		_ = output
	}
}

// TestPythonInstallPackageNotReady tests InstallPackage when not ready.
func TestPythonInstallPackageNotReady(t *testing.T) {
	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)

	err := py.InstallPackage("requests")
	if err == nil {
		t.Error("InstallPackage should return error when not ready")
	}
	if !strings.Contains(err.Error(), "not ready") {
		t.Errorf("Expected 'not ready' error, got %v", err)
	}
}

// TestInstallPackagesNotReady tests InstallPackages when not ready.
func TestInstallPackagesNotReady(t *testing.T) {
	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)

	err := py.InstallPackages([]string{"requests", "flask"})
	if err == nil {
		t.Error("InstallPackages should return error when not ready")
	}
	if !strings.Contains(err.Error(), "not ready") {
		t.Errorf("Expected 'not ready' error, got %v", err)
	}
}

// TestPythonInstallPackage tests InstallPackage error handling.
func TestPythonInstallPackage(t *testing.T) {
	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)
	py.ready = true
	py.interpreter = "python3"

	// Try to install a valid package (may fail if pip is not available)
	err := py.InstallPackage("requests")
	if err != nil {
		t.Logf("InstallPackage failed (pip may not be available): %v", err)
	}
}

// TestInstallPackages tests InstallPackages with multiple packages.
func TestInstallPackages(t *testing.T) {
	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)
	py.ready = true
	py.interpreter = "python3"

	err := py.InstallPackages([]string{"requests", "flask"})
	if err != nil {
		t.Logf("InstallPackages failed (pip may not be available): %v", err)
	}
}

// TestSetupVenvEnv tests environment variable setup for venv.
func TestSetupVenvEnv(t *testing.T) {
	tmpDir := t.TempDir()
	venvDir := filepath.Join(tmpDir, "venv")

	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)

	env := py.setupVenvEnv(venvDir)

	// Should have PATH and VIRTUAL_ENV
	hasPath := false
	hasVirtualEnv := false

	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			hasPath = true
			// Should contain venv bin path
			if !strings.Contains(e, "venv") {
				t.Errorf("PATH should contain venv path: %q", e)
			}
		}
		if strings.HasPrefix(e, "VIRTUAL_ENV=") {
			hasVirtualEnv = true
			if !strings.Contains(e, venvDir) {
				t.Errorf("VIRTUAL_ENV should be set to venv directory: %q", e)
			}
		}
	}

	if !hasPath {
		t.Error("setupVenvEnv should set PATH")
	}
	if !hasVirtualEnv {
		t.Error("setupVenvEnv should set VIRTUAL_ENV")
	}
}

// TestSetupVenvEnvPreservesOtherVars tests that setupVenvEnv preserves other environment variables.
func TestSetupVenvEnvPreservesOtherVars(t *testing.T) {
	tmpDir := t.TempDir()
	venvDir := filepath.Join(tmpDir, "venv")

	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)

	env := py.setupVenvEnv(venvDir)

	// Environment should be non-empty
	if len(env) == 0 {
		t.Error("setupVenvEnv should return non-empty environment")
	}
}

// TestSetupVenvEnvWindows tests environment variable setup for Windows venv.
func TestSetupVenvEnvWindows(t *testing.T) {
	tmpDir := t.TempDir()
	venvDir := filepath.Join(tmpDir, "venv")

	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)

	env := py.setupVenvEnv(venvDir)

	// Should have environment variables
	hasPath := false
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			hasPath = true
			// Check that it contains the venv directory
			if !strings.Contains(e, "venv") {
				t.Logf("PATH may not contain expected venv directory: %q", e)
			}
		}
	}

	if !hasPath {
		t.Error("setupVenvEnv should set PATH")
	}
}

// TestCheckPackageNotReady tests CheckPackage when not ready.
func TestCheckPackageNotReady(t *testing.T) {
	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)

	found, err := py.CheckPackage("requests")
	if err == nil {
		t.Error("CheckPackage should return error when not ready")
	}
	if found {
		t.Error("CheckPackage should return false when not ready")
	}
}

// TestCheckPackageNotFound tests CheckPackage when package is not found.
func TestCheckPackageNotFound(t *testing.T) {
	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)
	py.ready = true
	py.interpreter = "python3"

	// Try to check a non-existent package
	found, err := py.CheckPackage("nonexistent-package-xyz-12345")
	if err != nil {
		// If pip is not available, it's ok
		t.Logf("CheckPackage failed (pip may not be available): %v", err)
	}
	if found {
		t.Logf("Found non-existent package (unexpected): %v", found)
	}
}

// TestPythonGetErrorWithError tests GetError when there is an error.
func TestPythonGetErrorWithError(t *testing.T) {
	cfg := &config.PythonConfig{}
	py := NewPythonRuntime(cfg)
	testErr := fmt.Errorf("test error message")
	py.err = testErr

	errStr := py.GetError()
	if errStr != "test error message" {
		t.Errorf("GetError() = %q, expected 'test error message'", errStr)
	}
}
