package vision

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// #301 popups/tabs slice: a page that opens a new tab (window.open /
// target=_blank) must leave the session usable — the agent can list the
// popup, switch to it, read its content, switch back to the original page,
// and close the popup. Foreign targets are never attachable.
func TestPopupListSwitchClose(t *testing.T) {
	if os.Getenv("UIAI_INPUT_TESTS") == "" {
		t.Skip("set UIAI_INPUT_TESTS=1 for browser popup regression")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/popup-page" {
			fmt.Fprintf(w, `<!doctype html><html><head><title>Popup Content</title></head><body><h1>popup body text</h1></body></html>`)
			return
		}
		fmt.Fprintf(w, `<!doctype html><html><body>
		<button id="open-btn" onclick="window.open('/popup-page', '_blank')">open tab</button>
		<div id="state">original</div>
		</body></html>`)
	}))
	defer site.Close()

	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	sm := NewSessionManager(pool)
	defer sm.CloseAll()

	session, _, err := sm.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sm.Close(session.ID)

	original := session.OriginalTargetID()
	if original == "" {
		t.Fatal("original target id must be recorded")
	}

	// Open the popup through the page's own button.
	if _, err := session.page.Eval(`() => { document.getElementById('open-btn').click(); return 'clicked' }`); err != nil {
		t.Fatalf("click open: %v", err)
	}
	deadline := time.Now().Add(10 * time.Second)
	var popups []PopupTarget
	for time.Now().Before(deadline) {
		popups = session.Popups()
		if len(popups) > 0 {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if len(popups) == 0 {
		t.Fatal("popup target not tracked")
	}
	target := popups[0].TargetID
	if target == original {
		t.Fatalf("popup target must differ from the original page target")
	}

	// Switch to the popup and read its content.
	snap, err := session.SwitchToPopup(target)
	if err != nil {
		t.Fatalf("switch to popup: %v", err)
	}
	if snap == nil || snap.Size == 0 {
		t.Fatal("switch must return a capture of the popup")
	}
	res, err := session.page.Timeout(10 * time.Second).Eval(`() => document.body.textContent`)
	if err != nil {
		t.Fatalf("read popup content: %v", err)
	}
	if !strings.Contains(res.Value.Str(), "popup body text") {
		t.Fatalf("switched page is not the popup: %q", res.Value.Str())
	}

	// Switch back to the original page.
	if _, err := session.SwitchToPopup(original); err != nil {
		t.Fatalf("switch back: %v", err)
	}
	res, err = session.page.Timeout(10 * time.Second).Eval(`() => document.getElementById('state').textContent`)
	if err != nil {
		t.Fatalf("read original after switch back: %v", err)
	}
	if !strings.Contains(res.Value.Str(), "original") {
		t.Fatalf("switch back did not restore the original page: %q", res.Value.Str())
	}

	// Close the popup; the list shrinks and the original stays usable.
	if err := session.ClosePopup(target); err != nil {
		t.Fatalf("close popup: %v", err)
	}
	if got := session.Popups(); len(got) != 0 {
		t.Fatalf("closed popup must leave the ring: %+v", got)
	}
	if err := session.ClosePopup(original); err == nil {
		t.Fatal("closing the original target through the popup API must be refused")
	}
}
