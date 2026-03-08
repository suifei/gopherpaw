// Package agent provides lifecycle hooks for the ReAct agent loop.
package agent

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// SkillContentProvider is a function that returns skill content for a given query.
type SkillContentProvider func(query string) string

// Hook is a function invoked before each ReAct reasoning step.
// It may modify the messages slice in-place or return an error to abort.
type Hook func(ctx context.Context, agent *ReactAgent, chatID string, messages []Message) ([]Message, error)

// MemoryCompactionHook triggers automatic memory compaction when the
// estimated token count exceeds the configured threshold.
func MemoryCompactionHook(threshold int, keepRecent int) Hook {
	if threshold <= 0 {
		threshold = 100000
	}
	if keepRecent <= 0 {
		keepRecent = 10
	}

	return func(ctx context.Context, agent *ReactAgent, chatID string, messages []Message) ([]Message, error) {
		log := logger.L()

		var systemMsgs, remaining []Message
		for _, m := range messages {
			if m.Role == "system" {
				systemMsgs = append(systemMsgs, m)
			} else {
				remaining = append(remaining, m)
			}
		}

		if len(remaining) <= keepRecent {
			return messages, nil
		}

		totalTokens := 0
		for _, m := range messages {
			totalTokens += EstimateMessageTokens(m)
		}

		if totalTokens <= threshold {
			return messages, nil
		}

		log.Info("Memory compaction triggered",
			zap.Int("estimatedTokens", totalTokens),
			zap.Int("threshold", threshold),
			zap.Int("messageCount", len(remaining)),
		)

		if err := agent.memory.Compact(ctx, chatID); err != nil {
			log.Warn("Memory compaction failed", zap.Error(err))
			return messages, nil
		}

		history, err := agent.memory.Load(ctx, chatID, agent.cfg.Running.MaxTurns*4)
		if err != nil {
			return messages, nil
		}

		result := make([]Message, 0, len(systemMsgs)+len(history))
		result = append(result, systemMsgs...)
		result = append(result, history...)
		return result, nil
	}
}

// BootstrapHook checks for BOOTSTRAP.md on the first user interaction
// and runs the bootstrap flow via BootstrapRunner.
// This is the unified bootstrap path - it delegates to BootstrapRunner internally.
func BootstrapHook(workingDir string, language string) Hook {
	if language == "" {
		language = "zh"
	}

	return func(ctx context.Context, agent *ReactAgent, chatID string, messages []Message) ([]Message, error) {
		// Only run on first user interaction
		if !isFirstUserInteraction(messages) {
			return messages, nil
		}

		// Check if bootstrap already completed
		completedPath := filepath.Join(workingDir, ".bootstrap_completed")
		if _, err := os.Stat(completedPath); err == nil {
			return messages, nil
		}

		// Check if BOOTSTRAP.md exists
		bootstrapPath := filepath.Join(workingDir, fileBOOTSTRAP)
		if _, err := os.Stat(bootstrapPath); os.IsNotExist(err) {
			return messages, nil
		}

		// Inject bootstrap guidance into the first user message
		guidance := BuildBootstrapGuidance(language)
		if guidance == "" {
			return messages, nil
		}

		for i := range messages {
			if messages[i].Role == "user" {
				messages[i].Content = guidance + "\n\n" + messages[i].Content
				break
			}
		}

		// Mark as completed after injection
		if err := os.WriteFile(completedPath, []byte("done"), 0644); err != nil {
			logger.L().Warn("write .bootstrap_completed", zap.Error(err))
		}

		return messages, nil
	}
}

// EstimateMessageTokens returns a rough token estimate for a single message.
func EstimateMessageTokens(m Message) int {
	n := len(m.Content) / 4
	for _, tc := range m.ToolCalls {
		n += len(tc.Arguments) / 4
	}
	return n + 4 // per-message overhead
}

// isFirstUserInteraction returns true if there is exactly one user message.
func isFirstUserInteraction(messages []Message) bool {
	count := 0
	for _, m := range messages {
		if m.Role == "user" {
			count++
		}
	}
	return count == 1
}

// BuildBootstrapGuidance returns the bootstrap guidance text for the given language.
func BuildBootstrapGuidance(language string) string {
	switch language {
	case "zh":
		return `# 🌟 引导模式已激活

**重要：你正处于首次设置模式。**

你的工作目录中存在 ` + "`BOOTSTRAP.md`" + ` 文件。这意味着你应该引导用户完成引导流程，以建立你的身份和偏好。

**你的任务：**
1. 阅读 BOOTSTRAP.md 文件，友好地表示初次见面，引导用户完成引导流程。
2. 按照BOOTSTRAP.md 里面的指示执行。例如，帮助用户定义你的身份、他们的偏好，并建立工作关系
3. 按照指南中的描述创建和更新必要的文件（PROFILE.md、MEMORY.md 等）
4. 完成引导流程后，按照指示删除 BOOTSTRAP.md

**如果用户希望跳过：**
如果用户明确表示想跳过引导，那就继续回答下面的原始问题。你随时可以帮助他们完成引导。

**用户的原始消息：`
	default:
		return `# 🌟 BOOTSTRAP MODE ACTIVATED

**IMPORTANT: You are in first-time setup mode.**

A ` + "`BOOTSTRAP.md`" + ` file exists in your working directory. This means you should guide the user through the bootstrap process to establish your identity and preferences.

**Your task:**
1. Read the BOOTSTRAP.md file, greet the user warmly as a first meeting, and guide them through the bootstrap process.
2. Follow the instructions in BOOTSTRAP.md. For example, help the user define your identity, their preferences, and establish the working relationship.
3. Create and update the necessary files (PROFILE.md, MEMORY.md, etc.) as described in the guide.
4. After completing the bootstrap process, delete BOOTSTRAP.md as instructed.

**If the user wants to skip:**
If the user explicitly says they want to skip the bootstrap or just want their question answered directly, then proceed to answer their original question below. You can always help them bootstrap later.

**Original user message:`
	}
}

// DynamicSkillHook injects relevant skill content based on the last user message.
// It replaces the static system prompt with one containing dynamically selected skills.
func DynamicSkillHook(provider SkillContentProvider) Hook {
	return func(ctx context.Context, agent *ReactAgent, chatID string, messages []Message) ([]Message, error) {
		// Find the last user message
		var lastUserMsg string
		for i := len(messages) - 1; i >= 0; i-- {
			if messages[i].Role == "user" {
				lastUserMsg = messages[i].Content
				break
			}
		}

		if lastUserMsg == "" {
			return messages, nil
		}

		// Get relevant skill content
		skillContent := provider(lastUserMsg)
		if skillContent == "" {
			return messages, nil
		}

		// Find and update the system message
		for i := range messages {
			if messages[i].Role == "system" {
				// Check if we need to add skills section
				baseContent := messages[i].Content
				if !strings.Contains(baseContent, "# Active Skills") && skillContent != "" {
					messages[i].Content = baseContent + "\n\n# Active Skills\n\n" + skillContent
					logger.L().Debug("DynamicSkillHook: added skills to system prompt",
						zap.Int("skillContentLen", len(skillContent)),
						zap.Int("queryLen", len(lastUserMsg)))
				}
				break
			}
		}

		return messages, nil
	}
}
