package routes

import (
	"encoding/json"
	"fmt"
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
	packet, err := evidenceshare.AssembleGeneric(shareDir, evidenceshare.GenericInput{
		ArtifactRef: "uiai-artifact:sha256:" + digest, Revision: 1,
		Title: "resolver probe", Kind: "generated_report", MediaType: "application/json",
		Extension: "json", Payload: []byte("{\"verified\":true}\n"),
		CapturedAt: time.Date(2026, 9, 4, 1, 2, 3, 0, time.UTC), Scope: completeEvidenceScope(),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Hash order is not recency: unrelated directories must not hide this packet.
	for i := 0; i < 101; i++ {
		name := strings.Repeat("f", 60) + fmt.Sprintf("%04x", i)
		if err := os.MkdirAll(filepath.Join(shareDir, name), 0o750); err != nil {
			t.Fatal(err)
		}
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
	if !strings.Contains(payload.EPWA["record_url"].(string), "/e/"+packet.PackageID[:shortShareIDLength]+"/") {
		t.Fatalf("record_url missing short form: %+v", payload.EPWA)
	}

	// Bare digest resolves too.
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/artifact/"+digest, nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("bare digest status=%d", recorder.Code)
	}

	// An ambiguous short prefix must fall back to exact package identity.
	collision := packet.PackageID[:shortShareIDLength] + strings.Repeat("0", 64-shortShareIDLength)
	if err := os.MkdirAll(filepath.Join(shareDir, collision), 0o750); err != nil {
		t.Fatal(err)
	}
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/artifact/"+digest, nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "/e/"+packet.PackageID+"/") {
		t.Fatalf("ambiguous prefix was not replaced by exact identity: %s", recorder.Body.String())
	}

	// Corrupting a published asset must revoke readiness in both surfaces.
	manifest, _, err := evidenceshare.ValidateGenericPackage(packet.Directory, packet.PackageID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packet.Directory, strings.TrimPrefix(manifest.AssetRef, "./")), []byte("tampered"), 0o600); err != nil {
		t.Fatal(err)
	}
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/artifact/"+digest, nil))
	if recorder.Code == http.StatusOK || len(verifySharePackage(cfg, packet.PackageID)) == 0 {
		t.Fatal("corrupted package reported ready")
	}

	if err := os.Remove(filepath.Join(packet.Directory, strings.TrimPrefix(manifest.AssetRef, "./"))); err != nil {
		t.Fatal(err)
	}
	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/artifact/"+digest, nil))
	if recorder.Code == http.StatusOK || len(verifySharePackage(cfg, packet.PackageID)) == 0 {
		t.Fatal("missing payload reported ready")
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
