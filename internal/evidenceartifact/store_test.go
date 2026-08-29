package evidenceartifact

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

type testClock struct{ now time.Time }

func (c *testClock) Now() time.Time      { return c.now }
func (c *testClock) Add(d time.Duration) { c.now = c.now.Add(d) }

func TestStoreCommitReadListAndSeek(t *testing.T) {
	store, _, clock := newTestStore(t)
	payload := []byte("0123456789-evidence")
	manifest := storedManifest(t, "artifact:../../safe", 1, payload, RetentionWorkstream)
	result, err := store.Commit(context.Background(), manifest, readers(manifest, payload))
	if err != nil {
		t.Fatal(err)
	}
	if result.CommitID == "" || result.Deduplicated || result.CommittedAt != clock.Now().Format(time.RFC3339Nano) {
		t.Fatalf("unexpected result: %#v", result)
	}
	got, entry, err := store.GetManifest(manifest.ArtifactID, manifest.Revision)
	if err != nil || got.Integrity.ManifestSHA256 != manifest.Integrity.ManifestSHA256 || entry.CommitID != result.CommitID {
		t.Fatalf("GetManifest() = %#v, %#v, %v", got, entry, err)
	}
	if entries := store.List(); len(entries) != 1 || entries[0].ArtifactID != manifest.ArtifactID {
		t.Fatalf("List() = %#v", entries)
	}
	file, record, err := store.OpenAsset(manifest.Assets[0].SHA256)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.Seek(3, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	part := make([]byte, 4)
	if _, err := io.ReadFull(file, part); err != nil {
		t.Fatal(err)
	}
	if string(part) != "3456" || record.ByteSize != int64(len(payload)) {
		t.Fatalf("range read = %q, record = %#v", part, record)
	}
	if strings.Contains(store.blobPath(record.SHA256), manifest.ArtifactID) {
		t.Fatal("raw artifact ID leaked into storage path")
	}
}

func TestStoreCommitIsIdempotentAndConcurrent(t *testing.T) {
	store, _, _ := newTestStore(t)
	payload := []byte("same payload")
	manifest := storedManifest(t, "artifact:concurrent", 1, payload, RetentionProject)
	const workers = 12
	results := make(chan CommitResult, workers)
	errs := make(chan error, workers)
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := store.Commit(context.Background(), manifest, readers(manifest, payload))
			results <- result
			errs <- err
		}()
	}
	wg.Wait()
	close(results)
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	ids := map[string]struct{}{}
	for result := range results {
		ids[result.CommitID] = struct{}{}
	}
	if len(ids) != 1 || len(store.List()) != 1 {
		t.Fatalf("commit ids=%v entries=%d", ids, len(store.List()))
	}
}

func TestStoreDeduplicatesRepeatedBlobWithinManifest(t *testing.T) {
	store, _, _ := newTestStore(t)
	payload := []byte("same bytes twice")
	manifest := storedManifest(t, "artifact:two-assets", 1, payload, RetentionProject)
	copyAsset := manifest.Assets[0]
	copyAsset.AssetID = "asset:proof-copy"
	copyAsset.Path = "assets/proof-copy.json"
	manifest.Assets = append(manifest.Assets, copyAsset)
	manifest.Claims[0].EvidenceRefs = append(manifest.Claims[0].EvidenceRefs, copyAsset.AssetID)
	manifest.Integrity.ManifestSHA256 = ""
	var err error
	manifest, err = Seal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Commit(context.Background(), manifest, readers(manifest, payload)); err != nil {
		t.Fatal(err)
	}
	files, err := listFiles(filepath.Join(store.cfg.Root, "blobs", "sha256"))
	if err != nil || len(files) != 1 {
		t.Fatalf("blob files=%v err=%v", files, err)
	}
}

func TestStoreQuotaCountsOnlyUniqueBlobBytes(t *testing.T) {
	root := t.TempDir()
	clock := &testClock{now: time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)}
	cfg := testStoreConfig(root, clock)
	var admissions []Admission
	cfg.AdmissionProbe = func(admission Admission) error {
		admissions = append(admissions, admission)
		return nil
	}
	store, _, err := OpenStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	payload := bytes.Repeat([]byte("x"), 4096)
	for _, id := range []string{"artifact:unique-one", "artifact:unique-two"} {
		manifest := storedManifest(t, id, 1, payload, RetentionProject)
		if _, err := store.Commit(context.Background(), manifest, readers(manifest, payload)); err != nil {
			t.Fatal(err)
		}
	}
	if len(admissions) != 2 || admissions[0].AdditionalBytes-admissions[1].AdditionalBytes != int64(len(payload)) {
		t.Fatalf("admissions=%#v", admissions)
	}
}

func TestStoreDeduplicatesSharedBlob(t *testing.T) {
	store, _, _ := newTestStore(t)
	payload := []byte("shared bytes")
	first := storedManifest(t, "artifact:first", 1, payload, RetentionProject)
	second := storedManifest(t, "artifact:second", 1, payload, RetentionProject)
	if _, err := store.Commit(context.Background(), first, readers(first, payload)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Commit(context.Background(), second, readers(second, payload)); err != nil {
		t.Fatal(err)
	}
	files, err := listFiles(filepath.Join(store.cfg.Root, "blobs", "sha256"))
	if err != nil || len(files) != 1 {
		t.Fatalf("blob files=%v err=%v", files, err)
	}
}

func TestStoreRejectsMismatchConflictAndQuota(t *testing.T) {
	t.Run("asset_mismatch", func(t *testing.T) {
		store, _, _ := newTestStore(t)
		manifest := storedManifest(t, "artifact:mismatch", 1, []byte("expected"), RetentionProject)
		_, err := store.Commit(context.Background(), manifest, readers(manifest, []byte("wrong")))
		if !errors.Is(err, ErrAssetMismatch) || len(store.List()) != 0 {
			t.Fatalf("err=%v entries=%v", err, store.List())
		}
	})
	t.Run("revision_conflict", func(t *testing.T) {
		store, _, _ := newTestStore(t)
		first := storedManifest(t, "artifact:conflict", 1, []byte("one"), RetentionProject)
		second := storedManifest(t, "artifact:conflict", 1, []byte("two"), RetentionProject)
		if _, err := store.Commit(context.Background(), first, readers(first, []byte("one"))); err != nil {
			t.Fatal(err)
		}
		if _, err := store.Commit(context.Background(), second, readers(second, []byte("two"))); !errors.Is(err, ErrRevisionConflict) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("artifact_quota", func(t *testing.T) {
		root := t.TempDir()
		clock := &testClock{now: time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)}
		cfg := testStoreConfig(root, clock)
		cfg.MaxArtifacts = 1
		store, _, err := OpenStore(cfg)
		if err != nil {
			t.Fatal(err)
		}
		for i, id := range []string{"artifact:q1", "artifact:q2"} {
			manifest := storedManifest(t, id, 1, []byte(id), RetentionProject)
			_, err := store.Commit(context.Background(), manifest, readers(manifest, []byte(id)))
			if i == 0 && err != nil {
				t.Fatal(err)
			}
			if i == 1 && !errors.Is(err, ErrQuotaExceeded) {
				t.Fatalf("err=%v", err)
			}
		}
	})
}

func TestStoreCrashRecovery(t *testing.T) {
	t.Run("before_marker_invisible", func(t *testing.T) {
		store, cfg, _ := newTestStore(t)
		manifest := storedManifest(t, "artifact:before", 1, []byte("before"), RetentionProject)
		store.fault = func(stage string) error {
			if stage == "before_commit_marker" {
				return errors.New("crash")
			}
			return nil
		}
		if _, err := store.Commit(context.Background(), manifest, readers(manifest, []byte("before"))); !errors.Is(err, ErrStoreUnavailable) {
			t.Fatalf("err=%v", err)
		}
		reopened, health, err := OpenStore(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := reopened.GetManifest(manifest.ArtifactID, 1); !errors.Is(err, ErrArtifactNotFound) {
			t.Fatalf("err=%v", err)
		}
		if health.FreshStaging == 0 {
			t.Fatalf("health=%#v", health)
		}
	})
	t.Run("after_marker_replayed", func(t *testing.T) {
		store, cfg, _ := newTestStore(t)
		manifest := storedManifest(t, "artifact:after", 1, []byte("after"), RetentionProject)
		store.fault = func(stage string) error {
			if stage == "after_commit_marker" {
				return errors.New("response lost")
			}
			return nil
		}
		if _, err := store.Commit(context.Background(), manifest, readers(manifest, []byte("after"))); !errors.Is(err, ErrOutcomeUnknown) {
			t.Fatalf("err=%v", err)
		}
		reopened, _, err := OpenStore(cfg)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err := reopened.GetManifest(manifest.ArtifactID, 1); err != nil {
			t.Fatal(err)
		}
		result, err := reopened.Commit(context.Background(), manifest, readers(manifest, []byte("after")))
		if err != nil || !result.Deduplicated {
			t.Fatalf("result=%#v err=%v", result, err)
		}
	})
}

func TestStoreOpenAssetDetectsLiveCorruption(t *testing.T) {
	store, _, _ := newTestStore(t)
	manifest := storedManifest(t, "artifact:live-corrupt", 1, []byte("healthy"), RetentionProject)
	if _, err := store.Commit(context.Background(), manifest, readers(manifest, []byte("healthy"))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.blobPath(manifest.Assets[0].SHA256), []byte("corrupt"), 0o640); err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.OpenAsset(manifest.Assets[0].SHA256); !errors.Is(err, ErrStoreCorrupt) {
		t.Fatalf("err=%v", err)
	}
}

func TestStoreReconcileRejectsConflictingRevisionMarkers(t *testing.T) {
	store, _, clock := newTestStore(t)
	first := storedManifest(t, "artifact:duplicate-revision", 1, []byte("first"), RetentionProject)
	if _, err := store.Commit(context.Background(), first, readers(first, []byte("first"))); err != nil {
		t.Fatal(err)
	}
	second := storedManifest(t, first.ArtifactID, 1, []byte("second"), RetentionProject)
	installCommitForTest(t, store, second, []byte("second"), clock.Now())
	health, err := store.Reconcile()
	if err != nil {
		t.Fatal(err)
	}
	if health.CorruptRecords < 2 || len(store.List()) != 0 {
		t.Fatalf("health=%#v entries=%#v", health, store.List())
	}
}

func TestStoreReconcileQuarantinesCorruptionAndStaleStaging(t *testing.T) {
	store, cfg, clock := newTestStore(t)
	manifest := storedManifest(t, "artifact:corrupt", 1, []byte("healthy"), RetentionProject)
	if _, err := store.Commit(context.Background(), manifest, readers(manifest, []byte("healthy"))); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.blobPath(manifest.Assets[0].SHA256), []byte("corrupt"), 0o640); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(cfg.Root, "staging", "stale")
	if err := os.Mkdir(stale, 0o750); err != nil {
		t.Fatal(err)
	}
	old := clock.Now().Add(-2 * cfg.StagingQuarantineAge)
	if err := os.Chtimes(stale, old, old); err != nil {
		t.Fatal(err)
	}
	reopened, health, err := OpenStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if health.Status != "degraded" || health.CorruptRecords == 0 || health.Quarantined < 2 {
		t.Fatalf("health=%#v", health)
	}
	if _, _, err := reopened.GetManifest(manifest.ArtifactID, 1); !errors.Is(err, ErrArtifactNotFound) {
		t.Fatalf("err=%v", err)
	}
}

func TestStoreTombstoneLegalHoldAndReferenceAwareGC(t *testing.T) {
	store, _, clock := newTestStore(t)
	payload := []byte("shared-retention")
	first := storedManifest(t, "artifact:retire-one", 1, payload, RetentionProject)
	second := storedManifest(t, "artifact:retire-two", 1, payload, RetentionProject)
	for _, manifest := range []Manifest{first, second} {
		if _, err := store.Commit(context.Background(), manifest, readers(manifest, payload)); err != nil {
			t.Fatal(err)
		}
	}
	tombstone, err := store.Tombstone(first.ArtifactID, 1, "superseded", "authority:retention")
	if err != nil {
		t.Fatal(err)
	}
	again, err := store.Tombstone(first.ArtifactID, 1, "different ignored", "authority:other")
	if err != nil || again != tombstone {
		t.Fatalf("again=%#v err=%v", again, err)
	}
	if _, _, err := store.GetManifest(first.ArtifactID, 1); !errors.Is(err, ErrArtifactNotFound) {
		t.Fatalf("err=%v", err)
	}
	clock.Add(2 * store.cfg.GCGrace)
	result, err := store.GC()
	if err != nil {
		t.Fatal(err)
	}
	if result.RetiredCommits != 1 || result.RemovedBlobs != 0 {
		t.Fatalf("result=%#v", result)
	}
	if _, err := os.Stat(store.blobPath(first.Assets[0].SHA256)); err != nil {
		t.Fatal("shared blob removed")
	}
	if _, err := store.Tombstone(second.ArtifactID, 1, "expired", "authority:retention"); err != nil {
		t.Fatal(err)
	}
	clock.Add(2 * store.cfg.GCGrace)
	result, err = store.GC()
	if err != nil || result.RemovedBlobs != 1 {
		t.Fatalf("result=%#v err=%v", result, err)
	}

	legal := storedManifest(t, "artifact:legal", 1, []byte("hold"), RetentionLegalHold)
	if _, err := store.Commit(context.Background(), legal, readers(legal, []byte("hold"))); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Tombstone(legal.ArtifactID, 1, "delete", "authority:retention"); !errors.Is(err, ErrRetentionBlocked) {
		t.Fatalf("err=%v", err)
	}
}

func TestStoreIndexRestartAndBackupAreDeterministic(t *testing.T) {
	store, cfg, _ := newTestStore(t)
	manifest := storedManifest(t, "artifact:backup", 1, []byte("backup"), RetentionRelease)
	if _, err := store.Commit(context.Background(), manifest, readers(manifest, []byte("backup"))); err != nil {
		t.Fatal(err)
	}
	indexPath := filepath.Join(cfg.Root, "index", "index.v1.json")
	before, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	reopened, _, err := OpenStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(indexPath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("index changed across restart")
	}
	first, err := reopened.CreateBackupManifest()
	if err != nil {
		t.Fatal(err)
	}
	second, err := reopened.CreateBackupManifest()
	if err != nil || !reflect.DeepEqual(first, second) {
		t.Fatal("backup manifest is not deterministic")
	}
	if err := reopened.VerifyBackupManifest(first); err != nil {
		t.Fatal(err)
	}
	tampered := reopened.blobPath(manifest.Assets[0].SHA256)
	if err := os.WriteFile(tampered, []byte("tampered"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := reopened.VerifyBackupManifest(first); !errors.Is(err, ErrBackupInvalid) {
		t.Fatalf("file tamper err=%v", err)
	}
	first.Files[0].ByteSize++
	if err := reopened.VerifyBackupManifest(first); !errors.Is(err, ErrBackupInvalid) {
		t.Fatalf("manifest tamper err=%v", err)
	}
}

func installCommitForTest(t *testing.T, store *Store, manifest Manifest, payload []byte, now time.Time) {
	t.Helper()
	asset := manifest.Assets[0]
	if err := os.MkdirAll(filepath.Dir(store.blobPath(asset.SHA256)), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.blobPath(asset.SHA256), payload, 0o640); err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	manifestBytes = append(manifestBytes, '\n')
	if err := writeAtomicFile(store.manifestPath(manifest.Integrity.ManifestSHA256), manifestBytes, 0o640); err != nil {
		t.Fatal(err)
	}
	commitID := deterministicID(manifest.ArtifactID, uintString(manifest.Revision), manifest.Integrity.ManifestSHA256)
	inspection, err := store.inspector.Inspect(context.Background(), InspectionRequest{Path: store.blobPath(asset.SHA256), Asset: asset, Security: manifest.Security, Policy: manifest.Policy})
	if err != nil {
		t.Fatal(err)
	}
	record := CommitRecord{
		Schema: StoreSchemaV1, CommitID: commitID, ArtifactID: manifest.ArtifactID, Revision: manifest.Revision,
		ManifestSHA256: manifest.Integrity.ManifestSHA256, CommittedAt: now.UTC().Format(time.RFC3339Nano),
		Assets:      []AssetRecord{{AssetID: asset.AssetID, SHA256: asset.SHA256, ByteSize: asset.ByteSize, MediaType: asset.MediaType}},
		Inspections: []InspectionRecord{inspection},
	}
	if err := writeAtomicJSON(store.commitPath(commitID), record, 0o640); err != nil {
		t.Fatal(err)
	}
}

func newTestStore(t *testing.T) (*Store, StoreConfig, *testClock) {
	t.Helper()
	clock := &testClock{now: time.Date(2026, 8, 29, 12, 0, 0, 0, time.UTC)}
	cfg := testStoreConfig(t.TempDir(), clock)
	store, _, err := OpenStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return store, cfg, clock
}

func testStoreConfig(root string, clock *testClock) StoreConfig {
	return StoreConfig{
		Root: root, MaxStoreBytes: 64 * 1024 * 1024, MaxArtifacts: 100,
		MaxAssetBytes: 8 * 1024 * 1024, StagingQuarantineAge: time.Hour, GCGrace: time.Hour,
		Now: clock.Now,
	}
}

func storedManifest(t *testing.T, artifactID string, revision uint64, payload []byte, retention RetentionClass) Manifest {
	t.Helper()
	manifest := testManifest()
	manifest.ArtifactID = artifactID
	manifest.Revision = revision
	digest := sha256.Sum256(payload)
	manifest.Assets[0].SHA256 = hex.EncodeToString(digest[:])
	manifest.Assets[0].ByteSize = int64(len(payload))
	manifest.Assets[0].MediaType = "text/plain"
	manifest.Policy.RetentionClass = retention
	manifest.Integrity.ManifestSHA256 = ""
	sealed, err := Seal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return sealed
}

func readers(manifest Manifest, payload []byte) map[string]io.Reader {
	assets := append([]Asset(nil), manifest.Assets...)
	sort.Slice(assets, func(i, j int) bool { return assets[i].AssetID > assets[j].AssetID })
	out := make(map[string]io.Reader, len(assets))
	for _, asset := range assets {
		out[asset.AssetID] = bytes.NewReader(payload)
	}
	return out
}
