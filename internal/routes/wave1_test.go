package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// C-010-04: a handler wedged past its deadline must produce the structured
// envelope, never a client-side hang.
func TestDeadlineEnvelopeOnStalledHandler(t *testing.T) {
	h := WithDeadline(30 * time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	w := httptest.NewRecorder()
	start := time.Now()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/x", nil))
	if elapsed := time.Since(start); elapsed > 120*time.Millisecond {
		t.Fatalf("deadline did not bound execution: %v", elapsed)
	}
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("want 503 got %d", w.Code)
	}
	for _, want := range []string{`"error":"deadline_exceeded"`, `"retry":true`} {
		if !strings.Contains(w.Body.String(), want) {
			t.Fatalf("envelope missing %s: %s", want, w.Body.String())
		}
	}
}

// C-010-04: handlers finishing inside the deadline are untouched.
func TestDeadlinePassthroughWhenFast(t *testing.T) {
	h := WithDeadline(time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/x", nil))
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"ok":true`) {
		t.Fatalf("fast path altered: %d %s", w.Code, w.Body.String())
	}
}

// C-010-05: envelopes carry cost object; headers carry duration/bytes/pages.
func TestCostStamping(t *testing.T) {
	handler := CostMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		CostTouchPage(r, 2)
		writeJSON(w, 200, map[string]any{"ok": true})
	}))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/y", nil))
	body := w.Body.String()
	if !strings.Contains(body, `"cost":`) || !strings.Contains(body, `"duration_ms":`) ||
		!strings.Contains(body, `"pages_touched":2`) {
		t.Fatalf("cost object missing/wrong: %s", body)
	}
	tr := w.Result().Trailer
	if tr == nil || tr.Get("X-UIAI-Cost-Pages") != "2" || tr.Get("X-UIAI-Cost-Ms") == "" {
		t.Fatalf("cost trailers missing: %v", tr)
	}
}

// C-010-05: non-map payloads and non-instrumented writers are safe no-ops.
func TestCostInjectionSafeNoop(t *testing.T) {
	type payload struct{ OK bool }
	w := httptest.NewRecorder()
	writeJSON(w, 200, payload{OK: true}) // struct → no injection, must not panic
	if !strings.Contains(w.Body.String(), `"OK":true`) {
		t.Fatalf("struct passthrough broken: %s", w.Body.String())
	}
	plain := httptest.NewRecorder()
	got := InjectCost(map[string]any{"a": 1}, plain) // unwrapped writer → noop
	if _, ok := got.(map[string]any)["cost"]; ok {
		t.Fatal("injection must skip non-instrumented writer")
	}
}
