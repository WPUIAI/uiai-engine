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

// #80 acceptance: the actor set at open propagates to every audit surface —
// session info, error envelopes, and the fleet health grouping. Opaque
// strings: the engine never interprets them.
func TestActorPropagationAcrossAuditSurfaces(t *testing.T) {
	const actor = "agent:focusa:workstream-9"
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><body>ok</body></html>`))
	}))
	defer site.Close()
	pool, err := vision.NewPoolWithConfig(vision.PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := vision.NewSessionManager(pool)
	defer manager.CloseAll()
	router := chi.NewRouter()
	MountSessionRoutes(router, &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}, manager)

	opened := httptest.NewRecorder()
	router.ServeHTTP(opened, httptest.NewRequest("POST", "/",
		strings.NewReader(`{"url":"`+site.URL+`","width":640,"height":480,"actor_ref":"`+actor+`"}`)))
	if !strings.Contains(opened.Body.String(), `"actor_ref":"`+actor+`"`) {
		t.Fatalf("open response missing actor attribution: %.200s", opened.Body.String())
	}
	sid := sessionIDFromJSON(t, opened.Body.String())

	info := httptest.NewRecorder()
	router.ServeHTTP(info, httptest.NewRequest("GET", "/"+sid+"/", nil))
	if !strings.Contains(info.Body.String(), `"actor_ref":"`+actor+`"`) {
		t.Fatalf("session info missing actor attribution: %.200s", info.Body.String())
	}

	// Error envelope attribution: a failing op names the actor, not an
	// anonymous session.
	failed := httptest.NewRecorder()
	router.ServeHTTP(failed, httptest.NewRequest("POST", "/"+sid+"/fill",
		strings.NewReader(`{"selector":"#no-such-element","text":"x"}`)))
	if !strings.Contains(failed.Body.String(), actor) {
		t.Fatalf("error envelope missing actor attribution: %.300s", failed.Body.String())
	}

	// Fleet health groups by actor; anonymous sessions are named honestly.
	manager.List()[0].SetActorRef(actor)
	health := browserHealthPayload(pool, true, manager, &config.Config{})
	grouped, _ := health["sessions_by_actor"].(map[string]int)
	if grouped[actor] != 1 {
		t.Fatalf("health grouping missing the actor: %v", grouped)
	}
}

func sessionIDFromJSON(t *testing.T, raw string) string {
	t.Helper()
	start := strings.Index(raw, `"id":"`)
	if start < 0 {
		t.Fatalf("session id missing from response: %.200s", raw)
	}
	rest := raw[start+len(`"id":"`):]
	if end := strings.Index(rest, `"`); end > 0 {
		return rest[:end]
	}
	t.Fatalf("session id unparseable: %.200s", raw)
	return ""
}
