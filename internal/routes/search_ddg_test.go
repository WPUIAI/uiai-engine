package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestDDGInstalledKeylessSearchCapability(t *testing.T) {
	if searchProviderMeta("ddg").search == nil {
		t.Fatal("DDG adapter not installed")
	}
	cfg := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{"brave": {Disabled: true}, "exa": {Disabled: true}, "tavily": {Disabled: true}}}}
	if ready, reason := searchProviderConfigured(cfg, "ddg"); !ready || reason != "" {
		t.Fatalf("DDG keyless admission not available: %s", reason)
	}
	guide := searchSetupGuide(buildSearchProviderInventory(cfg, nil), false)
	if guide["default_provider"] != "ddg" || guide["general_web_search_ready"] != true {
		t.Fatal("clean install did not gain keyless web capability")
	}
	cfg.Search.DefaultProvider = "brave"
	if resolveDefaultSearchProvider(cfg) != "brave" {
		t.Fatal("explicit broken default silently replaced")
	}
	cfg.Search.Providers["ddg"] = config.SearchProviderConfig{Disabled: true}
	if ready, _ := searchProviderConfigured(cfg, "ddg"); ready {
		t.Fatal("disabled DDG admitted")
	}
}

const ddgFixtureHTML = `<!doctype html><html><body><div class="results">
<div class="result result--ad"><a class="result__a" href="https://ad.test/">Advert</a><div class="result"><a class="result__a" href="https://nested-ad.test/">Nested advert</a></div></div>
<div class="result web-result"><h2><a class="result__a" href="//duckduckgo.com/l/?uddg=https%3A%2F%2Fexample.test%2Fguide%3Ftoken%3DSYNTHETIC_LINK%26ok%3Dyes%23PRIVATE_FRAGMENT&amp;rut=tracking">Go &amp; <b>HTML</b></a></h2><a class="result__snippet">Useful <b>parser</b> text. API key: SYNTHETIC_SNIPPET_KEY</a></div>
<div class="result"><a class="result__a" href="javascript:alert(1)">Unsafe</a></div>
<div class="result"><a class="result__a" href="https://user:SYNTHETIC_PASS@example.test/">Userinfo</a></div>
<div class="result"><a class="result__a" href="https://duckduckgo.com/y.js?ad=1">Tracking ad</a></div>
<div class="result"><a class="result__a" href="https://second.test/path">Second</a><div class="result__snippet">Second snippet</div></div>
</div></body></html>`

func TestDDGHTMLProtocolMappingAndNoCredentials(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Method != "GET" || r.URL.Query().Get("q") != "fixture & 日本語" || r.Header.Get("Authorization") != "" || r.Header.Get("x-api-key") != "" || r.Header.Get("Cookie") != "" || r.Header.Get("User-Agent") == "" {
			t.Error("DDG protocol/credential policy changed")
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(ddgFixtureHTML))
	}))
	defer server.Close()
	cfg := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{"ddg": {APIURL: server.URL, APIKeyFile: "/nonexistent-synthetic-credential"}}}}
	if ready, _ := searchProviderConfigured(cfg, "ddg"); !ready {
		t.Fatal("keyless DDG tried to read an unused credential")
	}
	results, err := searchDDG(context.Background(), cfg, "fixture & 日本語", 10)
	if err != nil || len(results) != 2 || calls != 1 {
		t.Fatalf("DDG normalized results lost: %v count%d", err, len(results))
	}
	if results[0].Title != "Go & HTML" || results[0].Source != "DuckDuckGo" || !strings.Contains(results[0].Description, "Useful parser text") || results[1].URL != "https://second.test/path" {
		t.Fatal("DDG mapping or ad/unsafe URL exclusion changed")
	}
	data, _ := json.Marshal(results)
	if strings.Contains(string(data), "SYNTHETIC") || strings.Contains(string(data), "PRIVATE_FRAGMENT") {
		t.Fatal("DDG bypassed normalized redaction")
	}
	one, challenge, err := parseDDGResults([]byte(ddgFixtureHTML), 1)
	if err != nil || challenge || len(one) != 1 {
		t.Fatal("DDG limit lost")
	}
}

func TestDDGChallengeEmptyAndMarkupDrift(t *testing.T) {
	for _, tc := range []struct {
		name, body                string
		status                    int
		wantChallenge, validEmpty bool
	}{
		{"form", `<form id="challenge-form" action="/anomaly.js">SYNTHETIC_PRIVATE_BODY</form>`, 200, true, false},
		{"202", `<p>SYNTHETIC_PRIVATE_BODY</p>`, 202, true, false},
		{"mixed", ddgFixtureHTML + `<div class="result--ad"><form action="/anomaly.js"></form></div>`, 200, true, false},
		{"empty", `<div class="no-results__message">No results found</div>`, 200, false, true},
		{"drift", `<html><body>SYNTHETIC_PRIVATE_BODY</body></html>`, 200, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer server.Close()
			cfg := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{"ddg": {APIURL: server.URL}}}}
			results, err := searchDDG(context.Background(), cfg, "fixture", 2)
			if tc.validEmpty {
				if err != nil || len(results) != 0 {
					t.Fatal("explicit no-results not accepted")
				}
				return
			}
			if err == nil || strings.Contains(err.Error(), "SYNTHETIC_PRIVATE_BODY") {
				t.Fatal("unknown/challenge body became success or leaked")
			}
			var upstream *upstreamSearchError
			if tc.wantChallenge && (!errors.As(err, &upstream) || upstream.setupReason != "provider_challenge" || upstream.status != tc.status) {
				t.Fatal("challenge status/known safe recovery lost")
			}
		})
	}
}

func ddgManagedRequest(runtime *SearchRuntime) *http.Request {
	req := httptest.NewRequest("GET", "https://evidence.example/api/search", nil)
	return req.WithContext(context.WithValue(req.Context(), searchBreakersKey{}, runtime))
}
func TestDDGPacingCacheAndConcurrency(t *testing.T) {
	var calls atomic.Int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); _, _ = w.Write([]byte(ddgFixtureHTML)) }))
	defer server.Close()
	cfg := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{"ddg": {APIURL: server.URL}}}}
	now := time.Now()
	runtime := newSearchBreakers(func() time.Time { return now })
	req := ddgManagedRequest(runtime)
	query := "ddg-cache-and-pacing-unique-" + server.URL
	first, cached, err := executeSearchProvider(req, cfg, "ddg", query, 2, time.Hour)
	if err != nil || cached || len(first) != 2 || first[0].Provider != "ddg" || !strings.HasPrefix(first[0].EvidenceRef, "uiai-search:ddg:") {
		t.Fatal("DDG shared execution/provenance lost")
	}
	if _, cached, err = executeSearchProvider(req, cfg, "ddg", query, 2, time.Hour); err != nil || !cached || calls.Load() != 1 {
		t.Fatal("cached DDG read consumed request pacing or queried")
	}
	_, _, err = executeSearchProvider(req, cfg, "ddg", query+"next", 2, time.Hour)
	var spacing *searchRequestSpacingError
	if !errors.As(err, &spacing) || spacing.retryAfter != 15 || calls.Load() != 1 || runtime.status("ddg").ConsecutiveFailures != 0 {
		t.Fatal("local pacing sent provider traffic or counted upstream failure")
	}
	inv := buildSearchProviderInventory(cfg, runtime)
	for _, p := range inv.providers {
		if p["id"] == "ddg" && (p["status"] != "degraded" || p["degraded_reason"] != "local_rate_limit") {
			t.Fatal("pacing discovery not honest")
		}
	}
	for _, id := range inv.ready {
		if id == "ddg" {
			t.Fatal("paced provider contradicts ready inventory")
		}
	}
	now = now.Add(16 * time.Second)
	var allowed atomic.Int64
	var wg sync.WaitGroup
	for i := 0; i < 12; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if runtime.admitDDGRequest() == nil {
				allowed.Add(1)
			}
		}()
	}
	wg.Wait()
	if allowed.Load() != 1 {
		t.Fatal("concurrent caller pacing admitted more than one request")
	}
}
func TestDDGHTTPChallengeAndLocalSpacingRecovery(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(202)
		_, _ = w.Write([]byte(`<form id="challenge-form">SYNTHETIC_PRIVATE_BODY</form>`))
	}))
	defer server.Close()
	cfg := &config.Config{Search: config.SearchConfig{DefaultProvider: "ddg", Providers: map[string]config.SearchProviderConfig{"ddg": {APIURL: server.URL}}}}
	router := setupRouter(cfg, NewSearchRuntime())
	for i := 0; i < 2; i++ {
		out := httptest.NewRecorder()
		router.ServeHTTP(out, httptest.NewRequest("GET", fmt.Sprintf("/api/search/?q=ddg-challenge-recovery-%d&provider=ddg", i), nil))
		var data map[string]any
		if err := json.Unmarshal(out.Body.Bytes(), &data); err != nil {
			t.Fatal(err)
		}
		if i == 0 {
			if out.Code != 502 || data["upstream_status"] != float64(202) || data["retryable"] != false || !strings.Contains(out.Body.String(), "respect_provider_challenge") {
				t.Fatal("challenge honest failure/recovery lost")
			}
		} else {
			if out.Code != 429 || data["local"] != true || data["upstream_status"] != nil || out.Header().Get("Retry-After") == "" || !strings.Contains(out.Body.String(), "wait_then_retry") {
				t.Fatal("local pacing masqueraded as upstream traffic")
			}
		}
		if strings.Contains(out.Body.String(), "SYNTHETIC_PRIVATE_BODY") {
			t.Fatal("private provider body leaked")
		}
	}
	if calls != 1 {
		t.Fatal("challenge automatically retried")
	}
}
func TestDDGTransportAndEndpointBoundaries(t *testing.T) {
	for _, raw := range []string{"http://outside.test/html/", "https://user:SYNTHETIC_SECRET@html.duckduckgo.com/html/", "https://html.duckduckgo.com/html/?q=SYNTHETIC_SECRET", "https://outside.test/html/", "https://html.duckduckgo.com/html/#private"} {
		cfg := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{"ddg": {APIURL: raw}}}}
		if ready, _ := searchProviderConfigured(cfg, "ddg"); ready {
			t.Fatal("unsafe DDG endpoint ready")
		}
		if _, err := searchDDG(context.Background(), cfg, "fixture", 1); err == nil || strings.Contains(err.Error(), "SYNTHETIC_SECRET") {
			t.Fatal("unsafe endpoint used or disclosed")
		}
	}
	calls := 0
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++ }))
	defer target.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, target.URL, 302) }))
	defer redirect.Close()
	cfg := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{"ddg": {APIURL: redirect.URL}}}}
	if _, err := searchDDG(context.Background(), cfg, "fixture", 1); err == nil || calls != 0 {
		t.Fatal("cross-origin DDG redirect followed")
	}
	huge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("x", maxSearchResponseBytes+1)))
	}))
	defer huge.Close()
	cfg.Search.Providers["ddg"] = config.SearchProviderConfig{APIURL: huge.URL}
	if _, err := searchDDG(context.Background(), cfg, "fixture", 1); err == nil || !strings.Contains(err.Error(), "size limit") {
		t.Fatal("HTML body bound lost")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cfg.Search.Providers["ddg"] = config.SearchProviderConfig{APIURL: target.URL}
	if _, err := searchDDG(ctx, cfg, "fixture", 1); !errors.Is(err, context.Canceled) || calls != 0 {
		t.Fatal("canceled DDG request sent traffic")
	}
}

func TestDDGResearchFailuresKeepLocalAndChallengeMeaning(t *testing.T) {
	local := researchSearchFailure(nil, "ddg", &searchRequestSpacingError{retryAfter: 15})
	if local.Code != "search_request_spacing" || local.UpstreamStatus != 0 || !local.Retryable || local.RetryAfter != "15" || local.Setup == nil || local.Setup.NextAction != "wait_then_retry" {
		t.Fatal("research local request spacing lost identity")
	}
	challenge := researchSearchFailure(nil, "ddg", &upstreamSearchError{status: 202, setupReason: "provider_challenge"})
	if challenge.Retryable || challenge.UpstreamStatus != 202 || challenge.Setup == nil || challenge.Setup.NextAction != "respect_provider_challenge" {
		t.Fatal("research challenge lost identity")
	}
}
func TestDDGChallengeIsNeverCached(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(202)
			_, _ = w.Write([]byte(`<form id="challenge-form"></form>`))
			return
		}
		_, _ = w.Write([]byte(ddgFixtureHTML))
	}))
	defer server.Close()
	cfg := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{"ddg": {APIURL: server.URL}}}}
	now := time.Now()
	state := newSearchBreakers(func() time.Time { return now })
	req := ddgManagedRequest(state)
	query := "never-cache-ddg-challenge-" + server.URL
	if _, _, err := executeSearchProvider(req, cfg, "ddg", query, 1, time.Hour); err == nil {
		t.Fatal("challenge became successful cache candidate")
	}
	now = now.Add(16 * time.Second)
	results, cached, err := executeSearchProvider(req, cfg, "ddg", query, 1, time.Hour)
	if err != nil || cached || len(results) != 1 || calls != 2 {
		t.Fatal("challenge poisoned results cache")
	}
}
func TestDDGApprovedFallbackAndPinnedRouting(t *testing.T) {
	cfg, brave, wiki := fallbackFixture(t)
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); _, _ = w.Write([]byte(ddgFixtureHTML)) }))
	defer server.Close()
	cfg.Search.FallbackProviders = []string{"ddg"}
	cfg.Search.Providers["ddg"] = config.SearchProviderConfig{APIURL: server.URL}
	req := ddgManagedRequest(NewSearchRuntime())
	setCompleteEvidenceScopeHeaders(req, completeEvidenceScope())
	out := httptest.NewRecorder()
	runSearch(out, req, cfg, searchRequest{Query: "ddg-approved-fallback"})
	var body searchResponse
	if err := json.Unmarshal(out.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if out.Code != 200 || body.Provider != "ddg" || calls.Load() != 1 || brave.Load() != 1 || wiki.Load() != 0 {
		t.Fatalf("approved fallback: status=%d provider=%s calls=%d/%d/%d", out.Code, body.Provider, calls.Load(), brave.Load(), wiki.Load())
	}
	out = httptest.NewRecorder()
	runSearch(out, req, cfg, searchRequest{Query: "ddg-pinned-primary", Provider: "brave"})
	if out.Code != 429 || calls.Load() != 1 || wiki.Load() != 0 {
		t.Fatal("pinned query escaped to another provider")
	}
}

func TestDDGDoesNotDisplaceExistingConfiguredProvider(t *testing.T) {
	clearSearchSetupCredentials(t)
	cfg := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{"tavily": {APIKeyEnv: "SYNTHETIC_DDG_POLICY_KEY"}}}}
	t.Setenv("SYNTHETIC_DDG_POLICY_KEY", "synthetic-policy-key")
	if resolveDefaultSearchProvider(cfg) != "tavily" {
		t.Fatal("new keyless fallback displaced configured Tavily")
	}
	cfg.Search.Providers["searxng"] = config.SearchProviderConfig{APIURL: "http://127.0.0.1:8765/search"}
	if resolveDefaultSearchProvider(cfg) != "searxng" {
		t.Fatal("new keyless fallback changed legacy configured-provider ordering")
	}
	cfg.Search.DefaultProvider = "wikipedia"
	if resolveDefaultSearchProvider(cfg) != "wikipedia" {
		t.Fatal("explicit encyclopedia default displaced")
	}
}
