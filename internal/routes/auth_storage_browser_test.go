package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func TestAuthLoadInvalidStorageHTTPBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for browser-backed HTTP regression")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html><body>auth fixture</body></html>`))
	}))
	defer site.Close()
	pool, err := vision.NewPoolWithConfig(vision.PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := vision.NewSessionManager(pool)
	sess, _, err := manager.Open(site.URL, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(sess.ID)
	router := chi.NewRouter()
	router.Route("/api/session", func(r chi.Router) { MountSessionRoutes(r, &config.Config{}, manager) })
	for _, field := range []string{"localStorage", "sessionStorage"} {
		request := httptest.NewRequest(http.MethodPost, "/api/session/"+sess.ID+"/auth/load", strings.NewReader(`{"`+field+`":{"key":42}}`))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest || !strings.Contains(response.Body.String(), field) || strings.Contains(response.Body.String(), `"loaded"`) {
			t.Fatalf("%s: status=%d body=%s", field, response.Code, response.Body.String())
		}
	}
}
