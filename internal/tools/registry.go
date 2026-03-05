// Package tools provides built-in tools for the agent.
package tools

import (
	"log/slog"
	"sync"

	"github.com/suifei/gopherpaw/internal/agent"
)

var (
	mu       sync.RWMutex
	builtins []agent.Tool
)

func init() {
	RegisterBuiltin()
}

// RegisterBuiltin registers all built-in tools and returns them.
func RegisterBuiltin() []agent.Tool {
	mu.Lock()
	defer mu.Unlock()
	if len(builtins) > 0 {
		return builtins
	}
	builtins = []agent.Tool{
		&TimeTool{},
		&ShellTool{},
		&ReadFileTool{},
		&WriteFileTool{},
		&EditFileTool{},
		&AppendFileTool{},
		&GrepSearchTool{},
		&GlobSearchTool{},
		&MemorySearchTool{},
	}
	if ws, err := NewWebSearchTool(); err == nil {
		builtins = append(builtins, ws)
	} else {
		slog.Warn("web_search tool disabled", "error", err)
	}
	builtins = append(builtins, NewHTTPTool())
	return builtins
}

// GetBuiltins returns a copy of the built-in tools slice.
func GetBuiltins() []agent.Tool {
	mu.RLock()
	defer mu.RUnlock()
	out := make([]agent.Tool, len(builtins))
	copy(out, builtins)
	return out
}
