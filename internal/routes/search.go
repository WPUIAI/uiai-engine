package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/focusapacket"
	"github.com/go-chi/chi/v5"
)

const defaultSearchLimit = 5
const maxSearchLimit = 20
const defaultSearchCacheTTLSeconds = 60
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
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
	Source      string `json:"source,omitempty"`
	Age         string `json:"age,omitempty"`
	Rank        int    `json:"rank,omitempty"`
	EvidenceRef string `json:"evidence_ref,omitempty"`
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
	r.Get("/providers", func(w http.ResponseWriter, _ *http.Request) { handleSearchProviders(w, cfg) })
	r.Get("/", func(w http.ResponseWriter, req *http.Request) { handleSearchGET(w, req, cfg) })
	r.Post("/", func(w http.ResponseWriter, req *http.Request) { handleSearchPOST(w, req, cfg) })
}

// builtinSearchProvider describes a search provider the engine ships with.
type builtinSearchProvider struct {
	id           string
	name         string
	keyless      bool
	capabilities []string
}

var builtinSearchProviders = []builtinSearchProvider{
	{id: "brave", name: "Brave Search", capabilities: []string{"web_search", "source_urls", "snippets"}},
	{id: "wikipedia", name: "Wikipedia OpenSearch", keyless: true, capabilities: []string{"encyclopedia_search", "source_urls", "snippets", "keyless_public"}},
}

// searchProviderIDs returns every provider this deployment can serve: the built-in
// set plus any provider the operator added under search.providers. The list is
// derived from configuration rather than hardcoded so a new provider needs config
// only, never a bespoke code path.
func searchProviderIDs(cfg *config.Config) []string {
	seen := map[string]bool{}
	ids := make([]string, 0, len(builtinSearchProviders))
	add := func(id string) {
		if id == "" || seen[id] {
			return
		}
		seen[id] = true
		ids = append(ids, id)
	}
	for _, provider := range builtinSearchProviders {
		add(provider.id)
	}
	if cfg != nil {
		for id := range cfg.Search.Providers {
			add(strings.TrimSpace(id))
		}
	}
	sort.Strings(ids)
	return ids
}

// searchProviderMeta returns the declared name and capabilities for a provider.
func searchProviderMeta(id string) builtinSearchProvider {
	for _, provider := range builtinSearchProviders {
		if provider.id == id {
			return provider
		}
	}
	return builtinSearchProvider{id: id, name: id}
}

// searchProviderConfigured reports whether a provider currently has a usable
// credential, honouring explicit configuration first and the provider's
// conventional environment variable only as a fallback.
func searchProviderConfigured(cfg *config.Config, id string) (bool, string) {
	if cfg != nil {
		if provider, declared := cfg.Search.Providers[id]; declared && provider.Disabled {
			// An explicit disable always wins, even for keyless providers: the
			// operator retired this provider and the report must say so.
			return false, "disabled"
		}
	}
	meta := searchProviderMeta(id)
	if meta.keyless {
		return true, ""
	}
	if cfg != nil {
		if _, declared := cfg.Search.Providers[id]; declared {
			if key, err := cfg.Search.ResolveSearchProviderKey(id); err != nil {
				return false, "credential_unreadable"
			} else if key != "" {
				return true, ""
			}
			return false, "missing_key"
		}
	}
	if config.BuiltinSearchProviderKey(id) != "" {
		return true, ""
	}
	return false, "missing_key"
}

// resolveDefaultSearchProvider returns the provider a request should use when it
// names none: the operator's explicit default when it can serve, otherwise the
// first provider that can genuinely serve. Empty means nothing can serve, and the
// caller must say so instead of advertising a provider that would 503.
func resolveDefaultSearchProvider(cfg *config.Config) string {
	ids := searchProviderIDs(cfg)
	if cfg != nil {
		if wanted := strings.TrimSpace(cfg.Search.DefaultProvider); wanted != "" {
			for _, id := range ids {
				if id == wanted {
					if configured, _ := searchProviderConfigured(cfg, id); configured {
						return id
					}
					break
				}
			}
		}
	}
	for _, id := range ids {
		if configured, _ := searchProviderConfigured(cfg, id); configured {
			return id
		}
	}
	return ""
}

func handleSearchProviders(w http.ResponseWriter, cfg *config.Config) {
	ids := searchProviderIDs(cfg)
	providers := make([]map[string]any, 0, len(ids))
	ready := make([]string, 0, len(ids))
	for _, id := range ids {
		meta := searchProviderMeta(id)
		configured, reason := searchProviderConfigured(cfg, id)
		status := "ready"
		degradedReason := ""
		if !configured {
			status = "degraded"
			degradedReason = reason
		} else {
			ready = append(ready, id)
		}
		providers = append(providers, map[string]any{
			"id":                id,
			"name":              meta.name,
			"configured":        configured,
			"status":            status,
			"degraded_reason":   degradedReason,
			"cache_ttl_seconds": int(searchCacheTTL() / time.Second),
			"capabilities":      meta.capabilities,
		})
	}

	defaultProvider := resolveDefaultSearchProvider(cfg)

	writeJSON(w, 200, map[string]any{
		"schema":              "uiai.search_providers.v1",
		"default_provider":    defaultProvider,
		"available_providers": ids,
		"ready_providers":     ready,
		"providers":           providers,
	})
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
	if ttl > 0 {
		if cachedResults, ok := getCachedSearch(provider, query, limit, time.Now()); ok {
			writeSearchResponse(w, httpReq, cfg, provider, query, cachedResults, true, ttl)
			return
		}
	}

	var results []searchResult
	var err error
	switch provider {
	case "brave":
		results, err = searchBrave(cfg, query, limit)
	case "wikipedia":
		results, err = searchWikipedia(query, limit)
	default:
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": "unsupported_provider", "provider": provider, "supported_providers": searchProviderIDs(cfg)})
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
			"upstream_status": use.status, "retryable": true,
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
	return searchResult{
		Title:       truncateSearchField(result.Title, maxSearchTitleChars),
		URL:         sanitizeSearchURL(result.URL),
		Description: truncateSearchField(result.Description, maxSearchDescriptionChars),
		Source:      truncateSearchField(result.Source, maxSearchSourceChars),
		Age:         truncateSearchField(result.Age, maxSearchAgeChars),
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
		results[i].EvidenceRef = searchEvidenceRef(provider, query, rank)
	}
}

func searchWikipedia(query string, limit int) ([]searchResult, error) {
	base := strings.TrimSpace(os.Getenv("UIAI_WIKIPEDIA_SEARCH_API_URL"))
	if base == "" {
		base = "https://en.wikipedia.org/w/api.php"
	}
	u, err := searchAPIURL(base, "https://en.wikipedia.org/w/api.php", []string{"en.wikipedia.org", "wikipedia.org"})
	if err != nil {
		return nil, fmt.Errorf("invalid Wikipedia API URL: %w", err)
	}
	q := u.Query()
	q.Set("action", "opensearch")
	q.Set("format", "json")
	q.Set("namespace", "0")
	q.Set("limit", strconv.Itoa(limit))
	q.Set("search", query)
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequest(http.MethodGet, u.String(), nil) // #nosec G704 -- URL validated by searchAPIURL host allowlist/loopback gate.
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "uiai-engine/agent-search (+https://github.com/WPUIAI/uiai-engine)")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(httpReq) // #nosec G704 -- API URL is restricted by searchAPIURL allowlist/loopback test gate.
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, upstreamErr(resp)
	}

	var decoded []any
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}
	if len(decoded) < 4 {
		return nil, fmt.Errorf("wikipedia API returned unexpected OpenSearch payload")
	}
	titles := stringsFromJSONArray(decoded[1])
	descriptions := stringsFromJSONArray(decoded[2])
	urls := stringsFromJSONArray(decoded[3])
	results := make([]searchResult, 0, len(titles))
	for i, title := range titles {
		if i >= len(urls) || strings.TrimSpace(urls[i]) == "" {
			continue
		}
		desc := ""
		if i < len(descriptions) {
			desc = descriptions[i]
		}
		results = append(results, sanitizeSearchResult(searchResult{
			Title:       title,
			URL:         urls[i],
			Description: desc,
			Source:      "Wikipedia",
		}))
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

func searchAPIURL(raw, fallback string, allowedHosts []string) (*url.URL, error) {
	if strings.TrimSpace(raw) == "" {
		raw = fallback
	}
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if u.Scheme != "https" && !isLoopbackHost(u.Hostname()) {
		return nil, fmt.Errorf("search API URL must use https unless loopback")
	}
	host := strings.ToLower(u.Hostname())
	if isLoopbackHost(host) {
		return u, nil
	}
	for _, allowed := range allowedHosts {
		allowed = strings.ToLower(strings.TrimSpace(allowed))
		if host == allowed || strings.HasSuffix(host, "."+allowed) {
			return u, nil
		}
	}
	return nil, fmt.Errorf("search API host %q is not allowed", host)
}

func isLoopbackHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func stringsFromJSONArray(value any) []string {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// upstreamSearchError preserves the provider's HTTP status so the handler
// can return honest semantics (#64): 429 stays 429 (retryable, with
// Retry-After), never a blanket unrecoverable 503.
type upstreamSearchError struct {
	status     int
	retryAfter string
	message    string
}

func (e *upstreamSearchError) Error() string { return e.message }

func upstreamErr(resp *http.Response) *upstreamSearchError {
	e := &upstreamSearchError{status: resp.StatusCode, message: fmt.Sprintf("search provider returned HTTP %d", resp.StatusCode)}
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		e.retryAfter = ra
	}
	return e
}

func searchBrave(cfg *config.Config, query string, limit int) ([]searchResult, error) {
	key := ""
	credentialSource := config.BuiltinSearchProviderEnv["brave"]
	if cfg != nil {
		provider, declared := cfg.Search.Providers["brave"]
		if declared && provider.Disabled {
			return nil, fmt.Errorf("search provider brave is disabled by configuration")
		}
		if declared {
			resolved, err := cfg.Search.ResolveSearchProviderKey("brave")
			if err != nil {
				return nil, fmt.Errorf("resolve search provider brave credential: %w", err)
			}
			key = resolved
		}
	}
	if key == "" {
		// Fallback: the provider's conventional environment variable.
		key = config.BuiltinSearchProviderKey("brave")
	}
	if key == "" {
		// Name the configuration surface an operator can act on rather than
		// reporting a bare vendor-specific variable name.
		return nil, fmt.Errorf("search provider brave has no credential: set search.providers.brave.api_key, api_key_file, or api_key_env in the engine config (or export %s)", credentialSource)
	}

	base := ""
	if cfg != nil {
		if provider, ok := cfg.Search.Providers["brave"]; ok {
			base = strings.TrimSpace(provider.APIURL)
		}
	}
	if base == "" {
		base = strings.TrimSpace(os.Getenv("UIAI_BRAVE_SEARCH_API_URL"))
	}
	if base == "" {
		base = "https://api.search.brave.com/res/v1/web/search"
	}
	u, err := searchAPIURL(base, "https://api.search.brave.com/res/v1/web/search", []string{"api.search.brave.com"})
	if err != nil {
		return nil, fmt.Errorf("invalid Brave API URL: %w", err)
	}
	q := u.Query()
	q.Set("q", query)
	q.Set("count", strconv.Itoa(limit))
	u.RawQuery = q.Encode()

	httpReq, err := http.NewRequest(http.MethodGet, u.String(), nil) // #nosec G704 -- URL validated by searchAPIURL host allowlist/loopback gate.
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("X-Subscription-Token", key)
	httpReq.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Do(httpReq) // #nosec G704 -- API URL is restricted by searchAPIURL allowlist/loopback test gate.
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, upstreamErr(resp)
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
		results = append(results, sanitizeSearchResult(searchResult{
			Title:       item.Title,
			URL:         item.URL,
			Description: item.Description,
			Source:      item.Profile.Name,
			Age:         item.Age,
		}))
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}
