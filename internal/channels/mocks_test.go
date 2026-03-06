package channels

import (
	"context"
)

// mockAgent 是一个简单的 agent.Agent 实现，用于测试
type mockAgent struct {
	runFunc       func(ctx context.Context, chatID, text string) (string, error)
	runStreamFunc func(ctx context.Context, chatID, text string) (<-chan string, error)
}

func (m *mockAgent) Run(ctx context.Context, chatID, text string) (string, error) {
	if m.runFunc != nil {
		return m.runFunc(ctx, chatID, text)
	}
	return "mock response", nil
}

func (m *mockAgent) RunStream(ctx context.Context, chatID, text string) (<-chan string, error) {
	if m.runStreamFunc != nil {
		return m.runStreamFunc(ctx, chatID, text)
	}
	ch := make(chan string, 1)
	ch <- "mock stream"
	close(ch)
	return ch, nil
}

// newMockAgent 创建一个新的 mock agent
func newMockAgent() *mockAgent {
	return &mockAgent{}
}

// newMockAgentWithRunFunc 创建一个带有自定义 Run 函数的 mock agent
func newMockAgentWithRunFunc(runFunc func(ctx context.Context, chatID, text string) (string, error)) *mockAgent {
	return &mockAgent{
		runFunc: runFunc,
	}
}
