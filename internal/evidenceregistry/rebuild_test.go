package evidenceregistry

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
)

func TestRebuildFromArtifactStoreProjectsImmutableManifests(t *testing.T) {
	ctx := context.Background()
	artifactStore, _, err := evidenceartifact.OpenStore(evidenceartifact.StoreConfig{
		Root: t.TempDir(), MaxStoreBytes: 64 << 20, MaxArtifacts: 100, MaxAssetBytes: 8 << 20,
		StagingQuarantineAge: time.Hour, GCGrace: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	manifest := loadGoldenManifest(t)
	payload := []byte("public-safe immutable evidence")
	digest := sha256.Sum256(payload)
	manifest.Assets[0].SHA256 = hex.EncodeToString(digest[:])
	manifest.Assets[0].ByteSize = int64(len(payload))
	manifest.Assets[0].MediaType = "text/plain"
	manifest.Integrity.ManifestSHA256 = ""
	manifest, err = evidenceartifact.Seal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	commit, err := artifactStore.Commit(ctx, manifest, map[string]io.Reader{manifest.Assets[0].AssetID: bytes.NewReader(payload)})
	if err != nil {
		t.Fatal(err)
	}
	manager, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	result, err := manager.RebuildFromArtifactStore(ctx, artifactStore)
	if err != nil {
		t.Fatal(err)
	}
	if result.Artifacts != 1 || result.Projects != 1 || result.IndexErrors != 0 {
		t.Fatalf("result=%#v", result)
	}
	project, err := manager.Project(ctx, manifest.Scope.Project.ProjectRef)
	if err != nil {
		t.Fatal(err)
	}
	page, err := project.List(ctx, Query{ProjectRef: manifest.Scope.Project.ProjectRef, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != 1 || page.Rows[0].ManifestSHA256 != commit.ManifestSHA256 {
		t.Fatalf("page=%#v commit=%#v", page, commit)
	}
}
