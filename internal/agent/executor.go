// Package agent provides the core Agent runtime, ReAct loop, and domain types.
package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

// Executor 任务执行器，负责按计划执行任务并管理结果。
type Executor struct {
	mu         sync.RWMutex
	agent      *ReactAgent
	llm        LLMProvider
	tools      []Tool
	toolMap    map[string]Tool
	results    map[string]*TaskResult
	skillsMgr  any // skills.Manager 接口（避免循环依赖）
	mcpMgr     any // mcp.Manager 接口
	cfg        ExecutionConfig
}

// ExecutionConfig 执行器配置。
type ExecutionConfig struct {
	MaxRetries         int           // 最大重试次数
	SummarizeResults   bool          // 是否对结果进行精简
	SummaryMaxTokens   int           // 精简结果的最大 token 数
	TaskTimeout        time.Duration // 单个任务超时时间
	EnableParallel     bool          // 是否启用并行执行（无依赖任务）
}

// DefaultExecutionConfig 返回默认的执行器配置。
func DefaultExecutionConfig() ExecutionConfig {
	return ExecutionConfig{
		MaxRetries:       1,
		SummarizeResults: true,
		SummaryMaxTokens: 1000,
		TaskTimeout:      5 * time.Minute,
		EnableParallel:   false,
	}
}

// NewExecutor 创建一个新的任务执行器。
func NewExecutor(agent *ReactAgent, llm LLMProvider, tools []Tool, cfg ExecutionConfig) *Executor {
	toolMap := make(map[string]Tool)
	for _, t := range tools {
		toolMap[t.Name()] = t
	}

	return &Executor{
		agent:   agent,
		llm:     llm,
		tools:   tools,
		toolMap: toolMap,
		results: make(map[string]*TaskResult),
		cfg:     cfg,
	}
}

// SetSkillsManager 设置技能管理器（用于执行技能任务）。
func (e *Executor) SetSkillsManager(mgr any) {
	e.skillsMgr = mgr
}

// SetMCPManager 设置 MCP 管理器（用于执行 MCP 任务）。
func (e *Executor) SetMCPManager(mgr any) {
	e.mcpMgr = mgr
}

// Execute 执行计划并返回最终结果。
func (e *Executor) Execute(ctx context.Context, plan *Plan) (string, error) {
	log := logger.L()

	log.Info("Starting plan execution",
		zap.Int("totalTasks", len(plan.Tasks)),
		zap.String("planSummary", plan.Summary),
	)

	// 清空之前的结果
	e.mu.Lock()
	e.results = make(map[string]*TaskResult)
	e.mu.Unlock()

	// 执行每个任务
	for _, task := range plan.Tasks {
		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("context cancelled: %w", err)
		}

		// 检查依赖是否完成
		if !e.checkDependencies(task) {
			log.Warn("Task dependencies not satisfied, skipping",
				zap.String("taskID", task.ID),
				zap.String("dependsOn", strings.Join(task.DependsOn, ",")),
			)
			e.saveResult(&TaskResult{
				TaskID:    task.ID,
				Status:    "skipped",
				Timestamp: time.Now().Unix(),
			})
			continue
		}

		// 构建任务输入（包含依赖任务的摘要）
		input := e.buildTaskInput(task)

		// 执行任务
		result := e.executeTask(ctx, task, input)

		// 精简结果
		if e.cfg.SummarizeResults && result.Status == "success" {
			result.Summary = e.summarizeResult(ctx, task, result.Output)
		} else {
			result.Summary = truncate(result.Output, 500)
		}

		// 保存结果
		e.saveResult(result)

		log.Info("Task completed",
			zap.String("taskID", task.ID),
			zap.String("status", result.Status),
			zap.Int("outputLen", len(result.Output)),
			zap.Int("summaryLen", len(result.Summary)),
		)

		// 如果任务失败且是关键任务，停止执行
		if result.Status == "failed" && e.isCriticalTask(task) {
			return "", fmt.Errorf("critical task %s failed: %s", task.ID, result.Error)
		}
	}

	// 生成最终答案
	finalAnswer, err := e.generateFinalAnswer(ctx, plan)
	if err != nil {
		return "", fmt.Errorf("generate final answer: %w", err)
	}

	log.Info("Plan execution completed",
		zap.Int("completedTasks", len(e.results)),
		zap.Int("finalAnswerLen", len(finalAnswer)),
	)

	return finalAnswer, nil
}

// checkDependencies 检查任务的依赖是否已完成。
func (e *Executor) checkDependencies(task Task) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, depID := range task.DependsOn {
		result, ok := e.results[depID]
		if !ok || result.Status != "success" {
			return false
		}
	}
	return true
}

// buildTaskInput 构建任务输入，包含依赖任务的摘要结果。
func (e *Executor) buildTaskInput(task Task) map[string]any {
	input := make(map[string]any)

	// 复制任务的基础输入
	for k, v := range task.Input {
		input[k] = v
	}

	// 添加依赖任务的摘要
	e.mu.RLock()
	defer e.mu.RUnlock()

	if len(task.DependsOn) > 0 {
		var depSummaries []string
		for _, depID := range task.DependsOn {
			if result, ok := e.results[depID]; ok {
				depSummaries = append(depSummaries,
					fmt.Sprintf("[%s]: %s", depID, result.Summary))
			}
		}
		if len(depSummaries) > 0 {
			input["_dependency_summaries"] = strings.Join(depSummaries, "\n")
		}
	}

	return input
}

// executeTask 执行单个任务。
func (e *Executor) executeTask(ctx context.Context, task Task, input map[string]any) *TaskResult {
	log := logger.L()

	startTime := time.Now()
	result := &TaskResult{
		TaskID:    task.ID,
		Status:    "success",
		Timestamp: startTime.Unix(),
		Metadata:  make(map[string]any),
	}

	// 设置超时
	taskCtx := ctx
	if e.cfg.TaskTimeout > 0 {
		var cancel context.CancelFunc
		taskCtx, cancel = context.WithTimeout(ctx, e.cfg.TaskTimeout)
		defer cancel()
	}

	// 根据能力类型执行
	switch {
	case strings.HasPrefix(task.Capability, "tool:"):
		toolName := strings.TrimPrefix(task.Capability, "tool:")
		output, err := e.executeToolTask(taskCtx, toolName, input)
		if err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			result.Output = fmt.Sprintf("Error: %v", err)
		} else {
			result.Output = output
		}

	case strings.HasPrefix(task.Capability, "mcp:"):
		toolName := strings.TrimPrefix(task.Capability, "mcp:")
		output, err := e.executeMCPToolTask(taskCtx, toolName, input)
		if err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			result.Output = fmt.Sprintf("Error: %v", err)
		} else {
			result.Output = output
		}

	case strings.HasPrefix(task.Capability, "skill:"):
		skillName := strings.TrimPrefix(task.Capability, "skill:")
		output, err := e.executeSkillTask(taskCtx, skillName, input, task.Description)
		if err != nil {
			result.Status = "failed"
			result.Error = err.Error()
			result.Output = fmt.Sprintf("Error: %v", err)
		} else {
			result.Output = output
		}

	default:
		result.Status = "failed"
		result.Error = fmt.Sprintf("unknown capability type: %s", task.Capability)
		result.Output = result.Error
	}

	log.Debug("Task execution finished",
		zap.String("taskID", task.ID),
		zap.String("capability", task.Capability),
		zap.String("status", result.Status),
		zap.Duration("duration", time.Since(startTime)),
	)

	return result
}

// executeToolTask 执行工具任务。
func (e *Executor) executeToolTask(ctx context.Context, toolName string, input map[string]any) (string, error) {
	tool, ok := e.toolMap[toolName]
	if !ok {
		return "", fmt.Errorf("tool not found: %s", toolName)
	}

	// 将输入转换为 JSON 字符串
	argsJSON, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("marshal input: %w", err)
	}

	// 移除内部字段
	cleanInput := make(map[string]any)
	for k, v := range input {
		if !strings.HasPrefix(k, "_") {
			cleanInput[k] = v
		}
	}
	argsJSON, _ = json.Marshal(cleanInput)

	// 执行工具
	return tool.Execute(ctx, string(argsJSON))
}

// executeMCPToolTask 执行 MCP 工具任务。
func (e *Executor) executeMCPToolTask(ctx context.Context, toolName string, input map[string]any) (string, error) {
	// MCP 工具通过 agent 执行
	// 需要通过 agent 来访问 MCP 管理器
	if e.mcpMgr == nil {
		return "", fmt.Errorf("MCP manager not available")
	}

	// 尝试通过 agent 执行
	tool, ok := e.toolMap[toolName]
	if !ok {
		return "", fmt.Errorf("MCP tool not found: %s", toolName)
	}

	argsJSON, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("marshal input: %w", err)
	}

	return tool.Execute(ctx, string(argsJSON))
}

// executeSkillTask 执行技能任务。
func (e *Executor) executeSkillTask(ctx context.Context, skillName string, input map[string]any, taskDesc string) (string, error) {
	if e.skillsMgr == nil {
		return "", fmt.Errorf("skills manager not available")
	}

	// 技能执行通过 LLM 调用，而不是直接工具调用
	// 构建技能执行提示
	prompt := fmt.Sprintf("请使用 %s 技能完成以下任务：\n\n%s\n\n可用输入：%+v",
		skillName, taskDesc, input)

	req := &ChatRequest{
		Messages: []Message{
			{Role: "system", Content: "你是一个技能执行助手，使用指定的技能完成任务。"},
			{Role: "user", Content: prompt},
		},
		MaxTokens: 2000,
	}

	resp, err := e.llm.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("LLM chat for skill: %w", err)
	}

	return resp.Content, nil
}

// summarizeResult 精简任务结果。
func (e *Executor) summarizeResult(ctx context.Context, task Task, output string) string {
	if len(output) <= 500 {
		return output
	}

	log := logger.L()

	prompt := fmt.Sprintf(`请精简以下任务执行结果，只保留关键信息供后续任务使用：

任务：%s
结果（前 2000 字符）：
%s

请返回 JSON 格式的精简摘要：
{
  "summary": "关键信息摘要（100 字以内）",
  "key_facts": ["事实1", "事实2", ...],
  "next_step_hint": "对下个任务的提示（可选）"
}`,
		task.Description,
		truncate(output, 2000),
	)

	req := &ChatRequest{
		Messages: []Message{
			{Role: "system", Content: "你是结果精简助手，擅长提炼关键信息。"},
			{Role: "user", Content: prompt},
		},
		MaxTokens: e.cfg.SummaryMaxTokens,
	}

	resp, err := e.llm.Chat(ctx, req)
	if err != nil {
		log.Warn("Failed to summarize result, using truncation", zap.Error(err))
		return truncate(output, 300)
	}

	// 尝试解析 JSON，如果失败则直接使用响应
	var summary struct {
		Summary       string   `json:"summary"`
		KeyFacts      []string `json:"key_facts"`
		NextStepHint  string   `json:"next_step_hint"`
	}

	if err := json.Unmarshal([]byte(resp.Content), &summary); err == nil {
		if summary.Summary != "" {
			result := summary.Summary
			if len(summary.KeyFacts) > 0 {
				result += "\n关键事实：" + strings.Join(summary.KeyFacts, "；")
			}
			return result
		}
	}

	return truncate(resp.Content, 300)
}

// generateFinalAnswer 生成最终答案。
func (e *Executor) generateFinalAnswer(ctx context.Context, plan *Plan) (string, error) {
	log := logger.L()

	e.mu.RLock()
	allResults := make([]*TaskResult, 0, len(e.results))
	for _, result := range e.results {
		allResults = append(allResults, result)
	}
	e.mu.RUnlock()

	// 构建最终答案提示
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## 执行计划：\n%s\n\n", plan.Summary))
	sb.WriteString("## 任务执行结果：\n\n")

	successCount := 0
	failedCount := 0
	skippedCount := 0

	for _, result := range allResults {
		switch result.Status {
		case "success":
			successCount++
			sb.WriteString(fmt.Sprintf("✅ %s: %s\n", result.TaskID, result.Summary))
		case "failed":
			failedCount++
			sb.WriteString(fmt.Sprintf("❌ %s: %s\n", result.TaskID, result.Error))
		case "skipped":
			skippedCount++
			sb.WriteString(fmt.Sprintf("⏭️  %s: 已跳过\n", result.TaskID))
		}
	}

	sb.WriteString(fmt.Sprintf("\n总计：%d 成功，%d 失败，%d 跳过\n",
		successCount, failedCount, skippedCount))

	// 调用 LLM 生成最终答案
	prompt := fmt.Sprintf(`基于以下任务执行结果，生成给用户的最终回答。

%s

请生成一个简洁、专业的最终回答，总结所有任务的执行结果和最终结论。`, sb.String())

	req := &ChatRequest{
		Messages: []Message{
			{Role: "system", Content: "你是任务总结助手，负责将执行结果转化为用户友好的回答。"},
			{Role: "user", Content: prompt},
		},
		MaxTokens: 2000,
	}

	resp, err := e.llm.Chat(ctx, req)
	if err != nil {
		log.Warn("Failed to generate final answer via LLM, using raw summary", zap.Error(err))
		return sb.String(), nil
	}

	return resp.Content, nil
}

// saveResult 保存任务结果。
func (e *Executor) saveResult(result *TaskResult) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.results[result.TaskID] = result
}

// GetResult 获取特定任务的执行结果。
func (e *Executor) GetResult(taskID string) (*TaskResult, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result, ok := e.results[taskID]
	return result, ok
}

// GetAllResults 获取所有任务结果。
func (e *Executor) GetAllResults() map[string]*TaskResult {
	e.mu.RLock()
	defer e.mu.RUnlock()
	results := make(map[string]*TaskResult, len(e.results))
	for k, v := range e.results {
		results[k] = v
	}
	return results
}

// isCriticalTask 判断任务是否是关键任务。
func (e *Executor) isCriticalTask(task Task) bool {
	// 没有依赖的任务通常是关键任务
	return len(task.DependsOn) == 0
}

// Ensure Executor implements TaskExecutor.
var _ TaskExecutor = (*Executor)(nil)
