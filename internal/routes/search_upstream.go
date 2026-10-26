package routes

import (
	"context"
	"fmt"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

func searchWikipedia(query string, limit int) ([]searchResult, error) {
	return searchWikipediaContext(context.Background(), query, limit)
}

func searchWikipediaContext(ctx context.Context, query string, limit int) ([]searchResult, error) {
	u, err := searchProviderEndpoint(nil, "wikipedia")
	if err != nil {
		return nil, fmt.Errorf("invalid Wikipedia API URL")
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

	var decoded []any
	if err := searchJSON(httpReq, &decoded); err != nil {
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
		return nil, fmt.Errorf("invalid search API URL")
	}
	if u.RawQuery != "" || u.ForceQuery || strings.Contains(raw, "#") {
		return nil, fmt.Errorf("search API URL cannot contain query or fragment")
	}
	if port := u.Port(); port != "" {
		value, err := strconv.Atoi(port)
		if err != nil || value < 1 || value > 65535 {
			return nil, fmt.Errorf("invalid search API port")
		}
	}
	if u.User != nil {
		return nil, fmt.Errorf("search API URL cannot contain user information")
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, fmt.Errorf("search API URL must use HTTP or HTTPS")
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
	if ra := strings.TrimSpace(resp.Header.Get("Retry-After")); ra != "" {
		if _, err := strconv.ParseUint(ra, 10, 32); err == nil {
			e.retryAfter = ra
		} else if _, err := http.ParseTime(ra); err == nil {
			e.retryAfter = ra
		}
	}
	return e
}

func searchBrave(cfg *config.Config, query string, limit int) ([]searchResult, error) {
	return searchBraveContext(context.Background(), cfg, query, limit)
}

func searchBraveContext(ctx context.Context, cfg *config.Config, query string, limit int) ([]searchResult, error) {
	key, err := searchCredential(cfg, "brave")
	if err != nil {
		return nil, err
	}

	u, err := searchProviderEndpoint(cfg, "brave")
	if err != nil {
		return nil, fmt.Errorf("invalid Brave API URL")
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

	var decoded braveWebResponse
	if err := searchJSON(httpReq, &decoded); err != nil {
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
