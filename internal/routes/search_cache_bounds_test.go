package routes

import (
	"fmt"
	"testing"
	"time"
)

func isolatedSearchCache(t *testing.T) {
	t.Helper()
	searchCache.Lock()
	old := searchCache.entries
	searchCache.entries = make(map[string]searchCacheEntry)
	searchCache.Unlock()
	t.Cleanup(func() { searchCache.Lock(); searchCache.entries = old; searchCache.Unlock() })
}
func TestSearchCacheHourDefaultAndBoundedBurst(t *testing.T) {
	isolatedSearchCache(t)
	t.Setenv("UIAI_SEARCH_CACHE_TTL_SECONDS", "")
	if searchCacheTTL() != time.Hour {
		t.Fatal("default cache does not satisfy one-hour contract")
	}
	now := time.Now()
	for i := 0; i < maxSearchCacheEntries+32; i++ {
		setCachedSearch("brave", fmt.Sprintf("cache-burst-%d", i), 1, []searchResult{{Title: "fixture"}}, now.Add(time.Hour+time.Duration(i)*time.Second))
	}
	searchCache.Lock()
	count := len(searchCache.entries)
	searchCache.Unlock()
	if count != maxSearchCacheEntries {
		t.Fatal("unique queries grew cache beyond entry budget")
	}
	if _, ok := getCachedSearch("brave", "cache-burst-0", 1, now); ok {
		t.Fatal("earliest expiry not evicted")
	}
	if _, ok := getCachedSearch("brave", fmt.Sprintf("cache-burst-%d", maxSearchCacheEntries+31), 1, now); !ok {
		t.Fatal("new entry not retained")
	}
}
func TestSearchCacheReplacementKeepsOtherEntries(t *testing.T) {
	isolatedSearchCache(t)
	now := time.Now()
	for i := 0; i < maxSearchCacheEntries; i++ {
		setCachedSearch("brave", fmt.Sprint(i), 1, []searchResult{{Title: "before"}}, now.Add(time.Hour))
	}
	setCachedSearch("brave", "0", 1, []searchResult{{Title: "after"}}, now.Add(2*time.Hour))
	for i := 0; i < maxSearchCacheEntries; i++ {
		results, ok := getCachedSearch("brave", fmt.Sprint(i), 1, now)
		if !ok {
			t.Fatal("replacement evicted another query")
		}
		if i == 0 && results[0].Title != "after" {
			t.Fatal("replacement not visible")
		}
	}
}
func TestSearchCacheExpiredEntriesPrunedOnWriteAndHealth(t *testing.T) {
	isolatedSearchCache(t)
	now := time.Now()
	searchCache.Lock()
	searchCache.entries["expired"] = searchCacheEntry{ExpiresAt: now.Add(-time.Second)}
	searchCache.Unlock()
	setCachedSearch("brave", "live", 1, []searchResult{{Title: "live"}}, now.Add(time.Hour))
	searchCache.Lock()
	_, old := searchCache.entries["expired"]
	searchCache.entries["other-expired"] = searchCacheEntry{ExpiresAt: now.Add(-time.Second)}
	searchCache.Unlock()
	if old {
		t.Fatal("write left expired entries resident")
	}
	report := searchPressureSummary(nil)
	if report["cache_entries"] != 1 || report["cache_max_entries"] != maxSearchCacheEntries {
		t.Fatal("health reported stale or unbounded occupancy")
	}
}
