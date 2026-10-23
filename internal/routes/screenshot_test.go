package routes

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
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
	return &vision.ScreenshotResult{Data: append([]byte("\x89PNG\r\n\x1a\n"), []byte("automatic-pixel-evidence")...), Width: 375, Height: 812, Format: "png", Duration: 12 * time.Millisecond}, nil
}

func TestScreenshotAutomaticallyReturnsHumanViewableEvidenceShare(t *testing.T) {
	t.Setenv("UIAI_SCREENSHOT_DIR", t.TempDir())
	t.Setenv("UIAI_EVIDENCE_SHARE_DIR", "")
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	router := chi.NewRouter()
	MountShortShare(router, cfg)
	router.Route("/api/screenshot", func(r chi.Router) { MountScreenshotReal(r, cfg, screenshotSharePool{}, nil, nil) })
	requestBody, err := json.Marshal(map[string]any{
		"url": "https://focusa.dev/", "width": 375, "height": 812, "format": "png",
		"focusa_scope": map[string]string{"workpoint_id": "workpoint:homepage", "continuity_id": "focusa-dev-homepage-main"},
		"evidence_scope": evidenceshare.Scope{
			ProjectRef: "project:focusa", WorkstreamRef: "workstream:epwa", WorksetRef: "workset:t08",
			CallGraphRef: "callgraph:133", WorkpointRef: "workpoint:t08b", WorkItemRef: "work-item:screenshot-share",
			WorkItems: []evidencepwa.WorkItemProjection{{
				ProviderSurface: "github", WorkItemRef: "work-item:screenshot-share", ItemID: "153", ItemType: "task",
				Title: "Bind screenshot shares", Description: "Project the exact evidence work item.", DescriptionState: evidencepwa.WorkItemDescriptionVisible,
				Revision: "revision:1", Digest: strings.Repeat("d", 64), RevisionState: evidencepwa.WorkItemRevisionCurrent,
				StatusAtCapture: "in_progress", ClosurePosture: "evidence_pending",
			}},
			ContinuityRef: "epwa-t08b",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "https://engine.example/api/screenshot/", bytes.NewReader(requestBody))
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("capture status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"artifact_ref", "artifact_url", "portable_url", "epwa_delivery"} {
		if response[field] == nil || response[field] == "" {
			t.Fatalf("automatic EPWA delivery missing %s: %#v", field, response)
		}
	}
	if response["screenshot"] != nil || response["artifact_path"] != nil {
		t.Fatalf("raw-only delivery fields escaped the EPWA gate: %#v", response)
	}
	viewerURL := response["artifact_url"].(string)
	if !strings.HasPrefix(viewerURL, "https://engine.example/e/") || !strings.HasSuffix(viewerURL, "/") {
		t.Fatalf("artifact URL is not a friendly HTTPS EPWA webpage: %v", viewerURL)
	}
	portableURL := response["portable_url"].(string)
	if !strings.HasSuffix(portableURL, ".zip") || strings.TrimSuffix(portableURL, ".zip") != strings.TrimSuffix(viewerURL, "/") {
		t.Fatalf("portable EPWA URL missing: %v (viewer %v)", portableURL, viewerURL)
	}
	view := httptest.NewRecorder()
	router.ServeHTTP(view, httptest.NewRequest(http.MethodGet, mustPath(t, viewerURL), nil))
	if view.Code != http.StatusOK || !strings.Contains(view.Body.String(), "UIAI <b>×</b> Focusa") || !strings.Contains(view.Body.String(), `data-default-view="record"`) {
		t.Fatalf("generated share not viewable as its bound record: %d", view.Code)
	}
	projectionPath := strings.TrimSuffix(mustPath(t, viewerURL), "/") + "/projection.json"
	projection := httptest.NewRecorder()
	router.ServeHTTP(projection, httptest.NewRequest(http.MethodGet, projectionPath, nil))
	if projection.Code != http.StatusOK || !strings.Contains(projection.Body.String(), evidencepwa.ProjectionSchema) {
		t.Fatalf("canonical projection not served: %d %s", projection.Code, projection.Body.String())
	}
}

func TestScreenshotEPWADeliveryCannotBeDisabledAndMissingScopeIsBlocked(t *testing.T) {
	dataDir := t.TempDir()
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: dataDir}}
	settings, err := evidenceshare.NewSettingsStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := settings.Update(evidenceshare.SettingsScope{}, 0, map[string]any{"enablement": map[string]any{"auto_screenshot": false}}); err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	MountShortShare(router, cfg)
	router.Route("/api/screenshot", func(r chi.Router) { MountScreenshotReal(r, cfg, screenshotSharePool{}, nil, nil) })
	body, _ := json.Marshal(map[string]any{"url": "https://example.test", "format": "png", "inline": true})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "https://engine.example/api/screenshot/", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("blocked capture status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["delivery_state"] != "blocked" || response["artifact_url"] != nil || response["portable_url"] != nil || response["screenshot"] != nil || response["inline_posture"] != "withheld_by_mandatory_epwa_delivery" {
		t.Fatalf("mandatory blocked EPWA posture missing: %#v", response)
	}
	delivery := response["epwa_delivery"].(map[string]any)
	binding := delivery["epwa"].(map[string]any)
	packageID := binding["package_id"].(string)
	blockedViewer := httptest.NewRecorder()
	router.ServeHTTP(blockedViewer, httptest.NewRequest(http.MethodGet, "/e/"+packageID[:shortShareIDLength]+"/", nil))
	if blockedViewer.Code != http.StatusNotFound {
		t.Fatalf("unscoped package was exposed: status=%d", blockedViewer.Code)
	}
}

func TestScreenshotFailsClosedWithoutCanonicalHTTPSBase(t *testing.T) {
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	router := chi.NewRouter()
	MountShortShare(router, cfg)
	router.Route("/api/screenshot", func(r chi.Router) { MountScreenshotReal(r, cfg, screenshotSharePool{}, nil, nil) })
	body, _ := json.Marshal(map[string]any{"url": "https://example.test", "format": "png", "evidence_scope": completeEvidenceScope()})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/screenshot/", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(recorder, request)
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusAccepted || response["delivery_state"] != "pending_reconcile" || response["artifact_url"] != nil || response["portable_url"] != nil || response["screenshot"] != nil {
		t.Fatalf("non-HTTPS delivery did not fail closed: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestPendingEPWADeliveryReconcilesToStableHTTPSIdentity(t *testing.T) {
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	router := chi.NewRouter()
	MountShortShare(router, cfg)
	router.Route("/api/screenshot", func(r chi.Router) { MountScreenshotReal(r, cfg, screenshotSharePool{}, nil, nil) })
	body, _ := json.Marshal(map[string]any{"url": "https://example.test", "format": "png", "evidence_scope": completeEvidenceScope()})
	capture := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/screenshot/", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(capture, request)
	if capture.Code != http.StatusAccepted {
		t.Fatalf("pending capture status=%d body=%s", capture.Code, capture.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(capture.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	delivery := response["epwa_delivery"].(map[string]any)
	deliveryID := delivery["delivery_id"].(string)
	digest := strings.TrimPrefix(deliveryID, "uiai-epwa-delivery:sha256:")
	t.Setenv("UIAI_EPWA_PUBLIC_BASE_URL", "https://evidence.example/engine/")
	reconciled := httptest.NewRecorder()
	router.ServeHTTP(reconciled, httptest.NewRequest(http.MethodPost, "/api/screenshot/delivery/"+digest+"/reconcile", nil))
	if reconciled.Code != http.StatusOK {
		t.Fatalf("reconcile status=%d body=%s", reconciled.Code, reconciled.Body.String())
	}
	var ready map[string]any
	if err := json.Unmarshal(reconciled.Body.Bytes(), &ready); err != nil {
		t.Fatal(err)
	}
	if ready["delivery_id"] != deliveryID || ready["revision"] != float64(2) || ready["state"] != "ready" {
		t.Fatalf("reconciliation changed identity or missed revision: %#v", ready)
	}
	epwa := ready["epwa"].(map[string]any)
	if !strings.HasPrefix(epwa["record_url"].(string), "https://evidence.example/engine/e/") || !strings.HasSuffix(epwa["portable_url"].(string), ".zip") {
		t.Fatalf("reconciliation omitted canonical HTTPS EPWA URLs: %#v", epwa)
	}
	replayed := httptest.NewRecorder()
	router.ServeHTTP(replayed, httptest.NewRequest(http.MethodGet, "/api/screenshot/delivery/"+digest, nil))
	if replayed.Code != http.StatusOK || !strings.Contains(replayed.Body.String(), `"revision":2`) {
		t.Fatalf("delivery replay failed: status=%d body=%s", replayed.Code, replayed.Body.String())
	}
}

func TestEPWAReconcileDetectsCorruptPortablePackage(t *testing.T) {
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	router := chi.NewRouter()
	MountShortShare(router, cfg)
	router.Route("/api/screenshot", func(r chi.Router) { MountScreenshotReal(r, cfg, screenshotSharePool{}, nil, nil) })
	body, _ := json.Marshal(map[string]any{"url": "https://example.test", "format": "png", "evidence_scope": completeEvidenceScope()})
	capture := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "https://evidence.example/api/screenshot/", bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(capture, request)
	var response map[string]any
	if err := json.Unmarshal(capture.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	delivery := response["epwa_delivery"].(map[string]any)
	binding := delivery["epwa"].(map[string]any)
	packageID := binding["package_id"].(string)
	if err := os.WriteFile(filepath.Join(evidenceShareDir(cfg), packageID+".zip"), []byte("corrupt"), 0o640); err != nil {
		t.Fatal(err)
	}
	digest := strings.TrimPrefix(delivery["delivery_id"].(string), "uiai-epwa-delivery:sha256:")
	reconciled := httptest.NewRecorder()
	router.ServeHTTP(reconciled, httptest.NewRequest(http.MethodPost, "https://evidence.example/api/screenshot/delivery/"+digest+"/reconcile", nil))
	if reconciled.Code != http.StatusConflict || !strings.Contains(reconciled.Body.String(), `"state":"corrupt"`) || strings.Contains(reconciled.Body.String(), `"record_url":"https://`) {
		t.Fatalf("corrupt package did not fail closed: status=%d body=%s", reconciled.Code, reconciled.Body.String())
	}
}

func TestLegacyVisualProducerUsesMandatoryEPWAEnvelope(t *testing.T) {
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	pixels := append([]byte("\x89PNG\r\n\x1a\n"), []byte("legacy visual pixels")...)
	scope := completeEvidenceScope()
	request := httptest.NewRequest(http.MethodPost, "https://evidence.example/api/vision/capture", nil)
	request.Header.Set("X-UIAI-Project-Ref", scope.ProjectRef)
	request.Header.Set("X-UIAI-Workstream-Ref", scope.WorkstreamRef)
	request.Header.Set("X-UIAI-Workset-Ref", scope.WorksetRef)
	request.Header.Set("X-UIAI-CallGraph-Ref", scope.CallGraphRef)
	request.Header.Set("X-UIAI-Workpoint-Ref", scope.WorkpointRef)
	request.Header.Set("X-UIAI-Work-Item-Ref", scope.WorkItemRef)
	request.Header.Set("X-UIAI-Continuity-Ref", scope.ContinuityRef)
	workItems, _ := json.Marshal(scope.WorkItems)
	request.Header.Set("X-UIAI-Work-Items", string(workItems))
	recorder := httptest.NewRecorder()
	writeLegacyVisualEPWA(recorder, request, cfg, pixels, "png", 320, 700, "https://example.test", map[string]any{"screenshot": "forbidden", "analysis": map[string]any{"ok": true}})
	if recorder.Code != http.StatusCreated || strings.Contains(recorder.Body.String(), `"screenshot"`) || !strings.Contains(recorder.Body.String(), `"delivery_state":"ready"`) || !strings.Contains(recorder.Body.String(), `"artifact_url":"https://`) {
		t.Fatalf("legacy visual escaped mandatory EPWA: status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestShareThumbnailConfinesRasterPreview(t *testing.T) {
	t.Setenv("UIAI_EPWA_PUBLIC_BASE_URL", "https://evidence.example/engine/")
	req := httptest.NewRequest(http.MethodGet, "https://evidence.example/share", nil)
	for _, ref := range []string{"../secret.png", "/private/file.png", "a/../../secret.png", "https://other.example/image.png", "a\\b.png"} {
		if got := shareThumbnailURL(req, "/api/screenshot/share/package/", ref, "image/png"); got != "" {
			t.Errorf("unsafe preview %q: %s", ref, got)
		}
	}
	if got := shareThumbnailURL(req, "/api/screenshot/share/package/", "./image.svg", "image/svg+xml"); got != "" {
		t.Fatal("active image type exposed as thumbnail")
	}
	if got := shareThumbnailURL(req, "/api/screenshot/share/package/", "./image.png", "image/png"); got != "https://evidence.example/engine/api/screenshot/share/package/image.png" {
		t.Fatalf("preview is not package-bound: %s", got)
	}
}

func TestEvidenceShareRouteServesPortablePackage(t *testing.T) {
	t.Setenv("UIAI_EVIDENCE_SHARE_DIR", "")
	dataDir := t.TempDir()
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: dataDir}}
	share, err := evidenceshare.Assemble(evidenceShareDir(cfg), evidenceshare.Input{Screenshot: append([]byte("\x89PNG\r\n\x1a\n"), []byte("bounded-route-pixels")...), Format: "png", Width: 375, Height: 812, SourceURL: "https://example.com/page", CapturedAt: time.Date(2026, 8, 30, 1, 0, 0, 0, time.UTC), Scope: completeEvidenceScope()})
	if err != nil {
		t.Fatal(err)
	}
	id := filepath.Base(filepath.Clean(share.Directory))
	router := chi.NewRouter()
	mountEvidenceShare(router, cfg)
	for path, mime := range map[string]string{"/share/" + id + "/": "text/html", "/share/" + id + "/styles.css": "text/css", "/share/" + id + "/work-items.js": "application/javascript", "/share/" + id + "/locale.js": "application/javascript", "/share/" + id + "/generic-record.js": "application/javascript", "/share/" + id + "/pwa.js": "application/javascript", "/share/" + id + "/app.js": "application/javascript", "/share/" + id + "/manifest.webmanifest": "application/manifest+json", "/share/" + id + "/icon.svg": "image/svg+xml", "/share/" + id + "/sw.js": "application/javascript", "/share/" + id + "/artifact.json": "application/json", "/share/" + id + "/inspection.json": "application/json", "/share/" + id + "/screenshot.png": "image/png", "/share/" + id + "/portable.zip": "application/zip"} {
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
		if strings.HasSuffix(path, "/") && recorder.Header().Get("Content-Security-Policy") != evidenceShareCSP {
			t.Fatal("portable evidence page CSP missing or changed")
		}
		if strings.HasSuffix(path, "/sw.js") && recorder.Header().Get("Cache-Control") != "no-cache" {
			t.Fatal("service worker update policy is not no-cache")
		}
	}
	for _, path := range []string{"/share", "/share/" + id, "/share/" + id + "/verify"} {
		recorder := httptest.NewRecorder()
		requestTarget := path
		if path == "/share" {
			requestTarget = "https://evidence.example" + path
		}
		router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, requestTarget, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("packet API %s status=%d", path, recorder.Code)
		}
		if path == "/share" {
			var list struct {
				Packets []struct {
					ThumbnailURL string `json:"thumbnail_url"`
				} `json:"packets"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &list); err != nil || len(list.Packets) != 1 {
				t.Fatalf("invalid packet index: %s", recorder.Body.String())
			}
			if !strings.HasSuffix(list.Packets[0].ThumbnailURL, "/screenshot.png") || !strings.HasPrefix(list.Packets[0].ThumbnailURL, "https://") || !strings.Contains(list.Packets[0].ThumbnailURL, "/e/") {
				t.Fatalf("missing friendly thumbnail: %s", list.Packets[0].ThumbnailURL)
			}
		}
		if !strings.Contains(recorder.Body.String(), id) {
			t.Fatalf("packet API %s omitted ID + descriptor payload", path)
		}
	}
	verify := httptest.NewRecorder()
	router.ServeHTTP(verify, httptest.NewRequest(http.MethodGet, "/share/"+id+"/verify", nil))
	if !strings.Contains(verify.Body.String(), `"valid":true`) || !strings.Contains(verify.Body.String(), "EPWA evidence package") {
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

func TestArtifactURLRequiresCanonicalHTTPSAndPreservesConfiguredSubpath(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "https://share.example/api/screenshot", nil)
	req.Host = "internal.example:3000"
	if got := requestArtifactURL(req, "/api/screenshot/share/abc/"); got != "https://share.example/api/screenshot/share/abc/" {
		t.Fatalf("HTTPS URL=%q", got)
	}
	t.Setenv("UIAI_EPWA_PUBLIC_BASE_URL", "https://portable.example/engine/")
	if got := requestArtifactURL(req, "/api/screenshot/share/abc/"); got != "https://portable.example/engine/api/screenshot/share/abc/" {
		t.Fatalf("configured subpath URL=%q", got)
	}
}

func completeEvidenceScope() evidenceshare.Scope {
	return evidenceshare.Scope{
		ProjectRef: "project:focusa", WorkstreamRef: "workstream:epwa", WorksetRef: "workset:t08",
		CallGraphRef: "callgraph:133", WorkpointRef: "workpoint:t08b", WorkItemRef: "work-item:screenshot-share",
		WorkItems: []evidencepwa.WorkItemProjection{{
			ProviderSurface: "github", WorkItemRef: "work-item:screenshot-share", ItemID: "153", ItemType: "task",
			Title: "Bind screenshot shares", Description: "Project the exact evidence work item.", DescriptionState: evidencepwa.WorkItemDescriptionVisible,
			Revision: "revision:1", Digest: strings.Repeat("d", 64), RevisionState: evidencepwa.WorkItemRevisionCurrent,
			StatusAtCapture: "in_progress", ClosurePosture: "evidence_pending",
		}},
		ContinuityRef: "epwa-t08b",
	}
}

func setCompleteEvidenceScopeHeaders(req *http.Request, scope evidenceshare.Scope) {
	req.Header.Set("X-UIAI-Project-Ref", scope.ProjectRef)
	req.Header.Set("X-UIAI-Workstream-Ref", scope.WorkstreamRef)
	req.Header.Set("X-UIAI-Workset-Ref", scope.WorksetRef)
	req.Header.Set("X-UIAI-CallGraph-Ref", scope.CallGraphRef)
	req.Header.Set("X-UIAI-Workpoint-Ref", scope.WorkpointRef)
	req.Header.Set("X-UIAI-Work-Item-Ref", scope.WorkItemRef)
	req.Header.Set("X-UIAI-Continuity-Ref", scope.ContinuityRef)
	workItems, _ := json.Marshal(scope.WorkItems)
	req.Header.Set("X-UIAI-Work-Items", string(workItems))
}

func mustPath(t *testing.T, value string) string {
	t.Helper()
	request, err := http.NewRequest(http.MethodGet, value, nil)
	if err != nil {
		t.Fatal(err)
	}
	return request.URL.Path
}

// These TLS servers prove package portability, not production HTTPS delivery.
func TestEvidenceShareRelocatesAcrossHTTPSOrigins(t *testing.T) {
	t.Setenv("UIAI_EVIDENCE_SHARE_DIR", "")
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	originalRouter := chi.NewRouter()
	MountShortShare(originalRouter, cfg)
	originalRouter.Route("/api/screenshot", func(r chi.Router) { mountEvidenceShare(r, cfg) })
	original := httptest.NewTLSServer(originalRouter)
	defer original.Close()
	t.Setenv("UIAI_EPWA_PUBLIC_BASE_URL", original.URL) // no trailing slash
	delivery, err := publishScreenshotEPWA(nil, cfg, evidenceshare.Input{
		Screenshot: append([]byte("\x89PNG\r\n\x1a\n"), []byte("portable-fixture-pixels")...),
		Format:     "png", Width: 375, Height: 812, SourceURL: "https://example.com/source",
		CapturedAt: time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC), Scope: completeEvidenceScope(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(delivery.EPWA.RecordURL, original.URL+"/e/") {
		t.Fatalf("publication did not use configured HTTPS origin: %q", delivery.EPWA.RecordURL)
	}
	readResource := func(client *http.Client, target string) []byte {
		t.Helper()
		response, err := client.Get(target)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		body, err := io.ReadAll(io.LimitReader(response.Body, 8<<20))
		if err != nil || response.StatusCode != http.StatusOK {
			t.Fatalf("resource %s: status=%d read=%v", target, response.StatusCode, err)
		}
		return body
	}
	resources := []string{"", "artifact.json", "inspection.json", "screenshot.png", "portable.zip",
		"app.js", "styles.css", "locale.js", "work-items.js", "generic-record.js", "pwa.js", "sw.js", "manifest.webmanifest", "icon.svg"}
	baseline := make(map[string][]byte, len(resources))
	for _, resource := range resources {
		baseline[resource] = readResource(original.Client(), delivery.EPWA.RecordURL+resource)
		if bytes.Contains(baseline[resource], []byte(original.URL)) {
			t.Fatalf("package embeds installation origin in %q", resource)
		}
	}
	copyCfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	if err := os.CopyFS(evidenceShareDir(copyCfg), os.DirFS(evidenceShareDir(cfg))); err != nil {
		t.Fatal(err)
	}
	copyRouter := chi.NewRouter()
	MountShortShare(copyRouter, copyCfg)
	for _, prefix := range []string{"", "/tenant/deep/install"} {
		copyRouter.Route(prefix+"/api/screenshot", func(r chi.Router) { mountEvidenceShare(r, copyCfg) })
	}
	copied := httptest.NewTLSServer(copyRouter)
	defer copied.Close()
	if copied.URL == original.URL {
		t.Fatal("test requires distinct HTTPS origins")
	}
	original.Close() // relocated resources must not depend on the publishing server
	for _, prefix := range []string{"", "/tenant/deep/install"} {
		for _, slash := range []string{"", "/"} {
			t.Setenv("UIAI_EPWA_PUBLIC_BASE_URL", copied.URL+prefix+slash)
			base, err := canonicalEPWABase(nil)
			if err != nil {
				t.Fatal(err)
			}
			// Friendly URLs live at the host root (/e/{short}/); subpath
			// deployments keep the canonical share path, which is what this
			// half of the test proves (configured base + subpath preservation).
			recordURL := base.ResolveReference(&url.URL{Path: "api/screenshot/share/" + delivery.EPWA.PackageID + "/"}).String()
			if !strings.HasPrefix(recordURL, copied.URL+prefix+"/api/screenshot/share/") {
				t.Fatalf("configured subpath lost: %s", recordURL)
			}
			for _, resource := range resources {
				if got := readResource(copied.Client(), recordURL+resource); !bytes.Equal(got, baseline[resource]) {
					t.Fatalf("relocated resource changed: prefix=%q slash=%q resource=%q", prefix, slash, resource)
				}
			}
		}
	}
}
