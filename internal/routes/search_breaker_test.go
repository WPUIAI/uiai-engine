package routes

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

type searchTestClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *searchTestClock) now() time.Time      { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *searchTestClock) add(d time.Duration) { c.mu.Lock(); defer c.mu.Unlock(); c.t = c.t.Add(d) }
func testSearchBreakers() (*searchBreakers, *searchTestClock) {
	c := &searchTestClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	return newSearchBreakers(c.now), c
}
func failSearchCircuit(b *searchBreakers, id string, err error) {
	for i := 0; i < 2; i++ {
		a, wait := b.acquire(id)
		if wait != 0 {
			panic("unexpected cooldown in test fixture")
		}
		b.complete(id, a, err)
	}
}

func TestSearchBreakerConsecutiveFailuresAndRetryAfter(t *testing.T) {
	for _, status := range []int{408, 429, 500, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			b, c := testSearchBreakers()
			failSearchCircuit(b, "brave", &upstreamSearchError{status: status, retryAfter: "90"})
			s := b.status("brave")
			if s.State != "open" || s.RetryAfterSeconds != 90 || s.ConsecutiveFailures != 2 {
				t.Fatal("breaker or Retry-After floor lost")
			}
			c.add(90 * time.Second)
			a, wait := b.acquire("brave")
			if wait != 0 || !a.probe {
				t.Fatal("expired cooldown did not allow one probe")
			}
			b.complete("brave", a, &upstreamSearchError{status: status})
			if got := b.status("brave").RetryAfterSeconds; got != 60 {
				t.Fatalf("expected exponential60s, got%d", got)
			}
			c.add(60 * time.Second)
			a, _ = b.acquire("brave")
			b.complete("brave", a, nil)
			if b.status("brave").State != "closed" || b.status("brave").ConsecutiveFailures != 0 {
				t.Fatal("successful probe did not reset circuit")
			}
		})
	}
}
func TestSearchBreakerHTTPDateFloorAndBoundedBackoff(t *testing.T) {
	b, c := testSearchBreakers()
	floor := c.now().Add(10 * time.Minute)
	failSearchCircuit(b, "brave", &upstreamSearchError{status: 429, retryAfter: floor.Format(http.TimeFormat)})
	if b.status("brave").RetryAfterSeconds != 600 {
		t.Fatal("HTTP-date quota floor ignored")
	}
	c.add(10 * time.Minute)
	for i := 0; i < 12; i++ {
		a, wait := b.acquire("brave")
		if wait != 0 {
			t.Fatal("test clock did not reach retry")
		}
		b.complete("brave", a, &upstreamSearchError{status: 503})
		delay := b.status("brave").RetryAfterSeconds
		if delay > 300 {
			t.Fatal("local backoff unbounded")
		}
		c.add(time.Duration(delay) * time.Second)
	}
	if b.status("brave").RetryAfterSeconds != 0 {
		t.Fatal("expiry did not converge")
	}
}
func TestSearchBreakerSingleHalfOpenProbeUnderConcurrency(t *testing.T) {
	b, c := testSearchBreakers()
	failSearchCircuit(b, "brave", &upstreamSearchError{status: 503})
	c.add(30 * time.Second)
	var granted atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, wait := b.acquire("brave")
			if wait == 0 {
				granted.Add(1)
			}
		}()
	}
	wg.Wait()
	if granted.Load() != 1 || !b.status("brave").ProbeInFlight {
		t.Fatal("concurrent callers exceeded the single half-open probe")
	}
}
func TestSearchBreakerCancellationAndNonTransientFailures(t *testing.T) {
	for _, err := range []error{context.Canceled, context.DeadlineExceeded, errors.New("value-free configuration failure"), &upstreamSearchError{status: 400}, &upstreamSearchError{status: 401}, &upstreamSearchError{status: 600}} {
		b, _ := testSearchBreakers()
		for i := 0; i < 5; i++ {
			a, _ := b.acquire("brave")
			b.complete("brave", a, err)
		}
		if b.status("brave").State != "closed" {
			t.Fatal("caller/config/non-transient failures opened a circuit")
		}
	}
	b, c := testSearchBreakers()
	failSearchCircuit(b, "brave", &upstreamSearchError{status: 503})
	c.add(30 * time.Second)
	a, _ := b.acquire("brave")
	b.complete("brave", a, context.Canceled)
	if _, wait := b.acquire("brave"); wait != 0 {
		t.Fatal("canceled probe leaked its slot")
	}
}
func TestSearchBreakerStaleCompletionCannotClearNewerCircuit(t *testing.T) {
	b, _ := testSearchBreakers()
	stale, _ := b.acquire("brave")
	failSearchCircuit(b, "brave", &upstreamSearchError{status: 503})
	b.complete("brave", stale, nil)
	if b.status("brave").State != "open" {
		t.Fatal("stale pre-break response cleared a newer circuit")
	}
}
func TestMountedSearchBreakerFallbackPinningMetadataAndIsolation(t *testing.T) {
	cfg, brave, wiki := fallbackFixture(t)
	router := chi.NewRouter()
	MountSearchRoutes(router, cfg)
	call := func(r http.Handler, path string) *httptest.ResponseRecorder {
		out := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "https://evidence.example"+path, nil)
		setCompleteEvidenceScopeHeaders(req, completeEvidenceScope())
		r.ServeHTTP(out, req)
		return out
	}
	for i := 0; i < 3; i++ {
		out := call(router, "/?q=fixture")
		if out.Code != 200 {
			t.Fatalf("approved fallback failed%d", out.Code)
		}
	}
	if brave.Load() != 2 || wiki.Load() != 3 {
		t.Fatal("open primary continued consuming calls or blocked approved fallback")
	}
	pinned := call(router, "/?q=fixture&provider=brave")
	if pinned.Code != 503 || pinned.Header().Get("Retry-After") == "" || brave.Load() != 2 || wiki.Load() != 3 {
		t.Fatal("pinned cooling provider was contacted or bypassed")
	}
	meta := call(router, "/providers")
	var report struct {
		Providers []struct {
			ID         string              `json:"id"`
			Configured bool                `json:"configured"`
			Status     string              `json:"status"`
			Circuit    searchCircuitStatus `json:"circuit"`
		}
	}
	if err := json.NewDecoder(meta.Body).Decode(&report); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, p := range report.Providers {
		if p.ID == "brave" {
			found = true
			if !p.Configured || p.Status != "degraded" || p.Circuit.State != "open" {
				t.Fatal("provider metadata hid transient cooldown")
			}
		}
	}
	if !found {
		t.Fatal("missing primary metadata")
	}
	independent := chi.NewRouter()
	MountSearchRoutes(independent, cfg)
	if out := call(independent, "/?q=fixture&provider=brave"); out.Code != 429 || brave.Load() != 3 {
		t.Fatal("circuit state leaked between independent mounted routers")
	}
}
func TestSearchBreakerCachedHitsDoNotContactCoolingProvider(t *testing.T) {
	cfg, brave, _ := fallbackFixture(t)
	t.Setenv("UIAI_SEARCH_CACHE_TTL_SECONDS", "60")
	b, _ := testSearchBreakers()
	failSearchCircuit(b, "brave", &upstreamSearchError{status: 429})
	query := "breaker-owned-cache-fixture"
	t.Cleanup(func() {
		searchCache.Lock()
		defer searchCache.Unlock()
		delete(searchCache.entries, searchCacheKey("brave", query, 1))
	})
	setCachedSearch("brave", query, 1, []searchResult{{Title: "Cached", URL: "https://example.test/cache"}}, time.Now().Add(time.Minute))
	req := httptest.NewRequest(http.MethodGet, "https://evidence.example/api/search", nil).WithContext(context.WithValue(context.Background(), searchBreakersKey{}, b))
	setCompleteEvidenceScopeHeaders(req, completeEvidenceScope())
	out := httptest.NewRecorder()
	runSearch(out, req, cfg, searchRequest{Query: query, Provider: "brave", Limit: 1})
	var body searchResponse
	if err := json.NewDecoder(out.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if out.Code != 200 || !body.Cached || brave.Load() != 0 {
		t.Fatal("cached hit consumed quota or was blocked by breaker")
	}
}

func TestSearchBreakerFailedRecoveryTransportBacksOff(t *testing.T) {
	b, c := testSearchBreakers()
	failSearchCircuit(b, "brave", &upstreamSearchError{status: 503})
	c.add(30 * time.Second)
	a, _ := b.acquire("brave")
	b.complete("brave", a, errors.New("search provider transport failed"))
	if s := b.status("brave"); s.State != "open" || s.RetryAfterSeconds != 60 || s.ProbeInFlight {
		t.Fatal("failed recovery transport allowed an immediate probe storm")
	}
}
