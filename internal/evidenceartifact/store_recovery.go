package evidenceartifact

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (s *Store) reconcileLocked() (StoreHealth, error) {
	health := StoreHealth{Schema: StoreSchemaV1, Status: "healthy"}
	fresh, quarantined, err := s.reconcileStagingLocked()
	if err != nil {
		return health, storeError(ErrStoreUnavailable, "staging recovery")
	}
	health.FreshStaging = fresh
	health.Quarantined = quarantined

	tombstones, corruptTombstones, err := s.scanTombstonesLocked()
	if err != nil {
		return health, err
	}
	tombstoned := make(map[string]Tombstone, len(tombstones))
	for _, tombstone := range tombstones {
		tombstoned[tombstone.CommitID] = tombstone
	}

	commitFiles, err := listFiles(filepath.Join(s.cfg.Root, "commits", "sha256"))
	if err != nil {
		return health, storeError(ErrStoreUnavailable, "scan commits")
	}
	type commitCandidate struct {
		path     string
		record   CommitRecord
		manifest Manifest
	}
	candidates := make([]commitCandidate, 0, len(commitFiles))
	groups := make(map[string][]int, len(commitFiles))
	corrupt := corruptTombstones
	for _, commitPath := range commitFiles {
		record, manifest, err := s.validateCommitFile(commitPath)
		if err != nil {
			digest := deterministicID(filepath.Base(commitPath), "corrupt")
			if quarantineErr := s.quarantinePath(commitPath, "commit", digest); quarantineErr != nil {
				return health, storeError(ErrStoreUnavailable, "quarantine commit")
			}
			corrupt++
			continue
		}
		index := len(candidates)
		candidates = append(candidates, commitCandidate{path: commitPath, record: record, manifest: manifest})
		key := record.ArtifactID + "\x00" + uintString(record.Revision)
		groups[key] = append(groups[key], index)
	}
	candidateByCommit := make(map[string]commitCandidate, len(candidates))
	for _, candidate := range candidates {
		candidateByCommit[candidate.record.CommitID] = candidate
	}
	filteredTombstones := tombstones[:0]
	for _, tombstone := range tombstones {
		candidate, ok := candidateByCommit[tombstone.CommitID]
		if ok && candidate.manifest.Policy.RetentionClass == RetentionLegalHold {
			digest := deterministicID(tombstone.TombstoneID, "legal-hold")
			if err := s.quarantinePath(s.tombstonePath(tombstone.TombstoneID), "legal-hold-tombstone", digest); err != nil {
				return health, storeError(ErrStoreUnavailable, "quarantine legal hold tombstone")
			}
			delete(tombstoned, tombstone.CommitID)
			corrupt++
			continue
		}
		filteredTombstones = append(filteredTombstones, tombstone)
	}
	tombstones = filteredTombstones

	conflicts := make(map[int]struct{})
	for _, indexes := range groups {
		if len(indexes) > 1 {
			for _, index := range indexes {
				conflicts[index] = struct{}{}
			}
		}
	}
	entries := make([]Entry, 0, len(candidates))
	validCommitIDs := make(map[string]struct{}, len(candidates))
	for index, candidate := range candidates {
		if _, conflict := conflicts[index]; conflict {
			digest := deterministicID(candidate.record.CommitID, "revision-conflict")
			if err := s.quarantinePath(candidate.path, "revision-conflict", digest); err != nil {
				return health, storeError(ErrStoreUnavailable, "quarantine revision conflict")
			}
			corrupt++
			continue
		}
		record, manifest := candidate.record, candidate.manifest
		validCommitIDs[record.CommitID] = struct{}{}
		if _, hidden := tombstoned[record.CommitID]; hidden {
			continue
		}
		entries = append(entries, Entry{
			ArtifactID: record.ArtifactID, Revision: record.Revision, ManifestSHA256: record.ManifestSHA256,
			CommitID: record.CommitID, CommittedAt: record.CommittedAt, RetentionClass: manifest.Policy.RetentionClass,
			ExpiresAt: manifest.Policy.ExpiresAt, Assets: append([]AssetRecord(nil), record.Assets...),
		})
	}
	for commitID, tombstone := range tombstoned {
		if _, ok := validCommitIDs[commitID]; ok {
			continue
		}
		if _, err := os.Stat(s.retiredCommitPath(commitID)); err == nil {
			continue
		}
		path := s.tombstonePath(tombstone.TombstoneID)
		digest := deterministicID(tombstone.TombstoneID, "orphan")
		if err := s.quarantinePath(path, "tombstone", digest); err != nil {
			return health, storeError(ErrStoreUnavailable, "quarantine orphan tombstone")
		}
		corrupt++
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].ArtifactID == entries[j].ArtifactID {
			return entries[i].Revision < entries[j].Revision
		}
		return entries[i].ArtifactID < entries[j].ArtifactID
	})
	sort.Slice(tombstones, func(i, j int) bool {
		if tombstones[i].ArtifactID == tombstones[j].ArtifactID {
			return tombstones[i].Revision < tombstones[j].Revision
		}
		return tombstones[i].ArtifactID < tombstones[j].ArtifactID
	})
	index := storeIndex{
		Schema: StoreSchemaV1, Generation: uint64(len(entries) + len(tombstones)),
		Entries: entries, Tombstones: tombstones,
	}
	if err := writeAtomicJSON(filepath.Join(s.cfg.Root, "index", "index.v1.json"), index, 0o640); err != nil {
		return health, storeError(ErrStoreUnavailable, "write index")
	}
	storedBytes, err := directoryBytes(s.cfg.Root)
	if err != nil {
		return health, storeError(ErrStoreUnavailable, "health bytes")
	}
	quarantineEntries, err := os.ReadDir(filepath.Join(s.cfg.Root, "quarantine"))
	if err != nil {
		return health, storeError(ErrStoreUnavailable, "read quarantine")
	}
	health.LiveArtifacts = len(entries)
	health.TombstonedArtifacts = len(tombstones)
	health.CorruptRecords = corrupt
	health.Quarantined = len(quarantineEntries)
	health.StoredBytes = storedBytes
	health.IndexGeneration = index.Generation
	if corrupt > 0 || health.Quarantined > 0 || fresh > 0 {
		health.Status = "degraded"
	}
	s.index = index
	s.health = health
	return health, nil
}

func (s *Store) validateCommitFile(path string) (CommitRecord, Manifest, error) {
	var record CommitRecord
	if err := readJSONFile(path, 1024*1024, &record); err != nil {
		return record, Manifest{}, err
	}
	if record.Schema != StoreSchemaV1 || !validSHA256(record.CommitID) || !validRef(record.ArtifactID, true) || record.Revision == 0 || !validSHA256(record.ManifestSHA256) {
		return record, Manifest{}, ErrStoreCorrupt
	}
	if record.CommitID != deterministicID(record.ArtifactID, uintString(record.Revision), record.ManifestSHA256) {
		return record, Manifest{}, ErrStoreCorrupt
	}
	if _, ok := canonicalTime(record.CommittedAt, true); !ok {
		return record, Manifest{}, ErrStoreCorrupt
	}
	manifest, err := s.readManifest(record.ManifestSHA256)
	if err != nil {
		return record, Manifest{}, err
	}
	if manifest.ArtifactID != record.ArtifactID || manifest.Revision != record.Revision || manifest.Integrity.ManifestSHA256 != record.ManifestSHA256 {
		return record, Manifest{}, ErrStoreCorrupt
	}
	if len(record.Assets) != len(manifest.Assets) {
		return record, Manifest{}, ErrStoreCorrupt
	}
	manifestAssets := make(map[string]Asset, len(manifest.Assets))
	for _, asset := range manifest.Assets {
		manifestAssets[asset.AssetID] = asset
	}
	seen := make(map[string]struct{}, len(record.Assets))
	for _, asset := range record.Assets {
		manifestAsset, ok := manifestAssets[asset.AssetID]
		if !ok || asset.SHA256 != manifestAsset.SHA256 || asset.ByteSize != manifestAsset.ByteSize || asset.MediaType != manifestAsset.MediaType {
			return record, Manifest{}, ErrStoreCorrupt
		}
		if _, duplicate := seen[asset.AssetID]; duplicate {
			return record, Manifest{}, ErrStoreCorrupt
		}
		seen[asset.AssetID] = struct{}{}
		digest, size, err := hashFile(s.blobPath(asset.SHA256))
		if err != nil || digest != asset.SHA256 || size != asset.ByteSize {
			return record, Manifest{}, ErrStoreCorrupt
		}
	}
	return record, manifest, nil
}

func (s *Store) readManifest(digest string) (Manifest, error) {
	if !validSHA256(digest) {
		return Manifest{}, ErrStoreCorrupt
	}
	var manifest Manifest
	if err := readJSONFile(s.manifestPath(digest), MaxManifestBytes, &manifest); err != nil {
		if os.IsNotExist(err) {
			return Manifest{}, ErrStoreCorrupt
		}
		return Manifest{}, storeError(ErrStoreCorrupt, "manifest JSON")
	}
	if manifest.Integrity.ManifestSHA256 != digest {
		return Manifest{}, ErrStoreCorrupt
	}
	if err := VerifyManifestSHA256(manifest); err != nil {
		return Manifest{}, ErrStoreCorrupt
	}
	return Normalize(manifest), nil
}

func (s *Store) scanTombstonesLocked() ([]Tombstone, int, error) {
	files, err := listFiles(filepath.Join(s.cfg.Root, "tombstones", "sha256"))
	if err != nil {
		return nil, 0, storeError(ErrStoreUnavailable, "scan tombstones")
	}
	out := make([]Tombstone, 0, len(files))
	corrupt := 0
	for _, path := range files {
		var tombstone Tombstone
		if err := readJSONFile(path, 1024*1024, &tombstone); err != nil || !validTombstone(tombstone) {
			digest := deterministicID(filepath.Base(path), "corrupt")
			if err := s.quarantinePath(path, "tombstone", digest); err != nil {
				return nil, corrupt, storeError(ErrStoreUnavailable, "quarantine tombstone")
			}
			corrupt++
			continue
		}
		out = append(out, tombstone)
	}
	return out, corrupt, nil
}

func validTombstone(t Tombstone) bool {
	if t.Schema != StoreSchemaV1 || !validSHA256(t.TombstoneID) || !validRef(t.ArtifactID, true) || t.Revision == 0 || !validSHA256(t.CommitID) || !validText(t.Reason, 500, true) || !validRef(t.AuthorityRef, true) {
		return false
	}
	if t.TombstoneID != deterministicID(t.ArtifactID, uintString(t.Revision)) {
		return false
	}
	_, ok := canonicalTime(t.CreatedAt, true)
	return ok
}

func (s *Store) reconcileStagingLocked() (int, int, error) {
	entries, err := os.ReadDir(filepath.Join(s.cfg.Root, "staging"))
	if err != nil {
		return 0, 0, err
	}
	fresh, quarantined := 0, 0
	cutoff := s.cfg.Now().Add(-s.cfg.StagingQuarantineAge)
	for _, entry := range entries {
		path := filepath.Join(s.cfg.Root, "staging", entry.Name())
		info, err := entry.Info()
		if err != nil {
			return fresh, quarantined, err
		}
		if info.ModTime().After(cutoff) {
			fresh++
			continue
		}
		digest := deterministicID(entry.Name(), info.ModTime().UTC().Format(time.RFC3339Nano))
		if err := s.quarantinePath(path, "staging", digest); err != nil {
			return fresh, quarantined, err
		}
		quarantined++
	}
	return fresh, quarantined, nil
}

func decodeIndex(data []byte) (storeIndex, error) {
	var index storeIndex
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&index); err != nil {
		return storeIndex{}, err
	}
	if index.Schema != StoreSchemaV1 {
		return storeIndex{}, ErrStoreCorrupt
	}
	return index, nil
}

func isStoreClass(err, target error) bool {
	return errors.Is(err, target)
}
