package routes

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/focusapacket"
	"github.com/WPUIAI/uiai-engine/internal/secrettext"
	"github.com/go-chi/chi/v5"
)

const defaultSearchLimit = 5
const maxSearchLimit = 20
const defaultSearchCacheTTLSeconds = 3600
const maxSearchCacheEntries = 512
const maxSearchTitleChars = 200
const maxSearchDescriptionChars = 500
const maxSearchSourceChars = 120
const maxSearchAgeChars = 80

type searchRequest struct {
	Query    string `json:"query"`
	Provider string `json:"provider"`
	Limit    int    `json:"limit"`
}

type searchResult struct {
	Title       string   `json:"title"`
	URL         string   `json:"url"`
	Description string   `json:"description,omitempty"`
	Snippet     string   `json:"snippet"`
	Provider    string   `json:"provider"`
	Score       *float64 `json:"score,omitempty"`
	PublishedAt string   `json:"published_at,omitempty"`
	Source      string   `json:"source,omitempty"`
	Age         string   `json:"age,omitempty"`
	Rank        int      `json:"rank,omitempty"`
	EvidenceRef string   `json:"evidence_ref,omitempty"`
}

type searchResponse struct {
	Schema          string               `json:"schema"`
	Provider        string               `json:"provider"`
	Query           string               `json:"query"`
	Count           int                  `json:"count"`
	Cached          bool                 `json:"cached"`
	CacheTTLSeconds int                  `json:"cache_ttl_seconds"`
	Results         []searchResult       `json:"results"`
	Next            []string             `json:"next"`
	Focusa          searchFocusaMetadata `json:"focusa"`
}

type searchFocusaMetadata struct {
	TargetRef         string   `json:"target_ref"`
	EvidenceRef       string   `json:"evidence_ref,omitempty"`
	PreferredTool     string   `json:"preferred_tool"`
	Summary           string   `json:"summary"`
	NextTools         []string `json:"next_tools"`
	FocusaScopeStatus string   `json:"focusa_scope_status"`
}

type searchCacheEntry struct {
	Results   []searchResult
	ExpiresAt time.Time
}

var searchCache = struct {
	sync.Mutex
	entries map[string]searchCacheEntry
}{entries: map[string]searchCacheEntry{}}

type braveWebResponse struct {
	Web struct {
		Results []struct {
			Title       string `json:"title"`
			URL         string `json:"url"`
			Description string `json:"description"`
			Profile     struct {
				Name string `json:"name"`
			} `json:"profile"`
			Age string `json:"age"`
		} `json:"results"`
	} `json:"web"`
}

// MountSearchRoutes exposes provider-neutral web discovery for browser agents.
// Brave is the default provider, but the public contract stays provider-neutral
// so other providers can be added without changing browser/session semantics.
func MountSearchRoutes(r chi.Router, cfg *config.Config) {
	breakers := newSearchBreakers(time.Now)
	bind := func(req *http.Request) *http.Request {
		return req.WithContext(context.WithValue(req.Context(), searchBreakersKey{}, breakers))
	}
	r.Get("/providers", func(w http.ResponseWriter, _ *http.Request) { handleSearchProviders(w, cfg, breakers) })
	r.Get("/", func(w http.ResponseWriter, req *http.Request) { handleSearchGET(w, bind(req), cfg) })
	r.Post("/", func(w http.ResponseWriter, req *http.Request) { handleSearchPOST(w, bind(req), cfg) })
}

func handleSearchGET(w http.ResponseWriter, r *http.Request, cfg *config.Config) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	req := searchRequest{
		Query:    r.URL.Query().Get("q"),
		Provider: r.URL.Query().Get("provider"),
		Limit:    limit,
	}
	runSearch(w, r, cfg, req)
}

func handleSearchPOST(w http.ResponseWriter, r *http.Request, cfg *config.Config) {
	var req searchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json", "message": err.Error()})
		return
	}
	runSearch(w, r, cfg, req)
}

func runSearch(w http.ResponseWriter, httpReq *http.Request, cfg *config.Config, req searchRequest) {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "query_required", "message": "query or q is required"})
		return
	}
	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	if provider == "" {
		provider = resolveDefaultSearchProvider(cfg)
	}
	if provider == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"error":   "no_search_provider_available",
			"message": "no search provider can serve: configure search.providers.<id>.api_key, api_key_file, or api_key_env in the engine config",
		})
		return
	}
	limit := normalizeSearchLimit(req.Limit)

	ttl := searchCacheTTL()
	var results []searchResult
	var err error
	for _, candidate := range searchProviderPlan(cfg, req.Provider, provider) {
		if httpReq.Context().Err() != nil {
			err = httpReq.Context().Err()
			break
		}
		provider = candidate
		adapter := searchProviderMeta(provider).search
		if adapter == nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unsupported_provider", "provider": provider, "supported_providers": searchProviderIDs(cfg)})
			return
		}
		if cfg != nil && cfg.Search.Providers[provider].Disabled {
			err = fmt.Errorf("search provider %s is disabled by configuration", provider)
			break
		}
		if ttl > 0 {
			if cachedResults, ok := getCachedSearch(provider, query, limit, time.Now()); ok {
				writeSearchResponse(w, httpReq, cfg, provider, query, cachedResults, true, ttl)
				return
			}
		}
		breakers := searchBreakersFor(httpReq)
		attempt, wait := breakers.acquire(provider)
		if wait > 0 {
			err = &searchCircuitOpenError{retryAfter: wait}
			continue
		}
		results, err = adapter(httpReq.Context(), cfg, query, limit)
		if httpReq.Context().Err() != nil {
			err = httpReq.Context().Err()
		}
		breakers.complete(provider, attempt, err)
		if err == nil {
			break
		}
	}
	var circuit *searchCircuitOpenError
	if errors.As(err, &circuit) {
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Retry-After", strconv.Itoa(circuit.retryAfter))
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "search_provider_circuit_open", "provider": provider, "message": circuit.Error(), "retryable": true})
		return
	}
	var use *upstreamSearchError
	if errors.As(err, &use) {
		w.Header().Set("Cache-Control", "no-store")
		if use.retryAfter != "" {
			w.Header().Set("Retry-After", use.retryAfter)
		}
		code := http.StatusBadGateway
		errCode := "search_provider_error"
		if use.status == http.StatusTooManyRequests {
			code = http.StatusTooManyRequests
			errCode = "search_provider_limited"
		}
		writeJSON(w, code, map[string]any{
			"error": errCode, "provider": provider, "message": use.message,
			"upstream_status": use.status, "retryable": retryableSearchStatus(use.status),
		})
		return
	}
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "search_provider_unavailable", "provider": provider, "message": err.Error()})
		return
	}
	annotateSearchEvidence(results, provider, query)
	if ttl > 0 {
		setCachedSearch(provider, query, limit, results, time.Now().Add(ttl))
	}
	writeSearchResponse(w, httpReq, cfg, provider, query, results, false, ttl)
}

func normalizeSearchLimit(limit int) int {
	if limit <= 0 {
		return defaultSearchLimit
	}
	if limit > maxSearchLimit {
		return maxSearchLimit
	}
	return limit
}

func searchPressureSummary(cfg *config.Config) map[string]any {
	searchCache.Lock()
	pruneExpiredSearchCacheLocked(time.Now())
	entries := len(searchCache.entries)
	searchCache.Unlock()

	ids := searchProviderIDs(cfg)
	ready := make([]string, 0, len(ids))
	for _, id := range ids {
		if configured, _ := searchProviderConfigured(cfg, id); configured {
			ready = append(ready, id)
		}
	}
	providersStatus := "ready"
	pressure := "normal"
	if len(ready) == 0 {
		providersStatus = "degraded"
		pressure = "degraded"
	}
	defaultProvider := resolveDefaultSearchProvider(cfg)
	return map[string]any{
		"default_provider":    defaultProvider,
		"provider":            defaultProvider,
		"provider_status":     providersStatus,
		"pressure":            pressure,
		"available_providers": ids,
		"ready_providers":     ready,
		"cache_entries":       entries,
		"cache_max_entries":   maxSearchCacheEntries,
		"cache_ttl_seconds":   int(searchCacheTTL() / time.Second),
		"packet_surface":      "search",
	}
}

func searchCacheTTL() time.Duration {
	raw := strings.TrimSpace(os.Getenv("UIAI_SEARCH_CACHE_TTL_SECONDS"))
	if raw == "" {
		return time.Duration(defaultSearchCacheTTLSeconds) * time.Second
	}
	seconds, err := strconv.Atoi(raw)
	if err != nil || seconds < 0 {
		return time.Duration(defaultSearchCacheTTLSeconds) * time.Second
	}
	return time.Duration(seconds) * time.Second
}

func writeSearchResponse(w http.ResponseWriter, req *http.Request, cfg *config.Config, provider, query string, results []searchResult, cached bool, ttl time.Duration) {
	response := searchResponse{
		Schema:          "uiai.search_results.v1",
		Provider:        provider,
		Query:           query,
		Count:           len(results),
		Cached:          cached,
		CacheTTLSeconds: int(ttl / time.Second),
		Results:         results,
		Next:            []string{"browser_open a selected result URL", "browser_read page text", "browser_diagnostics on navigation failure", "cite selected result with evidence_ref"},
		Focusa:          buildSearchFocusaMetadata(provider, query, results),
	}
	writeJSONArtifactEPWA(w, req, cfg, evidenceScopeFromRequest(req), "", "Web search result packet", "research_packet", response, http.StatusOK)
}

func buildSearchFocusaMetadata(provider, query string, results []searchResult) searchFocusaMetadata {
	evidenceRef := ""
	if len(results) > 0 {
		evidenceRef = results[0].EvidenceRef
	}
	return searchFocusaMetadata{
		TargetRef:         fmt.Sprintf("search:%s:%s", provider, searchQueryHash(query)),
		EvidenceRef:       evidenceRef,
		PreferredTool:     "focusa_evidence_capture",
		Summary:           focusapacket.Truncate(fmt.Sprintf("Search %q via %s returned %d result(s); cite selected result evidence_ref instead of raw SERP.", query, provider, len(results)), focusapacket.MaxCaptureSummaryChars),
		NextTools:         []string{"focusa_evidence_capture", "focusa_active_object_resolve", "focusa_predict_record"},
		FocusaScopeStatus: string(focusapacket.ScopeMissing),
	}
}

func searchCacheKey(provider, query string, limit int) string {
	return fmt.Sprintf("%s:%s:%d", provider, searchQueryHash(query), limit)
}

func cloneSearchResults(results []searchResult) []searchResult {
	cloned := make([]searchResult, len(results))
	copy(cloned, results)
	for i := range cloned {
		if results[i].Score != nil {
			score := *results[i].Score
			cloned[i].Score = &score
		}
	}
	return cloned
}

func getCachedSearch(provider, query string, limit int, now time.Time) ([]searchResult, bool) {
	key := searchCacheKey(provider, query, limit)
	searchCache.Lock()
	defer searchCache.Unlock()
	entry, ok := searchCache.entries[key]
	if !ok {
		return nil, false
	}
	if !now.Before(entry.ExpiresAt) {
		delete(searchCache.entries, key)
		return nil, false
	}
	return cloneSearchResults(entry.Results), true
}

func setCachedSearch(provider, query string, limit int, results []searchResult, expiresAt time.Time) {
	key := searchCacheKey(provider, query, limit)
	searchCache.Lock()
	defer searchCache.Unlock()
	pruneExpiredSearchCacheLocked(time.Now())
	if _, exists := searchCache.entries[key]; !exists && len(searchCache.entries) >= maxSearchCacheEntries {
		oldest := ""
		var expiry time.Time
		for candidate, entry := range searchCache.entries {
			if oldest == "" || entry.ExpiresAt.Before(expiry) || (entry.ExpiresAt.Equal(expiry) && candidate < oldest) {
				oldest = candidate
				expiry = entry.ExpiresAt
			}
		}
		delete(searchCache.entries, oldest)
	}
	searchCache.entries[key] = searchCacheEntry{Results: cloneSearchResults(results), ExpiresAt: expiresAt}
}

// Caller holds searchCache's lock; expiry has one owner for writes and health.
func pruneExpiredSearchCacheLocked(now time.Time) {
	for key, entry := range searchCache.entries {
		if !now.Before(entry.ExpiresAt) {
			delete(searchCache.entries, key)
		}
	}
}

func truncateSearchField(value string, maxChars int) string {
	value = strings.TrimSpace(value)
	if maxChars <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxChars {
		return value
	}
	if maxChars <= 1 {
		return string(runes[:maxChars])
	}
	return string(runes[:maxChars-1]) + "…"
}

func isSecretQueryKey(key string) bool {
	key = strings.ToLower(key)
	secretParts := []string{"key", "token", "secret", "password", "passwd", "auth", "signature", "sig", "credential", "session", "api_key", "apikey", "access_token", "refresh_token"}
	for _, part := range secretParts {
		if strings.Contains(key, part) {
			return true
		}
	}
	return false
}

func sanitizeSearchURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return truncateSearchField(raw, 2048)
	}
	u.Fragment = ""
	q := u.Query()
	for key := range q {
		if isSecretQueryKey(key) {
			q.Set(key, "REDACTED")
		}
	}
	u.RawQuery = q.Encode()
	return truncateSearchField(u.String(), 2048)
}

func sanitizeSearchResult(result searchResult) searchResult {
	description := result.Description
	if description == "" {
		description = result.Snippet
	}
	description = truncateSearchField(secrettext.Text(description), maxSearchDescriptionChars)
	published := result.PublishedAt
	if published == "" {
		published = result.Age
	}
	published = searchPublishedAt(published)
	var score *float64
	if result.Score != nil && !math.IsNaN(*result.Score) && !math.IsInf(*result.Score, 0) {
		copy := *result.Score
		score = &copy
	}
	return searchResult{
		Title:       truncateSearchField(secrettext.Text(result.Title), maxSearchTitleChars),
		URL:         sanitizeSearchURL(result.URL),
		Description: description, Snippet: description, Provider: truncateSearchField(secrettext.Text(result.Provider), maxSearchSourceChars), Score: score, PublishedAt: published,
		Source:      truncateSearchField(secrettext.Text(result.Source), maxSearchSourceChars),
		Age:         truncateSearchField(secrettext.Text(result.Age), maxSearchAgeChars),
		Rank:        result.Rank,
		EvidenceRef: result.EvidenceRef,
	}
}

func searchQueryHash(query string) string {
	normalized := strings.ToLower(strings.Join(strings.Fields(query), " "))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])[:16]
}

func searchEvidenceRef(provider, query string, rank int) string {
	return fmt.Sprintf("uiai-search:%s:%s:%d", provider, searchQueryHash(query), rank)
}

func annotateSearchEvidence(results []searchResult, provider, query string) {
	for i := range results {
		rank := i + 1
		results[i].Rank = rank
		results[i].Provider = provider
		results[i].EvidenceRef = searchEvidenceRef(provider, query, rank)
	}
}

// Relative ages remain in the legacy age field; no date or timezone is invented.
func searchPublishedAt(raw string) string {
	if parsed, err := time.Parse(time.RFC3339Nano, raw); err == nil {
		return parsed.Format(time.RFC3339Nano)
	}
	if parsed, err := time.Parse("2006-01-02", raw); err == nil {
		return parsed.Format("2006-01-02")
	}
	return ""
}
