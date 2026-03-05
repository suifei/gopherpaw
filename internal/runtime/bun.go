package runtime

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"

	"github.com/suifei/gopherpaw/internal/config"
)

// BunRuntime manages Bun JavaScript runtime.
type BunRuntime struct {
	cfg    *config.NodeConfig
	path   string
	version string
	ready  bool
	err    error
}

// NewBunRuntime creates a new Bun runtime manager.
func NewBunRuntime(cfg *config.NodeConfig) *BunRuntime {
	return &BunRuntime{
		cfg: cfg,
	}
}

// Detect finds or downloads Bun.
func (b *BunRuntime) Detect() error {
	// Priority: bun_path > env GOPHERPAW_BUN_PATH > bundled > system bun
	bunPath := ""

	// 1. Check explicit bun_path setting
	if b.cfg.BunPath != "" {
		bunPath = ExpandPath(b.cfg.BunPath)
		if _, err := os.Stat(bunPath); err != nil {
			b.err = fmt.Errorf("configured bun not found: %s", bunPath)
			return b.err
		}
	} else if v := os.Getenv("GOPHERPAW_BUN_PATH"); v != "" {
		// 2. Check environment variable
		bunPath = ExpandPath(v)
		if _, err := os.Stat(bunPath); err != nil {
			b.err = fmt.Errorf("GOPHERPAW_BUN_PATH not found: %s", bunPath)
			return b.err
		}
	} else {
		// 3. Check bundled bun
		bundledPath := b.getBundledBunPath()
		if _, err := os.Stat(bundledPath); err == nil {
			bunPath = bundledPath
		} else {
			// 4. Try system bun
			if path, err := exec.LookPath("bun"); err == nil {
				bunPath = path
			} else {
				// 5. Try to download bundled bun
				downloaded, err := b.downloadBun()
				if err != nil {
					b.err = fmt.Errorf("bun not available and download failed: %w", err)
					return b.err
				}
				bunPath = downloaded
			}
		}
	}

	// Validate bun version
	version, err := b.getVersion(bunPath)
	if err != nil {
		b.err = fmt.Errorf("failed to get bun version: %w", err)
		return b.err
	}

	b.path = bunPath
	b.version = version
	b.ready = true
	b.err = nil
	return nil
}

// getBundledBunPath returns the path where bundled bun is stored.
func (b *BunRuntime) getBundledBunPath() string {
	binDir := GetDefaultBinDir()
	if isWindows() {
		return filepath.Join(binDir, "bun.exe")
	}
	return filepath.Join(binDir, "bun")
}

// downloadBun downloads Bun to the bundled location.
func (b *BunRuntime) downloadBun() (string, error) {
	bundledPath := b.getBundledBunPath()
	binDir := filepath.Dir(bundledPath)

	// Create bin directory
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create bin directory: %w", err)
	}

	// Determine download URL based on OS and arch
	downloadURL, err := b.getDownloadURL()
	if err != nil {
		return "", err
	}

	fmt.Printf("Downloading Bun from %s...\n", downloadURL)

	// Download to temp file first
	tmpFile, err := os.CreateTemp("", "bun-download-*")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	resp, err := http.Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("failed to download bun: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %s", resp.Status)
	}

	// Write to temp file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to write download: %w", err)
	}
	tmpFile.Close()

	// Extract if zip (Windows)
	if strings.HasSuffix(downloadURL, ".zip") {
		err = b.extractZip(tmpFile.Name(), binDir)
	} else {
		// Copy directly and make executable
		err = copyFile(tmpFile.Name(), bundledPath)
		if err == nil {
			os.Chmod(bundledPath, 0755)
		}
	}

	if err != nil {
		return "", fmt.Errorf("failed to install bun: %w", err)
	}

	return bundledPath, nil
}

// getDownloadURL returns the Bun download URL for current platform.
func (b *BunRuntime) getDownloadURL() (string, error) {
	// Bun release URL pattern: https://github.com/oven-sh/bun/releases/latest/download/bun-{os}-{arch}.{ext}
	baseURL := "https://github.com/oven-sh/bun/releases/latest/download"

	var filename string
	switch goruntime.GOOS {
	case "windows":
		filename = fmt.Sprintf("bun-windows-%s.zip", goruntime.GOARCH)
	case "darwin":
		filename = fmt.Sprintf("bun-darwin-%s.zip", goruntime.GOARCH)
	case "linux":
		filename = fmt.Sprintf("bun-linux-%s.zip", goruntime.GOARCH)
	default:
		return "", fmt.Errorf("unsupported OS: %s", goruntime.GOOS)
	}

	return fmt.Sprintf("%s/%s", baseURL, filename), nil
}

// extractZip extracts a zip file to the target directory.
func (b *BunRuntime) extractZip(zipPath, targetDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// Find bun executable in the zip
		if strings.Contains(f.Name, "bun") && !strings.Contains(f.Name, "/") {
			// This is the bun executable
			dstPath := filepath.Join(targetDir, filepath.Base(f.Name))
			if isWindows() && !strings.HasSuffix(dstPath, ".exe") {
				dstPath += ".exe"
			}

			if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
				return err
			}

			dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}

			srcFile, err := f.Open()
			if err != nil {
				dstFile.Close()
				return err
			}

			_, err = io.Copy(dstFile, srcFile)
			srcFile.Close()
			dstFile.Close()
			if err != nil {
				return err
			}

			// Make executable on Unix
			if !isWindows() {
				os.Chmod(dstPath, 0755)
			}

			return nil
		}
	}

	return fmt.Errorf("bun executable not found in zip")
}

// getVersion gets the Bun version string.
func (b *BunRuntime) getVersion(bunPath string) (string, error) {
	cmd := exec.Command(bunPath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

// IsReady returns true if Bun is available.
func (b *BunRuntime) IsReady() bool {
	return b.ready
}

// GetPath returns the Bun executable path.
func (b *BunRuntime) GetPath() string {
	return b.path
}

// GetVersion returns the Bun version.
func (b *BunRuntime) GetVersion() string {
	return b.version
}

// GetError returns the error if Bun is not available.
func (b *BunRuntime) GetError() string {
	if b.err != nil {
		return b.err.Error()
	}
	return ""
}

// RunScript executes a JavaScript/TypeScript file with Bun.
func (b *BunRuntime) RunScript(scriptPath string, args ...string) (string, error) {
	if !b.ready {
		return "", fmt.Errorf("bun runtime not ready: %s", b.err)
	}

	allArgs := append([]string{"run", scriptPath}, args...)
	cmd := exec.Command(b.path, allArgs...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("bun script failed: %w: %s", err, string(output))
	}
	return string(output), nil
}

// InstallPackage installs a package using bun install.
func (b *BunRuntime) InstallPackage(pkg string) error {
	if !b.ready {
		return fmt.Errorf("bun runtime not ready")
	}

	cmd := exec.Command(b.path, "add", pkg)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("bun add failed: %w: %s", err, string(output))
	}
	return nil
}

// RunCommand executes a bun command.
func (b *BunRuntime) RunCommand(args ...string) (string, error) {
	if !b.ready {
		return "", fmt.Errorf("bun runtime not ready: %s", b.err)
	}

	cmd := exec.Command(b.path, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("bun command failed: %w: %s", err, string(output))
	}
	return string(output), nil
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
