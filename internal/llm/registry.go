package llm

import (
	"fmt"
	"sync"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
)

// Factory creates an LLMProvider from config.
type Factory func(cfg config.LLMConfig) (agent.LLMProvider, error)

var (
	mu        sync.RWMutex
	registry  = make(map[string]Factory)
)

func init() {
	Register("openai", func(cfg config.LLMConfig) (agent.LLMProvider, error) {
		return NewOpenAI(cfg), nil
	})
	Register("ollama", func(cfg config.LLMConfig) (agent.LLMProvider, error) {
		return NewOllama(cfg)
	})
}

// Register adds a provider factory by name.
func Register(name string, f Factory) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = f
}

// Create creates an LLMProvider by provider name from config.
func Create(cfg config.LLMConfig) (agent.LLMProvider, error) {
	mu.RLock()
	f, ok := registry[cfg.Provider]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown LLM provider %q", cfg.Provider)
	}
	return f(cfg)
}

// SwitchProvider creates a new LLMProvider with the given provider name and model.
// Use with agent.SetLLMProvider for runtime switch.
func SwitchProvider(providerName, model string, baseCfg config.LLMConfig) (agent.LLMProvider, error) {
	cfg := baseCfg
	cfg.Provider = providerName
	cfg.Model = model
	return Create(cfg)
}
