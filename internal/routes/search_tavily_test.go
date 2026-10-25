package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

func TestSearchTavilyRequestNormalizationAndSafeDefaults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.Header.Get("Authorization") != "Bearer synthetic-tavily-key" || strings.Contains(r.URL.String(), "synthetic-tavily-key") {
			t.Error("invalid credential transport")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["query"] != "fixture" || body["max_results"] != float64(1) || body["search_depth"] != "basic" || body["auto_parameters"] != false || body["include_answer"] != false || body["include_raw_content"] != false {
			t.Error("request enabled unapproved work or changed query/limit")
		}
		if _, ok := body["api_key"]; ok {
			t.Error("credential copied into request JSON")
		}
		_, _ = w.Write([]byte(`{"results":[{"title":"First","url":"https://example.test/a?token=synthetic-url-secret","content":"password: synthetic-content-secret","published_date":"2026-01-01"},{"title":"Second","url":"https://example.test/b"}]}`))
	}))
	defer server.Close()
	cfg := &config.Config{}
	cfg.Search.Providers = map[string]config.SearchProviderConfig{"tavily": {APIURL: server.URL, APIKey: "synthetic-tavily-key"}}
	results, err := searchTavily(context.Background(), cfg, "fixture", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Source != "Tavily" || results[0].Age != "2026-01-01" {
		t.Fatal("normalization lost source or limit")
	}
	data, _ := json.Marshal(results)
	if strings.Contains(string(data), "synthetic-url-secret") || strings.Contains(string(data), "synthetic-content-secret") {
		t.Fatal("result secret survived canonical normalization")
	}
}

func TestSearchExplicitMissingCredentialCannotReuseEnvironment(t *testing.T) {
	t.Setenv("TAVILY_API_KEY", "synthetic-unapproved-environment-key")
	cfg := &config.Config{}
	cfg.Search.Providers = map[string]config.SearchProviderConfig{"tavily": {APIKeyEnv: "UIAI_MISSING_TAVILY_FIXTURE"}}
	t.Setenv("UIAI_MISSING_TAVILY_FIXTURE", "")
	if _, err := searchCredential(cfg, "tavily"); err == nil {
		t.Fatal("explicit credential source bypassed with conventional environment key")
	}
}

func TestSearchJSONRejectsOversizedAndCredentialBearingErrors(t *testing.T) {
	for _, tc := range []struct{ name, body string }{{"oversized", strings.Repeat("x", maxSearchResponseBytes+1)}, {"invalid", `{"synthetic-body-secret":`}} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = w.Write([]byte(tc.body)) }))
			defer server.Close()
			req, _ := http.NewRequest(http.MethodGet, server.URL, nil)
			var decoded any
			err := searchJSON(req, &decoded)
			if err == nil || strings.Contains(err.Error(), "synthetic-body-secret") {
				t.Fatal("invalid response accepted or provider body exposed")
			}
		})
	}
}

func TestSearchTavilyRetainsQuotaStatusWithoutProviderBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "synthetic-header-secret")
		w.WriteHeader(432)
		_, _ = w.Write([]byte("synthetic-provider-secret"))
	}))
	defer server.Close()
	cfg := &config.Config{}
	cfg.Search.Providers = map[string]config.SearchProviderConfig{"tavily": {APIURL: server.URL, APIKey: "synthetic-key"}}
	_, err := searchTavily(context.Background(), cfg, "fixture", 1)
	use, ok := err.(*upstreamSearchError)
	if !ok || use.status != 432 || use.retryAfter != "" || strings.Contains(err.Error(), "synthetic-provider-secret") {
		t.Fatal("quota provenance lost or provider-controlled material leaked")
	}
}

func TestSearchFallbackToTavilyUsesNativeNormalizedPacket(t *testing.T) {
	cfg, brave, wiki := fallbackFixture(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"results":[{"title":"Fixture","url":"https://example.test/tavily","content":"snippet"}]}`))
	}))
	defer server.Close()
	cfg.Search.FallbackProviders = []string{"tavily"}
	cfg.Search.Providers["tavily"] = config.SearchProviderConfig{APIURL: server.URL, APIKey: "synthetic-tavily-key"}
	req := httptest.NewRequest(http.MethodGet, "https://evidence.example/api/search", nil)
	setCompleteEvidenceScopeHeaders(req, completeEvidenceScope())
	res := httptest.NewRecorder()
	runSearch(res, req, cfg, searchRequest{Query: "fixture"})
	var body searchResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if res.Code != 200 || calls != 1 || brave.Load() != 1 || wiki.Load() != 0 || body.Provider != "tavily" || len(body.Results) != 1 || !strings.HasPrefix(body.Results[0].EvidenceRef, "uiai-search:tavily:") {
		t.Fatal("approved adapter fallback lost normalized winning evidence")
	}
}

func TestSearchTransportMasksInvalidCredentialHeaderAndPartialRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()
	for _, key := range []string{"synthetic-key", "synthetic-header-secret\ninvalid"} {
		cfg := &config.Config{}
		cfg.Search.Providers = map[string]config.SearchProviderConfig{"tavily": {APIURL: server.URL, APIKey: key}}
		_, err := searchTavily(context.Background(), cfg, "fixture", 1)
		if err == nil || strings.Contains(err.Error(), "synthetic-header-secret") {
			t.Fatal("partial read accepted or credential header exposed")
		}
	}
}
