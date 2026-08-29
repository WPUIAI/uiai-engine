package evidenceartifact

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"sync"
	"time"
)

const StoreSchemaV1 = "uiai.evidence_artifact_store.v1"

var (
	ErrStoreConfig      = errors.New("invalid evidence artifact store configuration")
	ErrStoreLocked      = errors.New("evidence artifact store locked")
	ErrStoreCorrupt     = errors.New("evidence artifact store corrupt")
	ErrStoreUnavailable = errors.New("evidence artifact store unavailable")
	ErrArtifactNotFound = errors.New("evidence artifact not found")
	ErrAssetNotFound    = errors.New("evidence artifact asset not found")
	ErrAssetMismatch    = errors.New("evidence artifact asset mismatch")
	ErrRevisionConflict = errors.New("evidence artifact revision conflict")
	ErrQuotaExceeded    = errors.New("evidence artifact store quota exceeded")
	ErrRetentionBlocked = errors.New("evidence artifact retention blocked")
	ErrOutcomeUnknown   = errors.New("evidence artifact store outcome unknown")
	ErrBackupInvalid    = errors.New("evidence artifact backup invalid")
)

type StoreConfig struct {
	Root                 string
	MaxStoreBytes        int64
	MaxArtifacts         int
	MaxAssetBytes        int64
	StagingQuarantineAge time.Duration
	GCGrace              time.Duration
	Now                  func() time.Time
	AdmissionProbe       func(Admission) error
}

type Admission struct {
	CurrentBytes       int64
	AdditionalBytes    int64
	ProjectedBytes     int64
	CurrentArtifacts   int
	ProjectedArtifacts int
}

type Store struct {
	mu     sync.Mutex
	cfg    StoreConfig
	index  storeIndex
	health StoreHealth
	fault  func(string) error
}

type CommitResult struct {
	ArtifactID     string `json:"artifact_id"`
	Revision       uint64 `json:"revision"`
	ManifestSHA256 string `json:"manifest_sha256"`
	CommitID       string `json:"commit_id"`
	CommittedAt    string `json:"committed_at"`
	Deduplicated   bool   `json:"deduplicated"`
}

type Entry struct {
	ArtifactID     string         `json:"artifact_id"`
	Revision       uint64         `json:"revision"`
	ManifestSHA256 string         `json:"manifest_sha256"`
	CommitID       string         `json:"commit_id"`
	CommittedAt    string         `json:"committed_at"`
	RetentionClass RetentionClass `json:"retention_class"`
	ExpiresAt      string         `json:"expires_at,omitempty"`
	Assets         []AssetRecord  `json:"assets"`
}

type AssetRecord struct {
	AssetID   string `json:"asset_id"`
	SHA256    string `json:"sha256"`
	ByteSize  int64  `json:"byte_size"`
	MediaType string `json:"media_type"`
}

type CommitRecord struct {
	Schema         string        `json:"schema"`
	CommitID       string        `json:"commit_id"`
	ArtifactID     string        `json:"artifact_id"`
	Revision       uint64        `json:"revision"`
	ManifestSHA256 string        `json:"manifest_sha256"`
	CommittedAt    string        `json:"committed_at"`
	Assets         []AssetRecord `json:"assets"`
}

type Tombstone struct {
	Schema       string `json:"schema"`
	TombstoneID  string `json:"tombstone_id"`
	ArtifactID   string `json:"artifact_id"`
	Revision     uint64 `json:"revision"`
	CommitID     string `json:"commit_id"`
	Reason       string `json:"reason"`
	AuthorityRef string `json:"authority_ref"`
	CreatedAt    string `json:"created_at"`
}

type StoreHealth struct {
	Schema              string `json:"schema"`
	Status              string `json:"status"`
	LiveArtifacts       int    `json:"live_artifacts"`
	TombstonedArtifacts int    `json:"tombstoned_artifacts"`
	CorruptRecords      int    `json:"corrupt_records"`
	FreshStaging        int    `json:"fresh_staging"`
	Quarantined         int    `json:"quarantined"`
	StoredBytes         int64  `json:"stored_bytes"`
	IndexGeneration     uint64 `json:"index_generation"`
}

type GCResult struct {
	RetiredCommits   int   `json:"retired_commits"`
	RemovedManifests int   `json:"removed_manifests"`
	RemovedBlobs     int   `json:"removed_blobs"`
	FreedBytes       int64 `json:"freed_bytes"`
}

type storeIndex struct {
	Schema     string      `json:"schema"`
	Generation uint64      `json:"generation"`
	Entries    []Entry     `json:"entries"`
	Tombstones []Tombstone `json:"tombstones"`
}

func OpenStore(cfg StoreConfig) (*Store, StoreHealth, error) {
	if cfg.Root == "" || cfg.MaxStoreBytes <= 0 || cfg.MaxArtifacts <= 0 || cfg.MaxAssetBytes <= 0 || cfg.StagingQuarantineAge <= 0 || cfg.GCGrace < 0 {
		return nil, StoreHealth{}, ErrStoreConfig
	}
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return time.Now().UTC() }
	}
	s := &Store{cfg: cfg}
	if err := s.initializeLayout(); err != nil {
		return nil, StoreHealth{}, err
	}
	health, err := s.Reconcile()
	if err != nil {
		return nil, health, err
	}
	return s, health, nil
}

func (s *Store) Commit(ctx context.Context, manifest Manifest, assets map[string]io.Reader) (CommitResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.commitLocked(ctx, manifest, assets)
}

func (s *Store) GetManifest(artifactID string, revision uint64) (Manifest, Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.entryLocked(artifactID, revision)
	if !ok {
		return Manifest{}, Entry{}, ErrArtifactNotFound
	}
	manifest, err := s.readManifest(entry.ManifestSHA256)
	if err != nil {
		return Manifest{}, Entry{}, err
	}
	return manifest, entry, nil
}

func (s *Store) OpenAsset(digest string) (*os.File, AssetRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !validSHA256(digest) {
		return nil, AssetRecord{}, ErrAssetNotFound
	}
	var record AssetRecord
	found := false
	for _, entry := range s.index.Entries {
		for _, asset := range entry.Assets {
			if asset.SHA256 == digest {
				record, found = asset, true
				break
			}
		}
	}
	if !found {
		return nil, AssetRecord{}, ErrAssetNotFound
	}
	path := s.blobPath(digest)
	actualDigest, actualSize, err := hashFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, AssetRecord{}, ErrAssetNotFound
		}
		return nil, AssetRecord{}, storeError(ErrStoreUnavailable, "verify asset")
	}
	if actualDigest != digest || actualSize != record.ByteSize {
		return nil, AssetRecord{}, ErrStoreCorrupt
	}
	file, err := os.Open(path) // #nosec G304 -- digest is validated and path is rooted.
	if err != nil {
		return nil, AssetRecord{}, storeError(ErrStoreUnavailable, "open asset")
	}
	return file, record, nil
}

func (s *Store) List() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := append([]Entry(nil), s.index.Entries...)
	for i := range out {
		out[i].Assets = append([]AssetRecord(nil), out[i].Assets...)
	}
	return out
}

func (s *Store) Health() StoreHealth {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.health
}

func (s *Store) Reconcile() (StoreHealth, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.reconcileLocked()
}

func (s *Store) entryLocked(artifactID string, revision uint64) (Entry, bool) {
	i := sort.Search(len(s.index.Entries), func(i int) bool {
		if s.index.Entries[i].ArtifactID == artifactID {
			return s.index.Entries[i].Revision >= revision
		}
		return s.index.Entries[i].ArtifactID >= artifactID
	})
	if i < len(s.index.Entries) && s.index.Entries[i].ArtifactID == artifactID && s.index.Entries[i].Revision == revision {
		return s.index.Entries[i], true
	}
	return Entry{}, false
}

func (s *Store) callFault(stage string) error {
	if s.fault == nil {
		return nil
	}
	return s.fault(stage)
}

func storeError(base error, field string) error {
	return fmt.Errorf("%w: %s", base, field)
}
