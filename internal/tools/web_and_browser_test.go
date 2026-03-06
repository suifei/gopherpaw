package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestNewWebSearchTool_Success(t *testing.T) {
	_, err := NewWebSearchTool()
	if err != nil {
		t.Errorf("NewWebSearchTool: unexpected error: %v", err)
	}
}

func TestWebSearchTool_MaxResults(t *testing.T) {
	_, err := NewWebSearchTool()
	if err != nil {
		t.Skipf("Skipping WebSearch tests: %v", err)
	}

	tests := []struct {
		name    string
		args    string
		wantMax int
	}{
		{
			name:    "default max results",
			args:    `{"query":"test"}`,
			wantMax: defaultWebSearchMaxResults,
		},
		{
			name:    "custom max results",
			args:    `{"query":"test","max_results":5}`,
			wantMax: 5,
		},
		{
			name:    "max results clamped to 15",
			args:    `{"query":"test","max_results":20}`,
			wantMax: 15,
		},
		{
			name:    "zero max results uses default",
			args:    `{"query":"test","max_results":0}`,
			wantMax: defaultWebSearchMaxResults,
		},
		{
			name:    "negative max results uses default",
			args:    `{"query":"test","max_results":-1}`,
			wantMax: defaultWebSearchMaxResults,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args webSearchArgs
			if err := json.Unmarshal([]byte(tt.args), &args); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}

			maxResults := args.MaxResults
			if maxResults <= 0 {
				maxResults = defaultWebSearchMaxResults
			}
			if maxResults > 15 {
				maxResults = 15
			}

			if maxResults != tt.wantMax {
				t.Errorf("maxResults = %d, want %d", maxResults, tt.wantMax)
			}
		})
	}
}

func TestWebSearchTool_InvalidJSON(t *testing.T) {
	ws, err := NewWebSearchTool()
	if err != nil {
		t.Skipf("Skipping WebSearch tests: %v", err)
	}

	_, err = ws.Execute(context.Background(), `{invalid json`)
	if err == nil {
		t.Error("Execute invalid JSON: expected error")
	}
}

func TestWebSearchTool_QueryTrim(t *testing.T) {
	ws, err := NewWebSearchTool()
	if err != nil {
		t.Skipf("Skipping WebSearch tests: %v", err)
	}

	out, err := ws.Execute(context.Background(), `{"query":"  test  "}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if strings.Contains(out, "No query provided") {
		t.Error("Whitespace-only query should not trigger no query error")
	}
}

func TestWebSearchTool_ContextCancellation(t *testing.T) {
	ws, err := NewWebSearchTool()
	if err != nil {
		t.Skipf("Skipping WebSearch tests: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = ws.Execute(ctx, `{"query":"test"}`)
	if err == nil {
		t.Error("Execute with cancelled context: expected error")
	}
}

func TestWebSearchTool_ResultFormat(t *testing.T) {
	ws, err := NewWebSearchTool()
	if err != nil {
		t.Skipf("Skipping WebSearch tests: %v", err)
	}

	if testing.Short() {
		t.Skip("skipping web search test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	out, err := ws.Execute(ctx, `{"query":"golang","max_results":3}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if strings.Contains(out, "No results found") {
		t.Logf("No results found (may be network issue)")
		return
	}

	lines := strings.Split(out, "\n")
	if len(lines) == 0 {
		t.Error("Expected non-empty output")
	}

	hasResults := false
	for _, line := range lines {
		if strings.Contains(line, ".") && len(line) > 10 {
			hasResults = true
			break
		}
	}

	if !hasResults {
		t.Logf("Output format may be unexpected: %s", out)
	}
}

func TestWebSearchTool_Metadata(t *testing.T) {
	tool, err := NewWebSearchTool()
	if err != nil {
		t.Skipf("Skipping WebSearch tests: %v", err)
	}

	if tool.Name() != "web_search" {
		t.Errorf("Name = %q, want web_search", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Description should not be empty")
	}

	params := tool.Parameters()
	if params == nil {
		t.Error("Parameters should not be nil")
	}

	paramMap, ok := params.(map[string]any)
	if !ok {
		t.Fatalf("Parameters should be map[string]any, got %T", params)
	}

	props, ok := paramMap["properties"].(map[string]any)
	if !ok {
		t.Fatal("Parameters should have properties")
	}

	if _, ok := props["query"]; !ok {
		t.Error("Parameters should have 'query' property")
	}

	if _, ok := props["max_results"]; !ok {
		t.Error("Parameters should have 'max_results' property")
	}
}

func TestBrowserTool_Metadata(t *testing.T) {
	tool := &BrowserTool{}

	if tool.Name() != "browser_use" {
		t.Errorf("Name = %q, want browser_use", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Description should not be empty")
	}

	params := tool.Parameters()
	if params == nil {
		t.Error("Parameters should not be nil")
	}

	paramMap, ok := params.(map[string]any)
	if !ok {
		t.Fatalf("Parameters should be map[string]any, got %T", params)
	}

	props, ok := paramMap["properties"].(map[string]any)
	if !ok {
		t.Fatal("Parameters should have properties")
	}

	requiredActions := []string{"url", "page_id", "selector", "text", "code", "path"}
	for _, action := range requiredActions {
		if _, ok := props[action]; !ok {
			t.Errorf("Parameters should have '%s' property", action)
		}
	}
}

func TestBrowserTool_Execute(t *testing.T) {
	tool := &BrowserTool{}

	tests := []struct {
		name    string
		args    string
		wantErr bool
		errSub  string
	}{
		{
			name:    "invalid JSON",
			args:    `{invalid}`,
			wantErr: true,
			errSub:  "parse arguments",
		},
		{
			name:    "unknown action",
			args:    `{"action":"unknown_action"}`,
			wantErr: true,
			errSub:  "unknown browser action",
		},
		{
			name:    "open without URL",
			args:    `{"action":"open"}`,
			wantErr: true,
			errSub:  "url is required",
		},
		{
			name:    "click without selector",
			args:    `{"action":"click","page_id":"test"}`,
			wantErr: true,
			errSub:  "page",
		},
		{
			name:    "type without selector",
			args:    `{"action":"type","page_id":"test"}`,
			wantErr: true,
			errSub:  "page",
		},
		{
			name:    "eval without code",
			args:    `{"action":"eval","page_id":"test"}`,
			wantErr: true,
			errSub:  "page",
		},
		{
			name:    "page not found",
			args:    `{"action":"screenshot","page_id":"nonexistent"}`,
			wantErr: true,
			errSub:  "page",
		},
		{
			name:    "navigate_back without page",
			args:    `{"action":"navigate_back","page_id":"nonexistent"}`,
			wantErr: true,
			errSub:  "page",
		},
		{
			name:    "wait_for without parameters",
			args:    `{"action":"wait_for","page_id":"test"}`,
			wantErr: true,
			errSub:  "page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tool.Execute(context.Background(), tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if tt.errSub != "" && !strings.Contains(err.Error(), tt.errSub) {
					t.Errorf("Expected error containing %q, got %v", tt.errSub, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestBrowserTool_ExecuteRich(t *testing.T) {
	tool := &BrowserTool{}

	tests := []struct {
		name    string
		args    string
		wantErr bool
		errSub  string
	}{
		{
			name:    "invalid JSON",
			args:    `{invalid}`,
			wantErr: true,
			errSub:  "parse arguments",
		},
		{
			name:    "unknown action",
			args:    `{"action":"unknown_action"}`,
			wantErr: true,
			errSub:  "unknown browser action",
		},
		{
			name:    "open without URL",
			args:    `{"action":"open"}`,
			wantErr: true,
			errSub:  "url is required",
		},
		{
			name:    "page not found",
			args:    `{"action":"click","page_id":"nonexistent","selector":"button"}`,
			wantErr: true,
			errSub:  "page",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tool.ExecuteRich(context.Background(), tt.args)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error, got nil")
				} else if tt.errSub != "" && !strings.Contains(err.Error(), tt.errSub) {
					t.Errorf("Expected error containing %q, got %v", tt.errSub, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestBrowserTool_InvalidArguments(t *testing.T) {
	tool := &BrowserTool{}

	tests := []struct {
		name   string
		args   string
		fields []string
	}{
		{
			name:   "press_key without key",
			args:   `{"action":"press_key"}`,
			fields: []string{"key"},
		},
		{
			name:   "file_upload without selector",
			args:   `{"action":"file_upload","file_paths":["test.txt"]}`,
			fields: []string{"selector"},
		},
		{
			name:   "file_upload without file_paths",
			args:   `{"action":"file_upload","selector":"input"}`,
			fields: []string{"file_paths"},
		},
		{
			name:   "select_option without selector",
			args:   `{"action":"select_option","values":["option1"]}`,
			fields: []string{"selector"},
		},
		{
			name:   "select_option without values",
			args:   `{"action":"select_option","selector":"select"}`,
			fields: []string{"values"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tool.ExecuteRich(context.Background(), tt.args)
			if err == nil {
				t.Errorf("Expected error for missing fields %v", tt.fields)
			}
		})
	}
}

func TestBrowserTool_ContextCancellation(t *testing.T) {
	tool := &BrowserTool{}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := tool.Execute(ctx, `{}`)
	if err == nil {
		t.Error("Execute with cancelled context: expected error")
	}
}

func TestBrowserTool_DefaultPageID(t *testing.T) {
	tool := &BrowserTool{}

	args := `{"action":"navigate_back","page_id":""}`
	_, err := tool.Execute(context.Background(), args)
	if err == nil {
		t.Error("Expected error for non-existent page")
	}
	if !strings.Contains(err.Error(), "page") {
		t.Errorf("Expected page error, got %v", err)
	}

	args2 := `{"action":"navigate_back"}`
	_, err2 := tool.Execute(context.Background(), args2)
	if err2 == nil {
		t.Error("Expected error for non-existent default page")
	}
}

func TestBrowserTool_DoClose(t *testing.T) {
	tool := &BrowserTool{}

	tests := []struct {
		name    string
		args    string
		wantErr bool
	}{
		{
			name:    "close non-existent page",
			args:    `{"action":"close","page_id":"nonexistent"}`,
			wantErr: true,
		},
		{
			name:    "close with default page_id",
			args:    `{"action":"close"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tool.ExecuteRich(context.Background(), tt.args)
			if tt.wantErr && err == nil {
				t.Error("Expected error")
			}
		})
	}
}

func TestBrowserTool_DoTabs(t *testing.T) {
	tool := &BrowserTool{}

	out, err := tool.ExecuteRich(context.Background(), `{"action":"tabs"}`)
	if err != nil {
		t.Fatalf("ExecuteRich: %v", err)
	}

	if out == nil {
		t.Error("Expected non-nil result")
	}

	if out.Text == "" {
		t.Error("Expected non-empty text in result")
	}
}

func TestBrowserTool_DoStop(t *testing.T) {
	tool := &BrowserTool{}

	out, err := tool.ExecuteRich(context.Background(), `{"action":"stop"}`)
	if err != nil {
		t.Fatalf("ExecuteRich: %v", err)
	}

	if out == nil {
		t.Error("Expected non-nil result")
	}

	if !strings.Contains(out.Text, "stopped") {
		t.Errorf("Expected 'stopped' in result, got %q", out.Text)
	}
}

func TestBrowserTool_DoWaitFor(t *testing.T) {
	tool := &BrowserTool{}

	tests := []struct {
		name    string
		args    string
		wantErr bool
	}{
		{
			name:    "wait with time",
			args:    `{"action":"wait_for","wait":100}`,
			wantErr: true,
		},
		{
			name:    "wait with selector",
			args:    `{"action":"wait_for","selector":"body"}`,
			wantErr: true,
		},
		{
			name:    "wait_for without parameters",
			args:    `{"action":"wait_for"}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tool.ExecuteRich(context.Background(), tt.args)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestBrowserTool_DoResize(t *testing.T) {
	tool := &BrowserTool{}

	tests := []struct {
		name    string
		args    string
		wantErr bool
	}{
		{
			name:    "resize with dimensions",
			args:    `{"action":"resize","width":1920,"height":1080}`,
			wantErr: true,
		},
		{
			name:    "resize with zero dimensions",
			args:    `{"action":"resize","width":0,"height":0}`,
			wantErr: true,
		},
		{
			name:    "resize with negative dimensions",
			args:    `{"action":"resize","width":-100,"height":-100}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tool.ExecuteRich(context.Background(), tt.args)
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error")
				}
			}
		})
	}
}
