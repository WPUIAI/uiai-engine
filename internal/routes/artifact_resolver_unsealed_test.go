package routes

import (
	"bytes"
	"encoding/json"
	"image"
	"image/png"
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

func TestArtifactResolverReportsUnsealedScreenshotWithoutPublishingIt(t *testing.T) {
	for _, mutation := range []string{"none", "partial_scope", "bytes", "manifest", "symlink"} {
		t.Run(mutation, func(t *testing.T) {
			root := t.TempDir()
			t.Setenv("UIAI_EVIDENCE_SHARE_DIR", root)
			var pixels bytes.Buffer
			if err := png.Encode(&pixels, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
				t.Fatal(err)
			}
			var scope evidenceshare.Scope
			if mutation == "partial_scope" {
				scope.ProjectRef = "private-project-must-not-escape"
			}
			packet, err := evidenceshare.Assemble(root, evidenceshare.Input{Screenshot: pixels.Bytes(), Format: "png", Width: 2, Height: 2, SourceURL: "https://private-fixture.invalid/must-not-escape", CapturedAt: time.Date(2026, 10, 27, 0, 0, 0, 0, time.UTC), Scope: scope})
			if err != nil {
				t.Fatal(err)
			}
			if mutation == "bytes" {
				if err := os.WriteFile(filepath.Join(packet.Directory, "screenshot.png"), []byte("corrupted"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if mutation == "manifest" {
				path := filepath.Join(packet.Directory, "artifact.json")
				body, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				var manifest evidenceshare.Manifest
				if err := json.Unmarshal(body, &manifest); err != nil {
					t.Fatal(err)
				}
				manifest.SourceURL = "https://changed-fixture.invalid"
				body, err = json.MarshalIndent(manifest, "", "  ")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, append(body, '\n'), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if mutation == "symlink" {
				path := filepath.Join(packet.Directory, "screenshot.png")
				if err := os.Rename(path, path+".owned-original"); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(path+".owned-original", path); err != nil {
					t.Fatal(err)
				}
			}
			router := chi.NewRouter()
			mountScreenshotArtifact(router, &config.Config{})
			mountEvidenceShare(router, &config.Config{})
			for _, path := range []string{"/artifact/" + packet.ArtifactRef, "/artifact/" + url.PathEscape(packet.ArtifactRef), "/share/" + packet.PackageID} {
				response := httptest.NewRecorder()
				router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
				if mutation != "none" && mutation != "partial_scope" {
					if response.Code != http.StatusNotFound && response.Code != http.StatusGone {
						t.Fatalf("corrupted screenshot resolved: %d %s", response.Code, response.Body.String())
					}
					continue
				}
				if response.Code != http.StatusAccepted {
					t.Fatalf("issued unsealed screenshot handle unresolved: %d %s", response.Code, response.Body.String())
				}
				var body map[string]any
				if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
					t.Fatal(err)
				}
				if body["artifact_ref"] != packet.ArtifactRef || body["delivery_state"] != "blocked" {
					t.Fatalf("incorrect metadata envelope: %s", response.Body.String())
				}
				for _, field := range []string{"epwa", "artifact_url", "portable_url", "screenshot", "scope", "source_url", "title"} {
					if _, exists := body[field]; exists {
						t.Fatalf("private field escaped: %s", field)
					}
				}
				if strings.Contains(response.Body.String(), "must-not-escape") {
					t.Fatal("private source URL escaped")
				}
			}
		})
	}
}

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
