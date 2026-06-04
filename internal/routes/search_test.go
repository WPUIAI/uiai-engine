package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestNormalizeSearchLimit(t *testing.T) {
	if got := normalizeSearchLimit(0); got != defaultSearchLimit {
		t.Fatalf("default limit = %d, want %d", got, defaultSearchLimit)
	}
	if got := normalizeSearchLimit(99); got != maxSearchLimit {
		t.Fatalf("max limit = %d, want %d", got, maxSearchLimit)
	}
	if got := normalizeSearchLimit(3); got != 3 {
		t.Fatalf("explicit limit = %d, want 3", got)
	}
}

func TestSearchBraveMapsWebResults(t *testing.T) {
	t.Setenv("BRAVE_SEARCH_API_KEY", "test-key")
	var sawToken bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Subscription-Token") == "test-key" {
			sawToken = true
		}
		if r.URL.Query().Get("q") != "agent browser" {
			t.Fatalf("query = %q", r.URL.Query().Get("q"))
		}
		if r.URL.Query().Get("count") != "2" {
			t.Fatalf("count = %q", r.URL.Query().Get("count"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"web":{"results":[{"title":"One","url":"https://example.com/one","description":"first","profile":{"name":"example.com"},"age":"1 day"},{"title":"Two","url":"https://example.com/two","description":"second","profile":{"name":"example.com"}}]}}`))
	}))
	defer server.Close()
	t.Setenv("UIAI_BRAVE_SEARCH_API_URL", server.URL)

	results, err := searchBrave("agent browser", 2)
	if err != nil {
		t.Fatalf("searchBrave error: %v", err)
	}
	if !sawToken {
		t.Fatalf("Brave auth header not sent")
	}
	if len(results) != 2 {
		t.Fatalf("len(results) = %d", len(results))
	}
	if results[0].Title != "One" || results[0].URL != "https://example.com/one" || results[0].Source != "example.com" {
		t.Fatalf("unexpected first result: %+v", results[0])
	}
}

func TestSearchBraveRequiresKey(t *testing.T) {
	oldKey, hadKey := os.LookupEnv("BRAVE_SEARCH_API_KEY")
	oldURL, hadURL := os.LookupEnv("UIAI_BRAVE_SEARCH_API_URL")
	_ = os.Unsetenv("BRAVE_SEARCH_API_KEY")
	_ = os.Unsetenv("UIAI_BRAVE_SEARCH_API_URL")
	defer func() {
		if hadKey {
			_ = os.Setenv("BRAVE_SEARCH_API_KEY", oldKey)
		}
		if hadURL {
			_ = os.Setenv("UIAI_BRAVE_SEARCH_API_URL", oldURL)
		}
	}()
	if _, err := searchBrave("missing key", 1); err == nil {
		t.Fatalf("expected missing key error")
	}
}
