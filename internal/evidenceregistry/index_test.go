package evidenceregistry

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
)

func TestIndexPreservesWorkGraphAndClosureEdges(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	store, err := Open(ctx, Config{Path: filepath.Join(t.TempDir(), "index.sqlite3"), ProjectRef: "project:uiai-engine", Now: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	input := IndexInput{
		Manifest:              loadGoldenManifest(t),
		ManifestSHA256:        "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		CompletionCaseRef:     "completion-case:t01",
		CompletionContractRef: "completion-contract:t01",
		SettlementPosture:     "open",
		ObservedAt:            now,
		Acceptances:           []AcceptanceBinding{{AcceptanceAtomRef: "atom:types", Revision: "1", State: AcceptanceAccepted, VerifierClass: "independent", VerifierRefs: []string{"verifier:one"}, DecisionRef: "decision:types", ReceiptRef: "receipt:types", Fresh: true, ScopeMatched: true}},
	}
	indexed, err := store.Index(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if indexed.Deduplicated || indexed.IndexRevision != 1 {
		t.Fatalf("unexpected index result: %#v", indexed)
	}

	row, err := store.Resolve(ctx, input.Manifest.ArtifactID, input.Manifest.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if row.WorkItemCount != 2 || row.FirstWorkItemType != "task" || row.FirstWorkItemTitle != "Implement artifact contract" || row.AcceptanceTotal != 2 || row.AcceptanceAccepted != 1 || row.Closure != ClosureBlocked {
		t.Fatalf("unexpected row: %#v", row)
	}

	items, err := store.WorkItems(ctx, input.Manifest.ArtifactID, input.Manifest.Revision)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Description != "Create immutable evidence manifest types." || len(items[1].DependencyRefs) != 1 || items[1].DependencyRefs[0] != "work-item:focusa-a1" || items[1].AcceptanceAtomRefs[0] != "atom:tests" {
		t.Fatalf("work graph not preserved: %#v", items)
	}

	page, err := store.List(ctx, Query{Text: "golden tests", WorkItemType: "task", PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Rows) != 1 || page.Rows[0].ArtifactRef != input.Manifest.ArtifactID {
		t.Fatalf("search mismatch: %#v", page)
	}
	digestPage, err := store.List(ctx, Query{Text: input.ManifestSHA256, PageSize: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(digestPage.Rows) != 1 || digestPage.Rows[0].ArtifactRef != input.Manifest.ArtifactID {
		t.Fatalf("digest lookup mismatch: %#v", digestPage)
	}

	forward, err := store.Edges(ctx, input.Manifest.ArtifactID, DirectionForward, RelationWorkItem, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(forward) != 2 {
		t.Fatalf("forward work-item edges=%d want=2", len(forward))
	}
	reverse, err := store.Edges(ctx, "work-item:focusa-a2", DirectionReverse, RelationWorkItem, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(reverse) != 1 || reverse[0].SourceRef != input.Manifest.ArtifactID {
		t.Fatalf("reverse edge mismatch: %#v", reverse)
	}

	eligible, err := store.Closure(ctx, "work-item:focusa-a1", "completion-case:t01")
	if err != nil {
		t.Fatal(err)
	}
	blocked, err := store.Closure(ctx, "work-item:focusa-a2", "completion-case:t01")
	if err != nil {
		t.Fatal(err)
	}
	if eligible.Posture != ClosureEligible || blocked.Posture != ClosureBlocked {
		t.Fatalf("closure posture eligible=%s blocked=%s", eligible.Posture, blocked.Posture)
	}

	again, err := store.Index(ctx, input)
	if err != nil {
		t.Fatal(err)
	}
	if !again.Deduplicated || again.IndexRevision != indexed.IndexRevision {
		t.Fatalf("idempotent result: %#v", again)
	}
}

func TestIndexRejectsImmutableIdentityConflict(t *testing.T) {
	ctx := context.Background()
	store, err := Open(ctx, Config{Path: filepath.Join(t.TempDir(), "index.sqlite3"), ProjectRef: "project:uiai-engine"})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	input := IndexInput{Manifest: loadGoldenManifest(t), ManifestSHA256: "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}
	if _, err := store.Index(ctx, input); err != nil {
		t.Fatal(err)
	}
	input.ManifestSHA256 = "eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"
	if _, err := store.Index(ctx, input); !errors.Is(err, ErrInputInvalid) {
		t.Fatalf("immutable conflict err=%v", err)
	}
}

func loadGoldenManifest(t *testing.T) evidenceartifact.Manifest {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "evidenceartifact", "testdata", "manifest.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest evidenceartifact.Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatal(err)
	}
	return manifest
}
