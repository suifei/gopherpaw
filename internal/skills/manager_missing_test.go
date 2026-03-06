package skills

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/suifei/gopherpaw/internal/config"
)

func TestDeriveSkillNameFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "github raw with SKILL.md",
			url:      "https://raw.githubusercontent.com/user/repo/main/skills/test/SKILL.md",
			expected: "test",
		},
		{
			name:     "github blob",
			url:      "https://github.com/user/repo/blob/main/skills/test/SKILL.md",
			expected: "test",
		},
		{
			name:     "simple path",
			url:      "https://example.com/skills/my-skill",
			expected: "my-skill",
		},
		{
			name:     "with .md suffix",
			url:      "https://example.com/skills/my-skill.md",
			expected: "my-skill",
		},
		{
			name:     "complex path",
			url:      "https://example.com/foo/bar/baz/skill.md",
			expected: "skill",
		},
		{
			name:     "default fallback",
			url:      "https://example.com/skills",
			expected: "skills",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := deriveSkillNameFromURL(tt.url)
			if got != tt.expected {
				t.Errorf("deriveSkillNameFromURL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestManager_ImportFromURL_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("---\nname: imported_skill\n---\nImported content"))
	}))
	defer server.Close()

	serverURL := server.URL + "/skills/imported_skill/SKILL.md"

	dir := t.TempDir()
	mgr := NewManager()
	cfg := config.SkillsConfig{
		ActiveDir:     "active_skills",
		CustomizedDir: "customized_skills",
	}

	ctx := context.Background()
	name, err := mgr.ImportFromURL(ctx, serverURL, dir, cfg)
	if err != nil {
		t.Fatalf("ImportFromURL failed: %v", err)
	}
	if name == "" {
		t.Fatal("expected non-empty skill name")
	}

	sk := mgr.GetSkill(name)
	if sk == nil {
		t.Fatal("expected skill to be loaded")
	}
	if !sk.Enabled {
		t.Error("expected skill to be enabled")
	}

	skillPath := filepath.Join(dir, "customized_skills", name, "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Error("expected SKILL.md file to exist")
	}
}

func TestManager_ImportFromURL_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	serverURL := server.URL + "/skills/test/SKILL.md"

	dir := t.TempDir()
	mgr := NewManager()
	cfg := config.SkillsConfig{
		ActiveDir:     "active_skills",
		CustomizedDir: "customized_skills",
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err := mgr.ImportFromURL(ctx, serverURL, dir, cfg)
	if err == nil {
		t.Error("expected error for HTTP 404")
	}
}

func TestManager_ImportFromURL_ContentTooShort(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("short"))
	}))
	defer server.Close()

	serverURL := server.URL + "/skills/test/SKILL.md"

	dir := t.TempDir()
	mgr := NewManager()
	cfg := config.SkillsConfig{
		ActiveDir:     "active_skills",
		CustomizedDir: "customized_skills",
	}

	ctx := context.Background()
	_, err := mgr.ImportFromURL(ctx, serverURL, dir, cfg)
	if err == nil {
		t.Error("expected error for content too short")
	}
}

func TestManager_ImportFromURL_ContextCancel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("---\nname: test\n---\ncontent"))
	}))
	defer server.Close()

	serverURL := server.URL + "/skills/test/SKILL.md"

	dir := t.TempDir()
	mgr := NewManager()
	cfg := config.SkillsConfig{
		ActiveDir:     "active_skills",
		CustomizedDir: "customized_skills",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := mgr.ImportFromURL(ctx, serverURL, dir, cfg)
	if err == nil {
		t.Error("expected error for context cancellation")
	}
}

func TestManager_ImportFromURL_InvalidContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("---\nname: test_skill\n---\nThis is valid content for a skill file"))
	}))
	defer server.Close()

	serverURL := server.URL + "/skills/test_skill/SKILL.md"

	dir := t.TempDir()
	mgr := NewManager()
	cfg := config.SkillsConfig{
		ActiveDir:     "active_skills",
		CustomizedDir: "customized_skills",
	}

	ctx := context.Background()
	name, err := mgr.ImportFromURL(ctx, serverURL, dir, cfg)
	if err != nil {
		t.Fatalf("ImportFromURL should succeed with valid content: %v", err)
	}
	if name == "" {
		t.Fatal("expected non-empty skill name")
	}

	sk := mgr.GetSkill(name)
	if sk == nil {
		t.Fatal("expected skill to be loaded")
	}
	if sk.Name != "test_skill" {
		t.Errorf("expected skill name test_skill, got %q", sk.Name)
	}
}

func TestManager_LoadSkills_DuplicatePaths(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "active_skills", "test_skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := `---
name: test_skill
description: A test skill
---
This is the content`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	mgr := NewManager()
	err := mgr.LoadSkills(dir, dir, config.SkillsConfig{
		ActiveDir:     "active_skills",
		CustomizedDir: "active_skills",
	})
	if err != nil {
		t.Fatalf("LoadSkills should handle duplicate paths: %v", err)
	}

	enabled := mgr.GetEnabledSkills()
	if len(enabled) != 1 {
		t.Errorf("expected 1 skill (duplicate paths skipped), got %d", len(enabled))
	}
}

func TestManager_LoadSkills_SkillLoadError(t *testing.T) {
	dir := t.TempDir()
	validDir := filepath.Join(dir, "active_skills", "valid_skill")
	if err := os.MkdirAll(validDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(validDir, "SKILL.md"), []byte("---\nname: valid\n---\nc"), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	anotherValidDir := filepath.Join(dir, "active_skills", "another_valid")
	if err := os.MkdirAll(anotherValidDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(anotherValidDir, "SKILL.md"), []byte("---\nname: another\n---\nc"), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	mgr := NewManager()
	err := mgr.LoadSkills(dir, dir, config.SkillsConfig{
		ActiveDir:     "active_skills",
		CustomizedDir: "customized_skills",
	})
	if err != nil {
		t.Fatalf("LoadSkills should succeed: %v", err)
	}

	enabled := mgr.GetEnabledSkills()
	if len(enabled) != 2 {
		t.Errorf("expected 2 valid skills, got %d", len(enabled))
	}
}

func TestLoadSkill_NoFrontMatter(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test_skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := `Just plain content without front matter`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	sk, err := loadSkill(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("loadSkill failed: %v", err)
	}
	if sk.Name != "test_skill" {
		t.Errorf("expected name from directory, got %q", sk.Name)
	}
	if sk.Content != content {
		t.Error("content mismatch")
	}
}

func TestLoadSkill_WithFrontMatter(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "my_skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := `---
name: custom_name
description: Custom description
---
The actual skill content here`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	sk, err := loadSkill(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("loadSkill failed: %v", err)
	}
	if sk.Name != "custom_name" {
		t.Errorf("expected name from front matter, got %q", sk.Name)
	}
	if sk.Description != "Custom description" {
		t.Errorf("expected description from front matter, got %q", sk.Description)
	}
	if !strings.Contains(sk.Content, "The actual skill content") {
		t.Error("content should include main content")
	}
}

func TestLoadSkill_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	skillDir := filepath.Join(dir, "test_skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	content := `---
name: test
invalid yaml: [unclosed
---
content`
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}

	sk, err := loadSkill(filepath.Join(skillDir, "SKILL.md"))
	if err != nil {
		t.Fatalf("loadSkill should handle invalid YAML gracefully: %v", err)
	}
	if sk.Name != "test_skill" {
		t.Errorf("expected fallback to directory name, got %q", sk.Name)
	}
}

func TestManager_DeleteSkill_WithoutDisk(t *testing.T) {
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

	skillPath := filepath.Join(dir, "customized_skills", "to_delete", "SKILL.md")
	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Fatal("skill file should exist on disk before delete")
	}

	err := mgr.DeleteSkill("to_delete", false)
	if err != nil {
		t.Fatalf("DeleteSkill: %v", err)
	}
	if mgr.GetSkill("to_delete") != nil {
		t.Error("skill should not exist after delete")
	}

	if _, err := os.Stat(skillPath); os.IsNotExist(err) {
		t.Error("skill file should still exist on disk when removeFromDisk=false")
	}
}

func TestManager_SyncSkillsToWorkingDir_SourceNotExist(t *testing.T) {
	configDir := t.TempDir()
	workingDir := t.TempDir()

	mgr := NewManager()
	cfg := config.SkillsConfig{
		ActiveDir:     "active_skills",
		CustomizedDir: "customized_skills",
	}
	err := mgr.SyncSkillsToWorkingDir(workingDir, configDir, cfg)
	if err != nil {
		t.Fatalf("SyncSkillsToWorkingDir should handle non-existent source: %v", err)
	}
}

func TestManager_ListAvailableSkills_DirectoryNotExist(t *testing.T) {
	dir := t.TempDir()
	cfg := config.SkillsConfig{
		ActiveDir:     "active_skills",
		CustomizedDir: "customized_skills",
	}

	mgr := NewManager()
	names := mgr.ListAvailableSkills(dir, dir, cfg)
	if len(names) != 0 {
		t.Errorf("expected 0 available skills, got %d", len(names))
	}
}

func TestManager_GetSystemPromptAddition_MultipleSkills(t *testing.T) {
	dir := t.TempDir()
	mgr := NewManager()
	cfg := config.SkillsConfig{
		ActiveDir:     "active_skills",
		CustomizedDir: "customized_skills",
	}

	_ = mgr.CreateSkill(dir, cfg, "skill1", "First skill", "Content 1")
	_ = mgr.CreateSkill(dir, cfg, "skill2", "Second skill", "Content 2")
	_ = mgr.CreateSkill(dir, cfg, "skill3", "", "Content 3")

	add := mgr.GetSystemPromptAddition()
	if !strings.Contains(add, "skill1") || !strings.Contains(add, "skill2") || !strings.Contains(add, "skill3") {
		t.Error("system prompt addition should include all enabled skills")
	}

	_ = mgr.DisableSkill("skill2")
	add = mgr.GetSystemPromptAddition()
	if strings.Contains(add, "skill2") {
		t.Error("disabled skill should not be in system prompt addition")
	}
}
