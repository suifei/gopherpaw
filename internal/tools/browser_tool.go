package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

const browserIdleTimeout = 60 * time.Minute

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

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", headless),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("no-first-run", true),
		chromedp.Flag("no-default-browser-check", true),
	)

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

	logger.L().Info("browser started (chromedp)", zap.Bool("headless", headless))
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
}

// BrowserTool provides browser automation via Chrome DevTools Protocol (chromedp).
type BrowserTool struct{}

func (t *BrowserTool) Name() string { return "browser_use" }

func (t *BrowserTool) Description() string {
	return "Browser automation tool. Use action parameter to control: start, stop, open, navigate, navigate_back, screenshot, click, type, hover, eval, snapshot, pdf, tabs, wait_for, resize."
}

func (t *BrowserTool) Parameters() any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action":    map[string]any{"type": "string", "description": "Action to perform: start, stop, open, navigate, navigate_back, screenshot, click, type, hover, eval, snapshot, pdf, tabs, wait_for, resize"},
			"url":       map[string]any{"type": "string", "description": "URL for open/navigate actions"},
			"page_id":   map[string]any{"type": "string", "description": "Page identifier (default: 'default')"},
			"selector":  map[string]any{"type": "string", "description": "CSS selector for click/type/hover/wait_for"},
			"text":      map[string]any{"type": "string", "description": "Text to type"},
			"code":      map[string]any{"type": "string", "description": "JavaScript code for eval action"},
			"path":      map[string]any{"type": "string", "description": "File path for screenshot/pdf output"},
			"full_page": map[string]any{"type": "boolean", "description": "Capture full page screenshot"},
			"width":     map[string]any{"type": "integer", "description": "Viewport width for resize"},
			"height":    map[string]any{"type": "integer", "description": "Viewport height for resize"},
			"headed":    map[string]any{"type": "boolean", "description": "Run browser in headed (visible) mode"},
			"wait":      map[string]any{"type": "integer", "description": "Wait time in milliseconds"},
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
	if err := chromedp.Run(pc.ctx, chromedp.Click(args.Selector, chromedp.ByQuery)); err != nil {
		return nil, fmt.Errorf("click %q: %w", args.Selector, err)
	}
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
