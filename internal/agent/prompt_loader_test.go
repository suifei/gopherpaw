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

func TestStripYAMLFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no YAML frontmatter",
			input:    "Just plain content",
			expected: "Just plain content",
		},
		{
			name:     "YAML frontmatter with summary",
			input:    "---\nsummary: Agent soul\n---\nActual content here",
			expected: "Actual content here",
		},
		{
			name:     "YAML frontmatter with multiple fields",
			input:    "---\nsummary: Test\ndescription: A test file\n---\nContent after YAML",
			expected: "Content after YAML",
		},
		{
			name:     "YAML frontmatter at start only",
			input:    "---start\n---\ncontent\n---still inside",
			expected: "content\n---still inside",
		},
		{
			name:     "not starting with dash",
			input:    "Some text --- then more",
			expected: "Some text --- then more",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "incomplete YAML frontmatter (only one dash)",
			input:    "---content",
			expected: "---content",
		},
		{
			name:     "incomplete YAML frontmatter (two dashes)",
			input:    "---\ncontent",
			expected: "---\ncontent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stripYAMLFrontmatter(tt.input)
			if result != tt.expected {
				t.Errorf("stripYAMLFrontmatter() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestPromptLoader_BuildSystemPrompt_FileHeaders(t *testing.T) {
	dir := t.TempDir()
	loader := NewPromptLoader(dir, "fallback")

	// Create required files
	if err := os.WriteFile(filepath.Join(dir, "SOUL.md"), []byte("soul content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("agents content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "PROFILE.md"), []byte("profile content"), 0644); err != nil {
		t.Fatal(err)
	}

	prompt := loader.BuildSystemPrompt("")

	// Verify file headers are present
	if !strings.Contains(prompt, "# SOUL.md") {
		t.Error("missing SOUL.md header")
	}
	if !strings.Contains(prompt, "# AGENTS.md") {
		t.Error("missing AGENTS.md header")
	}
	if !strings.Contains(prompt, "# PROFILE.md") {
		t.Error("missing PROFILE.md header")
	}

	// Verify content is present
	if !strings.Contains(prompt, "soul content") {
		t.Error("missing soul content")
	}
	if !strings.Contains(prompt, "agents content") {
		t.Error("missing agents content")
	}
	if !strings.Contains(prompt, "profile content") {
		t.Error("missing profile content")
	}
}

func TestPromptLoader_ReadFile_StripsYAML(t *testing.T) {
	dir := t.TempDir()
	loader := NewPromptLoader(dir, "fallback")

	// Create a file with YAML frontmatter
	contentWithYAML := `---
summary: Test file
description: This should be stripped
---
This is the actual content that should remain.`
	if err := os.WriteFile(filepath.Join(dir, "SOUL.md"), []byte(contentWithYAML), 0644); err != nil {
		t.Fatal(err)
	}

	// Load the file
	content, err := loader.LoadSOUL()
	if err != nil {
		t.Fatalf("LoadSOUL failed: %v", err)
	}

	// Verify YAML is stripped
	if strings.Contains(content, "summary:") {
		t.Error("YAML frontmatter should be stripped")
	}
	if strings.Contains(content, "description:") {
		t.Error("YAML frontmatter should be stripped")
	}
	if !strings.Contains(content, "This is the actual content") {
		t.Error("actual content should be present")
	}
}

func TestPromptLoader_BuildSystemPrompt_WithSkills(t *testing.T) {
	dir := t.TempDir()
	loader := NewPromptLoader(dir, "fallback")

	// Create required files
	if err := os.WriteFile(filepath.Join(dir, "SOUL.md"), []byte("soul"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("agents"), 0644); err != nil {
		t.Fatal(err)
	}

	skillsContent := "Available tools: search, browse"
	prompt := loader.BuildSystemPrompt(skillsContent)

	// Verify new AI collaboration framework elements are present
	if !strings.Contains(prompt, "# 🤝") && !strings.Contains(prompt, "# AI Collaboration Framework") && !strings.Contains(prompt, "# AI 智能协作框架") {
		t.Error("missing AI collaboration framework section")
	}
	if !strings.Contains(prompt, "# 📋") && !strings.Contains(prompt, "# Available Capabilities") && !strings.Contains(prompt, "# 可用能力索引") {
		t.Error("missing capabilities index section")
	}
	if !strings.Contains(prompt, skillsContent) {
		t.Error("missing skills content")
	}
}

// TestPromptLoader_BuildSystemPrompt_LoadOrder verifies the loading order is AGENTS → SOUL → PROFILE
// to align with CoPaw's prompt.py behavior.
func TestPromptLoader_BuildSystemPrompt_LoadOrder(t *testing.T) {
	dir := t.TempDir()
	loader := NewPromptLoader(dir, "fallback")

	// Create required files
	if err := os.WriteFile(filepath.Join(dir, "AGENTS.md"), []byte("agents content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SOUL.md"), []byte("soul content"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "PROFILE.md"), []byte("profile content"), 0644); err != nil {
		t.Fatal(err)
	}

	prompt := loader.BuildSystemPrompt("")

	// Find positions of each header
	agentsPos := strings.Index(prompt, "# AGENTS.md")
	soulPos := strings.Index(prompt, "# SOUL.md")
	profilePos := strings.Index(prompt, "# PROFILE.md")

	// Verify all headers exist
	if agentsPos == -1 {
		t.Error("missing AGENTS.md header")
	}
	if soulPos == -1 {
		t.Error("missing SOUL.md header")
	}
	if profilePos == -1 {
		t.Error("missing PROFILE.md header")
	}

	// Verify order: AGENTS → SOUL → PROFILE
	if agentsPos >= soulPos {
		t.Errorf("AGENTS.md should come before SOUL.md (agentsPos=%d, soulPos=%d)", agentsPos, soulPos)
	}
	if soulPos >= profilePos {
		t.Errorf("SOUL.md should come before PROFILE.md (soulPos=%d, profilePos=%d)", soulPos, profilePos)
	}
}
