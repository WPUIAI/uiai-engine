package vision

import (
	"encoding/json"
	"github.com/WPUIAI/uiai-engine/internal/secrettext"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestSemanticSecretNeighborsBrowserOutputs(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1")
	}
	const canary = "UIAI_SYNTHETIC_PAGE_TOKEN_0123456789"
	const callbackCanary = "UIAI_SYNTHETIC_CALLBACK_PROJECTION_0123456789"
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>` + canary + `</title></head><body><main><p>ordinary documentation remains useful</p><dl><dt>API key</dt><dd><code>` + canary + `</code></dd></dl><input id="api-key" name="api_key" readonly value="` + canary + `"></main></body></html>`))
	}))
	defer site.Close()
	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := NewSessionManager(pool)
	sess, _, err := manager.Open(site.URL, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(sess.ID)
	check := func(surface string, value any) {
		t.Helper()
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), canary) || strings.Contains(string(encoded), callbackCanary) {
			var decoded any
			_ = json.Unmarshal(encoded, &decoded)
			var paths []string
			var walk func(any, string)
			walk = func(item any, path string) {
				switch item := item.(type) {
				case string:
					if strings.Contains(item, canary) || strings.Contains(item, callbackCanary) {
						paths = append(paths, path)
					}
				case map[string]any:
					for key, child := range item {
						walk(child, path+"."+key)
					}
				case []any:
					for _, child := range item {
						walk(child, path+"[]")
					}
				}
			}
			walk(decoded, surface)
			t.Fatalf("synthetic page token leaked at %s", strings.Join(paths, ","))
		}
	}
	check("open title", map[string]any{"title": sess.Title})
	text, err := sess.TextContent("dd code")
	if err != nil {
		t.Fatal(err)
	}
	check("element text", text)
	text, err = sess.TextIntent("dd code")
	if err != nil {
		t.Fatal(err)
	}
	check("intent text", text)
	for _, format := range []string{"text", "markdown"} {
		result, err := sess.ReadPage(ReadOptions{Format: format, Mode: "full", MaxChars: 8000})
		if err != nil {
			t.Fatal(err)
		}
		check(format, result)
		if !strings.Contains(result.Text, "ordinary documentation remains useful") {
			t.Fatal("ordinary reading broken")
		}
		report, ok := result.Metadata["redaction"].(secrettext.Summary)
		if !ok || report.Count < 1 {
			t.Fatal("missing value-free redaction summary")
		}
	}
	snapshot, err := sess.Snapshot(SnapshotOptions{Compact: false})
	if err != nil {
		t.Fatal(err)
	}
	check("snapshot", snapshot)
	dom, err := sess.DOMInfo()
	if err != nil {
		t.Fatal(err)
	}
	check("dom", dom)
	_, _, err = sess.Eval(`console.error("API key", "` + canary + `"); true`)
	if err != nil {
		t.Fatal(err)
	}
	value, _, err := sess.Eval(`return document.querySelector('#api-key').value`)
	if err != nil {
		t.Fatal(err)
	}
	check("evaluation output", value)
	value, _, err = sess.Eval(`return {api_key: document.querySelector('#api-key').value}`)
	if err != nil {
		t.Fatal(err)
	}
	check("structured evaluation output", value)

	callback, _, err := sess.Eval(`history.replaceState({}, '', '?code=` + callbackCanary + `&ok=1'); return location.href`)
	if err != nil {
		t.Fatal(err)
	}
	check("callback evaluation", callback)
	callbackRead, err := sess.ReadPage(ReadOptions{MaxChars: 8000})
	if err != nil {
		t.Fatal(err)
	}
	check("callback read", callbackRead)
	callbackSnapshot, err := sess.Snapshot(SnapshotOptions{Compact: true})
	if err != nil {
		t.Fatal(err)
	}
	check("callback snapshot", callbackSnapshot)
	check("diagnostics", sess.Diagnostics(50, "all", false))
	func() {
		sess.diagnostics.mu.Lock()
		defer sess.diagnostics.mu.Unlock()
		check("stored console", sess.diagnostics.console)
	}()
}
