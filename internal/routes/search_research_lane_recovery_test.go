package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

type researchRecoveryBody struct {
	Error     string         `json:"error"`
	Hint      string         `json:"suggested_next_action"`
	Providers []string       `json:"providers"`
	Results   []searchResult `json:"results"`
	Failures  []struct {
		Provider string               `json:"provider"`
		Code     string               `json:"code"`
		Setup    *searchProviderSetup `json:"setup"`
		Hint     string               `json:"suggested_next_action"`
	} `json:"provider_errors"`
}

func TestSearchResearchAllFailedConfigurationRecovery(t *testing.T) {
	clearSearchSetupCredentials(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) }))
	defer server.Close()
	cfg := &config.Config{Search: config.SearchConfig{DefaultProvider: "exa", FallbackProviders: []string{"tavily"}, Providers: map[string]config.SearchProviderConfig{"exa": {APIURL: server.URL}, "tavily": {APIURL: server.URL}}}}
	payload := []byte(`{"query":"synthetic-config-recovery","providers":["exa","tavily"]}`)
	out := httptest.NewRecorder()
	setupRouter(cfg, NewSearchRuntime()).ServeHTTP(out, httptest.NewRequest(http.MethodPost, "/api/search/", bytes.NewReader(payload)))
	var body researchRecoveryBody
	if err := json.Unmarshal(out.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if out.Code != 502 || body.Error != "search_providers_failed" || len(body.Failures) != 2 || calls.Load() != 0 {
		t.Fatal("all-failure status/order or zero-traffic semantics changed")
	}
	for i, id := range []string{"exa", "tavily"} {
		f := body.Failures[i]
		if f.Provider != id || f.Code != "search_provider_unavailable" || f.Setup == nil || f.Setup.NextAction != "connect_existing_credential_reference" || !strings.Contains(f.Hint, "search.providers."+id+".api_key_env") || !strings.Contains(body.Hint, id+": "+f.Hint) {
			t.Fatal("missing per-provider and aggregate recovery")
		}
	}
	if strings.Contains(out.Body.String(), server.URL) {
		t.Fatal("private endpoint leaked in recovery")
	}
}

func TestSearchResearchPartialSuccessRetainsLaneRecovery(t *testing.T) {
	clearSearchSetupCredentials(t)
	cfg, brave, wiki := fallbackFixture(t)
	cfg.Search.DefaultProvider = "wikipedia"
	cfg.Search.FallbackProviders = []string{"exa"}
	cfg.Search.Providers["exa"] = config.SearchProviderConfig{APIURL: cfg.Search.Providers["brave"].APIURL}
	req := httptest.NewRequest(http.MethodPost, "https://evidence.example/api/search/", strings.NewReader(`{"query":"synthetic-partial-recovery","providers":["wikipedia","exa"]}`))
	setCompleteEvidenceScopeHeaders(req, completeEvidenceScope())
	out := httptest.NewRecorder()
	setupRouter(cfg, NewSearchRuntime()).ServeHTTP(out, req)
	var body researchRecoveryBody
	if err := json.Unmarshal(out.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if out.Code != 200 || len(body.Providers) != 1 || body.Providers[0] != "wikipedia" || len(body.Results) != 1 || len(body.Failures) != 1 || brave.Load() != 0 || wiki.Load() != 1 {
		t.Fatalf("partial status=%d providers=%d results=%d failures=%d calls=%d/%d", out.Code, len(body.Providers), len(body.Results), len(body.Failures), brave.Load(), wiki.Load())
	}
	f := body.Failures[0]
	if f.Provider != "exa" || f.Setup == nil || f.Setup.NextAction != "connect_existing_credential_reference" || !strings.Contains(f.Hint, "search.providers.exa.api_key_env") || body.Hint != "" {
		t.Fatal("partial success lost lane recovery or became a top-level error")
	}
}

func TestSearchResearchFailureRecoveryClassificationPriority(t *testing.T) {
	clearSearchSetupCredentials(t)
	missing := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{"brave": {APIURL: "http://127.0.0.1:9"}}}}
	ready := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{"brave": {APIURL: "http://127.0.0.1:9", APIKey: "synthetic-private-error-canary"}}}}
	invalid := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{"brave": {APIURL: "https://user:synthetic-private-error-canary@api.search.brave.com/search", APIKey: "synthetic-private-error-canary"}}}}
	for _, tc := range []struct {
		name         string
		cfg          *config.Config
		err          error
		code, action string
	}{
		{"missing", missing, errors.New("synthetic-private-error-canary"), "search_provider_unavailable", "connect_existing_credential_reference"},
		{"endpoint", invalid, errors.New("synthetic-private-error-canary"), "search_provider_unavailable", "repair_existing_endpoint"},
		{"circuit", missing, &searchCircuitOpenError{retryAfter: 15}, "search_provider_circuit_open", "wait_then_retry"},
		{"upstream429", missing, &upstreamSearchError{status: 429, retryAfter: "7", message: "synthetic-private-error-canary"}, "search_provider_error", ""},
		{"upstream401", missing, &upstreamSearchError{status: 401, message: "synthetic-private-error-canary"}, "search_provider_error", ""},
		{"cancelled", missing, context.Canceled, "search_cancelled", ""},
		{"deadline", missing, context.DeadlineExceeded, "search_deadline", ""},
		{"response_not_config", ready, errors.New("synthetic-private-error-canary"), "search_provider_unavailable", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := researchSearchFailure(tc.cfg, "brave", tc.err)
			if f.Code != tc.code {
				t.Fatal("error classification changed")
			}
			if tc.action == "" {
				if f.Setup != nil || f.SuggestedNextAction != "" {
					t.Fatal("upstream/cancellation/response failure incorrectly blamed on setup")
				}
			} else if f.Setup == nil || f.Setup.NextAction != tc.action || !strings.Contains(f.SuggestedNextAction, tc.action) {
				t.Fatal("missing known setup recovery")
			}
			if tc.name == "circuit" && (!f.Retryable || f.RetryAfter != "15" || f.Setup.RetryAfterSeconds != 15) {
				t.Fatal("circuit retry lost")
			}
			data, err := json.Marshal(f)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "synthetic-private-error-canary") {
				t.Fatal("provider/config private error echoed")
			}
		})
	}
}

func TestSearchResearchCoolingRetainsAggregateRetryWithoutProbe(t *testing.T) {
	clearSearchSetupCredentials(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) }))
	defer server.Close()
	cfg := &config.Config{Search: config.SearchConfig{DefaultProvider: "exa", FallbackProviders: []string{"tavily"}, Providers: map[string]config.SearchProviderConfig{"exa": {APIURL: server.URL, APIKey: "fixture"}, "tavily": {APIURL: server.URL, APIKey: "fixture"}}}}
	state, _ := testSearchBreakers()
	for _, id := range []string{"exa", "tavily"} {
		for i := 0; i < 2; i++ {
			a, _ := state.acquire(id)
			state.complete(id, a, &upstreamSearchError{status: 429, retryAfter: "60"})
		}
	}
	out := httptest.NewRecorder()
	setupRouter(cfg, state).ServeHTTP(out, httptest.NewRequest(http.MethodPost, "/api/search/", strings.NewReader(`{"query":"synthetic-cooling-recovery","providers":["exa","tavily"]}`)))
	var body researchRecoveryBody
	if err := json.Unmarshal(out.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if out.Code != 502 || out.Header().Get("Retry-After") != "60" || calls.Load() != 0 || len(body.Failures) != 2 || !strings.Contains(body.Hint, "wait_then_retry") {
		t.Fatal("cooling changed HTTP retry semantics or made probe")
	}
	for _, f := range body.Failures {
		if f.Setup == nil || f.Setup.NextAction != "wait_then_retry" || f.Setup.RetryAfterSeconds != 60 {
			t.Fatal("cooling lane recovery lost delay")
		}
	}
}
