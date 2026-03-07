package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestPromptLoader_LoadMEMORY(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("with MEMORY.md", func(t *testing.T) {
		memoryPath := filepath.Join(tmpDir, "MEMORY.md")
		expectedContent := "Memory content"
		if err := os.WriteFile(memoryPath, []byte(expectedContent), 0644); err != nil {
			t.Fatalf("write memory: %v", err)
		}

		loader := NewPromptLoader(tmpDir, "en")
		content, err := loader.LoadMEMORY()
		if err != nil {
			t.Fatalf("LoadMEMORY failed: %v", err)
		}
		if content != expectedContent {
			t.Errorf("content = %q, want %q", content, expectedContent)
		}
	})

	t.Run("with today's memory", func(t *testing.T) {
		today := time.Now().Format("2006-01-02")
		memoryPath := filepath.Join(tmpDir, "memory", today+".md")
		if err := os.MkdirAll(filepath.Dir(memoryPath), 0755); err != nil {
			t.Fatalf("create memory dir: %v", err)
		}
		expectedContent := "Today's memory"
		if err := os.WriteFile(memoryPath, []byte(expectedContent), 0644); err != nil {
			t.Fatalf("write today's memory: %v", err)
		}

		loader := NewPromptLoader(tmpDir, "en")
		content, err := loader.LoadMEMORY()
		if err != nil {
			t.Fatalf("LoadMEMORY failed: %v", err)
		}
		if content != expectedContent {
			t.Errorf("content = %q, want %q", content, expectedContent)
		}
	})

	t.Run("no memory file", func(t *testing.T) {
		emptyDir := t.TempDir()
		loader := NewPromptLoader(emptyDir, "en")
		content, err := loader.LoadMEMORY()
		if err != nil {
			t.Fatalf("LoadMEMORY failed: %v", err)
		}
		if content != "" {
			t.Errorf("expected empty content, got %q", content)
		}
	})
}

func TestPromptLoader_LoadHEARTBEAT(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("with HEARTBEAT.md", func(t *testing.T) {
		heartbeatPath := filepath.Join(tmpDir, "HEARTBEAT.md")
		expectedContent := "Heartbeat content"
		if err := os.WriteFile(heartbeatPath, []byte(expectedContent), 0644); err != nil {
			t.Fatalf("write heartbeat: %v", err)
		}

		loader := NewPromptLoader(tmpDir, "en")
		content, err := loader.LoadHEARTBEAT()
		if err != nil {
			t.Fatalf("LoadHEARTBEAT failed: %v", err)
		}
		if content != expectedContent {
			t.Errorf("content = %q, want %q", content, expectedContent)
		}
	})

	t.Run("no heartbeat file", func(t *testing.T) {
		emptyDir := t.TempDir()
		loader := NewPromptLoader(emptyDir, "en")
		content, err := loader.LoadHEARTBEAT()
		if err != nil {
			t.Fatalf("LoadHEARTBEAT failed: %v", err)
		}
		if content != "" {
			t.Errorf("expected empty content, got %q", content)
		}
	})
}
