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

func TestSearchExaProtocolNormalizationAndCredentialBoundary(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" || r.Header.Get("x-api-key") != "synthetic-exa-key" || strings.Contains(r.URL.String(), "synthetic-exa-key") {
			t.Error("invalid Exa credential transport")
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if body["query"] != "fixture" || body["numResults"] != float64(1) || body["type"] != "auto" {
			t.Error("Exa protocol query/limit/type lost")
		}
		if _, ok := body["api_key"]; ok {
			t.Error("credential copied into JSON")
		}
		_, _ = w.Write([]byte(`{"results":[{"title":"One","url":"https://example.test/a?token=synthetic-url-secret","publishedDate":"2026-01-01T00:00:00Z","highlights":["password: synthetic-highlight-secret"]},{"title":"Two","url":"https://example.test/b"}]}`))
	}))
	defer server.Close()
	cfg := &config.Config{}
	cfg.Search.Providers = map[string]config.SearchProviderConfig{"exa": {APIURL: server.URL, APIKey: "synthetic-exa-key"}}
	results, err := searchExa(context.Background(), cfg, "fixture", 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Source != "Exa" || results[0].Age != "2026-01-01T00:00:00Z" {
		t.Fatal("Exa mapping lost result limit or provenance")
	}
	data, _ := json.Marshal(results)
	if strings.Contains(string(data), "synthetic-url-secret") || strings.Contains(string(data), "synthetic-highlight-secret") {
		t.Fatal("Exa result secret escaped normalization")
	}
}

func TestSearchFallbackToExaPreservesWinningEvidence(t *testing.T) {
	cfg, brave, wiki := fallbackFixture(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`{"results":[{"title":"Fixture","url":"https://example.test/exa","highlights":["snippet"]}]}`))
	}))
	defer server.Close()
	cfg.Search.FallbackProviders = []string{"exa"}
	cfg.Search.Providers["exa"] = config.SearchProviderConfig{APIURL: server.URL, APIKey: "synthetic-exa-key"}
	req := httptest.NewRequest(http.MethodGet, "https://evidence.example/api/search", nil)
	setCompleteEvidenceScopeHeaders(req, completeEvidenceScope())
	res := httptest.NewRecorder()
	runSearch(res, req, cfg, searchRequest{Query: "fixture"})
	var body searchResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if res.Code != 200 || calls != 1 || brave.Load() != 1 || wiki.Load() != 0 || body.Provider != "exa" || len(body.Results) != 1 || !strings.HasPrefix(body.Results[0].EvidenceRef, "uiai-search:exa:") {
		t.Fatal("approved Exa fallback lost normalized winning evidence")
	}
}
