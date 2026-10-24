package vision

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

func storageIsolationFixture(t *testing.T, maxPages int) (*Pool, *SessionManager, string) {
	t.Helper()
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for real engine storage isolation")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!doctype html><p>session isolation fixture</p>`))
	}))
	t.Cleanup(site.Close)
	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: maxPages, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool, NewSessionManager(pool), site.URL
}

func seedSessionStorage(t *testing.T, page *rod.Page) {
	t.Helper()
	_, err := page.Timeout(5 * time.Second).Eval(`async () => {
  document.cookie='isolation_probe=synthetic; path=/';
  localStorage.setItem('isolation_probe','synthetic');
  sessionStorage.setItem('isolation_probe','synthetic');
  await new Promise((resolve,reject)=>{
   const request=indexedDB.open('isolation_probe',1);
   request.onupgradeneeded=()=>request.result.createObjectStore('state');
   request.onerror=()=>reject(new Error('fixture database error'));
   request.onsuccess=()=>{const db=request.result;const tx=db.transaction('state','readwrite');tx.objectStore('state').put('synthetic','probe');tx.oncomplete=()=>{db.close();resolve();};tx.onerror=()=>reject(new Error('fixture transaction error'));};
  });
  const cache=await caches.open('isolation_probe');
  await cache.put('/__isolation_probe',new Response('synthetic'));
 }`)
	if err != nil {
		t.Fatal(err)
	}
}

func requireEmptySessionStorage(t *testing.T, page *rod.Page) {
	t.Helper()
	result, err := page.Timeout(5 * time.Second).Eval(`async () => JSON.stringify({cookie:document.cookie,local:localStorage.getItem('isolation_probe'),session:sessionStorage.getItem('isolation_probe'),databases:(await indexedDB.databases()).map(db=>db.name),caches:await caches.keys()})`)
	if err != nil {
		t.Fatal(err)
	}
	var state struct {
		Cookie    string
		Local     *string
		Session   *string
		Databases []string
		Caches    []string
	}
	if err = json.Unmarshal([]byte(result.Value.Str()), &state); err != nil {
		t.Fatal(err)
	}
	if state.Cookie != "" || state.Local != nil || state.Session != nil || len(state.Databases) != 0 || len(state.Caches) != 0 {
		t.Fatal("separate engine sessions shared browser storage")
	}
}

func requireContextDisposed(t *testing.T, browser *rod.Browser, oldID proto.BrowserBrowserContextID) {
	t.Helper()
	contexts, err := (proto.TargetGetBrowserContexts{}).Call(browser)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range contexts.BrowserContextIDs {
		if id == oldID {
			t.Fatal("released session BrowserContext remains alive")
		}
	}
}

func TestEngineSessionsIsolateStorageBrowser(t *testing.T) {
	_, manager, site := storageIsolationFixture(t, 2)
	first, _, err := manager.Open(site, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(first.ID)
	seedSessionStorage(t, first.page)
	second, _, err := manager.Open(site, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(second.ID)
	firstID := first.page.Browser().BrowserContextID
	secondID := second.page.Browser().BrowserContextID
	if firstID == "" || secondID == "" || firstID == secondID {
		t.Fatal("sessions did not receive distinct non-default BrowserContexts")
	}
	if _, deadline := first.page.GetContext().Deadline(); deadline {
		t.Fatal("page creation timeout leaked into session lifetime")
	}
	requireEmptySessionStorage(t, second.page)
	original, err := first.page.Eval(`() => localStorage.getItem('isolation_probe')`)
	if err != nil || original.Value.Str() != "synthetic" {
		t.Fatal("opening another session cleared the first session's storage")
	}
}

func TestReleasedEngineSessionDisposesPriorOriginsAndPopupsBrowser(t *testing.T) {
	pool, manager, site := storageIsolationFixture(t, 1)
	first, _, err := manager.Open(site, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(first.ID)
	seedSessionStorage(t, first.page)
	contextBrowser := first.page.Browser()
	oldID := contextBrowser.BrowserContextID
	popup, err := contextBrowser.Page(proto.TargetCreateTarget{URL: site})
	if err != nil {
		t.Fatal(err)
	}
	popupID := popup.TargetID
	// The original origin is no longer visible when release occurs.
	if err = first.page.Timeout(5 * time.Second).Navigate("about:blank"); err != nil {
		t.Fatal(err)
	}
	if err = manager.Close(first.ID); err != nil {
		t.Fatal(err)
	}
	requireContextDisposed(t, pool.browser, oldID)
	targets, err := (proto.TargetGetTargets{}).Call(pool.browser)
	if err != nil {
		t.Fatal(err)
	}
	for _, target := range targets.TargetInfos {
		if target.TargetID == popupID {
			t.Fatal("released context retained its popup target")
		}
	}
	second, _, err := manager.Open(site, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(second.ID)
	requireEmptySessionStorage(t, second.page)
	if err = manager.Close(second.ID); err != nil {
		t.Fatal(err)
	}
	pool.mu.Lock()
	active, created, idle := pool.active, pool.created, len(pool.pages)
	pool.mu.Unlock()
	if active != 0 || created != 1 || idle != 1 {
		t.Fatalf("pool capacity leaked: active=%d created=%d idle=%d", active, created, idle)
	}
}

func TestIsolationReleaseWakesQueuedSessionBrowser(t *testing.T) {
	pool, manager, site := storageIsolationFixture(t, 1)
	first, _, err := manager.Open(site, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(first.ID)
	seedSessionStorage(t, first.page)
	type outcome struct {
		session *Session
		err     error
	}
	next := make(chan outcome, 1)
	go func() { session, _, err := manager.Open(site, 800, 600); next <- outcome{session, err} }()
	wait := time.NewTimer(3 * time.Second)
	defer wait.Stop()
	tick := time.NewTicker(5 * time.Millisecond)
	defer tick.Stop()
	for atomic.LoadInt64(&pool.queued) == 0 {
		select {
		case <-wait.C:
			t.Fatal("second session did not enter capacity queue")
		case <-tick.C:
		}
	}
	if err = manager.Close(first.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case result := <-next:
		if result.err != nil {
			t.Fatal(result.err)
		}
		defer manager.Close(result.session.ID)
		requireEmptySessionStorage(t, result.session.page)
	case <-time.After(8 * time.Second):
		t.Fatal("isolated release did not wake queued session")
	}
}
