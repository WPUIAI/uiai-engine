package vision

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestMultiPoolSessionStorageIsolationBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for browser isolation regression")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<!doctype html><p>multi-pool fixture</p>"))
	}))
	defer site.Close()
	multi, err := NewMultiPool([]PoolConfig{{MaxPages: 1, AllowPrivateURLs: true}, {MaxPages: 1, AllowPrivateURLs: true}})
	if err != nil {
		t.Fatal(err)
	}
	defer multi.Close()
	manager := NewSessionManagerWithPools(multi)
	// Three acquisitions cycle back to the first pool, not just a different process.
	for acquisition := 0; acquisition < 3; acquisition++ {
		session, _, err := manager.Open(site.URL, 800, 600)
		if err != nil {
			t.Fatal(err)
		}
		requireEmptySessionStorage(t, session.page)
		seedSessionStorage(t, session.page)
		if err = manager.Close(session.ID); err != nil {
			t.Fatal(err)
		}
		multi.mu.Lock()
		remainingOwners := len(multi.owners)
		multi.mu.Unlock()
		if remainingOwners != 0 {
			t.Fatal("multi-pool retained released page ownership")
		}
	}
	if multi.pools[0].browserPID <= 0 || multi.pools[1].browserPID <= 0 || multi.pools[0].browserPID == multi.pools[1].browserPID {
		t.Fatal("multi-pool did not launch two independent browser processes")
	}
	if multi.pools[0].launcher.Get("user-data-dir") == multi.pools[1].launcher.Get("user-data-dir") {
		t.Fatal("multi-pool browsers shared a profile directory")
	}
	for _, pool := range multi.pools {
		pool.mu.Lock()
		active, created, idle := pool.active, pool.created, len(pool.pages)
		pool.mu.Unlock()
		if active != 0 || created != 1 || idle != 1 {
			t.Fatalf("multi-pool capacity mismatch: active=%d created=%d idle=%d", active, created, idle)
		}
	}
	profiles := []string{multi.pools[0].launcher.Get("user-data-dir"), multi.pools[1].launcher.Get("user-data-dir")}
	for _, profile := range profiles {
		info, err := os.Stat(profile)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm()&0o077 != 0 {
			t.Fatal("browser profile is not private to its owning user")
		}
	}
	multi.Close()
	for _, profile := range profiles {
		if _, err := os.Stat(profile); !os.IsNotExist(err) {
			t.Fatal("managed browser profile was not removed by launcher cleanup")
		}
	}

}
