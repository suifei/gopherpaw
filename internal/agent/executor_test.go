// Package agent provides the core Agent runtime, ReAct loop, and domain types.
package agent

import (
	"context"
	"testing"
	"time"
)

// mockExecutorLLM 是用于执行器测试的 LLM 模拟实现。
type mockExecutorLLM struct {
	summarizeResponse string
}

func (m *mockExecutorLLM) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// 根据请求内容返回不同的响应
	content := req.Messages[len(req.Messages)-1].Content

	if len(content) > 100 && containsSubstring(content, "精简") {
		// 精简请求
		return &ChatResponse{
			Content: m.summarizeResponse,
		}, nil
	}

	if len(content) > 100 && containsSubstring(content, "最终回答") {
		// 生成最终答案
		return &ChatResponse{
			Content: "任务已全部完成。根据执行结果，所有步骤均成功执行。",
		}, nil
	}

	return &ChatResponse{
		Content: "LLM 响应",
	}, nil
}

func (m *mockExecutorLLM) ChatStream(ctx context.Context, req *ChatRequest) (ChatStream, error) {
	return nil, nil
}

func (m *mockExecutorLLM) Name() string {
	return "mock-executor"
}

// mockExecutorTool 是用于执行器测试的工具模拟实现。
type mockExecutorTool struct {
	name        string
	executeFunc func(ctx context.Context, arguments string) (string, error)
}

func (m *mockExecutorTool) Name() string {
	return m.name
}

func (m *mockExecutorTool) Description() string {
	return "Mock tool for testing"
}

func (m *mockExecutorTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"input": map[string]any{
				"type": "string",
			},
		},
	}
}

func (m *mockExecutorTool) Execute(ctx context.Context, arguments string) (string, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, arguments)
	}
	return "mock result", nil
}

func TestNewExecutor(t *testing.T) {
	agent := &ReactAgent{}
	llm := &mockExecutorLLM{}
	tools := []Tool{&mockExecutorTool{name: "test"}}

	executor := NewExecutor(agent, llm, tools, DefaultExecutionConfig())

	if executor == nil {
		t.Fatal("NewExecutor returned nil")
	}

	if len(executor.tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(executor.tools))
	}
}

func TestExecutionConfigDefaults(t *testing.T) {
	cfg := DefaultExecutionConfig()

	if cfg.MaxRetries != 1 {
		t.Errorf("Expected MaxRetries to be 1, got %d", cfg.MaxRetries)
	}

	if cfg.SummarizeResults != true {
		t.Errorf("Expected SummarizeResults to be true, got %v", cfg.SummarizeResults)
	}

	if cfg.TaskTimeout != 5*time.Minute {
		t.Errorf("Expected TaskTimeout to be 5 minutes, got %v", cfg.TaskTimeout)
	}
}

func TestCheckDependencies(t *testing.T) {
	llm := &mockExecutorLLM{}
	executor := NewExecutor(nil, llm, []Tool{}, DefaultExecutionConfig())

	// 测试没有依赖的任务
	task1 := Task{ID: "task_1", DependsOn: []string{}}
	if !executor.checkDependencies(task1) {
		t.Error("Task with no dependencies should pass check")
	}

	// 测试有依赖但依赖未满足的任务
	task2 := Task{ID: "task_2", DependsOn: []string{"task_1"}}
	if executor.checkDependencies(task2) {
		t.Error("Task with unsatisfied dependencies should fail check")
	}

	// 添加依赖任务的结果
	executor.saveResult(&TaskResult{
		TaskID:    "task_1",
		Status:    "success",
		Timestamp: time.Now().Unix(),
	})

	// 现在应该通过
	if !executor.checkDependencies(task2) {
		t.Error("Task with satisfied dependencies should pass check")
	}

	// 测试依赖失败的情况 - task_3 依赖于 task_2（失败）
	task3 := Task{ID: "task_3", DependsOn: []string{"task_2"}}
	if executor.checkDependencies(task3) {
		t.Error("Task with failed dependencies should fail check")
	}
}

func TestBuildTaskInput(t *testing.T) {
	llm := &mockExecutorLLM{}
	executor := NewExecutor(nil, llm, []Tool{}, DefaultExecutionConfig())

	// 添加一个依赖任务的摘要
	executor.saveResult(&TaskResult{
		TaskID:    "task_1",
		Summary:   "搜索结果：MacBook Air 价格约 8000 元",
		Status:    "success",
		Timestamp: time.Now().Unix(),
	})

	task := Task{
		ID:      "task_2",
		Input:   map[string]any{"query": "test"},
		DependsOn: []string{"task_1"},
	}

	input := executor.buildTaskInput(task)

	// 检查原始输入是否保留
	if input["query"] != "test" {
		t.Error("Original input should be preserved")
	}

	// 检查依赖摘要是否添加
	depSummaries, ok := input["_dependency_summaries"].(string)
	if !ok {
		t.Error("Dependency summaries should be added")
	}

	if !containsSubstring(depSummaries, "task_1") {
		t.Error("Dependency summary should contain task_1")
	}
}

func TestExecuteToolTask(t *testing.T) {
	llm := &mockExecutorLLM{}
	tool := &mockExecutorTool{
		name: "test_tool",
		executeFunc: func(ctx context.Context, arguments string) (string, error) {
			return "execution result", nil
		},
	}
	executor := NewExecutor(nil, llm, []Tool{tool}, DefaultExecutionConfig())

	ctx := context.Background()
	input := map[string]any{"param1": "value1"}

	result, err := executor.executeToolTask(ctx, "test_tool", input)
	if err != nil {
		t.Fatalf("executeToolTask failed: %v", err)
	}

	if result != "execution result" {
		t.Errorf("Expected 'execution result', got '%s'", result)
	}
}

func TestExecuteToolTaskNotFound(t *testing.T) {
	llm := &mockExecutorLLM{}
	executor := NewExecutor(nil, llm, []Tool{}, DefaultExecutionConfig())

	ctx := context.Background()
	input := map[string]any{"param1": "value1"}

	_, err := executor.executeToolTask(ctx, "nonexistent", input)
	if err == nil {
		t.Error("Expected error for nonexistent tool")
	}
}

func TestSaveAndGetResult(t *testing.T) {
	llm := &mockExecutorLLM{}
	executor := NewExecutor(nil, llm, []Tool{}, DefaultExecutionConfig())

	result := &TaskResult{
		TaskID:    "task_1",
		Status:    "success",
		Output:    "test output",
		Summary:   "test summary",
		Timestamp: time.Now().Unix(),
	}

	executor.saveResult(result)

	// 获取结果
	retrieved, ok := executor.GetResult("task_1")
	if !ok {
		t.Fatal("Result not found")
	}

	if retrieved.Status != "success" {
		t.Errorf("Expected status 'success', got '%s'", retrieved.Status)
	}

	if retrieved.Output != "test output" {
		t.Errorf("Expected output 'test output', got '%s'", retrieved.Output)
	}
}

func TestGetAllResults(t *testing.T) {
	llm := &mockExecutorLLM{}
	executor := NewExecutor(nil, llm, []Tool{}, DefaultExecutionConfig())

	// 保存多个结果
	results := []*TaskResult{
		{TaskID: "task_1", Status: "success", Timestamp: time.Now().Unix()},
		{TaskID: "task_2", Status: "success", Timestamp: time.Now().Unix()},
		{TaskID: "task_3", Status: "failed", Timestamp: time.Now().Unix()},
	}

	for _, r := range results {
		executor.saveResult(r)
	}

	allResults := executor.GetAllResults()

	if len(allResults) != 3 {
		t.Errorf("Expected 3 results, got %d", len(allResults))
	}

	// 验证结果是副本而不是原始 map 的引用
	allResults["task_4"] = &TaskResult{TaskID: "task_4"}

	_, ok := executor.GetResult("task_4")
	if ok {
		t.Error("GetAllResults should return a copy, not the internal map")
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{
			name:     "short string",
			input:    "hello",
			maxLen:   10,
			expected: "hello",
		},
		{
			name:     "exact length",
			input:    "hello",
			maxLen:   5,
			expected: "hello",
		},
		{
			name:     "truncate needed",
			input:    "hello world",
			maxLen:   5,
			expected: "hello" + "...",
		},
		{
			name:     "empty string",
			input:    "",
			maxLen:   10,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := truncate(tt.input, tt.maxLen)
			if result != tt.expected {
				t.Errorf("truncate() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestIsCriticalTask(t *testing.T) {
	llm := &mockExecutorLLM{}
	executor := NewExecutor(nil, llm, []Tool{}, DefaultExecutionConfig())

	task1 := Task{ID: "task_1", DependsOn: []string{}}
	if !executor.isCriticalTask(task1) {
		t.Error("Task with no dependencies should be critical")
	}

	task2 := Task{ID: "task_2", DependsOn: []string{"task_1"}}
	if executor.isCriticalTask(task2) {
		t.Error("Task with dependencies should not be critical")
	}
}

func TestExecuteSimplePlan(t *testing.T) {
	llm := &mockExecutorLLM{summarizeResponse: `{"summary": "任务完成", "key_facts": ["成功"]}`}
	tool := &mockExecutorTool{
		name: "test_tool",
		executeFunc: func(ctx context.Context, arguments string) (string, error) {
			return "tool execution result", nil
		},
	}

	executor := NewExecutor(nil, llm, []Tool{tool}, DefaultExecutionConfig())

	plan := &Plan{
		Summary: "简单测试计划",
		Tasks: []Task{
			{
				ID:          "task_1",
				Description: "执行测试工具",
				Capability:  "tool:test_tool",
				Input:       map[string]any{"test": "value"},
				DependsOn:   []string{},
			},
		},
	}

	ctx := context.Background()
	result, err := executor.Execute(ctx, plan)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == "" {
		t.Error("Result should not be empty")
	}

	// 验证任务结果已保存
	taskResult, ok := executor.GetResult("task_1")
	if !ok {
		t.Fatal("Task result should be saved")
	}

	if taskResult.Status != "success" {
		t.Errorf("Expected task status 'success', got '%s'", taskResult.Status)
	}
}

func TestExecutePlanWithDependencies(t *testing.T) {
	llm := &mockExecutorLLM{summarizeResponse: `{"summary": "第二步完成"}`}
	tool := &mockExecutorTool{
		name: "test_tool",
		executeFunc: func(ctx context.Context, arguments string) (string, error) {
			return "result", nil
		},
	}

	executor := NewExecutor(nil, llm, []Tool{tool}, DefaultExecutionConfig())
	executor.cfg.SummarizeResults = false // 禁用精简以加快测试

	plan := &Plan{
		Summary: "依赖测试计划",
		Tasks: []Task{
			{
				ID:          "task_1",
				Description: "第一步",
				Capability:  "tool:test_tool",
				Input:       map[string]any{"step": 1},
				DependsOn:   []string{},
			},
			{
				ID:          "task_2",
				Description: "第二步",
				Capability:  "tool:test_tool",
				Input:       map[string]any{"step": 2},
				DependsOn:   []string{"task_1"},
			},
		},
	}

	ctx := context.Background()
	result, err := executor.Execute(ctx, plan)

	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result == "" {
		t.Error("Result should not be empty")
	}

	// 验证两个任务都成功执行
	task1Result, _ := executor.GetResult("task_1")
	task2Result, _ := executor.GetResult("task_2")

	if task1Result.Status != "success" {
		t.Errorf("Task 1 should succeed, got status: %s", task1Result.Status)
	}

	if task2Result.Status != "success" {
		t.Errorf("Task 2 should succeed, got status: %s", task2Result.Status)
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && findSubstrExec(s, substr))
}

func findSubstrExec(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
