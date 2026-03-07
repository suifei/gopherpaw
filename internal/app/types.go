// Package app 提供统一的应用生命周期管理，参考 CoPaw app/_app.py
package app

import (
	"context"
	"sync"
	"time"

	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/channels"
	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/internal/mcp"
	"github.com/suifei/gopherpaw/internal/scheduler"
)

// App 是 GopherPaw 应用的核心管理器
// 负责：
// 1. 按正确顺序启动所有服务
// 2. 管理服务生命周期（启动、停止、热重载）
// 3. 提供依赖注入容器
// 4. 暴露 HTTP/WebSocket API
type App struct {
	// 核心组件
	Runner         *agent.ReactAgent
	ChannelManager *channels.Manager
	CronScheduler  *scheduler.CronScheduler
	MCPManager     *mcp.MCPManager
	ChatManager    *ChatManager
	Config         *config.Config

	// 配置和监控
	ConfigWatcher *ConfigWatcher
	MCPWatcher    *MCPConfigWatcher

	// HTTP 服务器
	Server *HTTPServer

	// 内部状态
	ctx          context.Context
	cancel       context.CancelFunc
	restartCh    chan struct{} // Single-flight 重启控制
	restarting   bool
	restartingMu sync.Mutex
}

// Manager 定义应用生命周期管理接口
type Manager interface {
	// Start 按依赖顺序启动所有服务
	Start(ctx context.Context) error

	// Stop 优雅停止所有服务（逆序）
	Stop(ctx context.Context) error

	// RestartServices 热重载所有服务（不停止 HTTP 服务器）
	// 参考 CoPaw _restart_services()
	RestartServices(ctx context.Context, opts RestartOptions) error

	// HealthCheck 健康检查
	HealthCheck() map[string]bool
}

// ChatManager 聊天管理器接口（待实现）
type ChatManager interface {
	// GetChat 获取聊天会话
	GetChat(chatID string) (*ChatSession, error)

	// ListChats 列出所有聊天会话
	ListChats() ([]*ChatSession, error)

	// CreateChat 创建聊天会话
	CreateChat(chatID string) (*ChatSession, error)

	// DeleteChat 删除聊天会话
	DeleteChat(chatID string) error
}

// ChatSession 聊天会话（占位符，待定义）
type ChatSession struct {
	ID        string
	CreatedAt time.Time
	UpdatedAt time.Time
	Metadata  map[string]interface{}
}

// LifecycleHook 定义生命周期钩子
type LifecycleHook func(ctx context.Context, app *App) error

// RestartOptions 热重载选项
type RestartOptions struct {
	ReloadChannels bool // 重载渠道配置
	ReloadCrons    bool // 重载定时任务
	ReloadMCP      bool // 重载 MCP 客户端
	ExcludeCaller  bool // 排除调用者任务（防止死锁）
}

// ServiceStatus 服务状态
type ServiceStatus struct {
	Name    string
	Running bool
	Error   error
}

// AppStatus 应用状态
type AppStatus struct {
	Services []ServiceStatus
	Uptime   time.Duration
	Version  string
}

// ConfigWatcher 配置文件监听器（占位符，待实现）
type ConfigWatcher struct {
	// TODO: 实现 fsnotify 配置监听
}

// MCPConfigWatcher MCP 配置监听器（占位符，待实现）
type MCPConfigWatcher struct {
	// TODO: 实现 MCP 配置热重载
}

// HTTPServer HTTP/WebSocket 服务器（占位符，待实现）
type HTTPServer struct {
	// TODO: 实现基于 net/http 的服务器
}
