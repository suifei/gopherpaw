// Package agent provides the core Agent runtime, ReAct loop, and domain types.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// TaskPlanner 任务规划器，负责根据用户请求生成结构化的执行计划。
type TaskPlanner struct {
	llm    LLMProvider
	cfg    PlanningConfig
}

// PlanningConfig 规划器配置。
type PlanningConfig struct {
	AISummaryEnabled    bool   // 是否启用 AI 能力总结
	SimpleTaskThreshold int    // 简单任务阈值
	ConfirmPlan         bool   // 是否在规划前显示计划给用户确认
	MaxPlanSteps        int    // 最大计划步骤数
}

// DefaultPlanningConfig 返回默认的规划器配置。
func DefaultPlanningConfig() PlanningConfig {
	return PlanningConfig{
		AISummaryEnabled:    true,
		SimpleTaskThreshold: 3,
		ConfirmPlan:         false,
		MaxPlanSteps:        20,
	}
}

// NewTaskPlanner 创建一个新的任务规划器。
func NewTaskPlanner(llm LLMProvider, cfg PlanningConfig) *TaskPlanner {
	if cfg.MaxPlanSteps <= 0 {
		cfg.MaxPlanSteps = 20
	}
	return &TaskPlanner{
		llm: llm,
		cfg: cfg,
	}
}

// Plan 根据用户请求生成执行计划。
func (p *TaskPlanner) Plan(ctx context.Context, req *PlanningRequest) (*Plan, error) {
	log := logger.L()

	log.Debug("Planning task",
		zap.String("userMessage", truncateString(req.UserMessage, 100)),
		zap.Int("summaryLen", len(req.CapabilitySummary)),
	)

	prompt := p.buildPlanningPrompt(req)

	chatReq := &ChatRequest{
		Messages: []Message{
			{Role: "system", Content: p.getSystemPrompt()},
			{Role: "user", Content: prompt},
		},
		MaxTokens: 3000,
		Temperature: 0.3, // 降低温度以获得更稳定的输出
	}

	resp, err := p.llm.Chat(ctx, chatReq)
	if err != nil {
		return nil, fmt.Errorf("llm chat: %w", err)
	}

	log.Debug("LLM planning response",
		zap.Int("contentLen", len(resp.Content)),
		zap.String("contentPreview", truncateString(resp.Content, 500)),
	)

	// 解析 JSON 响应
	plan, err := p.parsePlanResponse(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("parse plan response: %w", err)
	}

	// 拓扑排序，确保依赖顺序
	plan.Tasks = p.sortTasksByDependencies(plan.Tasks)

	// 验证计划
	if err := p.validatePlan(plan); err != nil {
		return nil, fmt.Errorf("invalid plan: %w", err)
	}

	log.Info("Plan generated",
		zap.Int("steps", len(plan.Tasks)),
		zap.String("summary", plan.Summary),
	)

	return plan, nil
}

// getSystemPrompt 返回规划器的系统提示词。
func (p *TaskPlanner) getSystemPrompt() string {
	return `你是一个专业的任务规划器。你的职责是将用户请求分解为结构化的执行计划。

规则：
1. 分析用户请求，识别需要执行的具体步骤
2. 为每个步骤选择最合适的能力（工具/技能/MCP）
3. 正确处理任务之间的依赖关系
4. 返回有效的 JSON 格式

输出格式（必须严格遵守）：
{
  "summary": "整体计划的简要描述（1-2 句话）",
  "reasoning": "规划推理过程（为什么这样分解任务）",
  "tasks": [
    {
      "id": "task_1",
      "description": "具体的任务描述",
      "capability": "tool:web_search 或 skill:docx 或 mcp:...",
      "input": {"参数名": "参数值"},
      "depends_on": [],
      "expected_output": "预期得到什么结果"
    }
  ]
}

注意事项：
- id 格式：task_1, task_2, ...（必须唯一）
- depends_on: 依赖任务的 id 列表，如 ["task_1", "task_2"]
- capability 格式：type:name（tool:, skill:, mcp:）
- 输入参数必须与能力定义匹配
- 简单任务（单个工具调用）可以简化计划`
}

// buildPlanningPrompt 构建规划提示词。
func (p *TaskPlanner) buildPlanningPrompt(req *PlanningRequest) string {
	var sb strings.Builder

	sb.WriteString("## 用户请求\n\n")
	sb.WriteString(req.UserMessage)
	sb.WriteString("\n\n")

	if req.Context != "" {
		sb.WriteString("## 对话上下文\n\n")
		sb.WriteString(req.Context)
		sb.WriteString("\n\n")
	}

	sb.WriteString("## 可用能力\n\n")
	sb.WriteString(req.CapabilitySummary)
	sb.WriteString("\n\n")

	sb.WriteString("请分析用户请求，生成执行计划。返回 JSON 格式。")

	return sb.String()
}

// parsePlanResponse 解析 LLM 返回的计划响应。
func (p *TaskPlanner) parsePlanResponse(content string) (*Plan, error) {
	// 尝试直接解析 JSON
	var plan Plan
	if err := json.Unmarshal([]byte(content), &plan); err == nil {
		return &plan, nil
	}

	// 尝试提取 JSON 代码块
	if jsonStart := strings.Index(content, "{"); jsonStart >= 0 {
		if jsonEnd := strings.LastIndex(content, "}"); jsonEnd > jsonStart {
			jsonStr := content[jsonStart : jsonEnd+1]
			if err := json.Unmarshal([]byte(jsonStr), &plan); err == nil {
				return &plan, nil
			}
		}
	}

	// 尝试提取 ```json 代码块
	markdownStart := strings.Index(content, "```json")
	if markdownStart >= 0 {
		markdownStart += 7
		markdownEnd := strings.Index(content[markdownStart:], "```")
		if markdownEnd > 0 {
			jsonStr := strings.TrimSpace(content[markdownStart : markdownStart+markdownEnd])
			if err := json.Unmarshal([]byte(jsonStr), &plan); err == nil {
				return &plan, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to parse plan from response")
}

// sortTasksByDependencies 按依赖关系对任务进行拓扑排序。
func (p *TaskPlanner) sortTasksByDependencies(tasks []Task) []Task {
	// 构建依赖图和入度表
	taskMap := make(map[string]*Task)
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)

	// 初始化
	for i := range tasks {
		taskMap[tasks[i].ID] = &tasks[i]
		inDegree[tasks[i].ID] = 0
		adjList[tasks[i].ID] = []string{}
	}

	// 填充依赖关系
	for _, task := range tasks {
		for _, dep := range task.DependsOn {
			adjList[dep] = append(adjList[dep], task.ID)
			inDegree[task.ID]++
		}
	}

	// 拓扑排序
	var result []Task
	queue := make([]string, 0)

	// 找到所有入度为 0 的节点
	for id, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, id)
		}
	}

	// 处理队列
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if task, ok := taskMap[current]; ok {
			result = append(result, *task)
		}

		// 减少依赖此任务的其他任务的入度
		for _, neighbor := range adjList[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}

	// 检查循环依赖
	if len(result) != len(tasks) {
		logger.L().Warn("Circular dependency detected in plan, using original order")
		return tasks
	}

	return result
}

// validatePlan 验证计划的有效性。
func (p *TaskPlanner) validatePlan(plan *Plan) error {
	if len(plan.Tasks) == 0 {
		return fmt.Errorf("plan must have at least one task")
	}

	if len(plan.Tasks) > p.cfg.MaxPlanSteps {
		return fmt.Errorf("plan exceeds maximum steps: %d > %d", len(plan.Tasks), p.cfg.MaxPlanSteps)
	}

	// 检查任务 ID 唯一性
	taskIDs := make(map[string]bool)
	for _, task := range plan.Tasks {
		if task.ID == "" {
			return fmt.Errorf("task has empty ID")
		}
		if taskIDs[task.ID] {
			return fmt.Errorf("duplicate task ID: %s", task.ID)
		}
		taskIDs[task.ID] = true
	}

	// 检查依赖有效性
	for _, task := range plan.Tasks {
		for _, dep := range task.DependsOn {
			if !taskIDs[dep] {
				return fmt.Errorf("task %s depends on non-existent task: %s", task.ID, dep)
			}
			// 检查循环依赖
			if p.hasCircularDependency(task.ID, dep, plan.Tasks) {
				return fmt.Errorf("circular dependency detected involving task %s", task.ID)
			}
		}
	}

	return nil
}

// hasCircularDependency 检查是否存在循环依赖。
func (p *TaskPlanner) hasCircularDependency(from, to string, tasks []Task) bool {
	taskMap := make(map[string]Task)
	for _, t := range tasks {
		taskMap[t.ID] = t
	}

	visited := make(map[string]bool)
	return p.checkCycle(to, from, taskMap, visited)
}

// checkCycle 递归检查循环。
func (p *TaskPlanner) checkCycle(current, target string, taskMap map[string]Task, visited map[string]bool) bool {
	if current == target {
		return true
	}
	if visited[current] {
		return false
	}
	visited[current] = true

	task := taskMap[current]
	for _, dep := range task.DependsOn {
		if p.checkCycle(dep, target, taskMap, visited) {
			return true
		}
	}

	return false
}

// truncateString 截断字符串。
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// Ensure TaskPlanner implements Planner.
var _ Planner = (*TaskPlanner)(nil)
