package routes

import (
	"bytes"
	"net/http"
	"net/http/httptest"
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
