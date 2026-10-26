package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

func TestSearchSetupErrorsCarrySafeNextActions(t *testing.T) {
	clearSearchSetupCredentials(t)
	for _, tc := range []struct {
		name, provider, action, field string
		config                        config.SearchProviderConfig
		status                        int
	}{
		{"missing", "brave", "connect_existing_credential_reference", "search.providers.brave.api_key_env", config.SearchProviderConfig{}, 503},
		{"disabled", "brave", "keep_disabled", "", config.SearchProviderConfig{Disabled: true}, 503},
		{"unreadable", "brave", "repair_existing_credential_reference", "", config.SearchProviderConfig{APIKeyFile: "/private/synthetic-reference-do-not-echo"}, 503},
		{"endpoint", "brave", "repair_existing_endpoint", "search.providers.brave.api_url", config.SearchProviderConfig{APIKey: "synthetic-key", APIURL: "https://user:synthetic-url-do-not-echo@api.search.brave.com/search"}, 503},
		{"unsupported", "not-an-adapter", "choose_supported_provider", "", config.SearchProviderConfig{}, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{tc.provider: tc.config}}}
			r := setupRouter(cfg, NewSearchRuntime())
			out := httptest.NewRecorder()
			r.ServeHTTP(out, httptest.NewRequest(http.MethodGet, "/api/search/?q=synthetic-recovery&provider="+tc.provider, nil))
			if out.Code != tc.status {
				t.Fatalf("status %d", out.Code)
			}
			var body struct {
				Hint  string              `json:"suggested_next_action"`
				Setup searchProviderSetup `json:"setup"`
			}
			if err := json.Unmarshal(out.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if tc.name == "unsupported" {
				var full struct {
					Supported []string `json:"supported_providers"`
				}
				if err := json.Unmarshal(out.Body.Bytes(), &full); err != nil {
					t.Fatal(err)
				}
				for _, id := range full.Supported {
					if id == tc.provider {
						t.Fatal("unimplemented configured provider advertised as supported")
					}
				}
			}
			if body.Setup.NextAction != tc.action || !strings.Contains(body.Hint, tc.action) || !strings.Contains(body.Hint, "/api/search/providers") || (tc.field != "" && !strings.Contains(body.Hint, tc.field)) {
				t.Fatal("missing machine and cross-client recovery guidance")
			}
			for _, secret := range []string{"synthetic-reference-do-not-echo", "synthetic-url-do-not-echo", "synthetic-key"} {
				if strings.Contains(out.Body.String(), secret) {
					t.Fatal("recovery leaked private configuration")
				}
			}
		})
	}
}

func TestSearchCoolingRecoveryPreservesRetryWithoutProbe(t *testing.T) {
	clearSearchSetupCredentials(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) }))
	defer server.Close()
	cfg := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{"brave": {APIKey: "synthetic-key", APIURL: server.URL}}}}
	state, _ := testSearchBreakers()
	for i := 0; i < 2; i++ {
		a, _ := state.acquire("brave")
		state.complete("brave", a, &upstreamSearchError{status: 429, retryAfter: "60"})
	}
	out := httptest.NewRecorder()
	setupRouter(cfg, state).ServeHTTP(out, httptest.NewRequest(http.MethodGet, "/api/search/?q=synthetic-cooling-recovery&provider=brave", nil))
	var body struct {
		Hint      string              `json:"suggested_next_action"`
		Setup     searchProviderSetup `json:"setup"`
		Retryable bool                `json:"retryable"`
	}
	if err := json.Unmarshal(out.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if out.Code != 503 || out.Header().Get("Retry-After") != "60" || !body.Retryable || body.Setup.NextAction != "wait_then_retry" || body.Setup.RetryAfterSeconds != 60 || calls.Load() != 0 || !strings.Contains(body.Hint, "wait_then_retry") {
		t.Fatal("cooling recovery lost retry semantics")
	}
}

func TestSearchRecoveryDoesNotMisdiagnoseUpstreamResponseAsSetup(t *testing.T) {
	clearSearchSetupCredentials(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		_, _ = w.Write([]byte("invalid JSON fixture"))
	}))
	defer server.Close()
	cfg := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{"brave": {APIKey: "synthetic-key", APIURL: server.URL}}}}
	out := httptest.NewRecorder()
	setupRouter(cfg, NewSearchRuntime()).ServeHTTP(out, httptest.NewRequest(http.MethodGet, "/api/search/?q=synthetic-response-failure&provider=brave", nil))
	var body map[string]any
	if err := json.Unmarshal(out.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if out.Code != 503 || calls.Load() != 1 || body["setup"] != nil {
		t.Fatal("provider response failure incorrectly asks to change setup")
	}
	cfg.Search.Providers = map[string]config.SearchProviderConfig{"wikipedia": {Disabled: true}}
	out = httptest.NewRecorder()
	setupRouter(cfg, NewSearchRuntime()).ServeHTTP(out, httptest.NewRequest(http.MethodGet, "/api/search/?q=synthetic-no-providers", nil))
	if out.Code != 503 || !strings.Contains(out.Body.String(), "suggested_next_action") || !strings.Contains(out.Body.String(), "/api/search/providers") {
		t.Fatal("empty setup has no discovery recovery")
	}
}
