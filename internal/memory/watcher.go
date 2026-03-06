package memory

import (
	"context"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// Watcher monitors memory files and triggers async index updates.
type Watcher struct {
	watcher  *fsnotify.Watcher
	onChange func(chatID string, path string)
	mu       sync.RWMutex
	chatDirs map[string]string // absPath -> chatID
	stop     chan struct{}
}

// NewWatcher creates a file watcher for memory directories.
func NewWatcher(onChange func(chatID string, path string)) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Watcher{
		watcher:  w,
		onChange: onChange,
		chatDirs: make(map[string]string),
		stop:     make(chan struct{}),
	}, nil
}

// AddChatDir watches the memory directory for a chat.
func (w *Watcher) AddChatDir(chatID string, dir string) error {
	if chatID == "" || dir == "" {
		return nil
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if _, ok := w.chatDirs[abs]; ok {
		return nil
	}
	if err := w.watcher.Add(abs); err != nil {
		return err
	}
	w.chatDirs[abs] = chatID
	return nil
}

// RemoveChatDir stops watching a chat's memory directory.
func (w *Watcher) RemoveChatDir(chatID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for path, cid := range w.chatDirs {
		if cid == chatID {
			_ = w.watcher.Remove(path)
			delete(w.chatDirs, path)
			break
		}
	}
}

// Run starts the watcher loop. Blocks until ctx is cancelled.
func (w *Watcher) Run(ctx context.Context) {
	log := logger.L()
	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stop:
			return
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}
			if filepath.Ext(event.Name) != ".md" {
				continue
			}
			w.mu.RLock()
			chatID := ""
			eventPath := filepath.Clean(event.Name)
			for dir, cid := range w.chatDirs {
				if eventPath == dir || strings.HasPrefix(eventPath, dir+string(filepath.Separator)) {
					chatID = cid
					break
				}
			}
			w.mu.RUnlock()
			if chatID != "" && w.onChange != nil {
				log.Debug("memory file changed", zap.String("chatID", chatID), zap.String("path", event.Name))
				go w.onChange(chatID, event.Name)
			}
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Warn("watcher error", zap.Error(err))
		}
	}
}

// Close releases watcher resources.
func (w *Watcher) Close() error {
	close(w.stop)
	return w.watcher.Close()
}
