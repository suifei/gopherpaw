package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// SkillBinary represents a binary dependency for a skill.
type SkillBinary struct {
	Name        string
	Description string
	Package     string // Package name for installation hint
	Platforms   []string
}

// Common skill binaries that may be required.
var skillBinaries = []SkillBinary{
	{
		Name:        "soffice",
		Description: "LibreOffice - required for docx/xlsx/pdf skills",
		Package:     "libreoffice",
		Platforms:   []string{"linux", "darwin", "windows"},
	},
	{
		Name:        "pdftoppm",
		Description: "Poppler - required for PDF to image conversion",
		Package:     "poppler-utils (linux) / poppler (mac) / poppler-windows (windows)",
		Platforms:   []string{"linux", "darwin", "windows"},
	},
	{
		Name:        "himalaya",
		Description: "Email CLI - required for email skills",
		Package:     "himalaya (cargo install himalaya)",
		Platforms:   []string{"linux", "darwin", "windows"},
	},
	{
		Name:        "pandoc",
		Description: "Document converter - required for document processing",
		Package:     "pandoc",
		Platforms:   []string{"linux", "darwin", "windows"},
	},
	{
		Name:        "ffmpeg",
		Description: "Media processor - required for audio/video skills",
		Package:     "ffmpeg",
		Platforms:   []string{"linux", "darwin", "windows"},
	},
	{
		Name:        "git",
		Description: "Version control - required for git operations",
		Package:     "git",
		Platforms:   []string{"linux", "darwin", "windows"},
	},
}

// CheckSkillBinaries checks for common skill binary dependencies.
func CheckSkillBinaries() []Status {
	var statuses []Status

	for _, bin := range skillBinaries {
		status := Status{
			Name: bin.Name,
		}

		// Check if binary exists
		path, err := exec.LookPath(bin.Name)
		if err != nil {
			// Try common variations
			variations := getBinaryVariations(bin.Name)
			for _, v := range variations {
				path, err = exec.LookPath(v)
				if err == nil {
					break
				}
			}
		}

		if err == nil {
			status.Ready = true
			status.Path = path
			// Try to get version
			version, _ := getBinaryVersion(bin.Name, path)
			status.Version = version
		} else {
			status.Ready = false
			status.Error = fmt.Sprintf("Not found. Install: %s", bin.Package)
		}

		statuses = append(statuses, status)
	}

	return statuses
}

// getBinaryVariations returns common variations of a binary name.
func getBinaryVariations(name string) []string {
	var vars []string

	if isWindows() {
		// Try .exe extension
		vars = append(vars, name+".exe")
	}

	// Try with version suffix (e.g., python3, python3.11)
	switch name {
	case "python":
		vars = append(vars, "python3", "python3.11", "python3.10", "python3.9")
	case "soffice":
		vars = append(vars, "libreoffice")
	}

	return vars
}

// getBinaryVersion tries to get version string for a binary.
func getBinaryVersion(name, path string) (string, error) {
	var args []string

	switch name {
	case "python", "python3":
		args = []string{"--version"}
	case "node":
		args = []string{"--version"}
	case "bun":
		args = []string{"--version"}
	case "git":
		args = []string{"--version"}
	case "soffice":
		args = []string{"--version"}
	case "pandoc":
		args = []string{"--version"}
	case "ffmpeg":
		args = []string{"-version"}
	case "himalaya":
		args = []string{"--version"}
	default:
		args = []string{"--version"}
	}

	output, err := runCommand(path, args...)
	if err != nil {
		return "", err
	}

	// Extract first line and clean up
	lines := strings.Split(output, "\n")
	if len(lines) > 0 {
		// Remove common prefixes
		version := strings.TrimSpace(lines[0])
		version = strings.TrimPrefix(version, name)
		version = strings.TrimSpace(version)
		return version, nil
	}

	return output, nil
}

// DetectPlatform returns the current platform identifier.
func DetectPlatform() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}

// IsCommandAvailable checks if a command is available on the system.
func IsCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// FindBinary finds a binary in PATH or common locations.
func FindBinary(name string) string {
	// First try PATH
	if path, err := exec.LookPath(name); err == nil {
		return path
	}

	// Try common locations
	commonPaths := []string{
		"/usr/local/bin",
		"/usr/bin",
		"/opt/homebrew/bin",
		"/opt/local/bin",
	}

	if isWindows() {
		// Windows common paths
		commonPaths = []string{
			"C:\\Program Files",
			"C:\\Program Files (x86)",
			os.Getenv("LOCALAPPDATA"),
			os.Getenv("PROGRAMFILES"),
		}
	}

	for _, dir := range commonPaths {
		if dir == "" {
			continue
		}
		candidate := filepath.Join(dir, name)
		if isWindows() {
			candidate += ".exe"
		}
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}

	return ""
}

// GetInstallHint returns installation hint for a binary.
func GetInstallHint(name string) string {
	for _, bin := range skillBinaries {
		if bin.Name == name {
			return bin.Package
		}
	}
	return ""
}
