package routes

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func TestFPVShareRequiresSessionID(t *testing.T) {
	r := chi.NewRouter()
	MountFPVRoutes(r, nil)
	req := httptest.NewRequest(http.MethodPost, "/share", bytes.NewBufferString(`{}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 without session manager, got %d", w.Code)
	}
}

func TestFPVEntryExpires(t *testing.T) {
	fpvShares.Store("expired", &fpvShare{Token: "expired", SessionID: "sid", ExpiresAt: time.Now().UTC().Add(-time.Second)})
	if _, ok := fpvEntry("expired"); ok {
		t.Fatal("expected expired FPV share to be rejected")
	}
}

func TestFPVRegistryPathFromStopsAtFilesystemRoot(t *testing.T) {
	start := filepath.Join(t.TempDir(), "nested", "directory")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, want := fpvRegistryPathFrom(start), filepath.Join("data", "fpv-shares.json"); got != want {
		t.Fatalf("fpvRegistryPathFrom() = %q, want %q", got, want)
	}
}

func TestFPVRegistryPathFromFindsModuleRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.test\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	start := filepath.Join(root, "nested")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, want := fpvRegistryPathFrom(start), filepath.Join(root, "data", "fpv-shares.json"); got != want {
		t.Fatalf("fpvRegistryPathFrom() = %q, want %q", got, want)
	}
}

func TestFPVToken(t *testing.T) {
	token, err := fpvToken()
	if err != nil {
		t.Fatalf("fpvToken returned error: %v", err)
	}
	if !regexp.MustCompile(`^[a-z]+-[a-z]+-[0-9a-f]{4}$`).MatchString(token) {
		t.Fatalf("token %q is not human-friendly slug format", token)
	}
}

func TestFPVFocusaContextDegradesWithoutScope(t *testing.T) {
	ctx := fpvFocusaContext(&vision.Session{}, "abc123")
	if ctx["status"] != "degraded" || ctx["workpoint"] != "unavailable" {
		t.Fatalf("expected degraded focusa context, got %#v", ctx)
	}
}

func TestFPVFocusaContextUsesSessionScope(t *testing.T) {
	sess := &vision.Session{FocusaScope: &vision.FocusaScope{WorkpointID: "wp1", ContinuityID: "cont1", ProjectRoot: "/project", EvidenceRef: "ev1"}}
	ctx := fpvFocusaContext(sess, "abc123")
	if ctx["status"] != "linked" || ctx["workpoint"] != "wp1" || ctx["continuity_id"] != "cont1" || ctx["project_root"] != "/project" {
		t.Fatalf("expected linked focusa context, got %#v", ctx)
	}
	evidence, ok := ctx["evidence"].([]string)
	if !ok || len(evidence) == 0 || evidence[len(evidence)-1] != "ev1" {
		t.Fatalf("expected evidence ref in context, got %#v", ctx["evidence"])
	}
}

func TestFPVContextUsesFocusaScopeAdapter(t *testing.T) {
	sess := &vision.Session{FocusaScope: &vision.FocusaScope{WorkpointID: "wp1", ContinuityID: "cont1", ProjectRoot: "/project", EvidenceRef: "ev1"}}
	ctx := fpvContext(sess)
	focusa, ok := ctx["focusa"].(map[string]any)
	if !ok || focusa["workpoint"] != "wp1" || focusa["status"] != "linked" {
		t.Fatalf("expected fpv context to use focusa scope adapter, got %#v", ctx["focusa"])
	}
}

func TestFPVLiveFocusaContextUsesDaemon(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"completed","rendered_summary":"live focusa ok"}`))
	}))
	defer srv.Close()
	t.Setenv("FOCUSA_DAEMON_URL", srv.URL)
	sess := &vision.Session{FocusaScope: &vision.FocusaScope{WorkpointID: "wp1", ContinuityID: "cont1", ProjectRoot: "/project", EvidenceRef: "ev1"}}
	ctx := fpvFocusaContext(sess, "abc123")
	if ctx["status"] != "live" {
		t.Fatalf("expected live focusa context, got %#v", ctx)
	}
	live, ok := ctx["live"].(map[string]any)
	if !ok || live["status"] != "linked" {
		t.Fatalf("expected linked live context, got %#v", ctx["live"])
	}
	if len(paths) != 2 || paths[0] != "/v1/workpoint/resume" || paths[1] != "/v1/trajectory/view" {
		t.Fatalf("unexpected focusa paths: %#v", paths)
	}
}

func TestFPVCreateSharePayloadIncludesPublicURLAndControls(t *testing.T) {
	share, err := fpvCreateShare("sid-auto", 5, true, false, 0)
	if err != nil {
		t.Fatalf("fpvCreateShare returned error: %v", err)
	}
	if share["controls"] != true || share["mode"] != "control" || share["public_url"] == "" {
		t.Fatalf("unexpected share payload: %#v", share)
	}
	if _, ok := fpvEntry(share["token"].(string)); !ok {
		t.Fatalf("expected share token to resolve: %#v", share)
	}
}

func TestFPVAuditEventsSince(t *testing.T) {
	before := fpvAuditSeq
	logged := fpvRecordAuditEvent("tok1", "sid1", fpvAuditLog{TS: time.Now().UTC(), Action: "message", Message: "hello"})
	if logged.Seq <= before {
		t.Fatalf("expected seq increment, got before=%d logged=%d", before, logged.Seq)
	}
	events, latest := fpvAuditEventsSince(before, 10)
	if latest < logged.Seq || len(events) == 0 || events[len(events)-1].Message != "hello" {
		t.Fatalf("expected audit event, latest=%d events=%#v", latest, events)
	}
	meta := events[len(events)-1].Meta
	if meta["token"] != "tok1" || meta["session_id"] != "sid1" {
		t.Fatalf("expected token/session metadata, got %#v", meta)
	}
}
