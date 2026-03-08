package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/chromedp/cdproto/cdp"
	cdppage "github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/target"
	"github.com/chromedp/chromedp"
	"github.com/suifei/gopherpaw/internal/agent"
	"github.com/suifei/gopherpaw/pkg/logger"
	"go.uber.org/zap"
)

const (
	browserIdleTimeout = 60 * time.Minute
	// chromeBinEnv 是环境变量名，用于指定 Chrome 可执行文件路径
	chromeBinEnv = "CHROME_BIN"
	// runningInContainerEnv 是环境变量名，用于显式指示运行在容器中
	runningInContainerEnv = "GOPHERPAW_RUNNING_IN_CONTAINER"
)

// chromePaths 定义了各平台常见的 Chrome/Chromium 路径
var chromePaths = map[string][]string{
	"linux": {
		"/usr/bin/google-chrome",
		"/usr/bin/google-chrome-stable",
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/lib/chromium/chromium",
		"/snap/bin/chromium",
		"/usr/bin/google-chrome-beta",
		"/usr/bin/google-chrome-dev",
		// ChromeOS 特别路径
		"/opt/google/chrome/google-chrome",
	},
	"darwin": {
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/Applications/Google Chrome Beta.app/Contents/MacOS/Google Chrome Beta",
		"/Applications/Google Chrome Dev.app/Contents/MacOS/Google Chrome Dev",
		"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
	},
	"windows": {
		"C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe",
		"C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe",
		"C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe",
		"C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe",
		"C:\\Program Files\\Chromium\\Application\\chrome.exe",
	},
}

// findChromeExecutable 查找系统中已安装的 Chrome/Chromium 浏览器。
// 按优先级顺序检查：
// 1. 环境变量 CHROME_BIN
// 2. 系统常见路径
// 返回找到的第一个有效可执行文件路径，如果未找到则返回空字符串。
func findChromeExecutable() string {
	// 1. 检查环境变量
	if envPath := os.Getenv(chromeBinEnv); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			logger.L().Debug("using chrome from environment variable",
				zap.String("env", chromeBinEnv),
				zap.String("path", envPath))
			return envPath
		}
		logger.L().Warn("chrome path from environment variable not accessible",
			zap.String("env", chromeBinEnv),
			zap.String("path", envPath))
	}

	// 2. 扫描系统常见路径
	platform := runtime.GOOS
	candidates, ok := chromePaths[platform]
	if !ok {
		// 未知平台，尝试 Linux 路径作为回退
		candidates = chromePaths["linux"]
	}

	for _, path := range candidates {
		// Windows 路径可能包含环境变量
		if strings.Contains(path, "%") {
			path = os.ExpandEnv(path)
		}
		if fileInfo, err := os.Stat(path); err == nil {
			// 确保是可执行文件（不是目录）
			if !fileInfo.IsDir() {
				logger.L().Debug("found chrome in system path", zap.String("path", path))
				return path
			}
		}
	}

	logger.L().Debug("no chrome executable found in system paths")
	return ""
}

// isRunningInContainer 检测当前进程是否运行在容器环境（Docker/Kubernetes）中。
// 按优先级检查：
// 1. 环境变量 GOPHERPAW_RUNNING_IN_CONTAINER（设为 1/true/yes）
// 2. /.dockerenv 文件存在
// 3. /proc/1/cgroup 包含 "docker" 或 "kubepods"
func isRunningInContainer() bool {
	// 1. 检查环境变量（显式设置）
	envVal := strings.ToLower(os.Getenv(runningInContainerEnv))
	if envVal == "1" || envVal == "true" || envVal == "yes" {
		logger.L().Debug("detected container environment from env variable",
			zap.String("env", runningInContainerEnv))
		return true
	}

	// 2. 检查 /.dockerenv 文件
	if _, err := os.Stat("/.dockerenv"); err == nil {
		logger.L().Debug("detected container environment from /.dockerenv")
		return true
	}

	// 3. 检查 /proc/1/cgroup
	cgroupData, err := os.ReadFile("/proc/1/cgroup")
	if err == nil {
		content := strings.ToLower(string(cgroupData))
		if strings.Contains(content, "docker") || strings.Contains(content, "kubepods") || strings.Contains(content, "containerd") {
			logger.L().Debug("detected container environment from /proc/1/cgroup")
			return true
		}
	}

	return false
}

type pageContext struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type browserState struct {
	mu          sync.Mutex
	allocCtx    context.Context
	allocCancel context.CancelFunc
	browserCtx  context.Context
	browserCanc context.CancelFunc
	pages       map[string]*pageContext
	lastUse     time.Time
	stopCh      chan struct{}
	running     bool
}

var globalBrowser = &browserState{
	pages:  make(map[string]*pageContext),
	stopCh: make(chan struct{}),
}

func (bs *browserState) ensureBrowser(headless bool) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.lastUse = time.Now()

	if bs.running {
		return nil
	}

	// 构建浏览器选项
	opts := []chromedp.ExecAllocatorOption{
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
		// 禁用 devshm 使用（容器环境可能需要）
		chromedp.Flag("disable-dev-shm-usage", true),
		// 设置浏览器语言为简体中文（操作系统级别的区域设置）
		chromedp.Flag("lang", "zh-CN"),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	}

	// 1. 尝试查找系统浏览器
	chromePath := findChromeExecutable()
	if chromePath != "" {
		opts = append(opts, chromedp.ExecPath(chromePath))
		logger.L().Info("using detected chrome executable",
			zap.String("path", chromePath))
	} else {
		logger.L().Info("no chrome executable detected, using system default")
	}

	// 2. 容器环境特殊处理
	if isRunningInContainer() {
		opts = append(opts,
			chromedp.Flag("no-sandbox", true),
			chromedp.Flag("disable-setuid-sandbox", true),
		)
		logger.L().Info("container environment detected, added no-sandbox flags")
	}

	// 添加默认选项（必须在自定义选项之后）
	opts = append(chromedp.DefaultExecAllocatorOptions[:], opts...)

	bs.allocCtx, bs.allocCancel = chromedp.NewExecAllocator(context.Background(), opts...)
	bs.browserCtx, bs.browserCanc = chromedp.NewContext(bs.allocCtx)

	if err := chromedp.Run(bs.browserCtx); err != nil {
		bs.allocCancel()
		return fmt.Errorf("launch browser: %w", err)
	}

	bs.pages = make(map[string]*pageContext)
	bs.running = true
	bs.stopCh = make(chan struct{})

	go bs.idleWatcher()

	logger.L().Info("browser started (chromedp)",
		zap.Bool("headless", headless),
		zap.String("chrome_path", chromePath),
		zap.Bool("in_container", isRunningInContainer()))
	return nil
}

func (bs *browserState) idleWatcher() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			bs.mu.Lock()
			if bs.running && time.Since(bs.lastUse) > browserIdleTimeout {
				bs.closeLocked()
				bs.mu.Unlock()
				logger.L().Info("browser auto-closed after idle timeout")
				return
			}
			bs.mu.Unlock()
		case <-bs.stopCh:
			return
		}
	}
}

func (bs *browserState) closeLocked() {
	if !bs.running {
		return
	}
	for _, pc := range bs.pages {
		pc.cancel()
	}
	bs.pages = make(map[string]*pageContext)
	bs.browserCanc()
	bs.allocCancel()
	bs.running = false
	select {
	case <-bs.stopCh:
	default:
		close(bs.stopCh)
	}
}

func (bs *browserState) close() {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.closeLocked()
}

func (bs *browserState) getPage(id string) *pageContext {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.lastUse = time.Now()
	return bs.pages[id]
}

func (bs *browserState) setPage(id string, pc *pageContext) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	bs.lastUse = time.Now()
	bs.pages[id] = pc
}

func (bs *browserState) removePage(id string) {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	if pc, ok := bs.pages[id]; ok {
		pc.cancel()
		delete(bs.pages, id)
	}
}

func (bs *browserState) pageIDs() []string {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	ids := make([]string, 0, len(bs.pages))
	for id := range bs.pages {
		ids = append(ids, id)
	}
	return ids
}

func (bs *browserState) getBrowserCtx() context.Context {
	bs.mu.Lock()
	defer bs.mu.Unlock()
	return bs.browserCtx
}

type browserArgs struct {
	Action   string `json:"action"`
	URL      string `json:"url,omitempty"`
	PageID   string `json:"page_id,omitempty"`
	Selector string `json:"selector,omitempty"`
	Text     string `json:"text,omitempty"`
	Code     string `json:"code,omitempty"`
	Path     string `json:"path,omitempty"`
	FullPage bool   `json:"full_page,omitempty"`
	Width    int    `json:"width,omitempty"`
	Height   int    `json:"height,omitempty"`
	Headed   bool   `json:"headed,omitempty"`
	Wait     int    `json:"wait,omitempty"`
	// New fields for additional actions
	Key        string   `json:"key,omitempty"`         // press_key action
	Accept     bool     `json:"accept,omitempty"`      // handle_dialog action
	PromptText string   `json:"prompt_text,omitempty"` // handle_dialog action
	FilePaths  []string `json:"file_paths,omitempty"`  // file_upload action
	Values     []string `json:"values,omitempty"`      // select_option action
	StartX     int      `json:"start_x,omitempty"`     // drag action
	StartY     int      `json:"start_y,omitempty"`     // drag action
	EndX       int      `json:"end_x,omitempty"`       // drag action
	EndY       int      `json:"end_y,omitempty"`       // drag action
}

// BrowserTool provides browser automation via Chrome DevTools Protocol (chromedp).
type BrowserTool struct{}

func (t *BrowserTool) Name() string { return "browser_use" }

func (t *BrowserTool) Description() string {
	return "Browser automation tool. Use action parameter to control: start, stop, close, open, navigate, navigate_back, screenshot, click, type, hover, eval, snapshot, pdf, tabs, wait_for, resize, press_key, handle_dialog, file_upload, select_option, drag, scroll."
}

func (t *BrowserTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action":      map[string]any{"type": "string", "description": "Action to perform: start, stop, close, open, navigate, navigate_back, screenshot, click, type, hover, eval, snapshot, pdf, tabs, wait_for, resize, press_key, handle_dialog, file_upload, select_option, drag, scroll"},
			"url":         map[string]any{"type": "string", "description": "URL for open/navigate actions"},
			"page_id":     map[string]any{"type": "string", "description": "Page identifier (default: 'default')"},
			"selector":    map[string]any{"type": "string", "description": "CSS selector for click/type/hover/wait_for/file_upload/select_option"},
			"text":        map[string]any{"type": "string", "description": "Text to type"},
			"code":        map[string]any{"type": "string", "description": "JavaScript code for eval action"},
			"path":        map[string]any{"type": "string", "description": "File path for screenshot/pdf output"},
			"full_page":   map[string]any{"type": "boolean", "description": "Capture full page screenshot"},
			"width":       map[string]any{"type": "integer", "description": "Viewport width for resize"},
			"height":      map[string]any{"type": "integer", "description": "Viewport height for resize"},
			"headed":      map[string]any{"type": "boolean", "description": "Run browser in headed (visible) mode"},
			"wait":        map[string]any{"type": "integer", "description": "Wait time in milliseconds"},
			"key":         map[string]any{"type": "string", "description": "Key to press for press_key action (e.g., 'Enter', 'Tab', 'Escape', 'ArrowDown')"},
			"accept":      map[string]any{"type": "boolean", "description": "Whether to accept the dialog for handle_dialog action"},
			"prompt_text": map[string]any{"type": "string", "description": "Text to enter in prompt dialog for handle_dialog action"},
			"file_paths":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "File paths to upload for file_upload action"},
			"values":      map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Values to select for select_option action"},
			"start_x":     map[string]any{"type": "integer", "description": "Start X coordinate for drag action"},
			"start_y":     map[string]any{"type": "integer", "description": "Start Y coordinate for drag action"},
			"end_x":       map[string]any{"type": "integer", "description": "End X coordinate for drag action"},
			"end_y":       map[string]any{"type": "integer", "description": "End Y coordinate for drag action"},
		},
		"required": []string{"action"},
	}
}

func (t *BrowserTool) Execute(ctx context.Context, arguments string) (string, error) {
	result, err := t.ExecuteRich(ctx, arguments)
	if err != nil {
		return "", err
	}
	return result.Text, nil
}

func (t *BrowserTool) ExecuteRich(ctx context.Context, arguments string) (*agent.ToolResult, error) {
	var args browserArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return nil, fmt.Errorf("parse arguments: %w", err)
	}
	if args.PageID == "" {
		args.PageID = "default"
	}

	switch strings.ToLower(args.Action) {
	case "start":
		return t.doStart(args)
	case "stop":
		return t.doStop()
	case "close":
		return t.doClose(args)
	case "open", "navigate":
		return t.doOpen(args)
	case "navigate_back":
		return t.doNavigateBack(args)
	case "screenshot", "take_screenshot":
		return t.doScreenshot(args)
	case "click":
		return t.doClick(args)
	case "type":
		return t.doType(args)
	case "hover":
		return t.doHover(args)
	case "eval", "evaluate", "run_code":
		return t.doEval(args)
	case "snapshot":
		return t.doSnapshot(args)
	case "pdf":
		return t.doPDF(args)
	case "tabs":
		return t.doTabs()
	case "wait_for":
		return t.doWaitFor(args)
	case "resize":
		return t.doResize(args)
	case "press_key":
		return t.doPressKey(args)
	case "handle_dialog":
		return t.doHandleDialog(args)
	case "file_upload":
		return t.doFileUpload(args)
	case "select_option":
		return t.doSelectOption(args)
	case "drag":
		return t.doDrag(args)
	case "scroll":
		return t.doScroll(args)
	default:
		return nil, fmt.Errorf("unknown browser action: %s", args.Action)
	}
}

func (t *BrowserTool) doStart(args browserArgs) (*agent.ToolResult, error) {
	headless := !args.Headed
	if err := globalBrowser.ensureBrowser(headless); err != nil {
		return nil, err
	}
	mode := "headless"
	if !headless {
		mode = "headed"
	}
	return &agent.ToolResult{Text: fmt.Sprintf("Browser started in %s mode", mode)}, nil
}

func (t *BrowserTool) doStop() (*agent.ToolResult, error) {
	globalBrowser.close()
	return &agent.ToolResult{Text: "Browser stopped"}, nil
}

func (t *BrowserTool) doOpen(args browserArgs) (*agent.ToolResult, error) {
	if args.URL == "" {
		return nil, fmt.Errorf("url is required for open action")
	}
	if err := globalBrowser.ensureBrowser(true); err != nil {
		return nil, err
	}

	bCtx := globalBrowser.getBrowserCtx()
	tabCtx, tabCancel := chromedp.NewContext(bCtx)

	if err := chromedp.Run(tabCtx, chromedp.Navigate(args.URL)); err != nil {
		tabCancel()
		return nil, fmt.Errorf("open page: %w", err)
	}

	globalBrowser.setPage(args.PageID, &pageContext{ctx: tabCtx, cancel: tabCancel})

	var title string
	_ = chromedp.Run(tabCtx, chromedp.Title(&title))

	return &agent.ToolResult{Text: fmt.Sprintf("Opened %s (title: %s, page_id: %s)", args.URL, title, args.PageID)}, nil
}

func (t *BrowserTool) doNavigateBack(args browserArgs) (*agent.ToolResult, error) {
	pc := globalBrowser.getPage(args.PageID)
	if pc == nil {
		return nil, fmt.Errorf("page %q not found, use open first", args.PageID)
	}
	if err := chromedp.Run(pc.ctx, chromedp.NavigateBack()); err != nil {
		return nil, fmt.Errorf("navigate back: %w", err)
	}
	return &agent.ToolResult{Text: "Navigated back"}, nil
}

func (t *BrowserTool) doScreenshot(args browserArgs) (*agent.ToolResult, error) {
	pc := globalBrowser.getPage(args.PageID)
	if pc == nil {
		return nil, fmt.Errorf("page %q not found, use open first", args.PageID)
	}

	outPath := args.Path
	if outPath == "" {
		outPath = filepath.Join(os.TempDir(), fmt.Sprintf("browser_screenshot_%d.png", time.Now().UnixMilli()))
	}

	var buf []byte
	var action chromedp.Action
	if args.FullPage {
		action = chromedp.FullScreenshot(&buf, 90)
	} else {
		action = chromedp.CaptureScreenshot(&buf)
	}
	if err := chromedp.Run(pc.ctx, action); err != nil {
		return nil, fmt.Errorf("screenshot: %w", err)
	}
	if err := os.WriteFile(outPath, buf, 0644); err != nil {
		return nil, fmt.Errorf("save screenshot: %w", err)
	}

	return &agent.ToolResult{
		Text: fmt.Sprintf("Screenshot saved to %s", outPath),
		Attachments: []agent.Attachment{{
			FilePath: outPath,
			MimeType: "image/png",
			FileName: filepath.Base(outPath),
		}},
	}, nil
}

func (t *BrowserTool) doClick(args browserArgs) (*agent.ToolResult, error) {
	pc := globalBrowser.getPage(args.PageID)
	if pc == nil {
		return nil, fmt.Errorf("page %q not found", args.PageID)
	}
	if args.Selector == "" {
		return nil, fmt.Errorf("selector is required for click")
	}

	logger.L().Debug("browser click", zap.String("selector", args.Selector))

	// 设置超时上下文，避免无限等待
	ctx, cancel := context.WithTimeout(pc.ctx, 30*time.Second)
	defer cancel()

	// 先等待元素可见，再执行点击
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(args.Selector, chromedp.ByQuery),
		chromedp.Click(args.Selector, chromedp.ByQuery),
	); err != nil {
		return nil, fmt.Errorf("click %q: %w", args.Selector, err)
	}

	logger.L().Debug("browser click done", zap.String("selector", args.Selector))
	return &agent.ToolResult{Text: fmt.Sprintf("Clicked %s", args.Selector)}, nil
}

func (t *BrowserTool) doType(args browserArgs) (*agent.ToolResult, error) {
	pc := globalBrowser.getPage(args.PageID)
	if pc == nil {
		return nil, fmt.Errorf("page %q not found", args.PageID)
	}
	if args.Selector == "" {
		return nil, fmt.Errorf("selector is required for type")
	}
	if err := chromedp.Run(pc.ctx,
		chromedp.Click(args.Selector, chromedp.ByQuery),
		chromedp.Clear(args.Selector, chromedp.ByQuery),
		chromedp.SendKeys(args.Selector, args.Text, chromedp.ByQuery),
	); err != nil {
		return nil, fmt.Errorf("type into %q: %w", args.Selector, err)
	}
	return &agent.ToolResult{Text: fmt.Sprintf("Typed into %s", args.Selector)}, nil
}

func (t *BrowserTool) doHover(args browserArgs) (*agent.ToolResult, error) {
	pc := globalBrowser.getPage(args.PageID)
	if pc == nil {
		return nil, fmt.Errorf("page %q not found", args.PageID)
	}
	if args.Selector == "" {
		return nil, fmt.Errorf("selector is required for hover")
	}
	if err := chromedp.Run(pc.ctx,
		chromedp.MouseClickXY(0, 0), // reset hover
		chromedp.WaitVisible(args.Selector, chromedp.ByQuery),
	); err != nil {
		logger.L().Debug("hover wait", zap.Error(err))
	}
	var nodes []*cdp.Node
	if err := chromedp.Run(pc.ctx,
		chromedp.Nodes(args.Selector, &nodes, chromedp.ByQuery),
	); err != nil || len(nodes) == 0 {
		return nil, fmt.Errorf("find element %q: %w", args.Selector, err)
	}
	if err := chromedp.Run(pc.ctx, chromedp.MouseClickNode(nodes[0])); err != nil {
		logger.L().Debug("hover click fallback", zap.Error(err))
	}
	return &agent.ToolResult{Text: fmt.Sprintf("Hovered over %s", args.Selector)}, nil
}

func (t *BrowserTool) doEval(args browserArgs) (*agent.ToolResult, error) {
	pc := globalBrowser.getPage(args.PageID)
	if pc == nil {
		return nil, fmt.Errorf("page %q not found", args.PageID)
	}
	if args.Code == "" {
		return nil, fmt.Errorf("code is required for eval")
	}
	var result interface{}
	if err := chromedp.Run(pc.ctx, chromedp.Evaluate(args.Code, &result)); err != nil {
		return nil, fmt.Errorf("eval: %w", err)
	}
	out, _ := json.Marshal(result)
	return &agent.ToolResult{Text: string(out)}, nil
}

func (t *BrowserTool) doSnapshot(args browserArgs) (*agent.ToolResult, error) {
	pc := globalBrowser.getPage(args.PageID)
	if pc == nil {
		return nil, fmt.Errorf("page %q not found", args.PageID)
	}
	var html string
	if err := chromedp.Run(pc.ctx, chromedp.OuterHTML("html", &html, chromedp.ByQuery)); err != nil {
		return nil, fmt.Errorf("snapshot: %w", err)
	}
	const maxLen = 8000
	if len(html) > maxLen {
		html = html[:maxLen] + "\n... [truncated]"
	}
	return &agent.ToolResult{Text: html}, nil
}

func (t *BrowserTool) doPDF(args browserArgs) (*agent.ToolResult, error) {
	pc := globalBrowser.getPage(args.PageID)
	if pc == nil {
		return nil, fmt.Errorf("page %q not found", args.PageID)
	}
	outPath := args.Path
	if outPath == "" {
		outPath = filepath.Join(os.TempDir(), fmt.Sprintf("browser_pdf_%d.pdf", time.Now().UnixMilli()))
	}

	var buf []byte
	if err := chromedp.Run(pc.ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		buf, _, err = cdppage.PrintToPDF().Do(ctx)
		return err
	})); err != nil {
		return nil, fmt.Errorf("pdf: %w", err)
	}
	if err := os.WriteFile(outPath, buf, 0644); err != nil {
		return nil, fmt.Errorf("save pdf: %w", err)
	}
	return &agent.ToolResult{
		Text: fmt.Sprintf("PDF saved to %s", outPath),
		Attachments: []agent.Attachment{{
			FilePath: outPath,
			MimeType: "application/pdf",
			FileName: filepath.Base(outPath),
		}},
	}, nil
}

func (t *BrowserTool) doTabs() (*agent.ToolResult, error) {
	ids := globalBrowser.pageIDs()
	if len(ids) == 0 {
		return &agent.ToolResult{Text: "No open tabs"}, nil
	}

	bCtx := globalBrowser.getBrowserCtx()
	var lines []string
	for _, id := range ids {
		pc := globalBrowser.getPage(id)
		if pc == nil {
			continue
		}
		var title, location string
		_ = chromedp.Run(pc.ctx, chromedp.Title(&title))
		_ = chromedp.Run(pc.ctx, chromedp.Location(&location))
		lines = append(lines, fmt.Sprintf("- %s: %s (%s)", id, title, location))
	}

	// Also list targets from the browser for completeness
	if bCtx != nil {
		targets, err := chromedp.Targets(bCtx)
		if err == nil {
			for _, ti := range targets {
				if ti.Type == "page" {
					found := false
					for _, id := range ids {
						pc := globalBrowser.getPage(id)
						if pc != nil {
							tid := chromedp.FromContext(pc.ctx)
							if tid != nil && tid.Target != nil && tid.Target.TargetID == ti.TargetID {
								found = true
								break
							}
						}
					}
					if !found {
						lines = append(lines, fmt.Sprintf("- (untracked): %s (%s)", ti.Title, ti.URL))
					}
				}
			}
		}
	}

	return &agent.ToolResult{Text: "Open tabs:\n" + strings.Join(lines, "\n")}, nil
}

func (t *BrowserTool) doWaitFor(args browserArgs) (*agent.ToolResult, error) {
	pc := globalBrowser.getPage(args.PageID)
	if pc == nil {
		return nil, fmt.Errorf("page %q not found", args.PageID)
	}
	if args.Wait > 0 {
		time.Sleep(time.Duration(args.Wait) * time.Millisecond)
		return &agent.ToolResult{Text: fmt.Sprintf("Waited %dms", args.Wait)}, nil
	}
	if args.Selector != "" {
		tCtx, tCancel := context.WithTimeout(pc.ctx, 30*time.Second)
		defer tCancel()
		if err := chromedp.Run(tCtx, chromedp.WaitVisible(args.Selector, chromedp.ByQuery)); err != nil {
			return nil, fmt.Errorf("wait for %q: %w", args.Selector, err)
		}
		return &agent.ToolResult{Text: fmt.Sprintf("Element %s found", args.Selector)}, nil
	}
	return nil, fmt.Errorf("wait_for requires selector or wait (ms)")
}

func (t *BrowserTool) doResize(args browserArgs) (*agent.ToolResult, error) {
	pc := globalBrowser.getPage(args.PageID)
	if pc == nil {
		return nil, fmt.Errorf("page %q not found", args.PageID)
	}
	w, h := int64(args.Width), int64(args.Height)
	if w <= 0 {
		w = 1280
	}
	if h <= 0 {
		h = 720
	}
	if err := chromedp.Run(pc.ctx, chromedp.EmulateViewport(w, h)); err != nil {
		return nil, fmt.Errorf("resize: %w", err)
	}
	return &agent.ToolResult{Text: fmt.Sprintf("Viewport resized to %dx%d", w, h)}, nil
}

// Suppress unused import warnings; these are used in doTabs via chromedp.Targets.
var (
	_ = (*target.Info)(nil)
	_ io.Reader

	_ agent.Tool         = (*BrowserTool)(nil)
	_ agent.RichExecutor = (*BrowserTool)(nil)
)

// doClose closes a specific page/tab.
func (t *BrowserTool) doClose(args browserArgs) (*agent.ToolResult, error) {
	pc := globalBrowser.getPage(args.PageID)
	if pc == nil {
		return nil, fmt.Errorf("page %q not found", args.PageID)
	}
	globalBrowser.removePage(args.PageID)
	return &agent.ToolResult{Text: fmt.Sprintf("Page %s closed", args.PageID)}, nil
}

// doPressKey sends a key press event to the page.
func (t *BrowserTool) doPressKey(args browserArgs) (*agent.ToolResult, error) {
	pc := globalBrowser.getPage(args.PageID)
	if pc == nil {
		return nil, fmt.Errorf("page %q not found", args.PageID)
	}
	if args.Key == "" {
		return nil, fmt.Errorf("key is required for press_key action")
	}

	// Map common key names to chromedp key constants
	key := args.Key
	if err := chromedp.Run(pc.ctx, chromedp.KeyEvent(key)); err != nil {
		return nil, fmt.Errorf("press key %q: %w", key, err)
	}
	return &agent.ToolResult{Text: fmt.Sprintf("Pressed key: %s", key)}, nil
}

// doHandleDialog handles JavaScript dialogs (alert, confirm, prompt).
func (t *BrowserTool) doHandleDialog(args browserArgs) (*agent.ToolResult, error) {
	pc := globalBrowser.getPage(args.PageID)
	if pc == nil {
		return nil, fmt.Errorf("page %q not found", args.PageID)
	}

	// Set up dialog handler
	chromedp.ListenTarget(pc.ctx, func(ev interface{}) {
		if e, ok := ev.(*cdppage.EventJavascriptDialogOpening); ok {
			logger.L().Debug("dialog opened", zap.String("type", e.Type.String()), zap.String("message", e.Message))
			go func() {
				err := chromedp.Run(pc.ctx,
					cdppage.HandleJavaScriptDialog(args.Accept).WithPromptText(args.PromptText),
				)
				if err != nil {
					logger.L().Warn("handle dialog failed", zap.Error(err))
				}
			}()
		}
	})

	action := "dismissed"
	if args.Accept {
		action = "accepted"
	}
	return &agent.ToolResult{Text: fmt.Sprintf("Dialog handler set to %s dialogs", action)}, nil
}

// doFileUpload uploads files to a file input element.
func (t *BrowserTool) doFileUpload(args browserArgs) (*agent.ToolResult, error) {
	pc := globalBrowser.getPage(args.PageID)
	if pc == nil {
		return nil, fmt.Errorf("page %q not found", args.PageID)
	}
	if args.Selector == "" {
		return nil, fmt.Errorf("selector is required for file_upload action")
	}
	if len(args.FilePaths) == 0 {
		return nil, fmt.Errorf("file_paths is required for file_upload action")
	}

	// Verify files exist
	for _, fp := range args.FilePaths {
		if _, err := os.Stat(fp); err != nil {
			return nil, fmt.Errorf("file not found: %s", fp)
		}
	}

	if err := chromedp.Run(pc.ctx, chromedp.SetUploadFiles(args.Selector, args.FilePaths, chromedp.ByQuery)); err != nil {
		return nil, fmt.Errorf("file upload to %q: %w", args.Selector, err)
	}
	return &agent.ToolResult{Text: fmt.Sprintf("Uploaded %d file(s) to %s", len(args.FilePaths), args.Selector)}, nil
}

// doSelectOption selects option(s) in a <select> element.
func (t *BrowserTool) doSelectOption(args browserArgs) (*agent.ToolResult, error) {
	pc := globalBrowser.getPage(args.PageID)
	if pc == nil {
		return nil, fmt.Errorf("page %q not found", args.PageID)
	}
	if args.Selector == "" {
		return nil, fmt.Errorf("selector is required for select_option action")
	}
	if len(args.Values) == 0 {
		return nil, fmt.Errorf("values is required for select_option action")
	}

	// Use JavaScript to select options by value
	js := fmt.Sprintf(`
		(function() {
			var sel = document.querySelector(%q);
			if (!sel) return 'selector not found';
			var values = %s;
			for (var i = 0; i < sel.options.length; i++) {
				sel.options[i].selected = values.includes(sel.options[i].value);
			}
			sel.dispatchEvent(new Event('change', { bubbles: true }));
			return 'ok';
		})()
	`, args.Selector, mustJSON(args.Values))

	var result string
	if err := chromedp.Run(pc.ctx, chromedp.Evaluate(js, &result)); err != nil {
		return nil, fmt.Errorf("select option: %w", err)
	}
	if result != "ok" {
		return nil, fmt.Errorf("select option: %s", result)
	}
	return &agent.ToolResult{Text: fmt.Sprintf("Selected %d option(s) in %s", len(args.Values), args.Selector)}, nil
}

// doDrag performs a drag-and-drop operation.
func (t *BrowserTool) doDrag(args browserArgs) (*agent.ToolResult, error) {
	pc := globalBrowser.getPage(args.PageID)
	if pc == nil {
		return nil, fmt.Errorf("page %q not found", args.PageID)
	}

	// Use mouse events to simulate drag
	if err := chromedp.Run(pc.ctx,
		chromedp.MouseClickXY(float64(args.StartX), float64(args.StartY), chromedp.ButtonLeft),
	); err != nil {
		return nil, fmt.Errorf("drag start: %w", err)
	}

	// Simulate drag by using JavaScript for more reliable results
	js := fmt.Sprintf(`
		(function() {
			var startEl = document.elementFromPoint(%d, %d);
			var endEl = document.elementFromPoint(%d, %d);
			if (!startEl) return 'start element not found';
			
			var dataTransfer = new DataTransfer();
			var dragStart = new DragEvent('dragstart', { bubbles: true, dataTransfer: dataTransfer });
			var dragOver = new DragEvent('dragover', { bubbles: true, dataTransfer: dataTransfer });
			var drop = new DragEvent('drop', { bubbles: true, dataTransfer: dataTransfer });
			var dragEnd = new DragEvent('dragend', { bubbles: true, dataTransfer: dataTransfer });
			
			startEl.dispatchEvent(dragStart);
			if (endEl) {
				endEl.dispatchEvent(dragOver);
				endEl.dispatchEvent(drop);
			}
			startEl.dispatchEvent(dragEnd);
			return 'ok';
		})()
	`, args.StartX, args.StartY, args.EndX, args.EndY)

	var result string
	if err := chromedp.Run(pc.ctx, chromedp.Evaluate(js, &result)); err != nil {
		return nil, fmt.Errorf("drag: %w", err)
	}
	return &agent.ToolResult{Text: fmt.Sprintf("Dragged from (%d,%d) to (%d,%d)", args.StartX, args.StartY, args.EndX, args.EndY)}, nil
}

// doScroll scrolls the page or an element.
func (t *BrowserTool) doScroll(args browserArgs) (*agent.ToolResult, error) {
	pc := globalBrowser.getPage(args.PageID)
	if pc == nil {
		return nil, fmt.Errorf("page %q not found", args.PageID)
	}

	// Default scroll amounts
	scrollX, scrollY := args.Width, args.Height
	if scrollX == 0 && scrollY == 0 {
		scrollY = 300 // Default vertical scroll
	}

	var js string
	if args.Selector != "" {
		// Scroll within element
		js = fmt.Sprintf(`
			(function() {
				var el = document.querySelector(%q);
				if (!el) return 'element not found';
				el.scrollBy(%d, %d);
				return 'ok';
			})()
		`, args.Selector, scrollX, scrollY)
	} else {
		// Scroll the page
		js = fmt.Sprintf(`window.scrollBy(%d, %d); 'ok'`, scrollX, scrollY)
	}

	var result string
	if err := chromedp.Run(pc.ctx, chromedp.Evaluate(js, &result)); err != nil {
		return nil, fmt.Errorf("scroll: %w", err)
	}
	if result != "ok" {
		return nil, fmt.Errorf("scroll: %s", result)
	}

	target := "page"
	if args.Selector != "" {
		target = args.Selector
	}
	return &agent.ToolResult{Text: fmt.Sprintf("Scrolled %s by (%d, %d)", target, scrollX, scrollY)}, nil
}

// mustJSON encodes a value to JSON, panicking on error (for internal use only).
func mustJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return string(b)
}
