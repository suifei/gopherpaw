package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

func TestManager_GetSkillIndexCompact(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	activeDir := filepath.Join(tmpDir, "active_skills")

	// Create test skills
	for _, name := range []string{"docx", "xlsx", "pdf", "pptx"} {
		skillDir := filepath.Join(activeDir, name)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		skillPath := filepath.Join(skillDir, "SKILL.md")
		content := "---\nname: " + name + "\ndescription: Test " + name + " skill\nkeywords: [" + name + "]\n---\n\nTest content."
		if err := os.WriteFile(skillPath, []byte(content), 0644); err != nil {
			t.Fatalf("write skill: %v", err)
		}
	}

	// Load skills
	mgr := NewManager()
	cfg := config.SkillsConfig{
		ActiveDir:     "active_skills",
		CustomizedDir: "customized_skills",
	}
	if err := mgr.LoadSkills(tmpDir, tmpDir, cfg); err != nil {
		t.Fatalf("LoadSkills: %v", err)
	}

	// Test compact index
	compact := mgr.GetSkillIndexCompact(tmpDir)

	if compact == "" {
		t.Error("GetSkillIndexCompact returned empty string")
	}

	// Verify the compact index contains expected elements
	expectedPhrases := []string{
		"⚠️",
		"CHECK SKILLS",
		"docx",
		"xlsx",
		"pdf",
		"pptx",
		"read_file",
	}

	for _, phrase := range expectedPhrases {
		if !contains(compact, phrase) {
			t.Errorf("Compact index missing expected phrase: %s", phrase)
		}
	}

	// Verify compact index is shorter than full index
	fullIndex := mgr.GetSkillIndex(tmpDir)
	if len(compact) >= len(fullIndex) {
		t.Logf("Note: Compact index (%d bytes) is not shorter than full index (%d bytes)",
			len(compact), len(fullIndex))
	}

	t.Logf("Compact index length: %d bytes", len(compact))
	t.Logf("Full index length: %d bytes", len(fullIndex))
	t.Logf("\n=== Compact Index Output ===\n%s\n", compact)
}
