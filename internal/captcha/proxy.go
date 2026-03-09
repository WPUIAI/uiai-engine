package captcha

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"

	"github.com/philoveracity/uiai-engine/internal/vision"
)

// ─── Proxy-rotated browser for captcha solving ─────────────────────────────
//
// reCAPTCHA flags IPs after failed attempts. This module provides:
//   1. Proxy list configuration (SOCKS5, HTTP, or residential proxy URLs)
//   2. Round-robin or random proxy rotation per solve attempt
//   3. Ephemeral browser launch per proxy (isolated from main vision pool)
//   4. Session creation on the proxied browser for the captcha solver

// ProxyConfig holds proxy rotation settings.
type ProxyConfig struct {
	Enabled  bool     `yaml:"enabled" json:"enabled"`
	Proxies  []string `yaml:"proxies" json:"proxies"`   // e.g. "socks5://user:pass@host:port"
	Strategy string   `yaml:"strategy" json:"strategy"` // "round_robin" | "random"
	index    int
	mu       sync.Mutex
}

// ProxiedBrowser is an ephemeral browser launched through a proxy.
type ProxiedBrowser struct {
	browser  *rod.Browser
	launcher *launcher.Launcher
	proxy    string
}

// NextProxy picks the next proxy from the list.
func (pc *ProxyConfig) NextProxy() string {
	pc.mu.Lock()
	defer pc.mu.Unlock()

	if !pc.Enabled || len(pc.Proxies) == 0 {
		return ""
	}

	switch pc.Strategy {
	case "random":
		return pc.Proxies[rand.Intn(len(pc.Proxies))]
	default: // round_robin
		proxy := pc.Proxies[pc.index%len(pc.Proxies)]
		pc.index++
		return proxy
	}
}

// LaunchProxiedBrowser starts a new headless Chrome with the given proxy.
func LaunchProxiedBrowser(proxyURL string) (*ProxiedBrowser, error) {
	if proxyURL == "" {
		return nil, fmt.Errorf("empty proxy URL")
	}

	// Validate proxy URL
	_, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL %q: %w", proxyURL, err)
	}

	chromePath := findChromium()

	l := launcher.New().
		Headless(true).
		Set("proxy-server", proxyURL).
		Set("disable-gpu").
		Set("no-sandbox").
		Set("disable-web-security").
		Set("disable-extensions").
		Set("disable-translate").
		Set("mute-audio").
		Set("disable-component-update").
		Set("disable-domain-reliability").
		Set("disable-crash-reporter").
		Set("disable-background-networking").
		Set("disable-default-apps").
		Set("disable-sync").
		Set("no-first-run")

	if chromePath != "" {
		l = l.Bin(chromePath)
	}

	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launch proxied browser: %w", err)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		l.Cleanup()
		return nil, fmt.Errorf("connect proxied browser: %w", err)
	}

	log.Printf("[captcha-proxy] Launched browser with proxy %s", maskProxy(proxyURL))

	return &ProxiedBrowser{
		browser:  browser,
		launcher: l,
		proxy:    proxyURL,
	}, nil
}

// Close shuts down the proxied browser.
func (pb *ProxiedBrowser) Close() {
	if pb.browser != nil {
		pb.browser.Close()
	}
	if pb.launcher != nil {
		pb.launcher.Cleanup()
	}
	log.Printf("[captcha-proxy] Closed proxied browser (%s)", maskProxy(pb.proxy))
}

// OpenPage opens a page in the proxied browser, applies anti-detection,
// and wraps it as a vision.Session for the captcha solver.
func (pb *ProxiedBrowser) OpenPage(targetURL string, width, height int) (*vision.Session, error) {
	page, err := pb.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}

	// Set viewport
	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width: width, Height: height, DeviceScaleFactor: 1,
	}); err != nil {
		page.Close()
		return nil, fmt.Errorf("viewport: %w", err)
	}

	// Anti-detection: patch navigator.webdriver before any navigation
	page.MustEvalOnNewDocument(`
		Object.defineProperty(navigator, 'webdriver', {get: () => undefined});
		Object.defineProperty(navigator, 'languages', {get: () => ['en-US', 'en']});
		Object.defineProperty(navigator, 'plugins', {get: () => [1,2,3,4,5]});
		window.chrome = {runtime: {}};
	`)

	// Navigate
	if err := page.Timeout(25 * time.Second).Navigate(targetURL); err != nil {
		page.Close()
		return nil, fmt.Errorf("navigate: %w", err)
	}
	page.Timeout(5 * time.Second).WaitDOMStable(200*time.Millisecond, 0.15)

	// Wrap raw rod.Page in a vision.Session
	sess := vision.WrapPage(page, targetURL, width, height)
	return sess, nil
}

// ─── Solver proxy integration ──────────────────────────────────────────────

// SolveViaProxy creates a proxied browser, navigates to the target,
// runs the setup func (form filling), then solves the captcha.
// The proxied browser is cleaned up after.
func (s *Solver) SolveViaProxy(ctx context.Context, targetURL string, width, height int,
	setup func(sess *vision.Session) error,
	solveReq SolveRequest,
) *SolveResponse {

	proxy := s.Config.Proxy.NextProxy()
	if proxy == "" {
		return &SolveResponse{
			Solved: false,
			Error:  "no proxies configured — add proxies to captcha config",
			Method: "proxy",
		}
	}

	pb, err := LaunchProxiedBrowser(proxy)
	if err != nil {
		return &SolveResponse{
			Solved: false,
			Error:  fmt.Sprintf("proxy browser launch failed: %v", err),
			Method: "proxy",
		}
	}
	defer pb.Close()

	sess, err := pb.OpenPage(targetURL, width, height)
	if err != nil {
		return &SolveResponse{
			Solved: false,
			Error:  fmt.Sprintf("proxy page open failed: %v", err),
			Method: "proxy",
		}
	}

	// Run caller's setup (fill form fields, select dropdowns, etc.)
	if setup != nil {
		if err := setup(sess); err != nil {
			return &SolveResponse{
				Solved: false,
				Error:  fmt.Sprintf("form setup failed: %v", err),
				Method: "proxy",
			}
		}
	}

	// Solve captcha in the proxied session
	return s.SolveInSession(ctx, sess, solveReq)
}

// ─── Helpers ───────────────────────────────────────────────────────────────

func findChromium() string {
	for _, c := range []string{
		"/usr/lib64/chromium-browser/chromium-browser",
		"/usr/lib/chromium-browser/chromium-browser",
		"/usr/bin/google-chrome-stable",
		"/usr/bin/google-chrome",
		"/usr/bin/chromium",
	} {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

func maskProxy(proxyURL string) string {
	u, err := url.Parse(proxyURL)
	if err != nil {
		return "***"
	}
	host := u.Hostname()
	if len(host) > 8 {
		return host[:4] + "..." + host[len(host)-4:]
	}
	return host + ":***"
}
