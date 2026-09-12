package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/evidenceshare"
	"github.com/go-chi/chi/v5"
)

// A manifest-shaped file is not a published, verified package.
func TestArtifactResolverRejectsUnpublishedManifest(t *testing.T) {
	root := t.TempDir()
	t.Setenv("UIAI_EVIDENCE_SHARE_DIR", root)
	cfg := &config.Config{}
	digest := strings.Repeat("ab", 32)
	directory := filepath.Join(root, strings.Repeat("cd", 32))
	if err := os.MkdirAll(directory, 0o750); err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(evidenceshare.GenericManifest{
		Schema:       evidenceshare.GenericArtifactSchema,
		ArtifactRef:  "uiai-artifact:sha256:" + digest,
		Availability: "ready", Access: "public_safe_read_only",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "artifact.json"), body, 0o600); err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	mountScreenshotArtifact(router, cfg)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/artifact/"+digest, nil))
	if response.Code == http.StatusOK || strings.Contains(response.Body.String(), `"delivery_state":"ready"`) {
		t.Fatalf("unpublished package reported ready: %d %s", response.Code, response.Body.String())
	}
}
