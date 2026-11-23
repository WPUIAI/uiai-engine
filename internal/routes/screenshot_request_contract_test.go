package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

// #311/#312: capture routes must keep a strict, documented wire contract —
// unknown request fields are rejected with a machine-readable reason_code
// instead of being silently ignored while the capture still succeeds.

func TestCompareScreenshotRejectsUnknownFields(t *testing.T) {
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	router := chi.NewRouter()
	MountScreenshotCompare(router, cfg, nil, nil, nil, nil, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/compare", strings.NewReader(`{"url1":"https://example.com","viewport":{"width":390,"height":844}}`)))
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unknown compare fields accepted: status=%d body=%s", response.Code, response.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["reason_code"] != "unknown_field" {
		t.Fatalf("missing unknown_field reason: %s", response.Body.String())
	}
}

func TestSessionScreenshotRejectsUnknownFields(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><body><h1>capture fixture</h1></body></html>`))
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
		t.Fatal("session fixture unavailable")
	}
	router := chi.NewRouter()
	cfg := &config.Config{}
	isolateEvidencePublicationStorage(t, cfg)
	MountSessionRoutes(router, cfg, manager)

	unknown := httptest.NewRecorder()
	router.ServeHTTP(unknown, httptest.NewRequest(http.MethodPost, "/"+session.ID+"/screenshot", strings.NewReader(`{"device":"mobile"}`)))
	if unknown.Code != http.StatusBadRequest {
		t.Fatalf("unknown session screenshot fields accepted: status=%d body=%s", unknown.Code, unknown.Body.String())
	}
	var rejection map[string]string
	if err := json.Unmarshal(unknown.Body.Bytes(), &rejection); err != nil {
		t.Fatal(err)
	}
	if rejection["reason_code"] != "unknown_field" {
		t.Fatalf("missing unknown_field reason: %s", unknown.Body.String())
	}

	// An empty body keeps the default-capture meaning and must not 400.
	empty := httptest.NewRecorder()
	router.ServeHTTP(empty, httptest.NewRequest(http.MethodPost, "/"+session.ID+"/screenshot", strings.NewReader("")))
	if empty.Code == http.StatusBadRequest {
		t.Fatalf("empty body rejected: status=%d body=%s", empty.Code, empty.Body.String())
	}

	// Known fields are accepted; the capture itself may still fail on
	// browser availability, but a strict decode must never reject them.
	known := httptest.NewRecorder()
	router.ServeHTTP(known, httptest.NewRequest(http.MethodPost, "/"+session.ID+"/screenshot", strings.NewReader(`{"format":"png","fullPage":false}`)))
	if known.Code == http.StatusBadRequest {
		t.Fatalf("known fields rejected: status=%d body=%s", known.Code, known.Body.String())
	}
}
