package vision

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// MaxSessions is the maximum number of concurrent browser sessions.
// Each session holds a Chrome page (~50-100MB).
const MaxSessions = 4

// SessionTTL is how long an idle session stays alive before auto-cleanup.
const SessionTTL = 5 * time.Minute

// Session is a persistent browser page with identity.
// Unlike the transactional pool (navigate → snap → forget), a session keeps
// the page alive between calls — enabling instant re-screenshots, scrolling,
// clicking, CSS injection, and JS evaluation without re-navigating.
type Session struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	Title     string    `json:"title"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	CreatedAt time.Time `json:"created_at"`
	LastUsed  time.Time `json:"last_used"`
	NavCount  int       `json:"nav_count"`
	SnapCount int       `json:"snap_count"`

	page  *rod.Page
	pool  *Pool
	timer *time.Timer
	mu    sync.Mutex
	refs  map[string]SnapshotRef // @ref → CSS selector, populated by Snapshot()
}

// SessionManager manages persistent browser sessions.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session
	pool     *Pool
}

// NewSessionManager creates a session manager backed by the vision pool.
func NewSessionManager(pool *Pool) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		pool:     pool,
	}
}

// WrapPage creates a Session from a raw rod.Page, without a pool or manager.
// Used by the captcha solver's proxy module to wrap ephemeral browser pages.
func WrapPage(page *rod.Page, url string, width, height int) *Session {
	now := time.Now()
	title := ""
	if el, err := page.Eval(`() => document.title`); err == nil {
		title = el.Value.Str()
	}
	return &Session{
		ID:        generateID(),
		URL:       url,
		Title:     title,
		Width:     width,
		Height:    height,
		CreatedAt: now,
		LastUsed:  now,
		page:      page,
		refs:      make(map[string]SnapshotRef),
	}
}

// generateID creates a short unique session ID.
func generateID() string {
	b := make([]byte, 6)
	for i := range b {
		b[i] = byte(time.Now().UnixNano()>>(i*8)) ^ byte(i*37+13)
	}
	s := base64.RawURLEncoding.EncodeToString(b)
	if len(s) > 8 {
		s = s[:8]
	}
	return s
}

// Open creates a new session, navigates to the URL, and returns the session
// with an initial screenshot.
func (sm *SessionManager) Open(url string, width, height int) (*Session, *SnapResult, error) {
	sm.mu.Lock()
	if len(sm.sessions) >= MaxSessions {
		sm.mu.Unlock()
		return nil, nil, fmt.Errorf("max sessions reached (%d) — close one first", MaxSessions)
	}
	sm.mu.Unlock()

	if width <= 0 {
		width = 1280
	}
	if height <= 0 {
		height = 800
	}

	page, err := sm.pool.GetPage()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get page: %w", err)
	}

	// Set viewport
	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width: width, Height: height, DeviceScaleFactor: 1,
	}); err != nil {
		sm.pool.ReleasePage(page)
		return nil, nil, fmt.Errorf("viewport failed: %w", err)
	}

	// Navigate
	if err := page.Timeout(15 * time.Second).Navigate(url); err != nil {
		sm.pool.ReleasePage(page)
		return nil, nil, fmt.Errorf("navigation failed: %w", err)
	}

	// Wait for DOM stability
	page.Timeout(4 * time.Second).WaitDOMStable(150*time.Millisecond, 0.15)

	title := safeEvalStr(page, `() => document.title`)

	now := time.Now()
	sess := &Session{
		ID:        generateID(),
		URL:       url,
		Title:     title,
		Width:     width,
		Height:    height,
		CreatedAt: now,
		LastUsed:  now,
		NavCount:  1,
		SnapCount: 0,
		page:      page,
		pool:      sm.pool,
	}

	// Auto-expire timer
	sess.timer = time.AfterFunc(SessionTTL, func() {
		log.Printf("[session] %s expired after %s idle", sess.ID, SessionTTL)
		sm.Close(sess.ID)
	})

	sm.mu.Lock()
	sm.sessions[sess.ID] = sess
	sm.mu.Unlock()

	// Take initial screenshot
	snap, err := sess.Screenshot("jpeg", 80)
	if err != nil {
		sm.Close(sess.ID)
		return nil, nil, fmt.Errorf("initial screenshot failed: %w", err)
	}

	log.Printf("[session] opened %s → %s (%dx%d)", sess.ID, url, width, height)
	return sess, snap, nil
}

// Get returns a session by ID.
func (sm *SessionManager) Get(id string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s, ok := sm.sessions[id]
	return s, ok
}

// Close destroys a session and returns its page to the pool.
func (sm *SessionManager) Close(id string) error {
	sm.mu.Lock()
	sess, ok := sm.sessions[id]
	if !ok {
		sm.mu.Unlock()
		return fmt.Errorf("session %s not found", id)
	}
	delete(sm.sessions, id)
	sm.mu.Unlock()

	sess.mu.Lock()
	defer sess.mu.Unlock()
	if sess.timer != nil {
		sess.timer.Stop()
	}
	if sess.page != nil {
		sm.pool.ReleasePage(sess.page)
		sess.page = nil
	}
	log.Printf("[session] closed %s (navs=%d, snaps=%d, age=%s)", id, sess.NavCount, sess.SnapCount, time.Since(sess.CreatedAt).Round(time.Second))
	return nil
}

// List returns all active sessions.
func (sm *SessionManager) List() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	out := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		out = append(out, s)
	}
	return out
}

// CloseAll destroys all sessions.
func (sm *SessionManager) CloseAll() {
	sm.mu.Lock()
	ids := make([]string, 0, len(sm.sessions))
	for id := range sm.sessions {
		ids = append(ids, id)
	}
	sm.mu.Unlock()
	for _, id := range ids {
		sm.Close(id)
	}
}

// ═══════════════════════════════════════════════════════
// Session actions — all instant (no navigation)
// ═══════════════════════════════════════════════════════

// SnapResult is the response from any session action that returns an image.
type SnapResult struct {
	Screenshot string         `json:"screenshot"` // base64
	Width      int            `json:"width"`
	Height     int            `json:"height"`
	Format     string         `json:"format"`
	Size       int            `json:"size"`
	URL        string         `json:"url"`
	Title      string         `json:"title"`
	Duration   int64          `json:"duration_ms"`
	DOM        map[string]any `json:"dom,omitempty"`
}

func (s *Session) touch() {
	s.LastUsed = time.Now()
	s.SnapCount++
	if s.timer != nil {
		s.timer.Reset(SessionTTL)
	}
}

// Screenshot takes an instant screenshot of the current page state.
// No navigation — just snap what's there.
func (s *Session) Screenshot(format string, quality int) (*SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	start := time.Now()

	f := proto.PageCaptureScreenshotFormatJpeg
	if format == "png" {
		f = proto.PageCaptureScreenshotFormatPng
	}
	if quality <= 0 {
		quality = 60
	}

	data, err := s.page.Screenshot(false, &proto.PageCaptureScreenshot{
		Format:  f,
		Quality: gson(quality),
	})
	if err != nil {
		return nil, fmt.Errorf("screenshot failed: %w", err)
	}

	s.Title = safeEvalStr(s.page, `() => document.title`)
	s.URL = safeEvalStr(s.page, `() => window.location.href`)
	s.touch()

	return &SnapResult{
		Screenshot: base64.StdEncoding.EncodeToString(data),
		Width:      s.Width,
		Height:     s.Height,
		Format:     string(f),
		Size:       len(data),
		URL:        s.URL,
		Title:      s.Title,
		Duration:   time.Since(start).Milliseconds(),
	}, nil
}

// ScreenshotFull takes a full-page screenshot.
func (s *Session) ScreenshotFull(format string, quality int) (*SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	start := time.Now()

	f := proto.PageCaptureScreenshotFormatJpeg
	if format == "png" {
		f = proto.PageCaptureScreenshotFormatPng
	}
	if quality <= 0 {
		quality = 60
	}

	data, err := s.page.Screenshot(true, &proto.PageCaptureScreenshot{
		Format:  f,
		Quality: gson(quality),
	})
	if err != nil {
		return nil, fmt.Errorf("fullpage screenshot failed: %w", err)
	}

	s.touch()

	return &SnapResult{
		Screenshot: base64.StdEncoding.EncodeToString(data),
		Width:      s.Width,
		Height:     -1, // full page, dynamic height
		Format:     string(f),
		Size:       len(data),
		URL:        s.URL,
		Title:      s.Title,
		Duration:   time.Since(start).Milliseconds(),
	}, nil
}

// Scroll scrolls the page by deltaX, deltaY pixels.
func (s *Session) Scroll(deltaX, deltaY int) (*SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	start := time.Now()

	js := fmt.Sprintf(`() => { window.scrollBy(%d, %d); return JSON.stringify({x: window.scrollX, y: window.scrollY}); }`, deltaX, deltaY)
	s.page.Eval(js)
	time.Sleep(100 * time.Millisecond) // let repaint happen

	data, err := s.page.Screenshot(false, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: gson(60),
	})
	if err != nil {
		return nil, err
	}

	s.touch()

	return &SnapResult{
		Screenshot: base64.StdEncoding.EncodeToString(data),
		Width:      s.Width,
		Height:     s.Height,
		Format:     "jpeg",
		Size:       len(data),
		URL:        s.URL,
		Title:      s.Title,
		Duration:   time.Since(start).Milliseconds(),
	}, nil
}

// ScrollTo scrolls to absolute position.
func (s *Session) ScrollTo(x, y int) (*SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	start := time.Now()

	js := fmt.Sprintf(`() => { window.scrollTo(%d, %d); }`, x, y)
	s.page.Eval(js)
	time.Sleep(100 * time.Millisecond)

	data, err := s.page.Screenshot(false, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: gson(60),
	})
	if err != nil {
		return nil, err
	}

	s.touch()

	return &SnapResult{
		Screenshot: base64.StdEncoding.EncodeToString(data),
		Width:      s.Width,
		Height:     s.Height,
		Format:     "jpeg",
		Size:       len(data),
		URL:        s.URL,
		Title:      s.Title,
		Duration:   time.Since(start).Milliseconds(),
	}, nil
}

// Click clicks a CSS-selected element and returns a screenshot after.
func (s *Session) Click(selector string) (*SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	start := time.Now()

	el, err := s.page.Timeout(5 * time.Second).Element(selector)
	if err != nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	if err := el.Click(proto.InputMouseButtonLeft, 1); err != nil {
		return nil, fmt.Errorf("click failed: %w", err)
	}

	// Wait for any triggered navigation or DOM change
	time.Sleep(300 * time.Millisecond)
	s.page.Timeout(2 * time.Second).WaitDOMStable(100*time.Millisecond, 0.2)

	s.URL = safeEvalStr(s.page, `() => window.location.href`)
	s.Title = safeEvalStr(s.page, `() => document.title`)

	data, err := s.page.Screenshot(false, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: gson(60),
	})
	if err != nil {
		return nil, err
	}

	s.touch()

	return &SnapResult{
		Screenshot: base64.StdEncoding.EncodeToString(data),
		Width:      s.Width,
		Height:     s.Height,
		Format:     "jpeg",
		Size:       len(data),
		URL:        s.URL,
		Title:      s.Title,
		Duration:   time.Since(start).Milliseconds(),
	}, nil
}

// Hover hovers over a CSS-selected element.
func (s *Session) Hover(selector string) (*SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	start := time.Now()

	el, err := s.page.Timeout(5 * time.Second).Element(selector)
	if err != nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	if err := el.Hover(); err != nil {
		return nil, fmt.Errorf("hover failed: %w", err)
	}
	time.Sleep(200 * time.Millisecond)

	data, err := s.page.Screenshot(false, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: gson(60),
	})
	if err != nil {
		return nil, err
	}

	s.touch()

	return &SnapResult{
		Screenshot: base64.StdEncoding.EncodeToString(data),
		Width:      s.Width,
		Height:     s.Height,
		Format:     "jpeg",
		Size:       len(data),
		URL:        s.URL,
		Title:      s.Title,
		Duration:   time.Since(start).Milliseconds(),
	}, nil
}

// Type types text into a CSS-selected input element.
func (s *Session) Type(selector, text string) (*SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	start := time.Now()

	el, err := s.page.Timeout(5 * time.Second).Element(selector)
	if err != nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	el.SelectAllText()
	el.Input(text)
	time.Sleep(100 * time.Millisecond)

	data, err := s.page.Screenshot(false, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: gson(60),
	})
	if err != nil {
		return nil, err
	}

	s.touch()

	return &SnapResult{
		Screenshot: base64.StdEncoding.EncodeToString(data),
		Width:      s.Width,
		Height:     s.Height,
		Format:     "jpeg",
		Size:       len(data),
		URL:        s.URL,
		Title:      s.Title,
		Duration:   time.Since(start).Milliseconds(),
	}, nil
}

// Eval runs JavaScript and returns the result string + a screenshot.
func (s *Session) Eval(js string) (string, *SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return "", nil, fmt.Errorf("session closed")
	}

	start := time.Now()

	result, err := s.page.Eval(`() => { ` + js + ` }`)
	jsResult := ""
	if err != nil {
		jsResult = "error: " + err.Error()
	} else if result != nil {
		jsResult = result.Value.Str()
		if jsResult == "" {
			raw, _ := result.Value.MarshalJSON()
			jsResult = string(raw)
		}
	}

	time.Sleep(100 * time.Millisecond) // let DOM settle after JS execution

	data, snapErr := s.page.Screenshot(false, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: gson(60),
	})
	if snapErr != nil {
		return jsResult, nil, nil // return JS result even if screenshot fails
	}

	s.URL = safeEvalStr(s.page, `() => window.location.href`)
	s.Title = safeEvalStr(s.page, `() => document.title`)
	s.touch()

	return jsResult, &SnapResult{
		Screenshot: base64.StdEncoding.EncodeToString(data),
		Width:      s.Width,
		Height:     s.Height,
		Format:     "jpeg",
		Size:       len(data),
		URL:        s.URL,
		Title:      s.Title,
		Duration:   time.Since(start).Milliseconds(),
	}, nil
}

// Navigate goes to a new URL within the same session.
func (s *Session) Navigate(url string) (*SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	start := time.Now()

	if err := s.page.Timeout(15 * time.Second).Navigate(url); err != nil {
		return nil, fmt.Errorf("navigation failed: %w", err)
	}

	s.page.Timeout(4 * time.Second).WaitDOMStable(150*time.Millisecond, 0.15)

	s.URL = url
	s.Title = safeEvalStr(s.page, `() => document.title`)
	s.NavCount++

	data, err := s.page.Screenshot(false, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: gson(60),
	})
	if err != nil {
		return nil, err
	}

	s.touch()

	return &SnapResult{
		Screenshot: base64.StdEncoding.EncodeToString(data),
		Width:      s.Width,
		Height:     s.Height,
		Format:     "jpeg",
		Size:       len(data),
		URL:        s.URL,
		Title:      s.Title,
		Duration:   time.Since(start).Milliseconds(),
	}, nil
}

// Resize changes the viewport and returns a screenshot.
func (s *Session) Resize(width, height int) (*SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	start := time.Now()

	if err := s.page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width: width, Height: height, DeviceScaleFactor: 1,
	}); err != nil {
		return nil, err
	}
	s.Width = width
	s.Height = height

	time.Sleep(200 * time.Millisecond) // let responsive CSS reflow

	data, err := s.page.Screenshot(false, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: gson(60),
	})
	if err != nil {
		return nil, err
	}

	s.touch()

	return &SnapResult{
		Screenshot: base64.StdEncoding.EncodeToString(data),
		Width:      s.Width,
		Height:     s.Height,
		Format:     "jpeg",
		Size:       len(data),
		URL:        s.URL,
		Title:      s.Title,
		Duration:   time.Since(start).Milliseconds(),
	}, nil
}

// InjectCSS injects or replaces CSS and returns a screenshot.
func (s *Session) InjectCSS(css string) (*SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	start := time.Now()

	js := `(css) => {
		const old = document.getElementById('llm-injected-css');
		if (old) old.remove();
		const s = document.createElement('style');
		s.textContent = css;
		s.id = 'llm-injected-css';
		document.head.appendChild(s);
		return 'ok';
	}`
	s.page.Eval(js, css)
	time.Sleep(100 * time.Millisecond)

	data, err := s.page.Screenshot(false, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: gson(60),
	})
	if err != nil {
		return nil, err
	}

	s.touch()

	return &SnapResult{
		Screenshot: base64.StdEncoding.EncodeToString(data),
		Width:      s.Width,
		Height:     s.Height,
		Format:     "jpeg",
		Size:       len(data),
		URL:        s.URL,
		Title:      s.Title,
		Duration:   time.Since(start).Milliseconds(),
	}, nil
}

// DOMInfo returns structured DOM information about the current page.
func (s *Session) DOMInfo() (map[string]any, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	js := `() => JSON.stringify({
		url: location.href,
		title: document.title,
		scroll: { x: scrollX, y: scrollY, maxY: document.body.scrollHeight - innerHeight },
		viewport: { width: innerWidth, height: innerHeight, scrollHeight: document.body.scrollHeight },
		headings: Array.from(document.querySelectorAll('h1,h2,h3')).slice(0,15).map(e => ({tag: e.tagName, text: e.textContent.trim().substring(0,100)})),
		links: document.querySelectorAll('a[href]').length,
		buttons: document.querySelectorAll('button,[role=button]').length,
		images: { total: document.images.length, broken: Array.from(document.images).filter(i => !i.complete || !i.naturalWidth).length },
		forms: document.forms.length,
		inputs: document.querySelectorAll('input,textarea,select').length,
		interactive: Array.from(document.querySelectorAll('button,[role=button],a[href],input,select,textarea,[tabindex]')).slice(0,30).map(e => ({
			tag: e.tagName.toLowerCase(),
			type: e.type || '',
			text: (e.textContent || e.value || e.placeholder || e.alt || '').trim().substring(0,60),
			selector: e.id ? '#'+e.id : (e.className ? e.tagName.toLowerCase()+'.'+e.className.split(' ')[0] : e.tagName.toLowerCase()),
			visible: e.offsetParent !== null
		}))
	})`

	result, err := s.page.Eval(js)
	if err != nil {
		return nil, err
	}

	var dom map[string]any
	if err := json.Unmarshal([]byte(result.Value.Str()), &dom); err != nil {
		return nil, err
	}

	s.touch()
	return dom, nil
}

// WaitFor waits for a CSS selector to appear, then screenshots.
func (s *Session) WaitFor(selector string, timeoutMs int) (*SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	start := time.Now()

	if timeoutMs <= 0 {
		timeoutMs = 5000
	}

	_, err := s.page.Timeout(time.Duration(timeoutMs) * time.Millisecond).Element(selector)
	if err != nil {
		return nil, fmt.Errorf("wait timeout: %s not found within %dms", selector, timeoutMs)
	}

	data, err := s.page.Screenshot(false, &proto.PageCaptureScreenshot{
		Format:  proto.PageCaptureScreenshotFormatJpeg,
		Quality: gson(60),
	})
	if err != nil {
		return nil, err
	}

	s.touch()

	return &SnapResult{
		Screenshot: base64.StdEncoding.EncodeToString(data),
		Width:      s.Width,
		Height:     s.Height,
		Format:     "jpeg",
		Size:       len(data),
		URL:        s.URL,
		Title:      s.Title,
		Duration:   time.Since(start).Milliseconds(),
	}, nil
}

// ═══════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════

func safeEvalStr(page *rod.Page, js string) string {
	result, err := page.Timeout(2 * time.Second).Eval(js)
	if err != nil || result == nil {
		return ""
	}
	s := result.Value.Str()
	if s == "" {
		raw, _ := result.Value.MarshalJSON()
		return strings.Trim(string(raw), `"`)
	}
	return s
}
