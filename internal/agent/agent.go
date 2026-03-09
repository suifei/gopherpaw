// Package agent provides the core Agent runtime, ReAct loop, and domain types.
package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// ReactAgent implements Agent with a ReAct loop: Thought -> Action -> Observation -> ... -> Final Answer.
// Supports both traditional ReAct mode and Planning-Execution separation mode.
type ReactAgent struct {
	llmMu              sync.RWMutex
	llm                LLMProvider
	memory             MemoryStore
	tools              []Tool
	toolMap            map[string]Tool
	cfg                config.AgentConfig
	loader             *PromptLoader
	skillsContent      string
	hooks              []Hook
	contextManager     ContextManager     // AI 协作框架的上下文管理器
	skillPaths         map[string]string  // 技能路径映射，用于智能提点
	// Planning-Execution 分离架构相关字段
	planningEnabled    bool               // 是否启用规划模式
	executionMode      string             // 执行模式: "react", "planning", "auto"
	extractor          *Extractor         // 能力提取器
	planner            *TaskPlanner       // 任务规划器
	skillsMgr          any                // skills.Manager (避免循环依赖)
	mcpMgr             any                // mcp.Manager (避免循环依赖)
	workingDir         string             // 工作目录
	skillsActiveDir    string             // 技能目录
	capabilityCacheTTL int                // 能力缓存 TTL (小时)
}

// NewReact creates a ReAct agent with the given dependencies.
func NewReact(llm LLMProvider, memory MemoryStore, tools []Tool, cfg config.AgentConfig) *ReactAgent {
	return NewReactWithPrompt(llm, memory, tools, cfg, nil, "")
}

// NewReactWithPrompt creates a ReAct agent with optional PromptLoader and skills content.
func NewReactWithPrompt(llm LLMProvider, memory MemoryStore, tools []Tool, cfg config.AgentConfig, loader *PromptLoader, skillsContent string) *ReactAgent {
	toolMap := make(map[string]Tool)
	strategy := NamesakeStrategy(cfg.Running.NamesakeStrategy)
	if strategy == "" {
		strategy = NamesakeSkip // Default: skip (CoPaw default)
	}

	// Register tools with namesake strategy
	for _, t := range tools {
		if err := registerTool(toolMap, t, strategy); err != nil {
			logger.L().Warn("tool registration failed",
				zap.String("tool", t.Name()),
				zap.Error(err),
			)
		}
	}

	// Build final tools list from toolMap (to handle duplicates)
	finalTools := make([]Tool, 0, len(toolMap))
	for _, t := range toolMap {
		finalTools = append(finalTools, t)
	}

	// 初始化 ContextManager
	ctxMgr := NewMemoryContextManager()

	agent := &ReactAgent{
		llm:            llm,
		memory:         memory,
		tools:          finalTools,
		toolMap:        toolMap,
		cfg:            cfg,
		loader:         loader,
		skillsContent:  skillsContent,
		hooks:          nil,
		contextManager: ctxMgr,
		skillPaths:     make(map[string]string),
		// Planning-Execution 默认值
		planningEnabled: false,
		executionMode:   "react", // 默认使用传统 ReAct 模式
		capabilityCacheTTL: 24,   // 默认 24 小时缓存
	}

	return agent
}

// SetSkillPaths 设置技能路径映射，用于智能提点。
func (a *ReactAgent) SetSkillPaths(paths map[string]string) {
	a.skillPaths = paths
	if mcm, ok := a.contextManager.(*memoryContextManager); ok {
		mcm.SetSkillPaths(paths)
	}
}

// GetContextManager 返回 ContextManager。
func (a *ReactAgent) GetContextManager() ContextManager {
	return a.contextManager
}

// registerTool registers a tool according to the namesake strategy.
func registerTool(toolMap map[string]Tool, tool Tool, strategy NamesakeStrategy) error {
	name := tool.Name()

	// No duplicate, simply add
	if _, exists := toolMap[name]; !exists {
		toolMap[name] = tool
		return nil
	}

	// Duplicate found, apply strategy
	switch strategy {
	case NamesakeOverride:
		toolMap[name] = tool
		return nil

	case NamesakeSkip:
		// Keep existing tool, ignore new one
		return nil

	case NamesakeRaise:
		return fmt.Errorf("duplicate tool name: %s", name)

	case NamesakeRename:
		// Auto-rename: try tool_2, tool_3, ...
		for i := 2; i < 100; i++ {
			newName := fmt.Sprintf("%s_%d", name, i)
			if _, exists := toolMap[newName]; !exists {
				// Note: We can't modify the tool's name directly, so we skip
				// In a real implementation, we'd need a wrapper or mutable name
				logger.L().Debug("auto-renamed tool",
					zap.String("original", name),
					zap.String("renamed", newName),
				)
				return fmt.Errorf("rename strategy not fully implemented for tool %s", name)
			}
		}
		return fmt.Errorf("failed to find unique name for tool: %s", name)

	default:
		return fmt.Errorf("unknown namesake strategy: %s", strategy)
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
		zap.String("executionMode", a.executionMode),
	)
	if reporter != nil {
		reporter.OnThinking()
	}

	// 检查是否使用规划模式
	if a.shouldUsePlanningMode(message) {
		log.Debug("Using planning mode for this request")
		result, err := a.runWithPlanning(ctx, chatID, message)
		if err != nil {
			log.Warn("Planning mode failed, falling back to ReAct", zap.Error(err))
			// 规划失败时降级到 ReAct 模式
			return a.runWithReAct(ctx, chatID, message)
		}
		return result, nil
	}

	// 使用传统 ReAct 模式
	return a.runWithReAct(ctx, chatID, message)
}

// runWithReAct 使用传统的 ReAct 模式运行。
func (a *ReactAgent) runWithReAct(ctx context.Context, chatID string, message string) (string, error) {
	log := logger.L()
	reporter := getProgressReporter(ctx)

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
	maxTurns := a.cfg.Running.MaxTurns
	if maxTurns <= 0 {
		maxTurns = 20
	}

	var finalContent string
	// 待批量保存的工具消息
	var pendingToolMessages []Message

	for turn := 0; turn < maxTurns; turn++ {
		// 检查上下文是否已取消，但返回友好响应而非错误
		if err := ctx.Err(); err != nil {
			log.Warn("Context cancelled during ReAct loop", zap.Error(err))
			// 如果已有工具结果，让 LLM 基于已有结果生成响应
			if turn > 0 {
				// 尝试生成最终响应
				if finalContent == "" {
					finalContent = "操作因超时中断，请稍后重试或使用其他方法。"
				}
				break
			}
			// 如果在第一轮就超时，返回提示
			return "请求处理超时，请稍后重试。如果问题持续，请尝试简化您的请求。", nil
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

		// 最终安全检查：确保消息总长度不超过智谱 API 限制
		// 如果超过 20K 字符，删除最旧的工具消息，直到符合要求
		const hardMaxLength = 20000
		messages = enforceMaxLength(messages, hardMaxLength)

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
			// 检查是否是上下文超时
			if ctx.Err() != nil {
				log.Warn("LLM chat cancelled due to context timeout", zap.Error(err))
				// 如果已有工具结果，返回部分响应
				if turn > 0 {
					return "操作因超时中断。基于已执行的工具结果，请稍后重试或尝试其他方法。", nil
				}
				return "请求处理超时，请稍后重试。如果问题持续，请尝试简化您的请求。", nil
			}
			return "", fmt.Errorf("llm chat: %w", err)
		}

		log.Debug("LLM response",
			zap.Int("contentLen", len(resp.Content)),
			zap.Int("toolCalls", len(resp.ToolCalls)),
			zap.Int("promptTokens", resp.Usage.PromptTokens),
			zap.Int("completionTokens", resp.Usage.CompletionTokens),
			zap.String("contentPreview", truncate(resp.Content, 200)),
		)

		// 解析结构化响应（AI 协作框架）
		structured, parseErr := ParseStructuredResponse(resp.Content)
		if parseErr != nil {
			log.Debug("Failed to parse structured response", zap.Error(parseErr))
		}
		if structured != nil {
			log.Debug("Structured response parsed",
				zap.String("thought", truncate(structured.Thought, 100)),
				zap.Int("capabilitiesNeeded", len(structured.CapabilitiesNeeded)),
				zap.Int("storageRequests", len(structured.StorageRequests)),
				zap.Int("retrievalRequests", len(structured.RetrievalRequests)))

			// 处理存储请求
			if len(structured.StorageRequests) > 0 {
				if err := a.contextManager.Store(ctx, chatID, structured.StorageRequests); err != nil {
					log.Warn("Failed to store content", zap.Error(err))
				}
			}

			// 处理能力需求 - 生成智能提点
			if len(structured.CapabilitiesNeeded) > 0 {
				reminder := a.contextManager.BuildCapabilityReminder(ctx, chatID, structured.CapabilitiesNeeded)
				if reminder != "" {
					log.Debug("Capability reminder generated", zap.String("reminder", truncate(reminder, 200)))
					// 将提醒注入到下一条消息中
					// 我们通过在 messages 列表中添加一个临时的 system 消息来实现
					// 这条消息不会被保存到 memory
					messages = append(messages, Message{
						Role:    "system",
						Content: reminder,
					})
				}
			}

			// 检查是否是最终响应
			if structured.FinalAnswer != "" && len(resp.ToolCalls) == 0 {
				finalContent = structured.FinalAnswer
				log.Info("Final answer from structured response", zap.String("content", truncate(finalContent, 200)))
				break
			}
		}

		// 处理检索请求 - 在下一轮开始前注入存储的内容
		if structured != nil && len(structured.RetrievalRequests) > 0 {
			retrieved, err := a.contextManager.Retrieve(ctx, chatID, structured.RetrievalRequests)
			if err == nil && len(retrieved) > 0 {
				var retrievedContent []string
				for _, item := range retrieved {
					retrievedContent = append(retrievedContent,
						fmt.Sprintf("**%s** (%s)\n%s", item.Name, item.Description, item.Content))
				}
				retrievalMsg := Message{
					Role:    "system",
					Content: "--- 📦 已检索存储内容 ---\n" + strings.Join(retrievedContent, "\n\n") + "\n--- 存储内容结束 ---",
				}
				messages = append(messages, retrievalMsg)
				log.Debug("Retrieved content injected", zap.Int("count", len(retrieved)))
			}
		}

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
			// 如果没有结构化响应的 FinalAnswer，使用原始内容
			if finalContent == "" {
				finalContent = strings.TrimSpace(resp.Content)
			}
			log.Info("Final answer", zap.String("content", truncate(finalContent, 200)))
			break
		}

		// Append assistant message, then execute tools in parallel and append results
		messages = append(messages, assistantMsg)
		toolResults := a.executeToolsParallel(ctx, chatID, resp.ToolCalls, reporter)

		// Append tool results in order (matching tool_call order)
		const maxToolResultLen = 5000 // 5K 字符硬限制（智谱 API 限制更严格）
		for i, tr := range toolResults {
			result := tr.Result
			// 工具结果硬截断：防止过大的响应（如整个网页 HTML）导致 token 超限
			if len(result) > maxToolResultLen {
				result = result[:maxToolResultLen] + "\n...[内容过长，已截断]"
			}

			log.Debug("Tool result",
				zap.String("tool", resp.ToolCalls[i].Name),
				zap.String("result", truncate(result, 500)),
			)
			if reporter != nil {
				reporter.OnToolResult(resp.ToolCalls[i].Name, result)
			}

			toolMsg := Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: resp.ToolCalls[i].ID,
			}
			messages = append(messages, toolMsg)
			// 收集待保存的工具消息，不立即保存
			pendingToolMessages = append(pendingToolMessages, toolMsg)
		}

		// 批量保存工具消息（在下一轮 LLM 调用前）
		if len(pendingToolMessages) > 0 {
			log.Debug("Batch saving tool messages", zap.Int("count", len(pendingToolMessages)))
			// 使用更短的超时上下文来保存，避免阻塞太久
			saveCtx, saveCancel := context.WithTimeout(context.Background(), 30*time.Second)
			for _, msg := range pendingToolMessages {
				if err := a.memory.Save(saveCtx, chatID, msg); err != nil {
					log.Warn("failed to save tool message", zap.Error(err))
					// 继续保存其他消息，不中断流程
				}
			}
			saveCancel()
			pendingToolMessages = nil
		}

		// 智能压缩：当对话历史过长时，进行总结提炼
		// 在第 1 轮后检查，智谱 API 限制较严格
		const compressThreshold = 15000 // 当 messages 总长度超过 15K 字符时触发压缩
		if turn >= 1 && totalMessageLength(messages) > compressThreshold {
			compressed, err := a.compressMessages(ctx, messages)
			if err == nil && len(compressed) > 0 {
				oldLen := len(messages)
				messages = compressed
				log.Debug("Compressed conversation history",
					zap.Int("originalCount", oldLen),
					zap.Int("compressedCount", len(compressed)),
					zap.Int("reduced", oldLen-len(compressed)),
				)
			} else if err != nil {
				log.Warn("Compression failed, continuing with original messages", zap.Error(err))
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

// enforceMaxLength 确保消息总长度不超过限制，优先删除最旧的工具消息
func enforceMaxLength(msgs []Message, maxLen int) []Message {
	for totalMessageLength(msgs) > maxLen && len(msgs) > 3 {
		// 找到最旧的工具消息并删除
		for i, msg := range msgs {
			if msg.Role == "tool" {
				// 删除这条工具消息和对应的 assistant 消息
				newMsgs := make([]Message, 0, len(msgs)-1)
				newMsgs = append(newMsgs, msgs[:i]...)
				if i+1 < len(msgs) {
					newMsgs = append(newMsgs, msgs[i+1:]...)
				} else {
					newMsgs = msgs[:i]
				}
				msgs = newMsgs
				break
			}
		}
		// 如果没有工具消息可删，但仍然超长，删除最旧的 assistant 消息
		if totalMessageLength(msgs) > maxLen {
			for i, msg := range msgs {
				if msg.Role == "assistant" && len(msg.ToolCalls) == 0 {
					msgs = append(msgs[:i], msgs[i+1:]...)
					break
				}
			}
		}
	}
	return msgs
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
	history, err := a.memory.Load(ctx, chatID, a.cfg.Running.MaxTurns*4) // rough limit
	if err != nil {
		return nil, fmt.Errorf("load history: %w", err)
	}

	// Sanitize tool messages to ensure proper pairing
	history = SanitizeToolMessages(history)

	// Context window check: compact when estimated tokens exceed 80% of maxInputLength
	if a.cfg.Running.MaxInputLength > 0 {
		sysPrompt := a.getSystemPrompt()
		estimated := estimateTokens(sysPrompt) + estimateMessagesTokens(history)
		threshold := int(float64(a.cfg.Running.MaxInputLength) * 0.8)
		if estimated > threshold {
			logger.L().Info("Context near limit, compacting memory",
				zap.Int("estimated", estimated),
				zap.Int("threshold", threshold),
			)
			if err := a.memory.Compact(ctx, chatID); err != nil {
				logger.L().Warn("Compact failed", zap.Error(err))
			} else {
				history, err = a.memory.Load(ctx, chatID, a.cfg.Running.MaxTurns*4)
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

// totalMessageLength 计算消息列表的总字符长度。
func totalMessageLength(messages []Message) int {
	total := 0
	for _, m := range messages {
		total += len(m.Content)
	}
	return total
}

// compressMessages 智能压缩对话历史。
// 保留最近 3 轮完整对话，将更早的历史压缩成简洁摘要。
func (a *ReactAgent) compressMessages(ctx context.Context, messages []Message) ([]Message, error) {
	const keepRecent = 6 // 保留最近 3 轮对话（user + assistant/tool 成对）
	if len(messages) <= keepRecent+1 { // +1 是 system prompt
		return messages, nil
	}

	// 分离需要压缩的部分和保留的最近对话
	// 跳过 system prompt (index 0)，压缩 1 到 len-keepRecent 的历史
	toCompress := messages[1 : len(messages)-keepRecent]
	recent := messages[len(messages)-keepRecent:]
	systemPrompt := messages[0]

	// 确保 recent 中包含至少一条 user 消息
	// 如果 recent 中没有 user 消息，需要从 toCompress 中找回最近的 user 消息
	// 这避免了压缩后的消息序列不符合 OpenAI API 格式要求（必须以 user 消息开头或包含 user 消息）
	hasUserInRecent := false
	for _, m := range recent {
		if m.Role == "user" {
			hasUserInRecent = true
			break
		}
	}

	if !hasUserInRecent && len(toCompress) > 0 {
		// 从后向前查找最近的 user 消息
		for i := len(toCompress) - 1; i >= 0; i-- {
			if toCompress[i].Role == "user" {
				// 将这条 user 消息及其之后的所有消息加入到 recent
				recent = append(toCompress[i:], recent...)
				// 更新 toCompress，只保留这条 user 消息之前的内容
				toCompress = toCompress[:i]
				break
			}
		}
	}

	// 如果需要压缩的内容不多，直接返回
	if len(toCompress) < 4 {
		return messages, nil
	}

	// 构建 LLM 压缩请求
	compressPrompt := `请将以下对话历史压缩成简洁的结构化摘要。

要求：
1. 保留用户的核心需求和意图
2. 保留关键的工具调用结果（如价格、数据等具体信息）
3. 删除中间过程和 AI 已知的通用知识
4. 输出格式为：用户意图：xxx | 关键信息：xxx
5. 摘要长度控制在 200 字以内

对话历史：
` + formatMessagesForCompression(toCompress)

	// 创建压缩专用的 LLM 请求（不使用工具）
	a.llmMu.RLock()
	provider := a.llm
	a.llmMu.RUnlock()

	compressReq := &ChatRequest{
		Messages: []Message{
			{Role: "system", Content: "你是对话历史压缩助手。"},
			{Role: "user", Content: compressPrompt},
		},
		MaxTokens: 500,
	}

	resp, err := provider.Chat(ctx, compressReq)
	if err != nil {
		return nil, fmt.Errorf("compression LLM call: %w", err)
	}

	// 检查压缩结果是否有效
	summaryContent := strings.TrimSpace(resp.Content)
	if summaryContent == "" {
		// 如果 LLM 返回空内容，跳过压缩摘要，使用默认行为
		// 这避免了发送只有前缀的空消息导致 API 返回 400 错误
		compressed := make([]Message, 0, len(recent)+1)
		compressed = append(compressed, systemPrompt)
		compressed = append(compressed, recent...)
		return compressed, nil
	}

	// 构建压缩后的消息列表
	compressed := make([]Message, 0, len(recent)+2)
	compressed = append(compressed, systemPrompt)
	// 将压缩摘要作为一条 system 消息插入
	compressed = append(compressed, Message{
		Role:    "system",
		Content: "【对话历史摘要】" + summaryContent,
	})
	compressed = append(compressed, recent...)

	return compressed, nil
}

// formatMessagesForCompression 将消息列表格式化为压缩用的文本。
func formatMessagesForCompression(messages []Message) string {
	var sb strings.Builder
	for i, m := range messages {
		switch m.Role {
		case "user":
			sb.WriteString(fmt.Sprintf("用户%d: %s\n", i, truncate(m.Content, 200)))
		case "assistant":
			if len(m.ToolCalls) > 0 {
				sb.WriteString(fmt.Sprintf("助手%d: [调用工具: %d个]\n", i, len(m.ToolCalls)))
			} else if m.Content != "" {
				sb.WriteString(fmt.Sprintf("助手%d: %s\n", i, truncate(m.Content, 200)))
			}
		case "tool":
			// 只记录工具结果的前 100 字符，避免摘要本身过长
			sb.WriteString(fmt.Sprintf("工具结果: %s\n", truncate(m.Content, 100)))
		}
	}
	return sb.String()
}

// ============================================================================
// Planning-Execution Separation Mode
// ============================================================================

// shouldUsePlanningMode 判断是否应该使用规划模式。
func (a *ReactAgent) shouldUsePlanningMode(message string) bool {
	switch a.executionMode {
	case "planning":
		return a.planningEnabled
	case "react":
		return false
	case "auto":
		// 根据消息复杂度自动判断
		return a.planningEnabled && a.isComplexTask(message)
	default:
		return false
	}
}

// isComplexTask 判断任务是否足够复杂，需要使用规划模式。
func (a *ReactAgent) isComplexTask(message string) bool {
	// 检测复杂任务的信号词
	complexSignals := []string{
		"生成", "报告", "文档", "分析", "对比", "搜索", "比较",
		"并", "然后", "最后", "步骤", "首先", "接着",
		"总结", "汇总", "提取", "收集", "整理",
	}

	messageLower := strings.ToLower(message)
	for _, signal := range complexSignals {
		if strings.Contains(messageLower, signal) {
			return true
		}
	}

	// 消息长度阈值（中文按字符计算，100 字符以上认为可能复杂）
	if utf8.RuneCountInString(message) > 100 {
		return true
	}

	// 包含多个句子或标点符号也可能表示复杂任务
	sentenceCount := 0
	for _, r := range message {
		if r == '。' || r == '！' || r == '？' || r == '.' || r == '!' || r == '?' {
			sentenceCount++
		}
	}
	if sentenceCount >= 2 {
		return true
	}

	return false
}

// runWithPlanning 使用规划-执行分离模式运行。
func (a *ReactAgent) runWithPlanning(ctx context.Context, chatID string, message string) (string, error) {
	log := logger.L()

	// 1. 获取/更新能力清单
	if a.extractor == nil {
		a.llmMu.RLock()
		llm := a.llm
		a.llmMu.RUnlock()

		a.extractor = NewExtractor(
			llm,
			a.tools,
			nil, // skillsMgr
			nil, // mcpMgr
			a.cfg,
			a.workingDir,
			a.skillsActiveDir,
			a.capabilityCacheTTL,
		)
	}

	registry, err := a.extractor.ExtractCapabilities(ctx)
	if err != nil {
		return "", fmt.Errorf("extract capabilities: %w", err)
	}

	// 2. 初始化规划器
	if a.planner == nil {
		a.llmMu.RLock()
		llm := a.llm
		a.llmMu.RUnlock()

		cfg := DefaultPlanningConfig()
		a.planner = NewTaskPlanner(llm, cfg)
	}

	// 3. 获取对话上下文摘要
	context, err := a.getConversationSummary(ctx, chatID)
	if err != nil {
		log.Warn("Failed to get conversation summary", zap.Error(err))
		context = ""
	}

	// 4. 规划阶段
	plan, err := a.planner.Plan(ctx, &PlanningRequest{
		UserMessage:       message,
		CapabilitySummary: registry.Summary,
		Context:           context,
		ConversationID:    chatID,
	})
	if err != nil {
		return "", fmt.Errorf("planning failed: %w", err)
	}

	log.Info("Execution plan generated",
		zap.Int("steps", len(plan.Tasks)),
		zap.String("summary", plan.Summary),
	)

	// 5. 执行阶段
	a.llmMu.RLock()
	llm := a.llm
	a.llmMu.RUnlock()

	executor := NewExecutor(a, llm, a.tools, DefaultExecutionConfig())

	// 设置技能和 MCP 管理器（如果有）
	if a.skillsMgr != nil {
		executor.SetSkillsManager(a.skillsMgr)
	}
	if a.mcpMgr != nil {
		executor.SetMCPManager(a.mcpMgr)
	}

	result, err := executor.Execute(ctx, plan)
	if err != nil {
		return "", fmt.Errorf("execution failed: %w", err)
	}

	// 6. 保存对话到记忆
	userMsg := Message{Role: "user", Content: message}
	if err := a.memory.Save(ctx, chatID, userMsg); err != nil {
		log.Warn("Failed to save user message", zap.Error(err))
	}

	assistantMsg := Message{Role: "assistant", Content: result}
	if err := a.memory.Save(ctx, chatID, assistantMsg); err != nil {
		log.Warn("Failed to save assistant message", zap.Error(err))
	}

	return result, nil
}

// getConversationSummary 获取对话的摘要，用于规划上下文。
func (a *ReactAgent) getConversationSummary(ctx context.Context, chatID string) (string, error) {
	// 获取最近的对话历史
	history, err := a.memory.Load(ctx, chatID, 10) // 最近 10 条消息
	if err != nil {
		return "", err
	}

	if len(history) == 0 {
		return "", nil
	}

	// 简单摘要：只保留最近几条用户消息的内容
	var summaryParts []string
	userCount := 0
	for i := len(history) - 1; i >= 0 && userCount < 3; i-- {
		if history[i].Role == "user" {
			summaryParts = append([]string{history[i].Content}, summaryParts...)
			userCount++
		}
	}

	if len(summaryParts) == 0 {
		return "", nil
	}

	return "最近的用户请求：\n" + strings.Join(summaryParts, "\n"), nil
}

// EnablePlanningMode 启用规划模式。
func (a *ReactAgent) EnablePlanningMode(enabled bool) {
	a.planningEnabled = enabled
	logger.L().Info("Planning mode state changed", zap.Bool("enabled", enabled))
}

// SetExecutionMode 设置执行模式。
func (a *ReactAgent) SetExecutionMode(mode string) {
	switch mode {
	case "react", "planning", "auto":
		a.executionMode = mode
		logger.L().Info("Execution mode changed", zap.String("mode", mode))
	default:
		logger.L().Warn("Invalid execution mode, using 'react'", zap.String("mode", mode))
		a.executionMode = "react"
	}
}

// SetSkillsManager 设置技能管理器（用于规划和执行）。
func (a *ReactAgent) SetSkillsManager(mgr any) {
	a.skillsMgr = mgr
}

// SetMCPManager 设置 MCP 管理器（用于规划和执行）。
func (a *ReactAgent) SetMCPManager(mgr any) {
	a.mcpMgr = mgr
}

// SetWorkingDirectories 设置工作目录和技能目录。
func (a *ReactAgent) SetWorkingDirectories(workingDir, skillsActiveDir string) {
	a.workingDir = workingDir
	a.skillsActiveDir = skillsActiveDir
}

// SetCapabilityCacheTTL 设置能力缓存 TTL。
func (a *ReactAgent) SetCapabilityCacheTTL(ttlHours int) {
	a.capabilityCacheTTL = ttlHours
}

// GetExecutionMode 返回当前执行模式。
func (a *ReactAgent) GetExecutionMode() string {
	return a.executionMode
}

// RefreshCapabilities 刷新能力缓存。
func (a *ReactAgent) RefreshCapabilities(ctx context.Context) error {
	if a.extractor == nil {
		return fmt.Errorf("extractor not initialized")
	}
	return a.extractor.Refresh(ctx)
}

// Ensure ReactAgent implements Agent.
var _ Agent = (*ReactAgent)(nil)
