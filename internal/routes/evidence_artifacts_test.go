package routes

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
	"github.com/WPUIAI/uiai-engine/internal/evidenceregistry"
	"github.com/go-chi/chi/v5"
)

func TestEvidenceArtifactCommitReadAndRebuildRoutes(t *testing.T) {
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
	router := chi.NewRouter()
	router.Route("/api/evidence/artifacts", func(r chi.Router) { MountEvidenceArtifacts(r, artifacts, registry) })
	request := httptest.NewRequest(http.MethodPost, "/api/evidence/artifacts/commit", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"registry"`) {
		t.Fatalf("commit status=%d body=%s", response.Code, response.Body.String())
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
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/evidence/artifacts/rebuild", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"artifacts":1`) {
		t.Fatalf("rebuild status=%d body=%s", response.Code, response.Body.String())
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
