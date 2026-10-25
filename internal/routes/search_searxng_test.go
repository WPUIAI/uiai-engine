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

func TestSearchSearxNGRequiresExplicitSafeEndpoint(t *testing.T) {
	for _, raw := range []string{"", "http://example.test/search", "https://user:synthetic-secret@example.test/search", "file:///etc/passwd", "https://example.test/search?token=synthetic-secret", "https://example.test/search#fragment"} {
		cfg := &config.Config{}
		cfg.Search.Providers = map[string]config.SearchProviderConfig{"searxng": {APIURL: raw}}
		if ready, _ := searchProviderConfigured(cfg, "searxng"); ready {
			t.Fatalf("unsafe/unconfigured endpoint advertised ready%q", raw)
		}
		if _, err := searchSearxNG(context.Background(), cfg, "fixture", 1); err == nil || strings.Contains(err.Error(), "synthetic-secret") {
			t.Fatal("endpoint accepted or sensitive URL exposed")
		}
	}
	cfg := &config.Config{}
	cfg.Search.Providers = map[string]config.SearchProviderConfig{"searxng": {APIURL: "https://operator.example.test/prefix/search"}}
	if ready, reason := searchProviderConfigured(cfg, "searxng"); !ready || reason != "" {
		t.Fatal("explicit HTTPS endpoint not accepted")
	}
	cfg.Search.Providers["searxng"] = config.SearchProviderConfig{APIURL: "https://operator.example.test/search", Disabled: true}
	if ready, _ := searchProviderConfigured(cfg, "searxng"); ready {
		t.Fatal("disabled keyless provider ready")
	}
}
func TestSearchSearxNGKeylessFormProtocolAndRedaction(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.URL.RawQuery != "" || r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" || r.Header.Get("Authorization") != "" || r.Header.Get("x-api-key") != "" {
			t.Error("SearxNG request leaked query/credential or changed protocol")
		}
		if err := r.ParseForm(); err != nil {
			t.Error(err)
		}
		if r.PostForm.Get("q") != "fixture & encoded" || r.PostForm.Get("format") != "json" {
			t.Error("SearxNG form encoding lost query")
		}
		_, _ = w.Write([]byte(`{"results":[{"title":"Fixture","url":"https://example.test/?token=synthetic-url-canary","content":"password: synthetic-content-canary","publishedDate":"2026-01-01"},{"title":"Second","url":"https://example.test/second"}]}`))
	}))
	defer server.Close()
	cfg := &config.Config{}
	cfg.Search.Providers = map[string]config.SearchProviderConfig{"searxng": {APIURL: server.URL}}
	results, err := searchSearxNG(context.Background(), cfg, "fixture & encoded", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Source != "SearxNG" || results[0].Age != "2026-01-01" {
		t.Fatal("SearxNG mapping/provenance/limit lost")
	}
	data, _ := json.Marshal(results)
	if strings.Contains(string(data), "synthetic-url-canary") || strings.Contains(string(data), "synthetic-content-canary") {
		t.Fatal("SearxNG result bypassed canonical redaction")
	}
}
func TestSearchFallbackToExplicitSearxNGAndForbiddenJSONSemantics(t *testing.T) {
	cfg, brave, wiki := fallbackFixture(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"results":[{"title":"Fixture","url":"https://example.test/searx","content":"snippet"}]}`))
	}))
	defer server.Close()
	cfg.Search.FallbackProviders = []string{"searxng"}
	cfg.Search.Providers["searxng"] = config.SearchProviderConfig{APIURL: server.URL}
	req := httptest.NewRequest(http.MethodGet, "https://evidence.example/api/search", nil)
	setCompleteEvidenceScopeHeaders(req, completeEvidenceScope())
	out := httptest.NewRecorder()
	runSearch(out, req, cfg, searchRequest{Query: "fixture"})
	var body searchResponse
	if err := json.NewDecoder(out.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if out.Code != 200 || brave.Load() != 1 || wiki.Load() != 0 || calls != 1 || body.Provider != "searxng" || !strings.HasPrefix(body.Results[0].EvidenceRef, "uiai-search:searxng:") {
		t.Fatal("keyless approved fallback or winning evidence lost")
	}
	forbidden := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(403)
		_, _ = w.Write([]byte("synthetic-private-provider-body"))
	}))
	defer forbidden.Close()
	cfg.Search.Providers["searxng"] = config.SearchProviderConfig{APIURL: forbidden.URL}
	out = httptest.NewRecorder()
	runSearch(out, req, cfg, searchRequest{Query: "fixture", Provider: "searxng"})
	var failure map[string]any
	_ = json.NewDecoder(out.Body).Decode(&failure)
	if out.Code != 502 || failure["upstream_status"] != float64(403) || failure["retryable"] != false || strings.Contains(out.Body.String(), "synthetic-private-provider-body") {
		t.Fatal("JSON-disabled provider403 became false success/retryable or leaked body")
	}
}
