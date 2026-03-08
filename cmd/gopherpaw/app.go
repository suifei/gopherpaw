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
	Use:   "app [task]",
	Short: "Start the GopherPaw service",
	Long:  "Starts channels (console, telegram, etc.) and scheduler. Press Ctrl+C to shutdown.\n\n" +
		"Args:\n" +
		"  task    Optional initial task to execute on startup.",
	RunE:  runApp,
}

var (
	skipEnvCheck bool
	runOnce      bool // 执行完初始任务后是否退出
)

func init() {
	appCmd.Flags().BoolVar(&skipEnvCheck, "skip-env-check", false, "Skip runtime environment check on startup")
	appCmd.Flags().BoolVar(&runOnce, "once", false, "Exit after executing the initial task")
}

// memoryHookCount returns 1 if memory compaction is configured, 0 otherwise.
func memoryHookCount(threshold int) int {
	if threshold > 0 {
		return 1
	}
	return 0
}

// buildSkillPathMap 构建技能名称到 SKILL.md 路径的简单映射。
// 只用于记录哪些技能可用，具体的技能选择由 AI 自己分析决定。
func buildSkillPathMap(skillMgr *skills.Manager, workingDir string) map[string]string {
	result := make(map[string]string)

	// 获取所有已启用的技能
	allSkills := skillMgr.GetEnabledSkills()
	for _, skill := range allSkills {
		if skill.Path == "" {
			continue
		}

		// 转换为相对于 workingDir 的路径
		relPath := skill.Path
		if filepath.IsAbs(skill.Path) {
			if rp, err := filepath.Rel(workingDir, skill.Path); err == nil {
				relPath = rp
			}
		}

		// 只记录技能名称 -> 路径的映射
		// AI 会基于技能的 description 和 content 来判断是否使用
		result[skill.Name] = relPath
	}

	return result
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

		memoryStore := memory.New(cfg.Memory, llmProvider)
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

		// Create PromptLoader with fallback to config system prompt
		agentCfg := cfg.Agent
		loader := agent.NewPromptLoader(workingDir, agentCfg.SystemPrompt)

		// Get skill index for lazy loading (AgentScope-style)
		// Use compact version for better visibility
		skillIndex := skillMgr.GetSkillIndexCompact(workingDir)

		// Debug log for skill index (temporary)
		if skillIndex != "" {
			preview := skillIndex
			if len(preview) > 300 {
				preview = preview[:300] + "..."
			}
			log.Info("Skill index generated",
				zap.Int("length", len(skillIndex)),
				zap.String("preview", preview))
		}

		// Create agent with PromptLoader and skill index
		// This ensures skill index is passed to BuildSystemPrompt() whether
		// using six-file system or falling back to config system prompt
		ag := agent.NewReactWithPrompt(llmProvider, memoryStore, toolsList, agentCfg, loader, skillIndex)

		// 设置技能路径映射，用于智能提点
		skillPathMap := buildSkillPathMap(skillMgr, workingDir)
		ag.SetSkillPaths(skillPathMap)
		log.Info("Skill paths configured for capability reminders", zap.Int("skills", len(skillPathMap)))

		// 配置规划-执行分离模式
		if agentCfg.Planning != nil && agentCfg.Planning.Enabled {
			// 获取执行模式：优先使用 planning.execution_mode，其次使用 running.execution_mode
			executionMode := agentCfg.Planning.ExecutionMode
			if executionMode == "" {
				executionMode = agentCfg.Running.ExecutionMode
			}
			if executionMode == "" {
				executionMode = "auto" // 默认自动模式
			}

			ag.SetExecutionMode(executionMode)
			ag.EnablePlanningMode(true)

			// 设置技能和 MCP 管理器（用于规划模式）
			ag.SetSkillsManager(skillMgr)
			ag.SetMCPManager(mcpMgr)

			// 设置工作目录
			skillsActiveDir := filepath.Join(configDir, cfg.Skills.ActiveDir)
			ag.SetWorkingDirectories(workingDir, skillsActiveDir)

			// 设置能力缓存 TTL
			cacheTTL := agentCfg.Planning.CapabilityCacheTTL
			if cacheTTL <= 0 {
				cacheTTL = 24 // 默认 24 小时
			}
			ag.SetCapabilityCacheTTL(cacheTTL)

			log.Info("Planning-Execution mode enabled",
				zap.String("executionMode", executionMode),
				zap.Int("cacheTTL", cacheTTL),
				zap.Bool("aiSummary", agentCfg.Planning.AISummaryEnabled),
			)
		}

		// Add Bootstrap Hook for first-time user guidance
		ag.AddHook(agent.BootstrapHook(workingDir, cfg.Agent.Language))

		// Add Skill Reminder Hook to detect relevant skills before processing
		ag.AddHook(agent.SkillReminderHook(skillMgr))

		// Add File Extension Skill Hook for document creation detection
		ag.AddHook(agent.FileExtensionSkillHook(skillMgr))

		// Add Memory Compaction Hook if threshold is configured
		if cfg.Memory.CompactThreshold > 0 {
			ag.AddHook(agent.MemoryCompactionHook(cfg.Memory.CompactThreshold, cfg.Memory.CompactKeepRecent))
		}

		log.Info("Agent ready",
			zap.String("agent", "react"),
			zap.Int("hooks", 3+memoryHookCount(cfg.Memory.CompactThreshold)), // BootstrapHook + SkillReminderHook + FileExtensionSkillHook + optional MemoryCompactionHook
		)

		channelMgr := channels.NewManager(ag, cfg.Channels)
		sched := scheduler.New(ag, cfg.Scheduler)

		// 设置 Console 渠道的初始任务和 runOnce 选项
		if len(args) > 0 {
			for _, ch := range channelMgr.Channels() {
				if consoleCh, ok := ch.(*channels.ConsoleChannel); ok {
					consoleCh.SetInitialTasks(args)
					consoleCh.SetRunOnce(runOnce)
					break
				}
			}
		}

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
