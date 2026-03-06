// Package runtime provides Python and Node.js/Bun runtime management for GopherPaw.
package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/suifei/gopherpaw/internal/config"
)

// Manager manages runtime environments (Python, Bun/Node).
type Manager struct {
	cfg    *config.RuntimeConfig
	python *PythonRuntime
	bun    *BunRuntime
}

// Status represents the status of a runtime environment.
type Status struct {
	Name     string `json:"name"`
	Ready    bool   `json:"ready"`
	Path     string `json:"path"`
	Version  string `json:"version"`
	Error    string `json:"error,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// NewManager creates a new runtime manager.
func NewManager(cfg *config.RuntimeConfig) *Manager {
	return &Manager{
		cfg: cfg,
	}
}

// Initialize detects and initializes all runtimes.
func (m *Manager) Initialize() error {
	// Initialize Python
	m.python = NewPythonRuntime(&m.cfg.Python)
	if err := m.python.Detect(); err != nil {
		// Non-fatal: Python skills will be unavailable
		fmt.Printf("Warning: Python runtime not available: %v\n", err)
	}

	// Initialize Bun
	m.bun = NewBunRuntime(&m.cfg.Node)
	if err := m.bun.Detect(); err != nil {
		// Non-fatal: Node-based skills will be unavailable
		fmt.Printf("Warning: Bun/Node runtime not available: %v\n", err)
	}

	return nil
}

// GetPython returns the Python runtime.
func (m *Manager) GetPython() *PythonRuntime {
	return m.python
}

// GetBun returns the Bun runtime.
func (m *Manager) GetBun() *BunRuntime {
	return m.bun
}

// CheckEnvironment performs environment diagnostics.
func (m *Manager) CheckEnvironment() []Status {
	var statuses []Status

	// Check Python
	if m.python != nil {
		status := Status{
			Name:  "Python",
			Ready: m.python.IsReady(),
		}
		if m.python.IsReady() {
			status.Path = m.python.GetInterpreter()
			status.Version = m.python.GetVersion()
		} else {
			status.Error = m.python.GetError()
		}
		statuses = append(statuses, status)
	} else {
		statuses = append(statuses, Status{
			Name:  "Python",
			Ready: false,
			Error: "Not initialized",
		})
	}

	// Check Bun
	if m.bun != nil {
		status := Status{
			Name:  "Bun",
			Ready: m.bun.IsReady(),
		}
		if m.bun.IsReady() {
			status.Path = m.bun.GetPath()
			status.Version = m.bun.GetVersion()
		} else {
			status.Error = m.bun.GetError()
		}
		statuses = append(statuses, status)
	} else {
		statuses = append(statuses, Status{
			Name:  "Bun",
			Ready: false,
			Error: "Not initialized",
		})
	}

	// Check optional binaries for skills
	statuses = append(statuses, CheckSkillBinaries()...)

	return statuses
}

// PrintEnvironmentReport prints a formatted environment report.
func (m *Manager) PrintEnvironmentReport() {
	fmt.Println()
	fmt.Println("========================================")
	fmt.Println("  GopherPaw Environment Check")
	fmt.Println("========================================")

	statuses := m.CheckEnvironment()
	var missingBinaries []Status

	for _, s := range statuses {
		if s.Ready {
			symbol := "OK"
			if s.Version != "" {
				fmt.Printf("  %-12s %s %s (%s)\n", s.Name+":", symbol, s.Path, s.Version)
			} else {
				fmt.Printf("  %-12s %s %s\n", s.Name+":", symbol, s.Path)
			}
		} else {
			if s.Name == "Python" || s.Name == "Bun" {
				fmt.Printf("  %-12s MISSING - %s\n", s.Name+":", s.Error)
			} else {
				missingBinaries = append(missingBinaries, s)
			}
		}
	}

	if len(missingBinaries) > 0 {
		fmt.Println()
		fmt.Println("  Optional binaries (for Skills):")
		for _, s := range missingBinaries {
			fmt.Printf("  - %s (%s)\n", s.Name, s.Error)
		}
	}

	fmt.Println()
	fmt.Println("  Run 'gopherpaw env --help' for setup instructions")
	fmt.Println("========================================")
	fmt.Println()
}

// ExpandPath expands ~ and environment variables in a path.
func ExpandPath(p string) string {
	if p == "" {
		return ""
	}

	// Expand ~ to home directory
	if strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		p = filepath.Join(home, p[2:])
	}

	// Expand environment variables
	p = os.ExpandEnv(p)

	// Convert to absolute path
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	return abs
}

// GetDefaultBinDir returns the default directory for bundled binaries.
func GetDefaultBinDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".gopherpaw/bin"
	}
	return filepath.Join(home, ".gopherpaw", "bin")
}

// runCommand executes a command and returns its output.
func runCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w: %s", name, err, string(output))
	}
	return strings.TrimSpace(string(output)), nil
}

// isWindows returns true if running on Windows.
func isWindows() bool {
	return runtime.GOOS == "windows"
}
