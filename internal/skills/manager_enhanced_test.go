package skills

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

func TestManager_CreateSkill(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager()
	cfg := config.SkillsConfig{
		ActiveDir:     "active_skills",
		CustomizedDir: "customized_skills",
	}

	err := mgr.CreateSkill(dir, cfg, "new_skill", "A new skill", "Skill content here.")
	if err != nil {
		t.Fatalf("CreateSkill: %v", err)
	}

	sk := mgr.GetSkill("new_skill")
	if sk == nil {
		t.Fatal("expected skill to be loaded")
	}
	if sk.Name != "new_skill" {
		t.Errorf("name: got %q", sk.Name)
	}
	if !sk.Enabled {
		t.Error("expected skill to be enabled")
	}

	skillPath := filepath.Join(dir, "customized_skills", "new_skill", "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Error("expected SKILL.md file to exist on disk")
	}
}

func TestManager_CreateSkill_EmptyName(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager()
	cfg := config.SkillsConfig{CustomizedDir: "customized_skills"}
	err := mgr.CreateSkill(dir, cfg, "", "desc", "content")
	if err == nil {
		t.Error("expected error for empty name")
	}
}

func TestManager_DeleteSkill(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager()
	cfg := config.SkillsConfig{
		ActiveDir:     "active_skills",
		CustomizedDir: "customized_skills",
	}

	_ = mgr.CreateSkill(dir, cfg, "to_delete", "desc", "content")
	if mgr.GetSkill("to_delete") == nil {
		t.Fatal("skill should exist before delete")
	}

	err := mgr.DeleteSkill("to_delete", true)
	if err != nil {
		t.Fatalf("DeleteSkill: %v", err)
	}
	if mgr.GetSkill("to_delete") != nil {
		t.Error("skill should not exist after delete")
	}

	skillDir := filepath.Join(dir, "customized_skills", "to_delete")
	if _, err := os.Stat(skillDir); !os.IsNotExist(err) {
		t.Error("expected skill directory to be removed from disk")
	}
}

func TestManager_DeleteSkill_NotFound(t *testing.T) {
	mgr := NewManager()
	err := mgr.DeleteSkill("nonexistent", false)
	if err == nil {
		t.Error("expected error for nonexistent skill")
	}
}

func TestManager_ListAllSkills(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager()
	cfg := config.SkillsConfig{
		ActiveDir:     "active_skills",
		CustomizedDir: "customized_skills",
	}

	_ = mgr.CreateSkill(dir, cfg, "skill_a", "", "a")
	_ = mgr.CreateSkill(dir, cfg, "skill_b", "", "b")

	all := mgr.ListAllSkills()
	if len(all) != 2 {
		t.Errorf("expected 2 skills, got %d", len(all))
	}
}

func TestManager_GetSkill_NotFound(t *testing.T) {
	mgr := NewManager()
	if mgr.GetSkill("nonexistent") != nil {
		t.Error("expected nil for nonexistent skill")
	}
}

func TestManager_SyncSkillsToWorkingDir(t *testing.T) {
	configDir := t.TempDir()
	workingDir := t.TempDir()

	srcDir := filepath.Join(configDir, "active_skills", "synced_skill")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("---\nname: synced\n---\ncontent"), 0644); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager()
	cfg := config.SkillsConfig{
		ActiveDir:     "active_skills",
		CustomizedDir: "customized_skills",
	}
	err := mgr.SyncSkillsToWorkingDir(workingDir, configDir, cfg)
	if err != nil {
		t.Fatalf("SyncSkillsToWorkingDir: %v", err)
	}

	dstPath := filepath.Join(workingDir, "active_skills", "synced_skill", "SKILL.md")
	if _, err := os.Stat(dstPath); os.IsNotExist(err) {
		t.Error("expected synced SKILL.md to exist in working dir")
	}
}

func TestManager_SyncSkillsToWorkingDir_NoOverwrite(t *testing.T) {
	configDir := t.TempDir()
	workingDir := t.TempDir()

	srcDir := filepath.Join(configDir, "active_skills", "existing")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "SKILL.md"), []byte("new content"), 0644); err != nil {
		t.Fatal(err)
	}

	dstDir := filepath.Join(workingDir, "active_skills", "existing")
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dstDir, "SKILL.md"), []byte("old content"), 0644); err != nil {
		t.Fatal(err)
	}

	mgr := NewManager()
	cfg := config.SkillsConfig{ActiveDir: "active_skills", CustomizedDir: "customized_skills"}
	_ = mgr.SyncSkillsToWorkingDir(workingDir, configDir, cfg)

	data, _ := os.ReadFile(filepath.Join(dstDir, "SKILL.md"))
	if string(data) != "old content" {
		t.Error("existing file should not be overwritten")
	}
}

func TestManager_ListAvailableSkills(t *testing.T) {
	dir := t.TempDir()
	cfg := config.SkillsConfig{
		ActiveDir:     "active_skills",
		CustomizedDir: "customized_skills",
	}

	for _, name := range []string{"skill_x", "skill_y"} {
		skillDir := filepath.Join(dir, "active_skills", name)
		if err := os.MkdirAll(skillDir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	mgr := NewManager()
	names := mgr.ListAvailableSkills(dir, dir, cfg)
	if len(names) != 2 {
		t.Errorf("expected 2 available skills, got %d", len(names))
	}
}

func TestManager_EnableDisable(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager()
	cfg := config.SkillsConfig{ActiveDir: "active_skills", CustomizedDir: "customized_skills"}
	_ = mgr.CreateSkill(dir, cfg, "toggle", "", "content")

	if err := mgr.DisableSkill("toggle"); err != nil {
		t.Fatalf("DisableSkill: %v", err)
	}
	sk := mgr.GetSkill("toggle")
	if sk.Enabled {
		t.Error("expected disabled")
	}

	if err := mgr.EnableSkill("toggle"); err != nil {
		t.Fatalf("EnableSkill: %v", err)
	}
	sk = mgr.GetSkill("toggle")
	if !sk.Enabled {
		t.Error("expected enabled")
	}
}
