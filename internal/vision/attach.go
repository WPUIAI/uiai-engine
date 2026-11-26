package vision

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/focusapacket"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// AttachPool is a PoolSource backed by an operator-owned, already-running
// browser exposed over CDP (#143). Unlike *Pool it never launches or owns a
// browser process: GetPage opens a new tab in the attached browser — which
// inherits the owner profile's cookies and authenticated sessions — and
// ReleasePage closes that tab only. Close() disconnects without ever asking
// the owner's browser to shut down.
//
// Security boundary (#143 acceptance 3): endpoints come exclusively from the
// operator-configured allow-list; the pool never connects to an
// attacker-supplied URL and logs no secrets.
type AttachPool struct {
	Name         string
	Endpoint     string // http(s) CDP debugger endpoint, e.g. http://127.0.0.1:9222
	AllowPrivate bool

	mu      sync.Mutex
	browser *rod.Browser
}

// NewAttachPool creates an attach source for one allow-listed endpoint.
// The browser is connected lazily on first use.
func NewAttachPool(name, endpoint string, allowPrivate bool) *AttachPool {
	return &AttachPool{Name: strings.TrimSpace(name), Endpoint: strings.TrimSpace(endpoint), AllowPrivate: allowPrivate}
}

type cdpVersionInfo struct {
	WebSocketDebuggerURL string `json:"webSocketDebuggerUrl"`
}

// debuggerWSURL resolves the browser-level CDP WebSocket URL from the
// endpoint's /json/version document. A short bounded timeout keeps a dead
// endpoint from stalling session opens.
func (a *AttachPool) debuggerWSURL() (string, error) {
	endpoint := strings.TrimRight(a.Endpoint, "/")
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"/json/version", nil)
	if err != nil {
		return "", fmt.Errorf("attach endpoint invalid: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("attach endpoint unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("attach endpoint answered %d, want 200", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return "", fmt.Errorf("attach endpoint read failed: %w", err)
	}
	var info cdpVersionInfo
	if err := json.Unmarshal(body, &info); err != nil || info.WebSocketDebuggerURL == "" {
		return "", fmt.Errorf("attach endpoint did not expose a webSocketDebuggerUrl")
	}
	return info.WebSocketDebuggerURL, nil
}

// connectLocked returns the cached browser connection, (re)resolving the
// debugger URL and connecting when the cache is empty. Callers must hold mu.
func (a *AttachPool) connectLocked() (*rod.Browser, error) {
	if a.browser != nil {
		return a.browser, nil
	}
	wsURL, err := a.debuggerWSURL()
	if err != nil {
		return nil, err
	}
	browser := rod.New().ControlURL(wsURL)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("attach connect failed: %w", err)
	}
	a.browser = browser
	return a.browser, nil
}

// GetPage opens a new tab in the attached browser. On failure the cached
// connection is dropped so the next attempt reconnects cleanly.
func (a *AttachPool) GetPage() (*rod.Page, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	browser, err := a.connectLocked()
	if err != nil {
		return nil, err
	}
	page, err := browser.Page(proto.TargetCreateTarget{})
	if err != nil {
		a.browser = nil // force a clean reconnect next time
		return nil, fmt.Errorf("attach tab open failed: %w", err)
	}
	return page, nil
}

// ReleasePage closes the attached tab only — never the owner's browser.
func (a *AttachPool) ReleasePage(page *rod.Page) {
	if page == nil {
		return
	}
	_ = page.Close()
}

// ValidateNavigationURL mirrors the pool policy: only http(s), and private
// addresses blocked unless the endpoint entry opts in.
func (a *AttachPool) ValidateNavigationURL(rawURL string) error {
	if isDangerousScheme(rawURL) {
		return fmt.Errorf("URL scheme not allowed: only http:// and https:// are supported")
	}
	if !a.AllowPrivate && isPrivateURL(rawURL) {
		return fmt.Errorf("URL not allowed: private/internal addresses blocked (set allow_private on the attach endpoint to enable)")
	}
	return nil
}

// IsBrowserAlive reports whether the attached browser still answers its
// debugger endpoint.
func (a *AttachPool) IsBrowserAlive() bool {
	_, err := a.debuggerWSURL()
	return err == nil
}

// MarkFailure drops the cached connection; the attached browser is never
// recycled or killed by the engine.
func (a *AttachPool) MarkFailure() {
	a.mu.Lock()
	a.browser = nil
	a.mu.Unlock()
}

// Reset drops the cached connection so the next open reconnects.
func (a *AttachPool) Reset() {
	a.MarkFailure()
}

// Screenshot runs a bounded one-shot capture through a temporary tab of the
// attached browser and closes that tab afterwards.
func (a *AttachPool) Screenshot(opts ScreenshotOpts) (*ScreenshotResult, error) {
	if err := a.ValidateNavigationURL(opts.URL); err != nil {
		return nil, err
	}
	timeout := 30 * time.Second
	if opts.Timeout > 0 {
		timeout = time.Duration(opts.Timeout) * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	page, err := a.GetPage()
	if err != nil {
		return nil, err
	}
	defer a.ReleasePage(page)
	page = page.Context(ctx)

	width, height := opts.Width, opts.Height
	if width <= 0 {
		width = 1280
	}
	if height <= 0 {
		height = 800
	}
	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{Width: width, Height: height, DeviceScaleFactor: 1}); err != nil {
		return nil, fmt.Errorf("set viewport failed: %w", err)
	}
	if err := page.Context(ctx).Timeout(18 * time.Second).Navigate(opts.URL); err != nil {
		return nil, fmt.Errorf("navigation failed: %w", err)
	}
	settleAfterNavigation(page, opts.URL)
	// Set cookies before navigation (authenticated one-shot captures).
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
		if err := page.Context(ctx).Timeout(18 * time.Second).Navigate(opts.URL); err != nil {
			return nil, fmt.Errorf("navigation failed: %w", err)
		}
		settleAfterNavigation(page, opts.URL)
	}
	if opts.WaitFor != "" {
		if _, err := page.Timeout(5 * time.Second).Element(opts.WaitFor); err != nil {
			// Match the pool's one-shot behavior: a slow selector degrades to
			// the settled page rather than failing the capture.
			log.Printf("[vision-attach] WaitFor %q timeout: %v (continuing)", opts.WaitFor, err)
		}
	}
	if opts.Delay > 0 && opts.Delay <= 5000 {
		time.Sleep(time.Duration(opts.Delay) * time.Millisecond)
	}
	format, quality := opts.Format, opts.Quality
	if format == "" {
		format = "jpeg"
	}
	if quality <= 0 {
		quality = 80
	}
	var shot []byte
	var err2 error
	if strings.EqualFold(format, "png") {
		shot, err2 = page.Context(ctx).Screenshot(true, &proto.PageCaptureScreenshot{})
	} else {
		q := quality
		shot, err2 = page.Context(ctx).Screenshot(true, &proto.PageCaptureScreenshot{Quality: &q})
	}
	if err2 != nil {
		return nil, fmt.Errorf("screenshot failed: %w", err2)
	}
	return &ScreenshotResult{Data: shot, Width: width, Height: height, Format: strings.ToLower(format)}, nil
}

// Stats reports the attach source's bounded identity — never endpoint
// secrets (the endpoint URL is operator config, not a credential).
func (a *AttachPool) Stats() map[string]any {
	a.mu.Lock()
	connected := a.browser != nil
	a.mu.Unlock()
	return map[string]any{
		"kind":      "attach",
		"name":      a.Name,
		"endpoint":  focusapacket.SanitizeURL(a.Endpoint),
		"connected": connected,
	}
}

// Close disconnects the cached connection only. It never sends a browser
// close to the owner's browser.
func (a *AttachPool) Close() {
	a.mu.Lock()
	a.browser = nil
	a.mu.Unlock()
}

// AttachRegistry caches one AttachPool per allow-listed endpoint name so
// repeated session opens reuse the browser connection instead of
// reconnecting per request.
type AttachRegistry struct {
	mu    sync.Mutex
	pools map[string]*AttachPool
}

// NewAttachRegistry creates an empty registry.
func NewAttachRegistry() *AttachRegistry {
	return &AttachRegistry{pools: make(map[string]*AttachPool)}
}

// GetOrCreate returns the cached pool for one allow-listed endpoint entry,
// creating it on first use.
func (r *AttachRegistry) GetOrCreate(name, endpoint string, allowPrivate bool) *AttachPool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if pool, ok := r.pools[name]; ok {
		return pool
	}
	pool := NewAttachPool(name, endpoint, allowPrivate)
	r.pools[name] = pool
	return pool
}

// Close disconnects every cached pool without touching any owner browser.
func (r *AttachRegistry) Close() {
	r.mu.Lock()
	pools := r.pools
	r.pools = make(map[string]*AttachPool)
	r.mu.Unlock()
	for _, pool := range pools {
		pool.Close()
	}
}
