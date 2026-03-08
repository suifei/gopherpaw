package agent

import (
	"context"
	"testing"
)

// TestParseStructuredResponse_ValidJSON 测试解析有效的结构化响应。
func TestParseStructuredResponse_ValidJSON(t *testing.T) {
	content := `{
		"thought": "需要创建 Word 文档",
		"current_focus": "准备生成文档",
		"next_step": "读取 docx 技能文件",
		"capabilities_needed": ["docx"],
		"progress_update": "开始任务",
		"storage_requests": [
			{
				"name": "test_data",
				"description": "测试数据",
				"content": "这是测试内容"
			}
		],
		"retrieval_requests": ["previous_data"],
		"final_answer": "已完成"
	}`

	resp, err := ParseStructuredResponse(content)
	if err != nil {
		t.Fatalf("ParseStructuredResponse failed: %v", err)
	}

	if resp.Thought != "需要创建 Word 文档" {
		t.Errorf("Thought = %q, want %q", resp.Thought, "需要创建 Word 文档")
	}
	if resp.CurrentFocus != "准备生成文档" {
		t.Errorf("CurrentFocus = %q, want %q", resp.CurrentFocus, "准备生成文档")
	}
	if resp.NextStep != "读取 docx 技能文件" {
		t.Errorf("NextStep = %q, want %q", resp.NextStep, "读取 docx 技能文件")
	}
	if len(resp.CapabilitiesNeeded) != 1 || resp.CapabilitiesNeeded[0] != "docx" {
		t.Errorf("CapabilitiesNeeded = %v, want [docx]", resp.CapabilitiesNeeded)
	}
	if len(resp.StorageRequests) != 1 {
		t.Errorf("StorageRequests length = %d, want 1", len(resp.StorageRequests))
	}
	if len(resp.RetrievalRequests) != 1 || resp.RetrievalRequests[0] != "previous_data" {
		t.Errorf("RetrievalRequests = %v, want [previous_data]", resp.RetrievalRequests)
	}
	if resp.FinalAnswer != "已完成" {
		t.Errorf("FinalAnswer = %q, want %q", resp.FinalAnswer, "已完成")
	}
}

// TestParseStructuredResponse_MarkdownCodeBlock 测试解析 Markdown 代码块中的 JSON。
func TestParseStructuredResponse_MarkdownCodeBlock(t *testing.T) {
	// 使用纯代码块格式
	content := "```json\n" +
		"{\n" +
		"	\"thought\": \"AI 的思考过程\",\n" +
		"	\"current_focus\": \"当前关注点\",\n" +
		"	\"next_step\": \"下一步行动\",\n" +
		"	\"capabilities_needed\": [\"search\", \"browse\"]\n" +
		"}\n" +
		"```"

	resp, err := ParseStructuredResponse(content)
	if err != nil {
		t.Fatalf("ParseStructuredResponse failed: %v", err)
	}

	if resp.Thought != "AI 的思考过程" {
		t.Errorf("Thought = %q, want %q", resp.Thought, "AI 的思考过程")
	}
	if len(resp.CapabilitiesNeeded) != 2 {
		t.Errorf("CapabilitiesNeeded length = %d, want 2", len(resp.CapabilitiesNeeded))
	}
}

// TestParseStructuredResponse_NoJSON 测试解析没有 JSON 的内容。
func TestParseStructuredResponse_NoJSON(t *testing.T) {
	content := "这是一个普通的回答，没有结构化数据。"

	resp, err := ParseStructuredResponse(content)
	if err != nil {
		t.Fatalf("ParseStructuredResponse should not error on non-JSON content: %v", err)
	}
	if resp != nil {
		t.Error("ParseStructuredResponse should return nil for non-JSON content")
	}
}

// TestParseStructuredResponse_InvalidJSON 测试解析无效的 JSON。
func TestParseStructuredResponse_InvalidJSON(t *testing.T) {
	content := `{
		"thought": "test",
		"invalid": this is not valid json
	}`

	resp, err := ParseStructuredResponse(content)
	if err != nil {
		t.Fatalf("ParseStructuredResponse should not error on invalid JSON: %v", err)
	}
	if resp != nil {
		t.Error("ParseStructuredResponse should return nil for invalid JSON")
	}
}

// TestMemoryContextManager_Store 测试存储功能。
func TestMemoryContextManager_Store(t *testing.T) {
	ctx := context.Background()
	cm := NewMemoryContextManager()
	chatID := "test_chat"

	requests := []StorageRequest{
		{
			Name:        "test_data",
			Description: "测试数据",
			Content:     "这是测试内容",
		},
	}

	err := cm.Store(ctx, chatID, requests)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// 验证存储
	retrieved, err := cm.Retrieve(ctx, chatID, []string{"test_data"})
	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}

	if len(retrieved) != 1 {
		t.Fatalf("Retrieved count = %d, want 1", len(retrieved))
	}

	if retrieved[0].Name != "test_data" {
		t.Errorf("Retrieved name = %q, want %q", retrieved[0].Name, "test_data")
	}
	if retrieved[0].Content != "这是测试内容" {
		t.Errorf("Retrieved content = %q, want %q", retrieved[0].Content, "这是测试内容")
	}
}

// TestMemoryContextManager_StoreMultiple 测试存储多个内容。
func TestMemoryContextManager_StoreMultiple(t *testing.T) {
	ctx := context.Background()
	cm := NewMemoryContextManager()
	chatID := "test_chat"

	requests := []StorageRequest{
		{Name: "data1", Description: "数据1", Content: "内容1"},
		{Name: "data2", Description: "数据2", Content: "内容2"},
		{Name: "data3", Description: "数据3", Content: "内容3"},
	}

	err := cm.Store(ctx, chatID, requests)
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	// 检索所有
	allNames, err := cm.ListAll(ctx, chatID)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(allNames) != 3 {
		t.Errorf("ListAll count = %d, want 3", len(allNames))
	}

	// 检索部分
	retrieved, err := cm.Retrieve(ctx, chatID, []string{"data1", "data3"})
	if err != nil {
		t.Fatalf("Retrieve failed: %v", err)
	}

	if len(retrieved) != 2 {
		t.Errorf("Retrieved count = %d, want 2", len(retrieved))
	}
}

// TestMemoryContextManager_Goals 测试目标管理。
func TestMemoryContextManager_Goals(t *testing.T) {
	ctx := context.Background()
	cm := NewMemoryContextManager()
	chatID := "test_chat"

	// 设置目标
	goal := "生成 MacBook Air 价格对比报告"
	err := cm.SetGoal(ctx, chatID, goal)
	if err != nil {
		t.Fatalf("SetGoal failed: %v", err)
	}

	// 获取目标
	retrieved, err := cm.GetGoal(ctx, chatID)
	if err != nil {
		t.Fatalf("GetGoal failed: %v", err)
	}

	if retrieved != goal {
		t.Errorf("GetGoal = %q, want %q", retrieved, goal)
	}
}

// TestMemoryContextManager_Milestones 测试里程碑管理。
func TestMemoryContextManager_Milestones(t *testing.T) {
	ctx := context.Background()
	cm := NewMemoryContextManager()
	chatID := "test_chat"

	// 添加里程碑
	milestone1 := Milestone{Name: "收集价格数据", Status: "完成"}
	milestone2 := Milestone{Name: "生成 Word 文档", Status: "进行中"}

	err := cm.AddMilestone(ctx, chatID, milestone1)
	if err != nil {
		t.Fatalf("AddMilestone failed: %v", err)
	}

	err = cm.AddMilestone(ctx, chatID, milestone2)
	if err != nil {
		t.Fatalf("AddMilestone failed: %v", err)
	}

	// 获取里程碑
	milestones, err := cm.GetMilestones(ctx, chatID)
	if err != nil {
		t.Fatalf("GetMilestones failed: %v", err)
	}

	if len(milestones) != 2 {
		t.Errorf("Milestones count = %d, want 2", len(milestones))
	}

	if milestones[0].Name != "收集价格数据" {
		t.Errorf("First milestone name = %q, want %q", milestones[0].Name, "收集价格数据")
	}
}

// TestMemoryContextManager_BuildCapabilityReminder 测试智能提点生成。
func TestMemoryContextManager_BuildCapabilityReminder(t *testing.T) {
	ctx := context.Background()
	cm := NewMemoryContextManager()
	chatID := "test_chat"

	// 设置技能路径
	skillPaths := map[string]string{
		"docx":  "configs/active_skills/docx/SKILL.md",
		"xlsx":  "configs/active_skills/xlsx/SKILL.md",
		"pptx":  "configs/active_skills/pptx/SKILL.md",
		"pdf":   "configs/active_skills/pdf/SKILL.md",
		"image": "configs/active_skills/image/SKILL.md",
	}

	if mcm, ok := cm.(*memoryContextManager); ok {
		mcm.SetSkillPaths(skillPaths)
	}

	// 请求一个已知能力
	reminder := cm.BuildCapabilityReminder(ctx, chatID, []string{"docx"})
	if reminder == "" {
		t.Error("Expected non-empty reminder for 'docx' capability")
	}

	// 验证提醒包含相关信息
	if !contains(reminder, "docx") || !contains(reminder, "SKILL.md") {
		t.Errorf("Reminder should contain 'docx' and 'SKILL.md', got: %s", reminder)
	}

	// 请求多个能力
	reminder = cm.BuildCapabilityReminder(ctx, chatID, []string{"docx", "xlsx"})
	if reminder == "" {
		t.Error("Expected non-empty reminder for multiple capabilities")
	}

	// 请求未知能力
	reminder = cm.BuildCapabilityReminder(ctx, chatID, []string{"unknown_skill"})
	if reminder != "" {
		t.Errorf("Expected empty reminder for unknown capability, got: %s", reminder)
	}
}

// TestMemoryContextManager_InjectContext 测试上下文注入。
func TestMemoryContextManager_InjectContext(t *testing.T) {
	ctx := context.Background()
	cm := NewMemoryContextManager()
	chatID := "test_chat"

	// 设置目标
	_ = cm.SetGoal(ctx, chatID, "生成报告")

	// 添加里程碑
	_ = cm.AddMilestone(ctx, chatID, Milestone{Name: "步骤1", Status: "完成"})

	// 存储一些数据
	_ = cm.Store(ctx, chatID, []StorageRequest{
		{Name: "price_data", Description: "价格数据", Content: "MacBook: $1099"},
	})

	// 创建消息
	messages := []Message{
		{Role: "user", Content: "继续生成报告"},
	}

	// 注入上下文（不检索）
	injected, err := cm.InjectContext(ctx, chatID, messages, nil)
	if err != nil {
		t.Fatalf("InjectContext failed: %v", err)
	}

	if len(injected) != 1 {
		t.Errorf("Injected messages count = %d, want 1", len(injected))
	}

	// 验证用户消息被修改
	if !contains(injected[0].Content, "当前目标") && !contains(injected[0].Content, "当前进度") {
		t.Error("Injected context should contain goal or milestone info")
	}

	// 测试带检索的注入
	injected, err = cm.InjectContext(ctx, chatID, messages, []string{"price_data"})
	if err != nil {
		t.Fatalf("InjectContext with retrieval failed: %v", err)
	}

	if !contains(injected[0].Content, "price_data") || !contains(injected[0].Content, "MacBook: $1099") {
		t.Error("Injected context should contain retrieved data")
	}
}

// TestExtractStructuredContent 测试从结构化响应中提取最终回答。
func TestExtractStructuredContent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "With FinalAnswer",
			input:    `{"thought": "thinking", "final_answer": "这是最终答案"}`,
			expected: "这是最终答案",
		},
		{
			name:     "No structured response",
			input:    "普通回答内容",
			expected: "普通回答内容",
		},
		{
			name:     "With Thought but no FinalAnswer",
			input:    `{"thought": "这是思考过程"}`,
			expected: "这是思考过程",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractStructuredContent(tt.input)
			if result != tt.expected {
				t.Errorf("ExtractStructuredContent() = %q, want %q", result, tt.expected)
			}
		})
	}
}

// TestIsFinalResponse 测试判断是否为最终响应。
func TestIsFinalResponse(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "With FinalAnswer",
			input:    `{"thought": "thinking", "final_answer": "完成"}`,
			expected: true,
		},
		{
			name:     "No FinalAnswer",
			input:    `{"thought": "thinking"}`,
			expected: false,
		},
		{
			name:     "No structured response",
			input:    "普通回答",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsFinalResponse(tt.input)
			if result != tt.expected {
				t.Errorf("IsFinalResponse() = %v, want %v", result, tt.expected)
			}
		})
	}
}
