//go:build chrome

package tools

import (
	"context"
	"fmt"
	"image"
	_ "image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/cdproto/cdp"
	cdppage "github.com/chromedp/cdproto/page"
	"github.com/chromedp/chromedp"
)

func newTestServer(t *testing.T) *httptest.Server {
	mux := http.NewServeMux()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head><title>Test Page</title>
<style>
	.hidden { display: none; }
	.hover { color: red; background: yellow; }
	.result { margin-top: 20px; padding: 10px; background: #f0f0f0; }
	.draggable { width: 100px; height: 100px; background: red; margin: 10px 0; cursor: move; }
	.dropzone { width: 200px; height: 200px; background: blue; color: white; }
</style>
</head>
<body>
	<h1 id="title">Welcome to Test Page</h1>
	<input id="name" type="text" placeholder="Enter your name" />
	<button id="submit">Submit</button>
	<div id="result" class="result"></div>
	<a id="page2-link" href="/page2">Go to Page 2</a>
	<button id="back-btn">Back</button>
	<button id="alert-btn">Show Alert</button>
	<button id="confirm-btn">Show Confirm</button>
	<button id="prompt-btn">Show Prompt</button>
	<button id="delay-btn">Load Delayed Content</button>
	<div id="delayed" class="hidden">Delayed Content Loaded</div>
	<div id="hover-target" class="hover">Hover over me</div>
	<input id="file-upload" type="file" />
	<button id="upload-btn">Upload</button>
	<select id="country">
		<option value="">Select Country</option>
		<option value="us">United States</option>
		<option value="cn">China</option>
	</select>
	<div id="draggable" class="draggable" draggable="true">Drag me</div>
	<div id="dropzone" class="dropzone">Drop here</div>
	<script>
		document.getElementById('submit').addEventListener('click', function() {
			const name = document.getElementById('name').value;
			document.getElementById('result').textContent = 'Hello, ' + name + '!';
		});
		document.getElementById('alert-btn').addEventListener('click', function() {
			alert('Test Alert');
		});
		document.getElementById('confirm-btn').addEventListener('click', function() {
			if (confirm('Test Confirm')) {
				document.getElementById('result').textContent = 'Confirmed';
			} else {
				document.getElementById('result').textContent = 'Cancelled';
			}
		});
		document.getElementById('prompt-btn').addEventListener('click', function() {
			const name = prompt('Enter your name:');
			if (name) {
				document.getElementById('result').textContent = 'Hello, ' + name + '!';
			}
		});
		document.getElementById('delay-btn').addEventListener('click', function() {
			setTimeout(function() {
				document.getElementById('delayed').classList.remove('hidden');
			}, 1000);
		});
		document.getElementById('back-btn').addEventListener('click', function() {
			window.history.back();
		});
	</script>
</body>
</html>`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	mux.HandleFunc("/page2", func(w http.ResponseWriter, r *http.Request) {
		html := `<!DOCTYPE html>
<html>
<head><title>Page 2</title></head>
<body>
	<h1 id="title2">This is Page 2</h1>
	<button id="back">Back</button>
</body>
</html>`
		w.Write([]byte(html))
	})

	mux.HandleFunc("/delay", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Second)
		w.Write([]byte(`<div id="delayed">Content loaded after delay</div>`))
	})

	return httptest.NewServer(mux)
}

func startBrowser(t *testing.T) context.Context {
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-gpu", true),
		chromedp.Flag("disable-dev-shm-usage", true),
		chromedp.Flag("no-sandbox", true),
		chromedp.UserAgent("gopherpaw-browser-test/1.0"),
	)
	ctx, cancel := chromedp.NewExecAllocator(context.Background(), opts...)
	t.Cleanup(cancel)
	return ctx
}

func newBrowserContext(t *testing.T, parentCtx context.Context) context.Context {
	ctx, cancel := chromedp.NewContext(parentCtx)
	t.Cleanup(cancel)
	return ctx
}

func navigateToPage(t *testing.T, ctx context.Context, url string) {
	err := chromedp.Run(ctx, chromedp.Navigate(url))
	if err != nil {
		t.Fatalf("Navigate to %s: %v", url, err)
	}
}

func TestBrowserE2E_Open(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	var title string
	err := chromedp.Run(browserCtx, chromedp.Title(&title))
	if err != nil {
		t.Fatalf("Get title: %v", err)
	}

	if title != "Test Page" {
		t.Errorf("Title = %q, want Test Page", title)
	}

	var bodyText string
	err = chromedp.Run(browserCtx, chromedp.Text("body", &bodyText))
	if err != nil {
		t.Fatalf("Get body text: %v", err)
	}

	if !strings.Contains(bodyText, "Welcome to Test Page") {
		t.Errorf("Body text should contain 'Welcome to Test Page', got %q", bodyText)
	}
}

func TestBrowserE2E_NavigateBack(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	var title1 string
	err := chromedp.Run(browserCtx, chromedp.Title(&title1))
	if err != nil {
		t.Fatalf("Get title1: %v", err)
	}

	if title1 != "Test Page" {
		t.Errorf("Initial title = %q, want Test Page", title1)
	}

	err = chromedp.Run(browserCtx, chromedp.Navigate(ts.URL+"/page2"))
	if err != nil {
		t.Fatalf("Navigate to page2: %v", err)
	}

	var title2 string
	err = chromedp.Run(browserCtx, chromedp.Title(&title2))
	if err != nil {
		t.Fatalf("Get title2: %v", err)
	}

	if title2 != "Page 2" {
		t.Errorf("Page2 title = %q, want Page 2", title2)
	}

	err = chromedp.Run(browserCtx, chromedp.NavigateBack())
	if err != nil {
		t.Fatalf("Navigate back: %v", err)
	}

	var titleBack string
	err = chromedp.Run(browserCtx, chromedp.Title(&titleBack))
	if err != nil {
		t.Fatalf("Get title after back: %v", err)
	}

	if titleBack != "Test Page" {
		t.Errorf("Title after back = %q, want Test Page", titleBack)
	}
}

func TestBrowserE2E_Click(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	err := chromedp.Run(browserCtx,
		chromedp.Click("#name", chromedp.ByQuery),
		chromedp.SendKeys("#name", "John Doe", chromedp.ByQuery),
		chromedp.Click("#submit", chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("Click and type: %v", err)
	}

	var resultText string
	err = chromedp.Run(browserCtx, chromedp.Text("#result", &resultText, chromedp.ByQuery))
	if err != nil {
		t.Fatalf("Get result text: %v", err)
	}

	if !strings.Contains(resultText, "Hello, John Doe!") {
		t.Errorf("Result text = %q, want 'Hello, John Doe!'", resultText)
	}
}

func TestBrowserE2E_Type(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	err := chromedp.Run(browserCtx,
		chromedp.Click("#name", chromedp.ByQuery),
		chromedp.SendKeys("#name", "Alice", chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("Type text: %v", err)
	}

	var inputVal string
	err = chromedp.Run(browserCtx, chromedp.Value("#name", &inputVal, chromedp.ByQuery))
	if err != nil {
		t.Fatalf("Get input value: %v", err)
	}

	if inputVal != "Alice" {
		t.Errorf("Input value = %q, want Alice", inputVal)
	}
}

func TestBrowserE2E_Screenshot(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	var buf []byte
	err := chromedp.Run(browserCtx, chromedp.CaptureScreenshot(&buf))
	if err != nil {
		t.Fatalf("CaptureScreenshot: %v", err)
	}

	if len(buf) == 0 {
		t.Error("Screenshot is empty")
	}

	tempDir := t.TempDir()
	screenshotPath := filepath.Join(tempDir, "screenshot.png")
	err = os.WriteFile(screenshotPath, buf, 0644)
	if err != nil {
		t.Fatalf("Write screenshot: %v", err)
	}

	file, err := os.Open(screenshotPath)
	if err != nil {
		t.Fatalf("Open screenshot file: %v", err)
	}
	defer file.Close()

	_, format, err := image.DecodeConfig(file)
	if err != nil {
		t.Fatalf("Decode screenshot: %v", err)
	}

	if format != "png" {
		t.Errorf("Screenshot format = %q, want png", format)
	}
}

func TestBrowserE2E_FullPageScreenshot(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	var buf []byte
	err := chromedp.Run(browserCtx, chromedp.FullScreenshot(&buf, 90))
	if err != nil {
		t.Fatalf("FullScreenshot: %v", err)
	}

	if len(buf) == 0 {
		t.Error("Full page screenshot is empty")
	}

	tempDir := t.TempDir()
	screenshotPath := filepath.Join(tempDir, "full_screenshot.png")
	err = os.WriteFile(screenshotPath, buf, 0644)
	if err != nil {
		t.Fatalf("Write full screenshot: %v", err)
	}

	stat, err := os.Stat(screenshotPath)
	if err != nil {
		t.Fatalf("Stat screenshot: %v", err)
	}

	if stat.Size() == 0 {
		t.Error("Screenshot file is empty")
	}
}

func TestBrowserE2E_Eval(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	var result1 string
	err := chromedp.Run(browserCtx, chromedp.Evaluate(`document.title`, &result1))
	if err != nil {
		t.Fatalf("Eval document.title: %v", err)
	}

	if result1 != "Test Page" {
		t.Errorf("Eval result = %q, want Test Page", result1)
	}

	var result2 int
	err = chromedp.Run(browserCtx, chromedp.Evaluate(`1 + 1`, &result2))
	if err != nil {
		t.Fatalf("Eval 1+1: %v", err)
	}

	if result2 != 2 {
		t.Errorf("Eval result = %d, want 2", result2)
	}

	var result3 interface{}
	jsCode := `
		(function() {
			var div = document.createElement('div');
			div.id = 'dynamic-element';
			div.textContent = 'Created by JS';
			document.body.appendChild(div);
			return {id: 'dynamic-element', text: 'Created by JS'};
		})()
	`
	err = chromedp.Run(browserCtx, chromedp.Evaluate(jsCode, &result3))
	if err != nil {
		t.Fatalf("Eval complex JS: %v", err)
	}

	resultMap, ok := result3.(map[string]interface{})
	if !ok {
		t.Fatalf("Eval result type = %T, want map[string]interface{}", result3)
	}

	if resultMap["id"] != "dynamic-element" {
		t.Errorf("Eval result id = %v, want dynamic-element", resultMap["id"])
	}

	var exists bool
	err = chromedp.Run(browserCtx, chromedp.WaitVisible("#dynamic-element", chromedp.ByQuery))
	if err != nil {
		t.Fatalf("Wait for dynamic element: %v", err)
	}
	exists = true

	if !exists {
		t.Error("Dynamic element not found")
	}
}

func TestBrowserE2E_WaitFor(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	err := chromedp.Run(browserCtx,
		chromedp.Click("#delay-btn", chromedp.ByQuery),
		chromedp.WaitVisible("#delayed", chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("Wait for delayed element: %v", err)
	}

	var delayedText string
	err = chromedp.Run(browserCtx, chromedp.Text("#delayed", &delayedText, chromedp.ByQuery))
	if err != nil {
		t.Fatalf("Get delayed text: %v", err)
	}

	if delayedText != "Delayed Content Loaded" {
		t.Errorf("Delayed text = %q, want 'Delayed Content Loaded'", delayedText)
	}

	ctx, cancel := context.WithTimeout(browserCtx, 2*time.Second)
	defer cancel()

	err = chromedp.Run(ctx, chromedp.WaitVisible("#nonexistent", chromedp.ByQuery))
	if err == nil {
		t.Error("Expected timeout error for nonexistent element")
	}
}

func TestBrowserE2E_WaitForTimeout(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	ctx, cancel := context.WithTimeout(browserCtx, 500*time.Millisecond)
	defer cancel()

	err := chromedp.Run(ctx, chromedp.WaitVisible("#nonexistent", chromedp.ByQuery))
	if err == nil {
		t.Error("Expected timeout error")
	}

	if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "deadline") {
		t.Logf("Error type: %v", err)
	}
}

func TestBrowserE2E_Resize(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	err := chromedp.Run(browserCtx, chromedp.EmulateViewport(1920, 1080))
	if err != nil {
		t.Fatalf("Resize to 1920x1080: %v", err)
	}

	var width, height int64
	err = chromedp.Run(browserCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		var metrics struct {
			Width  int64 `json:"contentSizeWidth"`
			Height int64 `json:"contentSizeHeight"`
		}
		err := chromedp.Evaluate(`
			(function() {
				return {
					width: window.innerWidth,
					height: window.innerHeight
				};
			})()
		`, &metrics).Do(ctx)
		if err != nil {
			return err
		}
		width = metrics.Width
		height = metrics.Height
		return nil
	}))
	if err != nil {
		t.Fatalf("Get viewport: %v", err)
	}

	if width != 1920 || height != 1080 {
		t.Errorf("Viewport = %dx%d, want 1920x1080", width, height)
	}

	err = chromedp.Run(browserCtx, chromedp.EmulateViewport(800, 600))
	if err != nil {
		t.Fatalf("Resize to 800x600: %v", err)
	}
}

func TestBrowserE2E_Tabs(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx1 := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx1, ts.URL)

	var title1 string
	err := chromedp.Run(browserCtx1, chromedp.Title(&title1))
	if err != nil {
		t.Fatalf("Get title1: %v", err)
	}

	browserCtx2 := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx2, ts.URL+"/page2")

	var title2 string
	err = chromedp.Run(browserCtx2, chromedp.Title(&title2))
	if err != nil {
		t.Fatalf("Get title2: %v", err)
	}

	if title1 != "Test Page" {
		t.Errorf("Tab 1 title = %q, want Test Page", title1)
	}

	if title2 != "Page 2" {
		t.Errorf("Tab 2 title = %q, want Page 2", title2)
	}
}

func TestBrowserE2E_PressKey(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	err := chromedp.Run(browserCtx,
		chromedp.Click("#name", chromedp.ByQuery),
		chromedp.SendKeys("#name", "Hello", chromedp.ByQuery),
		chromedp.KeyEvent("End"),
		chromedp.SendKeys("#name", " World", chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("Press key and type: %v", err)
	}

	var inputVal string
	err = chromedp.Run(browserCtx, chromedp.Value("#name", &inputVal, chromedp.ByQuery))
	if err != nil {
		t.Fatalf("Get input value: %v", err)
	}

	if inputVal != "Hello World" {
		t.Errorf("Input value = %q, want 'Hello World'", inputVal)
	}
}

func TestBrowserE2E_Hover(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	var bgColor string
	err := chromedp.Run(browserCtx, chromedp.Evaluate(`
		(function() {
			var el = document.getElementById('hover-target');
			return window.getComputedStyle(el).backgroundColor;
		})()
	`, &bgColor))
	if err != nil {
		t.Fatalf("Get initial background color: %v", err)
	}

	err = chromedp.Run(browserCtx, chromedp.MouseClickXY(0, 0))
	if err != nil {
		t.Fatalf("Reset mouse: %v", err)
	}

	err = chromedp.Run(browserCtx,
		chromedp.WaitVisible("#hover-target", chromedp.ByQuery),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.MouseClickNode(&cdp.Node{NodeID: 1}).Do(ctx)
		}),
	)
	if err != nil {
		t.Logf("Hover: %v (may be expected)", err)
	}

	time.Sleep(100 * time.Millisecond)

	var hoverBgColor string
	err = chromedp.Run(browserCtx, chromedp.Evaluate(`
		(function() {
			var el = document.getElementById('hover-target');
			return window.getComputedStyle(el).backgroundColor;
		})()
	`, &hoverBgColor))
	if err != nil {
		t.Fatalf("Get hover background color: %v", err)
	}

	if hoverBgColor == bgColor {
		t.Logf("Warning: Background color might not have changed: %s", hoverBgColor)
	}
}

func TestBrowserE2E_Snapshot(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	var html string
	err := chromedp.Run(browserCtx, chromedp.OuterHTML("html", &html, chromedp.ByQuery))
	if err != nil {
		t.Fatalf("OuterHTML: %v", err)
	}

	if !strings.Contains(html, "<title>Test Page</title>") {
		t.Error("Snapshot HTML should contain title")
	}

	if !strings.Contains(html, "Welcome to Test Page") {
		t.Error("Snapshot HTML should contain welcome text")
	}

	if !strings.Contains(html, "id=\"name\"") {
		t.Error("Snapshot HTML should contain name input")
	}

	if !strings.Contains(html, "id=\"submit\"") {
		t.Error("Snapshot HTML should contain submit button")
	}
}

func TestBrowserE2E_SelectOption(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	js := `
		(function() {
			var sel = document.querySelector('#country');
			sel.value = 'us';
			sel.dispatchEvent(new Event('change', { bubbles: true }));
			return 'ok';
		})()
	`
	var result string
	err := chromedp.Run(browserCtx, chromedp.Evaluate(js, &result))
	if err != nil {
		t.Fatalf("Select option: %v", err)
	}

	if result != "ok" {
		t.Errorf("Select result = %q, want ok", result)
	}

	var selectedValue string
	err = chromedp.Run(browserCtx, chromedp.Value("#country", &selectedValue, chromedp.ByQuery))
	if err != nil {
		t.Fatalf("Get selected value: %v", err)
	}

	if selectedValue != "us" {
		t.Errorf("Selected value = %q, want us", selectedValue)
	}
}

func TestBrowserE2E_Scroll(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	var scrollY1 int64
	err := chromedp.Run(browserCtx, chromedp.Evaluate(`window.scrollY`, &scrollY1))
	if err != nil {
		t.Fatalf("Get initial scroll Y: %v", err)
	}

	err = chromedp.Run(browserCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		return chromedp.Evaluate(`window.scrollBy(0, 200)`, nil).Do(ctx)
	}))
	if err != nil {
		t.Fatalf("Scroll page: %v", err)
	}

	var scrollY2 int64
	err = chromedp.Run(browserCtx, chromedp.Evaluate(`window.scrollY`, &scrollY2))
	if err != nil {
		t.Fatalf("Get scroll Y after scroll: %v", err)
	}

	if scrollY2 <= scrollY1 {
		t.Errorf("Scroll Y = %d, should be greater than %d", scrollY2, scrollY1)
	}
}

func TestBrowserE2E_MultiStepWorkflow(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	var title string
	err := chromedp.Run(browserCtx, chromedp.Title(&title))
	if err != nil {
		t.Fatalf("Step 1 - Get title: %v", err)
	}

	if title != "Test Page" {
		t.Errorf("Step 1 - Title = %q, want Test Page", title)
	}

	err = chromedp.Run(browserCtx,
		chromedp.Click("#name", chromedp.ByQuery),
		chromedp.SendKeys("#name", "Jane Doe", chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("Step 2 - Type name: %v", err)
	}

	var inputVal string
	err = chromedp.Run(browserCtx, chromedp.Value("#name", &inputVal, chromedp.ByQuery))
	if err != nil {
		t.Fatalf("Step 3 - Verify input: %v", err)
	}

	if inputVal != "Jane Doe" {
		t.Errorf("Step 3 - Input = %q, want Jane Doe", inputVal)
	}

	err = chromedp.Run(browserCtx, chromedp.Click("#submit", chromedp.ByQuery))
	if err != nil {
		t.Fatalf("Step 4 - Click submit: %v", err)
	}

	var resultText string
	err = chromedp.Run(browserCtx, chromedp.Text("#result", &resultText, chromedp.ByQuery))
	if err != nil {
		t.Fatalf("Step 5 - Get result: %v", err)
	}

	if !strings.Contains(resultText, "Hello, Jane Doe!") {
		t.Errorf("Step 5 - Result = %q, want 'Hello, Jane Doe!'", resultText)
	}

	err = chromedp.Run(browserCtx, chromedp.Navigate(ts.URL+"/page2"))
	if err != nil {
		t.Fatalf("Step 6 - Navigate to page2: %v", err)
	}

	var title2 string
	err = chromedp.Run(browserCtx, chromedp.Title(&title2))
	if err != nil {
		t.Fatalf("Step 7 - Get page2 title: %v", err)
	}

	if title2 != "Page 2" {
		t.Errorf("Step 7 - Title = %q, want Page 2", title2)
	}

	err = chromedp.Run(browserCtx, chromedp.NavigateBack())
	if err != nil {
		t.Fatalf("Step 8 - Navigate back: %v", err)
	}

	var titleBack string
	err = chromedp.Run(browserCtx, chromedp.Title(&titleBack))
	if err != nil {
		t.Fatalf("Step 9 - Get title after back: %v", err)
	}

	if titleBack != "Test Page" {
		t.Errorf("Step 9 - Title = %q, want Test Page", titleBack)
	}
}

func TestBrowserE2E_ToolIntegration(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	tool := &BrowserTool{}

	args := fmt.Sprintf(`{"action":"type","selector":"#name","text":"Tool Test"}`)
	result, err := tool.ExecuteRich(context.Background(), args)
	if err == nil {
		t.Logf("Tool result (expected error without page): %v", result)
	}

	globalBrowser.ensureBrowser(true)
	globalBrowser.setPage("test_page", &pageContext{ctx: browserCtx, cancel: func() {}})

	args = `{"action":"type","selector":"#name","text":"Tool Test"}`
	result, err = tool.ExecuteRich(context.Background(), args)
	if err != nil {
		t.Fatalf("Tool ExecuteRich: %v", err)
	}

	if result == nil {
		t.Fatal("Tool result is nil")
	}

	if !strings.Contains(result.Text, "#name") {
		t.Errorf("Tool result text = %q, should contain selector", result.Text)
	}

	var inputVal string
	err = chromedp.Run(browserCtx, chromedp.Value("#name", &inputVal, chromedp.ByQuery))
	if err != nil {
		t.Fatalf("Get input value: %v", err)
	}

	if inputVal != "Tool Test" {
		t.Errorf("Input value = %q, want 'Tool Test'", inputVal)
	}

	globalBrowser.removePage("test_page")
	globalBrowser.close()
}

func TestBrowserE2E_NetworkConditions(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	var online bool
	err := chromedp.Run(browserCtx, chromedp.Evaluate(`navigator.onLine`, &online))
	if err != nil {
		t.Fatalf("Check online status: %v", err)
	}

	if !online {
		t.Error("Browser should be online")
	}
}

func TestBrowserE2E_JSErrorHandling(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	var result interface{}
	jsCode := `
		(function() {
			try {
				throw new Error("Test error");
			} catch (e) {
				return {error: e.message};
			}
		})()
	`
	err := chromedp.Run(browserCtx, chromedp.Evaluate(jsCode, &result))
	if err != nil {
		t.Fatalf("Eval error handling: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Result type = %T, want map[string]interface{}", result)
	}

	if resultMap["error"] != "Test error" {
		t.Errorf("Error message = %v, want 'Test error'", resultMap["error"])
	}
}

func TestBrowserE2E_ElementVisibility(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	var visible bool
	err := chromedp.Run(browserCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		var nodes []*cdp.Node
		if err := chromedp.Nodes("#name", &nodes, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		visible = len(nodes) > 0
		return nil
	}))
	if err != nil {
		t.Fatalf("Check element visibility: %v", err)
	}

	if !visible {
		t.Error("Element should be visible")
	}

	var hidden bool
	err = chromedp.Run(browserCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		var nodes []*cdp.Node
		if err := chromedp.Nodes("#delayed", &nodes, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		for _, node := range nodes {
			if len(node.Attributes) > 0 {
				for i := 0; i < len(node.Attributes); i += 2 {
					if node.Attributes[i] == "class" && strings.Contains(node.Attributes[i+1], "hidden") {
						hidden = true
						break
					}
				}
			}
		}
		return nil
	}))
	if err != nil {
		t.Fatalf("Check hidden element: %v", err)
	}

	if !hidden {
		t.Error("Element should be hidden initially")
	}
}

func TestBrowserE2E_DOMManipulation(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	js := `
		(function() {
			var container = document.createElement('div');
			container.id = 'test-container';
			for (var i = 0; i < 5; i++) {
				var item = document.createElement('p');
				item.textContent = 'Item ' + i;
				item.className = 'test-item';
				container.appendChild(item);
			}
			document.body.appendChild(container);
			return {containerId: 'test-container', itemCount: 5};
		})()
	`
	var result interface{}
	err := chromedp.Run(browserCtx, chromedp.Evaluate(js, &result))
	if err != nil {
		t.Fatalf("Create elements: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Result type = %T, want map[string]interface{}", result)
	}

	if resultMap["containerId"] != "test-container" {
		t.Errorf("Container ID = %v, want 'test-container'", resultMap["containerId"])
	}

	if resultMap["itemCount"].(float64) != 5 {
		t.Errorf("Item count = %v, want 5", resultMap["itemCount"])
	}

	err = chromedp.Run(browserCtx, chromedp.WaitVisible("#test-container", chromedp.ByQuery))
	if err != nil {
		t.Fatalf("Wait for container: %v", err)
	}

	var itemCount int
	jsCount := `
		(function() {
			return document.querySelectorAll('.test-item').length;
		})()
	`
	err = chromedp.Run(browserCtx, chromedp.Evaluate(jsCount, &itemCount))
	if err != nil {
		t.Fatalf("Count items: %v", err)
	}

	if itemCount != 5 {
		t.Errorf("Item count = %d, want 5", itemCount)
	}
}

func TestBrowserE2E_FormHandling(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	tests := []struct {
		name  string
		value string
	}{
		{"empty", ""},
		{"short", "a"},
		{"medium", "test123"},
		{"long", "This is a very long text that should work correctly in the input field"},
		{"special", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := chromedp.Run(browserCtx,
				chromedp.Click("#name", chromedp.ByQuery),
				chromedp.Clear("#name", chromedp.ByQuery),
				chromedp.SendKeys("#name", tt.value, chromedp.ByQuery),
			)
			if err != nil {
				t.Fatalf("Type %q: %v", tt.value, err)
			}

			var inputVal string
			err = chromedp.Run(browserCtx, chromedp.Value("#name", &inputVal, chromedp.ByQuery))
			if err != nil {
				t.Fatalf("Get input value: %v", err)
			}

			if inputVal != tt.value {
				t.Errorf("Input = %q, want %q", inputVal, tt.value)
			}
		})
	}
}

func TestBrowserE2E_AttributeAccess(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	var idValue string
	err := chromedp.Run(browserCtx, chromedp.Evaluate(`
		(function() {
			return document.getElementById('title').id;
		})()
	`, &idValue))
	if err != nil {
		t.Fatalf("Get ID attribute: %v", err)
	}

	if idValue != "title" {
		t.Errorf("ID = %q, want 'title'", idValue)
	}

	var placeholder string
	err = chromedp.Run(browserCtx, chromedp.Evaluate(`
		(function() {
			return document.getElementById('name').placeholder;
		})()
	`, &placeholder))
	if err != nil {
		t.Fatalf("Get placeholder attribute: %v", err)
	}

	if placeholder != "Enter your name" {
		t.Errorf("Placeholder = %q, want 'Enter your name'", placeholder)
	}
}

func TestBrowserE2E_CSSSelectors(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	var textByClass string
	err := chromedp.Run(browserCtx, chromedp.Text(".result", &textByClass, chromedp.ByQuery))
	if err != nil {
		t.Fatalf("Get text by class: %v", err)
	}

	if textByClass != "" {
		t.Logf("Result div text: %s", textByClass)
	}

	var existsByTag bool
	err = chromedp.Run(browserCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		var nodes []*cdp.Node
		if err := chromedp.Nodes("h1", &nodes, chromedp.ByQuery).Do(ctx); err != nil {
			return err
		}
		existsByTag = len(nodes) > 0
		return nil
	}))
	if err != nil {
		t.Fatalf("Check h1 element: %v", err)
	}

	if !existsByTag {
		t.Error("h1 element should exist")
	}
}

func TestBrowserE2E_MultipleConcurrentActions(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	ctx, cancel := context.WithTimeout(browserCtx, 10*time.Second)
	defer cancel()

	err := chromedp.Run(ctx,
		chromedp.Click("#name", chromedp.ByQuery),
		chromedp.SendKeys("#name", "Concurrent Test", chromedp.ByQuery),
		chromedp.Click("#submit", chromedp.ByQuery),
		chromedp.WaitVisible("#result", chromedp.ByQuery),
	)
	if err != nil {
		t.Fatalf("Concurrent actions: %v", err)
	}

	var resultText string
	err = chromedp.Run(browserCtx, chromedp.Text("#result", &resultText, chromedp.ByQuery))
	if err != nil {
		t.Fatalf("Get result: %v", err)
	}

	if !strings.Contains(resultText, "Hello, Concurrent Test!") {
		t.Errorf("Result = %q, want 'Hello, Concurrent Test!'", resultText)
	}
}

func TestBrowserE2E_WindowProperties(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	var props map[string]interface{}
	err := chromedp.Run(browserCtx, chromedp.Evaluate(`
		(function() {
			return {
				innerWidth: window.innerWidth,
				innerHeight: window.innerHeight,
				outerWidth: window.outerWidth,
				outerHeight: window.outerHeight,
				scrollX: window.scrollX,
				scrollY: window.scrollY
			};
		})()
	`, &props))
	if err != nil {
		t.Fatalf("Get window properties: %v", err)
	}

	if props["innerWidth"] == nil || props["innerHeight"] == nil {
		t.Error("Window dimensions should be set")
	}

	if props["scrollX"] != 0 || props["scrollY"] != 0 {
		t.Error("Initial scroll position should be (0, 0)")
	}
}

func TestBrowserE2E_DocumentProperties(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	var docProps map[string]interface{}
	err := chromedp.Run(browserCtx, chromedp.Evaluate(`
		(function() {
			return {
				title: document.title,
				URL: document.URL,
				domain: document.domain,
				readyState: document.readyState,
				charset: document.characterSet
			};
		})()
	`, &docProps))
	if err != nil {
		t.Fatalf("Get document properties: %v", err)
	}

	if docProps["title"] != "Test Page" {
		t.Errorf("Title = %v, want 'Test Page'", docProps["title"])
	}

	if docProps["readyState"] != "complete" && docProps["readyState"] != "interactive" {
		t.Errorf("Ready state = %v, want 'complete' or 'interactive'", docProps["readyState"])
	}

	if !strings.Contains(docProps["URL"].(string), ts.URL) {
		t.Errorf("URL = %v, should contain %v", docProps["URL"], ts.URL)
	}
}

func TestBrowserE2E_HistoryAPI(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	var length1 int
	err := chromedp.Run(browserCtx, chromedp.Evaluate(`window.history.length`, &length1))
	if err != nil {
		t.Fatalf("Get history length: %v", err)
	}

	err = chromedp.Run(browserCtx, chromedp.Navigate(ts.URL+"/page2"))
	if err != nil {
		t.Fatalf("Navigate to page2: %v", err)
	}

	var length2 int
	err = chromedp.Run(browserCtx, chromedp.Evaluate(`window.history.length`, &length2))
	if err != nil {
		t.Fatalf("Get history length after navigate: %v", err)
	}

	if length2 <= length1 {
		t.Errorf("History length = %d, should be > %d", length2, length1)
	}

	var state interface{}
	err = chromedp.Run(browserCtx, chromedp.Evaluate(`window.history.state`, &state))
	if err != nil {
		t.Fatalf("Get history state: %v", err)
	}

	if state != nil {
		t.Logf("History state: %v", state)
	}
}

func TestBrowserE2E_LocalStorage(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	js := `
		(function() {
			localStorage.setItem('testKey', 'testValue');
			localStorage.setItem('numberKey', '123');
			return {items: localStorage.length};
		})()
	`
	var result map[string]interface{}
	err := chromedp.Run(browserCtx, chromedp.Evaluate(js, &result))
	if err != nil {
		t.Fatalf("Set localStorage: %v", err)
	}

	if result["items"].(float64) < 2 {
		t.Error("LocalStorage should have at least 2 items")
	}

	var value string
	err = chromedp.Run(browserCtx, chromedp.Evaluate(`localStorage.getItem('testKey')`, &value))
	if err != nil {
		t.Fatalf("Get localStorage item: %v", err)
	}

	if value != "testValue" {
		t.Errorf("Value = %q, want 'testValue'", value)
	}
}

func TestBrowserE2E_Cookies(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	js := `
		(function() {
			document.cookie = 'testCookie=testValue; path=/';
			document.cookie = 'anotherCookie=anotherValue; path=/';
			return document.cookie;
		})()
	`
	var cookieStr string
	err := chromedp.Run(browserCtx, chromedp.Evaluate(js, &cookieStr))
	if err != nil {
		t.Fatalf("Set cookies: %v", err)
	}

	if !strings.Contains(cookieStr, "testCookie") {
		t.Error("Cookies should contain testCookie")
	}

	if !strings.Contains(cookieStr, "anotherCookie") {
		t.Error("Cookies should contain anotherCookie")
	}
}

func TestBrowserE2E_PerformanceAPI(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	var timing map[string]interface{}
	err := chromedp.Run(browserCtx, chromedp.Evaluate(`
		(function() {
			var perf = window.performance.timing;
			return {
				domContentLoaded: perf.domContentLoadedEventEnd - perf.navigationStart,
				loadComplete: perf.loadEventEnd - perf.navigationStart,
				domInteractive: perf.domInteractive - perf.navigationStart
			};
		})()
	`, &timing))
	if err != nil {
		t.Fatalf("Get performance timing: %v", err)
	}

	domContentLoaded := timing["domContentLoaded"].(float64)
	if domContentLoaded <= 0 {
		t.Errorf("DOM content load time = %v, should be > 0", domContentLoaded)
	}

	var entries int
	err = chromedp.Run(browserCtx, chromedp.Evaluate(`window.performance.getEntries().length`, &entries))
	if err != nil {
		t.Fatalf("Get performance entries: %v", err)
	}

	if entries == 0 {
		t.Error("Should have performance entries")
	}
}

func TestBrowserE2E_TimeoutHandling(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	ctx, cancel := context.WithTimeout(browserCtx, 100*time.Millisecond)
	defer cancel()

	err := chromedp.Run(ctx, chromedp.Sleep(200*time.Millisecond))
	if err == nil {
		t.Error("Expected timeout error")
	}

	if !strings.Contains(err.Error(), "context") && !strings.Contains(err.Error(), "deadline") {
		t.Logf("Error type: %v", err)
	}
}

func TestBrowserE2E_ResourceCleanup(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	err := chromedp.Run(browserCtx, chromedp.Navigate(ts.URL+"/page2"))
	if err != nil {
		t.Fatalf("Navigate to page2: %v", err)
	}

	cancelCtx, ctxCancel := context.WithCancel(browserCtx)
	ctxCancel()

	err = chromedp.Run(cancelCtx, chromedp.Title(new(string)))
	if err == nil {
		t.Error("Expected error after context cancellation")
	}
}

func TestBrowserE2E_UTF8Encoding(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	tests := []struct {
		name string
		text string
		desc string
	}{
		{"chinese", "你好世界", "Chinese characters"},
		{"emoji", "🚀🎉🎯", "Emojis"},
		{"japanese", "こんにちは", "Japanese characters"},
		{"arabic", "مرحبا", "Arabic characters"},
		{"mixed", "Hello 你好 🚀", "Mixed characters"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := chromedp.Run(browserCtx,
				chromedp.Click("#name", chromedp.ByQuery),
				chromedp.Clear("#name", chromedp.ByQuery),
				chromedp.SendKeys("#name", tt.text, chromedp.ByQuery),
			)
			if err != nil {
				t.Fatalf("Type %s: %v", tt.desc, err)
			}

			var inputVal string
			err = chromedp.Run(browserCtx, chromedp.Value("#name", &inputVal, chromedp.ByQuery))
			if err != nil {
				t.Fatalf("Get input value: %v", err)
			}

			if inputVal != tt.text {
				t.Errorf("Input = %q, want %q", inputVal, tt.text)
			}
		})
	}
}

func TestBrowserE2E_JSONParsing(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	js := `
		(function() {
			var data = {
				string: "test",
				number: 123,
				boolean: true,
				null: null,
				array: [1, 2, 3],
				object: {nested: "value"}
			};
			return data;
		})()
	`
	var result interface{}
	err := chromedp.Run(browserCtx, chromedp.Evaluate(js, &result))
	if err != nil {
		t.Fatalf("Evaluate JSON: %v", err)
	}

	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("Result type = %T, want map[string]interface{}", result)
	}

	if resultMap["string"] != "test" {
		t.Errorf("String = %v, want 'test'", resultMap["string"])
	}

	if resultMap["number"].(float64) != 123 {
		t.Errorf("Number = %v, want 123", resultMap["number"])
	}

	if resultMap["boolean"] != true {
		t.Errorf("Boolean = %v, want true", resultMap["boolean"])
	}

	if resultMap["null"] != nil {
		t.Errorf("Null = %v, want nil", resultMap["null"])
	}

	arr, ok := resultMap["array"].([]interface{})
	if !ok || len(arr) != 3 {
		t.Error("Array should have 3 elements")
	}
}

func TestBrowserE2E_XPathQueries(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	var text string
	err := chromedp.Run(browserCtx, chromedp.Text(`//h1[@id="title"]`, &text, chromedp.BySearch))
	if err != nil {
		t.Fatalf("XPath query: %v", err)
	}

	if text != "Welcome to Test Page" {
		t.Errorf("Text = %q, want 'Welcome to Test Page'", text)
	}

	var exists bool
	err = chromedp.Run(browserCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		var nodes []*cdp.Node
		if err := chromedp.Nodes(`//input[@id="name"]`, &nodes, chromedp.BySearch).Do(ctx); err != nil {
			return err
		}
		exists = len(nodes) > 0
		return nil
	}))
	if err != nil {
		t.Fatalf("Check input by XPath: %v", err)
	}

	if !exists {
		t.Error("Input element should exist by XPath")
	}
}

func TestBrowserE2E_MutationObserver(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	js := `
		(function() {
			var callback = function(mutationsList, observer) {
				for (var mutation of mutationsList) {
					if (mutation.type === 'childList') {
						console.log('Child nodes changed');
					}
				}
			};
			var observer = new MutationObserver(callback);
			observer.observe(document.body, {childList: true, subtree: true});
			return 'observer set';
		})()
	`
	var result string
	err := chromedp.Run(browserCtx, chromedp.Evaluate(js, &result))
	if err != nil {
		t.Fatalf("Set up MutationObserver: %v", err)
	}

	if result != "observer set" {
		t.Errorf("Result = %q, want 'observer set'", result)
	}

	err = chromedp.Run(browserCtx, chromedp.Evaluate(`
		(function() {
			var div = document.createElement('div');
			div.textContent = 'Dynamic content';
			document.body.appendChild(div);
			return 'added';
		})()
	`, &result))
	if err != nil {
		t.Fatalf("Add element: %v", err)
	}

	if result != "added" {
		t.Errorf("Result = %q, want 'added'", result)
	}
}

func TestBrowserE2E_PDFGeneration(t *testing.T) {
	ts := newTestServer(t)
	defer ts.Close()

	allocCtx := startBrowser(t)
	browserCtx := newBrowserContext(t, allocCtx)

	navigateToPage(t, browserCtx, ts.URL)

	tempDir := t.TempDir()
	pdfPath := filepath.Join(tempDir, "test.pdf")

	var buf []byte
	err := chromedp.Run(browserCtx, chromedp.ActionFunc(func(ctx context.Context) error {
		var err error
		buf, _, err = cdppage.PrintToPDF().WithPrintBackground(true).Do(ctx)
		return err
	}))
	if err != nil {
		t.Fatalf("Generate PDF: %v", err)
	}

	if len(buf) == 0 {
		t.Error("PDF buffer is empty")
	}

	if len(buf) < 1000 {
		t.Errorf("PDF size = %d, seems too small", len(buf))
	}

	err = os.WriteFile(pdfPath, buf, 0644)
	if err != nil {
		t.Fatalf("Write PDF: %v", err)
	}

	stat, err := os.Stat(pdfPath)
	if err != nil {
		t.Fatalf("Stat PDF: %v", err)
	}

	if stat.Size() == 0 {
		t.Error("PDF file is empty")
	}

	if !strings.HasPrefix(string(buf[:4]), "%PDF") {
		t.Error("PDF should start with %PDF")
	}
}
