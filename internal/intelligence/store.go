// Package intelligence provides the Intelligence Layer:
// document indexing, metadata tracking, weighted search,
// embeddings, tier-gated access, and WASM artifact delivery.
//
// Spec: /home/wpuiai/public_html/wp-content/plugins/wpuiai/docs/INTELLIGENCE_LAYER.md
package intelligence

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// Store manages disk-backed document and metadata persistence.
// Thread-safe via per-run mutexes + in-memory cache for hot reads.
type Store struct {
	root  string            // e.g. data/indexes
	mu    sync.RWMutex      // protects the cache maps
	docs  map[string][]Document
	meta  map[string]*IndexMetadata
}

// NewStore creates a Store rooted at the given directory.
func NewStore(root string) *Store {
	os.MkdirAll(root, 0755)
	return &Store{
		root: root,
		docs: make(map[string][]Document),
		meta: make(map[string]*IndexMetadata),
	}
}

// ---------- Documents ----------

// WriteDocuments persists documents for a run to disk and cache.
func (s *Store) WriteDocuments(runID string, docs []Document) error {
	dir := s.ensureRunDir(runID)
	path := filepath.Join(dir, "documents.json")
	data, err := json.MarshalIndent(docs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal documents: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write documents: %w", err)
	}
	s.mu.Lock()
	s.docs[runID] = docs
	s.mu.Unlock()
	return nil
}

// ReadDocuments returns documents for a run (cache-first, disk fallback).
func (s *Store) ReadDocuments(runID string) ([]Document, error) {
	s.mu.RLock()
	if cached, ok := s.docs[runID]; ok {
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	path := filepath.Join(s.root, runID, "documents.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var docs []Document
	if err := json.Unmarshal(data, &docs); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.docs[runID] = docs
	s.mu.Unlock()
	return docs, nil
}

// ---------- Metadata ----------

// WriteMetadata persists metadata for a run to disk and cache.
func (s *Store) WriteMetadata(runID string, meta *IndexMetadata) error {
	dir := s.ensureRunDir(runID)
	path := filepath.Join(dir, "metadata.json")
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write metadata: %w", err)
	}
	s.mu.Lock()
	s.meta[runID] = meta
	s.mu.Unlock()
	return nil
}

// ReadMetadata returns metadata for a run (cache-first, disk fallback).
func (s *Store) ReadMetadata(runID string) (*IndexMetadata, error) {
	s.mu.RLock()
	if cached, ok := s.meta[runID]; ok {
		s.mu.RUnlock()
		return cached, nil
	}
	s.mu.RUnlock()

	path := filepath.Join(s.root, runID, "metadata.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var meta IndexMetadata
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.meta[runID] = &meta
	s.mu.Unlock()
	return &meta, nil
}

// UpdateMetadata reads existing metadata, applies the updater, and writes back.
func (s *Store) UpdateMetadata(runID string, updater func(*IndexMetadata)) (*IndexMetadata, error) {
	meta, _ := s.ReadMetadata(runID)
	if meta == nil {
		meta = &IndexMetadata{
			RunID:     runID,
			Status:    "queued",
			Artifacts: ArtifactStatus{},
		}
	}
	updater(meta)
	if err := s.WriteMetadata(runID, meta); err != nil {
		return nil, err
	}
	return meta, nil
}

// ---------- Artifacts ----------

// HasArtifact checks if a WASM/JS artifact exists on disk.
func (s *Store) HasArtifact(runID, filename string) bool {
	path := filepath.Join(s.root, runID, filename)
	_, err := os.Stat(path)
	return err == nil
}

// ArtifactPath returns the absolute path for an artifact.
func (s *Store) ArtifactPath(runID, filename string) string {
	return filepath.Join(s.root, runID, filename)
}

// WriteArtifact writes raw bytes to an artifact file.
func (s *Store) WriteArtifact(runID, filename string, data []byte) error {
	dir := s.ensureRunDir(runID)
	path := filepath.Join(dir, filename)
	return os.WriteFile(path, data, 0644)
}

// ---------- Helpers ----------

func (s *Store) ensureRunDir(runID string) string {
	dir := filepath.Join(s.root, runID)
	os.MkdirAll(dir, 0755)
	return dir
}
