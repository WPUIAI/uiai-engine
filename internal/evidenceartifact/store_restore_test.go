package evidenceartifact

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRestoreBackupRoundTrip(t *testing.T) {
	source, sourceCfg, clock := newTestStore(t)
	payload := []byte("restorable evidence")
	manifest := storedManifest(t, "artifact:restore", 1, payload, RetentionProject)
	if _, err := source.Commit(context.Background(), manifest, readers(manifest, payload)); err != nil {
		t.Fatal(err)
	}
	backup, err := source.CreateBackupManifest()
	if err != nil {
		t.Fatal(err)
	}
	targetRoot := filepath.Join(t.TempDir(), "restored")
	targetCfg := testStoreConfig(targetRoot, clock)
	restored, health, err := RestoreBackup(sourceCfg.Root, targetCfg, backup)
	if err != nil {
		t.Fatal(err)
	}
	if health.IndexGeneration != backup.StoreGeneration || health.LiveArtifacts != 1 {
		t.Fatalf("restored health = %#v, backup generation = %d", health, backup.StoreGeneration)
	}
	got, _, err := restored.GetManifest(manifest.ArtifactID, manifest.Revision)
	if err != nil || got.Integrity.ManifestSHA256 != manifest.Integrity.ManifestSHA256 {
		t.Fatalf("restored manifest = %#v, %v", got, err)
	}
	file, _, err := restored.OpenAsset(manifest.Assets[0].SHA256)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gotPayload, err := io.ReadAll(file)
	if err != nil || string(gotPayload) != string(payload) {
		t.Fatalf("restored payload = %q, %v", gotPayload, err)
	}
}

func TestRestoreBackupRejectsTamperWithoutPublishingTarget(t *testing.T) {
	source, sourceCfg, clock := newTestStore(t)
	payload := []byte("tamper evidence")
	manifest := storedManifest(t, "artifact:tamper", 1, payload, RetentionProject)
	if _, err := source.Commit(context.Background(), manifest, readers(manifest, payload)); err != nil {
		t.Fatal(err)
	}
	backup, err := source.CreateBackupManifest()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source.blobPath(manifest.Assets[0].SHA256), []byte("changed evidence"), 0o640); err != nil {
		t.Fatal(err)
	}
	targetRoot := filepath.Join(t.TempDir(), "must-not-exist")
	_, _, err = RestoreBackup(sourceCfg.Root, testStoreConfig(targetRoot, clock), backup)
	if !errors.Is(err, ErrBackupInvalid) {
		t.Fatalf("RestoreBackup() error = %v, want ErrBackupInvalid", err)
	}
	if _, statErr := os.Lstat(targetRoot); !os.IsNotExist(statErr) {
		t.Fatalf("restore target was published after failed verification: %v", statErr)
	}
}

func TestRestoreBackupRejectsSymlinkedBackupFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation is not reliably available on Windows test hosts")
	}
	source, sourceCfg, clock := newTestStore(t)
	payload := []byte("symlink evidence")
	manifest := storedManifest(t, "artifact:symlink", 1, payload, RetentionProject)
	if _, err := source.Commit(context.Background(), manifest, readers(manifest, payload)); err != nil {
		t.Fatal(err)
	}
	backup, err := source.CreateBackupManifest()
	if err != nil {
		t.Fatal(err)
	}
	blob := source.blobPath(manifest.Assets[0].SHA256)
	copyPath := filepath.Join(sourceCfg.Root, "unlisted-backup-copy")
	if err := os.WriteFile(copyPath, payload, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(blob); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(copyPath, blob); err != nil {
		t.Fatal(err)
	}
	targetRoot := filepath.Join(t.TempDir(), "must-not-exist")
	_, _, err = RestoreBackup(sourceCfg.Root, testStoreConfig(targetRoot, clock), backup)
	if !errors.Is(err, ErrBackupInvalid) {
		t.Fatalf("RestoreBackup() error = %v, want ErrBackupInvalid", err)
	}
	if _, statErr := os.Lstat(targetRoot); !os.IsNotExist(statErr) {
		t.Fatalf("restore target was published from symlinked input: %v", statErr)
	}
}

func TestRestoreBackupRejectsInvalidConfigWithoutPublishingTarget(t *testing.T) {
	source, sourceCfg, clock := newTestStore(t)
	backup, err := source.CreateBackupManifest()
	if err != nil {
		t.Fatal(err)
	}
	targetRoot := filepath.Join(t.TempDir(), "must-not-exist")
	badCfg := testStoreConfig(targetRoot, clock)
	badCfg.MaxArtifacts = 0
	_, _, err = RestoreBackup(sourceCfg.Root, badCfg, backup)
	if !errors.Is(err, ErrStoreConfig) {
		t.Fatalf("RestoreBackup() error = %v, want ErrStoreConfig", err)
	}
	if _, statErr := os.Lstat(targetRoot); !os.IsNotExist(statErr) {
		t.Fatalf("restore target was published with invalid config: %v", statErr)
	}
}

func TestRestoreBackupRejectsExistingOrOverlappingTarget(t *testing.T) {
	source, sourceCfg, clock := newTestStore(t)
	backup, err := source.CreateBackupManifest()
	if err != nil {
		t.Fatal(err)
	}
	for name, target := range map[string]string{
		"existing": sourceCfg.Root,
		"nested":   filepath.Join(sourceCfg.Root, "nested-restore"),
	} {
		t.Run(name, func(t *testing.T) {
			_, _, err := RestoreBackup(sourceCfg.Root, testStoreConfig(target, clock), backup)
			if !errors.Is(err, ErrStoreConfig) {
				t.Fatalf("RestoreBackup() error = %v, want ErrStoreConfig", err)
			}
		})
	}
}
