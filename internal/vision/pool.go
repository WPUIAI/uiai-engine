package vision

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

type Pool struct {
	mu       sync.Mutex
	browser  *rod.Browser
	launcher *launcher.Launcher
	pages    chan *rod.Page
	maxPages int
	created  int
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
		maxPages = 4
	}

	l := launcher.New().
		Headless(true).
		Set("disable-gpu").
		Set("no-sandbox").
		Set("disable-dev-shm-usage").
		Set("disable-web-security").
		Set("disable-extensions")

	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("failed to launch browser: %w", err)
	}

	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		return nil, fmt.Errorf("failed to connect browser: %w", err)
	}

	// Ignore certificate errors for local/dev sites
	browser.IgnoreCertErrors(true)

	p := &Pool{
		browser:  browser,
		launcher: l,
		pages:    make(chan *rod.Page, maxPages),
		maxPages: maxPages,
	}

	log.Printf("[vision] Pool created: max=%d pages, browser PID=%d", maxPages, l.PID())
	return p, nil
}

func (p *Pool) getPage() (*rod.Page, error) {
	// Try to get from pool
	select {
	case page := <-p.pages:
		return page, nil
	default:
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.created < p.maxPages {
		page, err := p.browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
		if err != nil {
			return nil, err
		}
		p.created++
		return page, nil
	}

	// Wait for a page to become available
	return <-p.pages, nil
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
	// Navigate to blank to free memory
	page.Navigate("about:blank")
	select {
	case p.pages <- page:
	default:
		page.Close()
		p.mu.Lock()
		p.created--
		p.mu.Unlock()
	}
}

func (p *Pool) Screenshot(opts ScreenshotOpts) (*ScreenshotResult, error) {
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
		// FIX #10: Stale page from pool — close it, create fresh one, retry
		if strings.Contains(err.Error(), "closed") || strings.Contains(err.Error(), "EOF") {
			log.Printf("[vision] Stale page detected, creating fresh page")
			page.Close()
			p.mu.Lock()
			p.created--
			p.mu.Unlock()

			page, err = p.getPage()
			if err != nil {
				return nil, fmt.Errorf("failed to get fresh page: %w", err)
			}
			if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
				Width: width, Height: height, DeviceScaleFactor: 1,
			}); err != nil {
				return nil, fmt.Errorf("set viewport failed (retry): %w", err)
			}
		} else {
			return nil, fmt.Errorf("set viewport failed: %w", err)
		}
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
		log.Printf("[vision] Set cookies for domain %s", domain)
	}

	// Navigate
	if err := page.Navigate(opts.URL); err != nil {
		return nil, fmt.Errorf("navigation failed: %w", err)
	}

	// Wait for page load (with timeout, not Must which panics)
	if err := page.Timeout(30 * time.Second).WaitLoad(); err != nil {
		log.Printf("[vision] WaitLoad timeout for %s: %v (continuing)", opts.URL, err)
	}

	// Wait for specific selector
	if opts.WaitFor != "" {
		if _, err := page.Timeout(5 * time.Second).Element(opts.WaitFor); err != nil {
			log.Printf("[vision] WaitFor '%s' timeout: %v (continuing)", opts.WaitFor, err)
		}
	}

	// Additional delay
	if opts.Delay > 0 {
		time.Sleep(time.Duration(opts.Delay) * time.Millisecond)
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
	domReport := ""
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
	domResult, domErr := page.Eval(domJS)
	if domErr != nil {
		log.Printf("[vision] DOM eval error: %v", domErr)
	} else if domResult != nil {
		s := domResult.Value.Str()
		if s != "" {
			domReport = s
		} else {
			// Try JSON marshaling the value
			b, merr := json.Marshal(domResult.Value)
			if merr == nil && len(b) > 4 {
				domReport = string(b)
			} else {
				log.Printf("[vision] DOM value type=%s str=%q marshal=%v", domResult.Type, s, merr)
			}
		}
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

func (p *Pool) Close() {
	p.browser.Close()
	p.launcher.Cleanup()
	log.Printf("[vision] Pool closed")
}

func (p *Pool) Stats() map[string]any {
	p.mu.Lock()
	defer p.mu.Unlock()
	return map[string]any{
		"max_pages":       p.maxPages,
		"created_pages":   p.created,
		"available_pages": len(p.pages),
	}
}

func gson(v int) *int {
	return &v
}
