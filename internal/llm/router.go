package llm

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

const defaultSlotName = "default"

var visionExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".webp": true, ".bmp": true, ".svg": true,
}

// ModelRouter implements agent.LLMProvider with multi-model routing.
// When only one model is configured (no Models map), it behaves identically
// to a plain provider. When multiple slots are configured, it can auto-select
// a vision-capable model when image content is detected.
type ModelRouter struct {
	providers map[string]agent.LLMProvider
	slots     map[string]config.ModelSlot
	active    string
	baseCfg   config.LLMConfig
	mu        sync.RWMutex
}

// NewModelRouter creates a ModelRouter from LLMConfig.
// If cfg.Models is empty, a single "default" slot is created from the top-level fields.
func NewModelRouter(cfg config.LLMConfig) (*ModelRouter, error) {
	r := &ModelRouter{
		providers: make(map[string]agent.LLMProvider),
		slots:     make(map[string]config.ModelSlot),
		baseCfg:   cfg,
		active:    defaultSlotName,
	}

	if len(cfg.Models) == 0 {
		p, err := Create(cfg)
		if err != nil {
			return nil, fmt.Errorf("create default provider: %w", err)
		}
		r.providers[defaultSlotName] = p
		r.slots[defaultSlotName] = config.ModelSlot{
			Model:        cfg.Model,
			Capabilities: []string{"text", "tools"},
		}
		return r, nil
	}

	for name, slot := range cfg.Models {
		resolved := cfg.ResolveSlot(name)
		p, err := Create(resolved)
		if err != nil {
			return nil, fmt.Errorf("create provider for slot %q: %w", name, err)
		}
		r.providers[name] = p
		r.slots[name] = slot
	}

	if _, ok := r.providers[defaultSlotName]; !ok {
		first := ""
		for name := range r.providers {
			first = name
			break
		}
		r.active = first
	}

	return r, nil
}

// Name returns "model-router" or the underlying provider name for single-slot mode.
func (r *ModelRouter) Name() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.providers) == 1 {
		for _, p := range r.providers {
			return p.Name()
		}
	}
	return "model-router"
}

// Chat routes the request to the appropriate provider based on content analysis.
func (r *ModelRouter) Chat(ctx context.Context, req *agent.ChatRequest) (*agent.ChatResponse, error) {
	provider, slotName := r.selectProvider(req)
	log := logger.L()
	log.Debug("ModelRouter routing",
		zap.String("slot", slotName),
		zap.String("model", r.slotModel(slotName)),
	)
	return provider.Chat(ctx, req)
}

// ChatStream routes the streaming request to the appropriate provider.
func (r *ModelRouter) ChatStream(ctx context.Context, req *agent.ChatRequest) (agent.ChatStream, error) {
	provider, _ := r.selectProvider(req)
	return provider.ChatStream(ctx, req)
}

// Switch changes the active model slot. Returns error if slot not found.
func (r *ModelRouter) Switch(slotName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.providers[slotName]; !ok {
		return fmt.Errorf("unknown model slot %q (available: %s)", slotName, r.slotNamesLocked())
	}
	r.active = slotName
	logger.L().Info("ModelRouter switched", zap.String("slot", slotName), zap.String("model", r.slotModelLocked(slotName)))
	return nil
}

// ActiveSlot returns the name of the currently active slot.
func (r *ModelRouter) ActiveSlot() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.active
}

// SlotNames returns all configured slot names.
func (r *ModelRouter) SlotNames() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return names
}

// HasCapability checks if any configured slot has the given capability.
func (r *ModelRouter) HasCapability(cap string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, slot := range r.slots {
		if slot.HasCapability(cap) {
			return true
		}
	}
	return false
}

// selectProvider picks the best provider for the request.
// If vision content is detected and the active slot lacks vision capability,
// it temporarily selects a vision-capable slot for this request only.
func (r *ModelRouter) selectProvider(req *agent.ChatRequest) (agent.LLMProvider, string) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	activeSlot := r.slots[r.active]
	if needsVision(req) && !activeSlot.HasCapability("vision") {
		if name, ok := r.findSlotWithCapLocked("vision"); ok {
			logger.L().Info("ModelRouter auto-routing to vision model",
				zap.String("from", r.active),
				zap.String("to", name),
			)
			return r.providers[name], name
		}
	}

	return r.providers[r.active], r.active
}

func (r *ModelRouter) findSlotWithCapLocked(cap string) (string, bool) {
	for name, slot := range r.slots {
		if slot.HasCapability(cap) {
			return name, true
		}
	}
	return "", false
}

func (r *ModelRouter) slotModel(name string) string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.slotModelLocked(name)
}

func (r *ModelRouter) slotModelLocked(name string) string {
	if slot, ok := r.slots[name]; ok && slot.Model != "" {
		return slot.Model
	}
	return r.baseCfg.Model
}

func (r *ModelRouter) slotNamesLocked() string {
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	return strings.Join(names, ", ")
}

// needsVision scans the last few messages for image references.
func needsVision(req *agent.ChatRequest) bool {
	scanCount := 6
	msgs := req.Messages
	if len(msgs) > scanCount {
		msgs = msgs[len(msgs)-scanCount:]
	}
	for _, m := range msgs {
		if m.Role == "system" {
			continue
		}
		if containsImageRef(m.Content) {
			return true
		}
	}
	return false
}

func containsImageRef(content string) bool {
	lower := strings.ToLower(content)

	if strings.Contains(lower, "base64,") && (strings.Contains(lower, "image/") || strings.Contains(lower, "data:")) {
		return true
	}

	for _, word := range strings.Fields(lower) {
		ext := filepath.Ext(word)
		if visionExts[ext] {
			return true
		}
	}

	return false
}
