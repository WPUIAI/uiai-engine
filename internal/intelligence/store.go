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
	"regexp"
	"sync"
)

// Store manages disk-backed document and metadata persistence.
// Thread-safe via per-run mutexes + in-memory cache for hot reads.
type Store struct {
	root string       // e.g. data/indexes
	mu   sync.RWMutex // protects the cache maps
	docs map[string][]Document
	meta map[string]*IndexMetadata
}

// NewStore creates a Store rooted at the given directory.
func NewStore(root string) *Store {
	os.MkdirAll(root, 0750)
	return &Store{
		root: root,
		docs: make(map[string][]Document),
		meta: make(map[string]*IndexMetadata),
	}
}

// ---------- Documents ----------

// WriteDocuments persists documents for a run to disk and cache.
func (s *Store) WriteDocuments(runID string, docs []Document) error {
	dir, err := s.ensureRunDir(runID)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "documents.json")
	data, err := json.MarshalIndent(docs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal documents: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil { // #nosec G703 -- path is under ensureRunDir with validated runID and fixed filename.
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

	path, err := s.safePath(runID, "documents.json")
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path) // #nosec G703,G304 -- path is from safePath with validated runID and fixed filename.
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
	dir, err := s.ensureRunDir(runID)
	if err != nil {
		return err
	}
	path := filepath.Join(dir, "metadata.json")
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil { // #nosec G703 -- path is under ensureRunDir with validated runID and fixed filename.
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

	path, err := s.safePath(runID, "metadata.json")
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path) // #nosec G703,G304 -- path is from safePath with validated runID and fixed filename.
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
	path, err := s.safePath(runID, filename)
	if err != nil {
		return false
	}
	_, err = os.Stat(path) // #nosec G703 -- path is from safePath with validated runID and filename.
	return err == nil
}

// ArtifactPath returns the absolute path for an artifact.
func (s *Store) ArtifactPath(runID, filename string) string {
	path, err := s.safePath(runID, filename)
	if err != nil {
		return filepath.Join(s.root, "__invalid__")
	}
	return path
}

// WriteArtifact writes raw bytes to an artifact file.
func (s *Store) WriteArtifact(runID, filename string, data []byte) error {
	dir, err := s.ensureRunDir(runID)
	if err != nil {
		return err
	}
	if err := validateSafeSegment(filename); err != nil {
		return err
	}
	path := filepath.Join(dir, filename)
	return os.WriteFile(path, data, 0600) // #nosec G703 -- path is under ensureRunDir with validated runID and filename.
}

// ---------- Helpers ----------

var safeStoreSegmentRe = regexp.MustCompile(`^[A-Za-z0-9._-]{1,160}$`)

func (s *Store) ensureRunDir(runID string) (string, error) {
	if err := validateSafeSegment(runID); err != nil {
		return "", err
	}
	dir := filepath.Join(s.root, runID)
	if err := os.MkdirAll(dir, 0750); err != nil { // #nosec G703 -- dir is rooted at Store.root with validated runID segment.
		return "", err
	}
	return dir, nil
}

func (s *Store) safePath(runID, filename string) (string, error) {
	if err := validateSafeSegment(runID); err != nil {
		return "", err
	}
	if err := validateSafeSegment(filename); err != nil {
		return "", err
	}
	return filepath.Join(s.root, runID, filename), nil
}

func validateSafeSegment(value string) error {
	if !safeStoreSegmentRe.MatchString(value) || value == "." || value == ".." {
		return fmt.Errorf("unsafe storage path segment: %q", value)
	}
	return nil
}
