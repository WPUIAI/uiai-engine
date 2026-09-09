package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
	"github.com/WPUIAI/uiai-engine/internal/evidenceshare"
	"github.com/go-chi/chi/v5"
)

func TestSettingsRoutesPreserveScopeAndRejectInvalidBodies(t *testing.T) {
	store, err := evidenceshare.NewSettingsStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	mountEvidenceShareSettings(router, store, nil)
	request := func(method, body string) *httptest.ResponseRecorder {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(method, "/settings", strings.NewReader(body)))
		return response
	}
	response := request(http.MethodPut, `{"project_ref":"project:one","workstream_ref":"workstream:one","expected_revision":0,"values":{"image":{"quality":61}}}`)
	if response.Code != 200 {
		t.Fatal(response.Body.String())
	}
	var result map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	scope := result["scope"].(map[string]any)
	if scope["project_ref"] != "project:one" || scope["workstream_ref"] != "workstream:one" {
		t.Fatalf("wrong wire scope: %v", scope)
	}
	reset := request(http.MethodDelete, `{"project_ref":"project:one","workstream_ref":"workstream:one","expected_revision":1}`)
	if reset.Code != 200 {
		t.Fatal(reset.Body.String())
	}
	if err := json.Unmarshal(reset.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result["revision"] != float64(2) || result["scope"].(map[string]any)["workstream_ref"] != "workstream:one" {
		t.Fatal("reset lost revision or scope")
	}
	for _, body := range []string{`{"ProjectRef":"wrong","values":{"image":{"quality":61}}}`, `{"values":{"image":{"quality":"bad"}}}`, `{"values":{}} {}`, `{"workstream_ref":"orphan","values":{}}`} {
		if got := request(http.MethodPut, body); got.Code != 400 {
			t.Fatalf("bad request accepted: %s", body)
		}
	}
	if result := store.Effective(evidenceshare.SettingsScope{}); result.Revision != 0 {
		t.Fatal("project operations changed global settings")
	}
}

func TestSettingsPublicationFailureIsServerErrorWithoutPathLeak(t *testing.T) {
	dir := t.TempDir()
	store, err := evidenceshare.NewSettingsStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "evidence-share-settings.json.tmp"), 0700); err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	mountEvidenceShareSettings(router, store, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/settings", strings.NewReader(`{"project_ref":"project:one","values":{"image":{"quality":61}}}`)))
	if response.Code != 500 || strings.Contains(response.Body.String(), dir) {
		t.Fatalf("wrong storage error boundary: %d %s", response.Code, response.Body.String())
	}
	if store.Effective(evidenceshare.SettingsScope{ProjectRef: "project:one"}).Revision != 0 {
		t.Fatal("failed save changed revision")
	}
}

func TestUnavailableSettingsBlockCaptureRatherThanFallback(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "evidence-share-settings.json"), []byte("malformed"), 0600); err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	MountScreenshotReal(router, &config.Config{Storage: config.StorageConfig{DataDir: dir}}, nil, nil, nil)
	for _, path := range []string{"/settings", "/settings/preview", "/"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"url":"https://example.com"}`)))
		if response.Code != 503 || !strings.Contains(response.Body.String(), "evidence_settings_unavailable") {
			t.Fatalf("silent settings fallback: %d %s", response.Code, response.Body.String())
		}
	}
}

func TestEvidenceShareSettingsRoutesPreviewUpdateConflict(t *testing.T) {
	store, err := evidenceshare.NewSettingsStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	r := chi.NewRouter()
	mountEvidenceShareSettings(r, store, nil)
	preview := httptest.NewRecorder()
	r.ServeHTTP(preview, httptest.NewRequest(http.MethodPost, "/settings/preview", strings.NewReader(`{"project_ref":"project:homepage","values":{"presentation":{"theme":"dark"}}}`)))
	if preview.Code != http.StatusOK {
		t.Fatalf("preview status=%d body=%s", preview.Code, preview.Body.String())
	}
	update := httptest.NewRecorder()
	r.ServeHTTP(update, httptest.NewRequest(http.MethodPut, "/settings", strings.NewReader(`{"project_ref":"project:homepage","expected_revision":0,"values":{"presentation":{"theme":"dark"}}}`)))
	if update.Code != http.StatusOK {
		t.Fatalf("update status=%d body=%s", update.Code, update.Body.String())
	}
	conflict := httptest.NewRecorder()
	r.ServeHTTP(conflict, httptest.NewRequest(http.MethodPut, "/settings", strings.NewReader(`{"project_ref":"project:homepage","expected_revision":0,"values":{"presentation":{"theme":"light"}}}`)))
	if conflict.Code != http.StatusConflict {
		t.Fatalf("conflict status=%d body=%s", conflict.Code, conflict.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(update.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["schema"] != evidenceshare.SettingsSchema {
		t.Fatalf("schema=%v", body["schema"])
	}
}

// openRetentionTestStore commits one live artifact into a temp immutable store
// for the governed retention route test.
func openRetentionTestStore(t *testing.T) (*evidenceartifact.Store, func(), error) {
	t.Helper()
	root := t.TempDir()
	store, _, err := evidenceartifact.OpenStore(evidenceartifact.StoreConfig{
		Root: root, MaxStoreBytes: 64 << 20, MaxArtifacts: 100, MaxAssetBytes: 8 << 20,
		StagingQuarantineAge: time.Hour, GCGrace: time.Hour,
	})
	if err != nil {
		return nil, nil, err
	}
	body, err := os.ReadFile(filepath.Join("..", "evidenceartifact", "testdata", "manifest.golden.json"))
	if err != nil {
		return nil, nil, err
	}
	var manifest evidenceartifact.Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, nil, err
	}
	manifest.Assets[0].Path = "assets/proof.json"
	manifest.Integrity.ManifestSHA256 = ""
	manifest.Integrity.ManifestSHA256, err = evidenceartifact.ComputeManifestSHA256(manifest)
	if err != nil {
		return nil, nil, err
	}
	payload := []byte(`{"probe":"route-retention"}`)
	if _, err := store.Commit(context.Background(), manifest, map[string]io.Reader{manifest.Assets[0].AssetID: bytes.NewReader(payload)}); err != nil {
		return nil, nil, err
	}
	return store, func() {}, nil
}
