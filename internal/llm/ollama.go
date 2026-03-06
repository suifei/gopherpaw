// Package llm provides LLM provider implementations and registry.
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

const ollamaProviderName = "ollama"

// OllamaProvider implements agent.LLMProvider using Ollama HTTP API.
type OllamaProvider struct {
	baseURL string
	model   string
	client  *http.Client
}

// NewOllama creates an Ollama provider.
func NewOllama(cfg config.LLMConfig) (*OllamaProvider, error) {
	baseURL := cfg.OllamaURL
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	model := cfg.Model
	if model == "" {
		model = "llama3.2"
	}
	return &OllamaProvider{
		baseURL: baseURL,
		model:   model,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}, nil
}

// Name returns the provider identifier.
func (p *OllamaProvider) Name() string {
	return ollamaProviderName
}

// Chat sends a chat completion request to Ollama /api/chat.
func (p *OllamaProvider) Chat(ctx context.Context, req *agent.ChatRequest) (*agent.ChatResponse, error) {
	log := logger.L()
	ollamaReq := toOllamaRequest(req, p.model)

	log.Debug("Ollama request",
		zap.String("model", p.model),
		zap.Int("messageCount", len(req.Messages)),
		zap.Int("toolCount", len(req.Tools)),
	)

	body, err := json.Marshal(ollamaReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := p.client.Do(httpReq)
	elapsed := time.Since(start)
	if err != nil {
		log.Error("Ollama request failed", zap.Error(err), zap.Duration("elapsed", elapsed))
		return nil, fmt.Errorf("ollama chat: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("ollama status %d: %s", resp.StatusCode, string(b))
	}

	var ollamaResp ollamaChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	agentResp := toOllamaAgentResponse(&ollamaResp)
	log.Debug("Ollama response",
		zap.Duration("elapsed", elapsed),
		zap.Int("contentLen", len(agentResp.Content)),
		zap.Int("toolCalls", len(agentResp.ToolCalls)),
	)
	return agentResp, nil
}

// ChatStream sends a streaming request. For now, delegates to Chat.
func (p *OllamaProvider) ChatStream(ctx context.Context, req *agent.ChatRequest) (agent.ChatStream, error) {
	resp, err := p.Chat(ctx, req)
	if err != nil {
		return nil, err
	}
	return &ollamaStreamAdapter{chunk: &agent.ChatChunk{Content: resp.Content}}, nil
}

type ollamaChatRequest struct {
	Model       string          `json:"model"`
	Messages    []ollamaMessage `json:"messages"`
	Stream      bool            `json:"stream"`
	Tools       []ollamaTool    `json:"tools,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	NumPredict  int             `json:"num_predict,omitempty"`
}

type ollamaMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []ollamaToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type ollamaToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type ollamaTool struct {
	Type     string     `json:"type"`
	Function ollamaFunc `json:"function"`
}

type ollamaFunc struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters"`
}

type ollamaChatResponse struct {
	Message struct {
		Role      string           `json:"role"`
		Content   string           `json:"content"`
		ToolCalls []ollamaToolCall `json:"tool_calls,omitempty"`
	} `json:"message"`
	Done            bool `json:"done"`
	PromptEvalCount int  `json:"prompt_eval_count"`
	EvalCount       int  `json:"eval_count"`
}

func toOllamaRequest(req *agent.ChatRequest, model string) ollamaChatRequest {
	msgs := make([]ollamaMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = ollamaMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCallID: m.ToolCallID,
		}
		if len(m.ToolCalls) > 0 {
			msgs[i].ToolCalls = make([]ollamaToolCall, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				msgs[i].ToolCalls[j] = ollamaToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{Name: tc.Name, Arguments: tc.Arguments},
				}
			}
		}
	}
	out := ollamaChatRequest{
		Model:    model,
		Messages: msgs,
		Stream:   false,
	}
	if req.Temperature > 0 {
		out.Temperature = &req.Temperature
	}
	if req.MaxTokens > 0 {
		out.NumPredict = req.MaxTokens
	}
	if len(req.Tools) > 0 {
		out.Tools = make([]ollamaTool, len(req.Tools))
		for i, t := range req.Tools {
			out.Tools[i] = ollamaTool{
				Type: "function",
				Function: ollamaFunc{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.Parameters,
				},
			}
		}
	}
	return out
}

func toOllamaAgentResponse(r *ollamaChatResponse) *agent.ChatResponse {
	out := &agent.ChatResponse{
		Content: r.Message.Content,
		Usage: agent.Usage{
			PromptTokens:     r.PromptEvalCount,
			CompletionTokens: r.EvalCount,
			TotalTokens:      r.PromptEvalCount + r.EvalCount,
		},
	}
	for _, tc := range r.Message.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, agent.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}
	return out
}

type ollamaStreamAdapter struct {
	chunk *agent.ChatChunk
	done  bool
}

func (a *ollamaStreamAdapter) Recv() (*agent.ChatChunk, error) {
	if a.done {
		return nil, io.EOF
	}
	a.done = true
	return a.chunk, nil
}

func (a *ollamaStreamAdapter) Close() error {
	return nil
}

var _ agent.ChatStream = (*ollamaStreamAdapter)(nil)
