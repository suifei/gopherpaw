// Package agent provides lifecycle hooks for the ReAct agent loop.
package agent

import (
	"context"
	"os"
	"path/filepath"

	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

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

		history, err := agent.memory.Load(ctx, chatID, agent.cfg.MaxTurns*4)
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
		return `这是你的第一次启动。请阅读工作目录下的 BOOTSTRAP.md 文件，按照其中的指示完成自我介绍和初始化设置。完成后将结果写入 PROFILE.md。`
	default:
		return `This is your first boot. Please read the BOOTSTRAP.md file in your working directory and follow its instructions to complete self-introduction and initial setup. Write the result to PROFILE.md.`
	}
}
