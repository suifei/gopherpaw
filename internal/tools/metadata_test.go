package tools

import (
	"context"
	"testing"
)

// TestReadFileToolMetadata 测试 ReadFileTool 的元数据方法
func TestReadFileToolMetadata(t *testing.T) {
	tool := &ReadFileTool{}

	if tool.Name() != "read_file" {
		t.Errorf("Name() = %q, want read_file", tool.Name())
	}

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() is empty")
	}

	params := tool.Parameters()
	if params == nil {
		t.Error("Parameters() is nil")
	}
}

// TestWriteFileToolMetadata 测试 WriteFileTool 的元数据方法
func TestWriteFileToolMetadata(t *testing.T) {
	tool := &WriteFileTool{}

	if tool.Name() != "write_file" {
		t.Errorf("Name() = %q, want write_file", tool.Name())
	}

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() is empty")
	}

	params := tool.Parameters()
	if params == nil {
		t.Error("Parameters() is nil")
	}
}

// TestEditFileToolMetadata 测试 EditFileTool 的元数据方法
func TestEditFileToolMetadata(t *testing.T) {
	tool := &EditFileTool{}

	if tool.Name() != "edit_file" {
		t.Errorf("Name() = %q, want edit_file", tool.Name())
	}

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() is empty")
	}

	params := tool.Parameters()
	if params == nil {
		t.Error("Parameters() is nil")
	}
}

// TestAppendFileToolMetadata 测试 AppendFileTool 的元数据方法
func TestAppendFileToolMetadata(t *testing.T) {
	tool := &AppendFileTool{}

	if tool.Name() != "append_file" {
		t.Errorf("Name() = %q, want append_file", tool.Name())
	}

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() is empty")
	}

	params := tool.Parameters()
	if params == nil {
		t.Error("Parameters() is nil")
	}
}

// TestTimeToolMetadata 测试 TimeTool 的元数据方法
func TestTimeToolMetadata(t *testing.T) {
	tool := &TimeTool{}

	if tool.Name() != "get_current_time" {
		t.Errorf("Name() = %q, want get_current_time", tool.Name())
	}

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() is empty")
	}

	params := tool.Parameters()
	if params == nil {
		t.Error("Parameters() is nil")
	}
}

// TestShellToolMetadata 测试 ShellTool 的元数据方法
func TestShellToolMetadata(t *testing.T) {
	tool := &ShellTool{}

	if tool.Name() != "execute_shell_command" {
		t.Errorf("Name() = %q, want execute_shell_command", tool.Name())
	}

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() is empty")
	}

	params := tool.Parameters()
	if params == nil {
		t.Error("Parameters() is nil")
	}
}

// TestMemorySearchToolMetadata 测试 MemorySearchTool 的元数据方法
func TestMemorySearchToolMetadata(t *testing.T) {
	tool := &MemorySearchTool{}

	if tool.Name() != "memory_search" {
		t.Errorf("Name() = %q, want memory_search", tool.Name())
	}

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() is empty")
	}

	params := tool.Parameters()
	if params == nil {
		t.Error("Parameters() is nil")
	}
}

// TestGrepSearchToolMetadata 测试 GrepSearchTool 的元数据方法
func TestGrepSearchToolMetadata(t *testing.T) {
	tool := &GrepSearchTool{}

	if tool.Name() != "grep_search" {
		t.Errorf("Name() = %q, want grep_search", tool.Name())
	}

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() is empty")
	}

	params := tool.Parameters()
	if params == nil {
		t.Error("Parameters() is nil")
	}
}

// TestGlobSearchToolMetadata 测试 GlobSearchTool 的元数据方法
func TestGlobSearchToolMetadata(t *testing.T) {
	tool := &GlobSearchTool{}

	if tool.Name() != "glob_search" {
		t.Errorf("Name() = %q, want glob_search", tool.Name())
	}

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() is empty")
	}

	params := tool.Parameters()
	if params == nil {
		t.Error("Parameters() is nil")
	}
}

// TestSendFileToolMetadata 测试 SendFileTool 的元数据方法
func TestSendFileToolMetadata(t *testing.T) {
	tool := &SendFileTool{}

	if tool.Name() != "send_file_to_user" {
		t.Errorf("Name() = %q, want send_file_to_user", tool.Name())
	}

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() is empty")
	}

	params := tool.Parameters()
	if params == nil {
		t.Error("Parameters() is nil")
	}
}

// TestWebSearchToolMetadata 测试 WebSearchTool 的元数据方法（如果已创建）
func TestWebSearchToolMetadata(t *testing.T) {
	ws, err := NewWebSearchTool()
	if err != nil {
		t.Fatalf("NewWebSearchTool failed: %v", err)
	}

	if ws.Name() != "web_search" {
		t.Errorf("Name() = %q, want web_search", ws.Name())
	}

	desc := ws.Description()
	if desc == "" {
		t.Error("Description() is empty")
	}

	params := ws.Parameters()
	if params == nil {
		t.Error("Parameters() is nil")
	}
}

// TestHTTPToolMetadata 测试 HTTPTool 的元数据方法
func TestHTTPToolMetadata(t *testing.T) {
	tool := NewHTTPTool()

	if tool.Name() != "http_request" {
		t.Errorf("Name() = %q, want http_request", tool.Name())
	}

	desc := tool.Description()
	if desc == "" {
		t.Error("Description() is empty")
	}

	params := tool.Parameters()
	if params == nil {
		t.Error("Parameters() is nil")
	}
}

// TestSendFileToolExecuteError 测试 SendFileTool 的错误处理
func TestSendFileToolExecuteError(t *testing.T) {
	tool := &SendFileTool{}

	// 无效的 JSON 应该返回错误
	_, err := tool.Execute(context.Background(), `{invalid}`)
	if err == nil {
		t.Error("Execute with invalid JSON should return error")
	}
}

// TestEditFileToolExecuteError 测试 EditFileTool 的错误处理
func TestEditFileToolExecuteError(t *testing.T) {
	tool := &EditFileTool{}

	// 无效的 JSON 应该返回错误
	_, err := tool.Execute(context.Background(), `{invalid}`)
	if err == nil {
		t.Error("Execute with invalid JSON should return error")
	}
}

// TestMemorySearchToolExecuteError 测试 MemorySearchTool 的错误处理
func TestMemorySearchToolExecuteError(t *testing.T) {
	tool := &MemorySearchTool{}

	// 无效的 JSON 可能不会返回错误，而是返回一个错误消息字符串
	out, err := tool.Execute(context.Background(), `{invalid}`)

	// 无论是返回错误还是错误消息都可以接受
	if err == nil && out == "" {
		t.Error("Execute with invalid JSON should return error or error message")
	}
}

// TestGrepSearchToolExecuteError 测试 GrepSearchTool 的错误处理
func TestGrepSearchToolExecuteError(t *testing.T) {
	tool := &GrepSearchTool{}

	// 无效的 JSON 应该返回错误
	_, err := tool.Execute(context.Background(), `{invalid}`)
	if err == nil {
		t.Error("Execute with invalid JSON should return error")
	}
}

// TestGlobSearchToolExecuteError 测试 GlobSearchTool 的错误处理
func TestGlobSearchToolExecuteError(t *testing.T) {
	tool := &GlobSearchTool{}

	// 无效的 JSON 应该返回错误
	_, err := tool.Execute(context.Background(), `{invalid}`)
	if err == nil {
		t.Error("Execute with invalid JSON should return error")
	}
}

// TestBuiltinToolsMetadata 测试所有内置工具都有正确的元数据
func TestBuiltinToolsMetadata(t *testing.T) {
	tools := RegisterBuiltin()

	for _, tool := range tools {
		t.Run(tool.Name(), func(t *testing.T) {
			// 每个工具都应该有名称
			if tool.Name() == "" {
				t.Error("Tool name is empty")
			}

			// 每个工具都应该有参数定义
			if tool.Parameters() == nil {
				t.Error("Tool Parameters() is nil")
			}

			// Description 可以为空（某些工具可能没有）
			// 但应该是可调用的
			_ = tool.Description()
		})
	}
}

// TestBuiltinToolsMetadataQuality 测试所有内置工具的元数据质量
func TestBuiltinToolsMetadataQuality(t *testing.T) {
	tools := RegisterBuiltin()

	for _, tool := range tools {
		t.Run(tool.Name(), func(t *testing.T) {
			// 检查名称格式
			name := tool.Name()
			if name == "" {
				t.Error("Tool name is empty")
				return
			}

			// 检查描述质量
			desc := tool.Description()
			if desc == "" {
				t.Logf("Warning: Tool %s has empty description", name)
			} else if len(desc) < 20 {
				t.Logf("Warning: Tool %s description might be too short: %q", name, desc)
			}

			// 检查参数定义质量
			params := tool.Parameters()
			if params == nil {
				t.Errorf("Tool %s has nil parameters", name)
				return
			}

			// 验证参数定义格式
			paramsMap, ok := params.(map[string]any)
			if !ok {
				t.Errorf("Tool %s parameters should be map[string]any, got %T", name, params)
				return
			}

			// 检查必需字段
			if paramsMap["type"] == nil {
				t.Errorf("Tool %s parameters missing 'type' field", name)
			}

			if paramsMap["properties"] == nil {
				t.Errorf("Tool %s parameters missing 'properties' field", name)
			} else {
				// 检查每个属性是否有描述
				props, ok := paramsMap["properties"].(map[string]any)
				if !ok {
					t.Errorf("Tool %s properties should be map[string]any, got %T", name, paramsMap["properties"])
					return
				}

				for propName, propValue := range props {
					prop, ok := propValue.(map[string]any)
					if !ok {
						t.Errorf("Tool %s property %s should be map[string]any, got %T", name, propName, propValue)
						continue
					}

					// 每个属性应该有 type
					if prop["type"] == nil {
						t.Errorf("Tool %s property %s missing 'type' field", name, propName)
					}

					// 每个属性应该有 description
					if prop["description"] == nil {
						t.Logf("Warning: Tool %s property %s missing 'description' field", name, propName)
					}
				}
			}
		})
	}
}
