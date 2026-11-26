package vision

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// #294 diagnose slice: the diagnostics snapshot inventories the page's media
// elements and names any present-but-unconsumed video/audio so an agent
// knows what it did not sample or transcribe.
func TestDiagnosticsMediaInventoryAndHint(t *testing.T) {
	if os.Getenv("UIAI_INPUT_TESTS") == "" {
		t.Skip("set UIAI_INPUT_TESTS=1 for browser media diagnostics regression")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `<!doctype html><html><body>
		<video id="v" src="https://media.example.test/clip.mp4"></video>
		<audio id="a" src="/podcast.mp3"></audio>
		</body></html>`)
	}))
	defer site.Close()

	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	sm := NewSessionManager(pool)
	defer sm.CloseAll()

	session, _, err := sm.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	snap := session.DiagnosticsWithOptions(DiagnosticsOptions{Limit: 10, Format: "full"})
	if len(snap.MediaElements) != 2 {
		t.Fatalf("expected 2 media elements, got %d: %+v", len(snap.MediaElements), snap.MediaElements)
	}
	var video, audio *MediaElementInfo
	for i := range snap.MediaElements {
		switch snap.MediaElements[i].Ref {
		case "video[0]":
			video = &snap.MediaElements[i]
		case "audio[0]":
			audio = &snap.MediaElements[i]
		}
	}
	if video == nil || audio == nil {
		t.Fatalf("media refs missing: %+v", snap.MediaElements)
	}
	if !video.CrossOrigin || audio.CrossOrigin {
		t.Fatalf("cross-origin detection wrong: %+v %+v", video, audio)
	}
	if len(snap.MediaIssues) != 2 {
		t.Fatalf("expected one hint per sourced element, got %d: %v", len(snap.MediaIssues), snap.MediaIssues)
	}
	joined := strings.Join(snap.MediaIssues, ";")
	if !strings.Contains(joined, "video[0]") || !strings.Contains(joined, "/video/frames") || !strings.Contains(joined, "/media/understand") {
		t.Fatalf("hints missing the consumption pointers: %v", snap.MediaIssues)
	}
	if !strings.Contains(joined, "cross-origin") {
		t.Fatalf("cross-origin element must be named in its hint: %v", snap.MediaIssues)
	}

	// A page without media carries no inventory and no hints.
	bare := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintf(w, `<!doctype html><html><body><p>text only</p></body></html>`)
	}))
	defer bare.Close()
	if err := session.page.Timeout(20 * time.Second).Navigate(bare.URL); err != nil {
		t.Fatalf("navigate: %v", err)
	}
	_ = session.page.WaitLoad()
	snap2 := session.DiagnosticsWithOptions(DiagnosticsOptions{Limit: 10, Format: "full"})
	if len(snap2.MediaElements) != 0 || len(snap2.MediaIssues) != 0 {
		t.Fatalf("bare page must carry no media inventory: %+v %v", snap2.MediaElements, snap2.MediaIssues)
	}
}
