// Package media handles media production jobs (mockups, GIFs, videos).
package media

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// JobStatus represents the state of a media production job.
type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusComplete   JobStatus = "complete"
	StatusFailed     JobStatus = "failed"
	StatusTimeout    JobStatus = "timeout"
)

// JobType represents the kind of media being produced.
type JobType string

const (
	TypeDeviceMockup JobType = "device_mockup"
	TypeAnimatedGIF  JobType = "animated_gif"
	TypeProductVideo JobType = "product_video"
	TypeIllustration JobType = "ai_illustration"
)

// Job represents a media production job.
type Job struct {
	ID          string            `json:"id"`
	Type        JobType           `json:"type"`
	Status      JobStatus         `json:"status"`
	Device      string            `json:"device,omitempty"` // macbook-pro, iphone-15, browser-window
	URLs        []string          `json:"urls"`
	Width       int               `json:"width,omitempty"`
	Height      int               `json:"height,omitempty"`
	Frames      int               `json:"frames,omitempty"` // For GIF
	Delay       int               `json:"delay,omitempty"`  // For GIF (centiseconds)
	Mode        string            `json:"mode,omitempty"`   // scroll, pages
	Palette     map[string]string `json:"palette,omitempty"`
	ResultURL   string            `json:"result_url,omitempty"`
	ResultPath  string            `json:"result_path,omitempty"` // Local file path
	Error       string            `json:"error,omitempty"`
	LicenseID   int               `json:"license_id,omitempty"`
	Credits     float64           `json:"credits_charged,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	StartedAt   *time.Time        `json:"started_at,omitempty"`
	CompletedAt *time.Time        `json:"completed_at,omitempty"`
}

// JobStore manages media production jobs.
type JobStore struct {
	mu       sync.RWMutex
	jobs     map[string]*Job
	dataDir  string
	filePath string
}

// NewJobStore creates a new job store.
func NewJobStore(dataDir string) *JobStore {
	s := &JobStore{
		jobs:     make(map[string]*Job),
		dataDir:  dataDir,
		filePath: filepath.Join(dataDir, "media_jobs.json"),
	}
	s.load()
	return s
}

// Create adds a new job.
func (s *JobStore) Create(job *Job) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.jobs[job.ID] = job
	s.saveLocked()
}

// Get returns a job by ID.
func (s *JobStore) Get(id string) *Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.jobs[id]
}

// Update modifies a job.
func (s *JobStore) Update(id string, fn func(*Job)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if job, ok := s.jobs[id]; ok {
		fn(job)
		s.saveLocked()
	}
}

// List returns all jobs, newest first.
func (s *JobStore) List(limit int) []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		result = append(result, j)
	}
	// Sort by created_at desc (simple bubble sort for small lists)
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].CreatedAt.After(result[i].CreatedAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	if limit > 0 && len(result) > limit {
		result = result[:limit]
	}
	return result
}

// CleanOld removes jobs older than the given duration.
func (s *JobStore) CleanOld(maxAge time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	cutoff := time.Now().Add(-maxAge)
	removed := 0
	for id, job := range s.jobs {
		if job.CreatedAt.Before(cutoff) {
			delete(s.jobs, id)
			removed++
		}
	}
	if removed > 0 {
		s.saveLocked()
	}
	return removed
}

func (s *JobStore) load() {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return // File doesn't exist yet
	}
	var store struct {
		Jobs map[string]*Job `json:"jobs"`
	}
	if err := json.Unmarshal(data, &store); err != nil {
		log.Printf("[media] Failed to parse jobs file: %v", err)
		return
	}
	if store.Jobs != nil {
		s.jobs = store.Jobs
	}
}

func (s *JobStore) saveLocked() {
	store := struct {
		Jobs map[string]*Job `json:"jobs"`
	}{Jobs: s.jobs}

	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		log.Printf("[media] Failed to marshal jobs: %v", err)
		return
	}

	dir := filepath.Dir(s.filePath)
	os.MkdirAll(dir, 0750)

	if err := os.WriteFile(s.filePath, data, 0600); err != nil {
		log.Printf("[media] Failed to save jobs: %v", err)
	}
}
