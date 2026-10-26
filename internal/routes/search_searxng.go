package routes

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

// The endpoint is operator-owned configuration, never a request-supplied URL or
// an implicit public-instance choice. All other adapters retain their allowlists.
func searxSearchEndpoint(cfg *config.Config) (*url.URL, string) {
	if cfg == nil {
		return nil, "missing_endpoint"
	}
	provider := cfg.Search.Providers["searxng"]
	if provider.Disabled {
		return nil, "disabled"
	}
	raw := strings.TrimSpace(provider.APIURL)
	if raw == "" {
		return nil, "missing_endpoint"
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return nil, "invalid_endpoint"
	}
	endpoint, err := searchAPIURL(raw, "", []string{parsed.Hostname()})
	if err != nil {
		return nil, "invalid_endpoint"
	}
	return endpoint, ""
}
func searxSearchConfigured(cfg *config.Config) (bool, string) {
	_, reason := searxSearchEndpoint(cfg)
	return reason == "", reason
}
func searchSearxNG(ctx context.Context, cfg *config.Config, query string, limit int) ([]searchResult, error) {
	endpoint, reason := searxSearchEndpoint(cfg)
	if reason != "" {
		return nil, fmt.Errorf("SearxNG search endpoint unavailable: %s", reason)
	}
	form := url.Values{"q": {query}, "format": {"json"}}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("invalid SearxNG search request")
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	var decoded struct {
		Results []struct {
			Score         *float64 `json:"score"`
			Title         string   `json:"title"`
			URL           string   `json:"url"`
			Content       string   `json:"content"`
			PublishedDate string   `json:"publishedDate"`
		} `json:"results"`
	}
	if err := searchJSON(req, &decoded); err != nil {
		return nil, err
	}
	results := make([]searchResult, 0, min(len(decoded.Results), normalizeSearchLimit(limit)))
	for _, item := range decoded.Results {
		if strings.TrimSpace(item.URL) == "" {
			continue
		}
		results = append(results, sanitizeSearchResult(searchResult{Title: item.Title, Score: item.Score, URL: item.URL, Description: item.Content, Age: item.PublishedDate, Source: "SearxNG"}))
		if len(results) >= normalizeSearchLimit(limit) {
			break
		}
	}
	return results, nil
}
