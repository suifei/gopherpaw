// Package agent provides the core Agent runtime, ReAct loop, and domain types.
package agent

import (
	"context"
)

// PlaceholderAgent is a minimal Agent implementation for bootstrap.
// It returns a fixed response without calling LLM. Used until ReAct loop is implemented.
type PlaceholderAgent struct{}

// NewPlaceholder creates a placeholder agent for dependency wiring.
func NewPlaceholder() *PlaceholderAgent {
	return &PlaceholderAgent{}
}

// Run processes a message and returns a placeholder response.
func (a *PlaceholderAgent) Run(ctx context.Context, chatID string, message string) (string, error) {
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	default:
		return "[Placeholder] Agent not yet implemented. Message received: " + message, nil
	}
}

// RunStream processes a message and streams a placeholder response.
func (a *PlaceholderAgent) RunStream(ctx context.Context, chatID string, message string) (<-chan string, error) {
	ch := make(chan string, 1)
	go func() {
		defer close(ch)
		select {
		case <-ctx.Done():
			return
		default:
			ch <- "[Placeholder] Agent not yet implemented. Message received: " + message
		}
	}()
	return ch, nil
}
