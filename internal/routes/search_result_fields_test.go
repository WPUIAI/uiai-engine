package routes

import (
	"context"
	"encoding/json"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSearchNormalizedFieldsPreserveLegacyAndUnknowns(t *testing.T) {
	score := 0.0
	r := sanitizeSearchResult(searchResult{Title: "Fixture", URL: "https://example.test", Description: "password: synthetic-private-canary", Age: "2026-01-01", Source: "Tavily", Score: &score})
	results := []searchResult{r}
	annotateSearchEvidence(results, "tavily", "fixture")
	r = results[0]
	if r.Description == "" || r.Snippet != r.Description || strings.Contains(r.Snippet, "synthetic-private-canary") || r.Provider != "tavily" || r.Source != "Tavily" || r.PublishedAt != "2026-01-01" || r.Score == nil || *r.Score != 0 {
		t.Fatal("normalized aliases or explicit zero score lost")
	}
	score = 99
	if *r.Score != 0 {
		t.Fatal("normalization retained mutable score alias")
	}
	legacy := sanitizeSearchResult(searchResult{Description: "legacy snippet", Age: "2 days ago"})
	if legacy.Snippet != "legacy snippet" || legacy.PublishedAt != "" || legacy.Score != nil || legacy.Age != "2 days ago" {
		t.Fatal("legacy snippet/relative age altered or unknown score/date invented")
	}
	for _, bad := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		r := sanitizeSearchResult(searchResult{Score: &bad})
		if r.Score != nil {
			t.Fatal("nonfinite score reached JSON contract")
		}
		if _, err := json.Marshal(r); err != nil {
			t.Fatal(err)
		}
	}
}
func TestSearchPublishedAtPreservesSourcePrecision(t *testing.T) {
	for raw, want := range map[string]string{"2026-01-01": "2026-01-01", "2026-01-01T03:04:05.123+02:00": "2026-01-01T03:04:05.123+02:00", "2026-02-30": "", "recently": "", "": ""} {
		if got := searchPublishedAt(raw); got != want {
			t.Fatalf("publication precision invented/lost for%q", raw)
		}
	}
}
func TestSearchCacheScoreDoesNotAliasCaller(t *testing.T) {
	isolatedSearchCache(t)
	value := 0.75
	setCachedSearch("tavily", "score-copy", 1, []searchResult{{Score: &value}}, time.Now().Add(time.Minute))
	value = 1
	result, ok := getCachedSearch("tavily", "score-copy", 1, time.Now())
	if !ok || *result[0].Score != 0.75 {
		t.Fatal("cache retained input score pointer")
	}
	*result[0].Score = 2
	result, _ = getCachedSearch("tavily", "score-copy", 1, time.Now())
	if *result[0].Score != 0.75 {
		t.Fatal("returned score pointer mutated cache")
	}
}
func TestSearchHTTPNormalizedFieldsPreserveLegacyConsumer(t *testing.T) {
	cfg, _, _ := fallbackFixture(t)
	req := httptest.NewRequest(http.MethodGet, "https://evidence.example/api/search", nil)
	setCompleteEvidenceScopeHeaders(req, completeEvidenceScope())
	out := httptest.NewRecorder()
	runSearch(out, req, cfg, searchRequest{Query: "fixture"})
	var packet struct {
		Schema         string `json:"schema"`
		ArtifactSchema string `json:"artifact_schema"`
		Provider       string `json:"provider"`
		Results        []struct{ Title, URL, Description, Snippet, Provider, Source, EvidenceRef string }
	}
	if err := json.NewDecoder(out.Body).Decode(&packet); err != nil {
		t.Fatal(err)
	}
	if out.Code != 200 || packet.Schema != "uiai.artifact_result.v2" || packet.ArtifactSchema != "uiai.search_results.v1" || len(packet.Results) != 1 {
		t.Fatal("legacy envelope contract changed")
	}
	r := packet.Results[0]
	if r.Description != "description" || r.Snippet != r.Description || r.Provider != packet.Provider || r.Source != "Wikipedia" || r.Title != "Fixture" || r.URL == "" {
		t.Fatal("legacy/new consumer mapping diverged")
	}
}

func TestSearchAdaptersPreserveObservedScoresAndAbsoluteDates(t *testing.T) {
	for _, id := range []string{"exa", "tavily", "searxng"} {
		t.Run(id, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = w.Write([]byte(`{"results":[{"title":"Fixture","url":"https://example.test","content":"snippet","highlights":["snippet"],"score":0.625,"published_date":"2026-01-01","publishedDate":"2026-01-01"}]}`))
			}))
			defer server.Close()
			cfg := &config.Config{}
			cfg.Search.Providers = map[string]config.SearchProviderConfig{id: {APIURL: server.URL, APIKey: "synthetic-fixture-key"}}
			results, err := searchProviderMeta(id).search(context.Background(), cfg, "fixture", 1)
			if err != nil {
				t.Fatal(err)
			}
			if len(results) != 1 || results[0].Score == nil || *results[0].Score != 0.625 || results[0].PublishedAt != "2026-01-01" || results[0].Snippet == "" {
				t.Fatal("observed vendor fields lost normalization")
			}
		})
	}
}
