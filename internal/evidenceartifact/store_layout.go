package evidenceartifact

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var storeDirs = []string{
	"blobs/sha256", "manifests/sha256", "commits/sha256", "tombstones/sha256",
	"retired/commits/sha256", "staging", "quarantine", "index", "backup",
}

func (s *Store) initializeLayout() error {
	if err := os.MkdirAll(s.cfg.Root, 0o750); err != nil {
		return storeError(ErrStoreUnavailable, "create root")
	}
	for _, dir := range storeDirs {
		if err := os.MkdirAll(filepath.Join(s.cfg.Root, filepath.FromSlash(dir)), 0o750); err != nil {
			return storeError(ErrStoreUnavailable, "create layout")
		}
	}
	versionPath := filepath.Join(s.cfg.Root, "STORE-VERSION")
	data, err := os.ReadFile(versionPath) // #nosec G304 -- fixed path under configured root.
	if err == nil {
		if string(data) != StoreSchemaV1+"\n" {
			return storeError(ErrStoreCorrupt, "store version")
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return storeError(ErrStoreUnavailable, "read store version")
	}
	if err := writeAtomicFile(versionPath, []byte(StoreSchemaV1+"\n"), 0o640); err != nil {
		return storeError(ErrStoreUnavailable, "write store version")
	}
	return syncDir(s.cfg.Root)
}

func (s *Store) blobPath(digest string) string {
	return hashPath(s.cfg.Root, "blobs", digest, "")
}

func (s *Store) manifestPath(digest string) string {
	return hashPath(s.cfg.Root, "manifests", digest, ".json")
}

func (s *Store) commitPath(commitID string) string {
	return hashPath(s.cfg.Root, "commits", commitID, ".json")
}

func (s *Store) tombstonePath(tombstoneID string) string {
	return hashPath(s.cfg.Root, "tombstones", tombstoneID, ".json")
}

func (s *Store) retiredCommitPath(commitID string) string {
	return hashPath(s.cfg.Root, filepath.Join("retired", "commits"), commitID, ".json")
}

func hashPath(root, kind, digest, suffix string) string {
	return filepath.Join(root, kind, "sha256", digest[:2], digest+suffix)
}

func deterministicID(parts ...string) string {
	h := sha256.New()
	for _, part := range parts {
		_, _ = io.WriteString(h, part)
		_, _ = h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func randomID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func writeAtomicJSON(path string, value any, mode os.FileMode) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return writeAtomicFile(path, append(data, '\n'), mode)
}

func writeAtomicFile(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".atomic-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		_ = tmp.Close()
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(mode); err != nil {
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	if err := syncDir(filepath.Dir(path)); err != nil {
		return err
	}
	ok = true
	return nil
}

func promoteFile(staged, target, expectedHash string, expectedSize int64) (bool, error) {
	if info, err := os.Stat(target); err == nil {
		if info.Size() != expectedSize {
			return false, ErrAssetMismatch
		}
		digest, size, err := hashFile(target)
		if err != nil || digest != expectedHash || size != expectedSize {
			return false, ErrAssetMismatch
		}
		if err := os.Remove(staged); err != nil && !os.IsNotExist(err) {
			return false, err
		}
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return false, err
	}
	if err := os.Rename(staged, target); err != nil {
		if _, statErr := os.Stat(target); statErr == nil {
			return promoteFile(staged, target, expectedHash, expectedSize)
		}
		return false, err
	}
	if err := os.Chmod(target, 0o640); err != nil {
		return false, err
	}
	if err := syncDir(filepath.Dir(target)); err != nil {
		return false, err
	}
	return true, nil
}

func hashFile(path string) (string, int64, error) {
	file, err := os.Open(path) // #nosec G304 -- callers provide rooted validated store paths.
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	h := sha256.New()
	size, err := io.Copy(h, file)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(h.Sum(nil)), size, nil
}

func syncDir(path string) error {
	dir, err := os.Open(path) // #nosec G304 -- caller passes configured store directory.
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

func readJSONFile(path string, maxBytes int64, out any) error {
	file, err := os.Open(path) // #nosec G304 -- callers pass rooted store paths.
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.Size() > maxBytes {
		return fmt.Errorf("JSON exceeds limit")
	}
	decoder := json.NewDecoder(io.LimitReader(file, maxBytes+1))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(out); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("trailing JSON")
	}
	return nil
}

func (s *Store) quarantinePath(source, kind, digest string) error {
	stamp := s.cfg.Now().UTC().Format("20060102T150405.000000000Z")
	nonce, err := randomID()
	if err != nil {
		return err
	}
	name := stamp + "-" + sanitizeKind(kind) + "-" + digest[:12] + "-" + nonce[:8]
	destDir := filepath.Join(s.cfg.Root, "quarantine", name)
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return err
	}
	dest := filepath.Join(destDir, filepath.Base(source))
	if err := os.Rename(source, dest); err != nil {
		return err
	}
	meta := map[string]string{"kind": kind, "digest": digest, "quarantined_at": s.cfg.Now().UTC().Format(time.RFC3339Nano)}
	if err := writeAtomicJSON(filepath.Join(destDir, "quarantine.json"), meta, 0o640); err != nil {
		return err
	}
	return syncDir(filepath.Dir(source))
}

func sanitizeKind(value string) string {
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "record"
	}
	return b.String()
}

func listFiles(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func directoryBytes(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || strings.Contains(filepath.ToSlash(path), "/staging/") {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		total += info.Size()
		return nil
	})
	return total, err
}
