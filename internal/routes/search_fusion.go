package routes

import (
	"context"
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

const maxSearchResearchProviders = 3
const searchResearchTimeout = 12 * time.Second

type searchResultSource struct {
	Provider    string `json:"provider"`
	EvidenceRef string `json:"evidence_ref"`
	Rank        int    `json:"rank"`
}
type searchResearchFailure struct {
	Provider       string `json:"provider"`
	Code           string `json:"code"`
	UpstreamStatus int    `json:"upstream_status,omitempty"`
	Retryable      bool   `json:"retryable"`
	RetryAfter     string `json:"retry_after,omitempty"`
}
type searchResearchDetails struct {
	Providers []string
	Failures  []searchResearchFailure
}
type searchResearchLane struct {
	provider string
	results  []searchResult
	cached   bool
	err      error
}

func researchSearchPlan(cfg *config.Config, req searchRequest) ([]string, int, string) {
	if strings.TrimSpace(req.Provider) != "" {
		return nil, 400, "conflicting_search_providers"
	}
	if len(req.Providers) > maxSearchResearchProviders {
		return nil, 400, "too_many_search_providers"
	}
	approved := map[string]bool{resolveDefaultSearchProvider(cfg): true}
	if cfg != nil {
		for _, id := range cfg.Search.FallbackProviders {
			approved[strings.ToLower(strings.TrimSpace(id))] = true
		}
	}
	ids := []string{}
	seen := map[string]bool{}
	for _, raw := range req.Providers {
		id := strings.ToLower(strings.TrimSpace(raw))
		if searchProviderMeta(id).search == nil {
			return nil, 400, "unsupported_provider"
		}
		if !approved[id] {
			return nil, 403, "search_provider_not_approved"
		}
		if cfg != nil && cfg.Search.Providers[id].Disabled {
			return nil, 403, "search_provider_disabled"
		}
		if !seen[id] {
			ids = append(ids, id)
			seen[id] = true
		}
	}
	if len(ids) < 2 {
		return nil, 400, "research_requires_distinct_providers"
	}
	return ids, 0, ""
}
func researchSearchFailure(provider string, err error) searchResearchFailure {
	failure := searchResearchFailure{Provider: provider, Code: "search_provider_unavailable"}
	var upstream *upstreamSearchError
	var circuit *searchCircuitOpenError
	if errors.As(err, &upstream) {
		failure.Code = "search_provider_error"
		failure.UpstreamStatus = upstream.status
		failure.Retryable = retryableSearchStatus(upstream.status)
		failure.RetryAfter = upstream.retryAfter
	} else if errors.As(err, &circuit) {
		failure.Code = "search_provider_circuit_open"
		failure.Retryable = true
		failure.RetryAfter = strconv.Itoa(circuit.retryAfter)
	} else if errors.Is(err, context.Canceled) {
		failure.Code = "search_cancelled"
	} else if errors.Is(err, context.DeadlineExceeded) {
		failure.Code = "search_deadline"
	}
	return failure
}
func runResearchSearch(w http.ResponseWriter, parent *http.Request, cfg *config.Config, query string, req searchRequest) {
	ids, status, code := researchSearchPlan(cfg, req)
	if status != 0 {
		writeJSON(w, status, map[string]any{"error": code})
		return
	}
	ctx, cancel := context.WithTimeout(parent.Context(), searchResearchTimeout)
	defer cancel()
	request := parent.Clone(ctx)
	limit := normalizeSearchLimit(req.Limit)
	ttl := searchCacheTTL()
	lanes := make([]searchResearchLane, len(ids))
	var wg sync.WaitGroup
	for i, id := range ids {
		lanes[i].provider = id
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			lane := &lanes[i]
			lane.results, lane.cached, lane.err = executeSearchProvider(request, cfg, lane.provider, query, limit, ttl)
		}(i)
	}
	wg.Wait()
	details := searchResearchDetails{}
	successes := 0
	allCached := true
	allLimited := true
	retryable := false
	for _, lane := range lanes {
		if lane.err == nil {
			successes++
			details.Providers = append(details.Providers, lane.provider)
			allCached = allCached && lane.cached
			continue
		}
		allCached = false
		failure := researchSearchFailure(lane.provider, lane.err)
		details.Failures = append(details.Failures, failure)
		allLimited = allLimited && failure.UpstreamStatus == 429
		retryable = retryable || failure.Retryable
	}
	if successes == 0 {
		status = 502
		code = "search_providers_failed"
		if allLimited {
			status = 429
			code = "search_provider_limited"
		}
		w.Header().Set("Cache-Control", "no-store")
		if len(details.Failures) > 0 {
			same := details.Failures[0].RetryAfter
			for _, f := range details.Failures[1:] {
				if f.RetryAfter != same {
					same = ""
					break
				}
			}
			if same != "" {
				w.Header().Set("Retry-After", same)
			}
		}
		writeJSON(w, status, map[string]any{"error": code, "provider_errors": details.Failures, "retryable": retryable})
		return
	}
	results := fuseSearchResults(lanes, limit)
	writeSearchResponse(w, parent, cfg, "fusion", query, results, allCached, ttl, details)
}
func fuseSearchResults(lanes []searchResearchLane, limit int) []searchResult {
	merged := []searchResult{}
	for _, lane := range lanes {
		if lane.err != nil {
			continue
		}
		for _, result := range lane.results {
			index := -1
			for i, old := range merged {
				if sameSearchCitation(result, old) {
					index = i
					break
				}
			}
			if index < 0 {
				result = cloneSearchResults([]searchResult{result})[0]
				result.Sources = nil
				result.FusionScore = 0
				merged = append(merged, result)
				index = len(merged) - 1
			}
			merged[index].FusionScore += 1 / (60 + float64(result.Rank))
			merged[index].Sources = append(merged[index].Sources, searchResultSource{Provider: lane.provider, EvidenceRef: result.EvidenceRef, Rank: result.Rank})
		}
	}
	sort.SliceStable(merged, func(i, j int) bool { return merged[i].FusionScore > merged[j].FusionScore })
	if len(merged) > limit {
		merged = merged[:limit]
	}
	for i := range merged {
		merged[i].Rank = i + 1
	}
	return merged
}
