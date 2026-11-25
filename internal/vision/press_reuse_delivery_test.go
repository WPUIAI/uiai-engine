package vision

import (
	"os"
	"strings"
	"testing"
	"time"
)

// #228: Input.dispatchKeyEvent was silently dropped by pages served by the
// engine's long-lived browser while identical fresh-pool runs passed. The
// engine-level regression must cover the reuse shape: cycle a session
// through the pool (open → navigate → close), then verify keydown delivery
// on the SECOND session served by the same long-lived browser.
func TestPressDeliveredOnReusedPoolBrowser(t *testing.T) {
	if os.Getenv("UIAI_INPUT_TESTS") == "" {
		t.Skip("input-delivery test requires a reliable local browser; set UIAI_INPUT_TESTS=1")
	}
	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 2, AllowPrivateURLs: true})
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()
	sm := NewSessionManager(pool)
	defer sm.CloseAll()

	// First session: open, move around, capture, close — leaving the pool
	// browser warm with prior history, the #228 failing shape.
	first, _, err := sm.Open("https://example.com", 800, 600)
	if err != nil {
		t.Skipf("browser unavailable, skipping input-delivery test: %v", err)
	}
	if _, err := first.Navigate("https://example.com/"); err != nil {
		t.Fatalf("navigate in first session: %v", err)
	}
	if err := sm.Close(first.ID); err != nil {
		t.Fatalf("close first session: %v", err)
	}

	// Second session on the same long-lived pool browser.
	second, _, err := sm.Open("https://example.com", 800, 600)
	if err != nil {
		t.Fatalf("open second session on reused browser: %v", err)
	}
	defer sm.Close(second.ID)

	if _, err := second.page.Eval(`() => { window.__k=[]; document.addEventListener('keydown', e => window.__k.push(e.key)); return 'ok' }`); err != nil {
		t.Fatalf("listener: %v", err)
	}
	if _, err := second.Press("a"); err != nil {
		t.Fatalf("press a on reused browser: %v", err)
	}
	if _, err := second.Press("Enter"); err != nil {
		t.Fatalf("press Enter on reused browser: %v", err)
	}
	deadline := time.Now().Add(10 * time.Second)
	var got string
	for time.Now().Before(deadline) {
		res, rerr := second.page.Eval(`() => JSON.stringify(window.__k)`)
		if rerr == nil {
			got = res.Value.Str()
			if strings.Contains(got, "a") && strings.Contains(got, "Enter") {
				break
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !strings.Contains(got, "a") || !strings.Contains(got, "Enter") {
		t.Fatalf("keydowns missing on reused pool browser: %s", got)
	}
}
