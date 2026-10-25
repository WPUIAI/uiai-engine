package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSearchDedupePreservesMeaningfulIdentityAndCitations(t *testing.T) {
	input := []searchResult{
		{Title: "Agent search guide", URL: "https://EXAMPLE.test/page?item=1&utm_source=one", Source: "first"},
		{Title: "Agent Search Guide!", URL: "https://example.test/page?utm_source=two&item=1"},
		{Title: "Different topic entirely", URL: "https://example.test/page?item=1&utm_source=three"},
		{Title: "Agent search guide", URL: "https://example.test/page?item=2&utm_source=one"},
		{Title: "Agent search guide", URL: "https://another.example.test/page?item=1"},
		{Title: "Changed title same exact URL", URL: "https://example.test/page?item=1&utm_source=one"},
	}
	out := deduplicateSearchResults(input)
	if len(out) != 4 {
		t.Fatalf("expected4 distinct identities, got%d", len(out))
	}
	if out[0].URL != input[0].URL || out[0].Title != input[0].Title || out[0].Source != "first" {
		t.Fatal("dedupe rewrote original citation or first-result provenance")
	}
	if len(input) != 6 || input[1].Title != "Agent Search Guide!" {
		t.Fatal("dedupe mutated input")
	}
}
func TestSearchDedupeRejectsUnsafeOrMalformedCitationURLs(t *testing.T) {
	for _, raw := range []string{"javascript:alert(1)", "data:text/plain,value", "https://user:synthetic-secret@example.test/", "/relative", "://invalid", "https://example.test/?x=%zz"} {
		if got := deduplicateSearchResults([]searchResult{{Title: "Fixture", URL: raw}}); len(got) != 0 {
			t.Fatal("unsafe/malformed citation survived")
		}
	}
	if similarSearchTitles("", "") || similarSearchTitles("guide to accounts", "unrelated subject") {
		t.Fatal("empty/unrelated titles merged")
	}
}
func TestSearchHTTPDedupeRanksAndCacheAreConsistent(t *testing.T) {
	cfg, _, _ := fallbackFixture(t)
	t.Setenv("UIAI_SEARCH_CACHE_TTL_SECONDS", "3600")
	isolatedSearchCache(t)
	calls := 0
	wiki := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		_, _ = w.Write([]byte(`["fixture",["Guide","Guide!","Distinct"],["first","duplicate","other"],["https://example.test/a?utm_source=one","https://example.test/a?utm_source=two","https://example.test/b?id=2"]]`))
	}))
	defer wiki.Close()
	t.Setenv("UIAI_WIKIPEDIA_SEARCH_API_URL", wiki.URL)
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "https://evidence.example/api/search", nil)
		setCompleteEvidenceScopeHeaders(req, completeEvidenceScope())
		out := httptest.NewRecorder()
		runSearch(out, req, cfg, searchRequest{Query: "dedupe-fixture", Provider: "wikipedia", Limit: 5})
		var response searchResponse
		if err := json.NewDecoder(out.Body).Decode(&response); err != nil {
			t.Fatal(err)
		}
		if out.Code != 200 || response.Count != 2 || len(response.Results) != 2 || response.Cached != (i == 1) {
			t.Fatal("deduped response and cached response disagree")
		}
		for rank, result := range response.Results {
			if result.Rank != rank+1 || result.Provider != "wikipedia" || result.EvidenceRef != searchEvidenceRef("wikipedia", "dedupe-fixture", rank+1) {
				t.Fatal("dedupe did not preserve dense citation ranks")
			}
		}
	}
	if calls != 1 {
		t.Fatal("dedupe triggered another provider fetch instead of preserving cache")
	}
}
