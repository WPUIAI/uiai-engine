package evidenceregistry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"
)

func TestRegistryTenThousandRowBudgets(t *testing.T) {
	runRegistryScaleBudgets(t, 10_000)
}

func TestRegistryHundredThousandRowBudgets(t *testing.T) {
	if os.Getenv("UIAI_EPWA_100K_CONFORMANCE") != "1" {
		t.Skip("set UIAI_EPWA_100K_CONFORMANCE=1 for the explicit 100k conformance fixture")
	}
	runRegistryScaleBudgets(t, 100_000)
}

func runRegistryScaleBudgets(t *testing.T, rows int) {
	t.Helper()
	ctx := context.Background()
	root := t.TempDir()
	store, err := Open(ctx, Config{Path: filepath.Join(root, "scale.sqlite3"), ProjectRef: "project:scale"})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	fixtureStarted := time.Now()
	fixtureRegistryRows(t, store, rows)
	fixtureDuration := time.Since(fixtureStarted)
	target := fmt.Sprintf("artifact:scale:%06d", rows-1)
	rareTerm := fmt.Sprintf("rareterm%d", rows-1)

	// Warm every measured index path before collecting comparable samples.
	if _, err := store.Resolve(ctx, target, 1); err != nil {
		t.Fatal(err)
	}
	if _, err := store.List(ctx, Query{PageSize: 25}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.List(ctx, Query{Text: rareTerm, PageSize: 25}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Edges(ctx, "work-item:scale:target", DirectionReverse, RelationWorkItem, 25); err != nil {
		t.Fatal(err)
	}

	exactP95 := measuredP95(t, 50, func() error { _, err := store.Resolve(ctx, target, 1); return err })
	listP95 := measuredP95(t, 50, func() error { _, err := store.List(ctx, Query{PageSize: 25}); return err })
	searchP95 := measuredP95(t, 30, func() error { _, err := store.List(ctx, Query{Text: rareTerm, PageSize: 25}); return err })
	reverseP95 := measuredP95(t, 50, func() error {
		_, err := store.Edges(ctx, "work-item:scale:target", DirectionReverse, RelationWorkItem, 25)
		return err
	})
	lowMem, err := store.List(ctx, Query{PageSize: MaxPageSize, ResourceProfile: ResourceLowMem})
	if err != nil {
		t.Fatal(err)
	}
	if lowMem.PageSize != MaxLowMemPageSize || len(lowMem.Rows) != MaxLowMemPageSize || lowMem.NextCursor == "" || lowMem.MediaPosture != MediaOmittedNonessential {
		t.Fatalf("LowMem page is not bounded: %#v", lowMem)
	}
	status, err := store.Status(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if status.ArtifactCount != uint64(rows) {
		t.Fatalf("artifact count=%d want=%d", status.ArtifactCount, rows)
	}

	report := map[string]any{
		"schema": "uiai.epwa_registry_scale_report.v1", "fixture_rows": rows,
		"hardware_class": fmt.Sprintf("%s/%s cpu=%d", runtime.GOOS, runtime.GOARCH, runtime.NumCPU()),
		"state":          "warm", "resource_profile": ResourceNormal, "index_revision": status.Revision,
		"fixture_duration_ms": fixtureDuration.Milliseconds(), "exact_lookup_p95_ms": millis(exactP95),
		"first_page_p95_ms": millis(listP95), "indexed_search_p95_ms": millis(searchP95),
		"reverse_edge_p95_ms": millis(reverseP95), "lowmem_page_size": lowMem.PageSize,
	}
	body, _ := json.Marshal(report)
	t.Log(string(body))
	if exactP95 > 25*time.Millisecond || listP95 > 100*time.Millisecond || searchP95 > 150*time.Millisecond || reverseP95 > 100*time.Millisecond {
		t.Fatalf("registry budget exceeded: %s", body)
	}
}

func fixtureRegistryRows(t *testing.T, store *Store, rows int) {
	t.Helper()
	ctx := context.Background()
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	artifactStatement, err := tx.PrepareContext(ctx, `INSERT INTO artifact(
		artifact_ref, revision, manifest_sha256, index_input_sha256, bundle_sha256, title, summary, kinds_json,
		project_ref, workstream_ref, workset_ref, callgraph_ref, workpoint_ref, verification, access_class,
		redaction_state, closure_posture, captured_at, freshness_observed_at, pwa_path, indexed_at)
		VALUES(?,1,?,?,?,?,?,'["screenshot"]','project:scale','workstream:scale','workset:scale','callgraph:scale','workpoint:scale','indeterminate','public_safe','applied','blocked',?,?,?,?)`)
	if err != nil {
		t.Fatal(err)
	}
	defer artifactStatement.Close()
	searchStatement, err := tx.PrepareContext(ctx, `INSERT INTO artifact_search(artifact_ref, revision, title, summary, kinds, work_items, acceptance_atoms) VALUES(?,1,?,?, 'screenshot', '', '')`)
	if err != nil {
		t.Fatal(err)
	}
	defer searchStatement.Close()
	base := time.Date(2026, 9, 4, 0, 0, 0, 0, time.UTC).UnixNano()
	for i := 0; i < rows; i++ {
		ref := fmt.Sprintf("artifact:scale:%06d", i)
		title := fmt.Sprintf("Evidence row %06d", i)
		summary := "deterministic registry scale fixture"
		if i == rows-1 {
			summary += fmt.Sprintf(" rareterm%d", i)
		}
		digestBytes := sha256.Sum256([]byte(ref))
		digest := hex.EncodeToString(digestBytes[:])
		captured := base + int64(i)
		if _, err := artifactStatement.ExecContext(ctx, ref, digest, digest, digest, title, summary, captured, captured, "/evidence/?artifact="+ref, captured); err != nil {
			t.Fatalf("insert artifact %d: %v", i, err)
		}
		if _, err := searchStatement.ExecContext(ctx, ref, title, summary); err != nil {
			t.Fatalf("insert search row %d: %v", i, err)
		}
	}
	last := fmt.Sprintf("artifact:scale:%06d", rows-1)
	if _, err := tx.ExecContext(ctx, `INSERT INTO artifact_edge(project_ref, source_ref, source_revision, target_ref, target_revision, relation, provenance_receipt, observed_at) VALUES('project:scale',?,'1','work-item:scale:target','1',?,'receipt:scale',?)`, last, RelationWorkItem, base+int64(rows)); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE registry_meta SET revision=?, updated_at=? WHERE id=1`, rows, base+int64(rows)); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func measuredP95(t *testing.T, samples int, operation func() error) time.Duration {
	t.Helper()
	values := make([]time.Duration, samples)
	for i := range values {
		started := time.Now()
		if err := operation(); err != nil {
			t.Fatal(err)
		}
		values[i] = time.Since(started)
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	return values[(len(values)*95+99)/100-1]
}

func millis(value time.Duration) float64 {
	return float64(value.Microseconds()) / 1000
}
