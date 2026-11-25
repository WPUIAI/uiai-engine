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

// #293: rod's WaitDOMStable polls DOMSnapshot.captureSnapshot and wedges the
// renderer main thread on continuously-mutating pages (docs.openclaw.ai
// reproduced 3/3 standalone: evaluate hangs forever, capture fails with
// {-32603}). The engine's post-navigation settle must be poll-free, so a
// page that keeps mutating after load must still navigate, capture, and
// evaluate within budget.
func TestNavigateSettlesContinuouslyMutatingPage(t *testing.T) {
	if os.Getenv("UIAI_INPUT_TESTS") == "" {
		t.Skip("set UIAI_INPUT_TESTS=1 for browser settle regression")
	}

	// Fixture: heavy node count plus a post-load mutation loop that never
	// stabilizes — the docs-page wedge mechanics without the public target.
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		rows := strings.Repeat(`<div class="row"><span>docs</span><a href="#">link</a></div>`, 800)
		fmt.Fprintf(w, `<!doctype html><html><body><h1>mutating fixture</h1>%s
		<script>
			let n = 0;
			setInterval(() => {
				const d = document.createElement('div');
				d.className = 'ripple-' + (n++);
				d.textContent = 'mutation ' + n;
				document.body.appendChild(d);
			}, 40);
		</script></body></html>`, rows)
	}))
	defer site.Close()

	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := NewSessionManager(pool)
	defer manager.CloseAll()

	session, _, err := manager.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatalf("open on mutating fixture must succeed (old wedge fired at open): %v", err)
	}

	// Second navigation with the mutation loop already running: the exact
	// shape that wedged docs.openclaw.ai via WaitDOMStable polling.
	snap, err := session.Navigate(site.URL + "?page=2")
	if err != nil {
		t.Fatalf("navigate on mutating fixture must succeed without wedging: %v", err)
	}
	if snap == nil || snap.Size == 0 {
		t.Fatal("navigate must still produce a capture on a mutating page")
	}
	if !strings.Contains(session.URL, "page=2") {
		t.Fatalf("navigation must complete to the requested URL, got %s", session.URL)
	}

	// The renderer must stay responsive after the settle.
	done := make(chan error, 1)
	go func() {
		_, eerr := session.page.Timeout(10 * time.Second).Eval("() => document.readyState")
		done <- eerr
	}()
	select {
	case eerr := <-done:
		if eerr != nil {
			t.Fatalf("renderer must evaluate after navigate settle: %v", eerr)
		}
	case <-time.After(12 * time.Second):
		t.Fatal("renderer wedged after navigate settle — WaitDOMStable regression returned")
	}
}
