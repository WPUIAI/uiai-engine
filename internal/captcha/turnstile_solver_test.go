package captcha

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/vision"
)

// #70: the Turnstile solve path waits bounded for the challenge token after
// one widget click. A widget that completes reports solved with the token; a
// page that never completes reports unsolved honestly.
func TestSolveTurnstileBoundedWait(t *testing.T) {
	if os.Getenv("UIAI_INPUT_TESTS") == "" {
		t.Skip("set UIAI_INPUT_TESTS=1 for browser turnstile regression")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// A fake turnstile widget: the response token appears after 1.5s,
		// simulating a managed challenge that accepts the environment.
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html><html><body>
		<div class="cf-turnstile" data-sitekey="fake"></div>
		<input name="cf-turnstile-response" value="">
		<script>setTimeout(() => { document.querySelector('input[name="cf-turnstile-response"]').value = 'tok_' + Math.random().toString(36).slice(2); }, 1500);</script>
		</body></html>`))
	}))
	defer site.Close()

	pool, err := vision.NewPoolWithConfig(vision.PoolConfig{MaxPages: 3, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	sm := vision.NewSessionManager(pool)
	defer sm.CloseAll()

	session, _, err := sm.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	solver := NewSolver(nil, CaptchaConfig{})
	resp := solver.SolveTurnstile(context.Background(), session, SolveConfig{TurnstileWaitSeconds: 8})
	if !resp.Solved || resp.Type != "turnstile" || resp.Token != "" {
		t.Fatalf("completing turnstile not solved, or it echoed a token capability: %+v", resp)
	}
	if resp.DurationMs < 1200 {
		t.Fatalf("solved without actually waiting for the token: %dms", resp.DurationMs)
	}

	// A widget that never completes reports unsolved with the observed state.
	stuck := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html><html><body>
		<div class="cf-turnstile" data-sitekey="fake"></div>
		<input name="cf-turnstile-response" value="">
		</body></html>`))
	}))
	defer stuck.Close()
	session2, _, err := sm.Open(stuck.URL, 640, 480)
	if err != nil {
		t.Fatalf("open stuck: %v", err)
	}
	resp2 := solver.SolveTurnstile(context.Background(), session2, SolveConfig{TurnstileWaitSeconds: 3})
	if resp2.Solved || resp2.Error == "" {
		t.Fatalf("never-completing turnstile must report unsolved honestly: %+v", resp2)
	}

	// No turnstile on the page: honest refusal, not a fabricated attempt.
	bare := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html><html><body><p>plain page</p></body></html>`))
	}))
	defer bare.Close()
	session3, _, err := sm.Open(bare.URL, 640, 480)
	if err != nil {
		t.Fatalf("open bare: %v", err)
	}
	resp3 := solver.SolveTurnstile(context.Background(), session3, SolveConfig{})
	if resp3.Solved || !strings.Contains(resp3.Error, "no turnstile challenge") {
		t.Fatalf("bare page must be refused: %+v", resp3)
	}

	// The dispatch routes the turnstile type to the new solver.
	resp4 := solver.SolveInSession(context.Background(), session, SolveRequest{Type: "turnstile", Config: SolveConfig{TurnstileWaitSeconds: 8}})
	if !resp4.Solved || resp4.Type != "turnstile" {
		t.Fatalf("dispatch did not route turnstile: %+v", resp4)
	}
}
