package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
)

func TestNewOllama(t *testing.T) {
	tests := []struct {
		name      string
		cfg       config.LLMConfig
		wantURL   string
		wantModel string
		wantErr   bool
	}{
		{
			name:      "default values",
			cfg:       config.LLMConfig{},
			wantURL:   "http://localhost:11434",
			wantModel: "llama3.2",
			wantErr:   false,
		},
		{
			name: "custom url and model",
			cfg: config.LLMConfig{
				OllamaURL: "http://custom-host:8080",
				Model:     "custom-model",
			},
			wantURL:   "http://custom-host:8080",
			wantModel: "custom-model",
			wantErr:   false,
		},
		{
			name: "only custom url",
			cfg: config.LLMConfig{
				OllamaURL: "http://another-host:9000",
			},
			wantURL:   "http://another-host:9000",
			wantModel: "llama3.2",
			wantErr:   false,
		},
		{
			name: "only custom model",
			cfg: config.LLMConfig{
				Model: "my-model",
			},
			wantURL:   "http://localhost:11434",
			wantModel: "my-model",
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewOllama(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOllama() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if p.baseURL != tt.wantURL {
					t.Errorf("baseURL = %q, want %q", p.baseURL, tt.wantURL)
				}
				if p.model != tt.wantModel {
					t.Errorf("model = %q, want %q", p.model, tt.wantModel)
				}
				if p.client == nil {
					t.Error("client should not be nil")
				}
				if p.client.Timeout != 120*time.Second {
					t.Errorf("client timeout = %v, want 120s", p.client.Timeout)
				}
			}
		})
	}
}

func TestOllamaProvider_Name(t *testing.T) {
	p, _ := NewOllama(config.LLMConfig{})
	if p.Name() != "ollama" {
		t.Errorf("Name() = %q, want ollama", p.Name())
	}
}

func TestOllamaProvider_Chat_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/api/chat" {
			t.Errorf("expected path /api/chat, got %s", r.URL.Path)
		}

		var req ollamaChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.Model != "test-model" {
			t.Errorf("expected model 'test-model', got %q", req.Model)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := ollamaChatResponse{
			Message: struct {
				Role      string           `json:"role"`
				Content   string           `json:"content"`
				ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
			}{
				Role:    "assistant",
				Content: "Hello, world!",
			},
			Done:            true,
			PromptEvalCount: 10,
			EvalCount:       20,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p, _ := NewOllama(config.LLMConfig{
		OllamaURL: server.URL,
		Model:     "test-model",
	})

	ctx := context.Background()
	req := &agent.ChatRequest{
		Messages: []agent.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	resp, err := p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if resp.Content != "Hello, world!" {
		t.Errorf("Content = %q, want 'Hello, world!'", resp.Content)
	}

	if resp.Usage.PromptTokens != 10 {
		t.Errorf("PromptTokens = %d, want 10", resp.Usage.PromptTokens)
	}

	if resp.Usage.CompletionTokens != 20 {
		t.Errorf("CompletionTokens = %d, want 20", resp.Usage.CompletionTokens)
	}

	if resp.Usage.TotalTokens != 30 {
		t.Errorf("TotalTokens = %d, want 30", resp.Usage.TotalTokens)
	}
}

func TestOllamaProvider_Chat_WithTools(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ollamaChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if len(req.Tools) != 1 {
			t.Errorf("expected 1 tool, got %d", len(req.Tools))
		}

		w.Header().Set("Content-Type", "application/json")
		resp := ollamaChatResponse{
			Message: struct {
				Role      string           `json:"role"`
				Content   string           `json:"content"`
				ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
			}{
				Role:    "assistant",
				Content: "",
				ToolCalls: []ollamaToolCall{
					{
						ID:   "call_123",
						Type: "function",
						Function: struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						}{
							Name:      "test_tool",
							Arguments: `{"arg":"value"}`,
						},
					},
				},
			},
			Done:            true,
			PromptEvalCount: 5,
			EvalCount:       10,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p, _ := NewOllama(config.LLMConfig{
		OllamaURL: server.URL,
		Model:     "test-model",
	})

	ctx := context.Background()
	req := &agent.ChatRequest{
		Messages: []agent.Message{
			{Role: "user", Content: "Use a tool"},
		},
		Tools: []agent.ToolDef{
			{
				Name:        "test_tool",
				Description: "A test tool",
				Parameters: map[string]interface{}{
					"type": "object",
				},
			},
		},
	}

	resp, err := p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(resp.ToolCalls))
	}

	if resp.ToolCalls[0].ID != "call_123" {
		t.Errorf("ToolCall ID = %q, want call_123", resp.ToolCalls[0].ID)
	}

	if resp.ToolCalls[0].Name != "test_tool" {
		t.Errorf("ToolCall Name = %q, want test_tool", resp.ToolCalls[0].Name)
	}

	if resp.ToolCalls[0].Arguments != `{"arg":"value"}` {
		t.Errorf("ToolCall Arguments = %q, want {\"arg\":\"value\"}", resp.ToolCalls[0].Arguments)
	}
}

func TestOllamaProvider_Chat_WithTemperature(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ollamaChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.Temperature == nil {
			t.Error("expected Temperature to be set")
		} else if *req.Temperature != 0.7 {
			t.Errorf("Temperature = %v, want 0.7", *req.Temperature)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := ollamaChatResponse{
			Message: struct {
				Role      string           `json:"role"`
				Content   string           `json:"content"`
				ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
			}{
				Role:    "assistant",
				Content: "response",
			},
			Done: true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p, _ := NewOllama(config.LLMConfig{
		OllamaURL: server.URL,
		Model:     "test-model",
	})

	ctx := context.Background()
	req := &agent.ChatRequest{
		Messages:    []agent.Message{{Role: "user", Content: "test"}},
		Temperature: 0.7,
	}

	_, err := p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
}

func TestOllamaProvider_Chat_WithMaxTokens(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req ollamaChatRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		if req.NumPredict != 100 {
			t.Errorf("NumPredict = %d, want 100", req.NumPredict)
		}

		w.Header().Set("Content-Type", "application/json")
		resp := ollamaChatResponse{
			Message: struct {
				Role      string           `json:"role"`
				Content   string           `json:"content"`
				ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
			}{
				Role:    "assistant",
				Content: "response",
			},
			Done: true,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p, _ := NewOllama(config.LLMConfig{
		OllamaURL: server.URL,
		Model:     "test-model",
	})

	ctx := context.Background()
	req := &agent.ChatRequest{
		Messages:  []agent.Message{{Role: "user", Content: "test"}},
		MaxTokens: 100,
	}

	_, err := p.Chat(ctx, req)
	if err != nil {
		t.Fatalf("Chat() error = %v", err)
	}
}

func TestOllamaProvider_Chat_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer server.Close()

	p, _ := NewOllama(config.LLMConfig{
		OllamaURL: server.URL,
		Model:     "test-model",
	})

	ctx := context.Background()
	req := &agent.ChatRequest{
		Messages: []agent.Message{{Role: "user", Content: "Hello"}},
	}

	_, err := p.Chat(ctx, req)
	if err == nil {
		t.Error("Chat() should return error for HTTP 500")
	}

	if !strings.Contains(err.Error(), "status 500") {
		t.Errorf("error should contain status 500, got %v", err)
	}
}

func TestOllamaProvider_Chat_NetworkError(t *testing.T) {
	p, _ := NewOllama(config.LLMConfig{
		OllamaURL: "http://invalid-host-that-does-not-exist:9999",
		Model:     "test-model",
	})

	ctx := context.Background()
	req := &agent.ChatRequest{
		Messages: []agent.Message{{Role: "user", Content: "Hello"}},
	}

	_, err := p.Chat(ctx, req)
	if err == nil {
		t.Error("Chat() should return error for network failure")
	}
}

func TestOllamaProvider_Chat_InvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	p, _ := NewOllama(config.LLMConfig{
		OllamaURL: server.URL,
		Model:     "test-model",
	})

	ctx := context.Background()
	req := &agent.ChatRequest{
		Messages: []agent.Message{{Role: "user", Content: "Hello"}},
	}

	_, err := p.Chat(ctx, req)
	if err == nil {
		t.Error("Chat() should return error for invalid JSON response")
	}

	if !strings.Contains(err.Error(), "decode response") {
		t.Errorf("error should mention decode response, got %v", err)
	}
}

func TestOllamaProvider_Chat_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	p, _ := NewOllama(config.LLMConfig{
		OllamaURL: server.URL,
		Model:     "test-model",
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := &agent.ChatRequest{
		Messages: []agent.Message{{Role: "user", Content: "Hello"}},
	}

	_, err := p.Chat(ctx, req)
	if err == nil {
		t.Error("Chat() should return error when context is cancelled")
	}
}

func TestOllamaProvider_ChatStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := ollamaChatResponse{
			Message: struct {
				Role      string           `json:"role"`
				Content   string           `json:"content"`
				ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
			}{
				Role:    "assistant",
				Content: "Hello, world!",
			},
			Done:            true,
			PromptEvalCount: 10,
			EvalCount:       20,
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	p, _ := NewOllama(config.LLMConfig{
		OllamaURL: server.URL,
		Model:     "test-model",
	})

	ctx := context.Background()
	req := &agent.ChatRequest{
		Messages: []agent.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	stream, err := p.ChatStream(ctx, req)
	if err != nil {
		t.Fatalf("ChatStream() error = %v", err)
	}
	defer stream.Close()

	chunk, err := stream.Recv()
	if err != nil {
		t.Fatalf("Recv() error = %v", err)
	}

	if chunk.Content == "" {
		t.Error("Expected non-empty content from stream")
	}

	_, err = stream.Recv()
	if err != io.EOF {
		t.Errorf("Expected EOF on second Recv(), got %v", err)
	}
}

func TestToOllamaRequest(t *testing.T) {
	tests := []struct {
		name         string
		req          *agent.ChatRequest
		model        string
		wantStream   bool
		wantMsgCount int
	}{
		{
			name: "simple user message",
			req: &agent.ChatRequest{
				Messages: []agent.Message{
					{Role: "user", Content: "Hello"},
				},
			},
			model:        "test-model",
			wantStream:   false,
			wantMsgCount: 1,
		},
		{
			name: "multiple messages",
			req: &agent.ChatRequest{
				Messages: []agent.Message{
					{Role: "user", Content: "Hello"},
					{Role: "assistant", Content: "Hi there!"},
					{Role: "user", Content: "How are you?"},
				},
			},
			model:        "test-model",
			wantStream:   false,
			wantMsgCount: 3,
		},
		{
			name: "message with tool calls",
			req: &agent.ChatRequest{
				Messages: []agent.Message{
					{Role: "assistant", Content: "", ToolCalls: []agent.ToolCall{
						{ID: "call_1", Name: "tool_name", Arguments: `{"arg":"value"}`},
					}},
				},
			},
			model:        "test-model",
			wantStream:   false,
			wantMsgCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toOllamaRequest(tt.req, tt.model)

			if result.Model != tt.model {
				t.Errorf("Model = %q, want %q", result.Model, tt.model)
			}

			if result.Stream != tt.wantStream {
				t.Errorf("Stream = %v, want %v", result.Stream, tt.wantStream)
			}

			if len(result.Messages) != tt.wantMsgCount {
				t.Errorf("Messages count = %d, want %d", len(result.Messages), tt.wantMsgCount)
			}
		})
	}
}

func TestToOllamaAgentResponse(t *testing.T) {
	tests := []struct {
		name                 string
		resp                 *ollamaChatResponse
		wantContent          string
		wantToolCalls        int
		wantPromptTokens     int
		wantCompletionTokens int
	}{
		{
			name: "simple response",
			resp: &ollamaChatResponse{
				Message: struct {
					Role      string           `json:"role"`
					Content   string           `json:"content"`
					ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
				}{
					Role:    "assistant",
					Content: "Hello!",
				},
				Done:            true,
				PromptEvalCount: 10,
				EvalCount:       20,
			},
			wantContent:          "Hello!",
			wantToolCalls:        0,
			wantPromptTokens:     10,
			wantCompletionTokens: 20,
		},
		{
			name: "response with tool calls",
			resp: &ollamaChatResponse{
				Message: struct {
					Role      string           `json:"role"`
					Content   string           `json:"content"`
					ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
				}{
					Role:    "assistant",
					Content: "",
					ToolCalls: []ollamaToolCall{
						{
							ID:   "call_1",
							Type: "function",
							Function: struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
							}{
								Name:      "tool_a",
								Arguments: `{"a":1}`,
							},
						},
						{
							ID:   "call_2",
							Type: "function",
							Function: struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
							}{
								Name:      "tool_b",
								Arguments: `{"b":2}`,
							},
						},
					},
				},
				Done:            true,
				PromptEvalCount: 5,
				EvalCount:       10,
			},
			wantContent:          "",
			wantToolCalls:        2,
			wantPromptTokens:     5,
			wantCompletionTokens: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toOllamaAgentResponse(tt.resp)

			if result.Content != tt.wantContent {
				t.Errorf("Content = %q, want %q", result.Content, tt.wantContent)
			}

			if len(result.ToolCalls) != tt.wantToolCalls {
				t.Errorf("ToolCalls count = %d, want %d", len(result.ToolCalls), tt.wantToolCalls)
			}

			if result.Usage.PromptTokens != tt.wantPromptTokens {
				t.Errorf("PromptTokens = %d, want %d", result.Usage.PromptTokens, tt.wantPromptTokens)
			}

			if result.Usage.CompletionTokens != tt.wantCompletionTokens {
				t.Errorf("CompletionTokens = %d, want %d", result.Usage.CompletionTokens, tt.wantCompletionTokens)
			}

			expectedTotal := tt.wantPromptTokens + tt.wantCompletionTokens
			if result.Usage.TotalTokens != expectedTotal {
				t.Errorf("TotalTokens = %d, want %d", result.Usage.TotalTokens, expectedTotal)
			}
		})
	}
}

func TestOllamaStreamAdapter_Recv(t *testing.T) {
	tests := []struct {
		name    string
		chunk   *agent.ChatChunk
		wantEOF bool
	}{
		{
			name:    "first recv returns chunk",
			chunk:   &agent.ChatChunk{Content: "test"},
			wantEOF: false,
		},
		{
			name:    "second recv returns EOF",
			chunk:   &agent.ChatChunk{Content: "test"},
			wantEOF: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "second recv returns EOF" {
				adapter := &ollamaStreamAdapter{chunk: &agent.ChatChunk{Content: "test"}, done: false}
				_, _ = adapter.Recv()
				_, err := adapter.Recv()
				if err != io.EOF {
					t.Errorf("expected EOF, got %v", err)
				}
				return
			}

			adapter := &ollamaStreamAdapter{chunk: tt.chunk, done: false}
			chunk, err := adapter.Recv()
			if err != nil {
				t.Errorf("Recv() error = %v", err)
			}
			if chunk.Content != tt.chunk.Content {
				t.Errorf("Content = %q, want %q", chunk.Content, tt.chunk.Content)
			}
		})
	}
}

func TestOllamaStreamAdapter_Close(t *testing.T) {
	adapter := &ollamaStreamAdapter{chunk: &agent.ChatChunk{Content: "test"}}
	err := adapter.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
