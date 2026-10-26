package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/go-chi/chi/v5"
)

func setupRouter(cfg *config.Config, state *SearchRuntime) http.Handler {
	r := chi.NewRouter()
	r.Route("/api/tools", func(r chi.Router) { MountToolsDiscovery(r, cfg, state) })
	r.Route("/api/search", func(r chi.Router) { MountSearchRoutes(r, cfg, state) })
	return r
}
func readSetup(t *testing.T, r http.Handler, path string) (map[string]any, string) {
	t.Helper()
	out := httptest.NewRecorder()
	r.ServeHTTP(out, httptest.NewRequest(http.MethodGet, path, nil))
	if out.Code != 200 {
		t.Fatalf("discovery status %d", out.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(out.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	setup, ok := body["setup"].(map[string]any)
	if !ok {
		t.Fatal("missing actual endpoint setup guide")
	}
	return setup, out.Body.String()
}
func setupAction(t *testing.T, guide map[string]any, id string) string {
	t.Helper()
	return guide["provider_setup"].(map[string]any)[id].(map[string]any)["next_action"].(string)
}

func clearSearchSetupCredentials(t *testing.T) {
	t.Helper()
	for _, env := range config.BuiltinSearchProviderEnv {
		t.Setenv(env, "")
	}
}
func TestSearchSetupWikipediaOnlyHasNoGeneralWebClaim(t *testing.T) {
	clearSearchSetupCredentials(t)
	r := setupRouter(&config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{"ddg": {Disabled: true}}}}, NewSearchRuntime())
	guide, _ := readSetup(t, r, "/api/tools/agent-card")
	same, _ := readSetup(t, r, "/api/search/providers")
	if !reflect.DeepEqual(guide, same) {
		t.Fatal("card and provider setup disagree")
	}
	if guide["general_web_search_ready"] != false || guide["default_provider"] != "wikipedia" {
		t.Fatal("encyclopedia presented as general web search")
	}
	if guide["requires_gui"] != false || guide["configuration_write_supported"] != false || guide["upstream_probes_performed"] != false || guide["remaining_quota_status"] != "unknown" {
		t.Fatal("unsupported setup or quota claim")
	}
	if setupAction(t, guide, "brave") != "connect_existing_credential_reference" || setupAction(t, guide, "searxng") != "connect_existing_endpoint" || setupAction(t, guide, "wikipedia") != "use_selected_provider" {
		t.Fatal("missing actionable clean-install guidance")
	}
	br := guide["provider_setup"].(map[string]any)["brave"].(map[string]any)
	if br["conventional_env_fallback"] != "BRAVE_SEARCH_API_KEY" || !strings.Contains(br["message"].(string), "never a key value in chat") {
		t.Fatal("missing safe credential instructions")
	}
}
func TestSearchSetupExplainsExistingFailuresWithoutSecretDisclosure(t *testing.T) {
	clearSearchSetupCredentials(t)
	const secret = "fixture-credential-value-never-print"
	const privatePath = "/private/fixture-do-not-report-token-path"
	cfg := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{
		"brave":         {APIKey: secret},
		"tavily":        {Disabled: true, APIKey: secret},
		"exa":           {APIKeyFile: privatePath},
		"searxng":       {APIURL: "https://username:" + secret + "@private.example/search"},
		"unimplemented": {APIKey: secret},
	}}}
	r := setupRouter(cfg, NewSearchRuntime())
	for _, path := range []string{"/api/tools/agent-card", "/api/search/providers"} {
		guide, raw := readSetup(t, r, path)
		for _, value := range []string{secret, privatePath, "private.example"} {
			if strings.Contains(raw, value) {
				t.Fatal("discovery leaked operator configuration")
			}
		}
		for id, want := range map[string]string{"brave": "use_selected_provider", "tavily": "keep_disabled", "exa": "repair_existing_credential_reference", "searxng": "repair_existing_endpoint", "unimplemented": "choose_supported_provider"} {
			if setupAction(t, guide, id) != want {
				t.Fatalf("wrong setup action for %s", id)
			}
		}
		if guide["general_web_search_ready"] != true {
			t.Fatal("configured web capability missing")
		}
	}
}
func TestSearchSetupDoesNotProbeProviderAndSharesCircuitState(t *testing.T) {
	clearSearchSetupCredentials(t)
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) }))
	defer upstream.Close()
	cfg := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{"searxng": {APIURL: upstream.URL + "/search"}, "ddg": {Disabled: true}}}}
	state, _ := testSearchBreakers()
	r := setupRouter(cfg, state)
	guide, _ := readSetup(t, r, "/api/tools/agent-card")
	if setupAction(t, guide, "searxng") != "use_selected_provider" {
		t.Fatal("configured endpoint not recognized")
	}
	for i := 0; i < 2; i++ {
		a, wait := state.acquire("searxng")
		if wait != 0 {
			t.Fatal("fixture not admitted")
		}
		state.complete("searxng", a, &upstreamSearchError{status: 429, retryAfter: "60"})
	}
	for _, path := range []string{"/api/tools/agent-card", "/api/search/providers"} {
		guide, _ = readSetup(t, r, path)
		if setupAction(t, guide, "searxng") != "wait_then_retry" || guide["general_web_search_ready"] != false {
			t.Fatal("discovery ignored shared search circuit")
		}
		s := guide["provider_setup"].(map[string]any)["searxng"].(map[string]any)
		if s["retry_after_seconds"] != float64(60) {
			t.Fatal("retry delay not preserved")
		}
	}
	if calls.Load() != 0 {
		t.Fatal("discovery contacted external provider")
	}
}
func TestSearchSetupExplicitEmptyCredentialDoesNotFallBack(t *testing.T) {
	clearSearchSetupCredentials(t)
	t.Setenv("BRAVE_SEARCH_API_KEY", "fixture-conventional-key")
	cfg := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{"brave": {}}}}
	guide, raw := readSetup(t, setupRouter(cfg, nil), "/api/tools/agent-card")
	if setupAction(t, guide, "brave") != "connect_existing_credential_reference" || guide["readiness_basis"] != "configuration_only" || strings.Contains(raw, "fixture-conventional-key") {
		t.Fatal("explicit source precedence or value-free legacy readiness violated")
	}
	matches := rankedToolSearch(openAITools(), "setup")
	found := false
	for _, tool := range matches {
		if tool["name"] == "uiai_agent_card" {
			found = true
		}
	}
	if !found {
		t.Fatal("setup not discoverable through existing tool search")
	}
}

// The agent-style walkthrough uses existing file/config mechanisms, not a new
// privileged configuration API. Everything is an owned fixture, including key
// and upstream; this is not proof of autonomous signup or a real paid query.
func TestSearchSetupExistingReferenceThroughConfigLoader(t *testing.T) {
	clearSearchSetupCredentials(t)
	const envName = "UIAI_SYNTHETIC_CUSTOM_SEARCH_REFERENCE"
	const secret = "UIAI_SYNTHETIC_SEARCH_SETUP_KEY_0123456789"
	t.Setenv(envName, secret)
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.Header.Get("Authorization") != "Bearer "+secret {
			t.Error("existing reference was not resolved into header")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"results":[{"title":"Owned fixture","url":"https://example.com/setup-fixture","content":"Synthetic setup success"}]}`))
	}))
	defer upstream.Close()
	path := filepath.Join(t.TempDir(), "owned-engine-config.yaml")
	writeConfig := func(reference string) {
		t.Helper()
		body := "search:\n  default_provider: tavily\n  providers:\n    tavily:\n      api_url: " + upstream.URL + "/search\n"
		if reference != "" {
			body += "      api_key_env: " + reference + "\n"
		}
		if err := os.WriteFile(path, []byte(body), 0600); err != nil {
			t.Fatal(err)
		}
	}
	load := func() *config.Config {
		t.Helper()
		cfg, err := config.Load(path)
		if err != nil {
			t.Fatal(err)
		}
		return cfg
	}
	writeConfig("")
	initial := setupRouter(load(), NewSearchRuntime())
	guide, _ := readSetup(t, initial, "/api/tools/agent-card")
	if setupAction(t, guide, "tavily") != "connect_existing_credential_reference" {
		t.Fatal("initial setup not actionable")
	}
	fields := guide["provider_setup"].(map[string]any)["tavily"].(map[string]any)["configuration_fields"].([]any)
	if fields[0] != "search.providers.tavily.api_key_env" {
		t.Fatal("guide does not identify canonical configuration field")
	}
	// A separately authorized agent can edit its own deployment's configuration
	// using ordinary tools, then the existing configuration loader re-reads it.
	writeConfig(envName)
	configured := setupRouter(load(), NewSearchRuntime())
	guide, raw := readSetup(t, configured, "/api/tools/agent-card")
	if setupAction(t, guide, "tavily") != "use_selected_provider" || guide["general_web_search_ready"] != true {
		t.Fatal("reference configuration did not become usable")
	}
	if calls.Load() != 0 {
		t.Fatal("setup discovery sent an unsolicited query")
	}
	out := httptest.NewRecorder()
	configured.ServeHTTP(out, httptest.NewRequest(http.MethodPost, "/api/search/", strings.NewReader(`{"query":"synthetic setup fixture `+upstream.URL+`","provider":"tavily","limit":1}`)))
	if (out.Code != 200 && out.Code != 202) || calls.Load() != 1 || !strings.Contains(out.Body.String(), "Synthetic setup success") {
		t.Fatalf("fixture search status=%d upstream_calls=%d content_present=%v", out.Code, calls.Load(), strings.Contains(out.Body.String(), "Synthetic setup success"))
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, secret) || strings.Contains(out.Body.String(), secret) || strings.Contains(string(data), secret) {
		t.Fatal("credential value escaped into configuration or output")
	}
}
