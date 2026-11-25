package vision

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// #301 download slice: a page-initiated download lands in the session's
// governed directory, is listed with its digest, is served back bounded, and
// the directory is removed when the session closes.
func TestDownloadsGovernedDirectory(t *testing.T) {
	if os.Getenv("UIAI_INPUT_TESTS") == "" {
		t.Skip("set UIAI_INPUT_TESTS=1 for browser download regression")
	}
	payload := strings.Repeat("engine-download-fixture\n", 200)
	sum := sha256.Sum256([]byte(payload))
	wantDigest := hex.EncodeToString(sum[:])

	mux := http.NewServeMux()
	mux.HandleFunc("/file.txt", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, payload)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `<!doctype html><html><body><a id="dl" href="/file.txt" download>save file</a></body></html>`)
	})
	site := httptest.NewServer(mux)
	defer site.Close()

	root := t.TempDir()
	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	sm := NewSessionManagerWithPoolsOptions(pool, SessionLimits{DownloadsRoot: root})
	defer sm.CloseAll()

	session, _, err := sm.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	// Trigger the download through the page's own link.
	if _, err := session.page.Eval(`() => { document.getElementById('dl').click(); return 'clicked' }`); err != nil {
		t.Fatalf("click download: %v", err)
	}

	deadline := time.Now().Add(15 * time.Second)
	var files []DownloadFile
	for time.Now().Before(deadline) {
		files = session.Downloads()
		if len(files) > 0 {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if len(files) == 0 {
		t.Fatal("download never landed in the governed directory")
	}
	found := false
	for _, f := range files {
		if strings.Contains(f.Name, "file.txt") {
			found = true
			if f.SHA256 != wantDigest {
				t.Fatalf("download digest mismatch: got %s want %s", f.SHA256, wantDigest)
			}
			if f.Bytes != int64(len(payload)) {
				t.Fatalf("download size mismatch: got %d want %d", f.Bytes, len(payload))
			}
		}
	}
	if !found {
		t.Fatalf("file.txt not in download list: %+v", files)
	}

	// Download events observed.
	events := session.DownloadEvents()
	if len(events) == 0 {
		t.Fatal("download events ring must record begin/progress")
	}
	sawCompletion := false
	for _, e := range events {
		if e.State == "completed" {
			sawCompletion = true
		}
	}
	if !sawCompletion {
		t.Fatalf("download completion not observed: %+v", events)
	}

	// Served back bounded.
	data, err := session.DownloadFileBytes(files[0].Name)
	if err != nil {
		t.Fatalf("serve download: %v", err)
	}
	if string(data) != payload {
		t.Fatalf("served bytes differ from the fixture (%d bytes)", len(data))
	}

	// Traversal is refused.
	if _, err := session.DownloadFileBytes("../escape"); err == nil {
		t.Fatal("path traversal must be refused")
	}

	// Session close removes the governed directory.
	dir := session.downloadsDir
	if err := sm.Close(session.ID); err != nil {
		t.Fatalf("close: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(root, "sessions", session.ID)); !os.IsNotExist(statErr) {
		t.Fatalf("governed download dir must be removed on close: %v", statErr)
	}
	_ = dir
}
