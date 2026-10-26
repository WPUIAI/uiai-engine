package routes

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

func TestSearchSetupRejectsInvalidEndpointBeforeQuery(t *testing.T) {
	clearSearchSetupCredentials(t)
	cfg := &config.Config{Search: config.SearchConfig{DefaultProvider: "brave", Providers: map[string]config.SearchProviderConfig{"brave": {APIKey: "synthetic-key", APIURL: "http://outside.example/search"}}}}
	guide, _ := readSetup(t, setupRouter(cfg, NewSearchRuntime()), "/api/tools/agent-card")
	if setupAction(t, guide, "brave") != "repair_existing_endpoint" || guide["default_provider"] != "brave" {
		t.Fatal("invalid configured endpoint falsely advertised as ready")
	}
}

func TestSearchEndpointSetupAndExecutionUseSameLocalPolicy(t *testing.T) {
	clearSearchSetupCredentials(t)
	for _, id := range []string{"brave", "ddg", "exa", "tavily", "wikipedia"} {
		t.Run(id, func(t *testing.T) {
			host := map[string]string{"brave": "api.search.brave.com", "ddg": "html.duckduckgo.com", "exa": "api.exa.ai", "tavily": "api.tavily.com", "wikipedia": "en.wikipedia.org"}[id]
			for _, raw := range []string{"http://outside.example/search", "https://outside.example/search", "https://user:synthetic-endpoint-canary@" + host + "/search", "https://" + host + "/search?api_key=synthetic-endpoint-canary", "https://" + host + "/search#synthetic-endpoint-canary", "http://127.0.0.1:99999/search", "https://" + host + "/search?", "https://" + host + "/search#"} {
				cfg := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{id: {APIKey: "synthetic-key", APIURL: raw}}}}
				// Wikipedia's existing endpoint override is environment-only.
				if id == "wikipedia" {
					t.Setenv("UIAI_WIKIPEDIA_SEARCH_API_URL", raw)
				}
				if ready, reason := searchProviderConfigured(cfg, id); ready || reason != "invalid_endpoint" {
					t.Fatalf("endpoint not rejected by readiness for %s", id)
				}
				guide, body := readSetup(t, setupRouter(cfg, NewSearchRuntime()), "/api/tools/agent-card")
				if setupAction(t, guide, id) != "repair_existing_endpoint" || strings.Contains(body, "synthetic-endpoint-canary") || strings.Contains(body, "outside.example") {
					t.Fatal("setup did not provide value-free endpoint repair")
				}
				if id != "searxng" && strings.Contains(guide["provider_setup"].(map[string]any)[id].(map[string]any)["message"].(string), "SearxNG") {
					t.Fatal("wrong provider setup instructions")
				}
				// A canceled context also fences against a regression sending an external
				// request: invalid URLs must fail locally, not merely return cancellation.
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				if _, err := searchProviderMeta(id).search(ctx, cfg, "synthetic-never-sent", 1); err == nil || errors.Is(err, context.Canceled) || strings.Contains(err.Error(), "synthetic-endpoint-canary") {
					t.Fatal("adapter did not reject URL locally or exposed raw URL")
				}
			}
		})
	}
}

func TestSearchEndpointDiscoveryDoesNotContactConfiguredUpstream(t *testing.T) {
	clearSearchSetupCredentials(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) }))
	defer server.Close()
	for _, id := range []string{"brave", "ddg", "exa", "tavily", "wikipedia"} {
		t.Run(id, func(t *testing.T) {
			cfg := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{id: {APIKey: "synthetic-key", APIURL: server.URL + "/search"}}}}
			if id == "wikipedia" {
				t.Setenv("UIAI_WIKIPEDIA_SEARCH_API_URL", server.URL+"/search")
			}
			for _, path := range []string{"/api/tools/agent-card", "/api/search/providers"} {
				guide, _ := readSetup(t, setupRouter(cfg, NewSearchRuntime()), path)
				if setupAction(t, guide, id) != "use_selected_provider" {
					t.Fatalf("approved loopback endpoint not recognized for %s", id)
				}
			}
		})
	}
	if calls.Load() != 0 {
		t.Fatal("endpoint validation made discovery queries")
	}
}

func TestSearchEndpointPrecedenceAndWikipediaLegacyConfig(t *testing.T) {
	clearSearchSetupCredentials(t)
	t.Setenv("UIAI_BRAVE_SEARCH_API_URL", "https://outside.example/should-not-win")
	cfg := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{"brave": {APIKey: "synthetic-key", APIURL: "https://api.search.brave.com/res/v1/web/search"}}}}
	if ready, _ := searchProviderConfigured(cfg, "brave"); !ready {
		t.Fatal("Brave explicit URL lost precedence")
	}
	cfg.Search.Providers["brave"] = config.SearchProviderConfig{APIKey: "synthetic-key"}
	if ready, reason := searchProviderConfigured(cfg, "brave"); ready || reason != "invalid_endpoint" {
		t.Fatal("Brave environment override not validated")
	}
	t.Setenv("UIAI_WIKIPEDIA_SEARCH_API_URL", "")
	cfg.Search.Providers["wikipedia"] = config.SearchProviderConfig{APIURL: "https://outside.example/legacy-unused-field"}
	if ready, _ := searchProviderConfigured(cfg, "wikipedia"); !ready {
		t.Fatal("Wikipedia legacy environment-only override changed")
	}
}

func TestSearchBadDefaultDoesNotDiscloseToUnapprovedProvider(t *testing.T) {
	clearSearchSetupCredentials(t)
	var calls atomic.Int32
	wiki := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`["fixture",["Approved result"],["approved endpoint fallback"],["https://example.com/approved"]]`))
	}))
	defer wiki.Close()
	t.Setenv("UIAI_WIKIPEDIA_SEARCH_API_URL", wiki.URL)
	cfg := &config.Config{Search: config.SearchConfig{DefaultProvider: "brave", Providers: map[string]config.SearchProviderConfig{"brave": {APIKey: "synthetic-key", APIURL: "http://outside.example/search"}}}}
	isolateEvidencePublicationStorage(t, cfg)
	if resolveDefaultSearchProvider(cfg) != "brave" {
		t.Fatal("broken explicit default silently replaced")
	}
	r := setupRouter(cfg, NewSearchRuntime())
	q := url.QueryEscape("synthetic endpoint approval " + wiki.URL)
	out := httptest.NewRecorder()
	r.ServeHTTP(out, httptest.NewRequest(http.MethodGet, "/api/search/?q="+q, nil))
	if out.Code != 503 || calls.Load() != 0 {
		t.Fatal("unapproved provider received query from broken default")
	}
	cfg.Search.FallbackProviders = []string{"wikipedia"}
	out = httptest.NewRecorder()
	r.ServeHTTP(out, httptest.NewRequest(http.MethodGet, "/api/search/?q="+q, nil))
	if (out.Code != 200 && out.Code != 202) || calls.Load() != 1 || !strings.Contains(out.Body.String(), "approved endpoint fallback") {
		t.Fatal("explicitly approved fallback not honored")
	}
	cfg.Search.DefaultProvider = "missing-adapter"
	guide, _ := readSetup(t, r, "/api/tools/agent-card")
	if guide["default_provider"] != "missing-adapter" || setupAction(t, guide, "missing-adapter") != "choose_supported_provider" {
		t.Fatal("invalid explicit default hidden by readiness")
	}
}
