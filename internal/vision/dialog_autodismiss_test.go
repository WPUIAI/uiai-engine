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

// #301 dialog slice: an unanswered JavaScript dialog blocks the renderer
// main thread. The session listener must auto-dismiss the dialog so the page
// stays usable, record it in diagnostics (javascript_dialogs) with its
// message/default prompt, and leave the dismiss result visible in the DOM
// for the agent's next read.
func TestJavaScriptDialogsAutoDismissAndReport(t *testing.T) {
	if os.Getenv("UIAI_INPUT_TESTS") == "" {
		t.Skip("set UIAI_INPUT_TESTS=1 for browser dialog regression")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `<!doctype html><html><body>
		<button id="alert-btn" onclick="alert('engine alert text')">fire alert</button>
		<button id="confirm-btn" onclick="document.getElementById('result').textContent = 'confirm=' + confirm('engine confirm text')">fire confirm</button>
		<div id="result">none</div>
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

	// Fire an alert: the renderer must survive the dialog (auto-dismiss)
	// and the dialog must be recorded.
	if _, err := session.page.Eval(`() => { document.getElementById('alert-btn').click(); return 'clicked' }`); err != nil {
		t.Fatalf("click alert: %v", err)
	}
	deadline := time.Now().Add(10 * time.Second)
	var diag DiagnosticsSnapshot
	okAlert := false
	for time.Now().Before(deadline) {
		diag = session.DiagnosticsWithOptions(DiagnosticsOptions{Limit: 20})
		for _, d := range diag.JavascriptDialogs {
			if d.Type == "alert" && strings.Contains(d.Message, "engine alert text") {
				okAlert = true
			}
		}
		if okAlert {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !okAlert {
		t.Fatalf("alert dialog not recorded in diagnostics: %+v", diag.JavascriptDialogs)
	}

	// The renderer must still respond after the dismissed alert.
	if _, err := session.page.Timeout(10 * time.Second).Eval("() => document.title + ':alive'"); err != nil {
		t.Fatalf("renderer wedged after alert: %v", err)
	}

	// Fire a confirm: auto-dismiss means confirm() returns false, and the
	// dismiss result must be visible in the DOM for the next read.
	if _, err := session.page.Eval(`() => { document.getElementById('confirm-btn').click(); return 'clicked' }`); err != nil {
		t.Fatalf("click confirm: %v", err)
	}
	deadline = time.Now().Add(10 * time.Second)
	var result string
	for time.Now().Before(deadline) {
		res, rerr := session.page.Eval(`() => document.getElementById('result').textContent`)
		if rerr == nil {
			result = res.Value.Str()
			if strings.Contains(result, "confirm=false") {
				break
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	if !strings.Contains(result, "confirm=false") {
		t.Fatalf("confirm dismiss result not visible in DOM: %q", result)
	}

	// Both dialogs recorded.
	diag = session.DiagnosticsWithOptions(DiagnosticsOptions{Limit: 20})
	seenAlert, seenConfirm := false, false
	for _, d := range diag.JavascriptDialogs {
		if d.Type == "alert" && strings.Contains(d.Message, "engine alert text") {
			seenAlert = true
		}
		if d.Type == "confirm" && strings.Contains(d.Message, "engine confirm text") {
			seenConfirm = true
		}
	}
	if !seenAlert || !seenConfirm {
		t.Fatalf("diagnostics must record both dialogs, alert=%v confirm=%v: %+v", seenAlert, seenConfirm, diag.JavascriptDialogs)
	}
}
