package routes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

// executeSearchProvider is the sole per-provider cache/admission/normalization
// path for both fallback and explicitly requested research fanout.
func executeSearchProvider(req *http.Request, cfg *config.Config, provider, query string, limit int, ttl time.Duration) (results []searchResult, cached bool, err error) {
	if err := req.Context().Err(); err != nil {
		return nil, false, err
	}
	adapter := searchProviderMeta(provider).search
	if adapter == nil {
		return nil, false, fmt.Errorf("unsupported search provider")
	}
	if cfg != nil && cfg.Search.Providers[provider].Disabled {
		return nil, false, fmt.Errorf("search provider %s is disabled by configuration", provider)
	}
	if ttl > 0 {
		if cached, ok := getCachedSearch(provider, query, limit, time.Now()); ok {
			return cached, true, nil
		}
	}
	runtime := searchBreakersFor(req)
	if provider == "ddg" {
		if err := runtime.admitDDGRequest(); err != nil {
			return nil, false, err
		}
	}
	attempt, wait := runtime.acquire(provider)
	if wait > 0 {
		return nil, false, &searchCircuitOpenError{retryAfter: wait}
	}
	// Fanout must not turn one adapter panic into a process-wide crash or leak a
	// half-open slot. No provider-controlled panic value enters diagnostics.
	defer func() {
		if recover() != nil {
			results = nil
			cached = false
			err = fmt.Errorf("search provider execution failed")
		}
		runtime.complete(provider, attempt, err)
	}()
	results, err = adapter(req.Context(), cfg, query, limit)
	if req.Context().Err() != nil {
		err = req.Context().Err()
	}
	if err != nil {
		return nil, false, err
	}
	results = deduplicateSearchResults(results)
	annotateSearchEvidence(results, provider, query)
	if ttl > 0 {
		setCachedSearch(provider, query, limit, results, time.Now().Add(ttl))
	}
	return results, false, nil
}
