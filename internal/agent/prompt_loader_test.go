package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPromptLoader_BuildSystemPrompt_Fallback(t *testing.T) {
	dir := t.TempDir()
	loader := NewPromptLoader(dir, "fallback prompt")
	s := loader.BuildSystemPrompt("")
	if s != "" {
		t.Errorf("expected empty when no SOUL/AGENTS, got %q", s)
	}
	sys, err := loader.LoadSystemPrompt()
	if err != nil {
		t.Fatal(err)
	}
	if sys != "fallback prompt" {
		t.Errorf("expected fallback, got %q", sys)
	}
}

func TestPromptLoader_LoadSOUL_AGENTS(t *testing.T) {
	dir := t.TempDir()
	loader := NewPromptLoader(dir, "fallback")
	if err := os.WriteFile(filepath.Join(dir, "SOUL.md"), []byte("soul"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("agents"), 0644); err != nil {
		t.Fatal(err)
	}
	s := loader.BuildSystemPrompt("")
	if s == "" {
		t.Error("expected non-empty prompt")
	}
	if !strings.Contains(s, "soul") || !strings.Contains(s, "agents") {
		t.Errorf("expected soul+agents in prompt, got %q", s)
	}
}

func TestPromptLoader_HasBootstrap(t *testing.T) {
	dir := t.TempDir()
	loader := NewPromptLoader(dir, "")
	if loader.HasBootstrap() {
		t.Error("expected no bootstrap")
	}
	if err := os.WriteFile(filepath.Join(dir, "BOOTSTRAP.md"), []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if !loader.HasBootstrap() {
		t.Error("expected bootstrap")
	}
	if err := loader.DeleteBootstrap(); err != nil {
		t.Fatal(err)
	}
	if loader.HasBootstrap() {
		t.Error("expected no bootstrap after delete")
	}
}

