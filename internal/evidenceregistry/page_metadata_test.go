package evidenceregistry

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestPageMetadataPreservesFreshnessAndFullStatusCounts(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, Config{Path: filepath.Join(t.TempDir(), "metadata.sqlite3"), ProjectRef: "project:uiai-engine"})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	if _, err := store.Index(ctx, IndexInput{Manifest: loadGoldenManifest(t), ManifestSHA256: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}); err != nil {
		t.Fatal(err)
	}
	before, err := store.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if before.ArtifactCount != 1 {
		t.Fatal("fixture inventory count missing")
	}
	for _, state := range []IndexState{IndexReady, IndexCorrupt} {
		if _, err := store.db.ExecContext(ctx, `UPDATE registry_meta SET state=? WHERE id=1`, state); err != nil {
			t.Fatal(err)
		}
		page, err := store.List(ctx, Query{PageSize: 1})
		if err != nil {
			t.Fatal(err)
		}
		status, err := store.Status(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if page.IndexState != state || page.IndexRevision != status.Revision || page.ProjectRef != status.ProjectRef || page.TotalPosture != "omitted" {
			t.Fatal("page metadata lost truthful scope, revision or degradation")
		}
		if status.ArtifactCount != before.ArtifactCount || status.EdgeCount != before.EdgeCount {
			t.Fatal("full status inventory counts changed")
		}
	}
	// Corrupt only this isolated fixture: metadata loss must not become an empty,
	// healthy page or status. No installed registry is touched.
	if _, err := store.db.ExecContext(ctx, `DROP TABLE registry_meta`); err != nil {
		t.Fatal(err)
	}
	if _, err := store.List(ctx, Query{}); !errors.Is(err, ErrIndexCorrupt) {
		t.Fatalf("page metadata loss: %v", err)
	}
	if _, err := store.Status(ctx); !errors.Is(err, ErrIndexCorrupt) {
		t.Fatalf("status metadata loss: %v", err)
	}
}
