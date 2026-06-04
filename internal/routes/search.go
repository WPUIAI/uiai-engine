package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/go-chi/chi/v5"
)

const defaultSearchLimit = 5
const maxSearchLimit = 20
const defaultSearchCacheTTLSeconds = 60

type searchRequest struct {
	Query    string `json:"query"`
	Provider string `json:"provider"`
	Limit    int    `json:"limit"`
}

type searchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source,omitempty"`
	Age         string `json:"age,omitempty"`
	Rank        int    `json:"rank,omitempty"`
	EvidenceRef string `json:"evidence_ref,omitempty"`
}

type searchResponse struct {
	Schema          string         `json:"schema"`
	Provider        string         `json:"provider"`
	Query           string         `json:"query"`
	Count           int            `json:"count"`
	Cached          bool           `json:"cached"`
	CacheTTLSeconds int            `json:"cache_ttl_seconds"`
	Results         []searchResult `json:"results"`
	Next            []string       `json:"next"`
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
func MountSearchRoutes(r chi.Router, _ *config.Config) {
	r.Get("/providers", handleSearchProviders)
	r.Get("/", handleSearchGET)
	r.Post("/", handleSearchPOST)
}

func handleSearchProviders(w http.ResponseWriter, _ *http.Request) {
	braveConfigured := strings.TrimSpace(os.Getenv("BRAVE_SEARCH_API_KEY")) != ""
	braveStatus := "ready"
	braveReason := ""
	if !braveConfigured {
		braveStatus = "degraded"
		braveReason = "missing_key"
	}

	writeJSON(w, 200, map[string]any{
		"schema":           "uiai.search_providers.v1",
		"default_provider": "brave",
		"providers": []map[string]any{
			{
				"id":                "brave",
				"name":              "Brave Search",
				"configured":        braveConfigured,
				"status":            braveStatus,
				"degraded_reason":   braveReason,
				"cache_ttl_seconds": int(searchCacheTTL() / time.Second),
				"capabilities":      []string{"web_search", "source_urls", "snippets"},
			},
		},
	})
}

func handleSearchGET(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	req := searchRequest{
		Query:    r.URL.Query().Get("q"),
		Provider: r.URL.Query().Get("provider"),
		Limit:    limit,
	}
	runSearch(w, req)
}

func handleSearchPOST(w http.ResponseWriter, r *http.Request) {
	var req searchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json", "message": err.Error()})
		return
	}
	runSearch(w, req)
}

func runSearch(w http.ResponseWriter, req searchRequest) {
	query := strings.TrimSpace(req.Query)
	if query == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "query_required", "message": "query or q is required"})
		return
	}
	provider := strings.ToLower(strings.TrimSpace(req.Provider))
	if provider == "" {
		provider = "brave"
	}
	limit := normalizeSearchLimit(req.Limit)

	ttl := searchCacheTTL()
	if ttl > 0 {
		if cachedResults, ok := getCachedSearch(provider, query, limit, time.Now()); ok {
			writeSearchResponse(w, provider, query, cachedResults, true, ttl)
			return
		}
	}

	var results []searchResult
	var err error
	switch provider {
	case "brave":
		results, err = searchBrave(query, limit)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unsupported_provider", "provider": provider, "supported_providers": []string{"brave"}})
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
	writeSearchResponse(w, provider, query, results, false, ttl)
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

func writeSearchResponse(w http.ResponseWriter, provider, query string, results []searchResult, cached bool, ttl time.Duration) {
	writeJSON(w, 200, searchResponse{
		Schema:          "uiai.search_results.v1",
		Provider:        provider,
		Query:           query,
		Count:           len(results),
		Cached:          cached,
		CacheTTLSeconds: int(ttl / time.Second),
		Results:         results,
		Next:            []string{"browser_open a selected result URL", "browser_read page text", "browser_diagnostics on navigation failure", "cite selected result with evidence_ref"},
	})
}

func searchCacheKey(provider, query string, limit int) string {
	return fmt.Sprintf("%s:%s:%d", provider, searchQueryHash(query), limit)
}

func cloneSearchResults(results []searchResult) []searchResult {
	cloned := make([]searchResult, len(results))
	copy(cloned, results)
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
	searchCache.entries[key] = searchCacheEntry{Results: cloneSearchResults(results), ExpiresAt: expiresAt}
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
		results[i].EvidenceRef = searchEvidenceRef(provider, query, rank)
	}
}

func searchBrave(query string, limit int) ([]searchResult, error) {
	key := strings.TrimSpace(os.Getenv("BRAVE_SEARCH_API_KEY"))
	if key == "" {
		return nil, fmt.Errorf("BRAVE_SEARCH_API_KEY is not configured")
	}
	base := strings.TrimSpace(os.Getenv("UIAI_BRAVE_SEARCH_API_URL"))
	if base == "" {
		base = "https://api.search.brave.com/res/v1/web/search"
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("invalid Brave API URL: %w", err)
	}
	q := u.Query()
	q.Set("q", query)
	q.Set("count", strconv.Itoa(limit))
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("X-Subscription-Token", key)
	httpReq.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Brave API returned HTTP %d", resp.StatusCode)
	}

	var decoded braveWebResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	results := make([]searchResult, 0, len(decoded.Web.Results))
	for _, item := range decoded.Web.Results {
		if strings.TrimSpace(item.URL) == "" {
			continue
		}
		results = append(results, searchResult{
			Title:       item.Title,
			URL:         item.URL,
			Description: item.Description,
			Source:      item.Profile.Name,
			Age:         item.Age,
		})
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}
