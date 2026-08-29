package evidenceartifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"
)

func (s *Store) commitLocked(ctx context.Context, manifest Manifest, inputs map[string]io.Reader) (CommitResult, error) {
	manifest = Normalize(manifest)
	if err := Validate(manifest); err != nil {
		return CommitResult{}, err
	}
	if err := VerifyManifestSHA256(manifest); err != nil {
		return CommitResult{}, err
	}
	if _, err := s.reconcileLocked(); err != nil {
		return CommitResult{}, err
	}
	for _, tombstone := range s.index.Tombstones {
		if tombstone.ArtifactID == manifest.ArtifactID && tombstone.Revision == manifest.Revision {
			return CommitResult{}, ErrRetentionBlocked
		}
	}
	commitID := deterministicID(manifest.ArtifactID, uintString(manifest.Revision), manifest.Integrity.ManifestSHA256)
	if entry, ok := s.entryLocked(manifest.ArtifactID, manifest.Revision); ok {
		if entry.ManifestSHA256 != manifest.Integrity.ManifestSHA256 {
			return CommitResult{}, ErrRevisionConflict
		}
		return CommitResult{ArtifactID: entry.ArtifactID, Revision: entry.Revision, ManifestSHA256: entry.ManifestSHA256, CommitID: entry.CommitID, CommittedAt: entry.CommittedAt, Deduplicated: true}, nil
	}
	if len(inputs) != len(manifest.Assets) {
		return CommitResult{}, storeError(ErrAssetMismatch, "asset input count")
	}
	assetByID := make(map[string]Asset, len(manifest.Assets))
	for _, asset := range manifest.Assets {
		assetByID[asset.AssetID] = asset
	}
	for id, reader := range inputs {
		if _, ok := assetByID[id]; !ok || reader == nil {
			return CommitResult{}, storeError(ErrAssetMismatch, "asset input binding")
		}
	}
	sealedBytes, err := json.Marshal(manifest)
	if err != nil {
		return CommitResult{}, storeError(ErrStoreUnavailable, "marshal manifest")
	}
	sealedBytes = append(sealedBytes, '\n')
	additional := int64(len(sealedBytes) + 4096)
	for _, asset := range manifest.Assets {
		if asset.ByteSize > s.cfg.MaxAssetBytes {
			return CommitResult{}, ErrQuotaExceeded
		}
		if _, err := os.Stat(s.blobPath(asset.SHA256)); os.IsNotExist(err) {
			additional += asset.ByteSize
		} else if err != nil {
			return CommitResult{}, storeError(ErrStoreUnavailable, "asset admission")
		}
	}
	currentBytes, err := directoryBytes(s.cfg.Root)
	if err != nil {
		return CommitResult{}, storeError(ErrStoreUnavailable, "quota scan")
	}
	admission := Admission{
		CurrentBytes: currentBytes, AdditionalBytes: additional, ProjectedBytes: currentBytes + additional,
		CurrentArtifacts: len(s.index.Entries), ProjectedArtifacts: len(s.index.Entries) + 1,
	}
	if admission.ProjectedBytes > s.cfg.MaxStoreBytes || admission.ProjectedArtifacts > s.cfg.MaxArtifacts {
		return CommitResult{}, ErrQuotaExceeded
	}
	if s.cfg.AdmissionProbe != nil {
		if err := s.cfg.AdmissionProbe(admission); err != nil {
			return CommitResult{}, ErrQuotaExceeded
		}
	}
	txID, err := randomID()
	if err != nil {
		return CommitResult{}, storeError(ErrStoreUnavailable, "transaction identity")
	}
	staging := filepath.Join(s.cfg.Root, "staging", txID)
	if err := os.Mkdir(staging, 0o750); err != nil {
		return CommitResult{}, storeError(ErrStoreUnavailable, "create staging")
	}
	assetRecords := make([]AssetRecord, 0, len(manifest.Assets))
	stageAssets := make(map[string]string, len(manifest.Assets))
	for _, asset := range manifest.Assets {
		if err := ctx.Err(); err != nil {
			return CommitResult{}, storeError(ErrStoreUnavailable, "context cancelled")
		}
		stagePath := filepath.Join(staging, "asset-"+asset.SHA256+"-"+deterministicID(asset.AssetID)[:12])
		if err := writeAsset(ctx, stagePath, inputs[asset.AssetID], asset, s.cfg.MaxAssetBytes); err != nil {
			return CommitResult{}, err
		}
		stageAssets[asset.AssetID] = stagePath
		assetRecords = append(assetRecords, AssetRecord{AssetID: asset.AssetID, SHA256: asset.SHA256, ByteSize: asset.ByteSize, MediaType: asset.MediaType})
	}
	if err := s.callFault("after_assets_staged"); err != nil {
		return CommitResult{}, storeError(ErrStoreUnavailable, "after assets staged")
	}
	inspections := make([]InspectionRecord, 0, len(manifest.Assets))
	for _, asset := range manifest.Assets {
		record, err := s.inspector.Inspect(ctx, InspectionRequest{
			Path: stageAssets[asset.AssetID], Asset: asset, Security: manifest.Security, Policy: manifest.Policy,
		})
		if err != nil {
			return CommitResult{}, err
		}
		if !validInspectionRecord(record) || record.AssetID != asset.AssetID || record.InspectionSHA256 != inspectionDigest(record, asset) {
			return CommitResult{}, ErrInspectionFailed
		}
		inspections = append(inspections, record)
	}
	sort.Slice(inspections, func(i, j int) bool { return inspections[i].AssetID < inspections[j].AssetID })
	manifestStage := filepath.Join(staging, "manifest.json")
	if err := writeSyncedFile(manifestStage, sealedBytes, 0o600); err != nil {
		return CommitResult{}, storeError(ErrStoreUnavailable, "stage manifest")
	}
	storageManifestHash := sha256.Sum256(sealedBytes)
	committedAt := s.cfg.Now().UTC().Format(time.RFC3339Nano)
	record := CommitRecord{
		Schema: StoreSchemaV1, CommitID: commitID, ArtifactID: manifest.ArtifactID, Revision: manifest.Revision,
		ManifestSHA256: manifest.Integrity.ManifestSHA256, CommittedAt: committedAt, Assets: assetRecords, Inspections: inspections,
	}
	sort.Slice(record.Assets, func(i, j int) bool { return record.Assets[i].AssetID < record.Assets[j].AssetID })
	commitBytes, err := json.Marshal(record)
	if err != nil {
		return CommitResult{}, storeError(ErrStoreUnavailable, "marshal commit")
	}
	commitBytes = append(commitBytes, '\n')
	commitStage := filepath.Join(staging, "commit.json")
	if err := writeSyncedFile(commitStage, commitBytes, 0o600); err != nil {
		return CommitResult{}, storeError(ErrStoreUnavailable, "stage commit")
	}
	if err := syncDir(staging); err != nil {
		return CommitResult{}, storeError(ErrStoreUnavailable, "sync staging")
	}
	for _, asset := range record.Assets {
		if _, err := promoteFile(stageAssets[asset.AssetID], s.blobPath(asset.SHA256), asset.SHA256, asset.ByteSize); err != nil {
			if errors.Is(err, ErrAssetMismatch) {
				return CommitResult{}, storeError(ErrAssetMismatch, "promote asset")
			}
			return CommitResult{}, storeError(ErrStoreUnavailable, "promote asset")
		}
	}
	if _, err := promoteFile(manifestStage, s.manifestPath(manifest.Integrity.ManifestSHA256), hex.EncodeToString(storageManifestHash[:]), int64(len(sealedBytes))); err != nil {
		if errors.Is(err, ErrAssetMismatch) {
			return CommitResult{}, storeError(ErrStoreCorrupt, "manifest collision")
		}
		return CommitResult{}, storeError(ErrStoreUnavailable, "promote manifest")
	}
	if err := s.callFault("before_commit_marker"); err != nil {
		return CommitResult{}, storeError(ErrStoreUnavailable, "before commit marker")
	}
	if err := writeAtomicFile(s.commitPath(commitID), commitBytes, 0o640); err != nil {
		return CommitResult{}, storeError(ErrStoreUnavailable, "commit marker")
	}
	if err := s.callFault("after_commit_marker"); err != nil {
		return commitResult(record, false), ErrOutcomeUnknown
	}
	if _, err := s.reconcileLocked(); err != nil {
		return commitResult(record, false), ErrOutcomeUnknown
	}
	if entry, ok := s.entryLocked(record.ArtifactID, record.Revision); !ok || entry.CommitID != record.CommitID {
		return commitResult(record, false), ErrOutcomeUnknown
	}
	if err := s.callFault("after_index"); err != nil {
		return commitResult(record, false), ErrOutcomeUnknown
	}
	if err := os.RemoveAll(staging); err != nil {
		return commitResult(record, false), ErrOutcomeUnknown
	}
	return commitResult(record, false), nil
}

func writeAsset(ctx context.Context, path string, reader io.Reader, asset Asset, maxBytes int64) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600) // #nosec G304 -- random staging root and hash filename.
	if err != nil {
		return storeError(ErrStoreUnavailable, "create staged asset")
	}
	ok := false
	defer func() {
		_ = file.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	h := sha256.New()
	written := int64(0)
	emptyReads := 0
	buf := make([]byte, 32*1024)
	for {
		if err := ctx.Err(); err != nil {
			return storeError(ErrStoreUnavailable, "asset context cancelled")
		}
		n, readErr := reader.Read(buf)
		if n > 0 {
			emptyReads = 0
			written += int64(n)
			if written > maxBytes || written > asset.ByteSize {
				return ErrAssetMismatch
			}
			if _, err := h.Write(buf[:n]); err != nil {
				return storeError(ErrStoreUnavailable, "hash asset")
			}
			if _, err := file.Write(buf[:n]); err != nil {
				return storeError(ErrStoreUnavailable, "write asset")
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return storeError(ErrStoreUnavailable, "read asset")
		}
		if n == 0 {
			emptyReads++
			if emptyReads > 100 {
				return storeError(ErrStoreUnavailable, "asset reader made no progress")
			}
		}
	}
	if written != asset.ByteSize || hex.EncodeToString(h.Sum(nil)) != asset.SHA256 {
		return ErrAssetMismatch
	}
	if err := file.Sync(); err != nil {
		return storeError(ErrStoreUnavailable, "sync asset")
	}
	if err := file.Close(); err != nil {
		return storeError(ErrStoreUnavailable, "close asset")
	}
	ok = true
	return nil
}

func writeSyncedFile(path string, data []byte, mode os.FileMode) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode) // #nosec G304 -- rooted staging path.
	if err != nil {
		return err
	}
	ok := false
	defer func() {
		_ = file.Close()
		if !ok {
			_ = os.Remove(path)
		}
	}()
	if _, err := file.Write(data); err != nil {
		return err
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	ok = true
	return nil
}

func (s *Store) readCommit(commitID string) (CommitRecord, error) {
	var record CommitRecord
	err := readJSONFile(s.commitPath(commitID), 1024*1024, &record)
	return record, err
}

func commitResult(record CommitRecord, deduplicated bool) CommitResult {
	return CommitResult{
		ArtifactID: record.ArtifactID, Revision: record.Revision, ManifestSHA256: record.ManifestSHA256,
		CommitID: record.CommitID, CommittedAt: record.CommittedAt, Deduplicated: deduplicated,
	}
}

func uintString(value uint64) string {
	const digits = "0123456789"
	if value == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for value > 0 {
		i--
		buf[i] = digits[value%10]
		value /= 10
	}
	return string(buf[i:])
}
