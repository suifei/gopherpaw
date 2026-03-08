// Package agent provides the core Agent runtime, ReAct loop, and domain types.
package agent

import (
	"context"
	"encoding/json"
	"testing"
)

// mockPlanningLLM 是一个专门用于规划测试的 LLM 模拟实现。
type mockPlanningLLM struct {
	response string
}

func (m *mockPlanningLLM) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	// 返回一个预设的计划响应
	response := `{
  "summary": "搜索 MacBook 价格并生成报告",
  "reasoning": "用户需要获取最新价格信息并生成文档",
  "tasks": [
    {
      "id": "task_1",
      "description": "搜索 MacBook Air 2024 价格",
      "capability": "tool:web_search",
      "input": {"query": "MacBook Air 2024 价格"},
      "depends_on": [],
      "expected_output": "搜索结果包含价格信息"
    },
    {
      "id": "task_2",
      "description": "生成价格对比报告",
      "capability": "skill:docx",
      "input": {"content": "基于搜索结果生成报告"},
      "depends_on": ["task_1"],
      "expected_output": "Word 文档"
    }
  ]
}`

	return &ChatResponse{
		Content: response,
		Usage: Usage{
			PromptTokens:     200,
			CompletionTokens: 150,
			TotalTokens:      350,
		},
	}, nil
}

func (m *mockPlanningLLM) ChatStream(ctx context.Context, req *ChatRequest) (ChatStream, error) {
	return nil, nil
}

func (m *mockPlanningLLM) Name() string {
	return "mock-planning"
}

func TestNewTaskPlanner(t *testing.T) {
	llm := &mockPlanningLLM{}
	cfg := DefaultPlanningConfig()

	planner := NewTaskPlanner(llm, cfg)

	if planner == nil {
		t.Fatal("NewTaskPlanner returned nil")
	}

	if planner.llm != llm {
		t.Error("LLM not set correctly")
	}
}

func TestPlan(t *testing.T) {
	llm := &mockPlanningLLM{}
	cfg := DefaultPlanningConfig()
	planner := NewTaskPlanner(llm, cfg)

	ctx := context.Background()
	req := &PlanningRequest{
		UserMessage:       "搜索 MacBook Air 2024 价格并生成 Word 报告",
		CapabilitySummary: "可用工具：web_search, docx",
		Context:           "用户之前询问过苹果产品信息",
	}

	plan, err := planner.Plan(ctx, req)

	if err != nil {
		t.Fatalf("Plan failed: %v", err)
	}

	if plan == nil {
		t.Fatal("Plan is nil")
	}

	// 检查计划摘要
	if plan.Summary == "" {
		t.Error("Plan summary is empty")
	}

	// 检查任务数量
	if len(plan.Tasks) != 2 {
		t.Errorf("Expected 2 tasks, got %d", len(plan.Tasks))
	}

	// 检查任务依赖关系
	if len(plan.Tasks[1].DependsOn) != 1 {
		t.Errorf("Expected task_2 to depend on 1 task, got %d", len(plan.Tasks[1].DependsOn))
	}

	if plan.Tasks[1].DependsOn[0] != "task_1" {
		t.Errorf("Expected task_2 to depend on task_1, got %s", plan.Tasks[1].DependsOn[0])
	}
}

func TestParsePlanResponse(t *testing.T) {
	planner := NewTaskPlanner(nil, DefaultPlanningConfig())

	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{
			name: "valid JSON",
			content: `{
				"summary": "test",
				"tasks": [{"id": "task_1", "description": "test", "capability": "tool:test", "depends_on": []}]
			}`,
			wantErr: false,
		},
		{
			name:    "JSON with markdown",
			content: "```json\n{\"summary\": \"test\", \"tasks\": []}\n```",
			wantErr: false,
		},
		{
			name:    "embedded JSON",
			content: "Some text before {\"summary\": \"test\", \"tasks\": []} some text after",
			wantErr: false,
		},
		{
			name:    "invalid JSON",
			content: "not a json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plan, err := planner.parsePlanResponse(tt.content)
			if (err != nil) != tt.wantErr {
				t.Errorf("parsePlanResponse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && plan == nil {
				t.Error("Expected non-nil plan")
			}
		})
	}
}

func TestSortTasksByDependencies(t *testing.T) {
	planner := NewTaskPlanner(nil, DefaultPlanningConfig())

	tasks := []Task{
		{ID: "task_3", Description: "Third", DependsOn: []string{"task_1", "task_2"}},
		{ID: "task_1", Description: "First", DependsOn: []string{}},
		{ID: "task_2", Description: "Second", DependsOn: []string{"task_1"}},
	}

	sorted := planner.sortTasksByDependencies(tasks)

	// 验证排序结果：task_1 应该在 task_2 之前，task_2 在 task_3 之前
	orderMap := make(map[string]int)
	for i, task := range sorted {
		orderMap[task.ID] = i
	}

	if orderMap["task_1"] >= orderMap["task_2"] {
		t.Error("task_1 should come before task_2")
	}

	if orderMap["task_2"] >= orderMap["task_3"] {
		t.Error("task_2 should come before task_3")
	}
}

func TestValidatePlan(t *testing.T) {
	planner := NewTaskPlanner(nil, DefaultPlanningConfig())

	tests := []struct {
		name    string
		plan    *Plan
		wantErr bool
	}{
		{
			name: "valid plan",
			plan: &Plan{
				Tasks: []Task{
					{ID: "task_1", Description: "test", DependsOn: []string{}},
				},
			},
			wantErr: false,
		},
		{
			name:    "empty plan",
			plan:    &Plan{Tasks: []Task{}},
			wantErr: true,
		},
		{
			name: "duplicate task IDs",
			plan: &Plan{
				Tasks: []Task{
					{ID: "task_1", Description: "test1", DependsOn: []string{}},
					{ID: "task_1", Description: "test2", DependsOn: []string{}},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid dependency",
			plan: &Plan{
				Tasks: []Task{
					{ID: "task_1", Description: "test", DependsOn: []string{"non_existent"}},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := planner.validatePlan(tt.plan)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePlan() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCircularDependencyDetection(t *testing.T) {
	planner := NewTaskPlanner(nil, DefaultPlanningConfig())

	tasks := []Task{
		{ID: "task_1", DependsOn: []string{"task_2"}},
		{ID: "task_2", DependsOn: []string{"task_1"}},
	}

	hasCycle := planner.hasCircularDependency("task_1", "task_2", tasks)
	if !hasCycle {
		t.Error("Expected to detect circular dependency")
	}
}

func TestPlanningConfigDefaults(t *testing.T) {
	cfg := DefaultPlanningConfig()

	if cfg.AISummaryEnabled != true {
		t.Errorf("Expected AISummaryEnabled to be true, got %v", cfg.AISummaryEnabled)
	}

	if cfg.SimpleTaskThreshold != 3 {
		t.Errorf("Expected SimpleTaskThreshold to be 3, got %d", cfg.SimpleTaskThreshold)
	}

	if cfg.MaxPlanSteps != 20 {
		t.Errorf("Expected MaxPlanSteps to be 20, got %d", cfg.MaxPlanSteps)
	}
}

// TestIntegration 端到端测试：完整的规划流程
func TestIntegration(t *testing.T) {
	llm := &mockPlanningLLM{}
	planner := NewTaskPlanner(llm, DefaultPlanningConfig())

	ctx := context.Background()
	req := &PlanningRequest{
		UserMessage:       "帮我分析苹果最新产品并生成对比报告",
		CapabilitySummary: "## 内置工具\n- web_search: 搜索网络信息\n- file_io: 文件读写\n\n## 技能\n- docx: 生成 Word 文档\n- xlsx: 生成 Excel 表格",
		Context:           "用户对苹果产品感兴趣",
	}

	plan, err := planner.Plan(ctx, req)
	if err != nil {
		t.Fatalf("Integration test failed: %v", err)
	}

	// 验证计划的完整性
	if plan.Summary == "" {
		t.Error("Plan should have a summary")
	}

	if len(plan.Tasks) == 0 {
		t.Error("Plan should have at least one task")
	}

	// 验证任务格式
	for _, task := range plan.Tasks {
		if task.ID == "" {
			t.Error("Task ID should not be empty")
		}
		if task.Description == "" {
			t.Error("Task description should not be empty")
		}
		if task.Capability == "" {
			t.Error("Task capability should not be empty")
		}

		// 验证能力格式
		if !isValidCapabilityID(task.Capability) {
			t.Errorf("Invalid capability ID format: %s", task.Capability)
		}
	}

	// 验证 JSON 可序列化
	data, err := json.MarshalIndent(plan, "", "  ")
	if err != nil {
		t.Errorf("Plan should be JSON serializable: %v", err)
	}
	if len(data) == 0 {
		t.Error("Serialized plan should not be empty")
	}
}

func isValidCapabilityID(id string) bool {
	return len(id) > 5 && (id[0:5] == "tool:" || id[0:6] == "skill:" || id[0:4] == "mcp:")
}
