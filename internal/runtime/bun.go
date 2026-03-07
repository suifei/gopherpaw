package runtime

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	goruntime "runtime"
	"strings"
	"time"

	"github.com/suifei/gopherpaw/internal/config"
)

// BunRuntime manages Bun JavaScript runtime.
type BunRuntime struct {
	cfg              *config.NodeConfig
	path             string
	version          string
	ready            bool
	err              error
	downloadAttempts int
}

// GitHubRelease represents a GitHub API response for releases
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name string `json:"name"`
		URL  string `json:"browser_download_url"`
	} `json:"assets"`
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
				// 5. Try to install bun using official script first
				if path, err := b.installWithOfficialScript(); err == nil {
					bunPath = path
				} else {
					// 6. Fallback to direct download
					fmt.Printf("Official script installation failed, trying direct download: %v\n", err)
					downloaded, err := b.downloadBunWithRetry(3)
					if err != nil {
						b.err = fmt.Errorf("bun not available and all installation methods failed: %w", err)
						return b.err
					}
					bunPath = downloaded
				}
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

// installWithOfficialScript attempts to install Bun using the official installation script
func (b *BunRuntime) installWithOfficialScript() (string, error) {
	binDir := GetDefaultBinDir()
	if err := os.MkdirAll(binDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create bin directory: %w", err)
	}

	var cmd *exec.Cmd
	if goruntime.GOOS == "windows" {
		// PowerShell command for Windows
		cmd = exec.Command("powershell", "-c", "irm bun.sh/install.ps1|iex")
	} else {
		// Unix-like systems
		cmd = exec.Command("bash", "-c", "curl -fsSL https://bun.sh/install | bash")
	}

	// Set environment to install to our custom directory
	cmd.Env = append(os.Environ(), fmt.Sprintf("BUN_INSTALL=%s", binDir))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("official script failed: %w, output: %s", err, string(output))
	}

	// Check if bun was installed
	bunPath := b.getBundledBunPath()
	if _, err := os.Stat(bunPath); err != nil {
		return "", fmt.Errorf("bun not found after official script installation: %w", err)
	}

	return bunPath, nil
}

// downloadBunWithRetry attempts to download Bun with retry mechanism
func (b *BunRuntime) downloadBunWithRetry(maxAttempts int) (string, error) {
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		b.downloadAttempts = attempt
		if attempt > 1 {
			fmt.Printf("Retrying Bun download, attempt %d/%d\n", attempt, maxAttempts)
			time.Sleep(time.Second * time.Duration(attempt))
		}

		path, err := b.downloadBun()
		if err == nil {
			return path, nil
		}
		lastErr = err
	}

	return "", fmt.Errorf("failed to download Bun after %d attempts: %w", maxAttempts, lastErr)
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

// getDownloadURL returns the Bun download URL for current platform with better version detection.
func (b *BunRuntime) getDownloadURL() (string, error) {
	// First try to get latest release info from GitHub API
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/oven-sh/bun/releases/latest")
	if err == nil && resp.StatusCode == http.StatusOK {
		defer resp.Body.Close()

		var release GitHubRelease
		if json.NewDecoder(resp.Body).Decode(&release) == nil {
			// Successfully got release info, now find the right asset
			for _, asset := range release.Assets {
				if b.isCorrectAsset(asset.Name) {
					return asset.URL, nil
				}
			}
		}
	}

	// Fallback to original hardcoded approach if API fails
	/*
		To install Bun vx.x.x
		curl -fsSL https://bun.sh/install | bash
		# or you can use npm
		# npm install -g bun

		Windows:
		powershell -c "irm bun.sh/install.ps1|iex"

		To upgrade to Bun vx.x.x:
		bun upgrade
	*/

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

// isCorrectAsset checks if the asset name matches the current platform
func (b *BunRuntime) isCorrectAsset(name string) bool {
	osArch := ""
	switch goruntime.GOOS {
	case "windows":
		osArch = fmt.Sprintf("windows-%s", goruntime.GOARCH)
	case "darwin":
		osArch = fmt.Sprintf("darwin-%s", goruntime.GOARCH)
	case "linux":
		osArch = fmt.Sprintf("linux-%s", goruntime.GOARCH)
	default:
		return false
	}

	pattern := fmt.Sprintf("bun-%s", osArch)
	return strings.Contains(name, pattern) && strings.HasSuffix(name, ".zip")
}

// extractZip extracts a zip file to the target directory with improved Windows support.
func (b *BunRuntime) extractZip(zipPath, targetDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	var bunFound bool

	for _, f := range r.File {
		// More robust bun executable detection
		if b.isBunExecutable(f.Name) {
			bunFound = true
			dstPath := b.getTargetExecutablePath(f.Name, targetDir)

			if err := os.MkdirAll(filepath.Dir(dstPath), 0755); err != nil {
				return fmt.Errorf("failed to create directory for %s: %w", dstPath, err)
			}

			if err := b.extractFile(f, dstPath); err != nil {
				return fmt.Errorf("failed to extract file: %w", err)
			}

			// Make executable on Unix systems
			if !isWindows() {
				if err := os.Chmod(dstPath, 0755); err != nil {
					return fmt.Errorf("failed to make executable: %w", err)
				}
			}

			// Successfully extracted bun
			return nil
		}
	}

	if !bunFound {
		return fmt.Errorf("bun executable not found in zip. Contents: %s", b.listZipContents(r))
	}

	return fmt.Errorf("bun executable extraction failed")
}

// isBunExecutable checks if a file is likely the bun executable
func (b *BunRuntime) isBunExecutable(name string) bool {
	base := filepath.Base(name)
	return strings.HasPrefix(base, "bun") &&
		(strings.HasSuffix(base, ".exe") || !strings.Contains(base, "."))
}

// getTargetExecutablePath determines where to place the extracted bun executable
func (b *BunRuntime) getTargetExecutablePath(zipPath, targetDir string) string {
	execName := "bun"
	if isWindows() {
		execName = "bun.exe"
	}
	return filepath.Join(targetDir, execName)
}

// extractFile handles the actual file extraction
func (b *BunRuntime) extractFile(f *zip.File, dstPath string) error {
	srcFile, err := f.Open()
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// listZipContents returns a string listing all files in the zip (for debugging)
func (b *BunRuntime) listZipContents(r *zip.ReadCloser) string {
	var names []string
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	return strings.Join(names, ", ")
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
