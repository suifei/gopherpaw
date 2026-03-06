package tools

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestHTTPTool_POST(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		w.Write([]byte(`{"success":true}`))
	}))
	defer srv.Close()

	tool := NewHTTPTool()
	out, err := tool.Execute(context.Background(), fmt.Sprintf(`{"url":"%s","method":"POST","body":"{\"test\":true}"}`, srv.URL))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "success") {
		t.Errorf("Expected 'success' in output, got %q", out)
	}
}

func TestHTTPTool_PUT(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("Expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"updated":true}`))
	}))
	defer srv.Close()

	tool := NewHTTPTool()
	out, err := tool.Execute(context.Background(), fmt.Sprintf(`{"url":"%s","method":"PUT","body":"{\"key\":\"value\"}"}`, srv.URL))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "updated") {
		t.Errorf("Expected 'updated' in output, got %q", out)
	}
}

func TestHTTPTool_DELETE(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	tool := NewHTTPTool()
	out, err := tool.Execute(context.Background(), fmt.Sprintf(`{"url":"%s","method":"DELETE"}`, srv.URL))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out != "" {
		t.Errorf("Expected empty output for 204, got %q", out)
	}
}

func TestHTTPTool_PATCH(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("Expected PATCH, got %s", r.Method)
		}
		w.Write([]byte(`{"patched":true}`))
	}))
	defer srv.Close()

	tool := NewHTTPTool()
	out, err := tool.Execute(context.Background(), fmt.Sprintf(`{"url":"%s","method":"PATCH","body":"{\"patch\":true}"}`, srv.URL))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "patched") {
		t.Errorf("Expected 'patched' in output, got %q", out)
	}
}

func TestHTTPTool_ErrorStatus(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		wantStatusText string
	}{
		{
			name:           "404 Not Found",
			statusCode:     http.StatusNotFound,
			responseBody:   "Not found",
			wantStatusText: "HTTP 404",
		},
		{
			name:           "500 Internal Server Error",
			statusCode:     http.StatusInternalServerError,
			responseBody:   "Server error",
			wantStatusText: "HTTP 500",
		},
		{
			name:           "403 Forbidden",
			statusCode:     http.StatusForbidden,
			responseBody:   "Forbidden",
			wantStatusText: "HTTP 403",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.responseBody))
			}))
			defer srv.Close()

			tool := NewHTTPTool()
			out, err := tool.Execute(context.Background(), fmt.Sprintf(`{"url":"%s"}`, srv.URL))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if !strings.Contains(out, tt.wantStatusText) {
				t.Errorf("Expected %q in output, got %q", tt.wantStatusText, out)
			}
			if !strings.Contains(out, tt.responseBody) {
				t.Errorf("Expected response body %q in output, got %q", tt.responseBody, out)
			}
		})
	}
}

func TestHTTPTool_CustomHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Custom-Header") != "test-value" {
			t.Errorf("Expected X-Custom-Header: test-value, got %q", r.Header.Get("X-Custom-Header"))
		}
		if r.Header.Get("Authorization") != "Bearer token123" {
			t.Errorf("Expected Authorization: Bearer token123, got %q", r.Header.Get("Authorization"))
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tool := NewHTTPTool()
	out, err := tool.Execute(context.Background(), fmt.Sprintf(`{
		"url":"%s",
		"headers":{
			"X-Custom-Header":"test-value",
			"Authorization":"Bearer token123"
		}
	}`, srv.URL))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("Expected 'ok' in output, got %q", out)
	}
}

func TestHTTPTool_CustomContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/xml" {
			t.Errorf("Expected Content-Type: application/xml, got %q", r.Header.Get("Content-Type"))
		}
		w.Write([]byte(`<ok/>`))
	}))
	defer srv.Close()

	tool := NewHTTPTool()
	out, err := tool.Execute(context.Background(), fmt.Sprintf(`{
		"url":"%s",
		"method":"POST",
		"body":"<test/>",
		"headers":{"Content-Type":"application/xml"}
	}`, srv.URL))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "<ok/>") {
		t.Errorf("Expected '<ok/>' in output, got %q", out)
	}
}

func TestHTTPTool_DefaultContentType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ct := r.Header.Get("Content-Type")
		if ct != "application/json" {
			t.Errorf("Expected default Content-Type: application/json, got %q", ct)
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tool := NewHTTPTool()
	_, err := tool.Execute(context.Background(), fmt.Sprintf(`{
		"url":"%s",
		"method":"POST",
		"body":"{\"test\":true}"
	}`, srv.URL))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestHTTPTool_LargeResponse(t *testing.T) {
	largeContent := strings.Repeat("a", 100*1024)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(largeContent))
	}))
	defer srv.Close()

	tool := NewHTTPTool()
	out, err := tool.Execute(context.Background(), fmt.Sprintf(`{"url":"%s"}`, srv.URL))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(out) < len(largeContent) {
		if !strings.Contains(out, "truncated") {
			t.Errorf("Expected truncation message, got %d bytes", len(out))
		}
	}
}

func TestHTTPTool_UnsupportedMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer srv.Close()

	tool := NewHTTPTool()
	out, err := tool.Execute(context.Background(), fmt.Sprintf(`{"url":"%s","method":"INVALID"}`, srv.URL))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Error") || !strings.Contains(out, "Unsupported HTTP method") {
		t.Errorf("Expected unsupported method error, got %q", out)
	}
}

func TestHTTPTool_NetworkError(t *testing.T) {
	tool := NewHTTPTool()
	_, err := tool.Execute(context.Background(), `{"url":"http://localhost:99999"}`)
	if err == nil {
		t.Error("Expected network error, got nil")
	}
}

func TestHTTPTool_InvalidJSON(t *testing.T) {
	tool := NewHTTPTool()
	_, err := tool.Execute(context.Background(), `{invalid json`)
	if err == nil {
		t.Error("Expected error for invalid JSON, got nil")
	}
}

func TestHTTPTool_EmptyArguments(t *testing.T) {
	tool := NewHTTPTool()
	out, err := tool.Execute(context.Background(), "")
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "Error") || !strings.Contains(out, "No URL provided") {
		t.Errorf("Expected no URL error, got %q", out)
	}
}

func TestHTTPTool_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping timeout test in short mode")
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Second)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tool := NewHTTPTool()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := tool.Execute(ctx, fmt.Sprintf(`{"url":"%s"}`, srv.URL))
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

func TestHTTPTool_QueryString(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.RawQuery != "key1=value1&key2=value2" {
			t.Errorf("Expected query 'key1=value1&key2=value2', got %q", r.URL.RawQuery)
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tool := NewHTTPTool()
	out, err := tool.Execute(context.Background(), fmt.Sprintf(`{"url":"%s?key1=value1&key2=value2"}`, srv.URL))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("Expected 'ok' in output, got %q", out)
	}
}

func TestHTTPTool_BodyWithoutMethod(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}
		if r.ContentLength > 0 {
			t.Errorf("Expected no body for GET request")
		}
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tool := NewHTTPTool()
	out, err := tool.Execute(context.Background(), fmt.Sprintf(`{
		"url":"%s",
		"body":"{\"should\":\"be ignored\"}"
	}`, srv.URL))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("Expected 'ok' in output, got %q", out)
	}
}

func TestHTTPTool_MethodCaseInsensitive(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		wantLen int
	}{
		{"lowercase get", "get", 3},
		{"uppercase GET", "GET", 3},
		{"mixed case Post", "Post", 3},
		{"lowercase post", "post", 3},
		{"uppercase POST", "POST", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("ok"))
			}))
			defer srv.Close()

			tool := NewHTTPTool()
			out, err := tool.Execute(context.Background(), fmt.Sprintf(`{
				"url":"%s",
				"method":"%s"
			}`, srv.URL, tt.method))
			if err != nil {
				t.Fatalf("Execute: %v", err)
			}
			if !strings.Contains(out, "ok") {
				t.Errorf("Expected 'ok' in output, got %q", out)
			}
		})
	}
}

func TestHTTPTool_WhitespaceURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	tool := NewHTTPTool()
	out, err := tool.Execute(context.Background(), fmt.Sprintf(`{
		"url":"  %s  ",
		"method":"  GET  "
	}`, srv.URL))
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(out, "ok") {
		t.Errorf("Expected 'ok' in output, got %q", out)
	}
}
