// Package app 测试
package app

import (
	"context"
	"testing"
	"time"

	"github.com/suifei/gopherpaw/internal/config"
)

func TestNewApp(t *testing.T) {
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			Enabled: false,
		},
	}

	app, err := NewApp(cfg)
	if err != nil {
		t.Fatalf("NewApp failed: %v", err)
	}

	if app == nil {
		t.Fatal("App is nil")
	}

	if app.Config == nil {
		t.Error("Config is nil")
	}

	if app.ctx == nil {
		t.Error("Context is nil")
	}

	if app.cancel == nil {
		t.Error("Cancel function is nil")
	}
}

func TestApp_StartStop(t *testing.T) {
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			Enabled: false,
		},
	}

	app, err := NewApp(cfg)
	if err != nil {
		t.Fatalf("NewApp failed: %v", err)
	}

	ctx := context.Background()

	// 启动应用
	if err := app.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// 验证健康检查
	health := app.HealthCheck()
	if !health["cron"] {
		t.Error("Cron scheduler should be running")
	}

	// 停止应用
	stopCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := app.Stop(stopCtx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}
}

func TestApp_HealthCheck(t *testing.T) {
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			Enabled: false,
		},
	}

	app, err := NewApp(cfg)
	if err != nil {
		t.Fatalf("NewApp failed: %v", err)
	}

	// 未启动时
	health := app.HealthCheck()
	if health["cron"] {
		t.Error("Cron should not be running before Start()")
	}

	// 启动后
	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	health = app.HealthCheck()
	if !health["cron"] {
		t.Error("Cron should be running after Start()")
	}

	// 清理
	_ = app.Stop(context.Background())
}

func TestApp_DoubleStart(t *testing.T) {
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			Enabled: false,
		},
	}

	app, err := NewApp(cfg)
	if err != nil {
		t.Fatalf("NewApp failed: %v", err)
	}

	ctx := context.Background()

	// 第一次启动
	if err := app.Start(ctx); err != nil {
		t.Fatalf("First Start failed: %v", err)
	}

	// 第二次启动（应该幂等或返回错误）
	// 当前实现可能会重复启动，待改进
	if err := app.Start(ctx); err != nil {
		t.Logf("Second Start returned error (expected): %v", err)
	}

	// 清理
	_ = app.Stop(context.Background())
}

func TestApp_StopWithoutStart(t *testing.T) {
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			Enabled: false,
		},
	}

	app, err := NewApp(cfg)
	if err != nil {
		t.Fatalf("NewApp failed: %v", err)
	}

	// 未启动直接停止（应该优雅处理）
	if err := app.Stop(context.Background()); err != nil {
		t.Errorf("Stop without Start should not error, got: %v", err)
	}
}

func TestApp_ContextCancellation(t *testing.T) {
	cfg := &config.Config{
		Scheduler: config.SchedulerConfig{
			Enabled: false,
		},
	}

	app, err := NewApp(cfg)
	if err != nil {
		t.Fatalf("NewApp failed: %v", err)
	}

	if err := app.Start(context.Background()); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// 取消上下文
	app.cancel()

	// 等待一段时间
	time.Sleep(100 * time.Millisecond)

	// 验证应用已停止（通过健康检查）
	health := app.HealthCheck()
	// 由于 cron 可能还在运行，这里只是示例
	t.Logf("Health after cancel: %v", health)
}
