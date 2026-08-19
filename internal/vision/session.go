package vision

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/focusapacket"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// MaxSessions is the maximum number of concurrent browser sessions.
// Each session holds a Chrome page (~50-100MB).
const MaxSessions = 4

// SessionTTL is how long an idle session stays alive before auto-cleanup.
const SessionTTL = 10 * time.Minute

const (
	sessionOpenAttempts = 2
	elementRetryBudget  = 5 * time.Second
	elementRetryStep    = 175 * time.Millisecond
)

// Session is a persistent browser page with identity.
// Unlike the transactional pool (navigate → snap → forget), a session keeps
// the page alive between calls — enabling instant re-screenshots, scrolling,
// clicking, CSS injection, and JS evaluation without re-navigating.
type FocusaScope struct {
	WorkpointID  string `json:"workpoint_id,omitempty"`
	ContinuityID string `json:"continuity_id,omitempty"`
	ProjectRoot  string `json:"project_root,omitempty"`
	EvidenceRef  string `json:"evidence_ref,omitempty"`
}

// Spec104ScopeRef is the typed scope key for UIAI Engine (Spec 104 §6.1, §7.2).
// Kind "project" => ID is project_root; Kind "host" => ID is host identifier.
type ScopeKind string

const (
	ScopeKindProject ScopeKind = "project"
	ScopeKindHost    ScopeKind = "host"
)

type ScopeRef struct {
	Kind ScopeKind `json:"kind"`
	ID   string    `json:"id"`
}

func (s ScopeRef) String() string { return string(s.Kind) + ":" + s.ID }
func DefaultHostScope() ScopeRef  { return ScopeRef{Kind: ScopeKindHost, ID: "loopback"} }

type Session struct {
	ID            string       `json:"id"`
	URL           string       `json:"url"`
	Title         string       `json:"title"`
	Width         int          `json:"width"`
	Height        int          `json:"height"`
	CreatedAt     time.Time    `json:"created_at"`
	LastUsed      time.Time    `json:"last_used"`
	NavCount      int          `json:"nav_count"`
	SnapCount     int          `json:"snap_count"`
	ReadCount     int          `json:"read_count"`
	SnapshotCount int          `json:"snapshot_count"`
	Scope         ScopeRef     `json:"scope"`
	FocusaScope   *FocusaScope `json:"focusa_scope,omitempty"`

	page              *rod.Page
	pool              PoolSource
	timer             *time.Timer
	mu                sync.Mutex
	refs              map[string]SnapshotRef // @ref → CSS selector, populated by Snapshot()
	diagnostics       *diagnosticsRecorder
	diagnosticsCancel func()
}

// SessionManager manages persistent browser sessions.
// Spec 104: authority-bearing singleton eliminated — sessions are indexed by (scope, id) and MaxSessions is enforced per-scope with a global cap.
type SessionManager struct {
	mu       sync.RWMutex
	sessions map[string]*Session // id → Session (Scope inside)
	pool     PoolSource
}

const globalMaxSessions = 16 // hard global cap across all scopes

func (sm *SessionManager) countForScopeLocked(scope ScopeRef) int {
	n := 0
	for _, s := range sm.sessions {
		if s.Scope == scope {
			n++
		}
	}
	return n
}

// NewSessionManager creates a session manager backed by a single vision pool.
func NewSessionManager(pool *Pool) *SessionManager {
	return &SessionManager{
		sessions: make(map[string]*Session),
		pool:     pool,
	}
}

// NewSessionManagerWithPools creates a session manager backed by a slice of independent pools.
// Pages are sourced from the pool whose GetPage() returns a page; this requires each
// Pool's *rod.Page to be tracked alongside the Session by extending PoolSource (already
// done via the implementation methods).
func NewSessionManagerWithPools(pool PoolSource) *SessionManager {
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
	sess := &Session{
		ID:          generateID(),
		URL:         url,
		Title:       title,
		Width:       width,
		Height:      height,
		CreatedAt:   now,
		LastUsed:    now,
		page:        page,
		refs:        make(map[string]SnapshotRef),
		diagnostics: newDiagnosticsRecorder(),
	}
	sess.initDiagnostics()
	return sess
}

// generateID creates a short unique session ID.
func generateID() string {
	b := make([]byte, 6)
	if _, err := cryptorand.Read(b); err != nil {
		return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%06x", time.Now().UnixNano()))[:6])
	}
	s := base64.RawURLEncoding.EncodeToString(b)
	if len(s) > 8 {
		s = s[:8]
	}
	return s
}

func (s *Session) SetFocusaScope(scope *FocusaScope) {
	if scope == nil || (scope.WorkpointID == "" && scope.ContinuityID == "" && scope.ProjectRoot == "" && scope.EvidenceRef == "") {
		s.FocusaScope = nil
		return
	}
	s.FocusaScope = scope
}

// Open creates a new session with the default host scope (backwards compat).
// New code should use OpenScoped with an explicit ScopeRef per Spec 104 §6.1.
func (sm *SessionManager) Open(url string, width, height int) (*Session, *SnapResult, error) {
	return sm.OpenScoped(url, width, height, DefaultHostScope())
}

// OpenScoped creates a session under an explicit typed scope (Spec 104 §7.2).
// MaxSessions is enforced per-scope (4) plus a global cap (16); pool-full errors include scope.
func (sm *SessionManager) OpenScoped(url string, width, height int, scope ScopeRef) (*Session, *SnapResult, error) {
	if err := sm.pool.ValidateNavigationURL(url); err != nil {
		return nil, nil, err
	}

	sm.mu.Lock()
	perScope := sm.countForScopeLocked(scope)
	if perScope >= MaxSessions {
		sm.mu.Unlock()
		return nil, nil, fmt.Errorf("max sessions for scope %s reached (%d) — close one in that scope first (global %d/%d)", scope.String(), MaxSessions, len(sm.sessions), globalMaxSessions)
	}
	if len(sm.sessions) >= globalMaxSessions {
		sm.mu.Unlock()
		return nil, nil, fmt.Errorf("global max sessions reached (%d) — close one first", globalMaxSessions)
	}
	sm.mu.Unlock()

	if width <= 0 {
		width = 1280
	}
	if height <= 0 {
		height = 800
	}

	var lastErr error
	for attempt := 1; attempt <= sessionOpenAttempts; attempt++ {
		sess, snap, err := sm.openOnceScoped(url, width, height, scope)
		if err == nil {
			if attempt > 1 {
				log.Printf("[session] open recovered on attempt %d → %s", attempt, url)
			}
			return sess, snap, nil
		}
		lastErr = err
		if attempt < sessionOpenAttempts {
			time.Sleep(time.Duration(attempt) * 250 * time.Millisecond)
		}
	}
	return nil, nil, fmt.Errorf("session open failed after %d attempts: %w", sessionOpenAttempts, lastErr)
}

func (sm *SessionManager) openOnce(url string, width, height int) (*Session, *SnapResult, error) {
	return sm.openOnceScoped(url, width, height, DefaultHostScope())
}

func (sm *SessionManager) openOnceScoped(url string, width, height int, scope ScopeRef) (*Session, *SnapResult, error) {
	page, err := sm.pool.GetPage()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get page: %w", err)
	}

	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{Width: width, Height: height, DeviceScaleFactor: 1}); err != nil {
		sm.pool.ReleasePage(page)
		return nil, nil, fmt.Errorf("viewport failed: %w", err)
	}

	now := time.Now()
	sess := &Session{ID: generateID(), URL: url, Width: width, Height: height, CreatedAt: now, LastUsed: now, Scope: scope, page: page, pool: sm.pool, refs: make(map[string]SnapshotRef), diagnostics: newDiagnosticsRecorder()}
	sess.initDiagnostics()

	if err := page.Timeout(18 * time.Second).Navigate(url); err != nil {
		if sess.diagnosticsCancel != nil {
			sess.diagnosticsCancel()
		}
		sm.pool.ReleasePage(page)
		return nil, nil, fmt.Errorf("navigation failed: %w", err)
	}

	page.Timeout(4*time.Second).WaitDOMStable(150*time.Millisecond, 0.15)
	sess.Title = safeEvalStr(page, `() => document.title`)
	sess.NavCount = 1
	sess.timer = time.AfterFunc(SessionTTL, func() {
		log.Printf("[session] %s expired after %s idle", sess.ID, SessionTTL)
		sm.Close(sess.ID)
	})

	sm.mu.Lock()
	sm.sessions[sess.ID] = sess
	sm.mu.Unlock()

	snap, err := sess.Screenshot("jpeg", 80)
	if err != nil {
		sm.Close(sess.ID)
		return nil, nil, fmt.Errorf("initial screenshot failed: %w", err)
	}
	log.Printf("[session] opened %s → %s (%dx%d)", sess.ID, url, width, height)
	return sess, snap, nil
}

func retryElement(page *rod.Page, selector string) (*rod.Element, error) {
	deadline := time.Now().Add(elementRetryBudget)
	var lastErr error
	for attempt := 1; ; attempt++ {
		el, err := page.Timeout(elementRetryStep).Element(selector)
		if err == nil {
			return el, nil
		}
		lastErr = err
		if time.Now().Add(elementRetryStep).After(deadline) {
			break
		}
		backoff := time.Duration(attempt) * 75 * time.Millisecond
		if backoff > 300*time.Millisecond {
			backoff = 300 * time.Millisecond
		}
		time.Sleep(backoff)
	}
	return nil, fmt.Errorf("element not found after %s: %s (%w)", elementRetryBudget.Round(time.Millisecond), selector, lastErr)
}

func retryNavigate(page *rod.Page, url string) error {
	var lastErr error
	for attempt := 1; attempt <= 2; attempt++ {
		err := page.Timeout(15 * time.Second).Navigate(url)
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt < 2 {
			time.Sleep(250 * time.Millisecond)
		}
	}
	return fmt.Errorf("navigation failed after 2 attempts: %w", lastErr)
}

// Get returns a session by ID (scope-agnostic, backwards compat).
func (sm *SessionManager) Get(id string) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s, ok := sm.sessions[id]
	return s, ok
}

// GetScoped returns a session only if scope matches (Spec 104 §7.2).
func (sm *SessionManager) GetScoped(id string, scope ScopeRef) (*Session, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	s, ok := sm.sessions[id]
	if !ok || s.Scope != scope {
		return nil, false
	}
	return s, true
}

// CountForScope returns live session count for a given scope.
func (sm *SessionManager) CountForScope(scope ScopeRef) int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.countForScopeLocked(scope)
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
	if sess.diagnosticsCancel != nil {
		sess.diagnosticsCancel()
		sess.diagnosticsCancel = nil
	}
	if sess.page != nil {
		sm.pool.ReleasePage(sess.page)
		sess.page = nil
	}
	log.Printf("[session] closed %s (navs=%d, snaps=%d, age=%s)", id, sess.NavCount, sess.SnapCount, time.Since(sess.CreatedAt).Round(time.Second))
	return nil
}

// List returns all active sessions (includes Scope per-spec).
func (sm *SessionManager) List() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	out := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		out = append(out, s)
	}
	return out
}

// ListScoped returns sessions for a single typed scope.
func (sm *SessionManager) ListScoped(scope ScopeRef) []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	out := []*Session{}
	for _, s := range sm.sessions {
		if s.Scope == scope {
			out = append(out, s)
		}
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

	data, err := s.page.Timeout(12*time.Second).Screenshot(false, &proto.PageCaptureScreenshot{
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

// CDPScreencast starts Chrome DevTools Protocol screencast and returns JPEG frames.
func (s *Session) CDPScreencast(ctx context.Context, quality int, everyNthFrame int) (<-chan []byte, func(), error) {
	s.mu.Lock()
	if s.page == nil {
		s.mu.Unlock()
		return nil, nil, fmt.Errorf("session closed")
	}
	page := s.page
	s.mu.Unlock()
	if quality <= 0 || quality > 100 {
		quality = 60
	}
	if everyNthFrame <= 0 {
		everyNthFrame = 1
	}
	frames := make(chan []byte, 8)
	ctx, cancel := context.WithCancel(ctx)
	wait := page.Context(ctx).EachEvent(func(e *proto.PageScreencastFrame) {
		_ = proto.PageScreencastFrameAck{SessionID: e.SessionID}.Call(page)
		select {
		case frames <- e.Data:
		default:
		}
	})
	done := make(chan struct{})
	go func() {
		wait()
		close(done)
		close(frames)
	}()
	if err := (proto.PageStartScreencast{Format: proto.PageStartScreencastFormatJpeg, Quality: &quality, EveryNthFrame: &everyNthFrame}).Call(page); err != nil {
		cancel()
		<-done
		return nil, nil, err
	}
	stop := func() {
		_ = proto.PageStopScreencast{}.Call(page)
		cancel()
		<-done
	}
	return frames, stop, nil
}

// SelectorAt returns a best-effort selector for the element at viewport coordinates.
func (s *Session) SelectorAt(x, y int) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return "", fmt.Errorf("session closed")
	}
	js := fmt.Sprintf(`() => {
		const el = document.elementFromPoint(%d, %d);
		if (!el) return "";
		if (el.id) return "#" + CSS.escape(el.id);
		const attr = ["data-testid","data-test","aria-label","name","role"].find(a => el.getAttribute(a));
		if (attr) return el.tagName.toLowerCase() + "[" + attr + "=" + JSON.stringify(el.getAttribute(attr)) + "]";
		const text = (el.innerText || el.textContent || "").trim().replace(/\s+/g, " ").slice(0, 60);
		if (text) return "text=" + text;
		let path = el.tagName.toLowerCase();
		let n = el;
		while (n.parentElement && n.parentElement !== document.body && path.length < 120) {
			const parent = n.parentElement;
			const idx = Array.from(parent.children).indexOf(n) + 1;
			path = parent.tagName.toLowerCase() + " > " + path + ":nth-child(" + idx + ")";
			n = parent;
		}
		return path;
	}`, x, y)
	res, err := s.page.Eval(js)
	if err != nil {
		return "", err
	}
	return res.Value.Str(), nil
}

// ClickAt clicks the element at viewport coordinates and returns a screenshot after.
func (s *Session) ClickAt(x, y int) (*SnapResult, string, error) {
	selector, err := s.SelectorAt(x, y)
	if err != nil {
		return nil, "", err
	}
	if selector == "" {
		return nil, "", fmt.Errorf("no element at %d,%d", x, y)
	}
	snap, err := s.Click(selector)
	return snap, selector, err
}

// Click clicks a CSS-selected element and returns a screenshot after.
func (s *Session) Click(selector string) (*SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	start := time.Now()

	el, err := retryElement(s.page, selector)
	if err != nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	if _, err := el.Eval(`() => {
		this.scrollIntoView({ block: 'center', inline: 'center' });
		this.click();
		return true;
	}`); err != nil {
		return nil, fmt.Errorf("click failed: %w", err)
	}

	// Wait for any triggered navigation or DOM change
	time.Sleep(300 * time.Millisecond)
	s.page.Timeout(2*time.Second).WaitDOMStable(100*time.Millisecond, 0.2)

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

	el, err := retryElement(s.page, selector)
	if err != nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	if err := el.Timeout(3 * time.Second).Hover(); err != nil {
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

	el, err := retryElement(s.page, selector)
	if err != nil {
		return nil, fmt.Errorf("element not found: %s", selector)
	}

	el.SelectAllText()
	el.Timeout(3 * time.Second).Input(text)
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

	result, err := s.page.Timeout(30 * time.Second).Eval(`() => { ` + js + ` }`)
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

// EvalAsync runs bounded async JavaScript and returns the awaited result string + a screenshot.
func (s *Session) EvalAsync(js string, timeoutMs int) (string, *SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return "", nil, fmt.Errorf("session closed")
	}
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	if timeoutMs > 15000 {
		timeoutMs = 15000
	}

	start := time.Now()
	wrapper := fmt.Sprintf(`async () => {
		const __timeout = new Promise((_, reject) => setTimeout(() => reject(new Error("eval_async timeout after %dms")), %d));
		const __work = (async () => { %s })();
		return await Promise.race([__work, __timeout]);
	}`, timeoutMs, timeoutMs, js)
	result, err := s.page.Timeout(time.Duration(timeoutMs+1000) * time.Millisecond).Eval(wrapper)
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

	time.Sleep(100 * time.Millisecond)
	data, snapErr := s.page.Screenshot(false, &proto.PageCaptureScreenshot{Format: proto.PageCaptureScreenshotFormatJpeg, Quality: gson(60)})
	if snapErr != nil {
		return jsResult, nil, nil
	}
	s.URL = safeEvalStr(s.page, `() => window.location.href`)
	s.Title = safeEvalStr(s.page, `() => document.title`)
	s.touch()
	return jsResult, &SnapResult{Screenshot: base64.StdEncoding.EncodeToString(data), Width: s.Width, Height: s.Height, Format: "jpeg", Size: len(data), URL: s.URL, Title: s.Title, Duration: time.Since(start).Milliseconds()}, nil
}

// Navigate goes to a new URL within the same session.
func (s *Session) Navigate(url string) (*SnapResult, error) {
	if err := s.pool.ValidateNavigationURL(url); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	start := time.Now()

	if err := retryNavigate(s.page, url); err != nil {
		return nil, err
	}

	s.page.Timeout(4*time.Second).WaitDOMStable(150*time.Millisecond, 0.15)

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

// ReadOptions controls bounded page text extraction for agent web surfing.
type ReadOptions struct {
	Selector      string `json:"selector,omitempty"`
	MaxChars      int    `json:"max_chars,omitempty"`
	IncludeLinks  bool   `json:"include_links,omitempty"`
	Format        string `json:"format,omitempty"`
	Mode          string `json:"mode,omitempty"`
	IncludeImages bool   `json:"include_images,omitempty"`
}

// PageReadResult is a compact, screenshot-free page reading payload.
type PageReadResult struct {
	Schema      string              `json:"schema,omitempty"`
	URL         string              `json:"url"`
	Title       string              `json:"title"`
	Description string              `json:"description,omitempty"`
	Selector    string              `json:"selector,omitempty"`
	Format      string              `json:"format,omitempty"`
	Mode        string              `json:"mode,omitempty"`
	Text        string              `json:"text"`
	Chars       int                 `json:"chars"`
	Truncated   bool                `json:"truncated"`
	Headings    []map[string]any    `json:"headings,omitempty"`
	Links       []map[string]any    `json:"links,omitempty"`
	Metadata    map[string]any      `json:"metadata,omitempty"`
	Focusa      *ReadFocusaMetadata `json:"focusa,omitempty"`
}

type ReadFocusaMetadata struct {
	TargetRef         string   `json:"target_ref"`
	EvidenceRef       string   `json:"evidence_ref"`
	PreferredTool     string   `json:"preferred_tool"`
	Summary           string   `json:"summary"`
	NextTools         []string `json:"next_tools"`
	FocusaScopeStatus string   `json:"focusa_scope_status"`
}

// ReadPage extracts bounded, readable page text without taking a screenshot.
func (s *Session) ReadPage(opts ReadOptions) (*PageReadResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}

	if opts.MaxChars <= 0 {
		opts.MaxChars = 8000
	}
	if opts.MaxChars > 30000 {
		opts.MaxChars = 30000
	}
	format := strings.ToLower(strings.TrimSpace(opts.Format))
	if format == "" {
		format = "text"
	}
	if format != "text" && format != "markdown" {
		return nil, fmt.Errorf("unsupported_read_format: %s", opts.Format)
	}
	mode := strings.ToLower(strings.TrimSpace(opts.Mode))
	if mode == "" {
		mode = "main_content"
	}

	selectorJSON, _ := json.Marshal(opts.Selector)
	formatJSON, _ := json.Marshal(format)
	modeJSON, _ := json.Marshal(mode)
	includeImagesJSON, _ := json.Marshal(opts.IncludeImages)
	js := fmt.Sprintf(`() => {
		const selector = %s;
		const format = %s;
		const mode = %s;
		const includeImages = %s;
		const root = selector ? document.querySelector(selector) : (mode === 'full' ? (document.body || document.documentElement) : (document.querySelector('main, article, [role="main"]') || document.body || document.documentElement));
		if (!root) return JSON.stringify({ error: 'selector_not_found', selector });
		const hidden = (el) => {
			const style = window.getComputedStyle(el);
			return style.display === 'none' || style.visibility === 'hidden' || el.getAttribute('aria-hidden') === 'true';
		};
		const clean = (s) => (s || '').replace(/[ \t\n]+/g, ' ').trim();
		const tick = String.fromCharCode(96);
		const esc = (s) => clean(s).replace(/\\/g, '\\\\').replace(new RegExp('([*_\\x60~\\[\\]#])', 'g'), '\\$1');
		const clone = root.cloneNode(true);
		clone.querySelectorAll('script,style,noscript,svg,canvas,iframe,template').forEach(e => e.remove());
		if (mode !== 'full') clone.querySelectorAll('nav,footer,header,aside').forEach(e => e.remove());
		clone.querySelectorAll('[hidden],[aria-hidden="true"]').forEach(e => e.remove());
		const mdNode = (node, depth = 0) => {
			if (!node) return '';
			if (node.nodeType === Node.TEXT_NODE) return esc(node.textContent);
			if (node.nodeType !== Node.ELEMENT_NODE || hidden(node)) return '';
			const tag = node.tagName.toLowerCase();
			const kids = () => Array.from(node.childNodes).map(n => mdNode(n, depth)).join('').replace(/[ \t]+\n/g, '\n').trim();
			if (/^h[1-6]$/.test(tag)) return '\n\n' + '#'.repeat(Number(tag[1])) + ' ' + esc(node.innerText || node.textContent) + '\n\n';
			if (tag === 'p') return '\n\n' + kids() + '\n\n';
			if (tag === 'br') return '\n';
			if (tag === 'strong' || tag === 'b') return '**' + kids() + '**';
			if (tag === 'em' || tag === 'i') return '*' + kids() + '*';
			if (tag === 'code') return tick + clean(node.textContent).replaceAll(tick, '\\' + tick) + tick;
			if (tag === 'pre') return '\n\n' + tick + tick + tick + '\n' + (node.textContent || '').trim() + '\n' + tick + tick + tick + '\n\n';
			if (tag === 'blockquote') return '\n\n' + (node.innerText || '').split('\n').map(l => '> ' + esc(l)).join('\n') + '\n\n';
			if (tag === 'a') { const label = kids() || esc(node.href); return node.href ? '[' + label + '](' + node.href + ')' : label; }
			if (tag === 'img') { if (!includeImages) return ''; const alt = esc(node.getAttribute('alt') || 'image'); const src = node.currentSrc || node.src || ''; return src ? '![' + alt + '](' + src + ')' : alt; }
			if (tag === 'li') return '\n' + '  '.repeat(depth) + '- ' + Array.from(node.childNodes).map(n => mdNode(n, depth + 1)).join('').trim();
			if (tag === 'ul' || tag === 'ol') return '\n' + Array.from(node.children).map(n => mdNode(n, depth)).join('') + '\n';
			if (tag === 'tr') return Array.from(node.children).map(c => clean(c.innerText || c.textContent)).join(' | ') + '\n';
			if (tag === 'table') return '\n\n' + Array.from(node.querySelectorAll('tr')).map(r => mdNode(r, depth)).join('').trim() + '\n\n';
			return Array.from(node.childNodes).map(n => mdNode(n, depth)).join('\n').replace(/\n{3,}/g, '\n\n').trim();
		};
		const plainText = (clone.innerText || clone.textContent || '').replace(/[ \t]+/g, ' ').replace(/\n{3,}/g, '\n\n').trim();
		const markdown = mdNode(clone).replace(/[ \t]+\n/g, '\n').replace(/\n{3,}/g, '\n\n').trim();
		const text = format === 'markdown' ? markdown : plainText;
		const headings = Array.from(root.querySelectorAll('h1,h2,h3')).filter(e => !hidden(e)).slice(0, 20).map(e => ({ level: e.tagName.toLowerCase(), text: (e.innerText || e.textContent || '').trim().substring(0, 160) })).filter(h => h.text);
		const links = Array.from(root.querySelectorAll('a[href]')).filter(e => !hidden(e)).slice(0, 40).map(e => ({ text: (e.innerText || e.textContent || '').trim().substring(0, 120), href: e.href })).filter(l => l.href);
		const description = document.querySelector('meta[name="description"], meta[property="og:description"]')?.content || '';
		const canonical = document.querySelector('link[rel="canonical"]')?.href || location.href;
		const siteName = document.querySelector('meta[property="og:site_name"]')?.content || location.hostname;
		return JSON.stringify({ url: location.href, canonical_url: canonical, site_name: siteName, title: document.title, description, selector, format, mode, text, headings, links });
	}`, string(selectorJSON), string(formatJSON), string(modeJSON), string(includeImagesJSON))

	result, err := s.page.Timeout(5 * time.Second).Eval(js)
	if err != nil {
		return nil, err
	}

	var raw struct {
		Error        string           `json:"error"`
		URL          string           `json:"url"`
		Title        string           `json:"title"`
		Description  string           `json:"description"`
		Selector     string           `json:"selector"`
		Format       string           `json:"format"`
		Mode         string           `json:"mode"`
		CanonicalURL string           `json:"canonical_url"`
		SiteName     string           `json:"site_name"`
		Text         string           `json:"text"`
		Headings     []map[string]any `json:"headings"`
		Links        []map[string]any `json:"links"`
	}
	if err := json.Unmarshal([]byte(result.Value.Str()), &raw); err != nil {
		return nil, err
	}
	if raw.Error != "" {
		return nil, fmt.Errorf("%s: %s", raw.Error, opts.Selector)
	}

	text := strings.TrimSpace(raw.Text)
	truncated := false
	if len(text) > opts.MaxChars {
		text = text[:opts.MaxChars]
		truncated = true
	}

	s.ReadCount++
	readSeq := s.ReadCount
	out := &PageReadResult{
		Schema:      "uiai.browser_read.v2",
		URL:         raw.URL,
		Title:       raw.Title,
		Description: raw.Description,
		Selector:    raw.Selector,
		Format:      format,
		Mode:        mode,
		Text:        text,
		Chars:       len(text),
		Truncated:   truncated,
		Headings:    raw.Headings,
		Metadata: map[string]any{
			"source_type":   "webpage",
			"canonical_url": raw.CanonicalURL,
			"site_name":     raw.SiteName,
			"captured_at":   time.Now().UTC().Format(time.RFC3339),
		},
		Focusa: buildReadFocusaMetadata(s.ID, readSeq, raw.URL, raw.Title, raw.Selector, len(text), truncated, format, s.FocusaScope),
	}
	if opts.IncludeLinks {
		out.Links = raw.Links
	}

	s.touch()
	return out, nil
}

func buildReadFocusaMetadata(sessionID string, readSeq int, pageURL, title, selector string, chars int, truncated bool, format string, scope *FocusaScope) *ReadFocusaMetadata {
	if readSeq <= 0 {
		readSeq = 1
	}
	targetRef := "browser:" + focusapacket.SanitizeURL(pageURL)
	if pageURL == "" {
		targetRef = "browser:session=" + focusapacket.Truncate(sessionID, 80)
	}
	format = strings.ToLower(strings.TrimSpace(format))
	label := "Read"
	if format == "markdown" {
		label = "Read Markdown"
	}
	summary := fmt.Sprintf("%s %d chars from %s", label, chars, focusapacket.Truncate(safePageLabel(title, pageURL, sessionID), 160))
	if selector != "" {
		summary += " selector=" + focusapacket.Truncate(selector, 80)
	}
	if truncated {
		summary += " (truncated)"
	}
	return &ReadFocusaMetadata{
		TargetRef:         targetRef,
		EvidenceRef:       fmt.Sprintf("uiai-browser:session=%s:read:%d", focusapacket.Truncate(sessionID, 80), readSeq),
		PreferredTool:     "focusa_evidence_capture",
		Summary:           focusapacket.Truncate(summary, focusapacket.MaxCaptureSummaryChars),
		NextTools:         []string{"focusa_evidence_capture", "focusa_active_object_resolve", "focusa_predict_record"},
		FocusaScopeStatus: readFocusaScopeStatus(scope),
	}
}

func readFocusaScopeStatus(scope *FocusaScope) string {
	if scope == nil {
		return string(focusapacket.ScopeMissing)
	}
	if scope.ProjectRoot != "" && scope.ContinuityID != "" {
		return string(focusapacket.ScopePresent)
	}
	return string(focusapacket.ScopePartial)
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
