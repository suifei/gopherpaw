package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDownloadModel_URL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("model data"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	ctx := context.Background()
	path, err := DownloadModel(ctx, server.URL+"/model.bin", SourceURL, "my-model.bin")
	if err != nil {
		t.Fatalf("DownloadModel() error = %v", err)
	}

	if !strings.Contains(path, "my-model.bin") {
		t.Errorf("path should contain my-model.bin, got %q", path)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("file should exist at %q", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}

	if string(content) != "model data" {
		t.Errorf("content = %q, want 'model data'", string(content))
	}

	os.RemoveAll(path)
}

func TestDownloadModel_HuggingFace(t *testing.T) {
	ctx := context.Background()
	_, err := DownloadModel(ctx, "some-model", SourceHuggingFace, "")
	if err == nil {
		t.Error("DownloadModel with HuggingFace source should return error")
	}

	if !strings.Contains(err.Error(), "huggingface") {
		t.Errorf("error should mention huggingface, got %v", err)
	}
}

func TestDownloadModel_ModelScope(t *testing.T) {
	ctx := context.Background()
	_, err := DownloadModel(ctx, "some-model", SourceModelScope, "")
	if err == nil {
		t.Error("DownloadModel with ModelScope source should return error")
	}

	if !strings.Contains(err.Error(), "modelscope") {
		t.Errorf("error should mention modelscope, got %v", err)
	}
}

func TestDownloadModel_UnknownSource(t *testing.T) {
	ctx := context.Background()
	_, err := DownloadModel(ctx, "http://example.com/model.bin", "unknown", "")
	if err == nil {
		t.Error("DownloadModel with unknown source should return error")
	}

	if !strings.Contains(err.Error(), "unknown source") {
		t.Errorf("error should mention unknown source, got %v", err)
	}
}

func TestDownloadModel_InvalidURL(t *testing.T) {
	ctx := context.Background()
	_, err := DownloadModel(ctx, "not-a-url", SourceURL, "")
	if err == nil {
		t.Error("DownloadModel with invalid URL should return error")
	}

	if !strings.Contains(err.Error(), "invalid URL") {
		t.Errorf("error should mention invalid URL, got %v", err)
	}
}

func TestDownloadModel_EmptyURL(t *testing.T) {
	ctx := context.Background()
	_, err := DownloadModel(ctx, "", SourceURL, "")
	if err == nil {
		t.Error("DownloadModel with empty URL should return error")
	}
}

func TestDownloadModel_AutoFilename(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("model data"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	ctx := context.Background()
	path, err := DownloadModel(ctx, server.URL+"/path/to/auto-model.bin?token=abc", SourceURL, "")
	if err != nil {
		t.Fatalf("DownloadModel() error = %v", err)
	}

	if !strings.Contains(path, "auto-model.bin") {
		t.Errorf("path should contain auto-model.bin, got %q", path)
	}

	os.RemoveAll(path)
}

func TestDownloadModel_WithQueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("model data"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	ctx := context.Background()
	path, err := DownloadModel(ctx, server.URL+"/file.bin?arg1=val1&arg2=val2", SourceURL, "")
	if err != nil {
		t.Fatalf("DownloadModel() error = %v", err)
	}

	if !strings.Contains(path, "file.bin") {
		t.Errorf("path should contain file.bin, got %q", path)
	}

	os.RemoveAll(path)
}

func TestDownloadModel_FilenameFromURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("model data"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	ctx := context.Background()
	path, err := DownloadModel(ctx, server.URL+"/custom-name.gguf", SourceURL, "")
	if err != nil {
		t.Fatalf("DownloadModel() error = %v", err)
	}

	if !strings.Contains(path, "custom-name.gguf") {
		t.Errorf("path should contain custom-name.gguf, got %q", path)
	}

	os.RemoveAll(path)
}

func TestDownloadFromURL_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("downloaded data"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	ctx := context.Background()
	path, err := DownloadFromURL(ctx, server.URL+"/myfile.bin")
	if err != nil {
		t.Fatalf("DownloadFromURL() error = %v", err)
	}

	if !strings.Contains(path, "myfile.bin") {
		t.Errorf("path should contain myfile.bin, got %q", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}

	if string(content) != "downloaded data" {
		t.Errorf("content = %q, want 'downloaded data'", string(content))
	}

	os.RemoveAll(path)
}

func TestDownloadFromURL_WithQueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("data"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	ctx := context.Background()
	path, err := DownloadFromURL(ctx, server.URL+"/file.bin?param=value")
	if err != nil {
		t.Fatalf("DownloadFromURL() error = %v", err)
	}

	if !strings.Contains(path, "file.bin") {
		t.Errorf("path should contain file.bin, got %q", path)
	}

	os.RemoveAll(path)
}

func TestDownloadFromURL_AutoFilename(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("data"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	ctx := context.Background()
	path, err := DownloadFromURL(ctx, server.URL+"/somefile")
	if err != nil {
		t.Fatalf("DownloadFromURL() error = %v", err)
	}

	expectedFilename := "somefile"
	if filepath.Base(path) != expectedFilename {
		t.Errorf("path base name should be %q, got %q", expectedFilename, filepath.Base(path))
	}

	os.RemoveAll(path)
}

func TestDownloadFromURL_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	ctx := context.Background()
	_, err := DownloadFromURL(ctx, server.URL+"/file.bin")
	if err == nil {
		t.Error("DownloadFromURL should return error for HTTP 404")
	}

	if !strings.Contains(err.Error(), "status 404") {
		t.Errorf("error should contain status 404, got %v", err)
	}
}

func TestDownloadFromURL_NetworkError(t *testing.T) {
	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	ctx := context.Background()
	_, err := DownloadFromURL(ctx, "http://invalid-host-that-does-not-exist:9999/file.bin")
	if err == nil {
		t.Error("DownloadFromURL should return error for network failure")
	}
}

func TestDownloadFromURL_ContextCancelled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	oldHome := os.Getenv("HOME")
	defer os.Setenv("HOME", oldHome)
	os.Setenv("HOME", tmpDir)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := DownloadFromURL(ctx, server.URL+"/file.bin")
	if err == nil {
		t.Error("DownloadFromURL should return error when context is cancelled")
	}
}

func TestDownloadFile_ContentDisposition(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Disposition", "attachment; filename=\"custom-name.bin\"")
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("model data"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	ctx := context.Background()
	path, err := downloadFile(ctx, server.URL+"/path", tmpDir, "")
	if err != nil {
		t.Fatalf("downloadFile() error = %v", err)
	}

	if !strings.Contains(path, "custom-name.bin") {
		t.Errorf("path should contain custom-name.bin from Content-Disposition, got %q", path)
	}

	os.RemoveAll(path)
}

func TestDownloadFile_EmptyFilename(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("data"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	ctx := context.Background()
	path, err := downloadFile(ctx, server.URL, tmpDir, "")
	if err != nil {
		t.Fatalf("downloadFile() error = %v", err)
	}

	expectedFilename := "downloaded"
	if !strings.Contains(filepath.Base(path), expectedFilename) {
		t.Errorf("path should contain %q, got %q", expectedFilename, filepath.Base(path))
	}

	os.RemoveAll(path)
}

func TestDownloadFile_WriteError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write([]byte("data"))
	}))
	defer server.Close()

	tmpDir := t.TempDir()

	ctx := context.Background()
	_, err := downloadFile(ctx, server.URL, tmpDir+"/nonexistent", "file.bin")
	if err == nil {
		t.Error("downloadFile should return error when parent directory doesn't exist")
	}
}
