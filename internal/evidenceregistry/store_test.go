package evidenceregistry

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenCreatesProjectLockedWALRegistry(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "project", "evidence-index.sqlite3")
	now := time.Date(2026, 9, 1, 3, 0, 0, 0, time.UTC)
	store, err := Open(ctx, Config{Path: path, ProjectRef: "project:focusa", Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	status, err := store.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.Schema != RegistrySchemaV1 || status.ProjectRef != "project:focusa" || status.State != IndexReady || status.Revision != 0 || status.ArtifactCount != 0 || status.EdgeCount != 0 {
		t.Fatalf("unexpected status: %#v", status)
	}
	if !status.LastIntegrityAt.Equal(now) {
		t.Fatalf("integrity time=%s want=%s", status.LastIntegrityAt, now)
	}
	var journal string
	if err := store.db.QueryRowContext(ctx, `PRAGMA journal_mode`).Scan(&journal); err != nil || journal != "wal" {
		t.Fatalf("journal_mode=%q err=%v", journal, err)
	}
	var foreignKeys, busyTimeout int
	if err := store.db.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		t.Fatalf("foreign_keys=%d err=%v", foreignKeys, err)
	}
	if err := store.db.QueryRowContext(ctx, `PRAGMA busy_timeout`).Scan(&busyTimeout); err != nil || busyTimeout != 5000 {
		t.Fatalf("busy_timeout=%d err=%v", busyTimeout, err)
	}
	for _, table := range []string{"artifact", "work_item_binding", "acceptance_binding", "artifact_edge", "closure_binding", "collection", "collection_member", "artifact_search"} {
		var found string
		if err := store.db.QueryRowContext(ctx, `SELECT name FROM sqlite_master WHERE name = ?`, table).Scan(&found); err != nil || found != table {
			t.Fatalf("table %s unavailable: found=%q err=%v", table, found, err)
		}
	}
}

func TestOpenRejectsCrossProjectDatabaseReuse(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "evidence-index.sqlite3")
	first, err := Open(ctx, Config{Path: path, ProjectRef: "project:one"})
	if err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	second, err := Open(ctx, Config{Path: path, ProjectRef: "project:two"})
	if second != nil {
		second.Close()
	}
	if !errors.Is(err, ErrProjectMismatch) {
		t.Fatalf("cross-project reuse err=%v", err)
	}
	reopened, err := Open(ctx, Config{Path: path, ProjectRef: "project:one"})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
}

func TestOpenRejectsInvalidConfiguration(t *testing.T) {
	ctx := context.Background()
	for _, cfg := range []Config{
		{},
		{Path: filepath.Join(t.TempDir(), "index.sqlite3")},
		{Path: filepath.Join(t.TempDir(), "index.sqlite3"), ProjectRef: "project:test", BusyTimeout: -time.Second},
		{Path: filepath.Join(t.TempDir(), "index.sqlite3"), ProjectRef: "project:test", BusyTimeout: 2 * time.Minute},
	} {
		store, err := Open(ctx, cfg)
		if store != nil {
			store.Close()
		}
		if !errors.Is(err, ErrConfig) {
			t.Fatalf("config %#v err=%v", cfg, err)
		}
	}
}
