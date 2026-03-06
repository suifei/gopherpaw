package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// FileStore handles file-based persistence for conversation history and long-term memory.
type FileStore struct {
	mu         sync.RWMutex
	workingDir string
	cfg        config.MemoryConfig
}

// NewFileStore creates a FileStore for the given working directory.
func NewFileStore(cfg config.MemoryConfig) *FileStore {
	wd := cfg.WorkingDir
	if wd == "" {
		wd = "."
	}
	abs, err := filepath.Abs(wd)
	if err != nil {
		abs = wd
	}
	return &FileStore{
		workingDir: abs,
		cfg:        cfg,
	}
}

// historyFile returns the path to the JSON history file for a chat.
func (f *FileStore) historyFile(chatID string) string {
	return filepath.Join(f.workingDir, "data", "chats", chatID+".json")
}

// memoryDir returns the memory directory for a chat (memory/ within chat dir).
func (f *FileStore) memoryDir(chatID string) string {
	return filepath.Join(f.workingDir, "data", "chats", chatID, "memory")
}

// memoryFile returns MEMORY.md path for a chat.
func (f *FileStore) memoryFile(chatID string) string {
	return filepath.Join(f.workingDir, "data", "chats", chatID, "MEMORY.md")
}

// dailyLogFile returns memory/YYYY-MM-DD.md path.
func (f *FileStore) dailyLogFile(chatID string, t time.Time) string {
	return filepath.Join(f.memoryDir(chatID), t.Format("2006-01-02")+".md")
}

type storedHistory struct {
	Messages []storedMessage `json:"messages"`
}

// LoadHistory loads conversation history from file.
func (f *FileStore) LoadHistory(ctx context.Context, chatID string) ([]storedMessage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	path := f.historyFile(chatID)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read history: %w", err)
	}
	var h storedHistory
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, fmt.Errorf("unmarshal history: %w", err)
	}
	return h.Messages, nil
}

// SaveHistory persists conversation history to file.
func (f *FileStore) SaveHistory(ctx context.Context, chatID string, msgs []storedMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	path := f.historyFile(chatID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	h := storedHistory{Messages: msgs}
	data, err := json.MarshalIndent(h, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal history: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write history: %w", err)
	}
	return nil
}

// SaveLongTerm appends content to MEMORY.md (category "memory") or memory/YYYY-MM-DD.md (category "log").
func (f *FileStore) SaveLongTerm(ctx context.Context, chatID string, content string, category string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if chatID == "" {
		return fmt.Errorf("chatID cannot be empty")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	if category == "log" {
		path := f.dailyLogFile(chatID, time.Now())
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return fmt.Errorf("mkdir: %w", err)
		}
		fh, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return fmt.Errorf("open log: %w", err)
		}
		defer fh.Close()
		_, err = fh.WriteString(content + "\n\n")
		return err
	}
	path := f.memoryFile(chatID)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}
	fh, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open MEMORY: %w", err)
	}
	defer fh.Close()
	_, err = fh.WriteString(content + "\n\n")
	return err
}

// LoadLongTerm reads MEMORY.md and all memory/*.md files for a chat.
func (f *FileStore) LoadLongTerm(ctx context.Context, chatID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if chatID == "" {
		return "", fmt.Errorf("chatID cannot be empty")
	}
	f.mu.RLock()
	defer f.mu.RUnlock()
	var out string
	memPath := f.memoryFile(chatID)
	if data, err := os.ReadFile(memPath); err == nil {
		out = string(data)
	}
	dir := f.memoryDir(chatID)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return "", fmt.Errorf("read memory dir: %w", err)
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			logger.L().Warn("read memory file", zap.String("path", path), zap.Error(err))
			continue
		}
		if out != "" {
			out += "\n\n---\n\n"
		}
		out += fmt.Sprintf("## %s\n\n%s", e.Name(), string(data))
	}
	return out, nil
}

// WorkingDir returns the working directory path.
func (f *FileStore) WorkingDir() string {
	return f.workingDir
}

// MemoryDir returns the memory directory for a chat (for watcher).
func (f *FileStore) MemoryDir(chatID string) string {
	return f.memoryDir(chatID)
}
