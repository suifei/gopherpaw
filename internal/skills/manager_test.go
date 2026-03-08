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

func TestManager_GetSkillIndex(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "active_skills", "docx")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := `---
name: docx
description: Word document processing skill
keywords:
  - word
  - docx
  - document
---
## Quick Start

This is the detailed skill content that should NOT appear in the index.`
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

	index := mgr.GetSkillIndex(dir)

	// Verify format - should contain warning box
	if !contains(index, "CRITICAL") {
		t.Error("missing 'CRITICAL' warning header")
	}

	// Should contain skill name
	if !contains(index, "docx") {
		t.Error("missing 'docx' skill name")
	}

	// Should contain description
	if !contains(index, "Word document processing") {
		t.Error("missing skill description")
	}

	// Should contain read instruction
	if !contains(index, "read_file") && !contains(index, "read its SKILL.md") {
		t.Error("missing read instruction")
	}

	// Should contain path reference
	if !contains(index, "SKILL.md") {
		t.Error("missing SKILL.md path reference")
	}

	// Should NOT contain full skill content (key sections from detailed content)
	if contains(index, "## Quick Start") {
		t.Error("index should not contain full skill content like '## Quick Start'")
	}

	// Should contain path to SKILL.md
	if !contains(index, "/") && !contains(index, "SKILL.md") {
		t.Error("index should contain path reference to SKILL.md")
	}
}

func TestManager_GetSkillIndex_MultipleSkills(t *testing.T) {
	dir := t.TempDir()
	// Create docx skill
	docxDir := filepath.Join(dir, "active_skills", "docx")
	if err := os.MkdirAll(docxDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(docxDir, "SKILL.md"), []byte("---\nname: docx\ndescription: Word docs\n---\nContent here"), 0644); err != nil {
		t.Fatalf("write docx: %v", err)
	}

	// Create pptx skill
	pptxDir := filepath.Join(dir, "active_skills", "pptx")
	if err := os.MkdirAll(pptxDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pptxDir, "SKILL.md"), []byte("---\nname: pptx\ndescription: PowerPoint\n---\nContent here"), 0644); err != nil {
		t.Fatalf("write pptx: %v", err)
	}

	// Create disabled skill
	pdfDir := filepath.Join(dir, "active_skills", "pdf")
	if err := os.MkdirAll(pdfDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(pdfDir, "SKILL.md"), []byte("---\nname: pdf\ndescription: PDF docs\n---\nContent here"), 0644); err != nil {
		t.Fatalf("write pdf: %v", err)
	}

	mgr := NewManager()
	if err := mgr.LoadSkills(dir, dir, config.SkillsConfig{
		ActiveDir:     "active_skills",
		CustomizedDir: "customized_skills",
	}); err != nil {
		t.Fatalf("LoadSkills: %v", err)
	}

	// Disable pdf skill
	if err := mgr.DisableSkill("pdf"); err != nil {
		t.Fatalf("DisableSkill: %v", err)
	}

	index := mgr.GetSkillIndex(dir)

	// Should contain enabled skills
	if !contains(index, "docx") {
		t.Error("missing 'docx' in index")
	}
	if !contains(index, "pptx") {
		t.Error("missing 'pptx' in index")
	}

	// Should NOT contain disabled skill
	if contains(index, "pdf") {
		t.Error("disabled skill 'pdf' should not appear in index")
	}
}
