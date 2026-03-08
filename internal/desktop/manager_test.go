package desktop_test

import (
	"context"
	"sync"
	"testing"

	"github.com/suifei/gopherpaw/internal/desktop"
)

// mockVNCServer 模拟 VNC 服务器
type mockVNCServer struct {
	startFunc func(ctx context.Context) error
	stopFunc  func(ctx context.Context) error
	running   bool
	mu        sync.RWMutex
}

func (m *mockVNCServer) Start(ctx context.Context) error {
	if m.startFunc != nil {
		return m.startFunc(ctx)
	}
	m.mu.Lock()
	m.running = true
	m.mu.Unlock()
	return nil
}

func (m *mockVNCServer) Stop(ctx context.Context) error {
	if m.stopFunc != nil {
		return m.stopFunc(ctx)
	}
	m.mu.Lock()
	m.running = false
	m.mu.Unlock()
	return nil
}

func (m *mockVNCServer) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// mockNoVNCProxy 模拟 noVNC 代理
type mockNoVNCProxy struct {
	startFunc func(ctx context.Context) error
	stopFunc  func(ctx context.Context) error
	running   bool
	mu        sync.RWMutex
}

func (m *mockNoVNCProxy) Start(ctx context.Context) error {
	if m.startFunc != nil {
		return m.startFunc(ctx)
	}
	m.mu.Lock()
	m.running = true
	m.mu.Unlock()
	return nil
}

func (m *mockNoVNCProxy) Stop(ctx context.Context) error {
	if m.stopFunc != nil {
		return m.stopFunc(ctx)
	}
	m.mu.Lock()
	m.running = false
	m.mu.Unlock()
	return nil
}

func (m *mockNoVNCProxy) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// 测试配置验证
func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *desktop.Config
		wantErr bool
	}{
		{
			name: "empty display",
			config: &desktop.Config{
				Display:   "",
				Password:  "test",
				Geometry:  "1920x1080",
				Depth:     24,
				VNCPort:   5901,
				NoVNCPort: 6080,
			},
			wantErr: false,
		},
		{
			name: "invalid display format",
			config: &desktop.Config{
				Display:   "invalid",
				Password:  "test",
				Geometry:  "1920x1080",
				Depth:     24,
				VNCPort:   5901,
				NoVNCPort: 6080,
			},
			wantErr: true,
		},
		{
			name: "empty password",
			config: &desktop.Config{
				Display:   ":1",
				Password:  "",
				Geometry:  "1920x1080",
				Depth:     24,
				VNCPort:   5901,
				NoVNCPort: 6080,
			},
			wantErr: true,
		},
		{
			name: "empty geometry",
			config: &desktop.Config{
				Display:   ":1",
				Password:  "test",
				Geometry:  "",
				Depth:     24,
				VNCPort:   5901,
				NoVNCPort: 6080,
			},
			wantErr: false,
		},
		{
			name: "invalid password",
			config: &desktop.Config{
				Display:   ":1",
				Password:  "",
				Geometry:  "1920x1080",
				Depth:     24,
				VNCPort:   5901,
				NoVNCPort: 6080,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := desktop.NewManager(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if !tt.wantErr {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}
