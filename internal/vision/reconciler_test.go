package vision

import "testing"

func TestDiffOrphans(t *testing.T) {
	targets := []PageTargetInfo{
		{ID: "t1", URL: "https://wpuiai.com/manage/"},
		{ID: "t2", URL: "about:blank"}, // warm pool page — spared
		{ID: "t3", URL: ""},            // indeterminate — spared
		{ID: "t4", URL: "https://focusa.dev/docs"},
		{ID: "", URL: "https://evil.example"}, // no ID — cannot close, spared
	}
	sessions := map[string]bool{"t1": true}

	orphanIDs := map[string]bool{}
	for _, o := range DiffOrphans(targets, sessions) {
		if orphanIDs[o.ID] {
			t.Fatalf("duplicate orphan %s", o.ID)
		}
		orphanIDs[o.ID] = true
	}

	if !orphanIDs["t4"] {
		t.Fatal("expected t4 flagged as orphan")
	}
	for _, id := range []string{"t1", "t2", "t3", ""} {
		if orphanIDs[id] {
			t.Fatalf("target %q must not be flagged", id)
		}
	}
}

func TestReconcilerSnapshotShape(t *testing.T) {
	r := &Reconciler{interval: 30_000_000_000, stopCh: make(chan struct{})}
	snap := r.Snapshot()
	for _, k := range []string{"running", "pages_leaked_total", "reaped_sessions_total", "last_run"} {
		if _, ok := snap[k]; !ok {
			t.Fatalf("snapshot missing %q", k)
		}
	}
}
