package routes

import (
	"fmt"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"strings"
)

// searchProviderSetup is guidance, not permission, a credential value, or a
// configuration writer. It derives only from the canonical inventory decision.
type searchProviderSetup struct {
	NextAction              string   `json:"next_action"`
	Message                 string   `json:"message"`
	ConfigurationFields     []string `json:"configuration_fields,omitempty"`
	ConventionalEnvFallback string   `json:"conventional_env_fallback,omitempty"`
	EndpointEnvFallback     string   `json:"endpoint_env_fallback,omitempty"`
	RetryAfterSeconds       int      `json:"retry_after_seconds,omitempty"`
}

func buildSearchProviderSetup(id, reason string, retry int) searchProviderSetup {
	s := searchProviderSetup{}
	switch reason {
	case "conflicting_search_providers":
		s.NextAction = "choose_search_mode"
		s.Message = "Send either provider for a pinned single-provider search or providers for explicitly approved research, not both; do not silently choose between them."
	case "too_many_search_providers":
		s.NextAction = "reduce_research_provider_count"
		s.Message = fmt.Sprintf("Use two to %d distinct explicitly approved providers; narrow the request without adding or substituting a provider automatically.", maxSearchResearchProviders)
	case "research_requires_distinct_providers":
		s.NextAction = "choose_distinct_approved_providers"
		s.Message = "Research requires at least two distinct explicitly approved providers. For one provider, use provider instead of providers; do not add a provider without disclosure authorization."
	case "search_provider_not_approved":
		s.NextAction = "request_provider_disclosure_approval"
		s.Message = "The requested provider is outside the explicitly approved research set. Obtain the user's authorization for query disclosure or narrow the request to already approved providers; do not change defaults or fallbacks, enable providers, acquire keys, or send the query automatically."
	case "disabled", "search_provider_disabled":
		s.NextAction = "keep_disabled"
		s.Message = "Explicitly disabled; do not enable or replace configuration without the user's direction."
	case "unsupported_provider":
		s.NextAction = "choose_supported_provider"
		s.Message = "No installed adapter; adding a key or endpoint cannot install one."
	case "credential_unreadable":
		s.NextAction = "repair_existing_credential_reference"
		s.Message = "Repair the existing credential reference or its access; do not silently select another key or create an account."
	case "missing_key":
		s.NextAction = "connect_existing_credential_reference"
		s.Message = "Use the name of an existing approved secret environment variable or an approved credential-file reference, never a key value in chat. Account creation is a separate user-approved action."
		prefix := "search.providers." + id + "."
		s.ConfigurationFields = []string{prefix + "api_key_env", prefix + "api_key_file"}
		s.ConventionalEnvFallback = config.BuiltinSearchProviderEnv[id]
	case "missing_endpoint":
		s.NextAction = "connect_existing_endpoint"
		s.Message = "Use an approved JSON-enabled SearxNG search endpoint: HTTPS (HTTP only for loopback development), no userinfo, query, or fragment. Do not pick a public instance or provision a service automatically."
		s.ConfigurationFields = []string{"search.providers.searxng.api_url"}
	case "invalid_endpoint":
		s.NextAction = "repair_existing_endpoint"
		s.Message = "Repair the existing provider API URL using its approved host policy: HTTPS (HTTP only for loopback development), valid port, no userinfo, query, or fragment. Discovery does not return private URL values or select another provider."
		meta := searchProviderMeta(id)
		if meta.configAPIURL || id == "searxng" {
			s.ConfigurationFields = []string{"search.providers." + id + ".api_url"}
		}
		s.EndpointEnvFallback = meta.apiURLEnv
	case "circuit_open":
		s.NextAction = "wait_then_retry"
		s.Message = "Respect the reported delay; do not replace credentials or run background probes."
		s.RetryAfterSeconds = retry
	default:
		s.NextAction = "use_selected_provider"
		s.Message = "Configured and admitted, not proof of external availability or remaining quota. Run only the user's requested search, not an automatic setup probe."
	}
	return s
}

// searchSetupGuide is the single read-only setup projection shared by provider
// discovery and the agent card. No browser, provider, account, or config mutation
// occurs here; future human surfaces can consume the same contract.
func searchSetupGuide(inventory searchProviderInventory, observed bool) map[string]any {
	byCapability := map[string][]string{"web_search": {}, "encyclopedia_search": {}}
	steps := make(map[string]any, len(inventory.providers))
	for _, p := range inventory.providers {
		id := p["id"].(string)
		steps[id] = p["setup"]
		if p["status"] == "ready" {
			for _, capability := range p["capabilities"].([]string) {
				if _, known := byCapability[capability]; known {
					byCapability[capability] = append(byCapability[capability], id)
				}
			}
		}
	}
	basis := "configuration_only"
	if observed {
		basis = "observed_circuits"
	}
	return map[string]any{
		"schema":                        "uiai.search_setup.v1",
		"discovery_endpoint":            "/api/search/providers",
		"default_provider":              inventory.defaultProvider,
		"default_capabilities":          searchProviderMeta(inventory.defaultProvider).capabilities,
		"ready_providers_by_capability": byCapability,
		"general_web_search_ready":      len(byCapability["web_search"]) > 0,
		"provider_setup":                steps,
		"readiness_basis":               basis,
		"remaining_quota_status":        "unknown",
		"requires_gui":                  false,
		"configuration_write_supported": false,
		"upstream_probes_performed":     false,
		"configuration_activation":      "Use the existing deployment configuration and supported restart/reload mechanism; discovery neither writes nor activates configuration.",
		"credential_precedence":         "Explicit provider configuration wins, even when empty. Conventional environment fallback applies only to undeclared providers.",
		"safety_rules":                  []string{"Never reveal credential values; prefer existing approved secret references.", "Account creation, spending, provider disclosure changes, and service provisioning require their own authorization; discovery grants none."},
	}
}

// addSearchSetupRecovery uses the existing cross-client suggested_next_action
// field. Native/MCP/CLI error renderers already preserve it; no client upgrade
// or privileged configuration API is required. Fields are static safe names,
// never private configured references, endpoint values, or credentials.
func addSearchSetupRecovery(body map[string]any, setup searchProviderSetup) {
	body["setup"] = setup
	hint := setup.NextAction + ": " + setup.Message
	if len(setup.ConfigurationFields) > 0 {
		hint += " Configuration fields: " + strings.Join(setup.ConfigurationFields, ", ") + "."
	}
	if setup.ConventionalEnvFallback != "" {
		hint += " Conventional credential env (undeclared provider only): " + setup.ConventionalEnvFallback + "."
	}
	if setup.EndpointEnvFallback != "" {
		hint += " Endpoint env fallback: " + setup.EndpointEnvFallback + "."
	}
	hint += " Inspect current setup with the agent card or GET /api/search/providers."
	body["suggested_next_action"] = hint
}
