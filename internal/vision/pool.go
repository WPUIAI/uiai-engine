package vision

import (
	"fmt"
	"log"
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
}

type ScreenshotResult struct {
	Data     []byte
	Width    int
	Height   int
	Format   string
	Duration time.Duration
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

	page.MustSetViewport(width, height, 1, false)

	// Navigate
	if err := page.Navigate(opts.URL); err != nil {
		return nil, fmt.Errorf("navigation failed: %w", err)
	}

	// Wait for page load
	page.MustWaitLoad()

	// Wait for specific selector
	if opts.WaitFor != "" {
		page.Timeout(5 * time.Second).MustElement(opts.WaitFor)
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

	return &ScreenshotResult{
		Data:     data,
		Width:    width,
		Height:   height,
		Format:   string(format),
		Duration: time.Since(start),
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
