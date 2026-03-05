package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/kbinani/screenshot"
	"github.com/suifei/gopherpaw/internal/agent"
)

type screenshotArgs struct {
	Path    string `json:"path,omitempty"`
	Display int    `json:"display,omitempty"`
}

// ScreenshotTool captures the desktop screen.
type ScreenshotTool struct{}

func (t *ScreenshotTool) Name() string { return "desktop_screenshot" }

func (t *ScreenshotTool) Description() string {
	return "Capture a screenshot of the desktop. Returns the saved image file path. Supports multiple displays."
}

func (t *ScreenshotTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":    map[string]any{"type": "string", "description": "Output file path (default: temp file)"},
			"display": map[string]any{"type": "integer", "description": "Display number to capture (default: 0 = primary, -1 = all displays combined)"},
		},
	}
}

func (t *ScreenshotTool) Execute(ctx context.Context, arguments string) (string, error) {
	result, err := t.ExecuteRich(ctx, arguments)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

func (t *ScreenshotTool) ExecuteRich(ctx context.Context, arguments string) (*agent.ToolResult, error) {
	var args screenshotArgs
	if arguments != "" {
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return nil, fmt.Errorf("parse arguments: %w", err)
		}
	}

	n := screenshot.NumActiveDisplays()
	if n == 0 {
		return nil, fmt.Errorf("no active displays found")
	}

	displayIdx := args.Display
	if displayIdx >= n {
		return nil, fmt.Errorf("display %d not found (available: 0-%d)", displayIdx, n-1)
	}

	outPath := args.Path
	if outPath == "" {
		outPath = filepath.Join(os.TempDir(), fmt.Sprintf("desktop_screenshot_%d.png", time.Now().UnixMilli()))
	}

	if displayIdx < 0 {
		displayIdx = 0
	}
	bounds := screenshot.GetDisplayBounds(displayIdx)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("capture screen: %w", err)
	}

	f, err := os.Create(outPath)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return nil, fmt.Errorf("encode png: %w", err)
	}

	return &agent.ToolResult{
		Text: fmt.Sprintf("Screenshot saved to %s (%dx%d)", outPath, bounds.Dx(), bounds.Dy()),
		Attachments: []agent.Attachment{{
			FilePath: outPath,
			MimeType: "image/png",
			FileName: filepath.Base(outPath),
		}},
	}, nil
}

var (
	_ agent.Tool         = (*ScreenshotTool)(nil)
	_ agent.RichExecutor = (*ScreenshotTool)(nil)
)
