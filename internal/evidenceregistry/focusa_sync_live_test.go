package evidenceregistry

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestLiveFocusaProviderGraph(t *testing.T) {
	baseURL := os.Getenv("UIAI_TEST_LIVE_FOCUSA_URL")
	brPath := os.Getenv("UIAI_TEST_LIVE_BR_PATH")
	if baseURL == "" || brPath == "" {
		t.Skip("live Focusa/provider graph test not configured")
	}
	manager, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	result, err := manager.SyncFocusa(ctx, FocusaSyncConfig{BaseURL: baseURL, BRPath: brPath, ProjectIDs: []string{"focusa"}, AllowedRootPrefixes: []string{"/home/wirebot/focusa"}, MaxProjects: 1, MaxItems: 100000})
	if err != nil {
		t.Fatal(err)
	}
	if result.Projects != 1 || result.Items == 0 {
		t.Fatalf("live provider graph empty: %#v", result)
	}
	store, err := manager.Project(ctx, "project:focusa")
	if err != nil {
		t.Fatal(err)
	}
	page, err := store.ListProviderWorkItems(ctx, ProviderWorkItemQuery{ProjectRef: "project:focusa", Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	items := page.WorkItems
	hasDescription, hasTypedItem := false, false
	for _, item := range items {
		hasDescription = hasDescription || item.Description != ""
		hasTypedItem = hasTypedItem || item.ItemType == "task" || item.ItemType == "epic" || item.ItemType == "bug"
	}
	if len(items) == 0 || !hasDescription || !hasTypedItem {
		t.Fatalf("live Focusa work item projection incomplete: items=%d description=%t typed=%t", len(items), hasDescription, hasTypedItem)
	}
	var edgeCount int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM work_item_edge WHERE project_ref = ?`, "project:focusa").Scan(&edgeCount); err != nil {
		t.Fatal(err)
	}
	if edgeCount == 0 {
		t.Fatal("live Focusa project has no provider task relationships")
	}
	t.Logf("indexed %d real work items and %d relationships", result.Items, edgeCount)
}

func TestLiveContinuousFocusaSyncPublishesFreshness(t *testing.T) {
	baseURL := os.Getenv("UIAI_TEST_LIVE_FOCUSA_URL")
	brPath := os.Getenv("UIAI_TEST_LIVE_BR_PATH")
	if baseURL == "" || brPath == "" {
		t.Skip("live Focusa/provider graph test not configured")
	}
	manager, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	events := manager.RegistryEvents(ctx)
	syncer, err := StartContinuousSync(manager, FocusaSyncConfig{BaseURL: baseURL, BRPath: brPath, ProjectIDs: []string{"focusa"}, AllowedRootPrefixes: []string{"/home/wirebot/focusa"}, MaxProjects: 1, MaxItems: 100000}, 5*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer syncer.Close()
	select {
	case event := <-events:
		if event.Status != "completed" || event.Freshness != "live" || event.Projects != 1 || event.Items == 0 {
			t.Fatalf("continuous sync event incomplete: %#v", event)
		}
		t.Logf("continuous sync published %d items via %s", event.Items, event.Trigger)
	case <-ctx.Done():
		t.Fatal("continuous sync did not publish within 30 seconds")
	}
}
