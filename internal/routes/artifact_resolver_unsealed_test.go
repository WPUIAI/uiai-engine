package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/evidenceshare"
	"github.com/go-chi/chi/v5"
)

func TestArtifactResolverReportsUnsealedArtifactWithoutPublishingIt(t *testing.T) {
	root := t.TempDir()
	t.Setenv("UIAI_EVIDENCE_SHARE_DIR", root)
	cfg := &config.Config{}
	packet, err := evidenceshare.AssembleGeneric(root, evidenceshare.GenericInput{
		Title: "private title must not escape", Kind: "browser_read", MediaType: "application/json", Extension: "json",
		Payload: []byte(`{"text":"private payload must not escape"}`), CapturedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	mountScreenshotArtifact(router, cfg)
	for _, ref := range []string{packet.ArtifactRef, strings.TrimPrefix(packet.ArtifactRef, "uiai-artifact:sha256:"), url.PathEscape(packet.ArtifactRef)} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/artifact/"+ref, nil))
		if response.Code != http.StatusAccepted {
			t.Fatalf("known unsealed reference must resolve truthfully, not as a retired raw-image request: %d %s", response.Code, response.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		if body["artifact_ref"] != packet.ArtifactRef || body["delivery_state"] != "blocked" || body["recovery_ref"] != "reconcile:epwa-scope-required" {
			t.Fatalf("incorrect degraded envelope: %s", response.Body.String())
		}
		for _, field := range []string{"epwa", "artifact_url", "portable_url", "text", "payload", "scope", "title", "screenshot"} {
			if _, ok := body[field]; ok {
				t.Fatalf("blocked lookup exposed %s", field)
			}
		}
		if strings.Contains(response.Body.String(), "must not escape") {
			t.Fatal("private content escaped")
		}
	}
	// Merely existing on disk is not sufficient: reject changed artifact bytes.
	manifestPath := filepath.Join(packet.Directory, "artifact.json")
	var manifest evidenceshare.GenericManifest
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(packet.Directory, strings.TrimPrefix(manifest.AssetRef, "./")), []byte("corrupted"), 0600); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/artifact/"+url.PathEscape(packet.ArtifactRef), nil))
	if response.Code != http.StatusGone {
		t.Fatalf("corrupted unsealed artifact was recognized as intact: %d", response.Code)
	}
}

func TestArtifactResolverRejectsChangedUnsealedManifest(t *testing.T) {
	root := t.TempDir()
	t.Setenv("UIAI_EVIDENCE_SHARE_DIR", root)
	packet, err := evidenceshare.AssembleGeneric(root, evidenceshare.GenericInput{
		Title: "original", Kind: "browser_read", MediaType: "application/json", Extension: "json",
		Payload: []byte(`{"text":"fixture"}`), CapturedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(packet.Directory, "artifact.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var manifest evidenceshare.GenericManifest
	if err = json.Unmarshal(raw, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.Title = "tampered metadata"
	raw, err = json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(path, append(raw, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	mountScreenshotArtifact(router, &config.Config{})
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/artifact/"+url.PathEscape(packet.ArtifactRef), nil))
	if response.Code != http.StatusGone {
		t.Fatalf("modified manifest accepted: %d %s", response.Code, response.Body.String())
	}
}
