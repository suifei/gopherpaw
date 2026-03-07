package agent

import (
	"context"
	"testing"
	"time"

	"github.com/suifei/gopherpaw/internal/config"
)

// Integration tests for Agent + LLM + Memory + Tools

func TestAgentIntegration_BasicConversation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup
	llm := &mockLLM{
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			return &ChatResponse{Content: "Hello! How can I help you?"}, nil
		},
	}

	mem := &mockMemory{}

	cfg := config.AgentConfig{
		SystemPrompt: "You are a helpful assistant.",
		Running: config.AgentRunningConfig{
			MaxTurns:       5,
			MaxInputLength: 4000,
		},
	}

	agent := NewReact(llm, mem, nil, cfg)
	ctx := context.Background()

	// Test basic conversation
	response, err := agent.Run(ctx, "chat1", "Hello")
	if err != nil {
		t.Fatalf("Agent.Run failed: %v", err)
	}

	if response == "" {
		t.Error("Expected non-empty response")
	}

	t.Logf("Response: %s", response)
}

func TestAgentIntegration_WithTools(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup tool
	executed := false
	tool := &mockTool{
		name:        "test_tool",
		description: "A test tool",
		executeFunc: func(ctx context.Context, args string) (string, error) {
			executed = true
			return "tool result", nil
		},
	}

	// Setup LLM that calls tool
	callCount := 0
	llm := &mockLLM{
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			callCount++
			if callCount == 1 {
				return &ChatResponse{
					ToolCalls: []ToolCall{
						{ID: "tc1", Name: "test_tool", Arguments: "{}"},
					},
				}, nil
			}
			return &ChatResponse{Content: "Done"}, nil
		},
	}

	mem := &mockMemory{}
	cfg := config.AgentConfig{
		Running: config.AgentRunningConfig{MaxTurns: 5},
	}

	agent := NewReact(llm, mem, []Tool{tool}, cfg)
	ctx := context.Background()

	// Test tool execution
	response, err := agent.Run(ctx, "chat1", "Use the tool")
	if err != nil {
		t.Fatalf("Agent.Run failed: %v", err)
	}

	if !executed {
		t.Error("Tool was not executed")
	}

	if response != "Done" {
		t.Errorf("Response = %q, want %q", response, "Done")
	}
}

func TestAgentIntegration_WithMemory(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Setup memory that tracks calls
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

	cfg := config.AgentConfig{
		Running: config.AgentRunningConfig{MaxTurns: 5},
	}

	agent := NewReact(llm, mem, nil, cfg)
	ctx := context.Background()

	// Test first message
	_, err := agent.Run(ctx, "chat1", "First message")
	if err != nil {
		t.Fatalf("First Run failed: %v", err)
	}

	// Test second message - should have history
	_, err = agent.Run(ctx, "chat1", "Second message")
	if err != nil {
		t.Fatalf("Second Run failed: %v", err)
	}

	// Verify memory was used
	if len(savedMessages) == 0 {
		t.Error("No messages were saved to memory")
	}
}

func TestAgentIntegration_ParallelRequests(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	llm := &mockLLM{
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			time.Sleep(50 * time.Millisecond)
			return &ChatResponse{Content: "Response"}, nil
		},
	}

	mem := &mockMemory{}
	cfg := config.AgentConfig{
		Running: config.AgentRunningConfig{MaxTurns: 5},
	}

	agent := NewReact(llm, mem, nil, cfg)

	// Run multiple requests concurrently
	done := make(chan bool, 3)

	for i := 0; i < 3; i++ {
		go func(id int) {
			ctx := context.Background()
			chatID := string(rune('A' + id))
			_, err := agent.Run(ctx, chatID, "Test")
			if err != nil {
				t.Errorf("Request %d failed: %v", id, err)
			}
			done <- true
		}(i)
	}

	// Wait for all to complete
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			// Good
		case <-time.After(2 * time.Second):
			t.Fatal("Timeout waiting for concurrent requests")
		}
	}
}

func TestAgentIntegration_WithHook(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	hookExecuted := false
	hook := func(ctx context.Context, a *ReactAgent, chatID string, messages []Message) ([]Message, error) {
		hookExecuted = true
		return messages, nil
	}

	llm := &mockLLM{
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			return &ChatResponse{Content: "OK"}, nil
		},
	}

	mem := &mockMemory{}
	cfg := config.AgentConfig{
		Running: config.AgentRunningConfig{MaxTurns: 5},
	}

	agent := NewReact(llm, mem, nil, cfg)
	agent.AddHook(hook)

	ctx := context.Background()
	_, err := agent.Run(ctx, "chat1", "Test")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if !hookExecuted {
		t.Error("Hook was not executed")
	}
}

func TestAgentIntegration_StreamResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	llm := &mockLLM{
		chatFunc: func(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
			return &ChatResponse{Content: "Stream response"}, nil
		},
	}

	mem := &mockMemory{}
	cfg := config.AgentConfig{
		Running: config.AgentRunningConfig{MaxTurns: 5},
	}

	agent := NewReact(llm, mem, nil, cfg)
	ctx := context.Background()

	stream, err := agent.RunStream(ctx, "chat1", "Test")
	if err != nil {
		t.Fatalf("RunStream failed: %v", err)
	}

	chunks := []string{}
	for chunk := range stream {
		chunks = append(chunks, chunk)
	}

	if len(chunks) == 0 {
		t.Error("No chunks received from stream")
	}
}
