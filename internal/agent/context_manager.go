// Package agent provides context management and storage for AI collaboration.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// memoryContextManager 实现 ContextManager 接口，使用内存存储。
type memoryContextManager struct {
	mu sync.RWMutex

	// per-chat storage
	storage   map[string]map[string]StoredContent // chatID -> name -> content
	goals     map[string]string                   // chatID -> current goal
	milestones map[string][]Milestone              // chatID -> milestones

	// 技能路径缓存 (用于能力提醒)
	skillPaths map[string]string // skill name -> skill path
}

// NewMemoryContextManager 创建一个基于内存的上下文管理器。
func NewMemoryContextManager() ContextManager {
	return &memoryContextManager{
		storage:     make(map[string]map[string]StoredContent),
		goals:       make(map[string]string),
		milestones:  make(map[string][]Milestone),
		skillPaths:  make(map[string]string),
	}
}

// SetSkillPaths 设置技能路径映射，用于智能提点。
func (m *memoryContextManager) SetSkillPaths(paths map[string]string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.skillPaths = paths
}

// Store 存储内容供后续使用。
func (m *memoryContextManager) Store(ctx context.Context, chatID string, requests []StorageRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.storage[chatID]; !exists {
		m.storage[chatID] = make(map[string]StoredContent)
	}

	now := time.Now().Unix()
	for _, req := range requests {
		if req.Name == "" {
			continue
		}
		m.storage[chatID][req.Name] = StoredContent{
			Name:        req.Name,
			Description: req.Description,
			Content:     req.Content,
			Timestamp:   now,
		}
		logger.L().Debug("content stored",
			zap.String("chatID", chatID),
			zap.String("name", req.Name),
			zap.Int("contentLen", len(req.Content)))
	}

	return nil
}

// Retrieve 检索已存储的内容。
func (m *memoryContextManager) Retrieve(ctx context.Context, chatID string, names []string) ([]StoredContent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chatStorage, exists := m.storage[chatID]
	if !exists {
		return []StoredContent{}, nil
	}

	var results []StoredContent
	for _, name := range names {
		if content, ok := chatStorage[name]; ok {
			results = append(results, content)
		}
	}

	return results, nil
}

// ListAll 列出所有已存储的内容名称。
func (m *memoryContextManager) ListAll(ctx context.Context, chatID string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	chatStorage, exists := m.storage[chatID]
	if !exists {
		return []string{}, nil
	}

	names := make([]string, 0, len(chatStorage))
	for name := range chatStorage {
		names = append(names, name)
	}
	return names, nil
}

// SetGoal 设置当前目标。
func (m *memoryContextManager) SetGoal(ctx context.Context, chatID string, goal string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.goals[chatID] = goal
	logger.L().Debug("goal set", zap.String("chatID", chatID), zap.String("goal", goal))
	return nil
}

// GetGoal 获取当前目标。
func (m *memoryContextManager) GetGoal(ctx context.Context, chatID string) (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.goals[chatID], nil
}

// AddMilestone 添加进度里程碑。
func (m *memoryContextManager) AddMilestone(ctx context.Context, chatID string, milestone Milestone) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if milestone.Timestamp == 0 {
		milestone.Timestamp = time.Now().Unix()
	}

	m.milestones[chatID] = append(m.milestones[chatID], milestone)
	logger.L().Debug("milestone added",
		zap.String("chatID", chatID),
		zap.String("name", milestone.Name),
		zap.String("status", milestone.Status))
	return nil
}

// GetMilestones 获取进度里程碑。
func (m *memoryContextManager) GetMilestones(ctx context.Context, chatID string) ([]Milestone, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.milestones[chatID], nil
}

// InjectContext 注入上下文到消息中。
// 如果有检索请求，会自动注入存储的内容。
// 也会注入当前目标和进展（如果存在）。
func (m *memoryContextManager) InjectContext(ctx context.Context, chatID string, messages []Message, retrievalRequests []string) ([]Message, error) {
	if len(messages) == 0 {
		return messages, nil
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var contextParts []string

	// 1. 处理检索请求
	if len(retrievalRequests) > 0 {
		chatStorage, exists := m.storage[chatID]
		if exists {
			var retrieved []string
			for _, name := range retrievalRequests {
				if content, ok := chatStorage[name]; ok {
					retrieved = append(retrieved, fmt.Sprintf("**%s** (%s)\n%s",
						content.Name, content.Description, content.Content))
				}
			}
			if len(retrieved) > 0 {
				contextParts = append(contextParts,
					"--- 📦 已检索存储内容 ---\n"+
						strings.Join(retrieved, "\n\n")+
						"\n--- 存储内容结束 ---")
			}
		}
	}

	// 2. 注入当前目标
	if goal, ok := m.goals[chatID]; ok && goal != "" {
		contextParts = append(contextParts, fmt.Sprintf("--- 🎯 当前目标 ---\n%s\n--- 目标结束 ---", goal))
	}

	// 3. 注入进展里程碑
	if milestones, ok := m.milestones[chatID]; ok && len(milestones) > 0 {
		var milestoneStrs []string
		for _, ms := range milestones {
			milestoneStrs = append(milestoneStrs,
				fmt.Sprintf("- [%s] %s", ms.Status, ms.Name))
		}
		contextParts = append(contextParts,
			"--- 📊 当前进度 ---\n"+
				strings.Join(milestoneStrs, "\n")+
				"\n--- 进度结束 ---")
	}

	// 如果没有上下文要注入，直接返回
	if len(contextParts) == 0 {
		return messages, nil
	}

	// 将上下文注入到最后一条用户消息中
	// 这样可以让 AI 在回复前看到上下文
	result := make([]Message, len(messages))
	copy(result, messages)

	for i := len(result) - 1; i >= 0; i-- {
		if result[i].Role == "user" {
			result[i].Content = result[i].Content + "\n\n" + strings.Join(contextParts, "\n\n")
			break
		}
	}

	return result, nil
}

// BuildCapabilityReminder 构建能力提醒（智能提点）。
// 当 AI 请求某个能力但可能没有加载对应技能时，返回友好的提醒。
func (m *memoryContextManager) BuildCapabilityReminder(ctx context.Context, chatID string, capabilitiesNeeded []string) string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(capabilitiesNeeded) == 0 {
		return ""
	}

	var reminders []string

	// 检查每个需要的能力
	for _, capability := range capabilitiesNeeded {
		// 如果是已知的技能，返回对应的 SKILL.md 路径
		if skillPath, ok := m.skillPaths[capability]; ok {
			// 将绝对路径转换为相对路径（假设相对于 working dir）
			relPath := skillPath
			if filepath.IsAbs(skillPath) {
				// 尝试简化路径显示
				if strings.Contains(skillPath, "configs/active_skills") {
					parts := strings.Split(skillPath, "configs/active_skills/")
					if len(parts) > 1 {
						relPath = "configs/active_skills/" + parts[1]
					}
				}
			}
			reminders = append(reminders,
				fmt.Sprintf("💡 你需要 **%s** 能力，可以先用 `read_file` 读取 `%s` 了解如何使用。",
					capability, relPath))
		}
	}

	if len(reminders) == 0 {
		return ""
	}

	return "\n\n--- 框架智能提示 ---\n" +
		strings.Join(reminders, "\n\n") +
		"\n--- 提示结束 ---\n"
}

// ============================================================================
// 辅助函数
// ============================================================================

// ParseStructuredResponse 从 AI 响应内容中解析结构化响应。
// 支持两种格式：
// 1. 纯 JSON：整个内容就是一个 JSON 对象
// 2. Markdown 代码块：内容包含 ```json ... ``` 代码块
func ParseStructuredResponse(content string) (*StructuredResponse, error) {
	trimmed := strings.TrimSpace(content)

	// 尝试提取 JSON 代码块
	if strings.HasPrefix(trimmed, "```") {
		// 找到代码块结束
		idx := strings.Index(trimmed, "\n```")
		if idx > 0 {
			// 跳过第一行 ```json 或 ```
			firstNewline := strings.Index(trimmed, "\n")
			if firstNewline > 0 && firstNewline < idx {
				trimmed = strings.TrimSpace(trimmed[firstNewline+1 : idx])
			}
		}
	}

	var resp StructuredResponse
	if err := json.Unmarshal([]byte(trimmed), &resp); err != nil {
		// JSON 解析失败，返回 nil（不是错误，AI 可能选择不使用结构化响应）
		return nil, nil
	}

	return &resp, nil
}

// ExtractStructuredContent 从结构化响应中提取最终回答内容。
// 如果 AI 使用了结构化响应，返回 FinalAnswer 字段；
// 否则返回原始内容。
func ExtractStructuredContent(originalContent string) string {
	structured, err := ParseStructuredResponse(originalContent)
	if err != nil || structured == nil {
		return originalContent
	}

	if structured.FinalAnswer != "" {
		return structured.FinalAnswer
	}

	// 如果没有 FinalAnswer 但有 Thought，返回 Thought
	if structured.Thought != "" {
		return structured.Thought
	}

	return originalContent
}

// IsFinalResponse 判断结构化响应是否表示最终响应。
func IsFinalResponse(content string) bool {
	structured, err := ParseStructuredResponse(content)
	if err != nil || structured == nil {
		// 不是结构化响应，无法判断，调用者应该根据其他逻辑判断
		return false
	}

	return structured.FinalAnswer != ""
}
