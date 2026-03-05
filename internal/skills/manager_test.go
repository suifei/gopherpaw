package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

func TestManager_LoadSkills_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager()
	err := mgr.LoadSkills(dir, dir, config.SkillsConfig{
		ActiveDir:     "active_skills",
		CustomizedDir: "customized_skills",
	})
	if err != nil {
		t.Fatalf("LoadSkills: %v", err)
	}
	enabled := mgr.GetEnabledSkills()
	if len(enabled) != 0 {
		t.Errorf("expected 0 skills, got %d", len(enabled))
	}
	if mgr.GetSystemPromptAddition() != "" {
		t.Errorf("expected empty addition, got %q", mgr.GetSystemPromptAddition())
	}
}

func TestManager_LoadSkills_WithSkill(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "active_skills", "test_skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := `---
name: test_skill
description: A test skill
---
This is the skill content.`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	mgr := NewManager()
	err := mgr.LoadSkills(dir, dir, config.SkillsConfig{
		ActiveDir:     "active_skills",
		CustomizedDir: "customized_skills",
	})
	if err != nil {
		t.Fatalf("LoadSkills: %v", err)
	}
	enabled := mgr.GetEnabledSkills()
	if len(enabled) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(enabled))
	}
	if enabled[0].Name != "test_skill" {
		t.Errorf("name: got %q", enabled[0].Name)
	}
	if enabled[0].Description != "A test skill" {
		t.Errorf("description: got %q", enabled[0].Description)
	}
	add := mgr.GetSystemPromptAddition()
	if add == "" || len(add) < 20 {
		t.Errorf("expected non-empty addition, got %q", add)
	}
}
