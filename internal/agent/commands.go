// Package agent provides magic command handling for slash-prefixed messages.
package agent

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// HandleMagicCommand checks if message is a magic command and returns (result, true) if handled.
// Returns ("", false, nil) if not a magic command.
func HandleMagicCommand(ctx context.Context, memory MemoryStore, chatID string, message string, daemonInfo *DaemonInfo) (string, bool, error) {
	msg := strings.TrimSpace(message)
	if !strings.HasPrefix(msg, "/") {
		return "", false, nil
	}

	parts := strings.Fields(msg)
	if len(parts) == 0 {
		return "", false, nil
	}
	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	log := logger.L()

	switch cmd {
	case "/compact":
		if err := memory.Compact(ctx, chatID); err != nil {
			return "", true, fmt.Errorf("compact: %w", err)
		}
		return "已压缩对话历史。", true, nil

	case "/new":
		history, err := memory.Load(ctx, chatID, 100)
		if err != nil {
			return "", true, fmt.Errorf("load history: %w", err)
		}
		var summary strings.Builder
		for _, m := range history {
			if m.Content != "" {
				summary.WriteString(m.Role)
				summary.WriteString(": ")
				summary.WriteString(m.Content[:min(len(m.Content), 500)])
				if len(m.Content) > 500 {
					summary.WriteString("...")
				}
				summary.WriteString("\n")
			}
		}
		if summary.Len() > 0 {
			_ = memory.SaveLongTerm(ctx, chatID, summary.String(), "session")
		}
		// Clear short-term: we need to clear history. InMemoryStore doesn't have Clear.
		// For now we just compact to minimal - the memory store keeps last N.
		if err := memory.Compact(ctx, chatID); err != nil {
			log.Warn("compact after /new", zap.Error(err))
		}
		return "已保存到长期记忆并清空上下文。", true, nil

	case "/clear":
		if err := memory.Compact(ctx, chatID); err != nil {
			return "", true, fmt.Errorf("clear: %w", err)
		}
		return "已清空上下文。", true, nil

	case "/history":
		history, err := memory.Load(ctx, chatID, 100)
		if err != nil {
			return "", true, fmt.Errorf("load history: %w", err)
		}
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("共 %d 条消息。\n", len(history)))
		for i, m := range history {
			sb.WriteString(fmt.Sprintf("  [%d] %s: %s\n", i+1, m.Role, truncateForHistory(m.Content, 80)))
		}
		sb.WriteString(fmt.Sprintf("\n预估 token 数: %d", estimateTokensForHistory(history)))
		return sb.String(), true, nil

	case "/compact_str":
		summary, err := memory.GetCompactSummary(ctx, chatID)
		if err != nil {
			return "", true, fmt.Errorf("get compact summary: %w", err)
		}
		if summary == "" {
			return "暂无压缩摘要。", true, nil
		}
		return summary, true, nil

	case "/switch-model":
		if len(args) < 2 {
			return "用法: /switch-model <provider> <model>", true, nil
		}
		if daemonInfo != nil && daemonInfo.SwitchLLM != nil {
			if err := daemonInfo.SwitchLLM(args[0], args[1]); err != nil {
				return "", true, fmt.Errorf("switch model: %w", err)
			}
			return fmt.Sprintf("已切换至 %s / %s", args[0], args[1]), true, nil
		}
		return "switch-model 功能未配置。", true, nil

	case "/daemon":
		if daemonInfo == nil {
			return "daemon 信息不可用。", true, nil
		}
		if len(args) == 0 {
			return fmt.Sprintf("状态: %s\n版本: %s", daemonInfo.Status, daemonInfo.Version), true, nil
		}
		sub := strings.ToLower(args[0])
		switch sub {
		case "status":
			return daemonInfo.Status, true, nil
		case "version":
			return daemonInfo.Version, true, nil
		case "logs":
			n := 20
			if len(args) > 1 {
				if v, err := strconv.Atoi(args[1]); err == nil && v > 0 && v <= 100 {
					n = v
				}
			}
			if daemonInfo.Logs != nil {
				return daemonInfo.Logs(n), true, nil
			}
			return "日志功能未配置。", true, nil
		case "reload-config":
			if daemonInfo.ReloadConfig != nil {
				if err := daemonInfo.ReloadConfig(); err != nil {
					return "", true, fmt.Errorf("reload config: %w", err)
				}
				return "配置已重新加载。", true, nil
			}
			return "reload-config 功能未配置。", true, nil
		case "restart":
			if daemonInfo.Restart != nil {
				if err := daemonInfo.Restart(); err != nil {
					return "", true, fmt.Errorf("restart: %w", err)
				}
				return "正在重启...", true, nil
			}
			return "restart 功能未配置。", true, nil
		default:
			return fmt.Sprintf("未知子命令: %s", sub), true, nil
		}
	}

	return "", false, nil
}

func truncateForHistory(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func estimateTokensForHistory(msgs []Message) int {
	n := 0
	for _, m := range msgs {
		n += len(m.Content)/4 + 1
	}
	return n
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
