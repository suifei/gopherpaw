// Package llm provides LLM provider implementations and registry.
package llm

import (
	"context"
	"encoding/json"
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
		cfg:    cfg,
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
		MaxTokens:   req.MaxTokens,
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
		// Handle tool_choice
		r.ToolChoice = toOpenAIToolChoice(req.ToolChoice, req.Tools)
	}
	return r
}

// toOpenAIToolChoice converts agent.ToolChoice to OpenAI format.
// Returns nil for default behavior (auto).
func toOpenAIToolChoice(tc *agent.ToolChoice, tools []agent.ToolDef) any {
	if tc == nil {
		return nil // default to auto
	}
	// Force specific tool
	if tc.ForceTool != "" {
		// Validate tool exists
		found := false
		for _, t := range tools {
			if t.Name == tc.ForceTool {
				found = true
				break
			}
		}
		if found {
			return openai.ToolChoice{
				Type: openai.ToolTypeFunction,
				Function: openai.ToolFunction{
					Name: tc.ForceTool,
				},
			}
		}
		// Tool not found, fall back to auto
		return nil
	}
	// Handle mode
	switch tc.Mode {
	case agent.ToolChoiceNone:
		return "none"
	case agent.ToolChoiceRequired:
		return "required"
	case agent.ToolChoiceAuto:
		return "auto"
	default:
		return nil
	}
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

// streamToolCallAccumulator accumulates tool_call fragments during streaming.
// OpenAI streaming API sends tool_calls in chunks:
//   - First chunk: Index, ID, Function.Name
//   - Subsequent chunks: Function.Arguments fragments
type streamToolCallAccumulator struct {
	// toolCalls accumulates tool_calls by Index
	toolCalls map[int]*agent.ToolCall
}

func newStreamToolCallAccumulator() *streamToolCallAccumulator {
	return &streamToolCallAccumulator{
		toolCalls: make(map[int]*agent.ToolCall),
	}
}

// accumulate merges streaming tool_call deltas into accumulated state.
// Returns the current snapshot of tool_calls (with accumulated Arguments).
func (acc *streamToolCallAccumulator) accumulate(deltas []openai.ToolCall) []agent.ToolCall {
	for _, delta := range deltas {
		idx := *delta.Index
		existing, ok := acc.toolCalls[idx]
		if !ok {
			// New tool_call: initialize with ID and Name
			acc.toolCalls[idx] = &agent.ToolCall{
				ID:        delta.ID,
				Name:      delta.Function.Name,
				Arguments: delta.Function.Arguments,
			}
		} else {
			// Accumulate Arguments fragments
			if delta.ID != "" {
				existing.ID = delta.ID
			}
			if delta.Function.Name != "" {
				existing.Name = delta.Function.Name
			}
			existing.Arguments += delta.Function.Arguments
		}
	}
	return acc.snapshot()
}

// snapshot returns the current accumulated tool_calls as a slice.
func (acc *streamToolCallAccumulator) snapshot() []agent.ToolCall {
	if len(acc.toolCalls) == 0 {
		return nil
	}
	// Find max index to preserve order
	maxIdx := -1
	for idx := range acc.toolCalls {
		if idx > maxIdx {
			maxIdx = idx
		}
	}
	result := make([]agent.ToolCall, 0, maxIdx+1)
	for i := 0; i <= maxIdx; i++ {
		if tc, ok := acc.toolCalls[i]; ok {
			result = append(result, *tc)
		}
	}
	return result
}

// sanitize validates and repairs accumulated tool_calls.
// Removes tool_calls with empty ID or Name (invalid), and repairs malformed JSON Arguments.
func (acc *streamToolCallAccumulator) sanitize() []agent.ToolCall {
	if len(acc.toolCalls) == 0 {
		return nil
	}
	result := make([]agent.ToolCall, 0, len(acc.toolCalls))
	for _, tc := range acc.toolCalls {
		// Skip invalid tool_calls (missing ID or Name)
		if tc.ID == "" || tc.Name == "" {
			continue
		}
		// Repair malformed JSON Arguments
		if tc.Arguments != "" && tc.Arguments != "{}" {
			// Try to validate JSON; if invalid, try common fixes
			tc.Arguments = sanitizeJSONArguments(tc.Arguments)
		}
		result = append(result, *tc)
	}
	return result
}

// sanitizeJSONArguments attempts to fix common JSON issues from streaming.
func sanitizeJSONArguments(args string) string {
	args = strings.TrimSpace(args)
	if args == "" {
		return "{}"
	}
	// Check if valid JSON
	var tmp any
	if err := json.Unmarshal([]byte(args), &tmp); err == nil {
		return args // Already valid
	}
	// Try common fixes:
	// 1. Trim trailing incomplete content (e.g., truncated string)
	// 2. Try to close unclosed braces/brackets
	fixed := tryFixJSON(args)
	if fixed != "" {
		return fixed
	}
	// Last resort: return empty object
	return "{}"
}

// tryFixJSON attempts to fix common JSON truncation issues.
func tryFixJSON(s string) string {
	// Count braces/brackets
	openBraces := strings.Count(s, "{") - strings.Count(s, "}")
	openBrackets := strings.Count(s, "[") - strings.Count(s, "]")

	// If more opens than closes, try adding closing chars
	if openBraces > 0 || openBrackets > 0 {
		fixed := s
		for i := 0; i < openBrackets; i++ {
			fixed += "]"
		}
		for i := 0; i < openBraces; i++ {
			fixed += "}"
		}
		var tmp any
		if err := json.Unmarshal([]byte(fixed), &tmp); err == nil {
			return fixed
		}
	}
	return ""
}

type openAIStreamAdapter struct {
	stream      *openai.ChatCompletionStream
	accumulator *streamToolCallAccumulator
	finished    bool
}

func (a *openAIStreamAdapter) Recv() (*agent.ChatChunk, error) {
	if a.finished {
		return nil, io.EOF
	}

	resp, err := a.stream.Recv()
	if err != nil {
		if err == io.EOF {
			a.finished = true
			// On EOF, return final sanitized tool_calls if any accumulated
			if a.accumulator != nil && len(a.accumulator.toolCalls) > 0 {
				finalToolCalls := a.accumulator.sanitize()
				if len(finalToolCalls) > 0 {
					return &agent.ChatChunk{
						ToolCalls: finalToolCalls,
					}, nil
				}
			}
			return nil, io.EOF
		}
		return nil, err
	}

	chunk := a.toAgentChunk(&resp)
	return chunk, nil
}

func (a *openAIStreamAdapter) Close() error {
	return a.stream.Close()
}

func (a *openAIStreamAdapter) toAgentChunk(resp *openai.ChatCompletionStreamResponse) *agent.ChatChunk {
	out := &agent.ChatChunk{}
	if len(resp.Choices) > 0 {
		delta := resp.Choices[0].Delta
		out.Content = delta.Content

		// Accumulate tool_calls if present
		if len(delta.ToolCalls) > 0 {
			if a.accumulator == nil {
				a.accumulator = newStreamToolCallAccumulator()
			}
			// Accumulate and return current snapshot (partial tool_calls)
			out.ToolCalls = a.accumulator.accumulate(delta.ToolCalls)
		}
	}
	return out
}
