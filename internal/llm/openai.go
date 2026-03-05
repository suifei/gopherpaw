// Package llm provides LLM provider implementations and registry.
package llm

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/sashabaranov/go-openai"
	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

const providerName = "openai"

// OpenAIProvider implements agent.LLMProvider using go-openai.
type OpenAIProvider struct {
	client *openai.Client
	cfg    config.LLMConfig
}

// NewOpenAI creates an OpenAI-compatible LLM provider.
// BaseURL can be set for DashScope, ModelScope, Ollama, etc.
func NewOpenAI(cfg config.LLMConfig) *OpenAIProvider {
	clientCfg := openai.DefaultConfig(cfg.APIKey)
	if cfg.BaseURL != "" {
		clientCfg.BaseURL = cfg.BaseURL
	}
	return &OpenAIProvider{
		client: openai.NewClientWithConfig(clientCfg),
		cfg:   cfg,
	}
}

// Name returns the provider identifier.
func (p *OpenAIProvider) Name() string {
	return providerName
}

const (
	maxRetries     = 3
	retryBaseDelay = 2 * time.Second
)

// isTransientError checks if the error message indicates a retryable server error.
func isTransientError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	for _, code := range []string{"status code: 429", "status code: 500", "status code: 502", "status code: 503", "status code: 504"} {
		if strings.Contains(s, code) {
			return true
		}
	}
	return false
}

// Chat sends a chat completion request with automatic retry on transient errors.
func (p *OpenAIProvider) Chat(ctx context.Context, req *agent.ChatRequest) (*agent.ChatResponse, error) {
	log := logger.L()
	openaiReq := toOpenAIRequest(req, p.cfg.Model)

	log.Debug("LLM request",
		zap.String("model", p.cfg.Model),
		zap.Int("messageCount", len(req.Messages)),
		zap.Int("toolCount", len(req.Tools)),
		zap.Float64("temperature", req.Temperature),
		zap.Int("maxTokens", req.MaxTokens),
	)
	if log.Core().Enabled(zap.DebugLevel) && len(req.Messages) > 0 {
		for i, m := range req.Messages {
			preview := truncateStr(m.Content, 300)
			log.Debug("LLM message",
				zap.Int("idx", i),
				zap.String("role", m.Role),
				zap.String("contentPreview", preview),
			)
		}
	}

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryBaseDelay * time.Duration(1<<(attempt-1))
			log.Warn("LLM retrying after transient error",
				zap.Int("attempt", attempt),
				zap.Duration("delay", delay),
				zap.Error(lastErr),
			)
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		start := time.Now()
		resp, err := p.client.CreateChatCompletion(ctx, openaiReq)
		elapsed := time.Since(start)

		if err != nil {
			lastErr = err
			log.Error("LLM request failed",
				zap.Error(err),
				zap.Duration("elapsed", elapsed),
				zap.Int("attempt", attempt+1),
				zap.Int("maxAttempts", maxRetries+1),
			)
			if isTransientError(err) && attempt < maxRetries {
				continue
			}
			return nil, fmt.Errorf("chat completion: %w", err)
		}

		agentResp := toOpenAIAgentResponse(&resp)
		log.Debug("LLM response",
			zap.Duration("elapsed", elapsed),
			zap.Int("contentLen", len(agentResp.Content)),
			zap.Int("toolCalls", len(agentResp.ToolCalls)),
			zap.Int("promptTokens", agentResp.Usage.PromptTokens),
			zap.Int("completionTokens", agentResp.Usage.CompletionTokens),
			zap.Int("totalTokens", agentResp.Usage.TotalTokens),
		)
		if log.Core().Enabled(zap.DebugLevel) && agentResp.Content != "" {
			log.Debug("LLM content preview", zap.String("content", truncateStr(agentResp.Content, 300)))
		}

		return agentResp, nil
	}
	return nil, fmt.Errorf("chat completion: all %d attempts failed: %w", maxRetries+1, lastErr)
}

func truncateStr(s string, maxLen int) string {
	if maxLen <= 0 || utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// ChatStream sends a streaming chat completion request.
func (p *OpenAIProvider) ChatStream(ctx context.Context, req *agent.ChatRequest) (agent.ChatStream, error) {
	openaiReq := toOpenAIRequest(req, p.cfg.Model)
	openaiReq.Stream = true
	stream, err := p.client.CreateChatCompletionStream(ctx, openaiReq)
	if err != nil {
		return nil, fmt.Errorf("chat stream: %w", err)
	}
	return &openAIStreamAdapter{stream: stream}, nil
}

func toOpenAIRequest(req *agent.ChatRequest, model string) openai.ChatCompletionRequest {
	msgs := make([]openai.ChatCompletionMessage, len(req.Messages))
	for i, m := range req.Messages {
		msgs[i] = openai.ChatCompletionMessage{
			Role:       m.Role,
			Content:    m.Content,
			ToolCalls:  toOpenAIToolCalls(m.ToolCalls),
			ToolCallID: m.ToolCallID,
		}
	}
	r := openai.ChatCompletionRequest{
		Model:       model,
		Messages:    msgs,
		Temperature: float32(req.Temperature),
		MaxTokens:  req.MaxTokens,
	}
	if len(req.Tools) > 0 {
		r.Tools = make([]openai.Tool, len(req.Tools))
		for i, t := range req.Tools {
			r.Tools[i] = openai.Tool{
				Type: openai.ToolTypeFunction,
				Function: &openai.FunctionDefinition{
					Name:        t.Name,
					Description: t.Description,
					Parameters:  t.Parameters,
				},
			}
		}
	}
	return r
}

func toOpenAIToolCalls(tcs []agent.ToolCall) []openai.ToolCall {
	if len(tcs) == 0 {
		return nil
	}
	out := make([]openai.ToolCall, len(tcs))
	for i, tc := range tcs {
		out[i] = openai.ToolCall{
			ID:   tc.ID,
			Type: openai.ToolTypeFunction,
			Function: openai.FunctionCall{
				Name:      tc.Name,
				Arguments: tc.Arguments,
			},
		}
	}
	return out
}

func toOpenAIAgentResponse(resp *openai.ChatCompletionResponse) *agent.ChatResponse {
	out := &agent.ChatResponse{
		Usage: agent.Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}
	if len(resp.Choices) > 0 {
		msg := resp.Choices[0].Message
		out.Content = msg.Content
		out.ToolCalls = toAgentToolCalls(msg.ToolCalls)
	}
	return out
}

func toAgentToolCalls(tcs []openai.ToolCall) []agent.ToolCall {
	if len(tcs) == 0 {
		return nil
	}
	out := make([]agent.ToolCall, len(tcs))
	for i, tc := range tcs {
		out[i] = agent.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		}
	}
	return out
}

type openAIStreamAdapter struct {
	stream *openai.ChatCompletionStream
}

func (a *openAIStreamAdapter) Recv() (*agent.ChatChunk, error) {
	resp, err := a.stream.Recv()
	if err != nil {
		if err == io.EOF {
			return nil, io.EOF
		}
		return nil, err
	}
	return toAgentChunk(&resp), nil
}

func (a *openAIStreamAdapter) Close() error {
	return a.stream.Close()
}

func toAgentChunk(resp *openai.ChatCompletionStreamResponse) *agent.ChatChunk {
	out := &agent.ChatChunk{}
	if len(resp.Choices) > 0 {
		delta := resp.Choices[0].Delta
		out.Content = delta.Content
		if len(delta.ToolCalls) > 0 {
			out.ToolCalls = make([]agent.ToolCall, len(delta.ToolCalls))
			for i, tc := range delta.ToolCalls {
				out.ToolCalls[i] = agent.ToolCall{
					ID:        tc.ID,
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				}
			}
		}
	}
	return out
}
