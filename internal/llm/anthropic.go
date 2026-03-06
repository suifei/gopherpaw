// Package llm provides LLM provider implementations and registry.
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

const anthropicProviderName = "anthropic"

// AnthropicProvider implements agent.LLMProvider using Anthropic's Claude API.
type AnthropicProvider struct {
	client anthropic.Client
	cfg    config.LLMConfig
}

// NewAnthropic creates an Anthropic Claude LLM provider.
func NewAnthropic(cfg config.LLMConfig) (*AnthropicProvider, error) {
	opts := []option.RequestOption{
		option.WithAPIKey(cfg.APIKey),
	}
	if cfg.BaseURL != "" {
		opts = append(opts, option.WithBaseURL(cfg.BaseURL))
	}
	client := anthropic.NewClient(opts...)
	return &AnthropicProvider{
		client: client,
		cfg:    cfg,
	}, nil
}

// Name returns the provider identifier.
func (p *AnthropicProvider) Name() string {
	return anthropicProviderName
}

// Chat sends a chat completion request with automatic retry on transient errors.
func (p *AnthropicProvider) Chat(ctx context.Context, req *agent.ChatRequest) (*agent.ChatResponse, error) {
	log := logger.L()

	// Convert to Anthropic format
	anthropicMsgs, systemPrompt := p.toAnthropicMessages(req)

	log.Debug("Anthropic LLM request",
		zap.String("model", p.cfg.Model),
		zap.Int("messageCount", len(req.Messages)),
		zap.Int("toolCount", len(req.Tools)),
		zap.Float64("temperature", req.Temperature),
		zap.Int("maxTokens", req.MaxTokens),
	)

	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			delay := retryBaseDelay * time.Duration(1<<(attempt-1))
			log.Warn("Anthropic LLM retrying after transient error",
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

		// Build params
		params := anthropic.MessageNewParams{
			Model:     anthropic.Model(p.cfg.Model),
			MaxTokens: int64(req.MaxTokens),
			Messages:  anthropicMsgs,
		}
		if systemPrompt != "" {
			params.System = []anthropic.TextBlockParam{
				{Text: systemPrompt},
			}
		}
		if req.Temperature > 0 {
			params.Temperature = anthropic.Float(req.Temperature)
		}
		if len(req.Tools) > 0 {
			params.Tools = p.toAnthropicTools(req.Tools)
			// Handle tool_choice
			tc := p.toAnthropicToolChoice(req.ToolChoice, req.Tools)
			if tc.OfAuto != nil || tc.OfAny != nil || tc.OfNone != nil || tc.OfTool != nil {
				params.ToolChoice = tc
			}
		}

		resp, err := p.client.Messages.New(ctx, params)
		elapsed := time.Since(start)

		if err != nil {
			lastErr = err
			log.Error("Anthropic LLM request failed",
				zap.Error(err),
				zap.Duration("elapsed", elapsed),
				zap.Int("attempt", attempt+1),
			)
			if p.isTransientError(err) && attempt < maxRetries {
				continue
			}
			return nil, fmt.Errorf("anthropic chat: %w", err)
		}

		agentResp := p.toAgentResponse(resp)
		log.Debug("Anthropic LLM response",
			zap.Duration("elapsed", elapsed),
			zap.Int("contentLen", len(agentResp.Content)),
			zap.Int("toolCalls", len(agentResp.ToolCalls)),
			zap.Int("inputTokens", agentResp.Usage.PromptTokens),
			zap.Int("outputTokens", agentResp.Usage.CompletionTokens),
		)

		return agentResp, nil
	}
	return nil, fmt.Errorf("anthropic chat: all %d attempts failed: %w", maxRetries+1, lastErr)
}

// ChatStream sends a streaming chat completion request.
func (p *AnthropicProvider) ChatStream(ctx context.Context, req *agent.ChatRequest) (agent.ChatStream, error) {
	anthropicMsgs, systemPrompt := p.toAnthropicMessages(req)

	params := anthropic.MessageNewParams{
		Model:     anthropic.Model(p.cfg.Model),
		MaxTokens: int64(req.MaxTokens),
		Messages:  anthropicMsgs,
	}
	if systemPrompt != "" {
		params.System = []anthropic.TextBlockParam{
			{Text: systemPrompt},
		}
	}
	if req.Temperature > 0 {
		params.Temperature = anthropic.Float(req.Temperature)
	}
	if len(req.Tools) > 0 {
		params.Tools = p.toAnthropicTools(req.Tools)
		tc := p.toAnthropicToolChoice(req.ToolChoice, req.Tools)
		if tc.OfAuto != nil || tc.OfAny != nil || tc.OfNone != nil || tc.OfTool != nil {
			params.ToolChoice = tc
		}
	}

	stream := p.client.Messages.NewStreaming(ctx, params)
	return &anthropicStreamAdapter{
		stream:  stream,
		message: &anthropic.Message{},
	}, nil
}

// toAnthropicMessages converts agent.ChatRequest messages to Anthropic format.
// Returns the messages and extracted system prompt.
func (p *AnthropicProvider) toAnthropicMessages(req *agent.ChatRequest) ([]anthropic.MessageParam, string) {
	var systemPrompt string
	messages := make([]anthropic.MessageParam, 0, len(req.Messages))

	// Anthropic requires alternating user/assistant messages
	// System messages should be passed via the system parameter
	for _, m := range req.Messages {
		switch m.Role {
		case "system":
			// Accumulate system messages
			if systemPrompt != "" {
				systemPrompt += "\n\n"
			}
			systemPrompt += m.Content
		case "user":
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(m.Content)))
		case "assistant":
			if len(m.ToolCalls) > 0 {
				// Assistant message with tool calls
				blocks := make([]anthropic.ContentBlockParamUnion, 0)
				if m.Content != "" {
					blocks = append(blocks, anthropic.NewTextBlock(m.Content))
				}
				for _, tc := range m.ToolCalls {
					// Parse input JSON
					var input map[string]interface{}
					if err := json.Unmarshal([]byte(tc.Arguments), &input); err != nil {
						input = map[string]interface{}{"raw": tc.Arguments}
					}
					blocks = append(blocks, anthropic.NewToolUseBlock(tc.ID, input, tc.Name))
				}
				messages = append(messages, anthropic.MessageParam{
					Role:    anthropic.MessageParamRoleAssistant,
					Content: blocks,
				})
			} else {
				messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(m.Content)))
			}
		case "tool":
			// Tool result - needs to be in a user message
			// Check if we can append to the previous user message or create new one
			toolResult := anthropic.NewToolResultBlock(m.ToolCallID, m.Content, false)
			if len(messages) > 0 {
				lastMsg := &messages[len(messages)-1]
				if lastMsg.Role == anthropic.MessageParamRoleUser {
					// Append to existing user message
					lastMsg.Content = append(lastMsg.Content, toolResult)
					continue
				}
			}
			// Create new user message with tool result
			messages = append(messages, anthropic.NewUserMessage(toolResult))
		}
	}

	return messages, systemPrompt
}

// toAnthropicTools converts agent.ToolDef to Anthropic format.
func (p *AnthropicProvider) toAnthropicTools(tools []agent.ToolDef) []anthropic.ToolUnionParam {
	result := make([]anthropic.ToolUnionParam, len(tools))
	for i, t := range tools {
		// Convert parameters to InputSchema format
		var props map[string]interface{}
		if t.Parameters != nil {
			if m, ok := t.Parameters.(map[string]interface{}); ok {
				if p, ok := m["properties"].(map[string]interface{}); ok {
					props = p
				}
			}
		}
		if props == nil {
			props = make(map[string]interface{})
		}

		result[i] = anthropic.ToolUnionParam{
			OfTool: &anthropic.ToolParam{
				Name:        t.Name,
				Description: anthropic.String(t.Description),
				InputSchema: anthropic.ToolInputSchemaParam{
					Properties: props,
				},
			},
		}
	}
	return result
}

// toAnthropicToolChoice converts agent.ToolChoice to Anthropic format.
func (p *AnthropicProvider) toAnthropicToolChoice(tc *agent.ToolChoice, tools []agent.ToolDef) anthropic.ToolChoiceUnionParam {
	if tc == nil {
		return anthropic.ToolChoiceUnionParam{}
	}
	if tc.ForceTool != "" {
		// Force specific tool - validate it exists
		for _, t := range tools {
			if t.Name == tc.ForceTool {
				return anthropic.ToolChoiceParamOfTool(tc.ForceTool)
			}
		}
		return anthropic.ToolChoiceUnionParam{}
	}
	switch tc.Mode {
	case agent.ToolChoiceNone:
		return anthropic.ToolChoiceUnionParam{
			OfNone: &anthropic.ToolChoiceNoneParam{},
		}
	case agent.ToolChoiceRequired:
		return anthropic.ToolChoiceUnionParam{
			OfAny: &anthropic.ToolChoiceAnyParam{},
		}
	case agent.ToolChoiceAuto:
		return anthropic.ToolChoiceUnionParam{
			OfAuto: &anthropic.ToolChoiceAutoParam{},
		}
	}
	return anthropic.ToolChoiceUnionParam{}
}

// toAgentResponse converts Anthropic response to agent.ChatResponse.
func (p *AnthropicProvider) toAgentResponse(resp *anthropic.Message) *agent.ChatResponse {
	var content strings.Builder
	var toolCalls []agent.ToolCall

	for _, block := range resp.Content {
		switch b := block.AsAny().(type) {
		case anthropic.TextBlock:
			content.WriteString(b.Text)
		case anthropic.ToolUseBlock:
			// Convert Input to JSON string
			inputJSON, _ := json.Marshal(b.Input)
			toolCalls = append(toolCalls, agent.ToolCall{
				ID:        b.ID,
				Name:      b.Name,
				Arguments: string(inputJSON),
			})
		}
	}

	return &agent.ChatResponse{
		Content:   content.String(),
		ToolCalls: toolCalls,
		Usage: agent.Usage{
			PromptTokens:     int(resp.Usage.InputTokens),
			CompletionTokens: int(resp.Usage.OutputTokens),
			TotalTokens:      int(resp.Usage.InputTokens + resp.Usage.OutputTokens),
		},
	}
}

// isTransientError checks if the error is retryable.
func (p *AnthropicProvider) isTransientError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	for _, code := range []string{"429", "500", "502", "503", "504", "overloaded"} {
		if strings.Contains(s, code) {
			return true
		}
	}
	return false
}

// anthropicStreamAdapter adapts Anthropic streaming to agent.ChatStream.
type anthropicStreamAdapter struct {
	stream  *ssestream.Stream[anthropic.MessageStreamEventUnion]
	message *anthropic.Message
	done    bool
	err     error

	// Accumulator for tool call JSON
	currentToolID   string
	currentToolName string
	currentToolJSON strings.Builder
}

// Recv receives the next streaming chunk.
func (s *anthropicStreamAdapter) Recv() (*agent.ChatChunk, error) {
	if s.done {
		return nil, io.EOF
	}
	if s.err != nil {
		return nil, s.err
	}

	for s.stream.Next() {
		event := s.stream.Current()

		// Accumulate into message
		if err := s.message.Accumulate(event); err != nil {
			s.err = err
			return nil, err
		}

		switch ev := event.AsAny().(type) {
		case anthropic.ContentBlockStartEvent:
			// Tool use block starts
			if ev.ContentBlock.Type == "tool_use" {
				s.currentToolID = ev.ContentBlock.ID
				s.currentToolName = ev.ContentBlock.Name
				s.currentToolJSON.Reset()
			}

		case anthropic.ContentBlockDeltaEvent:
			// Text delta
			if ev.Delta.Text != "" {
				return &agent.ChatChunk{
					Content: ev.Delta.Text,
				}, nil
			}
			// Tool use partial JSON
			if ev.Delta.PartialJSON != "" {
				s.currentToolJSON.WriteString(ev.Delta.PartialJSON)
			}

		case anthropic.ContentBlockStopEvent:
			// If we were building a tool call, emit it
			if s.currentToolID != "" {
				tc := agent.ToolCall{
					ID:        s.currentToolID,
					Name:      s.currentToolName,
					Arguments: s.currentToolJSON.String(),
				}
				s.currentToolID = ""
				s.currentToolName = ""
				s.currentToolJSON.Reset()
				return &agent.ChatChunk{
					ToolCalls: []agent.ToolCall{tc},
				}, nil
			}

		case anthropic.MessageStopEvent:
			s.done = true
			// Stream finished - no more content to return
			return nil, io.EOF
		}
	}

	if err := s.stream.Err(); err != nil {
		s.err = err
		return nil, err
	}

	s.done = true
	return nil, io.EOF
}

// Close closes the stream.
func (s *anthropicStreamAdapter) Close() error {
	s.done = true
	if s.stream != nil {
		return s.stream.Close()
	}
	return nil
}
