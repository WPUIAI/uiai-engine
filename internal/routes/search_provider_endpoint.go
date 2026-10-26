package routes

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

// searchProviderEndpoint shares the registry's endpoint source and host policy
// between local setup validation and actual adapter execution. It performs no
// DNS, credential, account, or provider request. Wikipedia retains its legacy
// environment-only override; an unused provider APIURL cannot change that path.
func searchProviderEndpoint(cfg *config.Config, id string) (*url.URL, error) {
	meta := searchProviderMeta(id)
	if meta.defaultAPIURL == "" {
		return nil, fmt.Errorf("search provider has no installed endpoint policy")
	}
	raw := ""
	if cfg != nil && meta.configAPIURL {
		raw = strings.TrimSpace(cfg.Search.Providers[id].APIURL)
	}
	if raw == "" && meta.apiURLEnv != "" {
		raw = strings.TrimSpace(os.Getenv(meta.apiURLEnv))
	}
	return searchAPIURL(raw, meta.defaultAPIURL, meta.apiHosts)
}
