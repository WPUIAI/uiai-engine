package vision

import (
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
)

// acquirePageWithTimeout bounds browser acquisition so CI runners without a
// working local chromium skip instead of hanging the suite (Windows CI).
func acquirePageWithTimeout(t *testing.T, p *Pool, d time.Duration) *rod.Page {
	t.Helper()
	ch := make(chan *rod.Page, 1)
	errCh := make(chan error, 1)
	go func() {
		page, err := p.GetPage()
		if err != nil {
			errCh <- err
			return
		}
		ch <- page
	}()
	select {
	case page := <-ch:
		return page
	case err := <-errCh:
		t.Skipf("browser unavailable, skipping input-delivery test: %v", err)
		return nil
	case <-time.After(d):
		t.Skip("browser did not launch in time; skipping input-delivery test")
		return nil
	}
}


// TestPressDispatchesKeydown isolates rod keyboard dispatch through the
// engine's pool/session path: if Press works, the page-level keydown listener
// must see the event. (Regression harness for the #225/#226 turn.)
func TestPressDispatchesKeydown(t *testing.T) {
	p, err := NewPoolWithConfig(PoolConfig{MaxPages: 2, AllowPrivateURLs: true})
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer p.Close()
	page := acquirePageWithTimeout(t, p, 60*time.Second)
	defer p.ReleasePage(page)

	if err := page.Timeout(20 * time.Second).Navigate("https://example.com"); err != nil {
		t.Fatalf("nav: %v", err)
	}
	page.Timeout(4 * time.Second).WaitDOMStable(150*time.Millisecond, 0.1)

	if _, err := page.Eval(`() => { window.__k=[]; document.addEventListener('keydown', e => window.__k.push(e.key)); return 'ok' }`); err != nil {
		t.Fatalf("listener: %v", err)
	}
	if err := page.Keyboard.Press(input.Enter); err != nil {
		t.Fatalf("press: %v", err)
	}
	if err := page.Keyboard.Type(input.Key('a')); err != nil {
		t.Fatalf("type: %v", err)
	}
	time.Sleep(300 * time.Millisecond)
	res, err := page.Eval(`() => JSON.stringify(window.__k)`)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	got := res.Value.Str()
	if !strings.Contains(got, "Enter") || !strings.Contains(got, "a") {
		t.Fatalf("keydowns missing: %s", got)
	}
}