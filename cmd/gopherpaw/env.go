package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/internal/runtime"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Environment management and diagnostics",
	Long:  "Manage GOPHERPAW_* environment variables and check runtime dependencies",
}

var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "List GOPHERPAW env vars",
	RunE:  runEnvList,
}

var envSetCmd = &cobra.Command{
	Use:   "set [KEY] [VALUE]",
	Short: "Set env var (prints hint)",
	Args:  cobra.MaximumNArgs(2),
	RunE:  runEnvSet,
}

var envDeleteCmd = &cobra.Command{
	Use:   "delete [KEY]",
	Short: "Delete env var (prints hint)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runEnvDelete,
}

var envCheckCmd = &cobra.Command{
	Use:   "check",
	Short: "Check runtime environment (Python, Bun, dependencies)",
	Long:  "Performs comprehensive environment diagnostics for GopherPaw skills and MCP support",
	RunE:  runEnvCheck,
}

var envSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup Python virtual environment",
	Long:  "Create and configure a Python virtual environment for GopherPaw skills",
	RunE:  runEnvSetup,
}

var (
	envSetupPath string
)

func init() {
	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envSetCmd)
	envCmd.AddCommand(envDeleteCmd)
	envCmd.AddCommand(envCheckCmd)
	envCmd.AddCommand(envSetupCmd)

	envSetupCmd.Flags().StringVar(&envSetupPath, "path", "", "Custom venv path (default: ~/.gopherpaw/venv)")
}

func runEnvList(cmd *cobra.Command, args []string) error {
	prefix := "GOPHERPAW_"
	found := false
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, prefix) {
			fmt.Println(e)
			found = true
		}
	}
	if !found {
		fmt.Println("No GOPHERPAW_* environment variables set")
	}
	return nil
}

func runEnvSet(cmd *cobra.Command, args []string) error {
	fmt.Println("Use export GOPHERPAW_<KEY>=<VALUE> to set environment variable")
	fmt.Println("Example: export GOPHERPAW_LLM_API_KEY=your-key")
	fmt.Println()
	fmt.Println("Common environment variables:")
	fmt.Println("  GOPHERPAW_LLM_API_KEY      - LLM API key")
	fmt.Println("  GOPHERPAW_LLM_BASE_URL     - LLM API base URL")
	fmt.Println("  GOPHERPAW_LLM_MODEL        - Default model name")
	fmt.Println("  GOPHERPAW_WORKING_DIR      - Working directory")
	fmt.Println("  GOPHERPAW_BUN_PATH         - Custom Bun executable path")
	return nil
}

func runEnvDelete(cmd *cobra.Command, args []string) error {
	fmt.Println("Use unset GOPHERPAW_<KEY> to delete environment variable")
	return nil
}

func runEnvCheck(cmd *cobra.Command, args []string) error {
	cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		// Use default config if file not found
		cfg = config.DefaultConfig()
	}

	rtMgr := runtime.NewManager(&cfg.Runtime)
	if err := rtMgr.Initialize(); err != nil {
		fmt.Printf("Warning: %v\n", err)
	}

	rtMgr.PrintEnvironmentReport()

	// Print setup hints if needed
	fmt.Println("Setup Instructions:")
	fmt.Println("===================")

	// Python setup
	if !rtMgr.GetPython().IsReady() {
		fmt.Println()
		fmt.Println("Python Setup:")
		fmt.Println("  1. Install Python 3.9+ (https://www.python.org/downloads/)")
		fmt.Println("  2. Create virtual environment:")
		fmt.Println("     python -m venv ~/.gopherpaw/venv")
		fmt.Println("  3. Activate virtual environment:")
		if goruntime.GOOS == "windows" {
			fmt.Println("     ~/.gopherpaw/venv/Scripts/activate")
		} else {
			fmt.Println("     source ~/.gopherpaw/venv/bin/activate")
		}
		fmt.Println("  4. Install dependencies:")
		fmt.Println("     pip install -r internal/runtime/requirements.txt")
		fmt.Println("  5. Configure in config.yaml:")
		fmt.Println("     runtime:")
		fmt.Println("       python:")
		fmt.Println("         venv_path: \"~/.gopherpaw/venv\"")
	}

	// Bun setup
	if !rtMgr.GetBun().IsReady() {
		fmt.Println()
		fmt.Println("Bun/Node Setup:")
		fmt.Println("  GopherPaw can auto-download Bun, or you can install manually:")
		fmt.Println("  - Bun (recommended): https://bun.sh")
		fmt.Println("  - Node.js: https://nodejs.org")
		fmt.Println()
		fmt.Println("  Or set GOPHERPAW_BUN_PATH to use a custom Bun installation.")
	}

	return nil
}

func runEnvSetup(cmd *cobra.Command, args []string) error {
	// Determine venv path
	venvPath := envSetupPath
	if venvPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		venvPath = filepath.Join(home, ".gopherpaw", "venv")
	}

	fmt.Printf("Setting up Python virtual environment at: %s\n", venvPath)

	// Check if Python is available
	pythonCmd := "python3"
	if goruntime.GOOS == "windows" {
		pythonCmd = "python"
	}

	if _, err := exec.LookPath(pythonCmd); err != nil {
		return fmt.Errorf("Python not found. Please install Python 3.9+ first")
	}

	// Create venv directory parent
	if err := os.MkdirAll(filepath.Dir(venvPath), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create virtual environment
	fmt.Println("Creating virtual environment...")
	createCmd := exec.Command(pythonCmd, "-m", "venv", venvPath)
	if output, err := createCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create venv: %w\n%s", err, string(output))
	}

	// Find pip in venv
	var pipPath string
	if goruntime.GOOS == "windows" {
		pipPath = filepath.Join(venvPath, "Scripts", "pip")
	} else {
		pipPath = filepath.Join(venvPath, "bin", "pip")
	}

	// Install dependencies
	fmt.Println("Installing dependencies...")
	installCmd := exec.Command(pipPath, "install", "--upgrade", "pip")
	if _, err := installCmd.CombinedOutput(); err != nil {
		fmt.Printf("Warning: failed to upgrade pip: %v\n", err)
	} else {
		fmt.Println("Pip upgraded successfully")
	}

	// Install requirements if available
	reqPath := "internal/runtime/requirements.txt"
	if _, err := os.Stat(reqPath); err == nil {
		reqInstallCmd := exec.Command(pipPath, "install", "-r", reqPath)
		fmt.Println("Installing requirements from:", reqPath)
		if output, err := reqInstallCmd.CombinedOutput(); err != nil {
			fmt.Printf("Warning: some packages may not have installed:\n%s\n", string(output))
		} else {
			fmt.Println("Requirements installed successfully")
		}
	}

	fmt.Println()
	fmt.Println("Virtual environment created successfully!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("1. Activate the virtual environment:")
	if goruntime.GOOS == "windows" {
		fmt.Printf("   %s\\Scripts\\activate\n", venvPath)
	} else {
		fmt.Printf("   source %s/bin/activate\n", venvPath)
	}
	fmt.Println()
	fmt.Println("2. Add to your config.yaml:")
	fmt.Println("   runtime:")
	fmt.Println("     python:")
	fmt.Println("       venv_path:", venvPath)
	fmt.Println()
	fmt.Println("3. Restart GopherPaw")

	return nil
}
