// Package mcp provides MCP (Model Context Protocol) client for connecting to external tool servers.
package mcp

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// ConfigWatcher watches MCP config files for changes and triggers reloads.
type ConfigWatcher struct {
	manager     *MCPManager
	watcher     *fsnotify.Watcher
	paths       []string
	mu          sync.Mutex
	running     bool
	stopCh      chan struct{}
	doneCh      chan struct{}
	debounceMs  int
	lastConfigs map[string]config.MCPServerConfig
}

// NewConfigWatcher creates a new MCP config watcher.
// debounceMs controls how long to wait after a change before triggering a reload (default: 500ms).
func NewConfigWatcher(manager *MCPManager, debounceMs int) *ConfigWatcher {
	if debounceMs <= 0 {
		debounceMs = 500
	}
	return &ConfigWatcher{
		manager:    manager,
		debounceMs: debounceMs,
		stopCh:     make(chan struct{}),
		doneCh:     make(chan struct{}),
	}
}

// AddPath adds a config file path to watch. Can be called before or after Start.
func (w *ConfigWatcher) AddPath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Check for duplicates
	for _, p := range w.paths {
		if p == absPath {
			return nil
		}
	}
	w.paths = append(w.paths, absPath)

	// If already running, add to fsnotify watcher
	if w.watcher != nil {
		// Watch the directory, not the file itself (handles file replacements)
		dir := filepath.Dir(absPath)
		if err := w.watcher.Add(dir); err != nil {
			logger.L().Warn("failed to watch MCP config directory", zap.String("dir", dir), zap.Error(err))
		}
	}
	return nil
}

// RemovePath removes a config file path from the watch list.
func (w *ConfigWatcher) RemovePath(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	for i, p := range w.paths {
		if p == absPath {
			w.paths = append(w.paths[:i], w.paths[i+1:]...)
			break
		}
	}
	return nil
}

// Start begins watching config files for changes.
func (w *ConfigWatcher) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = true
	w.mu.Unlock()

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w.watcher = watcher

	// Add directories for all watched paths
	dirs := make(map[string]bool)
	w.mu.Lock()
	for _, p := range w.paths {
		dir := filepath.Dir(p)
		if !dirs[dir] {
			dirs[dir] = true
			if err := watcher.Add(dir); err != nil {
				logger.L().Warn("failed to watch MCP config directory", zap.String("dir", dir), zap.Error(err))
			}
		}
	}
	w.mu.Unlock()

	// Load initial configs
	w.loadAndStoreConfigs()

	go w.watchLoop(ctx)
	logger.L().Info("MCP config watcher started", zap.Int("paths", len(w.paths)))
	return nil
}

// Stop stops watching config files.
func (w *ConfigWatcher) Stop() error {
	w.mu.Lock()
	if !w.running {
		w.mu.Unlock()
		return nil
	}
	w.running = false
	w.mu.Unlock()

	close(w.stopCh)
	<-w.doneCh

	if w.watcher != nil {
		w.watcher.Close()
		w.watcher = nil
	}
	return nil
}

func (w *ConfigWatcher) watchLoop(ctx context.Context) {
	defer close(w.doneCh)

	var debounceTimer *time.Timer
	var pendingReload bool

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			// Check if the event is for one of our watched files
			if !w.isWatchedFile(event.Name) {
				continue
			}
			// Only react to write/create events
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}

			logger.L().Debug("MCP config file changed", zap.String("file", event.Name), zap.String("op", event.Op.String()))

			// Debounce: wait a bit before reloading (handles rapid successive writes)
			pendingReload = true
			if debounceTimer == nil {
				debounceTimer = time.AfterFunc(time.Duration(w.debounceMs)*time.Millisecond, func() {
					w.mu.Lock()
					if pendingReload {
						pendingReload = false
						w.mu.Unlock()
						w.triggerReload(ctx)
					} else {
						w.mu.Unlock()
					}
				})
			} else {
				debounceTimer.Reset(time.Duration(w.debounceMs) * time.Millisecond)
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			logger.L().Warn("MCP config watcher error", zap.Error(err))
		}
	}
}

func (w *ConfigWatcher) isWatchedFile(path string) bool {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	for _, p := range w.paths {
		if p == absPath {
			return true
		}
	}
	return false
}

func (w *ConfigWatcher) loadAndStoreConfigs() map[string]config.MCPServerConfig {
	allConfigs := make(map[string]config.MCPServerConfig)

	w.mu.Lock()
	paths := make([]string, len(w.paths))
	copy(paths, w.paths)
	w.mu.Unlock()

	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			logger.L().Debug("failed to read MCP config file", zap.String("path", p), zap.Error(err))
			continue
		}
		configs, err := ParseMCPConfig(data)
		if err != nil {
			logger.L().Warn("failed to parse MCP config file", zap.String("path", p), zap.Error(err))
			continue
		}
		for k, v := range configs {
			allConfigs[k] = v
		}
	}

	w.mu.Lock()
	w.lastConfigs = allConfigs
	w.mu.Unlock()

	return allConfigs
}

func (w *ConfigWatcher) triggerReload(ctx context.Context) {
	newConfigs := w.loadAndStoreConfigs()
	if w.manager != nil {
		if err := w.manager.Reload(ctx, newConfigs); err != nil {
			logger.L().Warn("MCP config reload failed", zap.Error(err))
		} else {
			logger.L().Info("MCP config reloaded", zap.Int("servers", len(newConfigs)))
		}
	}
}

// GetConfigs returns the current loaded configs (for testing/debugging).
func (w *ConfigWatcher) GetConfigs() map[string]config.MCPServerConfig {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.lastConfigs == nil {
		return nil
	}
	out := make(map[string]config.MCPServerConfig, len(w.lastConfigs))
	for k, v := range w.lastConfigs {
		out[k] = v
	}
	return out
}
