package skills

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSkillWithDirectories(t *testing.T) {
	// Create temporary skill directory
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "test_skill")
	scriptsDir := filepath.Join(skillDir, "scripts")
	refsDir := filepath.Join(skillDir, "references")
	extraDir := filepath.Join(skillDir, "extra_files")

	os.MkdirAll(scriptsDir, 0755)
	os.MkdirAll(refsDir, 0755)
	os.MkdirAll(extraDir, 0755)

	// Create SKILL.md
	skillMd := `---
name: test_skill
description: Test skill
---
Content`
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644)

	// Create script
	os.WriteFile(filepath.Join(scriptsDir, "test.sh"), []byte("#!/bin/bash\necho test"), 0755)

	// Create reference
	os.WriteFile(filepath.Join(refsDir, "ref.md"), []byte("# Reference"), 0644)

	// Create extra file
	os.WriteFile(filepath.Join(extraDir, "data.json"), []byte(`{"key":"value"}`), 0644)

	// Load skill
	skill, err := loadSkill(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("loadSkill: %v", err)
	}

	// Verify basic fields
	if skill.Name != "test_skill" {
		t.Errorf("expected name 'test_skill', got %q", skill.Name)
	}

	// Verify scripts
	if len(skill.Scripts) != 1 {
		t.Errorf("expected 1 script, got %d", len(skill.Scripts))
	}
	if script, ok := skill.Scripts["test.sh"]; !ok {
		t.Error("test.sh not found in scripts")
	} else if script == "" {
		t.Error("test.sh content is empty")
	}

	// Verify references
	if len(skill.References) != 1 {
		t.Errorf("expected 1 reference, got %d", len(skill.References))
	}
	if ref, ok := skill.References["ref.md"]; !ok {
		t.Error("ref.md not found in references")
	} else if ref == "" {
		t.Error("ref.md content is empty")
	}

	// Verify extra files
	if len(skill.ExtraFiles) != 1 {
		t.Errorf("expected 1 extra file, got %d", len(skill.ExtraFiles))
	}
	if data, ok := skill.ExtraFiles["data.json"]; !ok {
		t.Error("data.json not found in extra files")
	} else if len(data) == 0 {
		t.Error("data.json content is empty")
	}
}

func TestLoadSkillEmptyDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	skillDir := filepath.Join(tmpDir, "empty_skill")
	os.MkdirAll(skillDir, 0755)

	skillMd := `---
name: empty_skill
---
Content`
	os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillMd), 0644)

	skill, err := loadSkill(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("loadSkill: %v", err)
	}

	// Should have empty maps, not nil
	if skill.Scripts == nil {
		t.Error("Scripts should not be nil")
	}
	if skill.References == nil {
		t.Error("References should not be nil")
	}
	if skill.ExtraFiles == nil {
		t.Error("ExtraFiles should not be nil")
	}
}

func TestLoadDirectoryTextFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "file2.md"), []byte("content2"), 0644)
	os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755) // Should be ignored

	result := loadDirectoryTextFiles(tmpDir)

	if len(result) != 2 {
		t.Errorf("expected 2 files, got %d", len(result))
	}
	if result["file1.txt"] != "content1" {
		t.Errorf("expected 'content1', got %q", result["file1.txt"])
	}
	if result["file2.md"] != "content2" {
		t.Errorf("expected 'content2', got %q", result["file2.md"])
	}
}

func TestLoadDirectoryBinaryFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test binary file
	binaryData := []byte{0x00, 0x01, 0x02, 0xFF}
	os.WriteFile(filepath.Join(tmpDir, "binary.bin"), binaryData, 0644)

	result := loadDirectoryBinaryFiles(tmpDir)

	if len(result) != 1 {
		t.Errorf("expected 1 file, got %d", len(result))
	}
	if len(result["binary.bin"]) != 4 {
		t.Errorf("expected 4 bytes, got %d", len(result["binary.bin"]))
	}
}
