// Package llm provides model download from HuggingFace/ModelScope.
package llm

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DownloadSource identifies the model source.
const (
	SourceHuggingFace = "huggingface"
	SourceModelScope  = "modelscope"
	SourceURL         = "url"
)

// DownloadModel downloads a model file to ./models/ (current directory).
// For source=url: repoID is the direct file URL, backend is optional filename.
// For HuggingFace/ModelScope: simplified impl uses direct URL; full API support TBD.
func DownloadModel(ctx context.Context, repoID, source, backend string) (string, error) {
	baseDir := "./models"
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	var url string
	switch strings.ToLower(source) {
	case SourceURL, "":
		url = repoID
	case SourceHuggingFace:
		return "", fmt.Errorf("huggingface: use source=url with raw.githubusercontent.com/... for now")
	case SourceModelScope:
		return "", fmt.Errorf("modelscope: use source=url with direct file URL for now")
	default:
		return "", fmt.Errorf("unknown source %q", source)
	}
	if url == "" || !strings.HasPrefix(url, "http") {
		return "", fmt.Errorf("invalid URL")
	}
	filename := backend
	if filename == "" {
		filename = filepath.Base(strings.Split(url, "?")[0])
		if filename == "" || filename == "." {
			filename = "model"
		}
	}
	return downloadFile(ctx, url, baseDir, filename)
}

// DownloadFromURL downloads a file from the given URL to ./models/ (current directory).
func DownloadFromURL(ctx context.Context, url string) (string, error) {
	baseDir := "./models"
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}
	name := filepath.Base(strings.Split(url, "?")[0])
	if name == "" || name == "." {
		name = "downloaded"
	}
	return downloadFile(ctx, url, baseDir, name)
}

func downloadFile(ctx context.Context, url, baseDir, filename string) (string, error) {
	client := &http.Client{Timeout: 300 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}
	if filename == "" {
		if cd := resp.Header.Get("Content-Disposition"); cd != "" {
			if i := strings.Index(cd, "filename="); i >= 0 {
				filename = strings.Trim(cd[i+9:], "\"")
			}
		}
	}
	if filename == "" {
		filename = "downloaded"
	}
	outPath := filepath.Join(baseDir, filename)
	out, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		os.Remove(outPath)
		return "", fmt.Errorf("write: %w", err)
	}
	return outPath, nil
}
