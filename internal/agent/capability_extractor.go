// Package agent provides the core Agent runtime, ReAct loop, and domain types.
package agent

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/suifei/gopherpaw/internal/agent/cache"
	"github.com/suifei/gopherpaw/internal/config"
	"github.com/suifei/gopherpaw/internal/skills"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// Extractor 能力提取器，从工具、技能、MCP 中提取系统能力。
type Extractor struct {
	mu            sync.RWMutex
	llm           LLMProvider
	tools         []Tool
	// 使用 any 类型避免循环导入
	// 实际类型可能是 *skills.Manager 或 *mcp.Manager
	skillsMgr     any
	mcpMgr        any
	registryMgr   *cache.RegistryManager
	cfg           config.AgentConfig
	workingDir    string
	skillsActiveDir string
}

// NewExtractor 创建一个新的能力提取器。
func NewExtractor(
	llm LLMProvider,
	tools []Tool,
	skillsMgr any,
	mcpMgr any,
	cfg config.AgentConfig,
	workingDir string,
	skillsActiveDir string,
	cacheTTLHours int,
) *Extractor {
	version := cache.DefaultVersion()
	return &Extractor{
		llm:           llm,
		tools:         tools,
		skillsMgr:     skillsMgr,
		mcpMgr:        mcpMgr,
		registryMgr:   cache.NewRegistryManager(cacheTTLHours, version),
		cfg:           cfg,
		workingDir:    workingDir,
		skillsActiveDir: skillsActiveDir,
	}
}

// ExtractCapabilities 提取所有系统能力并生成总结。
func (e *Extractor) ExtractCapabilities(ctx context.Context) (*cache.CapabilityRegistry, error) {
	log := logger.L()

	// 检查缓存是否可用
	if !e.registryMgr.Refresh() {
		registry, err := e.registryMgr.Get()
		if err == nil && registry != nil {
			log.Debug("Using cached capabilities",
				zap.Int("count", len(registry.Capabilities)),
			)
			return registry, nil
		}
	}

	log.Info("Extracting system capabilities")

	// 1. 提取所有能力
	caps, err := e.extractAllCapabilities()
	if err != nil {
		return nil, fmt.Errorf("extract capabilities: %w", err)
	}

	log.Info("Capabilities extracted",
		zap.Int("total", len(caps)),
		zap.Int("tools", e.countByType(caps, "tool")),
		zap.Int("skills", e.countByType(caps, "skill")),
		zap.Int("mcp", e.countByType(caps, "mcp")),
	)

	// 2. 生成 AI 总结
	summary, err := e.summarizeCapabilities(ctx, caps)
	if err != nil {
		log.Warn("Failed to generate AI summary, using fallback", zap.Error(err))
		summary = e.generateFallbackSummary(caps)
	}

	// 3. 构建注册表
	registry := &cache.CapabilityRegistry{
		Capabilities: caps,
		Summary:      summary,
		UpdatedAt:    0, // 将在 Set 中设置
	}

	// 4. 保存到缓存
	if err := e.registryMgr.Set(registry); err != nil {
		log.Warn("Failed to save capability cache", zap.Error(err))
	}

	return registry, nil
}

// extractAllCapabilities 从所有来源提取能力。
func (e *Extractor) extractAllCapabilities() ([]cache.Capability, error) {
	var caps []cache.Capability

	// 1. 提取内置工具
	for _, t := range e.tools {
		caps = append(caps, cache.Capability{
			ID:          fmt.Sprintf("tool:%s", t.Name()),
			Type:        "tool",
			Name:        t.Name(),
			Description: t.Description(),
			Parameters:  t.Parameters(),
			Metadata: map[string]string{
				"source": "builtin",
			},
		})
	}

	// 2. 提取 MCP 工具（通过接口调用）
	if e.mcpMgr != nil {
		mcpTools := e.getMCPTools()
		for _, t := range mcpTools {
			caps = append(caps, cache.Capability{
				ID:          fmt.Sprintf("mcp:%s", t.Name()),
				Type:        "mcp",
				Name:        t.Name(),
				Description: t.Description(),
				Parameters:  t.Parameters(),
				Metadata: map[string]string{
					"source": "mcp",
				},
			})
		}
	}

	// 3. 提取技能（通过接口调用）
	if e.skillsMgr != nil {
		skills := e.getSkills()
		for _, s := range skills {
			caps = append(caps, cache.Capability{
				ID:          fmt.Sprintf("skill:%s", s.name),
				Type:        "skill",
				Name:        s.name,
				Description: s.description,
				Metadata: map[string]string{
					"path":     s.path,
					"enabled":  fmt.Sprintf("%v", s.enabled),
				},
			})
		}
	}

	return caps, nil
}

// skillInfo 技能信息（避免直接导入 skills 包）
type skillInfo struct {
	name        string
	description string
	path        string
	enabled     bool
	keywords    []string
}

// getSkills 从技能管理器获取技能列表。
func (e *Extractor) getSkills() []skillInfo {
	var result []skillInfo

	// 使用类型断言获取技能管理器
	if mgr, ok := e.skillsMgr.(*skills.Manager); ok {
		for _, s := range mgr.GetEnabledSkills() {
			result = append(result, skillInfo{
				name:        s.GetName(),
				description: s.GetDescription(),
				path:        s.GetPath(),
				enabled:     s.GetEnabled(),
				keywords:    s.GetKeywords(),
			})
		}
	}

	return result
}

// getMCPTools 从 MCP 管理器获取工具列表。
func (e *Extractor) getMCPTools() []Tool {
	// 使用接口调用避免导入循环
	if mgr, ok := e.mcpMgr.(interface {
		GetTools() []Tool
	}); ok {
		return mgr.GetTools()
	}
	return nil
}

// summarizeCapabilities 调用 LLM 生成能力总结。
func (e *Extractor) summarizeCapabilities(ctx context.Context, caps []cache.Capability) (string, error) {
	log := logger.L()

	// 按类型分组
	byType := make(map[string][]cache.Capability)
	for _, c := range caps {
		byType[c.Type] = append(byType[c.Type], c)
	}

	// 构建提示词
	var promptBuilder strings.Builder
	promptBuilder.WriteString("请分析以下可用能力，生成一个简洁的能力总结。\n\n")
	promptBuilder.WriteString("要求：\n")
	promptBuilder.WriteString("1. 按类别（工具、技能、MCP）分组\n")
	promptBuilder.WriteString("2. 每个类别列出最重要的能力（不超过 10 个）\n")
	promptBuilder.WriteString("3. 突出关键功能和用途\n")
	promptBuilder.WriteString("4. 输出格式为 Markdown，简洁明了\n\n")
	promptBuilder.WriteString("可用能力：\n\n")

	// 按类型输出能力
	typeOrder := []string{"tool", "skill", "mcp"}
	typeNames := map[string]string{
		"tool":  "## 内置工具",
		"skill": "## 技能",
		"mcp":   "## MCP 工具",
	}

	for _, typ := range typeOrder {
		items, ok := byType[typ]
		if !ok || len(items) == 0 {
			continue
		}

		promptBuilder.WriteString(typeNames[typ] + "\n\n")

		// 限制每种类型最多输出 15 个
		maxItems := 15
		if len(items) > maxItems {
			items = items[:maxItems]
		}

		for _, c := range items {
			promptBuilder.WriteString(fmt.Sprintf("- **%s**: %s\n", c.Name, c.Description))
		}

		if len(byType[typ]) > maxItems {
			promptBuilder.WriteString(fmt.Sprintf("... 以及其他 %d 个%s\n", len(byType[typ])-maxItems, typ))
		}

		promptBuilder.WriteString("\n")
	}

	req := &ChatRequest{
		Messages: []Message{
			{Role: "system", Content: "你是一个能力分析助手，擅长总结和分类系统能力。"},
			{Role: "user", Content: promptBuilder.String()},
		},
		MaxTokens: 2000,
	}

	resp, err := e.llm.Chat(ctx, req)
	if err != nil {
		return "", err
	}

	log.Debug("AI capability summary generated",
		zap.Int("summaryLen", len(resp.Content)),
	)

	return resp.Content, nil
}

// generateFallbackSummary 生成简单的回退总结（当 LLM 调用失败时）。
func (e *Extractor) generateFallbackSummary(caps []cache.Capability) string {
	byType := make(map[string][]cache.Capability)
	for _, c := range caps {
		byType[c.Type] = append(byType[c.Type], c)
	}

	var sb strings.Builder
	sb.WriteString("# 系统能力概览\n\n")

	typeOrder := []string{"tool", "skill", "mcp"}
	typeNames := map[string]string{
		"tool":  "内置工具",
		"skill": "技能",
		"mcp":   "MCP 工具",
	}

	for _, typ := range typeOrder {
		items := byType[typ]
		if len(items) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("## %s (%d 个)\n\n", typeNames[typ], len(items)))

		for _, c := range items {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", c.Name, c.Description))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// GetRegistry 获取缓存的能力注册表。
func (e *Extractor) GetRegistry() (*cache.CapabilityRegistry, error) {
	return e.registryMgr.Get()
}

// Refresh 强制刷新能力注册表。
func (e *Extractor) Refresh(ctx context.Context) error {
	// 清除缓存
	if err := e.registryMgr.Clear(); err != nil {
		return fmt.Errorf("clear cache: %w", err)
	}

	// 重新提取
	_, err := e.ExtractCapabilities(ctx)
	return err
}

// GetSummary 快速获取能力总结。
func (e *Extractor) GetSummary() (string, error) {
	summary, err := e.registryMgr.GetSummary()
	if err == nil && summary != "" {
		return summary, nil
	}

	// 如果缓存不存在，尝试从完整注册表获取
	registry, err := e.registryMgr.Get()
	if err != nil {
		return "", err
	}

	return registry.Summary, nil
}

// countByType 统计指定类型的能力数量。
func (e *Extractor) countByType(caps []cache.Capability, typ string) int {
	count := 0
	for _, c := range caps {
		if c.Type == typ {
			count++
		}
	}
	return count
}
