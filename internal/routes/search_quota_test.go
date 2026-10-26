package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

func TestSearchQuotaObservedThroughSharedDiscovery(t *testing.T) {
	clearSearchSetupCredentials(t)
	calls := 0
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.Header().Set("X-RateLimit-Limit", "1, 15000")
		w.Header().Set("X-RateLimit-Policy", "1;w=1, 15000;w=2592000")
		w.Header().Set("X-RateLimit-Remaining", "0, 14523")
		w.Header().Set("X-RateLimit-Reset", "1, 1234567")
		_, _ = w.Write([]byte(`{"web":{"results":[{"title":"Fixture","url":"https://example.test/quota"}]}}`))
	}))
	defer upstream.Close()
	cfg := &config.Config{Search: config.SearchConfig{DefaultProvider: "brave", Providers: map[string]config.SearchProviderConfig{"brave": {APIURL: upstream.URL, APIKey: "synthetic-key"}}}}
	isolateEvidencePublicationStorage(t, cfg)
	now := time.Now()
	state := newSearchBreakers(func() time.Time { return now })
	router := setupRouter(cfg, state)
	out := httptest.NewRecorder()
	router.ServeHTTP(out, httptest.NewRequest("GET", "/api/search/?q=quota-fixture-only-"+upstream.URL+"&provider=brave", nil))
	if out.Code != 200 && out.Code != 202 {
		t.Fatalf("fixture search failed: %d", out.Code)
	}
	guide, _ := readSetup(t, router, "/api/tools/agent-card")
	if guide["remaining_quota_status"] != "unknown" {
		t.Fatal("historical header observation minted current quota")
	}
	observations, ok := guide["quota_observations"].(map[string]any)
	if !ok || observations["brave"].(map[string]any)["current_remaining_status"] != "unknown" {
		t.Fatal("agent card omitted historical quota or invented current budget")
	}
	pressure := searchPressureSummary(cfg, state)
	pressureSeen := false
	for _, p := range pressure["providers"].([]map[string]any) {
		if p["id"] == "brave" {
			q, ok := p["quota"].(searchQuotaProjection)
			pressureSeen = ok && q.Status == "observed_headers" && q.Windows[1].Remaining == 14523
		}
	}
	if !pressureSeen {
		t.Fatal("health and discovery lost shared quota owner")
	}

	out = httptest.NewRecorder()
	router.ServeHTTP(out, httptest.NewRequest("GET", "/api/search/providers", nil))
	var body map[string]any
	if err := json.Unmarshal(out.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range body["providers"].([]any) {
		provider := p.(map[string]any)
		if provider["id"] == "brave" {
			quota, ok := provider["quota"].(map[string]any)
			if !ok || quota["status"] != "observed_headers" {
				t.Fatal("shared discovery lacks validated as-of quota observation")
			}
			found = true
		}
	}
	if !found || calls != 1 {
		t.Fatal("quota discovery triggered traffic or lost provider")
	}
}

func fixtureQuotaHeaders() http.Header {
	return http.Header{"X-Ratelimit-Limit": []string{"1,15000"}, "X-Ratelimit-Policy": []string{"1;w=1,15000;w=2592000"}, "X-Ratelimit-Remaining": []string{"0,14523"}, "X-Ratelimit-Reset": []string{"1,1234567"}}
}
func TestSearchQuotaRejectsMalformedHeadersWithoutDisclosure(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct{ key, value string }{
		{"X-RateLimit-Limit", "SYNTHETIC_PRIVATE_HEADER"},
		{"X-RateLimit-Limit", strings.Repeat("1,", 1024)},
		{"X-RateLimit-Remaining", "-1,14523"},
		{"X-RateLimit-Remaining", "1.5,14523"},
		{"X-RateLimit-Remaining", "2,14523"},
		{"X-RateLimit-Remaining", "0"},
		{"X-RateLimit-Policy", "2;w=1,15000;w=2592000"},
		{"X-RateLimit-Policy", "1;w=0,15000;w=2592000"},
		{"X-RateLimit-Policy", "1;w=1;secret=SYNTHETIC_PRIVATE_HEADER,15000;w=2592000"},
		{"X-RateLimit-Reset", "1,99999999999999999999999"},
		{"X-RateLimit-Reset", "1,999999999"},
		{"X-RateLimit-Reset", ""},
	} {
		h := fixtureQuotaHeaders()
		h.Set(tc.key, tc.value)
		if _, ok := parseBraveSearchQuota(h, 200, now); ok {
			t.Fatal("malformed header accepted as quota observation")
		}
	}
	h := fixtureQuotaHeaders()
	h.Set("X-RateLimit-Limit", "0")
	h.Set("X-RateLimit-Policy", "0;w=2592000")
	h.Set("X-RateLimit-Remaining", "0")
	h.Set("X-RateLimit-Reset", "0")
	observation, ok := parseBraveSearchQuota(h, 429, now)
	if !ok || !observation.windows[0].Unlimited || observation.windows[0].Limit != 0 || observation.status != 429 {
		t.Fatal("documented unlimited plan or response status lost")
	}
	h = fixtureQuotaHeaders()
	h["X-Ratelimit-Limit"] = []string{"1", "15000"}
	if _, ok := parseBraveSearchQuota(h, 200, now); !ok {
		t.Fatal("coherent repeated headers not accepted")
	}
}
func TestSearchQuotaOrderingStalenessAndCloneBoundaries(t *testing.T) {
	now := time.Now()
	runtime := newSearchBreakers(func() time.Time { return now })
	request := func() *http.Request {
		req := ddgManagedRequest(runtime)
		return req.WithContext(runtime.bindQuotaObservation(req.Context(), "brave"))
	}
	old, newer := request(), request()
	h := fixtureQuotaHeaders()
	observeSearchQuota(newer, h, 429)
	h.Set("X-RateLimit-Remaining", "1,14999")
	observeSearchQuota(old, h, 200)
	quota := runtime.quotaProjection("brave")
	if quota.Status != "observed_headers" || quota.ResponseStatus != 429 || quota.Windows[1].Remaining != 14523 || quota.Authority != "advisory_only" || quota.AccountBinding != "not_verified" || quota.CurrentRemainingStatus != "unknown" {
		t.Fatal("old request overwrote observation or observation minted authority")
	}
	quota.Windows[1].Remaining = 123
	if runtime.quotaProjection("brave").Windows[1].Remaining != 14523 {
		t.Fatal("caller mutated runtime quota")
	}
	now = now.Add(2 * time.Second)
	quota = runtime.quotaProjection("brave")
	if !quota.Windows[0].ResetElapsed || quota.Windows[1].ResetElapsed || quota.Windows[0].Remaining != 0 {
		t.Fatal("elapsed reset inferred replenished credits")
	}
	now = now.Add(6 * time.Minute)
	if !runtime.quotaProjection("brave").Stale {
		t.Fatal("old observation not stale")
	}
	now = now.Add(-time.Hour)
	quota = runtime.quotaProjection("brave")
	if !quota.ClockUncertain || !quota.Stale || quota.Windows[0].ResetElapsed {
		t.Fatal("clock uncertainty became current quota")
	}
	if runtime.quotaProjection("ddg").Status != "unknown" || (*SearchRuntime)(nil).quotaProjection("brave").Status != "unknown" {
		t.Fatal("unsupported/unbound quota invented")
	}
}
func TestSearchQuotaDoesNotInferProviderOrRefreshFromCache(t *testing.T) {
	var calls atomic.Int32
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		for key, values := range fixtureQuotaHeaders() {
			for _, v := range values {
				w.Header().Add(key, v)
			}
		}
		_, _ = w.Write([]byte(`{"web":{"results":[{"title":"Fixture","url":"https://example.test/"}]}}`))
	}))
	defer upstream.Close()
	cfg := &config.Config{Search: config.SearchConfig{Providers: map[string]config.SearchProviderConfig{"brave": {APIURL: upstream.URL, APIKey: "synthetic-key"}}}}
	now := time.Now()
	runtime := newSearchBreakers(func() time.Time { return now })
	req := ddgManagedRequest(runtime)
	query := "quota-cache-fixture-" + upstream.URL
	if _, cached, err := executeSearchProvider(req, cfg, "brave", query, 1, time.Hour); err != nil || cached {
		t.Fatal("quota fixture first call failed")
	}
	received := *runtime.quotaProjection("brave").ReceivedAt
	now = now.Add(time.Minute)
	if _, cached, err := executeSearchProvider(req, cfg, "brave", query, 1, time.Hour); err != nil || !cached || calls.Load() != 1 {
		t.Fatal("cached search re-probed provider")
	}
	if !runtime.quotaProjection("brave").ReceivedAt.Equal(received) {
		t.Fatal("cache hit refreshed quota timestamp")
	}
	// Same valid headers without the internal execution binding are ignored.
	other := NewSearchRuntime()
	unbound := ddgManagedRequest(other)
	observeSearchQuota(unbound, fixtureQuotaHeaders(), 200)
	if other.quotaProjection("brave").Status != "unknown" {
		t.Fatal("headers fabricated provider binding")
	}
	wrong := unbound.WithContext(other.bindQuotaObservation(unbound.Context(), "ddg"))
	observeSearchQuota(wrong, fixtureQuotaHeaders(), 200)
	if other.quotaProjection("ddg").Status != "unknown" {
		t.Fatal("Brave headers reinterpreted as other-provider quota")
	}
}
func TestSearchQuotaConcurrentObservationsRemainBounded(t *testing.T) {
	runtime := NewSearchRuntime()
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req := ddgManagedRequest(runtime)
			req = req.WithContext(runtime.bindQuotaObservation(req.Context(), "brave"))
			observeSearchQuota(req, fixtureQuotaHeaders(), 200)
			_ = runtime.quotaProjection("brave")
		}()
	}
	wg.Wait()
	if len(runtime.quotaObservations) != 1 || len(runtime.quotaSequences) != 1 {
		t.Fatal("quota state escaped supported provider bound")
	}
}
