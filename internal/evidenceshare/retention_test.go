package evidenceshare

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
)

type retentionClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *retentionClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *retentionClock) advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func retentionTestStore(t *testing.T) (*evidenceartifact.Store, *retentionClock, string) {
	t.Helper()
	clock := &retentionClock{now: time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)}
	root := t.TempDir()
	store, _, err := evidenceartifact.OpenStore(evidenceartifact.StoreConfig{
		Root: root, MaxStoreBytes: 64 << 20, MaxArtifacts: 1000, MaxAssetBytes: 8 << 20,
		StagingQuarantineAge: time.Hour, GCGrace: time.Hour, Now: clock.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	return store, clock, root
}

func retentionManifest(t *testing.T, artifactID string, retention evidenceartifact.RetentionClass, expiresAt string, payload []byte) evidenceartifact.Manifest {
	t.Helper()
	body, err := os.ReadFile(filepath.Join("..", "evidenceartifact", "testdata", "manifest.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest evidenceartifact.Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatal(err)
	}
	manifest.ArtifactID = artifactID
	manifest.Revision = 1
	manifest.Assets[0].Path = "assets/proof.json"
	manifest.Assets[0].MediaType = "application/json"
	manifest.Policy.RetentionClass = retention
	manifest.Policy.ExpiresAt = expiresAt
	digest := sha256.Sum256(payload)
	manifest.Assets[0].SHA256 = hex.EncodeToString(digest[:])
	manifest.Assets[0].ByteSize = int64(len(payload))
	manifest.Integrity.BundleSHA256 = ""
	sum := sha256.Sum256([]byte("bundle-" + artifactID))
	manifest.Integrity.BundleSHA256 = hex.EncodeToString(sum[:])
	manifest.Integrity.ManifestSHA256 = ""
	manifest.Integrity.ManifestSHA256, err = evidenceartifact.ComputeManifestSHA256(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return manifest
}

func commitArtifact(t *testing.T, store *evidenceartifact.Store, artifactID string, retention evidenceartifact.RetentionClass, expiresAt string, payload []byte) {
	t.Helper()
	manifest := retentionManifest(t, artifactID, retention, expiresAt, payload)
	assets := map[string]io.Reader{manifest.Assets[0].AssetID: strings.NewReader(string(payload))}
	if _, err := store.Commit(context.Background(), manifest, assets); err != nil {
		t.Fatal(err)
	}
}

func jsonPayload(tag string) []byte {
	return []byte(`{"probe":"retention","tag":"` + tag + `","pad":"` + strings.Repeat("x", 32) + `"}`)
}

func defaultLifecycle() map[string]any {
	return map[string]any{
		"retention_days":        0,
		"max_packets":           1000,
		"max_bytes":             0,
		"pinning":               true,
		"archive_before_expire": true,
		"legal_hold":            false,
	}
}

func TestMapLifecycleSettingsDefaultsAndFailures(t *testing.T) {
	policy, err := MapLifecycleSettings(map[string]any{"lifecycle": defaultLifecycle()})
	if err != nil {
		t.Fatal(err)
	}
	if policy.RetentionDays != 0 || policy.MaxPackets != 1000 || policy.MaxBytes != 0 ||
		!policy.Pinning || !policy.ArchiveBeforeExpire || policy.LegalHold {
		t.Fatalf("default mapping wrong: %#v", policy)
	}
	if _, err := MapLifecycleSettings(map[string]any{}); err == nil {
		t.Fatal("missing lifecycle domain must fail")
	}
	bad := defaultLifecycle()
	bad["legal_hold"] = "yes"
	if _, err := MapLifecycleSettings(map[string]any{"lifecycle": bad}); err == nil {
		t.Fatal("mistyped legal_hold must fail")
	}
	bad2 := defaultLifecycle()
	bad2["retention_days"] = -5
	if _, err := MapLifecycleSettings(map[string]any{"lifecycle": bad2}); err == nil {
		t.Fatal("negative retention_days must fail")
	}
}

func TestSweepArchivesExpiredAndAgedButNeverDeletes(t *testing.T) {
	store, clock, root := retentionTestStore(t)
	payload := jsonPayload("sweep")
	commitArtifact(t, store, "uiai-artifact:sha256:"+strings.Repeat("a", 64), evidenceartifact.RetentionProject, "", payload)
	commitArtifact(t, store, "uiai-artifact:sha256:"+strings.Repeat("b", 64), evidenceartifact.RetentionWorkstream, "", payload)

	policy, err := MapLifecycleSettings(map[string]any{"lifecycle": map[string]any{
		"retention_days": 30, "max_packets": 0, "max_bytes": 0,
		"pinning": true, "archive_before_expire": true, "legal_hold": false,
	}})
	if err != nil {
		t.Fatal(err)
	}
	// Young store: nothing archived.
	receipt, err := Sweep(store, policy, SettingsScope{}, "authority:test-sweep", clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.AgedTombstoned != 0 || receipt.Remaining != 2 {
		t.Fatalf("young store swept: %#v", receipt)
	}
	// Age past retention: both archived, commits retired to the retired layout (no deletion).
	clock.advance(45 * 24 * time.Hour)
	receipt, err = Sweep(store, policy, SettingsScope{}, "authority:test-sweep", clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.AgedTombstoned != 2 || receipt.Remaining != 0 {
		t.Fatalf("expected both aged entries archived: %#v", receipt)
	}
	clock.advance(2 * time.Hour) // past GC grace
	if _, err := store.GC(); err != nil {
		t.Fatal(err)
	}
	retired := 0
	if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		if strings.Contains(path, string(os.PathSeparator)+"retired"+string(os.PathSeparator)) {
			retired++
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if retired == 0 {
		t.Fatal("GC retirement did not archive commits to the retired layout")
	}
}

func TestSweepHonorsLegalHoldAndPins(t *testing.T) {
	store, clock, _ := retentionTestStore(t)
	payload := jsonPayload("holds")
	holdID := "uiai-artifact:sha256:" + strings.Repeat("c", 64)
	releaseID := "uiai-artifact:sha256:" + strings.Repeat("d", 64)
	plainID := "uiai-artifact:sha256:" + strings.Repeat("e", 64)
	commitArtifact(t, store, holdID, evidenceartifact.RetentionLegalHold, "", payload)
	commitArtifact(t, store, releaseID, evidenceartifact.RetentionRelease, "", payload)
	commitArtifact(t, store, plainID, evidenceartifact.RetentionProject, "", payload)

	values := defaultLifecycle()
	values["retention_days"] = 1
	policy, err := MapLifecycleSettings(map[string]any{"lifecycle": values})
	if err != nil {
		t.Fatal(err)
	}
	clock.advance(10 * 24 * time.Hour)
	receipt, err := Sweep(store, policy, SettingsScope{}, "authority:test-sweep", clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.HeldLegal != 1 || receipt.HeldPinned != 1 || receipt.AgedTombstoned != 1 || receipt.Remaining != 2 {
		t.Fatalf("legal hold / pin semantics wrong: %#v", receipt)
	}

	// Pinning disabled: release-class artifacts archive like anything else.
	values["pinning"] = false
	policy, err = MapLifecycleSettings(map[string]any{"lifecycle": values})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err = Sweep(store, policy, SettingsScope{}, "authority:test-sweep", clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.HeldPinned != 0 || receipt.AgedTombstoned != 1 || receipt.Remaining != 1 {
		t.Fatalf("pinning-off sweep wrong: %#v", receipt)
	}

	// Whole-scope legal hold setting protects the remaining entry.
	values["legal_hold"] = true
	policy, err = MapLifecycleSettings(map[string]any{"lifecycle": values})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err = Sweep(store, policy, SettingsScope{}, "authority:test-sweep", clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.HeldLegal != 1 || receipt.Remaining != 1 {
		t.Fatalf("scope legal hold wrong: %#v", receipt)
	}
}

func TestSweepQuotaArchivesOldestFirst(t *testing.T) {
	store, clock, _ := retentionTestStore(t)
	payload := jsonPayload("quota")
	for i, b := range []byte{'f', 'g', 'h'} {
		id := "uiai-artifact:sha256:" + strings.Repeat(string(b), 64)
		_ = i
		commitArtifact(t, store, id, evidenceartifact.RetentionProject, "", payload)
	}
	values := defaultLifecycle()
	values["max_packets"] = 2
	policy, err := MapLifecycleSettings(map[string]any{"lifecycle": values})
	if err != nil {
		t.Fatal(err)
	}
	clock.advance(30 * time.Minute)
	receipt, err := Sweep(store, policy, SettingsScope{}, "authority:test-sweep", clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.QuotaTombstoned != 1 || receipt.Remaining != 2 {
		t.Fatalf("quota archive wrong: %#v", receipt)
	}
}

func TestSweepWithoutArchiveBlocksTombstones(t *testing.T) {
	store, clock, _ := retentionTestStore(t)
	commitArtifact(t, store, "uiai-artifact:sha256:"+strings.Repeat("i", 64), evidenceartifact.RetentionProject, "", jsonPayload("noarchive"))
	values := defaultLifecycle()
	values["archive_before_expire"] = false
	values["retention_days"] = 1
	policy, err := MapLifecycleSettings(map[string]any{"lifecycle": values})
	if err != nil {
		t.Fatal(err)
	}
	clock.advance(5 * 24 * time.Hour)
	receipt, err := Sweep(store, policy, SettingsScope{}, "authority:test-sweep", clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.SkippedUnarchived != 1 || receipt.AgedTombstoned != 0 || receipt.QuotaTombstoned != 0 || receipt.Remaining != 1 {
		t.Fatalf("archive-before-expire=false must be a no-op: %#v", receipt)
	}
}

func TestSweepExpiryFromManifestPolicy(t *testing.T) {
	store, clock, _ := retentionTestStore(t)
	past := clock.Now().Add(-time.Hour).UTC().Format(time.RFC3339)
	future := clock.Now().Add(24 * time.Hour).UTC().Format(time.RFC3339)
	payload := jsonPayload("expiry")
	expiredID := "uiai-artifact:sha256:" + strings.Repeat("j", 64)
	liveID := "uiai-artifact:sha256:" + strings.Repeat("k", 64)
	commitArtifact(t, store, expiredID, evidenceartifact.RetentionProject, past, payload)
	commitArtifact(t, store, liveID, evidenceartifact.RetentionProject, future, payload)
	values := defaultLifecycle()
	policy, err := MapLifecycleSettings(map[string]any{"lifecycle": values})
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := Sweep(store, policy, SettingsScope{}, "authority:test-sweep", clock.Now())
	if err != nil {
		t.Fatal(err)
	}
	if receipt.ExpiredTombstoned != 1 || receipt.Remaining != 1 {
		t.Fatalf("manifest expiry sweep wrong: %#v", receipt)
	}
}

func TestSweepRestartSafety(t *testing.T) {
	store, clock, root := retentionTestStore(t)
	commitArtifact(t, store, "uiai-artifact:sha256:"+strings.Repeat("l", 64), evidenceartifact.RetentionProject, "", jsonPayload("restart"))
	values := defaultLifecycle()
	values["retention_days"] = 1
	policy, err := MapLifecycleSettings(map[string]any{"lifecycle": values})
	if err != nil {
		t.Fatal(err)
	}
	clock.advance(2 * 24 * time.Hour)
	if _, err := Sweep(store, policy, SettingsScope{}, "authority:test-sweep", clock.Now()); err != nil {
		t.Fatal(err)
	}
	// Simulate restart: reopen the store from the same root.
	reopened, _, err := evidenceartifact.OpenStore(evidenceartifact.StoreConfig{
		Root: root, MaxStoreBytes: 64 << 20, MaxArtifacts: 1000, MaxAssetBytes: 8 << 20,
		StagingQuarantineAge: time.Hour, GCGrace: time.Hour, Now: clock.Now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := len(reopened.List()); got != 0 {
		t.Fatalf("tombstone did not survive restart: %d live", got)
	}
	health := reopened.Health()
	if health.TombstonedArtifacts != 1 {
		t.Fatalf("reopened store lost tombstone: %#v", health)
	}
}
