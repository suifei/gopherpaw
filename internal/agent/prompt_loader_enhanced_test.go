package agent

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPromptConfig(t *testing.T) {
	cfg := DefaultPromptConfig()
	if len(cfg.FileOrder) != 3 {
		t.Errorf("expected 3 file entries, got %d", len(cfg.FileOrder))
	}
	if cfg.FileOrder[0].Filename != "AGENTS.md" {
		t.Errorf("first file should be AGENTS.md, got %q", cfg.FileOrder[0].Filename)
	}
	if !cfg.FileOrder[0].Required {
		t.Error("AGENTS.md should be required")
	}
	if cfg.FileOrder[2].Required {
		t.Error("PROFILE.md should not be required")
	}
	if cfg.Language != "zh" {
		t.Errorf("default language should be zh, got %q", cfg.Language)
	}
}

func TestNewPromptLoaderWithConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := PromptConfig{
		FileOrder: []PromptFileEntry{{Filename: "AGENTS.md", Required: true}},
		Language:  "en",
	}
	loader := NewPromptLoaderWithConfig(dir, "fallback", cfg)
	if loader.Language() != "en" {
		t.Errorf("expected en, got %q", loader.Language())
	}
	if loader.Config().Language != "en" {
		t.Error("Config() should return the configured language")
	}
}

func TestPromptLoader_CopyMDFiles(t *testing.T) {
	srcDir := t.TempDir()
	workDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(srcDir, "AGENTS.md"), []byte("agents content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "SOUL.md"), []byte("soul content"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewPromptLoader(workDir, "")
	if err := loader.CopyMDFiles(srcDir); err != nil {
		t.Fatalf("CopyMDFiles: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(workDir, "AGENTS.md"))
	if err != nil {
		t.Fatalf("read AGENTS.md: %v", err)
	}
	if string(data) != "agents content" {
		t.Errorf("AGENTS.md content: got %q", string(data))
	}

	data, err = os.ReadFile(filepath.Join(workDir, "SOUL.md"))
	if err != nil {
		t.Fatalf("read SOUL.md: %v", err)
	}
	if string(data) != "soul content" {
		t.Errorf("SOUL.md content: got %q", string(data))
	}
}

func TestPromptLoader_CopyMDFiles_NoOverwrite(t *testing.T) {
	srcDir := t.TempDir()
	workDir := t.TempDir()

	if err := os.WriteFile(filepath.Join(srcDir, "AGENTS.md"), []byte("new content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "AGENTS.md"), []byte("existing content"), 0644); err != nil {
		t.Fatal(err)
	}

	loader := NewPromptLoader(workDir, "")
	_ = loader.CopyMDFiles(srcDir)

	data, _ := os.ReadFile(filepath.Join(workDir, "AGENTS.md"))
	if string(data) != "existing content" {
		t.Error("existing file should not be overwritten")
	}
}

func TestPromptLoader_CopyMDFiles_NonexistentDir(t *testing.T) {
	workDir := t.TempDir()
	loader := NewPromptLoader(workDir, "")
	err := loader.CopyMDFiles("/nonexistent/path/that/does/not/exist")
	if err != nil {
		t.Error("should not error for nonexistent source dir")
	}
}

func TestPromptLoader_DefaultWorkingDir(t *testing.T) {
	loader := NewPromptLoader("", "fallback")
	if loader.WorkingDir() == "" {
		t.Error("expected non-empty default working dir")
	}
}
