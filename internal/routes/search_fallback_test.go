package routes

import (
	"context"
	"encoding/json"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func fallbackFixture(t *testing.T) (*config.Config, *atomic.Int32, *atomic.Int32) {
	t.Helper()
	t.Setenv("UIAI_SEARCH_CACHE_TTL_SECONDS", "0")
	t.Setenv("BRAVE_SEARCH_API_KEY", "")
	braveCalls, wikiCalls := new(atomic.Int32), new(atomic.Int32)
	brave := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		braveCalls.Add(1)
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(429)
	}))
	t.Cleanup(brave.Close)
	wiki := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wikiCalls.Add(1)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`["fixture",["Fixture"],["description"],["https://example.test/fixture"]]`))
	}))
	t.Cleanup(wiki.Close)
	t.Setenv("UIAI_WIKIPEDIA_SEARCH_API_URL", wiki.URL)
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	cfg.Search.DefaultProvider = "brave"
	cfg.Search.FallbackProviders = []string{"wikipedia"}
	cfg.Search.Providers = map[string]config.SearchProviderConfig{"brave": {APIURL: brave.URL, APIKey: "synthetic-fixture-key"}}
	return cfg, braveCalls, wikiCalls
}

func TestSearchFallbackUsesApprovedAlternateAndWinningProvenance(t *testing.T) {
	cfg, brave, wiki := fallbackFixture(t)
	req := httptest.NewRequest(http.MethodGet, "https://evidence.example/api/search?q=fixture", nil)
	setCompleteEvidenceScopeHeaders(req, completeEvidenceScope())
	res := httptest.NewRecorder()
	runSearch(res, req, cfg, searchRequest{Query: "fixture"})
	if brave.Load() != 1 || wiki.Load() != 1 {
		t.Fatalf("expected primary then approved alternate, got%d/%d", brave.Load(), wiki.Load())
	}
	if res.Code != http.StatusOK {
		t.Fatalf("alternate success lost: status%d", res.Code)
	}
	var body searchResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Provider != "wikipedia" || len(body.Results) != 1 || !strings.HasPrefix(body.Results[0].EvidenceRef, "uiai-search:wikipedia:") {
		t.Fatal("fallback misattributed winning evidence")
	}
}

func TestSearchPinnedProviderNeverDisclosesToAlternate(t *testing.T) {
	cfg, brave, wiki := fallbackFixture(t)
	res := httptest.NewRecorder()
	runSearch(res, httptest.NewRequest(http.MethodGet, "/", nil), cfg, searchRequest{Query: "fixture", Provider: "brave"})
	if res.Code != 429 || res.Header().Get("Retry-After") != "7" || brave.Load() != 1 || wiki.Load() != 0 {
		t.Fatal("pinned provider was bypassed or throttling metadata lost")
	}
}

func TestSearchFallbackRequiresExplicitConfiguration(t *testing.T) {
	cfg, _, wiki := fallbackFixture(t)
	cfg.Search.FallbackProviders = nil
	res := httptest.NewRecorder()
	runSearch(res, httptest.NewRequest(http.MethodGet, "/", nil), cfg, searchRequest{Query: "fixture"})
	if res.Code != 429 || wiki.Load() != 0 {
		t.Fatal("query disclosed to an unapproved alternate")
	}
}

func TestSearchFallbackPlanSkipsDisabledUnknownAndDuplicateProviders(t *testing.T) {
	cfg, _, _ := fallbackFixture(t)
	cfg.Search.FallbackProviders = []string{"brave", "unimplemented", "wikipedia", " Wikipedia "}
	got := searchProviderPlan(cfg, "", "brave")
	if len(got) != 2 || got[1] != "wikipedia" {
		t.Fatalf("invalid fallback plan%v", got)
	}
	cfg.Search.Providers["wikipedia"] = config.SearchProviderConfig{Disabled: true}
	if got := searchProviderPlan(cfg, "", "brave"); len(got) != 1 {
		t.Fatal("disabled provider entered fallback plan")
	}
}

func TestSearchCanceledRequestNeverReachesAnyProvider(t *testing.T) {
	cfg, brave, wiki := fallbackFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	res := httptest.NewRecorder()
	runSearch(res, req, cfg, searchRequest{Query: "fixture"})
	if brave.Load() != 0 || wiki.Load() != 0 {
		t.Fatal("canceled search reached an upstream")
	}
}

func TestSearchProviderRequestInheritsCallerDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	t.Setenv("UIAI_WIKIPEDIA_SEARCH_API_URL", server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := searchWikipediaContext(ctx, "fixture", 1)
	if err == nil || time.Since(started) > time.Second {
		t.Fatal("search transport ignored caller deadline")
	}
}

func TestSearchFallbackFailurePreservesFinalProviderStatus(t *testing.T) {
	cfg, _, _ := fallbackFixture(t)
	failed := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Header().Set("Retry-After", "13"); w.WriteHeader(429) }))
	defer failed.Close()
	t.Setenv("UIAI_WIKIPEDIA_SEARCH_API_URL", failed.URL)
	res := httptest.NewRecorder()
	runSearch(res, httptest.NewRequest(http.MethodGet, "/", nil), cfg, searchRequest{Query: "fixture"})
	var body map[string]any
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if res.Code != 429 || res.Header().Get("Retry-After") != "13" || body["provider"] != "wikipedia" {
		t.Fatal("exhausted chain lost final provider's rate-limit provenance")
	}
}

func TestSearchDisabledProviderCannotReuseOldCachedResults(t *testing.T) {
	cfg, brave, wiki := fallbackFixture(t)
	t.Setenv("UIAI_SEARCH_CACHE_TTL_SECONDS", "60")
	cfg.Search.Providers["brave"] = config.SearchProviderConfig{Disabled: true}
	setCachedSearch("brave", "disabled-private-fixture", 1, []searchResult{{Title: "must-not-be-served"}}, time.Now().Add(time.Minute))
	res := httptest.NewRecorder()
	runSearch(res, httptest.NewRequest(http.MethodGet, "/", nil), cfg, searchRequest{Query: "disabled-private-fixture", Provider: "brave", Limit: 1})
	if res.Code == 200 || brave.Load() != 0 || wiki.Load() != 0 {
		t.Fatal("disabled provider served cached data or disclosed to alternate")
	}
}

func TestSearchTransportRedirectCannotExposeCredential(t *testing.T) {
	var leaked atomic.Int32
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { leaked.Add(1); w.WriteHeader(200) }))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) }))
	defer source.Close()
	cfg := &config.Config{}
	cfg.Search.Providers = map[string]config.SearchProviderConfig{"brave": {APIURL: source.URL, APIKey: "synthetic-secret-fixture"}}
	if _, err := searchBrave(cfg, "fixture", 1); err == nil {
		t.Fatal("cross-origin redirect accepted")
	}
	if leaked.Load() != 0 {
		t.Fatal("search credential escaped configured origin")
	}
}
