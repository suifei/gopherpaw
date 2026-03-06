// Package agent provides the core Agent runtime, ReAct loop, and domain types.
package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// ReactAgent implements Agent with a ReAct loop: Thought -> Action -> Observation -> ... -> Final Answer.
type ReactAgent struct {
	llmMu         sync.RWMutex
	llm           LLMProvider
	memory        MemoryStore
	tools         []Tool
	toolMap       map[string]Tool
	cfg           config.AgentConfig
	loader        *PromptLoader
	skillsContent string
	hooks         []Hook
}

// NewReact creates a ReAct agent with the given dependencies.
func NewReact(llm LLMProvider, memory MemoryStore, tools []Tool, cfg config.AgentConfig) *ReactAgent {
	return NewReactWithPrompt(llm, memory, tools, cfg, nil, "")
}

// NewReactWithPrompt creates a ReAct agent with optional PromptLoader and skills content.
func NewReactWithPrompt(llm LLMProvider, memory MemoryStore, tools []Tool, cfg config.AgentConfig, loader *PromptLoader, skillsContent string) *ReactAgent {
	toolMap := make(map[string]Tool)
	for _, t := range tools {
		toolMap[t.Name()] = t
	}
	return &ReactAgent{
		llm:           llm,
		memory:        memory,
		tools:         tools,
		toolMap:       toolMap,
		cfg:           cfg,
		loader:        loader,
		skillsContent: skillsContent,
		hooks:         nil,
	}
}

// AddHook registers a hook to be called before each ReAct turn.
// Hooks are executed in the order they are added.
func (a *ReactAgent) AddHook(h Hook) {
	a.hooks = append(a.hooks, h)
}

// AddHooks registers multiple hooks.
func (a *ReactAgent) AddHooks(hooks ...Hook) {
	a.hooks = append(a.hooks, hooks...)
}

// Run processes a message through the ReAct loop and returns the final response.
func (a *ReactAgent) Run(ctx context.Context, chatID string, message string) (string, error) {
	log := logger.L()
	reporter := getProgressReporter(ctx)

	if err := ctx.Err(); err != nil {
		return "", err
	}
	if chatID == "" {
		return "", fmt.Errorf("chatID cannot be empty")
	}
	if message == "" {
		return "", fmt.Errorf("message cannot be empty")
	}

	// Handle magic commands (e.g. /compact, /new, /clear, /history, /daemon)
	if result, handled, err := HandleMagicCommand(ctx, a.memory, chatID, message, getDaemonInfo(ctx)); handled {
		if err != nil {
			return "", err
		}
		return result, nil
	}

	log.Info("Agent processing message",
		zap.String("chatID", chatID),
		zap.Int("msgLen", len(message)),
		zap.String("lastUserMsg", truncate(message, 200)),
	)
	if reporter != nil {
		reporter.OnThinking()
	}

	// Save user message
	userMsg := Message{Role: "user", Content: message}
	if err := a.memory.Save(ctx, chatID, userMsg); err != nil {
		return "", fmt.Errorf("save user message: %w", err)
	}

	messages, err := a.buildMessages(ctx, chatID)
	if err != nil {
		return "", err
	}

	toolDefs := a.toolsToDefs()
	maxTurns := a.cfg.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 20
	}

	var finalContent string
	for turn := 0; turn < maxTurns; turn++ {
		if err := ctx.Err(); err != nil {
			return "", err
		}

		// Execute hooks before each turn
		for _, hook := range a.hooks {
			messages, err = hook(ctx, a, chatID, messages)
			if err != nil {
				return "", fmt.Errorf("hook error: %w", err)
			}
		}

		log.Debug("ReAct turn",
			zap.Int("turn", turn+1),
			zap.Int("maxTurns", maxTurns),
			zap.Int("messages", len(messages)),
			zap.Int("toolCount", len(toolDefs)),
		)
		lastUser := lastUserMessage(messages)
		log.Debug("Sending to LLM",
			zap.Int("messageCount", len(messages)),
			zap.Int("toolCount", len(toolDefs)),
			zap.String("lastUserMsg", truncate(lastUser, 200)),
		)

		req := &ChatRequest{
			Messages:    messages,
			Tools:       toolDefs,
			Temperature: 0.7,
			MaxTokens:   4096,
		}

		a.llmMu.RLock()
		provider := a.llm
		a.llmMu.RUnlock()
		resp, err := provider.Chat(ctx, req)
		if err != nil {
			return "", fmt.Errorf("llm chat: %w", err)
		}

		log.Debug("LLM response",
			zap.Int("contentLen", len(resp.Content)),
			zap.Int("toolCalls", len(resp.ToolCalls)),
			zap.Int("promptTokens", resp.Usage.PromptTokens),
			zap.Int("completionTokens", resp.Usage.CompletionTokens),
			zap.String("contentPreview", truncate(resp.Content, 200)),
		)

		// Save assistant message
		assistantMsg := Message{
			Role:      "assistant",
			Content:   resp.Content,
			ToolCalls: resp.ToolCalls,
		}
		if err := a.memory.Save(ctx, chatID, assistantMsg); err != nil {
			return "", fmt.Errorf("save assistant message: %w", err)
		}

		if len(resp.ToolCalls) == 0 {
			finalContent = strings.TrimSpace(resp.Content)
			log.Info("Final answer", zap.String("content", truncate(finalContent, 200)))
			break
		}

		// Append assistant message, then execute tools in parallel and append results
		messages = append(messages, assistantMsg)
		toolResults := a.executeToolsParallel(ctx, chatID, resp.ToolCalls, reporter)

		// Append tool results in order (matching tool_call order)
		for i, tr := range toolResults {
			log.Debug("Tool result",
				zap.String("tool", resp.ToolCalls[i].Name),
				zap.String("result", truncate(tr.Result, 500)),
			)
			if reporter != nil {
				reporter.OnToolResult(resp.ToolCalls[i].Name, tr.Result)
			}

			toolMsg := Message{
				Role:       "tool",
				Content:    tr.Result,
				ToolCallID: resp.ToolCalls[i].ID,
			}
			messages = append(messages, toolMsg)
			if err := a.memory.Save(ctx, chatID, toolMsg); err != nil {
				return "", fmt.Errorf("save tool message: %w", err)
			}
		}
	}

	if finalContent == "" {
		finalContent = "I'm sorry, I couldn't generate a response. Please try again."
		log.Warn("No final content after max turns")
	}
	if reporter != nil {
		reporter.OnFinalReply(finalContent)
	}
	return finalContent, nil
}

func lastUserMessage(msgs []Message) string {
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i].Role == "user" {
			return msgs[i].Content
		}
	}
	return ""
}

func (a *ReactAgent) executeTool(ctx context.Context, chatID string, tc ToolCall) (string, error) {
	tool, ok := a.toolMap[tc.Name]
	if !ok {
		return "", fmt.Errorf("unknown tool %q", tc.Name)
	}
	ctx = WithMemoryStore(ctx, a.memory)
	ctx = WithChatID(ctx, chatID)

	a.llmMu.RLock()
	if ms, ok := a.llm.(ModelSwitcher); ok {
		ctx = WithModelSwitcher(ctx, ms)
	}
	a.llmMu.RUnlock()

	if rich, ok := tool.(RichExecutor); ok {
		result, err := rich.ExecuteRich(ctx, tc.Arguments)
		if err != nil {
			return "", err
		}
		if sender := GetFileSender(ctx); sender != nil {
			for _, att := range result.Attachments {
				if sendErr := sender(ctx, att); sendErr != nil {
					logger.L().Warn("send attachment failed",
						zap.String("tool", tc.Name),
						zap.String("file", att.FilePath),
						zap.Error(sendErr),
					)
				}
			}
		}
		return result.Text, nil
	}

	return tool.Execute(ctx, tc.Arguments)
}

// toolResult holds the result of a parallel tool execution.
type toolResult struct {
	Result string
	Err    error
}

// executeToolsParallel executes multiple tool calls concurrently and returns
// results in the same order as the input toolCalls slice.
func (a *ReactAgent) executeToolsParallel(ctx context.Context, chatID string, toolCalls []ToolCall, reporter ProgressReporter) []toolResult {
	log := logger.L()
	n := len(toolCalls)
	if n == 0 {
		return nil
	}

	// For single tool, no need for goroutines
	if n == 1 {
		tc := toolCalls[0]
		log.Info("Calling tool",
			zap.String("tool", tc.Name),
			zap.String("args", truncate(tc.Arguments, 500)),
		)
		if reporter != nil {
			reporter.OnToolCall(tc.Name, tc.Arguments)
		}
		result, err := a.executeTool(ctx, chatID, tc)
		if err != nil {
			result = fmt.Sprintf("Error: %v", err)
		}
		return []toolResult{{Result: result, Err: err}}
	}

	// Parallel execution for multiple tools
	log.Info("Executing tools in parallel", zap.Int("count", n))
	results := make([]toolResult, n)
	var wg sync.WaitGroup
	wg.Add(n)

	for i, tc := range toolCalls {
		go func(idx int, tc ToolCall) {
			defer wg.Done()

			log.Info("Calling tool (parallel)",
				zap.Int("index", idx),
				zap.String("tool", tc.Name),
				zap.String("args", truncate(tc.Arguments, 500)),
			)
			if reporter != nil {
				reporter.OnToolCall(tc.Name, tc.Arguments)
			}

			result, err := a.executeTool(ctx, chatID, tc)
			if err != nil {
				result = fmt.Sprintf("Error: %v", err)
			}
			results[idx] = toolResult{Result: result, Err: err}
		}(i, tc)
	}

	wg.Wait()
	return results
}

func (a *ReactAgent) buildMessages(ctx context.Context, chatID string) ([]Message, error) {
	history, err := a.memory.Load(ctx, chatID, a.cfg.MaxTurns*4) // rough limit
	if err != nil {
		return nil, fmt.Errorf("load history: %w", err)
	}

	// Sanitize tool messages to ensure proper pairing
	history = SanitizeToolMessages(history)

	// Context window check: compact when estimated tokens exceed 80% of maxInputLength
	if a.cfg.MaxInputLength > 0 {
		sysPrompt := a.getSystemPrompt()
		estimated := estimateTokens(sysPrompt) + estimateMessagesTokens(history)
		threshold := int(float64(a.cfg.MaxInputLength) * 0.8)
		if estimated > threshold {
			logger.L().Info("Context near limit, compacting memory",
				zap.Int("estimated", estimated),
				zap.Int("threshold", threshold),
			)
			if err := a.memory.Compact(ctx, chatID); err != nil {
				logger.L().Warn("Compact failed", zap.Error(err))
			} else {
				history, err = a.memory.Load(ctx, chatID, a.cfg.MaxTurns*4)
				if err != nil {
					return nil, fmt.Errorf("load history after compact: %w", err)
				}
				// Re-sanitize after reload
				history = SanitizeToolMessages(history)
			}
		}
	}

	messages := make([]Message, 0, len(history)+2)
	messages = append(messages, Message{Role: "system", Content: a.getSystemPrompt()})
	messages = append(messages, history...)
	return messages, nil
}

func (a *ReactAgent) getSystemPrompt() string {
	var base string
	if a.loader != nil {
		base = a.loader.BuildSystemPrompt(a.skillsContent)
	}
	if base == "" {
		base = a.cfg.SystemPrompt
	}
	if base == "" {
		base = "You are a helpful AI assistant."
	}
	return base
}

func estimateTokens(s string) int {
	return CountStringTokens(s)
}

func estimateMessagesTokens(msgs []Message) int {
	return CountMessageTokens(msgs)
}

func (a *ReactAgent) toolsToDefs() []ToolDef {
	defs := make([]ToolDef, len(a.tools))
	for i, t := range a.tools {
		defs[i] = ToolDef{
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
		}
	}
	return defs
}

// SetLLMProvider replaces the LLM provider at runtime (for /daemon or config hot-switch).
func (a *ReactAgent) SetLLMProvider(llm LLMProvider) {
	if llm != nil {
		a.llmMu.Lock()
		a.llm = llm
		a.llmMu.Unlock()
	}
}

// RunStream processes a message and streams response chunks.
// For now, delegates to Run and sends the final result as one chunk.
func (a *ReactAgent) RunStream(ctx context.Context, chatID string, message string) (<-chan string, error) {
	ch := make(chan string, 1)
	go func() {
		defer close(ch)
		result, err := a.Run(ctx, chatID, message)
		if err != nil {
			select {
			case ch <- "Error: " + err.Error():
			case <-ctx.Done():
			}
			return
		}
		select {
		case ch <- result:
		case <-ctx.Done():
		}
	}()
	return ch, nil
}

// truncate limits s to at most maxLen runes, appending "..." if truncated.
func truncate(s string, maxLen int) string {
	if maxLen <= 0 || utf8.RuneCountInString(s) <= maxLen {
		return s
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// Ensure ReactAgent implements Agent.
var _ Agent = (*ReactAgent)(nil)
