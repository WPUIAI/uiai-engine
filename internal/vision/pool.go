package vision

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// IdleTimeout is how long Chrome sits idle before being killed to free memory.
const IdleTimeout = 5 * time.Minute

type Pool struct {
	mu       sync.Mutex
	browser  *rod.Browser
	launcher *launcher.Launcher
	pages    chan *rod.Page
	maxPages int
	created  int
	// Health tracking
	lastSuccess time.Time
	failCount   int
	browserPID  int
	// Lazy launch: Chrome starts on first request, auto-kills after IdleTimeout
	idleTimer *time.Timer
	active    int // number of pages currently checked out
}

type ScreenshotOpts struct {
	URL      string
	Width    int
	Height   int
	FullPage bool
	Format   string // "png" or "jpeg"
	Quality  int    // JPEG quality 0-100
	WaitFor  string // CSS selector to wait for
	Delay    int    // ms to wait after load
	Cookies  string // "name=value; name2=value2" — set before navigation
	Timeout  int    // overall timeout in seconds (default: 30)
}

type ScreenshotResult struct {
	Data      []byte
	Width     int
	Height    int
	Format    string
	Duration  time.Duration
	DOMReport string // lightweight DOM summary for Coach
}

func NewPool(maxPages int) (*Pool, error) {
	if maxPages <= 0 {
		maxPages = 2
	}

	p := &Pool{
		pages:       make(chan *rod.Page, maxPages),
		maxPages:    maxPages,
		lastSuccess: time.Now(),
	}

	// Lazy launch: Chrome is NOT started here. It starts on first getPage().
	// This saves ~411MB of RAM when no screenshots are being requested.
	log.Printf("[vision] Pool initialized: max=%d pages (Chrome starts on first request)", p.maxPages)

	return p, nil
}

// launchBrowser starts (or restarts) the Chrome process.
func (p *Pool) launchBrowser() error {
	// Clean up old browser if any
	if p.browser != nil {
		p.browser.Close()
	}
	if p.launcher != nil {
		p.launcher.Cleanup()
	}

	// Drain any old pages from the channel
	for {
		select {
		case pg := <-p.pages:
			pg.Close()
		default:
			goto drained
		}
	}
drained:
	p.created = 0

	// Find system chromium first, fall back to Rod's bundled one.
	// Use actual binaries, not shell wrappers (Rod needs direct binary).
	chromePath := ""
	for _, candidate := range []string{
		"/usr/lib64/chromium-browser/chromium-browser",
		"/usr/lib/chromium-browser/chromium-browser",
		"/usr/bin/google-chrome-stable",
		"/usr/bin/google-chrome",
		"/usr/bin/chromium",
	} {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			chromePath = candidate
			break
		}
	}

	l := launcher.New().
		Headless(true).
		// Flags NOT in Rod defaults — add explicitly
		Set("disable-gpu").
		Set("no-sandbox").
		Set("disable-web-security").
		Set("disable-extensions").
		Set("disable-translate").
		Set("mute-audio").
		Set("safebrowsing-disable-auto-update").
		// Memory reduction: kill unnecessary Chrome services
		Set("disable-component-update").
		Set("disable-domain-reliability").
		Set("disable-crash-reporter").
		Set("js-flags", "--max-old-space-size=256").
		// Append to Rod's default disable-features (site-per-process,TranslateUI)
		Append("disable-features",
			"OnDeviceModel",         // kills on_device_model.mojom process (~31MB)
			"ChromeMLService",       // ML service not needed for screenshots
			"OptimizationHints",     // network hints not needed in headless
			"MediaRouter",           // cast/media router useless in headless
			"Translate",             // translation service
			"ChromePasswordManager", // password manager
			"PaintHolding",          // delays first paint
		)

	if chromePath != "" {
		l = l.Bin(chromePath)
		log.Printf("[vision] Using system browser: %s", chromePath)
	}

	u, err := l.Launch()
	if err != nil {
		return fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		l.Cleanup()
		return fmt.Errorf("failed to connect browser: %w", err)
	}

	browser.IgnoreCertErrors(true)

	p.browser = browser
	p.launcher = l
	p.browserPID = l.PID()
	p.failCount = 0

	log.Printf("[vision] Pool created: max=%d pages, browser PID=%d", p.maxPages, p.browserPID)
	return nil
}

// isBrowserAlive checks if the Chrome process is still running.
func (p *Pool) isBrowserAlive() bool {
	if p.browserPID <= 0 {
		return false
	}
	proc, err := os.FindProcess(p.browserPID)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds. Send signal 0 to check.
	if err := proc.Signal(os.Signal(syscall.Signal(0))); err != nil {
		return false
	}
	return true
}

// restartBrowser kills the old browser and launches a fresh one.
func (p *Pool) restartBrowser() error {
	log.Printf("[vision] Restarting browser (old PID=%d, fails=%d)", p.browserPID, p.failCount)

	// Kill any orphan chrome processes from this launcher
	if p.browserPID > 0 {
		exec.Command("kill", "-9", fmt.Sprintf("%d", p.browserPID)).Run()
		time.Sleep(500 * time.Millisecond)
	}

	return p.launchBrowser()
}

func (p *Pool) getPage() (*rod.Page, error) {
	p.mu.Lock()

	// Cancel idle timer — browser is being used
	if p.idleTimer != nil {
		p.idleTimer.Stop()
		p.idleTimer = nil
	}

	// Lazy launch: start Chrome on first request
	if p.browser == nil {
		log.Printf("[vision] Launching Chrome on-demand")
		if err := p.launchBrowser(); err != nil {
			p.mu.Unlock()
			return nil, fmt.Errorf("on-demand launch failed: %w", err)
		}
	}

	p.mu.Unlock()

	// Try to get from pool (non-blocking)
	select {
	case page := <-p.pages:
		// Verify page is still alive with a quick eval
		_, err := page.Timeout(2 * time.Second).Eval(`() => document.readyState`)
		if err != nil {
			log.Printf("[vision] Stale page in pool, discarding")
			page.Close()
			p.mu.Lock()
			p.created--
			p.mu.Unlock()
			// Fall through to create new
		} else {
			p.mu.Lock()
			p.active++
			p.mu.Unlock()
			return page, nil
		}
	default:
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Check browser health before creating page
	if !p.isBrowserAlive() {
		log.Printf("[vision] Browser dead, restarting")
		if err := p.restartBrowser(); err != nil {
			return nil, fmt.Errorf("browser restart failed: %w", err)
		}
	}

	if p.created < p.maxPages {
		page, err := p.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
		if err != nil {
			// Browser connection might be dead
			if strings.Contains(err.Error(), "closed") || strings.Contains(err.Error(), "EOF") {
				log.Printf("[vision] Browser connection lost, restarting")
				if restartErr := p.restartBrowser(); restartErr != nil {
					return nil, fmt.Errorf("browser restart failed: %w", restartErr)
				}
				page, err = p.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
				if err != nil {
					return nil, fmt.Errorf("page create failed after restart: %w", err)
				}
			} else {
				return nil, err
			}
		}
		p.created++
		p.active++
		return page, nil
	}

	// Pool full — wait with timeout
	p.mu.Unlock() // release lock while waiting
	select {
	case page := <-p.pages:
		p.mu.Lock() // re-acquire for caller's defer
		p.active++
		return page, nil
	case <-time.After(15 * time.Second):
		p.mu.Lock()
		return nil, fmt.Errorf("pool exhausted: all %d pages busy for 15s", p.maxPages)
	}
}

// GetPage returns a page from the pool (exported for interactive routes)
func (p *Pool) GetPage() (*rod.Page, error) {
	return p.getPage()
}

// ReleasePage returns a page to the pool (exported for interactive routes)
func (p *Pool) ReleasePage(page *rod.Page) {
	p.releasePage(page)
}

func (p *Pool) releasePage(page *rod.Page) {
	// Navigate to blank with a short timeout — don't hang
	done := make(chan struct{})
	go func() {
		defer close(done)
		page.Timeout(3 * time.Second).Navigate("about:blank")
	}()

	select {
	case <-done:
		// Page reset OK, return to pool
	case <-time.After(3 * time.Second):
		// Page is stuck — close it
		log.Printf("[vision] Page stuck on release, closing")
		page.Close()
		p.mu.Lock()
		p.created--
		p.active--
		p.scheduleIdleShutdown()
		p.mu.Unlock()
		return
	}

	select {
	case p.pages <- page:
	default:
		page.Close()
		p.mu.Lock()
		p.created--
		p.mu.Unlock()
	}

	p.mu.Lock()
	p.active--
	p.scheduleIdleShutdown()
	p.mu.Unlock()
}

// scheduleIdleShutdown starts or resets the idle timer.
// Must be called with p.mu held.
func (p *Pool) scheduleIdleShutdown() {
	if p.active > 0 || p.browser == nil {
		return
	}
	if p.idleTimer != nil {
		p.idleTimer.Stop()
	}
	p.idleTimer = time.AfterFunc(IdleTimeout, func() {
		p.mu.Lock()
		defer p.mu.Unlock()
		if p.active > 0 || p.browser == nil {
			return
		}
		log.Printf("[vision] Chrome idle for %v, shutting down to free memory (PID=%d)", IdleTimeout, p.browserPID)
		// Drain pages
		for {
			select {
			case pg := <-p.pages:
				pg.Close()
			default:
				goto drained
			}
		}
	drained:
		p.browser.Close()
		if p.launcher != nil {
			p.launcher.Cleanup()
		}
		p.browser = nil
		p.launcher = nil
		p.created = 0
		p.browserPID = 0
	})
}

func (p *Pool) Screenshot(opts ScreenshotOpts) (*ScreenshotResult, error) {
	// Overall deadline for the entire operation
	timeout := time.Duration(opts.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	type result struct {
		res *ScreenshotResult
		err error
	}
	ch := make(chan result, 1)

	go func() {
		res, err := p.screenshotInner(ctx, opts)
		ch <- result{res, err}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			p.mu.Lock()
			p.failCount++
			fc := p.failCount
			p.mu.Unlock()

			// Auto-restart browser after 3 consecutive failures
			if fc >= 3 {
				log.Printf("[vision] %d consecutive failures, restarting browser", fc)
				p.mu.Lock()
				p.restartBrowser()
				p.mu.Unlock()
			}
			return nil, r.err
		}

		p.mu.Lock()
		p.lastSuccess = time.Now()
		p.failCount = 0
		p.mu.Unlock()
		return r.res, nil

	case <-ctx.Done():
		// Timeout — the goroutine is stuck. Mark failure.
		p.mu.Lock()
		p.failCount++
		fc := p.failCount
		p.mu.Unlock()

		if fc >= 2 {
			log.Printf("[vision] Timeout + %d failures, restarting browser", fc)
			p.mu.Lock()
			p.restartBrowser()
			p.mu.Unlock()
		}
		return nil, fmt.Errorf("screenshot timed out after %s for %s", timeout, opts.URL)
	}
}

func (p *Pool) screenshotInner(ctx context.Context, opts ScreenshotOpts) (*ScreenshotResult, error) {
	start := time.Now()

	page, err := p.getPage()
	if err != nil {
		return nil, fmt.Errorf("failed to get page: %w", err)
	}
	defer p.releasePage(page)

	// Set viewport
	width := opts.Width
	if width <= 0 {
		width = 1280
	}
	height := opts.Height
	if height <= 0 {
		height = 800
	}

	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width: width, Height: height, DeviceScaleFactor: 1,
	}); err != nil {
		return nil, fmt.Errorf("set viewport failed: %w", err)
	}

	// Set cookies before navigation (for authenticated screenshots)
	if opts.Cookies != "" {
		parsed, _ := url.Parse(opts.URL)
		domain := parsed.Hostname()
		for _, c := range strings.Split(opts.Cookies, ";") {
			c = strings.TrimSpace(c)
			parts := strings.SplitN(c, "=", 2)
			if len(parts) == 2 {
				page.MustSetCookies(&proto.NetworkCookieParam{
					Name:   strings.TrimSpace(parts[0]),
					Value:  strings.TrimSpace(parts[1]),
					Domain: domain,
					Path:   "/",
				})
			}
		}
	}

	// Check context before navigation
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Navigate with timeout — use DOMContentLoaded, NOT full load.
	// Full load waits for all subresources (WebSockets, long-polling, etc.)
	// which may never complete on dynamic sites.
	navErr := page.Timeout(15 * time.Second).Navigate(opts.URL)
	if navErr != nil {
		return nil, fmt.Errorf("navigation failed: %w", navErr)
	}

	// Wait for DOMContentLoaded (not load!) — this fires once HTML is parsed
	// and deferred scripts have run, but doesn't wait for images/XHR/WS.
	waitErr := page.Timeout(15 * time.Second).WaitDOMStable(300*time.Millisecond, 0.1)
	if waitErr != nil {
		log.Printf("[vision] WaitDOMStable timeout for %s: %v (continuing with screenshot)", opts.URL, waitErr)
	}

	// Wait for specific selector
	if opts.WaitFor != "" {
		if _, err := page.Timeout(5 * time.Second).Element(opts.WaitFor); err != nil {
			log.Printf("[vision] WaitFor '%s' timeout: %v (continuing)", opts.WaitFor, err)
		}
	}

	// Additional delay for AJAX content
	if opts.Delay > 0 {
		// Check context periodically during delay
		remaining := time.Duration(opts.Delay) * time.Millisecond
		step := 500 * time.Millisecond
		for remaining > 0 {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			sleep := step
			if remaining < step {
				sleep = remaining
			}
			time.Sleep(sleep)
			remaining -= sleep
		}
	}

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// Take screenshot
	format := proto.PageCaptureScreenshotFormatPng
	if opts.Format == "jpeg" || opts.Format == "jpg" {
		format = proto.PageCaptureScreenshotFormatJpeg
	}

	quality := opts.Quality
	if quality <= 0 {
		quality = 85
	}

	var data []byte
	if opts.FullPage {
		data, err = page.Screenshot(true, &proto.PageCaptureScreenshot{
			Format:  format,
			Quality: gson(quality),
		})
	} else {
		data, err = page.Screenshot(false, &proto.PageCaptureScreenshot{
			Format:  format,
			Quality: gson(quality),
		})
	}
	if err != nil {
		return nil, fmt.Errorf("screenshot failed: %w", err)
	}

	// Extract lightweight DOM summary for Coach vision
	domReport := extractDOMReport(page)

	return &ScreenshotResult{
		Data:      data,
		Width:     width,
		Height:    height,
		Format:    string(format),
		Duration:  time.Since(start),
		DOMReport: domReport,
	}, nil
}

func extractDOMReport(page *rod.Page) string {
	domJS := `() => {
		const h = document.querySelectorAll('h1,h2,h3');
		const headings = Array.from(h).slice(0,20).map(e => e.tagName + ': ' + e.textContent.trim().substring(0,80));
		const imgs = document.querySelectorAll('img');
		const imgCount = imgs.length;
		const missingAlt = Array.from(imgs).filter(i => !i.alt || i.alt === '').length;
		const links = document.querySelectorAll('a[href]');
		const buttons = document.querySelectorAll('.wp-block-button,.wp-block-buttons a,button');
		const sections = document.querySelectorAll('.wp-block-group,.wp-block-cover,.wp-block-columns');
		return JSON.stringify({
			title: document.title,
			headings: headings,
			images: {total: imgCount, missing_alt: missingAlt},
			links: links.length,
			buttons: buttons.length,
			sections: sections.length,
			viewport: {width: window.innerWidth, height: window.innerHeight, scrollHeight: document.body.scrollHeight}
		});
	}`

	result, err := page.Timeout(3 * time.Second).Eval(domJS)
	if err != nil {
		log.Printf("[vision] DOM eval error: %v", err)
		return ""
	}
	if result == nil {
		return ""
	}

	s := result.Value.Str()
	if s != "" {
		return s
	}

	b, err := json.Marshal(result.Value)
	if err == nil && len(b) > 4 {
		return string(b)
	}
	return ""
}

func (p *Pool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.idleTimer != nil {
		p.idleTimer.Stop()
		p.idleTimer = nil
	}
	if p.browser != nil {
		p.browser.Close()
		p.browser = nil
	}
	if p.launcher != nil {
		p.launcher.Cleanup()
		p.launcher = nil
	}
	log.Printf("[vision] Pool closed")
}

func (p *Pool) Stats() map[string]any {
	p.mu.Lock()
	defer p.mu.Unlock()

	browserState := "idle-off"
	if p.browser != nil {
		if p.isBrowserAlive() {
			browserState = "running"
		} else {
			browserState = "dead"
		}
	}

	return map[string]any{
		"max_pages":       p.maxPages,
		"created_pages":   p.created,
		"available_pages": len(p.pages),
		"active_pages":    p.active,
		"browser_pid":     p.browserPID,
		"browser_alive":   p.browser != nil && p.isBrowserAlive(),
		"browser_state":   browserState,
		"last_success":    p.lastSuccess.Format(time.RFC3339),
		"fail_count":      p.failCount,
	}
}

func gson(v int) *int {
	return &v
}
