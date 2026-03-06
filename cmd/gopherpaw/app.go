package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/internal/channels"
	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/internal/llm"
	"github.com/suifei/gopherpaw/internal/mcp"
	"github.com/suifei/gopherpaw/internal/memory"
	"github.com/suifei/gopherpaw/internal/runtime"
	"github.com/suifei/gopherpaw/internal/scheduler"
	"github.com/suifei/gopherpaw/internal/skills"
	"github.com/suifei/gopherpaw/internal/tools"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

var appCmd = &cobra.Command{
	Use:   "app",
	Short: "Start the GopherPaw service",
	Long:  "Starts channels (console, telegram, etc.) and scheduler. Press Ctrl+C to shutdown.",
	RunE:  runApp,
}

var skipEnvCheck bool

func init() {
	appCmd.Flags().BoolVar(&skipEnvCheck, "skip-env-check", false, "Skip runtime environment check on startup")
}

func runApp(cmd *cobra.Command, args []string) error {
	cfgPath, _ := cmd.Root().PersistentFlags().GetString("config")
	if cfgPath == "" {
		cfgPath = "configs/config.yaml"
	}
	if p := os.Getenv("GOPHERPAW_CONFIG"); p != "" {
		cfgPath = p
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if err := logger.Init(logger.Config{
		Level:  cfg.Log.Level,
		Format: cfg.Log.Format,
	}); err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer logger.L().Sync()

	log := logger.L()
	log.Info("GopherPaw starting", zap.String("config", cfgPath))

	// Initialize runtime environment
	rtMgr := runtime.NewManager(&cfg.Runtime)
	if err := rtMgr.Initialize(); err != nil {
		log.Warn("Runtime initialization had issues", zap.Error(err))
	}

	// Print environment check unless skipped
	if !skipEnvCheck {
		rtMgr.PrintEnvironmentReport()
	}

	restartCh := make(chan struct{}, 1)
	var cfgMu sync.RWMutex
	currentCfg := cfg

	reloadConfig := func() error {
		newCfg, err := config.Load(cfgPath)
		if err != nil {
			return err
		}
		cfgMu.Lock()
		currentCfg = newCfg
		cfgMu.Unlock()
		return nil
	}

	for {
		cfgMu.RLock()
		cfg = currentCfg
		cfgMu.RUnlock()

		workingDir := config.ResolveWorkingDir(cfg.WorkingDir)
		if err := os.MkdirAll(workingDir, 0755); err != nil {
			return fmt.Errorf("create working dir %s: %w", workingDir, err)
		}
		tools.SetWorkingDir(workingDir)

		llmProvider, err := llm.NewModelRouter(cfg.LLM)
		if err != nil {
			return fmt.Errorf("create LLM provider: %w", err)
		}
		log.Info("LLM provider ready",
			zap.String("provider", llmProvider.Name()),
			zap.Strings("slots", llmProvider.SlotNames()),
			zap.String("active", llmProvider.ActiveSlot()),
		)

		memoryStore := memory.New(cfg.Memory)
		toolsList := tools.RegisterBuiltin()
		log.Info("Memory and tools ready", zap.Int("tools", len(toolsList)))

		// Initialize MCP Manager and load tools from MCP servers
		mcpMgr := mcp.NewManager()
		if len(cfg.MCP.Servers) > 0 {
			if err := mcpMgr.LoadConfig(cfg.MCP.Servers); err != nil {
				log.Warn("MCP config load failed", zap.Error(err))
			}
			mcpCtx, mcpCancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := mcpMgr.Start(mcpCtx); err != nil {
				log.Warn("MCP manager start failed", zap.Error(err))
			}
			mcpCancel()
			mcpTools := mcpMgr.GetTools()
			if len(mcpTools) > 0 {
				toolsList = append(toolsList, mcpTools...)
				log.Info("MCP tools loaded", zap.Int("mcpTools", len(mcpTools)), zap.Int("totalTools", len(toolsList)))
			}
		}

		configDir := filepath.Dir(cfgPath)
		skillMgr := skills.NewManager()
		if err := skillMgr.LoadSkills(workingDir, configDir, cfg.Skills); err != nil {
			log.Warn("load skills failed, continuing without skills", zap.Error(err))
		}
		systemPrompt := cfg.Agent.SystemPrompt
		if add := skillMgr.GetSystemPromptAddition(); add != "" {
			systemPrompt = systemPrompt + "\n\n" + add
		}
		agentCfg := cfg.Agent
		agentCfg.SystemPrompt = systemPrompt

		ag := agent.NewReact(llmProvider, memoryStore, toolsList, agentCfg)
		log.Info("Agent ready", zap.String("agent", "react"))

		channelMgr := channels.NewManager(ag, cfg.Channels)
		sched := scheduler.New(ag, cfg.Scheduler)

		webhookSrv := channels.NewWebhookServer(cfg.Server.Host, cfg.Server.Port)
		for _, ch := range channelMgr.Channels() {
			if wh, ok := ch.(channels.WebhookHandler); ok {
				webhookSrv.Register(ch.Name(), wh)
			}
		}

		ctx, cancel := context.WithCancel(context.Background())

		daemonInfo := &agent.DaemonInfo{
			Version:      version,
			Status:       "running",
			ReloadConfig: reloadConfig,
			Restart: func() error {
				select {
				case restartCh <- struct{}{}:
				default:
				}
				return nil
			},
		}
		daemonInfo.SwitchLLM = func(provider, model string) error {
			if err := llmProvider.Switch(provider); err == nil {
				return nil
			}
			cfgMu.RLock()
			llmCfg := currentCfg.LLM
			cfgMu.RUnlock()
			p, err := llm.SwitchProvider(provider, model, llmCfg)
			if err != nil {
				return err
			}
			ag.SetLLMProvider(p)
			return nil
		}
		channelMgr.SetDaemonInfo(daemonInfo)

		if err := channelMgr.Start(ctx); err != nil {
			cancel()
			return fmt.Errorf("start channels: %w", err)
		}
		if err := sched.Start(ctx); err != nil {
			cancel()
			return fmt.Errorf("start scheduler: %w", err)
		}

		webhookDone := make(chan struct{})
		go func() {
			defer close(webhookDone)
			if err := webhookSrv.Start(ctx); err != nil && err != http.ErrServerClosed {
				log.Warn("webhook server", zap.Error(err))
			}
		}()

		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		log.Info("GopherPaw ready. Press Ctrl+C to shutdown.")
		var doRestart bool
		select {
		case <-sigCh:
		case <-restartCh:
			doRestart = true
		case <-ctx.Done():
		}
		signal.Stop(sigCh)
		log.Info("Shutdown signal received, exiting...")
		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 15*time.Second)

		if err := channelMgr.Stop(shutdownCtx); err != nil {
			log.Warn("channel stop", zap.Error(err))
		}
		if err := sched.Stop(shutdownCtx); err != nil {
			log.Warn("scheduler stop", zap.Error(err))
		}
		if err := webhookSrv.Stop(shutdownCtx); err != nil {
			log.Warn("webhook stop", zap.Error(err))
		}
		// Stop MCP manager
		if err := mcpMgr.Stop(); err != nil {
			log.Warn("mcp stop", zap.Error(err))
		}
		<-webhookDone
		shutdownCancel()

		if !doRestart {
			return nil
		}
		if err := reloadConfig(); err != nil {
			log.Error("reload config for restart failed", zap.Error(err))
			return err
		}
		log.Info("Restarting...")
	}
}
