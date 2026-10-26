package routes

import (
	"context"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestStandaloneReadOnlySSEDoesNotAddCredentialHeader(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := httptest.NewRequest("GET", "/api/events", nil).WithContext(ctx)
	r.Header.Set("Origin", "https://foreign.example.invalid")
	w := httptest.NewRecorder()
	MountSSEStream(&config.Config{})(w, r)
	if w.Header().Get("Access-Control-Allow-Origin") != "*" || w.Header().Get("Access-Control-Allow-Credentials") != "" || w.Header().Get("Content-Type") != "text/event-stream" || !strings.Contains(w.Body.String(), "event: connected") {
		t.Fatal("read-only SSE CORS contract changed")
	}
}
