package routes

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"
	"time"

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
