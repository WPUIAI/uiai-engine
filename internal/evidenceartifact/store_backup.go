package evidenceartifact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type BackupManifest struct {
	Schema          string             `json:"schema"`
	StoreGeneration uint64             `json:"store_generation"`
	Files           []BackupFileRecord `json:"files"`
	ManifestSHA256  string             `json:"manifest_sha256"`
}

type BackupFileRecord struct {
	Path     string `json:"path"`
	SHA256   string `json:"sha256"`
	ByteSize int64  `json:"byte_size"`
}

func (s *Store) CreateBackupManifest() (BackupManifest, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	manifest := BackupManifest{Schema: StoreSchemaV1, StoreGeneration: s.index.Generation, Files: []BackupFileRecord{}}
	for _, root := range s.backupRoots() {
		files, err := listFiles(root)
		if err != nil {
			return BackupManifest{}, storeError(ErrStoreUnavailable, "scan backup inventory")
		}
		for _, path := range files {
			relative, err := filepath.Rel(s.cfg.Root, path)
			if err != nil {
				return BackupManifest{}, ErrBackupInvalid
			}
			relative = filepath.ToSlash(relative)
			if !validBackupPath(relative) {
				return BackupManifest{}, ErrBackupInvalid
			}
			digest, size, err := hashFile(path)
			if err != nil {
				return BackupManifest{}, storeError(ErrStoreUnavailable, "hash backup inventory")
			}
			manifest.Files = append(manifest.Files, BackupFileRecord{Path: relative, SHA256: digest, ByteSize: size})
		}
	}
	sort.Slice(manifest.Files, func(i, j int) bool { return manifest.Files[i].Path < manifest.Files[j].Path })
	digest, err := backupHash(manifest)
	if err != nil {
		return BackupManifest{}, ErrBackupInvalid
	}
	manifest.ManifestSHA256 = digest
	if err := writeAtomicJSON(filepath.Join(s.cfg.Root, "backup", digest+".json"), manifest, 0o640); err != nil {
		return BackupManifest{}, storeError(ErrStoreUnavailable, "write backup manifest")
	}
	return manifest, nil
}

func (s *Store) VerifyBackupManifest(manifest BackupManifest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if manifest.Schema != StoreSchemaV1 || !validSHA256(manifest.ManifestSHA256) {
		return ErrBackupInvalid
	}
	digest, err := backupHash(manifest)
	if err != nil || digest != manifest.ManifestSHA256 {
		return ErrBackupInvalid
	}
	seen := make(map[string]struct{}, len(manifest.Files))
	previous := ""
	for _, record := range manifest.Files {
		if !validBackupPath(record.Path) || !validSHA256(record.SHA256) || record.ByteSize <= 0 || (previous != "" && record.Path <= previous) {
			return ErrBackupInvalid
		}
		if _, duplicate := seen[record.Path]; duplicate {
			return ErrBackupInvalid
		}
		seen[record.Path] = struct{}{}
		previous = record.Path
		path := filepath.Join(s.cfg.Root, filepath.FromSlash(record.Path))
		digest, size, err := hashFile(path)
		if err != nil || digest != record.SHA256 || size != record.ByteSize {
			return ErrBackupInvalid
		}
	}
	return nil
}

func backupHash(manifest BackupManifest) (string, error) {
	copy := manifest
	copy.ManifestSHA256 = ""
	data, err := json.Marshal(copy)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func (s *Store) backupRoots() []string {
	return []string{
		filepath.Join(s.cfg.Root, "STORE-VERSION"),
		filepath.Join(s.cfg.Root, "blobs", "sha256"),
		filepath.Join(s.cfg.Root, "manifests", "sha256"),
		filepath.Join(s.cfg.Root, "commits", "sha256"),
		filepath.Join(s.cfg.Root, "tombstones", "sha256"),
		filepath.Join(s.cfg.Root, "retired", "commits", "sha256"),
	}
}

func validBackupPath(path string) bool {
	if !validRelativePath(path) {
		return false
	}
	if path == "STORE-VERSION" {
		return true
	}
	for _, prefix := range []string{"blobs/sha256/", "manifests/sha256/", "commits/sha256/", "tombstones/sha256/", "retired/commits/sha256/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func loadBackupManifest(path string) (BackupManifest, error) {
	var manifest BackupManifest
	if _, err := os.Stat(path); err != nil {
		return manifest, err
	}
	if err := readJSONFile(path, 16*1024*1024, &manifest); err != nil {
		return BackupManifest{}, err
	}
	return manifest, nil
}
