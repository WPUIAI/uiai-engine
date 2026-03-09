package vision

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

// IdleTimeout is how long Chrome sits idle before being killed to free memory.
const IdleTimeout = 5 * time.Minute

// Queue constants
const (
	MaxQueueDepth  = 20               // max waiting requests before 429
	QueueWaitMax   = 30 * time.Second // max time in queue before 408
)

// WarmPageCount is how many blank pages to pre-create when Chrome launches.
// These are ready to navigate immediately, eliminating page creation latency (~50ms each).
const WarmPageCount = 1

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
	// Screenshot cache: avoids redundant Chrome renders
	cache *screenshotCache
	// Request queue: graceful degradation under load
	queued    int64 // atomic: requests currently waiting for a page
	queueMax  int
	queueDone int64 // total requests served from queue (waited > 0s)
	queueDrop int64 // total requests rejected (queue full or timeout)
	// SSRF: when true, block private/internal IPs in screenshot URLs.
	// Commercial deployments may set false to allow localhost screenshots.
	blockPrivateURLs bool
	// Periodic browser recycle to prevent Chrome memory bloat
	screenshotCount int64 // atomic: total screenshots taken since last Chrome restart
}

const (
	// RecycleAfter: restart Chrome after this many screenshots to prevent memory bloat.
	// Chrome's renderer process leaks memory over time (~1-5MB per page navigation).
	RecycleAfter = 200
)

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
	NoCache  bool   // skip cache, always take fresh screenshot
}

type ScreenshotResult struct {
	Data      []byte
	Width     int
	Height    int
	Format    string
	Duration  time.Duration
	DOMReport string // lightweight DOM summary for Coach
}

// PoolConfig controls pool behavior. Zero values use safe defaults.
type PoolConfig struct {
	MaxPages         int
	AllowPrivateURLs bool // when true, disables SSRF private-IP blocking (for local dev/staging)
}

func NewPool(maxPages int) (*Pool, error) {
	return NewPoolWithConfig(PoolConfig{MaxPages: maxPages})
}

func NewPoolWithConfig(cfg PoolConfig) (*Pool, error) {
	if cfg.MaxPages <= 0 {
		cfg.MaxPages = 2
	}

	p := &Pool{
		pages:            make(chan *rod.Page, cfg.MaxPages),
		maxPages:         cfg.MaxPages,
		lastSuccess:      time.Now(),
		cache:            newScreenshotCache(DefaultCacheTTL, DefaultCacheMaxSize),
		queueMax:         MaxQueueDepth,
		blockPrivateURLs: !cfg.AllowPrivateURLs,
	}

	// Lazy launch: Chrome is NOT started here. It starts on first getPage().
	// This saves ~411MB of RAM when no screenshots are being requested.
	ssrfStatus := "ON"
	if cfg.AllowPrivateURLs {
		ssrfStatus = "OFF (private URLs allowed)"
	}
	log.Printf("[vision] Pool initialized: max=%d pages, cache=50MB/5min, SSRF protection=%s (Chrome starts on first request)", p.maxPages, ssrfStatus)

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
		// === JET FUEL: Speed optimizations ===
		Set("disable-background-networking").       // no background fetches
		Set("disable-default-apps").                // don't load default apps
		Set("disable-sync").                        // no account sync
		Set("disable-backgrounding-occluded-windows"). // render even when "hidden"
		Set("disable-renderer-backgrounding").      // never throttle renderers
		Set("disable-ipc-flooding-protection").     // allow rapid IPC (we control the pages)
		Set("disable-hang-monitor").                // no hang detection overhead
		Set("disable-prompt-on-repost").            // no repost dialog
		Set("disable-client-side-phishing-detection"). // no phishing checks
		Set("metrics-recording-only").              // metrics without upload
		Set("no-first-run").                        // skip first-run experience
		Set("enable-features", "NetworkServiceInProcess2"). // network service in main process (fewer context switches)
		// Block analytics/tracking at DNS level — zero network overhead
		Set("host-resolver-rules",
			"MAP www.google-analytics.com 0.0.0.0, "+
				"MAP google-analytics.com 0.0.0.0, "+
				"MAP www.googletagmanager.com 0.0.0.0, "+
				"MAP googletagmanager.com 0.0.0.0, "+
				"MAP connect.facebook.net 0.0.0.0, "+
				"MAP static.hotjar.com 0.0.0.0, "+
				"MAP us.posthog.com 0.0.0.0, "+
				"MAP app.posthog.com 0.0.0.0, "+
				"MAP clarity.ms 0.0.0.0, "+
				"MAP pagead2.googlesyndication.com 0.0.0.0, "+
				"MAP www.googleadservices.com 0.0.0.0").
		// Note: --js-flags=--max-old-space-size was tested but causes Chrome to
		// spin at 100% CPU during initialization. Removed in favor of feature disabling.
		// Append to Rod's default disable-features (site-per-process,TranslateUI)
		Append("disable-features",
			"OnDeviceModel",         // kills on_device_model.mojom process (~31MB)
			"ChromeMLService",       // ML service not needed for screenshots
			"OptimizationHints",     // network hints not needed in headless
			"MediaRouter",           // cast/media router useless in headless
			"Translate",             // translation service
			"ChromePasswordManager", // password manager
			"PaintHolding",          // delays first paint — JET FUEL: faster first render
			"BackForwardCache",      // no BFCache overhead for screenshot pages
			"AutofillServerCommunication", // no autofill network calls
			"CalculateNativeWinOcclusion", // Linux: skip Windows occlusion calc
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

	// Pre-warm pages so first requests don't pay page-creation cost
	warmCount := WarmPageCount
	if warmCount > p.maxPages {
		warmCount = p.maxPages
	}
	for i := 0; i < warmCount; i++ {
		page, err := p.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
		if err != nil {
			log.Printf("[vision] Pre-warm page %d failed: %v", i, err)
			break
		}
		p.pages <- page
		p.created++
	}

	log.Printf("[vision] Browser launched: PID=%d, max=%d pages, pre-warmed=%d", p.browserPID, p.maxPages, warmCount)
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

// ErrQueueFull is returned when the request queue is at capacity.
var ErrQueueFull = fmt.Errorf("queue full")

// ErrQueueTimeout is returned when a request waited too long in the queue.
var ErrQueueTimeout = fmt.Errorf("queue timeout")

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
		// Quick health check — if browser is alive, trust the page.
		// Full Eval check (2s timeout) is too expensive for hot path.
		if p.isBrowserAlive() {
			p.mu.Lock()
			p.active++
			p.mu.Unlock()
			return page, nil
		}
		// Browser died — discard page
		log.Printf("[vision] Browser dead, discarding pooled page")
		page.Close()
		p.mu.Lock()
		p.created--
		p.mu.Unlock()
		// Fall through to restart + create new
	default:
	}

	p.mu.Lock()

	// Check browser health before creating page
	if !p.isBrowserAlive() {
		log.Printf("[vision] Browser dead, restarting")
		if err := p.restartBrowser(); err != nil {
			p.mu.Unlock()
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
					p.mu.Unlock()
					return nil, fmt.Errorf("browser restart failed: %w", restartErr)
				}
				page, err = p.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
				if err != nil {
					p.mu.Unlock()
					return nil, fmt.Errorf("page create failed after restart: %w", err)
				}
			} else {
				p.mu.Unlock()
				return nil, err
			}
		}
		p.created++
		p.active++
		p.mu.Unlock()
		return page, nil
	}

	// Pool full — check queue depth before waiting
	depth := atomic.LoadInt64(&p.queued)
	if int(depth) >= p.queueMax {
		p.mu.Unlock()
		atomic.AddInt64(&p.queueDrop, 1)
		return nil, ErrQueueFull
	}
	p.mu.Unlock()

	// Enter queue — wait for a page to be released
	atomic.AddInt64(&p.queued, 1)
	log.Printf("[vision] queued request (depth=%d, active=%d, max=%d)", depth+1, p.active, p.maxPages)

	select {
	case page := <-p.pages:
		atomic.AddInt64(&p.queued, -1)
		atomic.AddInt64(&p.queueDone, 1)
		p.mu.Lock()
		p.active++
		p.mu.Unlock()
		return page, nil
	case <-time.After(QueueWaitMax):
		atomic.AddInt64(&p.queued, -1)
		atomic.AddInt64(&p.queueDrop, 1)
		return nil, ErrQueueTimeout
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
	// Reset page context to a fresh one (the screenshot ctx may be cancelled)
	cleanPage := page.Context(context.Background())

	// Fast release: navigate to about:blank asynchronously with short timeout.
	// This clears cookies/state and frees renderer memory for the previous page.
	done := make(chan bool, 1)
	go func() {
		err := cleanPage.Timeout(2 * time.Second).Navigate("about:blank")
		done <- (err == nil)
	}()

	select {
	case ok := <-done:
		if !ok {
			// Navigation failed — page may be in bad state, discard
			cleanPage.Close()
			p.mu.Lock()
			p.created--
			p.active--
			p.scheduleIdleShutdown()
			p.mu.Unlock()
			return
		}
	case <-time.After(2 * time.Second):
		// Page is stuck — close it and discard
		log.Printf("[vision] Page stuck on release, closing")
		cleanPage.Close()
		p.mu.Lock()
		p.created--
		p.active--
		p.scheduleIdleShutdown()
		p.mu.Unlock()
		return
	}

	select {
	case p.pages <- cleanPage:
	default:
		cleanPage.Close()
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
	// Check cache first (unless NoCache is set or Delay > 0 means caller wants fresh)
	if !opts.NoCache && opts.Delay == 0 {
		key := cacheKey(opts)
		if cached := p.cache.get(key); cached != nil {
			log.Printf("[vision] cache hit for %s (saved Chrome render)", opts.URL)
			return cached, nil
		}
	}

	// Overall deadline for the entire operation (default 15s, capped at 60s)
	timeout := time.Duration(opts.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	if timeout > 60*time.Second {
		timeout = 60 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	type result struct {
		res *ScreenshotResult
		err error
	}
	ch := make(chan result, 1)

	// timedOut is set when the outer select hits ctx.Done().
	// The orphaned goroutine checks this to force-close its page.
	var timedOut int32

	go func() {
		res, err := p.screenshotInner(ctx, opts)
		ch <- result{res, err}

		// If the outer caller already timed out, the page was released
		// inside screenshotInner's defer — but we force-reclaim just in case
		// the goroutine was stuck and the page is leaked.
		if atomic.LoadInt32(&timedOut) == 1 {
			log.Printf("[vision] orphaned goroutine finished for %s after caller timed out", opts.URL)
		}
	}()

	select {
	case r := <-ch:
		if r.err != nil {
			// Queue errors are not browser failures — don't count them
			if !errors.Is(r.err, ErrQueueFull) && !errors.Is(r.err, ErrQueueTimeout) {
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
			}
			return nil, r.err
		}

		count := atomic.AddInt64(&p.screenshotCount, 1)

		p.mu.Lock()
		p.lastSuccess = time.Now()
		p.failCount = 0
		p.mu.Unlock()

		// Periodic Chrome recycle to prevent memory bloat
		if count > 0 && count%RecycleAfter == 0 {
			log.Printf("[vision] %d screenshots taken, scheduling Chrome recycle", count)
			go func() {
				// Wait for active pages to drain
				for i := 0; i < 30; i++ {
					p.mu.Lock()
					active := p.active
					p.mu.Unlock()
					if active == 0 {
						break
					}
					time.Sleep(time.Second)
				}
				p.mu.Lock()
				if p.active == 0 {
					log.Printf("[vision] Recycling Chrome (screenshot count=%d)", count)
					p.restartBrowser()
				}
				p.mu.Unlock()
			}()
		}

		// Store in cache (skip if caller used delay — result is time-sensitive)
		if !opts.NoCache && r.res != nil {
			p.cache.put(cacheKey(opts), r.res)
		}
		return r.res, nil

	case <-ctx.Done():
		// Timeout — the goroutine may still hold a page.
		atomic.StoreInt32(&timedOut, 1)

		p.mu.Lock()
		p.failCount++
		fc := p.failCount
		p.mu.Unlock()

		// Restart aggressively — a timed-out screenshot likely means Chrome
		// renderer is stuck. Don't make the next customer wait 15s too.
		if fc >= 1 {
			log.Printf("[vision] Timeout + %d failures, restarting browser", fc)
			p.mu.Lock()
			p.restartBrowser()
			p.mu.Unlock()
		}
		return nil, fmt.Errorf("screenshot timed out after %s for %s", timeout, opts.URL)
	}
}

// containsAny checks if s contains any of the substrings.
func containsAny(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// isDangerousScheme blocks non-HTTP schemes unconditionally (file://, ftp://, data://, etc).
func isDangerousScheme(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return true
	}
	return parsed.Scheme != "http" && parsed.Scheme != "https" && parsed.Scheme != ""
}

// isPrivateURL checks if a URL points to a private/internal IP.
func isPrivateURL(rawURL string) bool {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return true
	}
	host := parsed.Hostname()
	for _, prefix := range []string{
		"127.", "10.", "192.168.", "172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.", "172.24.", "172.25.",
		"172.26.", "172.27.", "172.28.", "172.29.", "172.30.", "172.31.",
		"0.", "169.254.", "::1", "fc", "fd", "fe80:",
	} {
		if strings.HasPrefix(host, prefix) {
			return true
		}
	}
	return host == "localhost" || host == ""
}

func (p *Pool) screenshotInner(ctx context.Context, opts ScreenshotOpts) (*ScreenshotResult, error) {
	start := time.Now()

	// Always block dangerous schemes (file://, ftp://, data://, etc)
	if isDangerousScheme(opts.URL) {
		return nil, fmt.Errorf("URL scheme not allowed: only http:// and https:// are supported")
	}
	// SSRF protection: block private/internal URLs (configurable for commercial use)
	if p.blockPrivateURLs && isPrivateURL(opts.URL) {
		return nil, fmt.Errorf("URL not allowed: private/internal addresses blocked (set allow_private_urls to enable)")
	}

	page, err := p.getPage()
	if err != nil {
		return nil, fmt.Errorf("failed to get page: %w", err)
	}
	defer p.releasePage(page)

	// Bind page to our context — all Rod operations will cancel when ctx expires.
	// This prevents orphaned goroutines holding pages when Screenshot() times out.
	page = page.Context(ctx)

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

	// Cap navigation + DOM stable timeout to remaining context budget
	navTimeout := 15 * time.Second
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline) - 5*time.Second // reserve 5s for screenshot
		if remaining < navTimeout && remaining > 0 {
			navTimeout = remaining
		}
	}

	// Navigate with timeout — use DOMContentLoaded, NOT full load.
	t0 := time.Now()
	navErr := page.Timeout(navTimeout).Navigate(opts.URL)
	navDur := time.Since(t0)
	if navErr != nil {
		return nil, fmt.Errorf("navigation failed: %w", navErr)
	}

	// Freeze animations/transitions before waiting for DOM stability.
	// This prevents CSS animations from causing perpetual DOM instability.
	page.Eval(`() => {
		const s = document.createElement('style');
		s.id = '__rod_freeze';
		s.textContent = '*, *::before, *::after { animation-duration: 0s !important; transition-duration: 0s !important; animation-delay: 0s !important; transition-delay: 0s !important; }';
		document.head.appendChild(s);
	}`)

	// Wait for DOM stability — faster now that animations are frozen
	stableTimeout := navTimeout / 3
	if stableTimeout < 1*time.Second {
		stableTimeout = 1 * time.Second
	}
	if stableTimeout > 3*time.Second {
		stableTimeout = 3 * time.Second
	}
	t1 := time.Now()
	waitErr := page.Timeout(stableTimeout).WaitDOMStable(100*time.Millisecond, 0.2)
	domDur := time.Since(t1)

	// Remove freeze style before screenshot (so hover states etc work)
	page.Eval(`() => { const f = document.getElementById('__rod_freeze'); if (f) f.remove(); }`)

	if waitErr != nil {
		log.Printf("[vision] WaitDOMStable timeout for %s after %s (continuing)", opts.URL, domDur.Round(time.Millisecond))
	}
	log.Printf("[vision] %s: nav=%s dom=%s", opts.URL, navDur.Round(time.Millisecond), domDur.Round(time.Millisecond))

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
		// Default quality 60 for JPEG — imperceptible difference for AI vision
		// but ~27% smaller transfers vs 80, ~44% smaller vs 85.
		// Callers can override to 80-90 if pixel-perfect comparison needed.
		if format == proto.PageCaptureScreenshotFormatJpeg {
			quality = 60
		} else {
			quality = 85 // PNG quality is lossless, this is compression level
		}
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

	// Extract DOM summary — fast path: skip if context is nearly expired
	domReport := ""
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 2*time.Second {
		domReport = extractDOMReport(page)
	}

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

	stats := map[string]any{
		"max_pages":       p.maxPages,
		"created_pages":   p.created,
		"available_pages": len(p.pages),
		"active_pages":    p.active,
		"browser_pid":     p.browserPID,
		"browser_alive":   p.browser != nil && p.isBrowserAlive(),
		"browser_state":   browserState,
		"last_success":    p.lastSuccess.Format(time.RFC3339),
		"fail_count":        p.failCount,
		"screenshot_count":  atomic.LoadInt64(&p.screenshotCount),
		"recycle_every":     RecycleAfter,
		"queue": map[string]any{
			"depth":    atomic.LoadInt64(&p.queued),
			"max":      p.queueMax,
			"served":   atomic.LoadInt64(&p.queueDone),
			"rejected": atomic.LoadInt64(&p.queueDrop),
		},
	}
	if p.cache != nil {
		stats["cache"] = p.cache.stats()
	}
	return stats
}

func gson(v int) *int {
	return &v
}
