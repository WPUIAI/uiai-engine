package routes

import (
	"context"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"net/http"
	"sort"
	"strings"
	"time"
)

// builtinSearchProvider describes a search provider the engine ships with.
type builtinSearchProvider struct {
	id           string
	name         string
	keyless      bool
	capabilities []string
	search       func(context.Context, *config.Config, string, int) ([]searchResult, error)
}

var builtinSearchProviders = []builtinSearchProvider{
	{id: "brave", name: "Brave Search", search: searchBraveContext, capabilities: []string{"web_search", "source_urls", "snippets"}},
	{id: "wikipedia", name: "Wikipedia OpenSearch", search: func(ctx context.Context, _ *config.Config, query string, limit int) ([]searchResult, error) {
		return searchWikipediaContext(ctx, query, limit)
	}, keyless: true, capabilities: []string{"encyclopedia_search", "source_urls", "snippets", "keyless_public"}},
}

// searchProviderIDs returns every provider this deployment can serve: the built-in
// set plus configured ids for diagnostic visibility. A configured credential
// does not make an unimplemented adapter ready.
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
	if meta.search == nil {
		return false, "unsupported_provider"
	}
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

// searchProviderPlan permits query disclosure only to explicitly configured
// alternates. An explicit request provider always stays pinned.
func searchProviderPlan(cfg *config.Config, explicit, primary string) []string {
	plan := []string{primary}
	if strings.TrimSpace(explicit) != "" || cfg == nil {
		return plan
	}
	seen := map[string]bool{primary: true}
	for _, id := range cfg.Search.FallbackProviders {
		id = strings.ToLower(strings.TrimSpace(id))
		if seen[id] {
			continue
		}
		if ready, _ := searchProviderConfigured(cfg, id); !ready {
			continue
		}
		seen[id] = true
		plan = append(plan, id)
	}
	return plan
}
