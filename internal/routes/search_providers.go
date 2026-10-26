package routes

import (
	"context"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"
)

// builtinSearchProvider describes a search provider the engine ships with.
type builtinSearchProvider struct {
	id            string
	name          string
	keyless       bool
	configured    func(*config.Config) (bool, string)
	capabilities  []string
	defaultAPIURL string
	apiURLEnv     string
	apiHosts      []string
	configAPIURL  bool
	search        func(context.Context, *config.Config, string, int) ([]searchResult, error)
}

// Initialize once after declarations: adapters consult this same registry at runtime.
var builtinSearchProviders []builtinSearchProvider

func init() {
	builtinSearchProviders = []builtinSearchProvider{
		{id: "ddg", name: "DuckDuckGo HTML", keyless: true, defaultAPIURL: "https://html.duckduckgo.com/html/", apiHosts: []string{"html.duckduckgo.com"}, configAPIURL: true, search: searchDDG, capabilities: []string{"web_search", "source_urls", "snippets", "keyless_public"}},
		{id: "searxng", name: "SearxNG", keyless: true, configured: searxSearchConfigured, search: searchSearxNG, capabilities: []string{"web_search", "source_urls", "snippets", "keyless_configured_endpoint"}},
		{id: "exa", defaultAPIURL: "https://api.exa.ai/search", apiHosts: []string{"api.exa.ai"}, configAPIURL: true, name: "Exa Search", search: searchExa, capabilities: []string{"web_search", "source_urls", "snippets"}},
		{id: "tavily", defaultAPIURL: "https://api.tavily.com/search", apiHosts: []string{"api.tavily.com"}, configAPIURL: true, name: "Tavily Search", search: searchTavily, capabilities: []string{"web_search", "source_urls", "snippets"}},
		{id: "brave", defaultAPIURL: "https://api.search.brave.com/res/v1/web/search", apiURLEnv: "UIAI_BRAVE_SEARCH_API_URL", apiHosts: []string{"api.search.brave.com"}, configAPIURL: true, name: "Brave Search", search: searchBraveContext, capabilities: []string{"web_search", "source_urls", "snippets"}},
		{id: "wikipedia", defaultAPIURL: "https://en.wikipedia.org/w/api.php", apiURLEnv: "UIAI_WIKIPEDIA_SEARCH_API_URL", apiHosts: []string{"en.wikipedia.org", "wikipedia.org"}, name: "Wikipedia OpenSearch", search: func(ctx context.Context, _ *config.Config, query string, limit int) ([]searchResult, error) {
			return searchWikipediaContext(ctx, query, limit)
		}, keyless: true, capabilities: []string{"encyclopedia_search", "source_urls", "snippets", "keyless_public"}},
	}
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
		add(strings.ToLower(strings.TrimSpace(cfg.Search.DefaultProvider)))
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
	if meta.configured != nil {
		return meta.configured(cfg)
	}
	if _, err := searchProviderEndpoint(cfg, id); err != nil {
		return false, "invalid_endpoint"
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
// names none: an explicit default is query-disclosure intent, not permission to
// silently choose another configured provider when it is broken. Only the
// approved fallback plan can move its queries. Without an explicit default,
// prefer established configured providers, then public keyless defaults; empty means none ready.
func resolveDefaultSearchProvider(cfg *config.Config) string {
	ids := searchProviderIDs(cfg)
	if cfg != nil {
		if wanted := strings.ToLower(strings.TrimSpace(cfg.Search.DefaultProvider)); wanted != "" {
			return wanted
		}
	}
	// Preserve established configured-provider selection before introducing a
	// new implicit public fallback. DDG must not silently displace an existing
	// Exa/Tavily credential or explicitly configured SearxNG endpoint.
	for _, id := range ids {
		if slices.Contains(searchProviderMeta(id).capabilities, "keyless_public") {
			continue
		}
		if configured, _ := searchProviderConfigured(cfg, id); configured {
			return id
		}
	}
	for _, id := range ids {
		if configured, _ := searchProviderConfigured(cfg, id); configured {
			return id
		}
	}
	return ""
}

type searchProviderInventory struct {
	defaultProvider string
	ids, ready      []string
	providers       []map[string]any
}

func buildSearchProviderInventory(cfg *config.Config, breakers *SearchRuntime) searchProviderInventory {
	ids := searchProviderIDs(cfg)
	providers := make([]map[string]any, 0, len(ids))
	ready := make([]string, 0, len(ids))
	for _, id := range ids {
		meta := searchProviderMeta(id)
		configured, reason := searchProviderConfigured(cfg, id)
		circuit := breakers.status(id)
		status := "ready"
		degradedReason := ""
		if !configured {
			status = "degraded"
			degradedReason = reason
		} else if circuit.RetryAfterSeconds > 0 {
			status = "degraded"
			degradedReason = "circuit_open"
		}
		pacingDelay := 0
		if id == "ddg" {
			pacingDelay = breakers.ddgRequestDelay()
		}
		if configured && circuit.RetryAfterSeconds == 0 && pacingDelay > 0 {
			status = "degraded"
			degradedReason = "local_rate_limit"
		}
		retryDelay := circuit.RetryAfterSeconds
		if degradedReason == "local_rate_limit" {
			retryDelay = pacingDelay
		}
		if status == "ready" {
			ready = append(ready, id)
		}
		providers = append(providers, map[string]any{
			"id":                id,
			"name":              meta.name,
			"configured":        configured,
			"status":            status,
			"degraded_reason":   degradedReason,
			"setup":             buildSearchProviderSetup(id, degradedReason, retryDelay),
			"cache_ttl_seconds": int(searchCacheTTL() / time.Second),
			"capabilities":      meta.capabilities,
			"circuit":           circuit,
		})
	}

	return searchProviderInventory{defaultProvider: resolveDefaultSearchProvider(cfg), ids: ids, ready: ready, providers: providers}
}
func handleSearchProviders(w http.ResponseWriter, cfg *config.Config, states ...*SearchRuntime) {
	inventory := buildSearchProviderInventory(cfg, firstSearchRuntime(states...))
	writeJSON(w, 200, map[string]any{"schema": "uiai.search_providers.v1", "default_provider": inventory.defaultProvider, "available_providers": inventory.ids, "ready_providers": inventory.ready, "providers": inventory.providers, "setup": searchSetupGuide(inventory, firstSearchRuntime(states...) != nil)})
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
