package routes

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func fpvViewerFixture(t *testing.T, token string, max int) {
	t.Helper()
	fpvShares.Store(token, &fpvShare{Token: token, SessionID: "owned-viewer-fixture", ExpiresAt: time.Now().Add(time.Minute), MaxViews: max})
	t.Cleanup(func() {
		fpvShares.Delete(token)
		fpvStreamMu.Lock()
		delete(fpvStreamViewers, token)
		fpvStreamMu.Unlock()
	})
}

func TestFPVViewerCookiesAndStreamLifecycle(t *testing.T) {
	t.Chdir(t.TempDir())
	token := fmt.Sprintf("synthetic-viewer-lifecycle-%d", time.Now().UnixNano())
	fpvViewerFixture(t, token, 0)
	first := httptest.NewRecorder()
	entry, ok := fpvRecordViewer(first, httptest.NewRequest("GET", "/m/"+token, nil), token, true)
	if !ok || entry.ViewerJoins != 1 || entry.Views != 1 || entry.ViewerSince.IsZero() {
		t.Fatal("first accepted page arrival missing")
	}
	cookies := first.Result().Cookies()
	if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteStrictMode || cookies[0].Path != "/m/"+token {
		t.Fatal("viewer cookie is not private/share scoped")
	}
	repeat := httptest.NewRequest("GET", "/m/"+token+"/status", nil)
	repeat.AddCookie(cookies[0])
	entry, ok = fpvRecordViewer(httptest.NewRecorder(), repeat, token, false)
	if !ok || entry.ViewerJoins != 1 || entry.Views != 1 {
		t.Fatal("status poll duplicated page arrival")
	}
	for i := 0; i < 3; i++ {
		if !fpvAcquireViewer(token) {
			t.Fatal("bounded viewer admission failed")
		}
	}
	if fpvAcquireViewer(token) {
		t.Fatal("fourth stream admitted")
	}
	for i := 0; i < 4; i++ {
		fpvReleaseViewer(token, "owned-viewer-fixture")
	}
	if fpvViewerCount(token) != 0 {
		t.Fatal("stream capacity leaked")
	}
	events, _ := fpvAuditEventsFiltered(0, 200, "", fpvShareRef(token))
	joined, left := 0, 0
	for _, event := range events {
		switch event.Action {
		case "fpv_viewer_joined":
			joined++
		case "fpv_viewer_left":
			left++
		}
	}
	if joined != 4 || left != 3 {
		t.Fatal("page and admitted stream lifecycle signals differ")
	}
	data, _ := json.Marshal(events)
	if strings.Contains(string(data), token) {
		t.Fatal("capability leaked into arrival audit")
	}
}

func TestFPVConcurrentArrivalsPreserveAuditAndViews(t *testing.T) {
	t.Chdir(t.TempDir())
	token := fmt.Sprintf("synthetic-concurrent-viewers-%d", time.Now().UnixNano())
	fpvViewerFixture(t, token, 0)
	var workers sync.WaitGroup
	for i := 0; i < 32; i++ {
		workers.Add(2)
		go func() {
			defer workers.Done()
			_, ok := fpvRecordViewer(httptest.NewRecorder(), httptest.NewRequest("GET", "/m/fixture", nil), token, true)
			if !ok {
				t.Error("arrival lost")
			}
		}()
		go func() {
			defer workers.Done()
			_, ok := fpvUpdateShare(token, func(next *fpvShare) bool {
				next.Audit = append(next.Audit, fpvAuditLog{Action: "owned-audit"})
				return true
			})
			if !ok {
				t.Error("audit update lost")
			}
		}()
	}
	workers.Wait()
	entry, ok := fpvEntry(token)
	if !ok || entry.Views != 32 || entry.ViewerJoins != 32 || len(entry.Audit) != 32 {
		t.Fatal("concurrent share updates lost data")
	}
	_, ok = fpvUpdateShare(token, func(next *fpvShare) bool { next.Revoked = true; return true })
	if !ok {
		t.Fatal("revocation failed")
	}
	if _, ok = fpvRecordViewer(httptest.NewRecorder(), httptest.NewRequest("GET", "/m/fixture", nil), token, true); ok {
		t.Fatal("revoked share announced a viewer")
	}
}

func TestFPVViewerDeniedAndFilteredCursors(t *testing.T) {
	t.Chdir(t.TempDir())
	token := fmt.Sprintf("synthetic-bounded-viewer-%d", time.Now().UnixNano())
	fpvViewerFixture(t, token, 1)
	if _, ok := fpvRecordViewer(httptest.NewRecorder(), httptest.NewRequest("GET", "/m/fixture", nil), token, true); !ok {
		t.Fatal("first view rejected")
	}
	events, cursor := fpvAuditEventsFiltered(0, 100, "", fpvShareRef(token))
	if len(events) != 1 {
		t.Fatal("first-view event missing")
	}
	if _, ok := fpvRecordViewer(httptest.NewRecorder(), httptest.NewRequest("GET", "/m/fixture", nil), token, true); ok {
		t.Fatal("view limit ignored")
	}
	fpvRecordAuditEvent("synthetic-unrelated-share", "unrelated-owned-session", fpvAuditLog{Action: "fpv_viewer_joined"})
	events, latest := fpvAuditEventsFiltered(cursor, 1, "", fpvShareRef(token))
	if len(events) != 0 || latest != cursor {
		t.Fatal("unrelated share leaked events or cursor activity")
	}
	// Filtering must happen before limiting, not hide this share behind other traffic.
	events, _ = fpvAuditEventsFiltered(0, 1, "", fpvShareRef(token))
	if len(events) != 1 || events[0].Meta["share_ref"] != fpvShareRef(token) {
		t.Fatal("filter applied after limit")
	}
	fpvShares.Store("synthetic-expired-viewer", &fpvShare{ExpiresAt: time.Now().Add(-time.Second)})
	defer fpvShares.Delete("synthetic-expired-viewer")
	if _, ok := fpvRecordViewer(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil), "synthetic-expired-viewer", true); ok {
		t.Fatal("expired share announced an arrival")
	}
}

func TestFPVViewerHTTPConsumerAndClosedSession(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for actual Chromium session")
	}
	t.Chdir(t.TempDir())
	t.Setenv("FOCUSA_DAEMON_URL", "")
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<!doctype html><title>Owned viewer event fixture</title>")
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
	token := fmt.Sprintf("synthetic-http-viewer-%d", time.Now().UnixNano())
	fpvLoadRegistry()
	fpvViewerFixture(t, token, 0)
	fpvShares.Store(token, &fpvShare{Token: token, SessionID: session.ID, ExpiresAt: time.Now().Add(time.Minute)})
	router := chi.NewRouter()
	router.Route("/m", func(r chi.Router) { MountFPVPublicRoutes(r, manager) })
	router.Get("/api/fpv/events", fpvAuditEventsHandler)
	server := httptest.NewServer(router)
	defer server.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar, Timeout: 5 * time.Second}
	get := func(path string, status int) []byte {
		t.Helper()
		response, err := client.Get(server.URL + path)
		if err != nil {
			t.Fatal("HTTP consumer failed")
		}
		defer response.Body.Close()
		data, _ := io.ReadAll(response.Body)
		if response.StatusCode != status {
			t.Fatalf("HTTP consumer status=%d want%d", response.StatusCode, status)
		}
		return data
	}
	for i := 0; i < 3; i++ {
		data := get("/m/"+token+"/status", 200)
		var body map[string]any
		if json.Unmarshal(data, &body) != nil {
			t.Fatal("invalid status JSON")
		}
		state := body["viewer_state"].(map[string]any)
		if state["viewer_join_count"] != float64(1) || state["viewer_count"] != float64(0) {
			t.Fatal("poll state did not deduplicate arrivals")
		}
		if body["context"].(map[string]any)["viewer_state"] == nil {
			t.Fatal("collaboration context lost viewer state")
		}
	}
	get("/m/"+token, 200)
	events := get("/m/"+token+"/events", 200)
	if strings.Contains(string(events), token) || !strings.Contains(string(events), "fpv_viewer_joined") {
		t.Fatal("public arrival projection unsafe or absent")
	}
	agent := get("/api/fpv/events?session_id="+session.ID, 200)
	if strings.Contains(string(agent), token) || !strings.Contains(string(agent), "fpv_viewer_joined") {
		t.Fatal("agent poll lost redacted session event")
	}
	if err := manager.Close(session.ID); err != nil {
		t.Fatal(err)
	}
	for _, suffix := range []string{"", "/status", "/events"} {
		get("/m/"+token+suffix, 410)
	}
}
