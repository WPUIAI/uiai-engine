package evidenceartifact

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Store) Tombstone(artifactID string, revision uint64, reason, authorityRef string) (Tombstone, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !validRef(artifactID, true) || revision == 0 || !validText(reason, 500, true) || !validRef(authorityRef, true) {
		return Tombstone{}, ErrRetentionBlocked
	}
	tombstoneID := deterministicID(artifactID, uintString(revision))
	var existing Tombstone
	if err := readJSONFile(s.tombstonePath(tombstoneID), 1024*1024, &existing); err == nil {
		if validTombstone(existing) {
			return existing, nil
		}
		return Tombstone{}, ErrStoreCorrupt
	} else if !os.IsNotExist(err) {
		return Tombstone{}, storeError(ErrStoreUnavailable, "read tombstone")
	}
	entry, ok := s.entryLocked(artifactID, revision)
	if !ok {
		return Tombstone{}, ErrArtifactNotFound
	}
	manifest, err := s.readManifest(entry.ManifestSHA256)
	if err != nil {
		return Tombstone{}, err
	}
	if manifest.Policy.RetentionClass == RetentionLegalHold {
		return Tombstone{}, ErrRetentionBlocked
	}
	tombstone := Tombstone{
		Schema: StoreSchemaV1, TombstoneID: tombstoneID, ArtifactID: artifactID, Revision: revision,
		CommitID: entry.CommitID, Reason: reason, AuthorityRef: authorityRef,
		CreatedAt: s.cfg.Now().UTC().Format(time.RFC3339Nano),
	}
	if err := writeAtomicJSON(s.tombstonePath(tombstoneID), tombstone, 0o640); err != nil {
		return Tombstone{}, storeError(ErrStoreUnavailable, "write tombstone")
	}
	if _, err := s.reconcileLocked(); err != nil {
		return tombstone, ErrOutcomeUnknown
	}
	return tombstone, nil
}

func (s *Store) GC() (GCResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result := GCResult{}
	mutated := false
	fail := func(base error, field string) error {
		if mutated {
			return ErrOutcomeUnknown
		}
		return storeError(base, field)
	}
	cutoff := s.cfg.Now().Add(-s.cfg.GCGrace)
	for _, tombstone := range s.index.Tombstones {
		created, ok := canonicalTime(tombstone.CreatedAt, true)
		if !ok {
			return result, ErrStoreCorrupt
		}
		if created.After(cutoff) {
			continue
		}
		source := s.commitPath(tombstone.CommitID)
		target := s.retiredCommitPath(tombstone.CommitID)
		if _, err := os.Stat(target); err == nil {
			continue
		} else if !os.IsNotExist(err) {
			return result, fail(ErrStoreUnavailable, "inspect retired commit")
		}
		if _, err := os.Stat(source); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return result, fail(ErrStoreUnavailable, "inspect commit")
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return result, fail(ErrStoreUnavailable, "create retired layout")
		}
		if err := os.Rename(source, target); err != nil {
			return result, fail(ErrStoreUnavailable, "retire commit")
		}
		mutated = true
		if err := syncDir(filepath.Dir(source)); err != nil {
			return result, ErrOutcomeUnknown
		}
		if err := syncDir(filepath.Dir(target)); err != nil {
			return result, ErrOutcomeUnknown
		}
		result.RetiredCommits++
	}

	manifestRefs, blobRefs, err := s.currentReferencesLocked()
	if err != nil {
		if mutated {
			return result, ErrOutcomeUnknown
		}
		return result, err
	}
	manifestFiles, err := listFiles(filepath.Join(s.cfg.Root, "manifests", "sha256"))
	if err != nil {
		return result, fail(ErrStoreUnavailable, "scan manifests for GC")
	}
	for _, path := range manifestFiles {
		digest := strings.TrimSuffix(filepath.Base(path), ".json")
		if !validSHA256(digest) {
			return result, ErrStoreCorrupt
		}
		if _, live := manifestRefs[digest]; live {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			return result, fail(ErrStoreUnavailable, "stat manifest for GC")
		}
		if err := os.Remove(path); err != nil {
			return result, fail(ErrStoreUnavailable, "remove manifest")
		}
		mutated = true
		if err := syncDir(filepath.Dir(path)); err != nil {
			return result, ErrOutcomeUnknown
		}
		result.RemovedManifests++
		result.FreedBytes += info.Size()
	}
	blobFiles, err := listFiles(filepath.Join(s.cfg.Root, "blobs", "sha256"))
	if err != nil {
		return result, fail(ErrStoreUnavailable, "scan blobs for GC")
	}
	for _, path := range blobFiles {
		digest := filepath.Base(path)
		if !validSHA256(digest) {
			return result, ErrStoreCorrupt
		}
		if _, live := blobRefs[digest]; live {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			return result, fail(ErrStoreUnavailable, "stat blob for GC")
		}
		if err := os.Remove(path); err != nil {
			return result, fail(ErrStoreUnavailable, "remove blob")
		}
		mutated = true
		if err := syncDir(filepath.Dir(path)); err != nil {
			return result, ErrOutcomeUnknown
		}
		result.RemovedBlobs++
		result.FreedBytes += info.Size()
	}
	if _, err := s.reconcileLocked(); err != nil {
		return result, ErrOutcomeUnknown
	}
	return result, nil
}

func (s *Store) currentReferencesLocked() (map[string]struct{}, map[string]struct{}, error) {
	manifestRefs := make(map[string]struct{})
	blobRefs := make(map[string]struct{})
	files, err := listFiles(filepath.Join(s.cfg.Root, "commits", "sha256"))
	if err != nil {
		return nil, nil, storeError(ErrStoreUnavailable, "scan live references")
	}
	for _, path := range files {
		record, _, err := s.validateCommitFile(path)
		if err != nil {
			return nil, nil, ErrStoreCorrupt
		}
		manifestRefs[record.ManifestSHA256] = struct{}{}
		for _, asset := range record.Assets {
			if !validSHA256(asset.SHA256) {
				return nil, nil, ErrStoreCorrupt
			}
			blobRefs[asset.SHA256] = struct{}{}
		}
	}
	return manifestRefs, blobRefs, nil
}
