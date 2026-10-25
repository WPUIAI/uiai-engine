package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/httpboundary"
)

const maxSearchResponseBytes = 2 << 20

// searchJSON is the shared bounded, origin-confined transport. Provider bodies,
// credential-bearing URLs and header errors never become diagnostic messages.
func searchJSON(req *http.Request, out any) error {
	client := httpboundary.OriginBoundClient(&http.Client{Timeout: 12 * time.Second})
	resp, err := client.Do(req)
	if err != nil {
		if req.Context().Err() != nil {
			return req.Context().Err()
		}
		return fmt.Errorf("search provider transport failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return upstreamErr(resp)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxSearchResponseBytes+1))
	if err != nil {
		return fmt.Errorf("search provider response read failed")
	}
	if len(body) > maxSearchResponseBytes {
		return fmt.Errorf("search provider response exceeds size limit")
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("search provider returned invalid JSON")
	}
	return nil
}

// searchCredential gives explicit operator configuration precedence even when
// empty: a missing configured key cannot silently reuse a conventional key.
func searchCredential(cfg *config.Config, id string) (string, error) {
	key := ""
	declared := false
	if cfg != nil {
		var provider config.SearchProviderConfig
		provider, declared = cfg.Search.Providers[id]
		if declared && provider.Disabled {
			return "", fmt.Errorf("search provider %s is disabled by configuration", id)
		}
		if declared {
			var err error
			key, err = cfg.Search.ResolveSearchProviderKey(id)
			if err != nil {
				return "", fmt.Errorf("search provider %s credential unreadable", id)
			}
		}
	}
	if !declared {
		key = config.BuiltinSearchProviderKey(id)
	}
	if key == "" {
		return "", fmt.Errorf("search provider %s has no credential: set search.providers.%s.api_key, search.providers.%s.api_key_file, or search.providers.%s.api_key_env", id, id, id, id)
	}
	return key, nil
}
