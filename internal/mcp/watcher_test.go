package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/suifei/gopherpaw/internal/config"
)

func TestConfigWatcher_AddRemovePath(t *testing.T) {
	w := NewConfigWatcher(nil, 100)

	// Add a path
	tmpFile := filepath.Join(t.TempDir(), "mcp.json")
	if err := w.AddPath(tmpFile); err != nil {
		t.Fatalf("AddPath: %v", err)
	}

	// Adding same path again should be a no-op
	if err := w.AddPath(tmpFile); err != nil {
		t.Fatalf("AddPath duplicate: %v", err)
	}
	if len(w.paths) != 1 {
		t.Errorf("expected 1 path, got %d", len(w.paths))
	}

	// Remove path
	if err := w.RemovePath(tmpFile); err != nil {
		t.Fatalf("RemovePath: %v", err)
	}
	if len(w.paths) != 0 {
		t.Errorf("expected 0 paths, got %d", len(w.paths))
	}
}

func TestConfigWatcher_LoadConfigs(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "mcp.json")
	configContent := `{
		"mcpServers": {
			"test-server": {
				"transport": "stdio",
				"command": "echo",
				"args": ["hello"]
			}
		}
	}`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	w := NewConfigWatcher(nil, 100)
	if err := w.AddPath(configFile); err != nil {
		t.Fatalf("AddPath: %v", err)
	}

	configs := w.loadAndStoreConfigs()
	if len(configs) != 1 {
		t.Errorf("expected 1 config, got %d", len(configs))
	}
	if cfg, ok := configs["test-server"]; !ok {
		t.Error("test-server not found in configs")
	} else {
		if cfg.Command != "echo" {
			t.Errorf("expected command 'echo', got %q", cfg.Command)
		}
	}
}

func TestConfigWatcher_StartStop(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "mcp.json")
	configContent := `{"mcpServers": {}}`
	if err := os.WriteFile(configFile, []byte(configContent), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	w := NewConfigWatcher(nil, 100)
	if err := w.AddPath(configFile); err != nil {
		t.Fatalf("AddPath: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start watcher
	if err := w.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Starting again should be no-op
	if err := w.Start(ctx); err != nil {
		t.Fatalf("Start again: %v", err)
	}

	// Give it time to initialize
	time.Sleep(50 * time.Millisecond)

	// Stop watcher
	if err := w.Stop(); err != nil {
		t.Fatalf("Stop: %v", err)
	}

	// Stopping again should be no-op
	if err := w.Stop(); err != nil {
		t.Fatalf("Stop again: %v", err)
	}
}

func TestConfigWatcher_FileChange(t *testing.T) {
	dir := t.TempDir()
	configFile := filepath.Join(dir, "mcp.json")
	initialContent := `{"mcpServers": {"initial": {"command": "echo"}}}`
	if err := os.WriteFile(configFile, []byte(initialContent), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	w := NewConfigWatcher(nil, 50)
	if err := w.AddPath(configFile); err != nil {
		t.Fatalf("AddPath: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := w.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer w.Stop()

	// Initial load should have happened
	time.Sleep(100 * time.Millisecond)
	configs := w.GetConfigs()
	if len(configs) != 1 {
		t.Errorf("expected 1 initial config, got %d", len(configs))
	}

	// Modify the config file
	updatedContent := `{"mcpServers": {"updated": {"command": "test"}}}`
	if err := os.WriteFile(configFile, []byte(updatedContent), 0644); err != nil {
		t.Fatalf("WriteFile updated: %v", err)
	}

	// Wait for debounce + reload
	time.Sleep(200 * time.Millisecond)

	// Check that configs were updated
	configs = w.GetConfigs()
	if _, ok := configs["updated"]; !ok {
		t.Error("expected 'updated' server in configs after file change")
	}
}

func TestConfigWatcher_MultipleFiles(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "mcp1.json")
	file2 := filepath.Join(dir, "mcp2.json")

	content1 := `{"mcpServers": {"server1": {"command": "cmd1"}}}`
	content2 := `{"mcpServers": {"server2": {"command": "cmd2"}}}`

	if err := os.WriteFile(file1, []byte(content1), 0644); err != nil {
		t.Fatalf("WriteFile 1: %v", err)
	}
	if err := os.WriteFile(file2, []byte(content2), 0644); err != nil {
		t.Fatalf("WriteFile 2: %v", err)
	}

	w := NewConfigWatcher(nil, 100)
	w.AddPath(file1)
	w.AddPath(file2)

	configs := w.loadAndStoreConfigs()
	if len(configs) != 2 {
		t.Errorf("expected 2 configs from 2 files, got %d", len(configs))
	}
	if _, ok := configs["server1"]; !ok {
		t.Error("server1 not found")
	}
	if _, ok := configs["server2"]; !ok {
		t.Error("server2 not found")
	}
}

func TestConfigWatcher_IsWatchedFile(t *testing.T) {
	dir := t.TempDir()
	watchedFile := filepath.Join(dir, "watched.json")
	unwatchedFile := filepath.Join(dir, "unwatched.json")

	w := NewConfigWatcher(nil, 100)
	w.AddPath(watchedFile)

	if !w.isWatchedFile(watchedFile) {
		t.Error("expected watchedFile to be watched")
	}
	if w.isWatchedFile(unwatchedFile) {
		t.Error("expected unwatchedFile to not be watched")
	}
}

func TestConfigWatcher_GetConfigs_Empty(t *testing.T) {
	w := NewConfigWatcher(nil, 100)
	configs := w.GetConfigs()
	if configs != nil {
		t.Errorf("expected nil configs, got %v", configs)
	}
}

// Ensure config import is used
var _ config.MCPServerConfig
