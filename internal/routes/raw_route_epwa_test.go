package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/evidenceshare"
	"github.com/go-chi/chi/v5"
)

func TestLegacyScreenshotArtifactRouteIsRetired(t *testing.T) {
	cfg := &config.Config{}
	router := chi.NewRouter()
	mountScreenshotArtifact(router, cfg)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/artifact/deadbeefdeadbeef", nil))
	if recorder.Code != http.StatusGone {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "legacy_raw_artifact_removed") {
		t.Fatalf("missing retired-route code: %s", recorder.Body.String())
	}
}

func TestArtifactResolverResolvesIssuedArtifactRef(t *testing.T) {
	shareDir := t.TempDir()
	cfg := &config.Config{}
	cfg.Storage.DataDir = shareDir
	t.Setenv("UIAI_EVIDENCE_SHARE_DIR", shareDir)

	digest := strings.Repeat("ab", 32)
	packageID := digest[:12] + strings.Repeat("cd", 26)
	directory := filepath.Join(shareDir, packageID)
	if err := os.MkdirAll(directory, 0o750); err != nil {
		t.Fatal(err)
	}
	generic := evidenceshare.GenericManifest{
		Schema: evidenceshare.GenericArtifactSchema, ArtifactRef: "uiai-artifact:sha256:" + digest,
		Title: "resolver probe", MediaType: "application/markdown", AssetRef: "assets/screenshot." + digest[:12] + ".md",
		CapturedAt: time.Now().UTC(), Scope: evidenceshare.Scope{ProjectRef: "project:probe", ContinuityRef: "cont:probe"},
	}
	body, err := json.Marshal(generic)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "artifact.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}

	router := chi.NewRouter()
	mountScreenshotArtifact(router, cfg)

	// Full typed ref resolves.
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/artifact/uiai-artifact:sha256:"+digest, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("typed ref status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var payload struct {
		Schema        string         `json:"schema"`
		ArtifactRef   string         `json:"artifact_ref"`
		DeliveryState string         `json:"delivery_state"`
		EPWA          map[string]any `json:"epwa"`
		Posture       string         `json:"posture"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Schema != "uiai.evidence_artifact_resolve.v1" || payload.DeliveryState != "ready" {
		t.Fatalf("resolve payload: %+v", payload)
	}
	if !strings.Contains(payload.EPWA["record_url"].(string), "/e/"+digest[:12]+"/") {
		t.Fatalf("record_url missing short form: %+v", payload.EPWA)
	}

	// Bare digest resolves too.
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/artifact/"+digest, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("bare digest status=%d", recorder.Code)
	}

	// Unknown digest keeps the fail-closed tombstone.
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/artifact/"+strings.Repeat("cd", 32), nil))
	if recorder.Code != http.StatusGone {
		t.Fatalf("unknown digest status=%d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "legacy_raw_artifact_removed") {
		t.Fatalf("missing tombstone code: %s", recorder.Body.String())
	}
}
