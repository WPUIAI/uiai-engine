package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

func TestSearchResearchRejectionRecoveryPreservesPolicyAndZeroTraffic(t *testing.T) {
	for _, tc := range []struct {
		name                   string
		request                searchRequest
		status                 int
		code, action, provider string
		disable                string
	}{
		{"conflict", searchRequest{Provider: "brave", Providers: []string{"brave", "wikipedia"}}, 400, "conflicting_search_providers", "choose_search_mode", "", ""},
		{"too_many", searchRequest{Providers: []string{"brave", "wikipedia", "exa", "tavily"}}, 400, "too_many_search_providers", "reduce_research_provider_count", "", ""},
		{"unsupported", searchRequest{Providers: []string{"brave", " Missing-Adapter "}}, 400, "unsupported_provider", "choose_supported_provider", "missing-adapter", ""},
		{"not_approved_despite_key", searchRequest{Providers: []string{"brave", " TAVILY "}}, 403, "search_provider_not_approved", "request_provider_disclosure_approval", "tavily", ""},
		{"disabled", searchRequest{Providers: []string{"brave", " WIKIPEDIA "}}, 403, "search_provider_disabled", "keep_disabled", "wikipedia", "wikipedia"},
		{"approval_before_disable", searchRequest{Providers: []string{"brave", "tavily"}}, 403, "search_provider_not_approved", "request_provider_disclosure_approval", "tavily", "tavily"},
		{"duplicates", searchRequest{Providers: []string{" BRAVE ", "brave"}}, 400, "research_requires_distinct_providers", "choose_distinct_approved_providers", "", ""},
		{"single", searchRequest{Providers: []string{"brave"}}, 400, "research_requires_distinct_providers", "choose_distinct_approved_providers", "", ""},
		{"blank_id", searchRequest{Providers: []string{"brave", " "}}, 400, "unsupported_provider", "choose_supported_provider", "", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			clearSearchSetupCredentials(t)
			cfg, brave, wiki := fallbackFixture(t)
			// Even a present credential/valid local endpoint grants no research disclosure.
			cfg.Search.Providers["tavily"] = config.SearchProviderConfig{APIKey: "synthetic-private-key-never-echo", APIKeyFile: "/private/synthetic-reference-never-echo", APIURL: cfg.Search.Providers["brave"].APIURL}
			cfg.Search.Providers["missing-adapter"] = config.SearchProviderConfig{APIKey: "synthetic-private-key-never-echo"}
			if tc.disable != "" {
				p := cfg.Search.Providers[tc.disable]
				p.Disabled = true
				cfg.Search.Providers[tc.disable] = p
			}
			before, err := json.Marshal(cfg.Search)
			if err != nil {
				t.Fatal(err)
			}
			tc.request.Query = "synthetic-query-never-echo"
			payload, err := json.Marshal(tc.request)
			if err != nil {
				t.Fatal(err)
			}
			out := httptest.NewRecorder()
			setupRouter(cfg, NewSearchRuntime()).ServeHTTP(out, httptest.NewRequest(http.MethodPost, "/api/search/", bytes.NewReader(payload)))
			var body struct {
				Error, Provider string
				Hint            string              `json:"suggested_next_action"`
				Setup           searchProviderSetup `json:"setup"`
				Supported       []string            `json:"supported_providers"`
			}
			if err = json.Unmarshal(out.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if out.Code != tc.status || body.Error != tc.code || body.Provider != tc.provider {
				t.Fatalf("rejection policy changed: status=%d code=%s provider=%s", out.Code, body.Error, body.Provider)
			}
			if body.Setup.NextAction != tc.action || !strings.HasPrefix(body.Hint, tc.action+": ") || !strings.Contains(body.Hint, "/api/search/providers") {
				t.Fatal("missing shared research recovery")
			}
			if tc.code == "search_provider_not_approved" && !strings.Contains(body.Hint, "authorization") {
				t.Fatal("approval recovery does not preserve consent")
			}
			if tc.code == "unsupported_provider" {
				if len(body.Supported) == 0 {
					t.Fatal("no supported alternatives")
				}
				for _, id := range body.Supported {
					if id == "missing-adapter" {
						t.Fatal("configured unsupported adapter advertised as installed")
					}
				}
			}
			for _, canary := range []string{"synthetic-private-key-never-echo", "synthetic-reference-never-echo", "synthetic-query-never-echo", cfg.Search.Providers["brave"].APIURL} {
				if strings.Contains(out.Body.String(), canary) {
					t.Fatal("private configuration/query echoed into recovery")
				}
			}
			after, err := json.Marshal(cfg.Search)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) || brave.Load() != 0 || wiki.Load() != 0 {
				t.Fatal("recovery changed configuration or disclosed a query")
			}
		})
	}
}
