package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestDeadlinePreservesFastStatusHeadersAndBody(t *testing.T) {
	handler := WithDeadline(time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Add("Set-Cookie", "a=synthetic")
		w.Header().Add("Set-Cookie", "b=synthetic")
		w.WriteHeader(http.StatusAccepted)
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("first"))
		_, _ = w.Write([]byte("second"))
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusAccepted || recorder.Body.String() != "firstsecond" || recorder.Header().Get("Content-Type") != "text/plain" || len(recorder.Header().Values("Set-Cookie")) != 2 {
		t.Fatalf("fast response changed: %d %v %s", recorder.Code, recorder.Header(), recorder.Body.String())
	}
}

func TestDeadlinePropagatesHandlerPanic(t *testing.T) {
	defer func() {
		if recover() != "synthetic-panic" {
			t.Fatal("handler panic was swallowed")
		}
	}()
	WithDeadline(time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { panic("synthetic-panic") })).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestDeadlinePreservesCostInjection(t *testing.T) {
	handler := CostMiddleware(WithDeadline(time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		CostTouchPage(r, 2)
		writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
	})))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	if recorder.Code != http.StatusCreated || !strings.Contains(recorder.Body.String(), `"pages_touched":2`) || recorder.Result().Trailer.Get("X-UIAI-Cost-Pages") != "2" {
		t.Fatalf("cost instrumentation changed: %s %v", recorder.Body.String(), recorder.Result().Trailer)
	}
}

func TestDeadlineLateCostMutationDoesNotRaceTrailers(t *testing.T) {
	release := make(chan struct{})
	done := make(chan struct{})
	handler := CostMiddleware(WithDeadline(5 * time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-release
		CostTouchPage(r, 1)
		_, _ = w.Write([]byte("late"))
		close(done)
	})))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))
	close(release)
	<-done
	if recorder.Code != http.StatusServiceUnavailable || strings.Contains(recorder.Body.String(), "late") {
		t.Fatal("late response escaped buffer")
	}
}
