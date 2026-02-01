package storage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type UsageRecord struct {
	Type         string  `json:"type"`
	Model        string  `json:"model"`
	InputTokens  int     `json:"inputTokens"`
	OutputTokens int     `json:"outputTokens"`
	CostUSD      float64 `json:"costUsd"`
	Status       string  `json:"status"`
	CreatedAt    string  `json:"createdAt"`
}

type UsageStore struct {
	path    string
	records []UsageRecord
	mu      sync.Mutex
}

func NewUsageStore(dataDir, filename string) *UsageStore {
	path := filepath.Join(dataDir, filename)
	s := &UsageStore{path: path}
	s.load()
	return s
}

func (s *UsageStore) Record(r UsageRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, r)
	s.save()
}

func (s *UsageStore) All() []UsageRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]UsageRecord, len(s.records))
	copy(out, s.records)
	return out
}

func (s *UsageStore) ByType(t string) []UsageRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []UsageRecord
	for _, r := range s.records {
		if r.Type == t {
			out = append(out, r)
		}
	}
	return out
}

func (s *UsageStore) load() {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return
	}
	json.Unmarshal(data, &s.records)
}

func (s *UsageStore) save() {
	data, _ := json.MarshalIndent(s.records, "", "  ")
	os.MkdirAll(filepath.Dir(s.path), 0755)
	os.WriteFile(s.path, data, 0644)
}
