package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/observability"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func TestFPVMirrorScreenshotAfterResizeAndScroll20Cycles(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for real mirror capture")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<!doctype html><style>html,body{margin:0}body{height:4000px;background:linear-gradient(#224488,#6688aa)}</style><title>Owned mirror fixture</title>`)
	}))
	defer site.Close()
	pool, err := vision.NewPoolWithConfig(vision.PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := vision.NewSessionManager(pool)
	defer manager.CloseAll()
	session, _, err := manager.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatal(err)
	}
	token := "synthetic-mirror-regression"
	fpvShares.Store(token, &fpvShare{Token: token, SessionID: session.ID, ExpiresAt: time.Now().Add(5 * time.Minute)})
	defer fpvShares.Delete(token)
	router := chi.NewRouter()
	router.Route("/m", func(r chi.Router) { MountFPVPublicRoutes(r, manager) })
	for cycle := 0; cycle < 20; cycle++ {
		width, height := 640+(cycle%3)*16, 480+(cycle%2)*16
		if _, err := session.Resize(width, height); err != nil {
			t.Fatalf("cycle %d resize: %v", cycle, err)
		}
		if _, err := session.ScrollTo(0, cycle*60); err != nil {
			t.Fatalf("cycle %d scroll: %v", cycle, err)
		}
		if _, err := session.Screenshot("jpeg", 60); err != nil {
			t.Fatalf("cycle %d direct capture: %v", cycle, err)
		}
		var viewers sync.WaitGroup
		for viewer := 0; viewer < 2; viewer++ {
			viewers.Add(1)
			go func() {
				defer viewers.Done()
				response := httptest.NewRecorder()
				router.ServeHTTP(response, httptest.NewRequest("GET", "/m/"+token+"/screenshot.jpg", nil))
				if response.Code != 200 || response.Header().Get("Content-Type") != "image/jpeg" {
					t.Errorf("cycle %d mirror status=%d", cycle, response.Code)
					return
				}
				frame, err := jpeg.Decode(bytes.NewReader(response.Body.Bytes()))
				if err != nil {
					t.Errorf("cycle %d invalid JPEG: %v", cycle, err)
					return
				}
				if frame.Bounds().Dx() != width || frame.Bounds().Dy() != height {
					t.Errorf("cycle %d stale viewport %v", cycle, frame.Bounds())
				}
			}()
		}
		viewers.Wait()
	}
	if err := manager.Close(session.ID); err != nil {
		t.Fatal(err)
	}
	failure := httptest.NewRecorder()
	router.ServeHTTP(failure, httptest.NewRequest("GET", "/m/"+token+"/screenshot.jpg", nil))
	if failure.Code != 410 {
		t.Fatalf("closed-session mirror status=%d", failure.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(failure.Body.Bytes(), &body); err != nil || body["error"] == nil {
		t.Fatal("closed mirror lacked structured failure")
	}
	// No active capture errors may be hidden by a passing direct screenshot.
	for _, event := range observability.Recent(100, "browser_session", "") {
		if event.SessionID == session.ID {
			t.Fatalf("mirror emitted error telemetry: %s", event.Class)
		}
	}
}
