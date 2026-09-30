package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

func TestNormalizeSearchLimit(t *testing.T) {
	if got := normalizeSearchLimit(0); got != defaultSearchLimit {
		t.Fatalf("default limit = %d, want %d", got, defaultSearchLimit)
	}
	if got := normalizeSearchLimit(99); got != maxSearchLimit {
		t.Fatalf("max limit = %d, want %d", got, maxSearchLimit)
	}
	if got := normalizeSearchLimit(3); got != 3 {
		t.Fatalf("explicit limit = %d, want 3", got)
	}
}

func TestSanitizeSearchResultBoundsFieldsAndRedactsURLSecrets(t *testing.T) {
	result := sanitizeSearchResult(searchResult{
		Title:       strings.Repeat("t", maxSearchTitleChars+10),
		URL:         "https://example.test/path?api_key=secret&ok=1&token=abc#fragment",
		Description: strings.Repeat("d", maxSearchDescriptionChars+10),
		Source:      strings.Repeat("s", maxSearchSourceChars+10),
		Age:         strings.Repeat("a", maxSearchAgeChars+10),
	})
	if len([]rune(result.Title)) != maxSearchTitleChars || !strings.HasSuffix(result.Title, "…") {
		t.Fatalf("title not bounded: len=%d value=%q", len([]rune(result.Title)), result.Title)
	}
	if len([]rune(result.Description)) != maxSearchDescriptionChars || !strings.HasSuffix(result.Description, "…") {
		t.Fatalf("description not bounded: len=%d", len([]rune(result.Description)))
	}
	if strings.Contains(result.URL, "secret") || strings.Contains(result.URL, "abc") || strings.Contains(result.URL, "fragment") {
		t.Fatalf("url secret/fragment not redacted: %s", result.URL)
	}
	if !strings.Contains(result.URL, "api_key=REDACTED") || !strings.Contains(result.URL, "token=REDACTED") || !strings.Contains(result.URL, "ok=1") {
		t.Fatalf("url query not preserved/redacted as expected: %s", result.URL)
	}
}

func TestSearchCacheTTLDefaultsAndEnv(t *testing.T) {
	t.Setenv("UIAI_SEARCH_CACHE_TTL_SECONDS", "")
	if got := searchCacheTTL(); got != time.Duration(defaultSearchCacheTTLSeconds)*time.Second {
		t.Fatalf("default cache ttl = %s", got)
	}
	t.Setenv("UIAI_SEARCH_CACHE_TTL_SECONDS", "0")
	if got := searchCacheTTL(); got != 0 {
		t.Fatalf("disabled cache ttl = %s, want 0", got)
	}
	t.Setenv("UIAI_SEARCH_CACHE_TTL_SECONDS", "7")
	if got := searchCacheTTL(); got != 7*time.Second {
		t.Fatalf("explicit cache ttl = %s, want 7s", got)
	}
}

func TestSearchCacheStoresClonesAndExpires(t *testing.T) {
	searchCache.Lock()
	searchCache.entries = map[string]searchCacheEntry{}
	searchCache.Unlock()

	now := time.Unix(100, 0)
	setCachedSearch("brave", "agent browser", 2, []searchResult{{Title: "One"}}, now.Add(time.Minute))
	got, ok := getCachedSearch("brave", "agent browser", 2, now)
	if !ok || len(got) != 1 || got[0].Title != "One" {
		t.Fatalf("expected cached result, got ok=%v results=%+v", ok, got)
	}
	got[0].Title = "mutated"
	again, ok := getCachedSearch("brave", "agent browser", 2, now)
	if !ok || again[0].Title != "One" {
		t.Fatalf("cache should return clones, got ok=%v results=%+v", ok, again)
	}
	if _, ok := getCachedSearch("brave", "agent browser", 2, now.Add(2*time.Minute)); ok {
		t.Fatal("expected expired cache miss")
	}
}

func TestSearchEvidenceRefsAreStableAndRanked(t *testing.T) {
	ref1 := searchEvidenceRef("brave", "  Agent   Browser  ", 1)
	ref2 := searchEvidenceRef("brave", "agent browser", 1)
	if ref1 != ref2 {
		t.Fatalf("refs should normalize query whitespace/case: %s %s", ref1, ref2)
	}
	if !strings.HasPrefix(ref1, "uiai-search:brave:") || !strings.HasSuffix(ref1, ":1") {
		t.Fatalf("unexpected evidence ref: %s", ref1)
	}

	results := []searchResult{{Title: "One"}, {Title: "Two"}}
	annotateSearchEvidence(results, "brave", "agent browser")
	if results[0].Rank != 1 || results[1].Rank != 2 {
		t.Fatalf("unexpected ranks: %+v", results)
	}
	if results[0].EvidenceRef == results[1].EvidenceRef || !strings.HasSuffix(results[1].EvidenceRef, ":2") {
		t.Fatalf("unexpected refs: %+v", results)
	}
}

func TestSearchWikipediaMapsOpenSearchResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("action") != "opensearch" || r.URL.Query().Get("search") != "agent browser" || r.URL.Query().Get("limit") != "2" {
			t.Fatalf("bad wikipedia query: %s", r.URL.RawQuery)
		}
		if !strings.Contains(r.Header.Get("User-Agent"), "uiai-engine") {
			t.Fatalf("missing user-agent: %q", r.Header.Get("User-Agent"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`["agent browser",["Agent Browser","Browser agent"],["first desc","second desc"],["https://en.wikipedia.org/wiki/Agent?token=secret#frag","https://en.wikipedia.org/wiki/Browser_agent"]]`))
	}))
	defer server.Close()
	t.Setenv("UIAI_WIKIPEDIA_SEARCH_API_URL", server.URL)

	results, err := searchWikipedia("agent browser", 2)
	if err != nil {
		t.Fatalf("searchWikipedia error: %v", err)
	}
	if len(results) != 2 || results[0].Title != "Agent Browser" || results[0].Source != "Wikipedia" {
		t.Fatalf("unexpected wikipedia results: %+v", results)
	}
	if strings.Contains(results[0].URL, "secret") || strings.Contains(results[0].URL, "#frag") || !strings.Contains(results[0].URL, "token=REDACTED") {
		t.Fatalf("wikipedia result URL not redacted: %s", results[0].URL)
	}
}

func TestSearchWikipediaRejectsBadPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"bad":true}`))
	}))
	defer server.Close()
	t.Setenv("UIAI_WIKIPEDIA_SEARCH_API_URL", server.URL)
	if _, err := searchWikipedia("agent browser", 2); err == nil {
		t.Fatalf("expected bad payload error")
	}
}

func TestSearchBraveMapsWebResults(t *testing.T) {
	t.Setenv("BRAVE_SEARCH_API_KEY", "test-key")
	var sawToken bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Subscription-Token") == "test-key" {
			sawToken = true
		}
		if r.URL.Query().Get("q") != "agent browser" {
			t.Fatalf("query = %q", r.URL.Query().Get("q"))
		}
		if r.URL.Query().Get("count") != "2" {
			t.Fatalf("count = %q", r.URL.Query().Get("count"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"web":{"results":[{"title":"One","url":"https://example.com/one","description":"first","profile":{"name":"example.com"},"age":"1 day"},{"title":"Two","url":"https://example.com/two","description":"second","profile":{"name":"example.com"}}]}}`))
	}))
	defer server.Close()
	t.Setenv("UIAI_BRAVE_SEARCH_API_URL", server.URL)

	results, err := searchBrave(nil, "agent browser", 2)
	if err != nil {
		t.Fatalf("searchBrave error: %v", err)
	}
	if !sawToken {
		t.Fatalf("Brave auth header not sent")
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d", len(results))
	}
	if results[0].Title != "One" || results[0].URL != "https://example.com/one" || results[0].Source != "example.com" {
		t.Fatalf("unexpected first result: %+v", results[0])
	}
}

func TestSearchProvidersReportsMissingKeyDegraded(t *testing.T) {
	oldKey, hadKey := os.LookupEnv("BRAVE_SEARCH_API_KEY")
	_ = os.Unsetenv("BRAVE_SEARCH_API_KEY")
	defer func() {
		if hadKey {
			_ = os.Setenv("BRAVE_SEARCH_API_KEY", oldKey)
		}
	}()

	res := httptest.NewRecorder()
	handleSearchProviders(res, nil)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d", res.Code)
	}
	var body struct {
		Providers []struct {
			ID             string `json:"id"`
			Configured     bool   `json:"configured"`
			Status         string `json:"status"`
			DegradedReason string `json:"degraded_reason"`
		} `json:"providers"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode providers: %v", err)
	}
	if len(body.Providers) != 2 || body.Providers[0].ID != "brave" || body.Providers[1].ID != "wikipedia" {
		t.Fatalf("unexpected providers: %+v", body.Providers)
	}
	brave := body.Providers[0]
	if brave.Configured || brave.Status != "degraded" || brave.DegradedReason != "missing_key" {
		t.Fatalf("expected missing-key degraded Brave provider, got %+v", brave)
	}
}

func TestSearchBraveRequiresKey(t *testing.T) {
	oldKey, hadKey := os.LookupEnv("BRAVE_SEARCH_API_KEY")
	oldURL, hadURL := os.LookupEnv("UIAI_BRAVE_SEARCH_API_URL")
	_ = os.Unsetenv("BRAVE_SEARCH_API_KEY")
	_ = os.Unsetenv("UIAI_BRAVE_SEARCH_API_URL")
	defer func() {
		if hadKey {
			_ = os.Setenv("BRAVE_SEARCH_API_KEY", oldKey)
		}
		if hadURL {
			_ = os.Setenv("UIAI_BRAVE_SEARCH_API_URL", oldURL)
		}
	}()
	if _, err := searchBrave(nil, "missing key", 1); err == nil {
		t.Fatalf("expected missing key error")
	}
}

func TestWriteSearchResponseIncludesFocusaMetadata(t *testing.T) {
	results := []searchResult{{
		Title:       "One",
		URL:         "https://example.com/one?token=secret#frag",
		Description: "first",
	}}
	annotateSearchEvidence(results, "brave", "agent browser")
	results[0] = sanitizeSearchResult(results[0])

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "https://evidence.example/api/search?q=agent+browser", nil)
	setCompleteEvidenceScopeHeaders(req, completeEvidenceScope())
	writeSearchResponse(res, req, &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}, "brave", "agent browser", results, false, time.Minute)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d", res.Code)
	}
	var body struct {
		Focusa struct {
			TargetRef         string   `json:"target_ref"`
			EvidenceRef       string   `json:"evidence_ref"`
			PreferredTool     string   `json:"preferred_tool"`
			Summary           string   `json:"summary"`
			NextTools         []string `json:"next_tools"`
			FocusaScopeStatus string   `json:"focusa_scope_status"`
		} `json:"focusa"`
		Results []searchResult `json:"results"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode search response: %v", err)
	}
	if body.Focusa.TargetRef == "" || !strings.HasPrefix(body.Focusa.TargetRef, "search:brave:") {
		t.Fatalf("missing target_ref: %+v", body.Focusa)
	}
	if body.Focusa.EvidenceRef != results[0].EvidenceRef {
		t.Fatalf("evidence_ref = %q, want %q", body.Focusa.EvidenceRef, results[0].EvidenceRef)
	}
	if body.Focusa.PreferredTool != "focusa_evidence_capture" || body.Focusa.FocusaScopeStatus != "missing" {
		t.Fatalf("unexpected focusa metadata: %+v", body.Focusa)
	}
	if len(body.Focusa.NextTools) == 0 || body.Focusa.Summary == "" {
		t.Fatalf("incomplete focusa metadata: %+v", body.Focusa)
	}
	encoded := res.Body.String()
	if strings.Contains(encoded, "secret") || strings.Contains(encoded, "#frag") {
		t.Fatalf("search response leaked secret/fragment: %s", encoded)
	}
}

// TestSearchProviderConfiguredFromConfigFileCredential proves a provider credential
// can be supplied from an operator-owned file with no vendor-specific environment
// variable and no bespoke code path.
func TestSearchProviderConfiguredFromConfigFileCredential(t *testing.T) {
	dir := t.TempDir()
	keyFile := filepath.Join(dir, "brave.key")
	if err := os.WriteFile(keyFile, []byte("  file-supplied-key\n"), 0o600); err != nil {
		t.Fatalf("write key file: %v", err)
	}
	t.Setenv("BRAVE_SEARCH_API_KEY", "")
	t.Setenv("BRAVE_SEARCH_API_KEY_ALT", "")

	cfg := &config.Config{}
	cfg.Search.Providers = map[string]config.SearchProviderConfig{
		"brave": {APIKeyFile: keyFile},
	}

	key, err := cfg.Search.ResolveSearchProviderKey("brave")
	if err != nil {
		t.Fatalf("ResolveSearchProviderKey error: %v", err)
	}
	if key != "file-supplied-key" {
		t.Fatalf("key = %q, want trimmed file contents", key)
	}
	if !cfg.Search.SearchProviderEnabled("brave") {
		t.Fatalf("expected provider enabled from file credential")
	}
}

// TestSearchProviderCredentialFromOperatorEnvName proves the credential location is
// operator-chosen: any variable name works, not only the vendor's convention.
func TestSearchProviderCredentialFromOperatorEnvName(t *testing.T) {
	t.Setenv("BRAVE_SEARCH_API_KEY", "")
	t.Setenv("ACME_SEARCH_TOKEN", "operator-named-secret")

	cfg := &config.Config{}
	cfg.Search.Providers = map[string]config.SearchProviderConfig{
		"brave": {APIKeyEnv: "ACME_SEARCH_TOKEN"},
	}

	key, err := cfg.Search.ResolveSearchProviderKey("brave")
	if err != nil {
		t.Fatalf("ResolveSearchProviderKey error: %v", err)
	}
	if key != "operator-named-secret" {
		t.Fatalf("key = %q, want value from operator-selected variable", key)
	}
}

// TestSearchProviderDisabledByConfig proves an operator can retire a provider
// without deleting its configuration.
func TestSearchProviderDisabledByConfig(t *testing.T) {
	cfg := &config.Config{}
	cfg.Search.Providers = map[string]config.SearchProviderConfig{
		"brave": {APIKeyEnv: "BRAVE_SEARCH_API_KEY", Disabled: true},
	}
	if cfg.Search.SearchProviderEnabled("brave") {
		t.Fatalf("expected disabled provider to report not enabled")
	}
}

// TestSearchProvidersReportsConfigOnlyProvider proves an operator-configured
// provider is surfaced honestly in the providers projection.
func TestSearchProvidersReportsConfigOnlyProvider(t *testing.T) {
	t.Setenv("BRAVE_SEARCH_API_KEY", "")
	cfg := &config.Config{}
	cfg.Search.Providers = map[string]config.SearchProviderConfig{
		"brave": {APIKeyEnv: "MISSING_BRAVE_KEY"},
		"acme":  {APIKeyEnv: "ACME_SEARCH_TOKEN"},
	}
	t.Setenv("ACME_SEARCH_TOKEN", "acme-secret")

	res := httptest.NewRecorder()
	handleSearchProviders(res, cfg)
	if res.Code != http.StatusOK {
		t.Fatalf("status = %d", res.Code)
	}
	var body struct {
		DefaultProvider    string   `json:"default_provider"`
		AvailableProviders []string `json:"available_providers"`
		ReadyProviders     []string `json:"ready_providers"`
		Providers          []struct {
			ID             string `json:"id"`
			Configured     bool   `json:"configured"`
			Status         string `json:"status"`
			DegradedReason string `json:"degraded_reason"`
		} `json:"providers"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode providers: %v", err)
	}

	status := map[string]string{}
	for _, provider := range body.Providers {
		status[provider.ID] = provider.Status
	}
	if status["acme"] != "ready" {
		t.Fatalf("acme status = %q, want ready; all=%v", status["acme"], body.Providers)
	}
	if status["brave"] != "degraded" {
		t.Fatalf("brave status = %q, want degraded", status["brave"])
	}
	if body.DefaultProvider != "acme" {
		t.Fatalf("default_provider = %q, want the only genuinely configured provider", body.DefaultProvider)
	}
	if len(body.ReadyProviders) != 2 { // acme + keyless wikipedia
		t.Fatalf("ready_providers = %v, want acme and wikipedia", body.ReadyProviders)
	}
}

// TestSearchPressureSummaryDoesNotClaimReadyProviders proves the health surface
// derives readiness from configuration instead of hardcoded literals.
func TestSearchPressureSummaryDoesNotClaimReadyProviders(t *testing.T) {
	t.Setenv("BRAVE_SEARCH_API_KEY", "")
	cfg := &config.Config{}

	summary := searchPressureSummary(cfg)
	ready, _ := summary["ready_providers"].([]string)
	if len(ready) != 1 || ready[0] != "wikipedia" {
		t.Fatalf("ready_providers = %v, want only the keyless provider", ready)
	}
	if summary["pressure"] != "normal" {
		t.Fatalf("pressure = %v, want normal while a keyless provider can serve", summary["pressure"])
	}

	cfg.Search.Providers = map[string]config.SearchProviderConfig{"wikipedia": {Disabled: true}}
	summary = searchPressureSummary(cfg)
	if summary["pressure"] != "degraded" {
		t.Fatalf("pressure = %v, want degraded when no provider can serve", summary["pressure"])
	}
	if summary["default_provider"] != "" {
		t.Fatalf("default_provider = %v, want empty when nothing can serve", summary["default_provider"])
	}
}

// TestSearchBraveErrorNamesConfigSurface proves the failure message points an
// operator at portable configuration instead of only a vendor variable name.
func TestSearchBraveErrorNamesConfigSurface(t *testing.T) {
	t.Setenv("BRAVE_SEARCH_API_KEY", "")
	cfg := &config.Config{}

	_, err := searchBrave(cfg, "query", 1)
	if err == nil {
		t.Fatalf("expected missing credential error")
	}
	for _, want := range []string{"search.providers.brave.api_key", "api_key_file", "api_key_env"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("error %q does not name config surface %q", err.Error(), want)
		}
	}
}

// TestZeroKeyBoxServesKeylessDefault proves a deployment with no credentials at
// all still answers searches through the keyless provider and reports that
// provider honestly, instead of 503ing on a hardcoded default.
func TestZeroKeyBoxServesKeylessDefault(t *testing.T) {
	t.Setenv("BRAVE_SEARCH_API_KEY", "")

	if got := resolveDefaultSearchProvider(nil); got != "wikipedia" {
		t.Fatalf("default = %q, want the keyless provider", got)
	}
	if got := resolveDefaultSearchProvider(&config.Config{}); got != "wikipedia" {
		t.Fatalf("default = %q, want the keyless provider", got)
	}
}
