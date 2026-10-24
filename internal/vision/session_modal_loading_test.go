package vision

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAsyncModalLoadingPreservesObservationsAndRefClick(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for the modal regression")
	}
	release := make(chan struct{})
	started := make(chan struct{}, 1)
	var once sync.Once
	finish := func() { once.Do(func() { close(release) }) }
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/config" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			w.(http.Flusher).Flush()
			select {
			case started <- struct{}{}:
			default:
			}
			select {
			case <-release:
				fmt.Fprint(w, `{"ready":true}`)
			case <-r.Context().Done():
			}
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!doctype html><title>Synthetic async modal</title><div id="modal">Loading configuration</div>`)
	}))
	t.Cleanup(site.Close)
	t.Cleanup(finish)
	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	manager := NewSessionManager(pool)
	t.Cleanup(manager.CloseAll)
	session, _, err := manager.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatal(err)
	}
	_, err = session.page.Timeout(2 * time.Second).Eval(`() => {
  window.loading=true;window.syntheticClicks=0;
  window.modalPromise=fetch('/config',{method:'POST'}).then(r=>r.json()).then(()=>{
   document.querySelector('#modal').innerHTML='<button id="launch">Synthetic launch</button>';
   document.querySelector('#launch').addEventListener('click',()=>{
    window.syntheticClicks++;const until=performance.now()+450;while(performance.now()<until){}
   });
   window.loading=false;
  });
  return true;
 }`)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("fixture config request did not start")
	}
	if result, snapshot, err := session.Eval("return window.loading"); err != nil || result != "true" || snapshot == nil {
		t.Fatalf("eval aborted during pending configuration: %v", err)
	}
	if result, snapshot, err := session.EvalAsync("return window.loading", 1000); err != nil || result != "true" || snapshot == nil {
		t.Fatalf("async eval aborted during pending configuration: %v", err)
	}
	if _, err := session.Snapshot(SnapshotOptions{Compact: true}); err != nil {
		t.Fatalf("snapshot aborted during pending configuration: %v", err)
	}
	if _, err := session.Screenshot("jpeg", 60); err != nil {
		t.Fatalf("screenshot aborted during pending configuration: %v", err)
	}
	clue := false
	for _, event := range session.Diagnostics(100, "all", false).Network {
		if strings.HasSuffix(event.URL, "/config") && event.Method == "POST" && event.Status == 202 {
			clue = true
		}
	}
	if !clue {
		t.Fatal("diagnostics lost the accepted configuration-request clue")
	}
	finish()
	if _, err := session.WaitFor("#launch", 3000); err != nil {
		t.Fatal(err)
	}
	snapshot, err := session.Snapshot(SnapshotOptions{Interactive: true, Compact: true})
	if err != nil {
		t.Fatal(err)
	}
	session.StoreRefs(snapshot.Refs)
	reference := ""
	for ref, element := range snapshot.Refs {
		if element.Name == "Synthetic launch" {
			reference = "@" + ref
		}
	}
	if reference == "" {
		t.Fatal("loaded modal button has no snapshot reference")
	}
	selector, err := session.ResolveSelector(reference)
	if err != nil {
		t.Fatal(err)
	}
	_, clickErr := session.Click(selector)
	count, err := session.page.Timeout(2 * time.Second).Eval(`() => window.syntheticClicks`)
	if err != nil {
		t.Fatal(err)
	}
	if clickErr != nil {
		t.Fatalf("ref click reported failure after modal loading (effects=%v): %v", count.Value.Num(), clickErr)
	}
	if count.Value.Num() != 1 {
		t.Fatal("synthetic action did not execute exactly once")
	}
}
