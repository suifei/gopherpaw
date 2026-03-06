package tools

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func BenchmarkHTTPTool_GET(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	tool := NewHTTPTool()
	args, _ := json.Marshal(httpArgs{URL: server.URL, Method: "GET"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkHTTPTool_GET_WithHeaders(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	tool := NewHTTPTool()
	args, _ := json.Marshal(httpArgs{
		URL:     server.URL,
		Method:  "GET",
		Headers: map[string]string{"Content-Type": "application/json"},
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkHTTPTool_POST(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	tool := NewHTTPTool()
	args, _ := json.Marshal(httpArgs{
		URL:    server.URL,
		Method: "POST",
		Body:   `{"test": "data"}`,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkHTTPTool_POST_LargeBody(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	largeBody := `{"data": "` + string(make([]byte, 10000)) + `"}`

	tool := NewHTTPTool()
	args, _ := json.Marshal(httpArgs{
		URL:    server.URL,
		Method: "POST",
		Body:   largeBody,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkHTTPTool_PUT(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	tool := NewHTTPTool()
	args, _ := json.Marshal(httpArgs{
		URL:    server.URL,
		Method: "PUT",
		Body:   `{"test": "data"}`,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkHTTPTool_DELETE(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	tool := NewHTTPTool()
	args, _ := json.Marshal(httpArgs{
		URL:    server.URL,
		Method: "DELETE",
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkHTTPTool_PATCH(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	tool := NewHTTPTool()
	args, _ := json.Marshal(httpArgs{
		URL:    server.URL,
		Method: "PATCH",
		Body:   `{"test": "data"}`,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkHTTPTool_LargeResponse(b *testing.B) {
	largeData := string(make([]byte, 1024*1024))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeData))
	}))
	defer server.Close()

	tool := NewHTTPTool()
	args, _ := json.Marshal(httpArgs{URL: server.URL, Method: "GET"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkHTTPTool_ErrorResponse(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	tool := NewHTTPTool()
	args, _ := json.Marshal(httpArgs{URL: server.URL, Method: "GET"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}

func BenchmarkHTTPTool_WithTimeout(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	tool := NewHTTPTool()
	tool.client.Timeout = 100 * time.Millisecond
	args, _ := json.Marshal(httpArgs{URL: server.URL, Method: "GET"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tool.Execute(context.Background(), string(args))
	}
}
