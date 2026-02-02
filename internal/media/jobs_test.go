package media

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestJobStoreCreate(t *testing.T) {
	dir := t.TempDir()
	s := NewJobStore(dir)

	job := &Job{
		ID:        "test-001",
		Type:      TypeDeviceMockup,
		Status:    StatusPending,
		Device:    "macbook-pro",
		URLs:      []string{"https://example.com"},
		CreatedAt: time.Now().UTC(),
	}
	s.Create(job)

	got := s.Get("test-001")
	if got == nil {
		t.Fatal("expected job, got nil")
	}
	if got.ID != "test-001" {
		t.Errorf("expected ID test-001, got %s", got.ID)
	}
	if got.Type != TypeDeviceMockup {
		t.Errorf("expected type device_mockup, got %s", got.Type)
	}
	if got.Status != StatusPending {
		t.Errorf("expected status pending, got %s", got.Status)
	}
}

func TestJobStoreGetMissing(t *testing.T) {
	dir := t.TempDir()
	s := NewJobStore(dir)

	got := s.Get("nonexistent")
	if got != nil {
		t.Errorf("expected nil for nonexistent job, got %+v", got)
	}
}

func TestJobStoreUpdate(t *testing.T) {
	dir := t.TempDir()
	s := NewJobStore(dir)

	s.Create(&Job{
		ID:        "test-002",
		Type:      TypeAnimatedGIF,
		Status:    StatusPending,
		CreatedAt: time.Now().UTC(),
	})

	now := time.Now().UTC()
	s.Update("test-002", func(j *Job) {
		j.Status = StatusComplete
		j.ResultPath = "/tmp/test.gif"
		j.CompletedAt = &now
	})

	got := s.Get("test-002")
	if got.Status != StatusComplete {
		t.Errorf("expected status complete, got %s", got.Status)
	}
	if got.ResultPath != "/tmp/test.gif" {
		t.Errorf("expected result path /tmp/test.gif, got %s", got.ResultPath)
	}
	if got.CompletedAt == nil {
		t.Error("expected completed_at to be set")
	}
}

func TestJobStoreUpdateNonexistent(t *testing.T) {
	dir := t.TempDir()
	s := NewJobStore(dir)

	// Should not panic
	s.Update("nope", func(j *Job) {
		j.Status = StatusFailed
	})
}

func TestJobStoreList(t *testing.T) {
	dir := t.TempDir()
	s := NewJobStore(dir)

	// Create 3 jobs with different timestamps
	for i, id := range []string{"a", "b", "c"} {
		s.Create(&Job{
			ID:        id,
			Type:      TypeDeviceMockup,
			Status:    StatusComplete,
			CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Second),
		})
	}

	// List all
	all := s.List(0)
	if len(all) != 3 {
		t.Fatalf("expected 3 jobs, got %d", len(all))
	}
	// Should be sorted newest first
	if all[0].ID != "c" {
		t.Errorf("expected newest first (c), got %s", all[0].ID)
	}

	// List with limit
	limited := s.List(2)
	if len(limited) != 2 {
		t.Errorf("expected 2 jobs with limit=2, got %d", len(limited))
	}
}

func TestJobStoreCleanOld(t *testing.T) {
	dir := t.TempDir()
	s := NewJobStore(dir)

	old := time.Now().UTC().Add(-48 * time.Hour)
	recent := time.Now().UTC()

	s.Create(&Job{ID: "old", Type: TypeDeviceMockup, Status: StatusComplete, CreatedAt: old})
	s.Create(&Job{ID: "new", Type: TypeDeviceMockup, Status: StatusComplete, CreatedAt: recent})

	removed := s.CleanOld(24 * time.Hour)
	if removed != 1 {
		t.Errorf("expected 1 removed, got %d", removed)
	}
	if s.Get("old") != nil {
		t.Error("old job should have been cleaned")
	}
	if s.Get("new") == nil {
		t.Error("new job should still exist")
	}
}

func TestJobStorePersistence(t *testing.T) {
	dir := t.TempDir()

	// Create and save
	s1 := NewJobStore(dir)
	s1.Create(&Job{
		ID:        "persist-001",
		Type:      TypeAnimatedGIF,
		Status:    StatusComplete,
		ResultPath: "/tmp/out.gif",
		CreatedAt: time.Now().UTC(),
	})

	// Verify file exists
	fp := filepath.Join(dir, "media_jobs.json")
	if _, err := os.Stat(fp); os.IsNotExist(err) {
		t.Fatal("jobs file not created on disk")
	}

	// Load new store from same dir
	s2 := NewJobStore(dir)
	got := s2.Get("persist-001")
	if got == nil {
		t.Fatal("expected persisted job, got nil")
	}
	if got.ResultPath != "/tmp/out.gif" {
		t.Errorf("expected result path /tmp/out.gif, got %s", got.ResultPath)
	}
}

func TestJobTypes(t *testing.T) {
	// Verify type constants match expected strings
	tests := []struct {
		got  JobType
		want string
	}{
		{TypeDeviceMockup, "device_mockup"},
		{TypeAnimatedGIF, "animated_gif"},
		{TypeProductVideo, "product_video"},
		{TypeIllustration, "ai_illustration"},
	}
	for _, tt := range tests {
		if string(tt.got) != tt.want {
			t.Errorf("JobType %v != %q", tt.got, tt.want)
		}
	}
}

func TestJobStatuses(t *testing.T) {
	tests := []struct {
		got  JobStatus
		want string
	}{
		{StatusPending, "pending"},
		{StatusProcessing, "processing"},
		{StatusComplete, "complete"},
		{StatusFailed, "failed"},
		{StatusTimeout, "timeout"},
	}
	for _, tt := range tests {
		if string(tt.got) != tt.want {
			t.Errorf("JobStatus %v != %q", tt.got, tt.want)
		}
	}
}
