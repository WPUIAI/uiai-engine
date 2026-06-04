package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/go-chi/chi/v5"
)

const defaultSearchLimit = 5
const maxSearchLimit = 20

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
}

type searchResponse struct {
	Schema   string         `json:"schema"`
	Provider string         `json:"provider"`
	Query    string         `json:"query"`
	Count    int            `json:"count"`
	Results  []searchResult `json:"results"`
	Next     []string       `json:"next"`
}

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
	writeJSON(w, 200, map[string]any{
		"schema":           "uiai.search_providers.v1",
		"default_provider": "brave",
		"providers": []map[string]any{
			{
				"id":           "brave",
				"name":         "Brave Search",
				"configured":   strings.TrimSpace(os.Getenv("BRAVE_SEARCH_API_KEY")) != "",
				"capabilities": []string{"web_search", "source_urls", "snippets"},
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
	writeJSON(w, 200, searchResponse{
		Schema:   "uiai.search_results.v1",
		Provider: provider,
		Query:    query,
		Count:    len(results),
		Results:  results,
		Next:     []string{"browser_open a selected result URL", "browser_read page text", "browser_diagnostics on navigation failure"},
	})
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
