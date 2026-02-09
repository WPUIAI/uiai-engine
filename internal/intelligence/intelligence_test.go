package intelligence

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDocumentValidation(t *testing.T) {
	valid := Document{
		ID: "doc1", RunID: "run1", Title: "Test", Body: "Content",
		SourceType: "upload", Category: "hero", DocumentType: "content",
		CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid doc failed: %v", err)
	}

	// Missing required fields
	cases := []struct {
		name   string
		modify func(*Document)
		field  string
	}{
		{"no id", func(d *Document) { d.ID = "" }, "id"},
		{"no runId", func(d *Document) { d.RunID = "" }, "runId"},
		{"no title", func(d *Document) { d.Title = "" }, "title"},
		{"no body", func(d *Document) { d.Body = "" }, "body"},
		{"no sourceType", func(d *Document) { d.SourceType = "" }, "sourceType"},
		{"bad sourceType", func(d *Document) { d.SourceType = "invalid" }, "sourceType"},
		{"no category", func(d *Document) { d.Category = "" }, "category"},
		{"no documentType", func(d *Document) { d.DocumentType = "" }, "documentType"},
		{"no createdAt", func(d *Document) { d.CreatedAt = "" }, "createdAt"},
		{"no updatedAt", func(d *Document) { d.UpdatedAt = "" }, "updatedAt"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := valid
			tc.modify(&d)
			err := d.Validate()
			if err == nil {
				t.Fatal("expected validation error")
			}
			ve, ok := err.(*ValidationError)
			if !ok {
				t.Fatalf("expected ValidationError, got %T", err)
			}
			if ve.Field != tc.field {
				t.Fatalf("expected field %q, got %q", tc.field, ve.Field)
			}
		})
	}
}

func TestStorePersistence(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	docs := []Document{
		{ID: "d1", RunID: "r1", Title: "Alpha", Body: "First document about design",
			SourceType: "upload", Category: "hero", DocumentType: "content",
			CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z",
			Keywords: []string{"design", "hero"}},
		{ID: "d2", RunID: "r1", Title: "Beta", Body: "Second document about colors",
			SourceType: "intake", Category: "features", DocumentType: "brand",
			CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-01T00:00:00Z",
			Keywords: []string{"colors", "brand"}, Boost: 2.0},
	}

	// Write documents
	if err := store.WriteDocuments("r1", docs); err != nil {
		t.Fatalf("write docs: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(filepath.Join(dir, "r1", "documents.json")); err != nil {
		t.Fatal("documents.json not on disk")
	}

	// Read back from fresh store (no cache)
	store2 := NewStore(dir)
	got, err := store2.ReadDocuments("r1")
	if err != nil {
		t.Fatalf("read docs: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(got))
	}
	if got[0].Title != "Alpha" || got[1].Title != "Beta" {
		t.Fatal("document content mismatch")
	}

	// Non-existent run
	empty, err := store2.ReadDocuments("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if empty != nil {
		t.Fatal("expected nil for non-existent run")
	}
}

func TestMetadataPersistence(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	meta, err := store.UpdateMetadata("r1", func(m *IndexMetadata) {
		m.BuildID = "build-abc"
		m.Status = "queued"
		m.DocCount = 5
		m.ChunkCount = 5
		m.CreatedAt = "2026-01-01"
		m.UpdatedAt = "2026-01-01"
		m.Source = "trigger"
	})
	if err != nil {
		t.Fatal(err)
	}
	if meta.BuildID != "build-abc" {
		t.Fatal("buildId mismatch")
	}

	// Read from fresh store
	store2 := NewStore(dir)
	got, err := store2.ReadMetadata("r1")
	if err != nil {
		t.Fatal(err)
	}
	if got == nil {
		t.Fatal("metadata not found")
	}
	if got.DocCount != 5 || got.Status != "queued" {
		t.Fatal("metadata content mismatch")
	}
}

func TestArtifacts(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	if store.HasArtifact("r1", "docfind_bg.wasm") {
		t.Fatal("should not exist yet")
	}

	store.WriteArtifact("r1", "docfind_bg.wasm", []byte("fake wasm"))
	if !store.HasArtifact("r1", "docfind_bg.wasm") {
		t.Fatal("should exist after write")
	}
}

func TestWeightedSearch(t *testing.T) {
	docs := []Document{
		{ID: "d1", Title: "Color Palette Design", Body: "This document describes the color palette for the website.",
			Summary: "Color design overview", Keywords: []string{"color", "palette", "design"}},
		{ID: "d2", Title: "Typography Guide", Body: "Font choices and color usage in typography.",
			Summary: "How to use fonts", Keywords: []string{"typography", "fonts"}},
		{ID: "d3", Title: "Excluded Doc", Body: "This has color info but is excluded.",
			ExcludeFromSearch: true},
		{ID: "d4", Title: "Navigation Structure", Body: "Menu layout and page hierarchy.",
			Summary: "Site navigation", Keywords: []string{"menu", "navigation"}},
	}

	results := Search(docs, "color", 10)

	if len(results) != 2 {
		t.Fatalf("expected 2 results (d3 excluded), got %d", len(results))
	}
	// d1 should rank higher: "color" in title (4x), summary (2x), body (1x), keywords (3x)
	if results[0].ID != "d1" {
		t.Fatalf("expected d1 first (highest title+keyword match), got %s", results[0].ID)
	}
	// d2 has "color" only in body
	if results[1].ID != "d2" {
		t.Fatalf("expected d2 second, got %s", results[1].ID)
	}
	if len(results[0].Snippet) == 0 {
		t.Fatal("snippet should not be empty")
	}
}

func TestSearchBoost(t *testing.T) {
	docs := []Document{
		{ID: "d1", Title: "Normal Doc", Body: "design patterns for sites", Boost: 1},
		{ID: "d2", Title: "Boosted Doc", Body: "design patterns for sites", Boost: 5},
	}

	results := Search(docs, "design", 10)
	if len(results) != 2 {
		t.Fatalf("expected 2, got %d", len(results))
	}
	// d2 should rank first due to 5x boost
	if results[0].ID != "d2" {
		t.Fatalf("expected d2 first (boosted), got %s", results[0].ID)
	}
}

func TestSearchLimit(t *testing.T) {
	docs := make([]Document, 20)
	for i := range docs {
		docs[i] = Document{ID: "d" + string(rune('a'+i)), Title: "test", Body: "keyword"}
	}
	results := Search(docs, "keyword", 5)
	if len(results) != 5 {
		t.Fatalf("expected 5 (limit), got %d", len(results))
	}
}

func TestTierLimits(t *testing.T) {
	free := GetTierLimits("free")
	if free.AllowEmbeddings {
		t.Fatal("free should not allow embeddings")
	}
	if free.MaxDocs != 10 {
		t.Fatalf("free maxDocs: expected 10, got %d", free.MaxDocs)
	}

	pro := GetTierLimits("pro")
	if !pro.AllowEmbeddings {
		t.Fatal("pro should allow embeddings")
	}
	if pro.EmbedDailyLimit != 10000 {
		t.Fatalf("pro embedDailyLimit: expected 10000, got %d", pro.EmbedDailyLimit)
	}

	unknown := GetTierLimits("bogus")
	if unknown.AllowEmbeddings {
		t.Fatal("unknown tier should not allow embeddings")
	}
}

func TestUsageTracker(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "usage.json")
	tracker := NewUsageTracker(path)

	r := tracker.Get("key1", "2026-02-08")
	if r.Count != 0 {
		t.Fatalf("expected 0, got %d", r.Count)
	}

	tracker.Increment("key1", "2026-02-08", 5)
	tracker.Increment("key1", "2026-02-08", 3)
	r = tracker.Get("key1", "2026-02-08")
	if r.Count != 8 {
		t.Fatalf("expected 8, got %d", r.Count)
	}

	// New date resets
	tracker.Increment("key1", "2026-02-09", 1)
	r = tracker.Get("key1", "2026-02-09")
	if r.Count != 1 {
		t.Fatalf("expected 1 after date rollover, got %d", r.Count)
	}

	// Persistence: load from disk
	tracker2 := NewUsageTracker(path)
	r2 := tracker2.Get("key1", "2026-02-09")
	if r2.Count != 1 {
		t.Fatalf("expected 1 from disk, got %d", r2.Count)
	}
}

func TestGitHubConfigNotConfigured(t *testing.T) {
	cfg := &GitHubConfig{}
	if cfg.IsConfigured() {
		t.Fatal("should not be configured without token+repo")
	}
	result := TriggerDocfindBuild(cfg, "run1", "build1")
	if result.OK {
		t.Fatal("should fail without config")
	}
}
