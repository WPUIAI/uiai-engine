package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/httpboundary"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func searchWikipedia(query string, limit int) ([]searchResult, error) {
	return searchWikipediaContext(context.Background(), query, limit)
}

func searchWikipediaContext(ctx context.Context, query string, limit int) ([]searchResult, error) {
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

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil) // #nosec G704 -- URL validated by searchAPIURL host allowlist/loopback gate.
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "uiai-engine/agent-search (+https://github.com/WPUIAI/uiai-engine)")

	client := httpboundary.OriginBoundClient(&http.Client{Timeout: 12 * time.Second})
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
	return searchBraveContext(context.Background(), cfg, query, limit)
}

func searchBraveContext(ctx context.Context, cfg *config.Config, query string, limit int) ([]searchResult, error) {
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

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil) // #nosec G704 -- URL validated by searchAPIURL host allowlist/loopback gate.
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("X-Subscription-Token", key)
	httpReq.Header.Set("Accept", "application/json")

	client := httpboundary.OriginBoundClient(&http.Client{Timeout: 12 * time.Second})
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
