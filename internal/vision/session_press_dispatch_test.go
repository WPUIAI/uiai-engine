package vision

import (
	"strings"
	"testing"
)

// TestSessionPressDispatchesKeydown exercises the full engine session path
// (SessionManager.Open → Session.Press) — the same path the HTTP route uses.
func TestSessionPressDispatchesKeydown(t *testing.T) {
	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 2, AllowPrivateURLs: true})
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	defer pool.Close()
	sm := NewSessionManager(pool)

	sess, _, err := sm.Open("https://example.com", 1280, 800)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sm.Close(sess.ID)

	if _, err := sess.page.Eval(`() => { window.__k=[]; document.addEventListener('keydown', e => window.__k.push(e.key)); return 'ok' }`); err != nil {
		t.Fatalf("listener: %v", err)
	}
	if _, err := sess.Press("a"); err != nil {
		t.Fatalf("press a: %v", err)
	}
	res, err := sess.page.Eval(`() => JSON.stringify(window.__k)`)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	got := res.Value.Str()
	if !strings.Contains(got, "a") {
		t.Fatalf("keydowns missing after Session.Press: %s", got)
	}
}