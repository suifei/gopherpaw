package agent

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/suifei/gopherpaw/internal/config"
)

// Alignment tests to verify compatibility with CoPaw

func TestAlignment_PromptLoader(t *testing.T) {
	tmpDir := t.TempDir()

	// Test MD file loading
	tests := []struct {
		name     string
		filename string
		content  string
	}{
		{"SOUL", "SOUL.md", "# Soul\nYou are a helpful AI assistant."},
		{"AGENTS", "AGENTS.md", "# Agents\nAvailable agents and their roles."},
		{"PROFILE", "PROFILE.md", "# Profile\nUser profile information."},
		{"MEMORY", "MEMORY.md", "# Memory\nImportant memories."},
		{"HEARTBEAT", "HEARTBEAT.md", "# Heartbeat\nDaily tasks and reminders."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(tmpDir, tt.filename)
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatalf("write file: %v", err)
			}

			loader := NewPromptLoader(tmpDir, "en")

			var result string
			var err error

			switch tt.filename {
			case "SOUL.md":
				result, err = loader.LoadSOUL()
			case "AGENTS.md":
				result, err = loader.LoadAGENTS()
			case "PROFILE.md":
				result, err = loader.LoadPROFILE()
			case "MEMORY.md":
				result, err = loader.LoadMEMORY()
			case "HEARTBEAT.md":
				result, err = loader.LoadHEARTBEAT()
			}

			if err != nil {
				t.Errorf("Load%s failed: %v", tt.name, err)
			}
			if result != tt.content {
				t.Errorf("Load%s = %q, want %q", tt.name, result, tt.content)
			}
		})
	}
}

func TestAlignment_Bootstrap(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("bootstrap flow", func(t *testing.T) {
		// Create BOOTSTRAP.md
		bootstrapPath := filepath.Join(tmpDir, "BOOTSTRAP.md")
		bootstrapContent := "Please set up my profile"
		if err := os.WriteFile(bootstrapPath, []byte(bootstrapContent), 0644); err != nil {
			t.Fatalf("write bootstrap: %v", err)
		}

		// Setup agent
		llm := &mockLLM{
			chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
				return &ChatResponse{Content: "Profile created"}, nil
			},
		}

		mem := &mockMemory{}
		agent := NewReact(llm, mem, nil, config.AgentConfig{
			Running: config.AgentRunningConfig{MaxTurns: 5},
		})

		loader := NewPromptLoader(tmpDir, "en")
		runner := NewBootstrapRunner(agent, loader)

		// Run bootstrap
		err := runner.RunIfNeeded(context.Background(), "chat1")
		if err != nil {
			t.Fatalf("RunIfNeeded failed: %v", err)
		}

		// Verify PROFILE.md was created
		profilePath := filepath.Join(tmpDir, "PROFILE.md")
		if _, err := os.Stat(profilePath); os.IsNotExist(err) {
			t.Error("PROFILE.md was not created")
		}

		// Verify BOOTSTRAP.md was deleted
		if _, err := os.Stat(bootstrapPath); !os.IsNotExist(err) {
			t.Error("BOOTSTRAP.md was not deleted")
		}
	})
}

func TestAlignment_MagicCommands(t *testing.T) {
	mem := &mockMemory{
		compactFunc: func(ctx context.Context, chatID string) error {
			return nil
		},
		loadFunc: func(ctx context.Context, chatID string, limit int) ([]Message, error) {
			return []Message{
				{Role: "user", Content: "test"},
				{Role: "assistant", Content: "response"},
			}, nil
		},
	}

	tests := []struct {
		name    string
		command string
		wantErr bool
		handled bool
	}{
		{"compact", "/compact", false, true},
		{"clear", "/clear", false, true},
		{"history", "/history", false, true},
		{"new", "/new", false, true},
		{"non-magic", "hello", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, handled, err := HandleMagicCommand(context.Background(), mem, "chat1", tt.command, nil)

			if (err != nil) != tt.wantErr {
				t.Errorf("HandleMagicCommand() error = %v, wantErr %v", err, tt.wantErr)
			}

			if handled != tt.handled {
				t.Errorf("HandleMagicCommand() handled = %v, want %v", handled, tt.handled)
			}

			if tt.handled && result == "" {
				t.Error("Expected non-empty result for handled command")
			}
		})
	}
}

func TestAlignment_ToolExecution(t *testing.T) {
	// Test tool execution flow matches CoPaw
	tool := &mockTool{
		name:        "test_tool",
		description: "Test tool",
		executeFunc: func(ctx context.Context, args string) (string, error) {
			return "tool result", nil
		},
	}

	llm := &mockLLM{
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			// First call: request tool
			if len(req.Tools) > 0 && req.Messages[len(req.Messages)-1].Role == "user" {
				return &ChatResponse{
					ToolCalls: []ToolCall{
						{ID: "tc1", Name: "test_tool", Arguments: `{"arg": "value"}`},
					},
				}, nil
			}
			// Second call: final response
			return &ChatResponse{Content: "Final answer"}, nil
		},
	}

	mem := &mockMemory{}
	agent := NewReact(llm, mem, []Tool{tool}, config.AgentConfig{
		Running: config.AgentRunningConfig{MaxTurns: 5},
	})

	response, err := agent.Run(context.Background(), "chat1", "Use tool")
	if err != nil {
		t.Fatalf("Agent.Run failed: %v", err)
	}

	if response != "Final answer" {
		t.Errorf("Response = %q, want %q", response, "Final answer")
	}
}

func TestAlignment_MemoryIntegration(t *testing.T) {
	// Test memory integration matches CoPaw behavior
	savedMessages := []Message{}

	mem := &mockMemory{
		saveFunc: func(ctx context.Context, chatID string, msg Message) error {
			savedMessages = append(savedMessages, msg)
			return nil
		},
		loadFunc: func(ctx context.Context, chatID string, limit int) ([]Message, error) {
			return savedMessages, nil
		},
	}

	llm := &mockLLM{
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			return &ChatResponse{Content: "OK"}, nil
		},
	}

	agent := NewReact(llm, mem, nil, config.AgentConfig{
		Running: config.AgentRunningConfig{MaxTurns: 5},
	})

	// Send multiple messages
	for i := 0; i < 3; i++ {
		_, err := agent.Run(context.Background(), "chat1", "test")
		if err != nil {
			t.Fatalf("Run %d failed: %v", i, err)
		}
	}

	// Verify messages were saved
	if len(savedMessages) < 3 {
		t.Errorf("Expected at least 3 saved messages, got %d", len(savedMessages))
	}

	// Verify different message types
	hasUser := false
	hasAssistant := false
	for _, msg := range savedMessages {
		if msg.Role == "user" {
			hasUser = true
		}
		if msg.Role == "assistant" {
			hasAssistant = true
		}
	}

	if !hasUser {
		t.Error("No user messages saved")
	}
	if !hasAssistant {
		t.Error("No assistant messages saved")
	}
}

func TestAlignment_ConfigCompatibility(t *testing.T) {
	// Test that config structures match CoPaw
	cfg := config.AgentConfig{
		SystemPrompt: "Test prompt",
		WorkingDir:   "/tmp",
		Running: config.AgentRunningConfig{
			MaxTurns:       10,
			MaxInputLength: 8000,
		},
	}

	agent := NewReact(&mockLLM{}, &mockMemory{}, nil, cfg)
	if agent == nil {
		t.Fatal("Failed to create agent with config")
	}

	// Verify config is applied
	if agent.cfg.Running.MaxTurns != 10 {
		t.Errorf("MaxTurns = %d, want 10", agent.cfg.Running.MaxTurns)
	}
}
