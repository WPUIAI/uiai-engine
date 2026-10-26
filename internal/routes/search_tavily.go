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

// searchTavily follows the documented /search JSON contract using header-only
// credentials. Auto-upgrades, generated answers and raw-page retrieval are off.
func searchTavily(ctx context.Context, cfg *config.Config, query string, limit int) ([]searchResult, error) {
	key, err := searchCredential(cfg, "tavily")
	if err != nil {
		return nil, err
	}
	endpoint, err := searchProviderEndpoint(cfg, "tavily")
	if err != nil {
		return nil, fmt.Errorf("invalid Tavily API URL")
	}
	body, err := json.Marshal(map[string]any{"query": query, "max_results": normalizeSearchLimit(limit), "search_depth": "basic", "auto_parameters": false, "include_answer": false, "include_raw_content": false, "include_images": false})
	if err != nil {
		return nil, fmt.Errorf("invalid Tavily search request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint.String(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("invalid Tavily search request")
	}
	req.Header.Set("Authorization", "Bearer "+key)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	var decoded struct {
		Results []struct {
			Score         *float64 `json:"score"`
			Title         string   `json:"title"`
			URL           string   `json:"url"`
			Content       string   `json:"content"`
			PublishedDate string   `json:"published_date"`
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
		results = append(results, sanitizeSearchResult(searchResult{Title: item.Title, Score: item.Score, URL: item.URL, Description: item.Content, Source: "Tavily", Age: item.PublishedDate}))
		if len(results) >= normalizeSearchLimit(limit) {
			break
		}
	}
	return results, nil
}
