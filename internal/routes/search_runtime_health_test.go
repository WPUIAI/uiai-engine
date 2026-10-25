package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func TestSearchSharedRuntimeHealthHonorsApprovedFallback(t *testing.T) {
	cfg, brave, wiki := fallbackFixture(t)
	cfg.Search.FallbackProviders = nil
	state, clock := testSearchBreakers()
	router := chi.NewRouter()
	router.Route("/api/search", func(r chi.Router) { MountSearchRoutes(r, cfg, state) })
	router.Route("/api/health", func(r chi.Router) { MountBrowserHealth(r, &vision.Pool{}, true, nil, cfg, state) })
	router.Route("/api/metrics", func(r chi.Router) { MountBrowserHealth(r, &vision.Pool{}, true, nil, cfg, state) })
	for i := 0; i < 2; i++ {
		out := httptest.NewRecorder()
		router.ServeHTTP(out, httptest.NewRequest(http.MethodGet, "/api/search/?q=fixture&provider=brave", nil))
		if out.Code != 429 {
			t.Fatal("fixture primary did not return rate limit")
		}
	}
	for _, path := range []string{"/api/health/browser", "/api/metrics/browser"} {
		out := httptest.NewRecorder()
		router.ServeHTTP(out, httptest.NewRequest(http.MethodGet, path, nil))
		var body struct {
			AgentPressure struct {
				Search struct {
					Basis     string `json:"readiness_basis"`
					Ready     bool   `json:"default_request_ready"`
					Status    string `json:"provider_status"`
					Quota     string `json:"remaining_quota_status"`
					Providers []struct {
						ID      string              `json:"id"`
						Circuit searchCircuitStatus `json:"circuit"`
					} `json:"providers"`
				} `json:"search"`
			} `json:"agent_pressure"`
		}
		if err := json.NewDecoder(out.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		s := body.AgentPressure.Search
		if s.Basis != "observed_circuits" || s.Ready || s.Status != "degraded" || s.Quota != "unknown" {
			t.Fatalf("unexpected health basis=%s ready=%v status=%s quota=%s", s.Basis, s.Ready, s.Status, s.Quota)
		}
		found := false
		for _, p := range s.Providers {
			if p.ID == "brave" {
				found = true
				if p.Circuit.State != "open" {
					t.Fatal("health did not share actual search state")
				}
			}
		}
		if !found {
			t.Fatal("health omitted provider details")
		}
	}
	if brave.Load() != 2 || wiki.Load() != 0 {
		t.Fatal("health observation issued provider requests")
	}
	cfg.Search.FallbackProviders = []string{"wikipedia"}
	if s := searchPressureSummary(cfg, state); s["default_request_ready"] != true || s["pressure"] != "normal" {
		t.Fatal("approved healthy alternate not reflected")
	}
	clock.add(30 * time.Second)
	cfg.Search.FallbackProviders = nil
	if s := searchPressureSummary(cfg, state); s["default_request_ready"] != true {
		t.Fatal("eligible half-open probe not reflected")
	}
	a, wait := state.acquire("brave")
	if wait != 0 {
		t.Fatal("probe not eligible")
	}
	if searchPressureSummary(cfg, state)["default_request_ready"] != false {
		t.Fatal("in-flight single probe advertised unlimited availability")
	}
	state.complete("brave", a, nil)
	if searchPressureSummary(cfg, state)["default_request_ready"] != true {
		t.Fatal("recovered default not reflected")
	}
}
func TestSearchHealthLabelsLegacyConfigurationOnly(t *testing.T) {
	cfg, _, _ := fallbackFixture(t)
	s := searchPressureSummary(cfg)
	if s["readiness_basis"] != "configuration_only" || s["remaining_quota_status"] != "unknown" {
		t.Fatal("legacy configuration snapshot presented as observed runtime/credit authority")
	}
}
