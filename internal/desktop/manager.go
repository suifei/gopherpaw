// Package desktop 提供远程桌面环境管理功能
package desktop

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"
)

// 常见错误定义
var (
	ErrVNCAlreadyRunning = errors.New("VNC server already running")
	ErrVNCPortInUse      = errors.New("VNC port already in use")
	ErrInvalidPassword   = errors.New("invalid VNC password")
	ErrDisplayNotFound   = errors.New("X11 display not found")
	ErrNoVNCFailed       = errors.New("noVNC proxy failed to start")
	ErrNotRunning        = errors.New("desktop not running")
)

// Manager 管理远程桌面环境
type Manager struct {
	config     *Config
	vncServer  *VNCServer
	novncProxy *NoVNCProxy
	mu         sync.RWMutex
	running    bool
}

// Config 定义桌面环境配置
type Config struct {
	Display   string // X11 显示号（如 ":1"）
	Password  string // VNC 密码
	Geometry  string // 屏幕分辨率（如 "1920x1080"）
	Depth     int    // 色深（默认 24）
	VNCPort   int    // VNC 服务器端口（默认 5901）
	NoVNCPort int    // noVNC 代理端口（默认 6080）
}

// VNCServer 管理 TigerVNC 服务器进程
type VNCServer struct {
	display    string
	password   string
	geometry   string
	depth      int
	port       int
	passwordDB string // VNC 密码文件路径
	process    *os.Process
	cmd        *exec.Cmd
	mu         sync.RWMutex
}

// NoVNCProxy 管理 noVNC WebSocket 代理
type NoVNCProxy struct {
	vncPort    int
	listenPort int
	process    *os.Process
	cmd        *exec.Cmd
	mu         sync.RWMutex
}

// NewManager 创建桌面管理器实例
func NewManager(cfg *Config) (*Manager, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	return &Manager{
		config: cfg,
	}, nil
}

// Start 启动桌面环境（VNC + noVNC）
func (m *Manager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return ErrVNCAlreadyRunning
	}

	// 1. 启动 VNC 服务器
	vncServer, err := NewVNCServer(&VNCConfig{
		Display:  m.config.Display,
		Password: m.config.Password,
		Geometry: m.config.Geometry,
		Depth:    m.config.Depth,
		Port:     m.config.VNCPort,
	})
	if err != nil {
		return fmt.Errorf("create VNC server: %w", err)
	}

	if err := vncServer.Start(ctx); err != nil {
		return fmt.Errorf("start VNC server: %w", err)
	}

	m.vncServer = vncServer

	// 2. 启动 noVNC 代理
	novncProxy, err := NewNoVNCProxy(&NoVNCConfig{
		VNCPort:    m.config.VNCPort,
		ListenPort: m.config.NoVNCPort,
	})
	if err != nil {
		vncServer.Stop(ctx)
		return fmt.Errorf("create noVNC proxy: %w", err)
	}

	if err := novncProxy.Start(ctx); err != nil {
		vncServer.Stop(ctx)
		return fmt.Errorf("start noVNC proxy: %w", err)
	}

	m.novncProxy = novncProxy
	m.running = true

	zap.L().Info("Desktop started",
		zap.String("display", m.config.Display),
		zap.Int("vnc_port", m.config.VNCPort),
		zap.Int("novnc_port", m.config.NoVNCPort),
		zap.String("url", m.GetAccessURL()))

	return nil
}

// Stop 停止桌面环境
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	var errs []error

	// 1. 停止 noVNC 代理
	if m.novncProxy != nil {
		if err := m.novncProxy.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("stop noVNC proxy: %w", err))
		}
	}

	// 2. 停止 VNC 服务器
	if m.vncServer != nil {
		if err := m.vncServer.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("stop VNC server: %w", err))
		}
	}

	m.running = false

	if len(errs) > 0 {
		return fmt.Errorf("errors during stop: %v", errs)
	}

	zap.L().Info("Desktop stopped")
	return nil
}

// IsRunning 返回桌面环境是否正在运行
func (m *Manager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.running
}

// GetAccessURL 返回 Web 访问 URL
func (m *Manager) GetAccessURL() string {
	hostname := "localhost"
	if h := os.Getenv("HOSTNAME"); h != "" {
		hostname = h
	}
	return fmt.Sprintf("http://%s:%d/vnc.html", hostname, m.config.NoVNCPort)
}

// HealthCheck 返回健康状态
func (m *Manager) HealthCheck() map[string]bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]bool)

	if m.vncServer != nil {
		status["vnc"] = m.vncServer.IsRunning()
	}

	if m.novncProxy != nil {
		status["novnc"] = m.novncProxy.IsRunning()
	}

	return status
}

// validateConfig 验证配置
func validateConfig(cfg *Config) error {
	if cfg.Display == "" {
		cfg.Display = ":1"
	}

	if !strings.HasPrefix(cfg.Display, ":") {
		return fmt.Errorf("invalid display format: %s (expected ':N')", cfg.Display)
	}

	if cfg.Password == "" {
		return ErrInvalidPassword
	}

	if cfg.Geometry == "" {
		cfg.Geometry = "1920x1080"
	}

	if cfg.Depth <= 0 {
		cfg.Depth = 24
	}

	if cfg.VNCPort <= 0 {
		cfg.VNCPort = 5901
	}

	if cfg.NoVNCPort <= 0 {
		cfg.NoVNCPort = 6080
	}

	return nil
}

// VNCConfig VNC 服务器配置
type VNCConfig struct {
	Display  string
	Password string
	Geometry string
	Depth    int
	Port     int
}

// NewVNCServer 创建 VNC 服务器实例
func NewVNCServer(cfg *VNCConfig) (*VNCServer, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("get home directory: %w", err)
	}

	vncDir := filepath.Join(homeDir, ".vnc")
	if err := os.MkdirAll(vncDir, 0700); err != nil {
		return nil, fmt.Errorf("create VNC directory: %w", err)
	}

	return &VNCServer{
		display:    cfg.Display,
		password:   cfg.Password,
		geometry:   cfg.Geometry,
		depth:      cfg.Depth,
		port:       cfg.Port,
		passwordDB: filepath.Join(vncDir, "passwd"),
	}, nil
}

// Start 启动 VNC 服务器
func (v *VNCServer) Start(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.process != nil {
		return ErrVNCAlreadyRunning
	}

	// 1. 创建 VNC 密码文件
	if err := v.createPasswordFile(); err != nil {
		return fmt.Errorf("create password file: %w", err)
	}

	// 2. 启动 Xtigervnc
	args := []string{
		v.display,
		"-geometry", v.geometry,
		"-depth", fmt.Sprint(v.depth),
		"-rfbport", fmt.Sprint(v.port),
		"-rfbauth", v.passwordDB,
		"-SecurityTypes", "VncAuth",
		"-localhost", "no",
	}

	cmd := exec.CommandContext(ctx, "Xtigervnc", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("DISPLAY=%s", v.display))

	// 捕获输出
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start Xtigervnc: %w", err)
	}

	v.process = cmd.Process
	v.cmd = cmd

	// 3. 等待 X11 socket 就绪
	if err := v.waitForX11(ctx); err != nil {
		v.process.Kill()
		v.process = nil
		return fmt.Errorf("wait for X11: %w", err)
	}

	zap.L().Info("VNC server started",
		zap.String("display", v.display),
		zap.Int("port", v.port))

	return nil
}

// Stop 停止 VNC 服务器
func (v *VNCServer) Stop(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if v.process == nil {
		return nil
	}

	// 发送 SIGTERM
	if err := v.process.Signal(syscall.SIGTERM); err != nil {
		zap.L().Warn("Failed to send SIGTERM to VNC server", zap.Error(err))
	}

	// 等待进程退出
	done := make(chan error, 1)
	go func() {
		done <- v.cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// 超时，强制杀死
		v.process.Kill()
	case <-done:
		// 进程已退出
	}

	v.process = nil
	v.cmd = nil

	zap.L().Info("VNC server stopped")
	return nil
}

// IsRunning 返回 VNC 服务器是否正在运行
func (v *VNCServer) IsRunning() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.process == nil {
		return false
	}

	// 检查进程是否还存在
	return v.process.Signal(syscall.Signal(0)) == nil
}

// GetDisplay 返回 X11 显示号
func (v *VNCServer) GetDisplay() string {
	return v.display
}

// HealthCheck 检查 X11 socket 是否可访问
func (v *VNCServer) HealthCheck() error {
	displayNum := strings.TrimPrefix(v.display, ":")
	socketPath := fmt.Sprintf("/tmp/.X11-unix/X%s", displayNum)

	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return ErrDisplayNotFound
	}

	return nil
}

// createPasswordFile 创建 VNC 密码文件
func (v *VNCServer) createPasswordFile() error {
	// 使用 tigervncpasswd 创建密码文件
	cmd := exec.Command("tigervncpasswd", "-f")
	cmd.Stdin = strings.NewReader(v.password + "\n" + v.password + "\n")
	cmd.Stdout = nil
	cmd.Stderr = nil

	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("run tigervncpasswd: %w", err)
	}

	// 写入密码文件
	if err := os.WriteFile(v.passwordDB, output, 0600); err != nil {
		return fmt.Errorf("write password file: %w", err)
	}

	return nil
}

// waitForX11 等待 X11 socket 就绪
func (v *VNCServer) waitForX11(ctx context.Context) error {
	displayNum := strings.TrimPrefix(v.display, ":")
	socketPath := fmt.Sprintf("/tmp/.X11-unix/X%s", displayNum)

	timeout := time.After(10 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for X11 socket: %s", socketPath)
		case <-ticker.C:
			if _, err := os.Stat(socketPath); err == nil {
				return nil
			}
		}
	}
}

// NoVNCConfig noVNC 代理配置
type NoVNCConfig struct {
	VNCPort    int
	ListenPort int
}

// NewNoVNCProxy 创建 noVNC 代理实例
func NewNoVNCProxy(cfg *NoVNCConfig) (*NoVNCProxy, error) {
	return &NoVNCProxy{
		vncPort:    cfg.VNCPort,
		listenPort: cfg.ListenPort,
	}, nil
}

// Start 启动 noVNC 代理
func (n *NoVNCProxy) Start(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.process != nil {
		return ErrNoVNCFailed
	}

	// 查找 novnc_proxy 脚本
	novncPath := "/usr/share/novnc/utils/novnc_proxy"
	if _, err := os.Stat(novncPath); os.IsNotExist(err) {
		return fmt.Errorf("novnc_proxy not found at %s: %w", novncPath, err)
	}

	// 启动 websockify
	args := []string{
		"--vnc", fmt.Sprintf("localhost:%d", n.vncPort),
		"--listen", fmt.Sprint(n.listenPort),
	}

	cmd := exec.CommandContext(ctx, novncPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start novnc_proxy: %w", err)
	}

	n.process = cmd.Process
	n.cmd = cmd

	// 等待端口监听
	if err := n.waitForPort(ctx); err != nil {
		n.process.Kill()
		n.process = nil
		return fmt.Errorf("wait for port: %w", err)
	}

	zap.L().Info("noVNC proxy started",
		zap.Int("listen_port", n.listenPort),
		zap.Int("vnc_port", n.vncPort))

	return nil
}

// Stop 停止 noVNC 代理
func (n *NoVNCProxy) Stop(ctx context.Context) error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.process == nil {
		return nil
	}

	// 发送 SIGTERM
	if err := n.process.Signal(syscall.SIGTERM); err != nil {
		zap.L().Warn("Failed to send SIGTERM to noVNC proxy", zap.Error(err))
	}

	// 等待进程退出
	done := make(chan error, 1)
	go func() {
		done <- n.cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// 超时，强制杀死
		n.process.Kill()
	case <-done:
		// 进程已退出
	}

	n.process = nil
	n.cmd = nil

	zap.L().Info("noVNC proxy stopped")
	return nil
}

// IsRunning 返回 noVNC 代理是否正在运行
func (n *NoVNCProxy) IsRunning() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()

	if n.process == nil {
		return false
	}

	return n.process.Signal(syscall.Signal(0)) == nil
}

// GetURL 返回 Web 访问 URL
func (n *NoVNCProxy) GetURL() string {
	hostname := "localhost"
	if h := os.Getenv("HOSTNAME"); h != "" {
		hostname = h
	}
	return fmt.Sprintf("http://%s:%d/vnc.html", hostname, n.listenPort)
}

// HealthCheck 检查端口是否监听
func (n *NoVNCProxy) HealthCheck() error {
	// 简单检查：尝试连接端口
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", n.listenPort), 1*time.Second)
	if err != nil {
		return fmt.Errorf("port %d not listening: %w", n.listenPort, err)
	}
	conn.Close()
	return nil
}

// waitForPort 等待端口开始监听
func (n *NoVNCProxy) waitForPort(ctx context.Context) error {
	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for port %d", n.listenPort)
		case <-ticker.C:
			if n.HealthCheck() == nil {
				return nil
			}
		}
	}
}
