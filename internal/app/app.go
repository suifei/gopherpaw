// Package app 提供统一的应用生命周期管理
package app

import (
	"context"
	"fmt"

	"go.uber.org/zap"

	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/internal/scheduler"
)

// NewApp 创建应用实例
func NewApp(cfg *config.Config) (*App, error) {
	ctx, cancel := context.WithCancel(context.Background())
	return &App{
		Config:  cfg,
		ctx:     ctx,
		cancel:  cancel,
	}, nil
}

// Start 启动所有服务
func (a *App) Start(ctx context.Context) error {
	zap.L().Info("Starting GopherPaw application...")

	// 1. 启动 Cron Scheduler（最简单的组件）
	if err := a.startCron(ctx); err != nil {
		return fmt.Errorf("start cron: %w", err)
	}

	zap.L().Info("GopherPaw application started successfully")
	return nil
}

// Stop 停止所有服务
func (a *App) Stop(ctx context.Context) error {
	zap.L().Info("Stopping GopherPaw application...")

	if err := a.stopCron(ctx); err != nil {
		zap.L().Error("Failed to stop cron", zap.Error(err))
	}

	a.cancel()
	zap.L().Info("GopherPaw application stopped successfully")
	return nil
}

// HealthCheck 健康检查
func (a *App) HealthCheck() map[string]bool {
	return map[string]bool{
		"cron": a.CronScheduler != nil,
	}
}

func (a *App) startCron(ctx context.Context) error {
	cron := scheduler.New(nil, a.Config.Scheduler)
	if err := cron.Start(ctx); err != nil {
		return err
	}
	a.CronScheduler = cron
	return nil
}

func (a *App) stopCron(ctx context.Context) error {
	if a.CronScheduler == nil {
		return nil
	}
	return a.CronScheduler.Stop(ctx)
}
