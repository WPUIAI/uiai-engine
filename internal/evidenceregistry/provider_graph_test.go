package evidenceregistry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSyncFocusaImportsRealProjectTaskAndDependencyShapes(t *testing.T) {
	projectRoot := filepath.Join(t.TempDir(), "project")
	if err := os.MkdirAll(filepath.Join(projectRoot, ".beads"), 0o755); err != nil {
		t.Fatal(err)
	}
	brPath := filepath.Join(t.TempDir(), "br")
	script := `#!/bin/sh
case "$1" in
  list) printf '%s' '{"issues":[{"id":"focusa-epic","title":"EPWA epic","description":"Ship the complete evidence registry.","status":"open","priority":0,"issue_type":"epic","updated_at":"2026-09-01T01:00:00Z"},{"id":"focusa-task","title":"Registry task","description":"Index real provider work.","status":"in_progress","priority":1,"issue_type":"task","updated_at":"2026-09-01T02:00:00Z","external_ref":"github:test/1"}]}' ;;
  graph) printf '%s' '{"components":[{"edges":[["focusa-task","focusa-epic"]]}]}' ;;
  *) exit 2 ;;
esac`
	if err := os.WriteFile(brPath, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/project/list" {
			http.NotFound(w, req)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"schema":"focusa.project_dashboard.v1","projects":[{"project_id":"focusa","canonical_name":"Focusa","project_root":"` + projectRoot + `","fingerprint":"project-fingerprint:abc","workspace_kind":"rust-monorepo","scope_safety":"safe","last_verified_at":"2026-09-01T00:00:00Z"}]}`))
	}))
	defer server.Close()
	manager, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	result, err := manager.SyncFocusa(context.Background(), FocusaSyncConfig{BaseURL: server.URL, BRPath: brPath, ProjectIDs: []string{"focusa"}, AllowedRootPrefixes: []string{projectRoot}, MaxProjects: 1, MaxItems: 100})
	if err != nil {
		t.Fatal(err)
	}
	if result.Projects != 1 || result.ChangedProjects != 1 || result.Items != 2 || len(result.Warnings) != 0 {
		t.Fatalf("unexpected sync result: %#v", result)
	}
	repeated, err := manager.SyncFocusa(context.Background(), FocusaSyncConfig{BaseURL: server.URL, BRPath: brPath, ProjectIDs: []string{"focusa"}, AllowedRootPrefixes: []string{projectRoot}, MaxProjects: 1, MaxItems: 100})
	if err != nil {
		t.Fatal(err)
	}
	if repeated.ChangedProjects != 0 || len(repeated.Results) != 1 || repeated.Results[0].Changed {
		t.Fatalf("unchanged graph advanced revision: %#v", repeated)
	}
	store, err := manager.Project(context.Background(), "project:focusa")
	if err != nil {
		t.Fatal(err)
	}
	page, err := store.ListProviderWorkItems(context.Background(), ProviderWorkItemQuery{ProjectRef: "project:focusa", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	items := page.WorkItems
	if len(items) != 2 || items[0].ItemType != "epic" || items[0].Description != "Ship the complete evidence registry." || items[1].BindingState != "focusa_binding_pending" || len(items[1].DependencyRefs) != 1 || items[1].DependencyRefs[0] != "work-item:br:focusa-epic" {
		t.Fatalf("provider graph mismatch: %#v", items)
	}
	firstPage, err := store.ListProviderWorkItems(context.Background(), ProviderWorkItemQuery{ProjectRef: "project:focusa", Limit: 1})
	if err != nil || firstPage.NextCursor == "" || len(firstPage.WorkItems) != 1 {
		t.Fatalf("provider keyset page missing: page=%#v err=%v", firstPage, err)
	}
	secondPage, err := store.ListProviderWorkItems(context.Background(), ProviderWorkItemQuery{ProjectRef: "project:focusa", Limit: 1, Cursor: firstPage.NextCursor})
	if err != nil || len(secondPage.WorkItems) != 1 || secondPage.WorkItems[0].ItemID == firstPage.WorkItems[0].ItemID {
		t.Fatalf("provider keyset continuation mismatch: page=%#v err=%v", secondPage, err)
	}
	searchPage, err := store.ListProviderWorkItems(context.Background(), ProviderWorkItemQuery{ProjectRef: "project:focusa", Text: "complete evidence", Limit: 1})
	if err != nil || len(searchPage.WorkItems) != 1 || searchPage.WorkItems[0].ItemID != "focusa-epic" {
		t.Fatalf("provider FTS mismatch: page=%#v err=%v", searchPage, err)
	}
	lowMemPage, err := store.ListProviderWorkItems(context.Background(), ProviderWorkItemQuery{ProjectRef: "project:focusa", Limit: MaxPageSize, ResourceProfile: ResourceLowMem})
	if err != nil || lowMemPage.PageSize != MaxLowMemPageSize || lowMemPage.ResourceProfile != ResourceLowMem || lowMemPage.MediaPosture != MediaOmittedNonessential {
		t.Fatalf("provider LowMem profile mismatch: page=%#v err=%v", lowMemPage, err)
	}
	var rootLeak int
	if err := store.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM project_projection WHERE project_ref = ? AND (display_name LIKE ? OR source_revision LIKE ?)`, "project:focusa", "%"+projectRoot+"%", "%"+projectRoot+"%").Scan(&rootLeak); err != nil {
		t.Fatal(err)
	}
	if rootLeak != 0 {
		t.Fatal("private project root entered registry projection")
	}
	reverse := 0
	if err := store.db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM work_item_edge WHERE project_ref = ? AND target_ref = ? AND relation = 'depends_on'`, "project:focusa", "work-item:br:focusa-epic").Scan(&reverse); err != nil {
		t.Fatal(err)
	}
	if reverse != 1 || strings.Contains(items[0].ProjectRef, projectRoot) {
		t.Fatalf("dependency/root posture invalid: reverse=%d", reverse)
	}
}

func TestAllowedProviderRootFailsClosed(t *testing.T) {
	allowed := t.TempDir()
	outside := t.TempDir()
	roots, err := canonicalAllowedRoots([]string{allowed})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := allowedProviderRoot(outside, roots); err == nil {
		t.Fatal("provider root outside configured prefixes was accepted")
	}
	child := filepath.Join(allowed, "project")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	resolved, err := allowedProviderRoot(child, roots)
	if err != nil || resolved == "" {
		t.Fatalf("allowed provider root rejected: root=%q err=%v", resolved, err)
	}
}
