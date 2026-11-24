package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

// #300: the session read surface must accept a bounded caller deadline on both
// transports (POST body and GET query) instead of silently ignoring it.
func TestSessionReadAcceptsCallerDeadline(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><body><h1>read deadline fixture</h1><p>bounded text</p></body></html>`))
	}))
	defer site.Close()
	pool, err := vision.NewPoolWithConfig(vision.PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := vision.NewSessionManager(pool)
	defer manager.CloseAll()
	session, _, err := manager.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatal("read fixture unavailable")
	}
	router := chi.NewRouter()
	cfg := &config.Config{}
	isolateEvidencePublicationStorage(t, cfg)
	MountSessionRoutes(router, cfg, manager)

	postRead := httptest.NewRecorder()
	router.ServeHTTP(postRead, httptest.NewRequest(http.MethodPost, "/"+session.ID+"/read", strings.NewReader(`{"max_chars":200,"timeout_ms":15000}`)))
	if postRead.Code == http.StatusBadRequest {
		t.Fatalf("POST timeout_ms rejected: %d %s", postRead.Code, postRead.Body.String())
	}

	getRead := httptest.NewRecorder()
	router.ServeHTTP(getRead, httptest.NewRequest(http.MethodGet, "/"+session.ID+"/read?timeout_ms=15000&max_chars=200", nil))
	if getRead.Code == http.StatusBadRequest {
		t.Fatalf("GET timeout_ms rejected: %d %s", getRead.Code, getRead.Body.String())
	}
}
