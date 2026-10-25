package routes

import (
	"context"
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func researchFixture(t *testing.T) (func(searchRequest) *httptest.ResponseRecorder, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	cfg, _, _ := fallbackFixture(t)
	isolatedSearchCache(t)
	t.Setenv("UIAI_SEARCH_CACHE_TTL_SECONDS", "3600")
	brave, wiki := new(atomic.Int32), new(atomic.Int32)
	a := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		brave.Add(1)
		_, _ = w.Write([]byte(`{"web":{"results":[{"title":"Shared guide","url":"https://example.test/shared?utm_source=brave","description":"shared"},{"title":"Brave only","url":"https://example.test/brave"}]}}`))
	}))
	t.Cleanup(a.Close)
	b := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wiki.Add(1)
		_, _ = w.Write([]byte(`["fixture",["Wiki only","Shared Guide!"],["wiki","shared"],["https://example.test/wiki","https://example.test/shared?utm_source=wiki"]]`))
	}))
	t.Cleanup(b.Close)
	provider := cfg.Search.Providers["brave"]
	provider.APIURL = a.URL
	cfg.Search.Providers["brave"] = provider
	t.Setenv("UIAI_WIKIPEDIA_SEARCH_API_URL", b.URL)
	router := chi.NewRouter()
	MountSearchRoutes(router, cfg)
	call := func(input searchRequest) *httptest.ResponseRecorder {
		body, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest(http.MethodPost, "https://evidence.example/", strings.NewReader(string(body)))
		setCompleteEvidenceScopeHeaders(req, completeEvidenceScope())
		out := httptest.NewRecorder()
		router.ServeHTTP(out, req)
		return out
	}
	return call, brave, wiki
}
func TestSearchResearchFusionPreservesSourceEvidenceAndCache(t *testing.T) {
	call, brave, wiki := researchFixture(t)
	for i := 0; i < 2; i++ {
		out := call(searchRequest{Query: "fixture", Providers: []string{"brave", "wikipedia"}, Limit: 3})
		var response searchResponse
		if err := json.NewDecoder(out.Body).Decode(&response); err != nil {
			t.Fatal(err)
		}
		if out.Code != 200 || response.Provider != "fusion" || response.FusionMethod != "rrf60" || len(response.Providers) != 2 || len(response.Results) != 3 || response.Cached != (i == 1) {
			t.Fatalf("unexpected fused response status=%d count=%d", out.Code, len(response.Results))
		}
		first := response.Results[0]
		if first.Title != "Shared guide" || first.Provider != "brave" || len(first.Sources) != 2 || first.FusionScore <= response.Results[1].FusionScore || first.Rank != 1 || first.EvidenceRef != searchEvidenceRef("brave", "fixture", 1) {
			t.Fatal("fusion lost deterministic first citation/contribution rank")
		}
		if first.Sources[1].Provider != "wikipedia" || first.Sources[1].Rank != 2 || first.Sources[1].EvidenceRef != searchEvidenceRef("wikipedia", "fixture", 2) {
			t.Fatal("fusion invented or lost source-specific evidence")
		}
	}
	if brave.Load() != 1 || wiki.Load() != 1 {
		t.Fatal("research repeat did not reuse existing provider caches")
	}
}
func TestSearchResearchPolicyRejectsBeforeAnyTraffic(t *testing.T) {
	call, brave, wiki := researchFixture(t)
	for _, request := range []searchRequest{
		{Provider: "brave", Providers: []string{"brave", "wikipedia"}},
		{Providers: []string{"brave", "tavily"}},
		{Providers: []string{"brave", "unknown"}},
		{Providers: []string{"brave", "brave"}},
		{Providers: []string{"brave", "wikipedia", "exa", "tavily"}},
	} {
		request.Query = "fixture"
		out := call(request)
		if out.Code != 400 && out.Code != 403 {
			t.Fatal("invalid/unapproved research request accepted")
		}
	}
	if brave.Load() != 0 || wiki.Load() != 0 {
		t.Fatal("validation leaked query to a provider")
	}
}
func TestSearchResearchPartialSuccessAndAllLimitedAreHonest(t *testing.T) {
	cfg, _, wiki := fallbackFixture(t)
	req := httptest.NewRequest(http.MethodPost, "https://evidence.example/api/search", nil)
	setCompleteEvidenceScopeHeaders(req, completeEvidenceScope())
	input := searchRequest{Query: "fixture", Providers: []string{"brave", "wikipedia"}}
	out := httptest.NewRecorder()
	runSearch(out, req, cfg, input)
	var response searchResponse
	if err := json.NewDecoder(out.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if out.Code != 200 || len(response.Providers) != 1 || response.Providers[0] != "wikipedia" || len(response.ProviderErrors) != 1 || response.ProviderErrors[0].UpstreamStatus != 429 || response.Cached || wiki.Load() != 1 {
		t.Fatal("partial fusion hid a failed lane or reported full cached success")
	}
	denied := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(429)
		_, _ = w.Write([]byte("synthetic-private-body"))
	}))
	defer denied.Close()
	t.Setenv("UIAI_WIKIPEDIA_SEARCH_API_URL", denied.URL)
	out = httptest.NewRecorder()
	runSearch(out, req, cfg, input)
	if out.Code != 429 || out.Header().Get("Retry-After") != "7" || strings.Contains(out.Body.String(), "synthetic-private-body") {
		t.Fatal("all-limited became success/503 or lost bounded retry metadata")
	}
}
func TestSearchFusionSourceSlicesAreCloned(t *testing.T) {
	source := []searchResult{{Sources: []searchResultSource{{Provider: "brave", Rank: 1}}}}
	copy := cloneSearchResults(source)
	copy[0].Sources[0].Provider = "mutated"
	if source[0].Sources[0].Provider != "brave" {
		t.Fatal("cloned result source metadata aliased input")
	}
}

func TestSearchResearchInheritsCallerDeadline(t *testing.T) {
	cfg, _, _ := fallbackFixture(t)
	var calls atomic.Int32
	held := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); <-r.Context().Done() }))
	defer held.Close()
	provider := cfg.Search.Providers["brave"]
	provider.APIURL = held.URL
	cfg.Search.Providers["brave"] = provider
	t.Setenv("UIAI_WIKIPEDIA_SEARCH_API_URL", held.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	req := httptest.NewRequest(http.MethodPost, "https://evidence.example/api/search", nil).WithContext(ctx)
	out := httptest.NewRecorder()
	started := time.Now()
	runSearch(out, req, cfg, searchRequest{Query: "deadline-fixture", Providers: []string{"brave", "wikipedia"}})
	if time.Since(started) > time.Second || out.Code == 200 || calls.Load() != 2 {
		t.Fatal("research ignored caller deadline or failed to bound concurrent lanes")
	}
}
