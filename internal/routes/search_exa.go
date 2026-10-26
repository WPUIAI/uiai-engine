package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

// searchExa keeps the vendor protocol at this boundary; credentials, HTTP
// origin/size/error controls and result redaction use shared canonical owners.
func searchExa(ctx context.Context, cfg *config.Config, query string, limit int) ([]searchResult, error) {
	key, err := searchCredential(cfg, "exa")
	if err != nil {
		return nil, err
	}
	endpoint, err := searchProviderEndpoint(cfg, "exa")
	if err != nil {
		return nil, fmt.Errorf("invalid Exa API URL")
	}
	payload, err := json.Marshal(map[string]any{"query": query, "numResults": normalizeSearchLimit(limit), "type": "auto", "contents": map[string]bool{"highlights": true}})
	if err != nil {
		return nil, fmt.Errorf("invalid Exa search request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("invalid Exa search request")
	}
	req.Header.Set("x-api-key", key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	var decoded struct {
		Results []struct {
			Score         *float64 `json:"score"`
			Title         string   `json:"title"`
			URL           string   `json:"url"`
			PublishedDate string   `json:"publishedDate"`
			Highlights    []string `json:"highlights"`
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
		results = append(results, sanitizeSearchResult(searchResult{Title: item.Title, Score: item.Score, URL: item.URL, Description: strings.Join(item.Highlights, " "), Source: "Exa", Age: item.PublishedDate}))
		if len(results) >= normalizeSearchLimit(limit) {
			break
		}
	}
	return results, nil
}
