package server

import (
	"context"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/routes"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCORSAllowlistDoesNotAuthorizeForeignCookieOrigins(t *testing.T) {
	cfg := &config.Config{}
	cfg.CORS.Origins = []string{"https://approved.example.invalid"}
	cfg.CORS.Methods = []string{"GET", "POST"}
	cfg.CORS.Headers = []string{"Authorization", "Content-Type"}
	for _, origin := range []string{"https://approved.example.invalid", "https://foreign.example.invalid", "null"} {
		for _, method := range []string{"OPTIONS", "POST"} {
			called := false
			h := corsMiddleware(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { called = true; w.WriteHeader(204) }))
			r := httptest.NewRequest(method, "/api/fixture", nil)
			r.Header.Set("Origin", origin)
			r.Header.Set("Access-Control-Request-Headers", "Authorization")
			r.AddCookie(&http.Cookie{Name: "session", Value: "fixture"})
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			want := ""
			if origin == cfg.CORS.Origins[0] {
				want = origin
			}
			if w.Header().Get("Access-Control-Allow-Origin") != want || w.Header().Get("Access-Control-Allow-Headers") != "Authorization, Content-Type" || w.Header().Get("Access-Control-Allow-Methods") != "GET, POST" || called != (method != "OPTIONS") {
				t.Fatal("CORS allowlist/preflight contract changed")
			}
		}
	}
}

func TestSSECORSCompositionDoesNotReflectForeignOrigins(t *testing.T) {
	cfg := &config.Config{}
	cfg.CORS.Origins = []string{"https://approved.example.invalid"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := httptest.NewRequest("GET", "/api/events", nil).WithContext(ctx)
	r.Header.Set("Origin", "https://foreign.example.invalid")
	w := httptest.NewRecorder()
	corsMiddleware(cfg)(routes.MountSSEStream(cfg)).ServeHTTP(w, r)
	if w.Header().Get("Access-Control-Allow-Origin") != "*" || w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("legacy composed SSE CORS posture changed")
	}
	// Wildcard plus credentials is NOT browser permission for credentialed cross-origin reads; auth remains independently header-based.
}
