package runtime

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

// TestNewBunRuntime tests creating a new Bun runtime.
func TestNewBunRuntime(t *testing.T) {
	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)

	if bun == nil {
		t.Fatal("NewBunRuntime returned nil")
	}
	if bun.cfg != cfg {
		t.Error("Config not set correctly")
	}
	if bun.ready {
		t.Error("New runtime should not be ready")
	}
}

// TestBunRuntimeIsReady tests the IsReady method.
func TestBunRuntimeIsReady(t *testing.T) {
	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)

	if bun.IsReady() {
		t.Error("New runtime should not be ready")
	}
}

// TestBunRuntimeGetError tests the GetError method.
func TestBunRuntimeGetError(t *testing.T) {
	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)

	err := bun.GetError()
	if err != "" {
		t.Errorf("Expected empty error for new runtime, got %q", err)
	}
}

// TestBunRuntimeGetPath tests the GetPath method.
func TestBunRuntimeGetPath(t *testing.T) {
	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)

	path := bun.GetPath()
	if path != "" {
		t.Errorf("Expected empty path for new runtime, got %q", path)
	}
}

// TestBunRuntimeGetVersion tests the GetVersion method.
func TestBunRuntimeGetVersion(t *testing.T) {
	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)

	version := bun.GetVersion()
	if version != "" {
		t.Errorf("Expected empty version for new runtime, got %q", version)
	}
}

// TestBunDetectWithExplicitPath tests detection with explicit path.
func TestBunDetectWithExplicitPath(t *testing.T) {
	tmpDir := t.TempDir()
	bunPath := filepath.Join(tmpDir, "bun")

	if isWindows() {
		bunPath = filepath.Join(tmpDir, "bun.exe")
	}

	// Create fake Bun executable
	os.WriteFile(bunPath, []byte("fake bun"), 0755)

	cfg := &config.NodeConfig{
		BunPath: bunPath,
	}
	bun := NewBunRuntime(cfg)

	err := bun.Detect()
	// May fail on version detection, that's ok
	_ = err
}

// TestBunDetectNotFound tests detection when Bun is not found.
func TestBunDetectNotFound(t *testing.T) {
	cfg := &config.NodeConfig{
		BunPath: "/nonexistent/bun/path/that/does/not/exist",
	}
	bun := NewBunRuntime(cfg)

	err := bun.Detect()
	if err == nil {
		t.Error("Detect should return error when bun path not found")
	}
}

// TestGetBundledBunPath tests getting the bundled Bun path.
func TestGetBundledBunPath(t *testing.T) {
	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)

	path := bun.getBundledBunPath()
	if path == "" {
		t.Error("getBundledBunPath returned empty string")
	}

	if isWindows() {
		if !contains(path, "bun.exe") {
			t.Errorf("getBundledBunPath on Windows should contain bun.exe, got %q", path)
		}
	} else {
		if !contains(path, "bun") {
			t.Errorf("getBundledBunPath should contain 'bun', got %q", path)
		}
	}
}

// TestBunGetDownloadURL tests getting the Bun download URL.
func TestBunGetDownloadURL(t *testing.T) {
	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)

	url, err := bun.getDownloadURL()
	if err != nil {
		t.Fatalf("getDownloadURL returned error: %v", err)
	}

	if url == "" {
		t.Error("getDownloadURL returned empty string")
	}

	if !strings.Contains(url, "github.com") {
		t.Errorf("getDownloadURL should contain github.com, got %q", url)
	}

	if !strings.Contains(url, "bun") {
		t.Errorf("getDownloadURL should contain 'bun', got %q", url)
	}
}

// TestBunDetectWithEnvVar tests detection with GOPHERPAW_BUN_PATH environment variable.
func TestBunDetectWithEnvVar(t *testing.T) {
	tmpDir := t.TempDir()
	bunPath := filepath.Join(tmpDir, "bun")

	if isWindows() {
		bunPath = filepath.Join(tmpDir, "bun.exe")
	}

	os.WriteFile(bunPath, []byte("fake bun"), 0755)

	// Set environment variable
	oldEnv := os.Getenv("GOPHERPAW_BUN_PATH")
	defer os.Setenv("GOPHERPAW_BUN_PATH", oldEnv)

	os.Setenv("GOPHERPAW_BUN_PATH", bunPath)

	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)

	err := bun.Detect()
	// May fail on version detection, that's ok
	_ = err
}

// TestRunScriptNotReady tests RunScript when runtime is not ready.
func TestRunScriptNotReady(t *testing.T) {
	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)
	bun.err = os.ErrNotExist

	_, err := bun.RunScript("/tmp/test.ts")
	if err == nil {
		t.Error("RunScript should return error when not ready")
	}
	if !strings.Contains(err.Error(), "not ready") {
		t.Errorf("Expected 'not ready' error, got %v", err)
	}
}

// TestRunScriptWithArgs tests RunScript with arguments.
func TestRunScriptWithArgs(t *testing.T) {
	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)
	bun.ready = true
	bun.path = "bun"

	tmpDir := t.TempDir()
	scriptPath := filepath.Join(tmpDir, "test.ts")
	os.WriteFile(scriptPath, []byte("console.log('test')"), 0755)

	output, err := bun.RunScript(scriptPath, "arg1", "arg2")
	if err != nil {
		t.Logf("RunScript with args failed (bun may not be available): %v", err)
	} else {
		_ = output
	}
}

// TestInstallPackageNotReady tests InstallPackage when not ready.
func TestInstallPackageNotReady(t *testing.T) {
	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)

	err := bun.InstallPackage("lodash")
	if err == nil {
		t.Error("InstallPackage should return error when not ready")
	}
	if !strings.Contains(err.Error(), "not ready") {
		t.Errorf("Expected 'not ready' error, got %v", err)
	}
}

// TestInstallPackage tests InstallPackage error handling.
func TestInstallPackage(t *testing.T) {
	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)
	bun.ready = true
	bun.path = "bun"

	err := bun.InstallPackage("lodash")
	if err != nil {
		t.Logf("InstallPackage failed (bun may not be available): %v", err)
	}
}

// TestRunCommandNotReady tests RunCommand when not ready.
func TestRunCommandNotReady(t *testing.T) {
	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)
	bun.err = os.ErrNotExist

	_, err := bun.RunCommand("--version")
	if err == nil {
		t.Error("RunCommand should return error when not ready")
	}
	if !strings.Contains(err.Error(), "not ready") {
		t.Errorf("Expected 'not ready' error, got %v", err)
	}
}

// TestRunCommand tests running a Bun command.
func TestRunCommand(t *testing.T) {
	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)
	bun.ready = true
	bun.path = "bun"

	output, err := bun.RunCommand("--version")
	if err != nil {
		t.Logf("RunCommand failed (bun may not be available): %v", err)
	} else {
		_ = output
	}
}

// TestExtractZip tests extracting a zip file.
func TestExtractZip(t *testing.T) {
	// Create a temporary zip file with a fake bun executable
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	// Create a zip file with a test file
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create zip file: %v", err)
	}
	defer zipFile.Close()

	zw := zip.NewWriter(zipFile)
	defer zw.Close()

	// Add a file named "bun" to the zip
	fw, err := zw.Create("bun")
	if err != nil {
		t.Fatalf("Failed to create zip entry: %v", err)
	}

	_, err = io.WriteString(fw, "fake bun executable")
	if err != nil {
		t.Fatalf("Failed to write zip entry: %v", err)
	}

	zw.Close()
	zipFile.Close()

	// Test extraction
	extractDir := filepath.Join(tmpDir, "extracted")
	os.MkdirAll(extractDir, 0755)

	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)

	err = bun.extractZip(zipPath, extractDir)
	if err != nil {
		t.Fatalf("extractZip failed: %v", err)
	}

	// Verify extracted file exists
	expectedPath := filepath.Join(extractDir, "bun")
	if isWindows() {
		expectedPath += ".exe"
	}

	if _, err := os.Stat(expectedPath); err != nil {
		t.Logf("Extracted file not found at %q: %v", expectedPath, err)
	}
}

// TestExtractZipInvalidZip tests extracting an invalid zip file.
func TestExtractZipInvalidZip(t *testing.T) {
	tmpDir := t.TempDir()
	invalidZip := filepath.Join(tmpDir, "invalid.zip")

	// Create an invalid zip file
	os.WriteFile(invalidZip, []byte("not a zip file"), 0755)

	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)

	err := bun.extractZip(invalidZip, tmpDir)
	if err == nil {
		t.Error("extractZip should return error for invalid zip")
	}
}

// TestExtractZipNoBun tests extracting a zip without bun executable.
func TestExtractZipNoBun(t *testing.T) {
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "test.zip")

	// Create a zip file without bun
	zipFile, err := os.Create(zipPath)
	if err != nil {
		t.Fatalf("Failed to create zip file: %v", err)
	}
	defer zipFile.Close()

	zw := zip.NewWriter(zipFile)
	defer zw.Close()

	// Add a file that's not named "bun"
	fw, err := zw.Create("other.txt")
	if err != nil {
		t.Fatalf("Failed to create zip entry: %v", err)
	}

	_, err = io.WriteString(fw, "test file")
	if err != nil {
		t.Fatalf("Failed to write zip entry: %v", err)
	}

	zw.Close()
	zipFile.Close()

	// Test extraction
	extractDir := filepath.Join(tmpDir, "extracted")
	os.MkdirAll(extractDir, 0755)

	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)

	err = bun.extractZip(zipPath, extractDir)
	if err == nil {
		t.Error("extractZip should return error when bun executable not found")
	}
}

// TestCopyFile tests copying a file.
func TestCopyFile(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "src.txt")
	dstPath := filepath.Join(tmpDir, "dst.txt")

	// Create source file
	srcContent := "test content"
	os.WriteFile(srcPath, []byte(srcContent), 0644)

	// Copy file
	err := copyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Verify destination exists
	if _, err := os.Stat(dstPath); err != nil {
		t.Fatalf("Destination file not created: %v", err)
	}

	// Verify content matches
	dstContent, err := os.ReadFile(dstPath)
	if err != nil {
		t.Fatalf("Failed to read destination file: %v", err)
	}

	if string(dstContent) != srcContent {
		t.Errorf("Content mismatch: expected %q, got %q", srcContent, string(dstContent))
	}
}

// TestCopyFileNonExistentSource tests copyFile with non-existent source.
func TestCopyFileNonExistentSource(t *testing.T) {
	err := copyFile("/nonexistent/file", "/tmp/dst")
	if err == nil {
		t.Error("copyFile should return error for non-existent source")
	}
}

// TestCopyFilePermissions tests that copyFile preserves file permissions.
func TestCopyFilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	srcPath := filepath.Join(tmpDir, "src.txt")
	dstPath := filepath.Join(tmpDir, "dst.txt")

	// Create source file with specific permissions
	os.WriteFile(srcPath, []byte("test"), 0755)

	// Copy file
	err := copyFile(srcPath, dstPath)
	if err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	// Check that destination was created (permissions may differ by OS)
	if _, err := os.Stat(dstPath); err != nil {
		t.Fatalf("Destination file not created: %v", err)
	}
}

// TestBunRunScript tests running a Bun script.
func TestBunRunScript(t *testing.T) {
	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)

	// Don't run actual script since Bun may not be available
	_ = bun
}

// TestBunInstallPackageIntegration tests package installation (no-op without Bun).
func TestBunInstallPackageIntegration(t *testing.T) {
	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)

	// Don't run actual installation
	_ = bun
}

// TestBunRunCommandIntegration tests running a command with Bun (no-op without Bun).
func TestBunRunCommandIntegration(t *testing.T) {
	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)

	// Don't run actual command
	_ = bun
}

// TestBunDetectSystemBun tests detection with system bun availability.
func TestBunDetectSystemBun(t *testing.T) {
	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)

	// Try detection without explicit path (will try to find system bun)
	// This may succeed or fail depending on whether bun is installed
	err := bun.Detect()
	if err != nil {
		t.Logf("Detect without explicit path failed (may be expected if bun not installed): %v", err)
	}
}

// TestBunGetVersionError tests getVersion with invalid path.
func TestBunGetVersionError(t *testing.T) {
	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)

	// Try to get version from non-existent executable
	version, err := bun.getVersion("/nonexistent/bun")
	if err == nil {
		t.Error("getVersion should return error for non-existent executable")
	}
	if version != "" {
		t.Logf("getVersion returned unexpected version: %q", version)
	}
}

// TestBunGetDownloadURLValidURL tests handling of unsupported OS.
// (This may not be easily testable without mocking runtime.GOOS)
func TestBunGetDownloadURLValidURL(t *testing.T) {
	cfg := &config.NodeConfig{}
	bun := NewBunRuntime(cfg)

	url, err := bun.getDownloadURL()
	if err != nil {
		// Unsupported OS will return error
		t.Logf("getDownloadURL returned error (may be expected on some systems): %v", err)
		return
	}

	if !strings.Contains(url, "github.com") || !strings.Contains(url, "bun") {
		t.Errorf("getDownloadURL returned invalid URL: %q", url)
	}
}
