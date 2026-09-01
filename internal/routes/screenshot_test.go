package routes

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/evidenceshare"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
	"github.com/go-rod/rod"
)

type screenshotSharePool struct{}

func (screenshotSharePool) GetPage() (*rod.Page, error)        { return nil, errors.New("unused") }
func (screenshotSharePool) ReleasePage(*rod.Page)              {}
func (screenshotSharePool) ValidateNavigationURL(string) error { return nil }
func (screenshotSharePool) IsBrowserAlive() bool               { return true }
func (screenshotSharePool) MarkFailure()                       {}
func (screenshotSharePool) Reset()                             {}
func (screenshotSharePool) Stats() map[string]any              { return map[string]any{} }
func (screenshotSharePool) Close()                             {}
func (screenshotSharePool) Screenshot(vision.ScreenshotOpts) (*vision.ScreenshotResult, error) {
	return &vision.ScreenshotResult{Data: []byte("automatic-png-evidence"), Width: 375, Height: 812, Format: "png", Duration: 12 * time.Millisecond}, nil
}

func TestScreenshotAutomaticallyReturnsHumanViewableEvidenceShare(t *testing.T) {
	t.Setenv("UIAI_SCREENSHOT_DIR", t.TempDir())
	t.Setenv("UIAI_EVIDENCE_SHARE_DIR", "")
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	router := chi.NewRouter()
	router.Route("/api/screenshot", func(r chi.Router) { MountScreenshotReal(r, cfg, screenshotSharePool{}, nil) })
	request := httptest.NewRequest(http.MethodPost, "https://engine.example/api/screenshot/", bytes.NewBufferString(`{"url":"https://focusa.dev/","width":375,"height":812,"format":"png","focusa_scope":{"workpoint_id":"workpoint:homepage","continuity_id":"focusa-dev-homepage-main"}}`))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("capture status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"artifact_ref", "artifact_path", "artifact_url"} {
		if response[field] == nil || response[field] == "" {
			t.Fatalf("automatic share missing %s: %#v", field, response)
		}
	}
	if got := response["artifact_url"].(string); !strings.HasPrefix(got, "/api/screenshot/share/") || strings.Contains(got, "://") || strings.Contains(got, ":3000") {
		t.Fatalf("non-portable artifact URL: %v", got)
	}
	view := httptest.NewRecorder()
	router.ServeHTTP(view, httptest.NewRequest(http.MethodGet, response["artifact_path"].(string), nil))
	if view.Code != http.StatusOK || !strings.Contains(view.Body.String(), "UIAI Evidence") {
		t.Fatalf("generated share not viewable: %d", view.Code)
	}
}

func TestEvidenceShareRouteServesPortablePackage(t *testing.T) {
	t.Setenv("UIAI_EVIDENCE_SHARE_DIR", "")
	dataDir := t.TempDir()
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: dataDir}}
	share, err := evidenceshare.Assemble(evidenceShareDir(cfg), evidenceshare.Input{Screenshot: []byte("png-bytes"), Format: "png", Width: 375, Height: 812, SourceURL: "https://example.com/page", CapturedAt: time.Date(2026, 8, 30, 1, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatal(err)
	}
	id := filepath.Base(filepath.Clean(share.Directory))
	router := chi.NewRouter()
	mountEvidenceShare(router, cfg)
	for path, mime := range map[string]string{"/share/" + id + "/": "text/html", "/share/" + id + "/styles.css": "text/css", "/share/" + id + "/app.js": "application/javascript", "/share/" + id + "/artifact.json": "application/json", "/share/" + id + "/screenshot.png": "image/png"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, req)
		if recorder.Code != http.StatusOK {
			t.Fatalf("%s status=%d", path, recorder.Code)
		}
		if !strings.HasPrefix(recorder.Header().Get("Content-Type"), mime) {
			t.Fatalf("%s mime=%q", path, recorder.Header().Get("Content-Type"))
		}
		if recorder.Header().Get("X-Content-Type-Options") != "nosniff" {
			t.Fatal("nosniff missing")
		}
	}
	for _, path := range []string{"/share", "/share/" + id, "/share/" + id + "/verify"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("packet API %s status=%d", path, recorder.Code)
		}
		if !strings.Contains(recorder.Body.String(), id) {
			t.Fatalf("packet API %s omitted ID + descriptor payload", path)
		}
	}
	verify := httptest.NewRecorder()
	router.ServeHTTP(verify, httptest.NewRequest(http.MethodGet, "/share/"+id+"/verify", nil))
	if !strings.Contains(verify.Body.String(), `"valid":true`) || !strings.Contains(verify.Body.String(), "Screenshot evidence share packet") {
		t.Fatal("verify response lacks validity or descriptor")
	}
	for _, path := range []string{"/share/" + id + "/../artifact.json", "/share/not-a-digest/", "/share/" + id + "/secret.env"} {
		recorder := httptest.NewRecorder()
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code == http.StatusOK {
			t.Fatalf("unsafe path served: %s", path)
		}
	}
}

func TestArtifactURLUsesPortableRelativeRef(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://share.example/api/screenshot", nil)
	req.Host = "share.example:3000"
	if got := requestArtifactURL(req, "/api/screenshot/share/abc/"); got != "/api/screenshot/share/abc/" {
		t.Fatalf("non-portable URL=%q", got)
	}
}
