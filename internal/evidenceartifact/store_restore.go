package evidenceartifact

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// RestoreBackup atomically materializes a verified backup into a store root that
// does not yet exist. It never merges backup data into an existing store.
func RestoreBackup(backupRoot string, cfg StoreConfig, manifest BackupManifest) (*Store, StoreHealth, error) {
	if backupRoot == "" || cfg.Root == "" {
		return nil, StoreHealth{}, ErrStoreConfig
	}
	sourceRoot, err := filepath.EvalSymlinks(backupRoot)
	if err != nil {
		return nil, StoreHealth{}, storeError(ErrBackupInvalid, "resolve backup root")
	}
	sourceRoot, err = filepath.Abs(sourceRoot)
	if err != nil {
		return nil, StoreHealth{}, storeError(ErrBackupInvalid, "resolve backup root")
	}
	targetRoot, err := filepath.Abs(cfg.Root)
	if err != nil {
		return nil, StoreHealth{}, storeError(ErrStoreConfig, "resolve restore root")
	}
	parent := filepath.Dir(targetRoot)
	if err := os.MkdirAll(parent, 0o750); err != nil {
		return nil, StoreHealth{}, storeError(ErrStoreUnavailable, "create restore parent")
	}
	resolvedParent, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return nil, StoreHealth{}, storeError(ErrStoreConfig, "resolve restore parent")
	}
	targetRoot = filepath.Join(resolvedParent, filepath.Base(targetRoot))
	if pathsOverlap(sourceRoot, targetRoot) {
		return nil, StoreHealth{}, storeError(ErrStoreConfig, "backup and restore roots overlap")
	}
	if _, err := os.Lstat(targetRoot); err == nil {
		return nil, StoreHealth{}, storeError(ErrStoreConfig, "restore root already exists")
	} else if !os.IsNotExist(err) {
		return nil, StoreHealth{}, storeError(ErrStoreUnavailable, "inspect restore root")
	}
	if err := verifyBackupManifestAt(sourceRoot, manifest); err != nil {
		return nil, StoreHealth{}, err
	}

	parent = filepath.Dir(targetRoot)
	stage, err := os.MkdirTemp(parent, "."+filepath.Base(targetRoot)+".restore-*")
	if err != nil {
		return nil, StoreHealth{}, storeError(ErrStoreUnavailable, "create restore staging")
	}
	promoted := false
	defer func() {
		if !promoted {
			_ = os.RemoveAll(stage)
		}
	}()

	for _, record := range manifest.Files {
		if err := restoreBackupFile(sourceRoot, stage, record); err != nil {
			return nil, StoreHealth{}, err
		}
	}
	if err := verifyBackupManifestAt(stage, manifest); err != nil {
		return nil, StoreHealth{}, err
	}
	probeCfg := cfg
	probeCfg.Root = stage
	store, health, err := OpenStore(probeCfg)
	if err != nil {
		return nil, health, err
	}
	if health.IndexGeneration != manifest.StoreGeneration {
		return nil, health, storeError(ErrStoreCorrupt, "restored generation mismatch")
	}
	if err := syncDir(stage); err != nil {
		return nil, StoreHealth{}, storeError(ErrStoreUnavailable, "sync restore staging")
	}
	if err := os.Rename(stage, targetRoot); err != nil {
		return nil, StoreHealth{}, storeError(ErrStoreUnavailable, "promote restore")
	}
	promoted = true
	store.cfg.Root = targetRoot
	if err := syncDir(parent); err != nil {
		return nil, StoreHealth{}, storeError(ErrOutcomeUnknown, "sync restored root")
	}
	return store, health, nil
}

func restoreBackupFile(sourceRoot, stage string, record BackupFileRecord) error {
	source := filepath.Join(sourceRoot, filepath.FromSlash(record.Path))
	originalInfo, err := os.Lstat(source)
	if err != nil || originalInfo.Mode()&os.ModeSymlink != 0 || !originalInfo.Mode().IsRegular() {
		return storeError(ErrBackupInvalid, "backup file is not regular")
	}
	resolved, err := filepath.EvalSymlinks(source)
	if err != nil || !pathWithin(sourceRoot, resolved) {
		return storeError(ErrBackupInvalid, "resolve backup file")
	}
	info, err := os.Lstat(resolved)
	if err != nil || !info.Mode().IsRegular() {
		return storeError(ErrBackupInvalid, "backup file is not regular")
	}
	target := filepath.Join(stage, filepath.FromSlash(record.Path))
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return storeError(ErrStoreUnavailable, "create restore directory")
	}
	in, err := os.Open(resolved) // #nosec G304 -- resolved path is bounded by verified backup root.
	if err != nil {
		return storeError(ErrBackupInvalid, "open backup file")
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o640) // #nosec G304 -- target is bounded by private restore staging.
	if err != nil {
		return storeError(ErrStoreUnavailable, "create restored file")
	}
	copied, copyErr := io.Copy(out, in)
	syncErr := out.Sync()
	closeErr := out.Close()
	if copyErr != nil || syncErr != nil || closeErr != nil || copied != record.ByteSize {
		return storeError(ErrStoreUnavailable, "copy restored file")
	}
	digest, size, err := hashFile(target)
	if err != nil || digest != record.SHA256 || size != record.ByteSize {
		return storeError(ErrBackupInvalid, "restored file mismatch")
	}
	if err := syncDir(filepath.Dir(target)); err != nil {
		return storeError(ErrStoreUnavailable, "sync restore directory")
	}
	return nil
}

func pathsOverlap(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	return pathWithin(a, b) || pathWithin(b, a)
}

func pathWithin(root, candidate string) bool {
	root = filepath.Clean(root)
	candidate = filepath.Clean(candidate)
	if runtime.GOOS == "windows" {
		root = strings.ToLower(root)
		candidate = strings.ToLower(candidate)
	}
	return candidate == root || strings.HasPrefix(candidate, root+string(os.PathSeparator))
}
