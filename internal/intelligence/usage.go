package intelligence

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// UsageRecord tracks embed usage per API key per day.
type UsageRecord struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

// UsageTracker provides disk-backed daily usage tracking for embeddings.
type UsageTracker struct {
	path string
	mu   sync.Mutex
	data map[string]UsageRecord // key → record
}

// NewUsageTracker creates a tracker backed by the given JSON file.
func NewUsageTracker(path string) *UsageTracker {
	t := &UsageTracker{
		path: path,
		data: make(map[string]UsageRecord),
	}
	t.load()
	return t
}

// Get returns the current usage for a key on a given date.
func (t *UsageTracker) Get(key, date string) UsageRecord {
	t.mu.Lock()
	defer t.mu.Unlock()
	if r, ok := t.data[key]; ok && r.Date == date {
		return r
	}
	return UsageRecord{Date: date, Count: 0}
}

// Increment adds count to the key's usage for the given date. Resets if date changed.
func (t *UsageTracker) Increment(key, date string, count int) UsageRecord {
	t.mu.Lock()
	defer t.mu.Unlock()

	existing, ok := t.data[key]
	if !ok || existing.Date != date {
		existing = UsageRecord{Date: date, Count: 0}
	}
	existing.Count += count
	existing.Date = date
	t.data[key] = existing
	t.save()
	return existing
}

func (t *UsageTracker) load() {
	data, err := os.ReadFile(t.path)
	if err != nil {
		return
	}
	json.Unmarshal(data, &t.data)
}

func (t *UsageTracker) save() {
	dir := filepath.Dir(t.path)
	os.MkdirAll(dir, 0755)
	data, _ := json.MarshalIndent(t.data, "", "  ")
	os.WriteFile(t.path, data, 0644)
}
