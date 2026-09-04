package routes

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/epwadelivery"
	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
	"github.com/WPUIAI/uiai-engine/internal/evidenceregistry"
	"github.com/go-chi/chi/v5"
)

func TestEvidenceArtifactCommitReadAndRebuildRoutes(t *testing.T) {
	dataRoot := t.TempDir()
	artifacts, _, err := evidenceartifact.OpenStore(evidenceartifact.StoreConfig{
		Root: t.TempDir(), MaxStoreBytes: 64 << 20, MaxArtifacts: 100, MaxAssetBytes: 8 << 20,
		StagingQuarantineAge: time.Hour, GCGrace: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	registry, err := evidenceregistry.NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()
	manifest := routeGoldenManifest(t)
	payload := []byte("route evidence bytes")
	digest := sha256.Sum256(payload)
	manifest.Assets[0].SHA256 = hex.EncodeToString(digest[:])
	manifest.Assets[0].ByteSize = int64(len(payload))
	manifest.Assets[0].MediaType = "text/plain"
	manifest.Policy.AccessClass = evidenceartifact.AccessPublicSafe
	manifest.Scope.ContinuityRef = "continuity:artifact-route"
	manifest.Integrity.ManifestSHA256 = ""
	manifest, err = evidenceartifact.Seal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("manifest", string(manifestJSON)); err != nil {
		t.Fatal(err)
	}
	part, err := writer.CreateFormFile(manifest.Assets[0].AssetID, "proof.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(payload); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: dataRoot}}
	router := chi.NewRouter()
	router.Route("/api/evidence/artifacts", func(r chi.Router) { MountEvidenceArtifacts(r, cfg, artifacts, registry) })
	router.Route("/api/screenshot", func(r chi.Router) { mountEvidenceShare(r, cfg) })
	request := httptest.NewRequest(http.MethodPost, "https://evidence.example/api/evidence/artifacts/commit", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"registry"`) || !strings.Contains(response.Body.String(), `"schema":"uiai.epwa_delivery.v1"`) {
		t.Fatalf("commit status=%d body=%s", response.Code, response.Body.String())
	}
	var commitResponse map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &commitResponse); err != nil {
		t.Fatal(err)
	}
	if commitResponse["artifact_ref"] != manifest.ArtifactID || commitResponse["delivery_state"] != "ready" || commitResponse["artifact_url"] == nil || commitResponse["portable_url"] == nil || commitResponse["epwa_delivery"] == nil {
		t.Fatalf("commit omitted mandatory EPWA delivery binding: %#v", commitResponse)
	}
	for _, key := range []string{"artifact_url", "portable_url"} {
		target := commitResponse[key].(string)
		served := httptest.NewRecorder()
		router.ServeHTTP(served, httptest.NewRequest(http.MethodGet, mustPath(t, target), nil))
		if served.Code != http.StatusOK {
			t.Fatalf("%s was not served: status=%d body=%s", key, served.Code, served.Body.String())
		}
	}

	manifestPath := "/api/evidence/artifacts/manifest?artifact_id=" + url.QueryEscape(manifest.ArtifactID) + "&revision=1"
	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, manifestPath, nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), manifest.ArtifactID) {
		t.Fatalf("manifest status=%d body=%s", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/evidence/artifacts/assets/"+manifest.Assets[0].SHA256, nil))
	if response.Code != http.StatusOK || response.Body.String() != string(payload) {
		t.Fatalf("asset status=%d body=%q", response.Code, response.Body.String())
	}

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/evidence/artifacts/rebuild?epwa_cursor=0&epwa_limit=1", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"artifacts":1`) {
		t.Fatalf("rebuild status=%d body=%s", response.Code, response.Body.String())
	}
	var firstRebuild map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &firstRebuild); err != nil {
		t.Fatal(err)
	}
	backfill, ok := firstRebuild["epwa_backfill"].(map[string]any)
	if !ok || backfill["cursor"] != float64(0) || backfill["processed"] != float64(1) || backfill["total"] != float64(1) {
		t.Fatalf("unexpected bounded backfill projection: %#v", firstRebuild)
	}
	deliveries, ok := backfill["deliveries"].([]any)
	if !ok || len(deliveries) != 1 {
		t.Fatalf("missing backfill delivery: %#v", backfill)
	}
	firstDelivery := deliveries[0].(map[string]any)
	firstDeliveryID := firstDelivery["delivery_id"]
	firstPackageID := firstDelivery["epwa"].(map[string]any)["package_id"]

	response = httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/evidence/artifacts/rebuild?epwa_cursor=0&epwa_limit=1", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("idempotent rebuild status=%d body=%s", response.Code, response.Body.String())
	}
	var retryRebuild map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &retryRebuild); err != nil {
		t.Fatal(err)
	}
	retryDelivery := retryRebuild["epwa_backfill"].(map[string]any)["deliveries"].([]any)[0].(map[string]any)
	if retryDelivery["delivery_id"] != firstDeliveryID || retryDelivery["epwa"].(map[string]any)["package_id"] != firstPackageID {
		t.Fatalf("backfill retry changed stable identity: first=%#v retry=%#v", firstDelivery, retryDelivery)
	}

	project, err := registry.Project(context.Background(), manifest.Scope.Project.ProjectRef)
	if err != nil {
		t.Fatal(err)
	}
	page, err := project.List(context.Background(), evidenceregistry.Query{ProjectRef: manifest.Scope.Project.ProjectRef, PageSize: 10})
	if err != nil || len(page.Rows) != 1 {
		t.Fatalf("page=%#v err=%v", page, err)
	}
}

func TestPublishStoredArtifactEPWARecordFailureReturnsPendingDelivery(t *testing.T) {
	artifacts, _, err := evidenceartifact.OpenStore(evidenceartifact.StoreConfig{
		Root: t.TempDir(), MaxStoreBytes: 64 << 20, MaxArtifacts: 10, MaxAssetBytes: 8 << 20,
		StagingQuarantineAge: time.Hour, GCGrace: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest := routeGoldenManifest(t)
	payload := []byte("pending delivery evidence")
	digest := sha256.Sum256(payload)
	manifest.Assets[0].SHA256 = hex.EncodeToString(digest[:])
	manifest.Assets[0].ByteSize = int64(len(payload))
	manifest.Assets[0].MediaType = "text/plain"
	manifest.Policy.AccessClass = evidenceartifact.AccessPublicSafe
	manifest.Scope.ContinuityRef = "continuity:pending-delivery"
	manifest.Integrity.ManifestSHA256 = ""
	manifest, err = evidenceartifact.Seal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	commit, err := artifacts.Commit(context.Background(), manifest, map[string]io.Reader{
		manifest.Assets[0].AssetID: bytes.NewReader(payload),
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("UIAI_EVIDENCE_SHARE_DIR", t.TempDir())
	invalidRoot := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(invalidRoot, []byte("blocks delivery ledger directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "https://evidence.example/api/evidence/artifacts", nil)
	delivery, publishErr := publishStoredArtifactEPWA(request, &config.Config{Storage: config.StorageConfig{DataDir: invalidRoot}}, artifacts, commit.ArtifactID, commit.Revision, "artifact:"+commit.CommitID)
	if publishErr == nil {
		t.Fatal("delivery ledger failure unexpectedly succeeded")
	}
	if delivery.Schema != epwadelivery.Schema || delivery.State != epwadelivery.StatePendingReconcile || delivery.RecoveryRef != "reconcile:epwa-delivery-record" {
		t.Fatalf("invalid pending delivery: %#v err=%v", delivery, publishErr)
	}
	if delivery.EPWA.RecordURL != "" || delivery.EPWA.PortableURL != "" || delivery.EPWA.PackageID == "" || delivery.EPWA.PackageSHA256 == "" {
		t.Fatalf("pending delivery leaked URLs or lost package identity: %#v", delivery)
	}
	if err := epwadelivery.Validate(delivery); err != nil {
		t.Fatalf("pending delivery contract invalid: %v", err)
	}
}
