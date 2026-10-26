package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/vision"
)

func TestSourceMarkdownWaitsForInitiallyEmptyJSBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><main id="content"></main><script>setTimeout(()=>{document.querySelector('#content').innerHTML='<h1>Delayed docs ready</h1><p>Useful content.</p><dl><dt>API key</dt><dd><code>SYNTHETIC_DELAYED_KEY_0123456789</code></dd></dl>'},1200)</script></body></html>`))
	}))
	defer site.Close()
	pool, err := vision.NewPoolWithConfig(vision.PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	sm := vision.NewSessionManager(pool)
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	payload, _ := json.Marshal(markdownRequest{URL: site.URL, MaxChars: 2000})
	req := httptest.NewRequest("POST", "https://evidence.example/api/markdown", bytes.NewReader(payload))
	out := httptest.NewRecorder()
	handleMarkdownRequest(out, req, markdownRequest{URL: site.URL, MaxChars: 2000}, cfg, sm)
	var body map[string]any
	if err := json.Unmarshal(out.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if out.Code != 200 && out.Code != 202 {
		t.Fatalf("readiness conversion failed: %d", out.Code)
	}
	if !strings.Contains(body["markdown"].(string), "Delayed docs ready") {
		t.Fatal("one-shot conversion returned empty before JS docs rendered")
	}
	if strings.Contains(out.Body.String(), "SYNTHETIC_DELAYED_KEY_0123456789") {
		t.Fatal("delayed content bypassed canonical credential redaction")
	}
	if len(sm.List()) != 0 || body["cleanup"].(map[string]any)["closed"] != true {
		t.Fatal("owned readiness session leaked")
	}
}

func TestSourceMarkdownEmptyBudgetAndCancellationBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1")
	}
	calls := 0
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><main></main></body></html>`))
	}))
	defer site.Close()
	pool, err := vision.NewPoolWithConfig(vision.PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	sm := vision.NewSessionManager(pool)
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	out := httptest.NewRecorder()
	started := time.Now()
	handleMarkdownRequest(out, httptest.NewRequest("POST", "https://evidence.example/api/markdown", nil), markdownRequest{URL: site.URL}, cfg, sm)
	var body map[string]any
	if err := json.Unmarshal(out.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if out.Code != 422 || body["error"] != "source_content_unavailable" || body["suggested_next_action"] == nil || body["artifact_ref"] != nil || len(sm.List()) != 0 || body["cleanup"].(map[string]any)["closed"] != true {
		t.Fatal("empty document became snapshot success or leaked owned session")
	}
	if time.Since(started) > 10*time.Second {
		t.Fatal("empty-page readiness was unbounded")
	}
	// A caller already cancelled before entry must not navigate/open any page.
	before := calls
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out = httptest.NewRecorder()
	req := httptest.NewRequest("POST", "https://evidence.example/api/markdown", nil).WithContext(ctx)
	handleMarkdownRequest(out, req, markdownRequest{URL: site.URL}, cfg, sm)
	if out.Code != 408 || calls != before || len(sm.List()) != 0 {
		t.Fatal("pre-cancelled source request opened a page")
	}
	sess, _, err := sm.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatal(err)
	}
	defer sm.Close(sess.ID)
	caller, stop := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer stop()
	started = time.Now()
	if _, err := readSourcePageReady(caller, sess, vision.ReadOptions{Format: "markdown"}); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("caller deadline lost during empty readiness")
	}
	if time.Since(started) > time.Second {
		t.Fatal("readiness ignored caller cancellation")
	}
	if _, err := sess.ReadPage(vision.ReadOptions{}); err != nil {
		t.Fatal("cancelled read poisoned shared session/page lifetime")
	}
}
