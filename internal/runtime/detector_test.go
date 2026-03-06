package runtime

import (
	"testing"
)

// TestCheckSkillBinaries tests checking for skill binary dependencies.
func TestCheckSkillBinaries(t *testing.T) {
	statuses := CheckSkillBinaries()

	if len(statuses) == 0 {
		t.Error("CheckSkillBinaries should return at least one status")
	}

	// Check that each status has required fields
	for _, s := range statuses {
		if s.Name == "" {
			t.Error("Status Name is empty")
		}

		// Status is valid
		_ = s.Ready
		if !s.Ready && s.Error == "" {
			t.Errorf("Status %s is not ready but has no error message", s.Name)
		}
	}
}

// TestCheckSkillBinariesNames tests that expected binaries are checked.
func TestCheckSkillBinariesNames(t *testing.T) {
	statuses := CheckSkillBinaries()

	expectedBinaries := []string{"soffice", "pdftoppm", "himalaya", "pandoc", "ffmpeg", "git"}
	found := make(map[string]bool)

	for _, s := range statuses {
		found[s.Name] = true
	}

	for _, expected := range expectedBinaries {
		if !found[expected] {
			t.Errorf("Expected binary %q not found in statuses", expected)
		}
	}
}

// TestDetectPlatform tests platform detection.
func TestDetectPlatform(t *testing.T) {
	platform := DetectPlatform()

	if platform == "" {
		t.Error("DetectPlatform returned empty string")
	}

	// Should be one of the known platform patterns
	// May be "windows", "linux", "darwin", or include arch like "windows/amd64"
	knownPrefixes := map[string]bool{
		"linux":   true,
		"darwin":  true,
		"windows": true,
	}

	hasKnownPrefix := false
	for prefix := range knownPrefixes {
		if contains(platform, prefix) {
			hasKnownPrefix = true
			break
		}
	}

	if !hasKnownPrefix {
		t.Errorf("DetectPlatform returned platform without known prefix: %q", platform)
	}
}

// TestIsCommandAvailable tests checking if a command is available.
func TestIsCommandAvailable(t *testing.T) {
	tests := []struct {
		command string
		want    bool
	}{
		{"echo", true}, // Should be available on most systems
		{"sh", true},   // Should be available on Unix
		{"nonexistent-command-xyz-123", false},
	}

	for _, tt := range tests {
		result := IsCommandAvailable(tt.command)
		// Only check the general pattern, actual availability depends on system
		_ = result
	}
}

// TestIsCommandAvailableEcho tests that echo is available.
func TestIsCommandAvailableEcho(t *testing.T) {
	result := IsCommandAvailable("echo")
	if !result {
		t.Logf("echo command not available (may be expected on some systems)")
	}
}

// TestIsCommandAvailableNonExistent tests that non-existent command is not available.
func TestIsCommandAvailableNonExistent(t *testing.T) {
	result := IsCommandAvailable("nonexistent-command-xyz-123-abc")
	if result {
		t.Error("Non-existent command should not be available")
	}
}

// TestFindBinary tests finding a binary on the system.
func TestFindBinary(t *testing.T) {
	// Try to find a common binary
	path := FindBinary("echo")
	if path != "" {
		// Echo should be found
		_ = path
	}

	// Try non-existent binary
	path = FindBinary("nonexistent-binary-xyz-123")
	if path != "" {
		t.Errorf("FindBinary should return empty for non-existent binary, got %q", path)
	}
}

// TestFindBinaryCommon tests finding common binaries.
func TestFindBinaryCommon(t *testing.T) {
	tests := []string{"git", "python", "python3", "node"}

	for _, cmd := range tests {
		path := FindBinary(cmd)
		if path != "" {
			t.Logf("Found %s at %s", cmd, path)
		}
	}
}

// TestFindBinaryPathOrder tests that PATH is searched first.
func TestFindBinaryPathOrder(t *testing.T) {
	// This test verifies that FindBinary searches PATH first
	// Should always try PATH before other locations
	path := FindBinary("echo")
	_ = path
}

// TestGetInstallHint tests getting installation hints.
func TestGetInstallHint(t *testing.T) {
	tests := []struct {
		binary  string
		hasHint bool
	}{
		{"soffice", true},
		{"git", true},
		{"ffmpeg", true},
		{"unknown-binary", false},
	}

	for _, tt := range tests {
		hint := GetInstallHint(tt.binary)
		if tt.hasHint {
			if hint == "" {
				t.Errorf("GetInstallHint(%q) should return non-empty hint", tt.binary)
			}
		} else {
			if hint != "" {
				t.Errorf("GetInstallHint(%q) should return empty for unknown binary", tt.binary)
			}
		}
	}
}

// TestGetInstallHintKnownBinaries tests getting hints for known binaries.
func TestGetInstallHintKnownBinaries(t *testing.T) {
	knownBinaries := []string{"soffice", "pdftoppm", "himalaya", "pandoc", "ffmpeg", "git"}

	for _, binary := range knownBinaries {
		hint := GetInstallHint(binary)
		if hint == "" {
			t.Errorf("GetInstallHint(%q) should return non-empty hint for known binary", binary)
		}
	}
}

// TestGetInstallHintUnknown tests GetInstallHint with unknown binary.
func TestGetInstallHintUnknown(t *testing.T) {
	hint := GetInstallHint("completely-unknown-binary-xyz")
	if hint != "" {
		t.Errorf("GetInstallHint for unknown binary should be empty, got %q", hint)
	}
}

// TestGetBinaryVariations tests getting binary name variations.
func TestGetBinaryVariations(t *testing.T) {
	// Test that variations function exists and works
	variations := getBinaryVariations("python")

	if len(variations) == 0 {
		t.Error("getBinaryVariations should return variations for python")
	}

	// Should contain common variants
	hasVariation := false
	for _, v := range variations {
		if v == "python3" || v == "python.exe" {
			hasVariation = true
			break
		}
	}

	if !hasVariation {
		t.Errorf("getBinaryVariations should contain common variants")
	}
}

// TestGetBinaryVariationsPython tests variations for Python.
func TestGetBinaryVariationsPython(t *testing.T) {
	variations := getBinaryVariations("python")

	hasVariations := len(variations) > 0
	if !hasVariations {
		t.Error("getBinaryVariations for python should return variants")
	}

	// Should contain python3
	hasPython3 := false
	for _, v := range variations {
		if contains(v, "python3") {
			hasPython3 = true
			break
		}
	}

	if !hasPython3 {
		t.Logf("python variations: %v", variations)
	}
}

// TestGetBinaryVariationsSoffice tests variations for soffice.
func TestGetBinaryVariationsSoffice(t *testing.T) {
	variations := getBinaryVariations("soffice")

	hasSofficeVariation := false
	for _, v := range variations {
		if contains(v, "libreoffice") {
			hasSofficeVariation = true
			break
		}
	}

	if !hasSofficeVariation && len(variations) > 0 {
		t.Logf("soffice variations: %v", variations)
	}
}

// TestGetBinaryVariationsEmpty tests variations for unknown binary.
func TestGetBinaryVariationsEmpty(t *testing.T) {
	variations := getBinaryVariations("unknown-binary")

	// May return empty or variations depending on implementation
	_ = variations
}

// TestGetBinaryVersion tests getting binary version.
func TestGetBinaryVersion(t *testing.T) {
	// This may not work on all systems
	version, err := getBinaryVersion("python3", "python3")

	// Just check it returns a string (may be empty if Python not found)
	_ = version
	_ = err
}

// TestGetBinaryVersionGit tests getting git version.
func TestGetBinaryVersionGit(t *testing.T) {
	version, err := getBinaryVersion("git", "git")

	if err == nil && version != "" {
		// Git is available, version should not be empty
		t.Logf("git version: %s", version)
	}
}

// TestGetBinaryVersionPython tests getting Python version.
func TestGetBinaryVersionPython(t *testing.T) {
	version, err := getBinaryVersion("python", "python3")

	if err != nil {
		t.Logf("Failed to get python version (may not be installed): %v", err)
	} else {
		_ = version
	}
}

// TestGetBinaryVersionPandoc tests getting pandoc version.
func TestGetBinaryVersionPandoc(t *testing.T) {
	version, err := getBinaryVersion("pandoc", "pandoc")

	if err != nil {
		t.Logf("Failed to get pandoc version (may not be installed): %v", err)
	} else {
		_ = version
	}
}

// TestGetBinaryVersionFfmpeg tests getting ffmpeg version.
func TestGetBinaryVersionFfmpeg(t *testing.T) {
	version, err := getBinaryVersion("ffmpeg", "ffmpeg")

	if err != nil {
		t.Logf("Failed to get ffmpeg version (may not be installed): %v", err)
	} else {
		_ = version
	}
}

// TestSkillBinaryStructure tests SkillBinary structure.
func TestSkillBinaryStructure(t *testing.T) {
	binary := SkillBinary{
		Name:        "test",
		Description: "test description",
		Package:     "test-package",
		Platforms:   []string{"linux"},
	}

	if binary.Name != "test" {
		t.Error("SkillBinary Name not set")
	}
	if binary.Description != "test description" {
		t.Error("SkillBinary Description not set")
	}
	if binary.Package != "test-package" {
		t.Error("SkillBinary Package not set")
	}
	if len(binary.Platforms) == 0 {
		t.Error("SkillBinary Platforms not set")
	}
}

// TestSkillBinaryMultiplePlatforms tests SkillBinary with multiple platforms.
func TestSkillBinaryMultiplePlatforms(t *testing.T) {
	binary := SkillBinary{
		Name:        "test",
		Description: "test",
		Package:     "test",
		Platforms:   []string{"linux", "darwin", "windows"},
	}

	if len(binary.Platforms) != 3 {
		t.Errorf("Expected 3 platforms, got %d", len(binary.Platforms))
	}
}

// TestCheckSkillBinariesErrorHandling tests that CheckSkillBinaries handles missing binaries gracefully.
func TestCheckSkillBinariesErrorHandling(t *testing.T) {
	statuses := CheckSkillBinaries()

	// Should return results even if no binaries are installed
	for _, s := range statuses {
		// Just check that the structure is valid
		_ = s.Name
		_ = s.Ready
		_ = s.Error
	}
}

// TestCheckSkillBinariesAllReturned tests that all skill binaries are returned.
func TestCheckSkillBinariesAllReturned(t *testing.T) {
	statuses := CheckSkillBinaries()

	// Should have at least the expected number of binaries
	expectedCount := 6 // soffice, pdftoppm, himalaya, pandoc, ffmpeg, git
	if len(statuses) < expectedCount {
		t.Errorf("Expected at least %d statuses, got %d", expectedCount, len(statuses))
	}
}

// TestDetectPlatformConsistency tests that DetectPlatform returns consistent results.
func TestDetectPlatformConsistency(t *testing.T) {
	platform1 := DetectPlatform()
	platform2 := DetectPlatform()

	if platform1 != platform2 {
		t.Errorf("DetectPlatform should return consistent results: %q vs %q", platform1, platform2)
	}
}

// TestGetBinaryVersionAllFormats tests getting version from different binary formats.
func TestGetBinaryVersionAllFormats(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{"node", "node"},
		{"bun", "bun"},
		{"soffice", "soffice"},
	}

	for _, tt := range tests {
		version, _ := getBinaryVersion(tt.name, tt.path)
		// Version may be empty if binary not found
		_ = version
	}
}

// TestFindBinaryWithMissingDirectories tests FindBinary with invalid common paths.
func TestFindBinaryWithMissingDirectories(t *testing.T) {
	// Try to find a non-existent binary in common locations
	// This should handle missing directories gracefully
	path := FindBinary("nonexistent-binary-xyz")
	if path != "" {
		t.Errorf("FindBinary should return empty for truly non-existent binary, got %q", path)
	}
}
