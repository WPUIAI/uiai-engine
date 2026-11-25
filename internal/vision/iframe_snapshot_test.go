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

// #301 iframes slice: the snapshot descends into same-origin iframe
// documents with frame-qualified selectors, and click/fill resolve through
// those selectors so the agent can act on iframe content. Cross-origin
// frames stay opaque without breaking the walk.
func TestIframeSnapshotAndActions(t *testing.T) {
	if os.Getenv("UIAI_INPUT_TESTS") == "" {
		t.Skip("set UIAI_INPUT_TESTS=1 for browser iframe regression")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/frame", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `<!doctype html><html><head><title>Inner</title></head><body>
		<button id="inner-btn">inner action</button>
		<input id="inner-input" placeholder="inner field">
		<div id="inner-state">idle</div>
		<script>document.getElementById('inner-btn').onclick = () => { document.getElementById('inner-state').textContent = 'clicked-inside-frame'; };</script>
		</body></html>`)
	})
	mux.HandleFunc("/page", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `<!doctype html><html><body>
		<h1>host page</h1>
		<iframe id="host-frame" src="/frame" title="same origin frame"></iframe>
		<iframe id="opaque-frame" src="https://example.com/" title="cross origin frame"></iframe>
		</body></html>`)
	})
	site := httptest.NewServer(mux)
	defer site.Close()

	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	sm := NewSessionManager(pool)
	defer sm.CloseAll()

	session, _, err := sm.Open(site.URL+"/page", 640, 480)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer sm.Close(session.ID)

	deadline := time.Now().Add(10 * time.Second)
	settled := false
	for time.Now().Before(deadline) {
		res, rerr := session.page.Eval(`() => { const f = document.getElementById('host-frame'); return !!(f && f.contentDocument && f.contentDocument.getElementById('inner-btn')); }`)
		if rerr == nil && strings.Contains(res.Value.Str(), "true") {
			settled = true
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !settled {
		t.Fatal("same-origin iframe never settled with its inner button")
	}

	snap, err := session.Snapshot(SnapshotOptions{})
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	text := snap.Tree
	if !strings.Contains(text, "iframe") {
		t.Fatalf("snapshot must surface the iframe nodes: %.300s", text)
	}

	// Frame-qualified selectors must resolve through the shared action path.
	res, err := session.page.Eval(`(sel) => { let doc = document; while (sel.startsWith('frame:')) { const b = sel.indexOf('::'); const idx = parseInt(sel.substring(6, b), 10); const ifr = doc.querySelectorAll('iframe')[idx]; if (!ifr) return 'no-frame'; try { doc = ifr.contentDocument; } catch (e) { return 'cross-origin'; } if (!doc) return 'no-doc'; sel = sel.substring(b + 2); } const el = doc.querySelector(sel); return el ? 'ok:' + el.tagName : 'no-element'; }`, "frame:0::#inner-btn")
	if err != nil {
		t.Fatalf("frame probe: %v", err)
	}
	if !strings.Contains(res.Value.Str(), "ok:BUTTON") {
		t.Fatalf("frame:0::#inner-btn did not resolve: %s", res.Value.Str())
	}

	// Click through the frame-qualified selector path.
	clickSnap, err := session.Click("frame:0::#inner-btn")
	if err != nil {
		t.Fatalf("click inside iframe: %v", err)
	}
	if clickSnap == nil {
		t.Fatal("click must return a capture")
	}
	res2, rerr := session.page.Eval(`() => document.getElementById('host-frame').contentDocument.getElementById('inner-state').textContent`)
	if rerr != nil || !strings.Contains(res2.Value.Str(), "clicked-inside-frame") {
		t.Fatalf("iframe click had no visible effect: %v %v", rerr, res2.Value.Str())
	}

	// Fill inside the frame document.
	if _, err := session.Fill("frame:0::#inner-input", "typed-via-frame-ref"); err != nil {
		t.Fatalf("fill inside iframe: %v", err)
	}
	res3, ferr := session.page.Eval(`() => document.getElementById('host-frame').contentDocument.getElementById('inner-input').value`)
	if ferr != nil || !strings.Contains(res3.Value.Str(), "typed-via-frame-ref") {
		t.Fatalf("iframe fill not visible: %v %v", ferr, res3.Value.Str())
	}

	// Cross-origin frame stays opaque but must not break lookups.
	if _, err := session.Click("frame:1::#anything"); err == nil {
		t.Fatal("cross-origin frame action must fail closed, not succeed")
	}
}
