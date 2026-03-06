// Package channels provides HTTP webhook server for DingTalk/Feishu/QQ callbacks.
package channels

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// WebhookHandler processes incoming webhook POST body.
type WebhookHandler interface {
	HandleWebhook(ctx context.Context, body []byte) error
}

// WebhookServer serves HTTP endpoints for channel webhooks.
type WebhookServer struct {
	host     string
	port     int
	handlers map[string]WebhookHandler
	server   *http.Server
	mu       sync.RWMutex
}

// NewWebhookServer creates a webhook server.
func NewWebhookServer(host string, port int) *WebhookServer {
	return &WebhookServer{
		host:     host,
		port:     port,
		handlers: make(map[string]WebhookHandler),
	}
}

// Register adds a webhook handler for path (e.g. "dingtalk", "feishu").
func (s *WebhookServer) Register(path string, h WebhookHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[path] = h
}

// Start starts the HTTP server. Blocks until Stop is called.
func (s *WebhookServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/webhook/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		path := strings.TrimPrefix(r.URL.Path, "/webhook/")
		path = strings.TrimSuffix(path, "/")
		parts := strings.SplitN(path, "/", 2)
		channel := parts[0]
		s.mu.RLock()
		h := s.handlers[channel]
		s.mu.RUnlock()
		if h == nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		runCtx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		if err := h.HandleWebhook(runCtx, body); err != nil {
			logger.L().Warn("webhook handler error", zap.String("channel", channel), zap.Error(err))
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	s.server = &http.Server{
		Addr:    fmtAddr(s.host, s.port),
		Handler: mux,
	}
	go func() {
		<-ctx.Done()
		_ = s.server.Shutdown(context.Background())
	}()
	logger.L().Info("webhook server listening", zap.String("addr", s.server.Addr))
	return s.server.ListenAndServe()
}

func fmtAddr(host string, port int) string {
	if host == "" {
		host = "0.0.0.0"
	}
	if port <= 0 {
		port = 8080
	}
	return fmt.Sprintf("%s:%d", host, port)
}

// Stop shuts down the server.
func (s *WebhookServer) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}
	return s.server.Shutdown(ctx)
}
