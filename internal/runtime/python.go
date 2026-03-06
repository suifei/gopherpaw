package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/suifei/gopherpaw/internal/config"
)

// PythonRuntime manages Python interpreter and virtual environment.
type PythonRuntime struct {
	cfg         *config.PythonConfig
	interpreter string
	version     string
	ready       bool
	err         error
}

// NewPythonRuntime creates a new Python runtime manager.
func NewPythonRuntime(cfg *config.PythonConfig) *PythonRuntime {
	return &PythonRuntime{
		cfg: cfg,
	}
}

// Detect finds and validates the Python interpreter.
func (p *PythonRuntime) Detect() error {
	// Priority: interpreter > venv_path > system python
	pythonPath := ""

	// 1. Check explicit interpreter setting
	if p.cfg.Interpreter != "" {
		pythonPath = ExpandPath(p.cfg.Interpreter)
		if _, err := os.Stat(pythonPath); err != nil {
			p.err = fmt.Errorf("configured interpreter not found: %s", pythonPath)
			return p.err
		}
	} else if p.cfg.VenvPath != "" {
		// 2. Check virtual environment
		venvPath := ExpandPath(p.cfg.VenvPath)
		pythonPath = p.findVenvPython(venvPath)
		if pythonPath == "" {
			p.err = fmt.Errorf("python not found in venv: %s", venvPath)
			return p.err
		}
	} else {
		// 3. Try system python
		pythonPath = p.findSystemPython()
		if pythonPath == "" {
			p.err = fmt.Errorf("no python interpreter found on system")
			return p.err
		}
	}

	// Validate python version
	version, err := p.getVersion(pythonPath)
	if err != nil {
		p.err = fmt.Errorf("failed to get python version: %w", err)
		return p.err
	}

	// Check minimum version (3.9+)
	if !p.isVersionAtLeast(version, "3.9") {
		p.err = fmt.Errorf("python version %s is too old, minimum required is 3.9", version)
		return p.err
	}

	p.interpreter = pythonPath
	p.version = version
	p.ready = true
	p.err = nil
	return nil
}

// findVenvPython finds the Python executable in a virtual environment.
func (p *PythonRuntime) findVenvPython(venvPath string) string {
	// Check common locations
	candidates := []string{
		filepath.Join(venvPath, "bin", "python"),
		filepath.Join(venvPath, "bin", "python3"),
		filepath.Join(venvPath, "Scripts", "python.exe"), // Windows
		filepath.Join(venvPath, "Scripts", "python3.exe"),
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// findSystemPython finds Python on the system PATH.
func (p *PythonRuntime) findSystemPython() string {
	// Try common names
	names := []string{"python3", "python"}
	if isWindows() {
		names = []string{"python.exe", "python3.exe"}
	}

	for _, name := range names {
		if path, err := exec.LookPath(name); err == nil {
			return path
		}
	}
	return ""
}

// getVersion gets the Python version string.
func (p *PythonRuntime) getVersion(pythonPath string) (string, error) {
	cmd := exec.Command(pythonPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	// Output format: "Python 3.11.2"
	parts := strings.Fields(string(output))
	if len(parts) >= 2 {
		return parts[1], nil
	}
	return strings.TrimSpace(string(output)), nil
}

// isVersionAtLeast checks if version is >= minVersion.
func (p *PythonRuntime) isVersionAtLeast(version, minVersion string) bool {
	// Simple comparison: assumes format "X.Y.Z"
	vParts := strings.Split(version, ".")
	mParts := strings.Split(minVersion, ".")

	for i := 0; i < len(vParts) && i < len(mParts); i++ {
		vNum := 0
		mNum := 0
		fmt.Sscanf(vParts[i], "%d", &vNum)
		fmt.Sscanf(mParts[i], "%d", &mNum)
		if vNum < mNum {
			return false
		} else if vNum > mNum {
			return true
		}
	}
	return true
}

// IsReady returns true if Python is available.
func (p *PythonRuntime) IsReady() bool {
	return p.ready
}

// GetInterpreter returns the Python interpreter path.
func (p *PythonRuntime) GetInterpreter() string {
	return p.interpreter
}

// GetVersion returns the Python version.
func (p *PythonRuntime) GetVersion() string {
	return p.version
}

// GetError returns the error if Python is not available.
func (p *PythonRuntime) GetError() string {
	if p.err != nil {
		return p.err.Error()
	}
	return ""
}

// RunScript executes a Python script with the given arguments.
func (p *PythonRuntime) RunScript(scriptPath string, args ...string) (string, error) {
	if !p.ready {
		return "", fmt.Errorf("python runtime not ready: %s", p.err)
	}

	allArgs := append([]string{scriptPath}, args...)
	cmd := exec.Command(p.interpreter, allArgs...)

	// Set up environment for venv if configured
	if p.cfg.VenvPath != "" {
		venvPath := ExpandPath(p.cfg.VenvPath)
		cmd.Env = p.setupVenvEnv(venvPath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("python script failed: %w: %s", err, string(output))
	}
	return string(output), nil
}

// RunModule executes a Python module with the given arguments.
func (p *PythonRuntime) RunModule(module string, args ...string) (string, error) {
	if !p.ready {
		return "", fmt.Errorf("python runtime not ready: %s", p.err)
	}

	allArgs := append([]string{"-m", module}, args...)
	cmd := exec.Command(p.interpreter, allArgs...)

	if p.cfg.VenvPath != "" {
		venvPath := ExpandPath(p.cfg.VenvPath)
		cmd.Env = p.setupVenvEnv(venvPath)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("python module failed: %w: %s", err, string(output))
	}
	return string(output), nil
}

// InstallPackage installs a Python package using pip.
func (p *PythonRuntime) InstallPackage(pkg string) error {
	if !p.ready {
		return fmt.Errorf("python runtime not ready")
	}

	_, err := p.RunModule("pip", "install", pkg)
	return err
}

// InstallPackages installs multiple Python packages.
func (p *PythonRuntime) InstallPackages(packages []string) error {
	if !p.ready {
		return fmt.Errorf("python runtime not ready")
	}

	args := append([]string{"install"}, packages...)
	_, err := p.RunModule("pip", args...)
	return err
}

// setupVenvEnv sets up environment variables for virtual environment.
func (p *PythonRuntime) setupVenvEnv(venvPath string) []string {
	env := os.Environ()

	// Update PATH to include venv bin
	binPath := filepath.Join(venvPath, "bin")
	if isWindows() {
		binPath = filepath.Join(venvPath, "Scripts")
	}

	// Find and update PATH
	newEnv := make([]string, 0, len(env))
	pathUpdated := false
	for _, e := range env {
		if strings.HasPrefix(e, "PATH=") {
			oldPath := strings.TrimPrefix(e, "PATH=")
			newPath := binPath + string(os.PathListSeparator) + oldPath
			newEnv = append(newEnv, "PATH="+newPath)
			pathUpdated = true
		} else if strings.HasPrefix(e, "VIRTUAL_ENV=") {
			// Skip old VIRTUAL_ENV
		} else {
			newEnv = append(newEnv, e)
		}
	}

	if !pathUpdated {
		newEnv = append(newEnv, "PATH="+binPath)
	}
	newEnv = append(newEnv, "VIRTUAL_ENV="+venvPath)

	return newEnv
}

// CheckPackage checks if a Python package is installed.
func (p *PythonRuntime) CheckPackage(pkg string) (bool, error) {
	if !p.ready {
		return false, fmt.Errorf("python runtime not ready")
	}

	_, err := p.RunModule("pip", "show", pkg)
	if err != nil {
		// pip show returns error if package not found
		return false, nil
	}
	return true, nil
}
