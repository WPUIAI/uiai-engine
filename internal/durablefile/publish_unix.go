//go:build !windows

// Package durablefile owns platform-specific durable filesystem publication.
package durablefile

import (
	"errors"
	"fmt"
	"os"
)

// Rename publishes a file or directory; callers sync file contents first and
// SyncDirectory on each changed parent afterwards.
func Rename(oldPath, newPath string) error { return os.Rename(oldPath, newPath) }

// SyncDirectory persists directory metadata on Unix-like systems.
func SyncDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	info, statErr := dir.Stat()
	if statErr != nil {
		return errors.Join(statErr, dir.Close())
	}
	if !info.IsDir() {
		return errors.Join(fmt.Errorf("sync directory %q: not a directory", path), dir.Close())
	}
	return errors.Join(dir.Sync(), dir.Close())
}
