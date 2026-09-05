//go:build windows

package durablefile

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// Rename uses Windows write-through publication. File contents must already be
// flushed by the caller. No copy fallback is permitted across volumes.
func Rename(oldPath, newPath string) error {
	oldName, err := windows.UTF16PtrFromString(oldPath)
	if err != nil {
		return err
	}
	newName, err := windows.UTF16PtrFromString(newPath)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(oldName, newName, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}

// SyncDirectory validates the parent on Windows. Windows does not support the
// POSIX directory fsync operation; Rename supplies the write-through publication
// boundary instead. This is not a claim of POSIX power-loss durability.
func SyncDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("sync directory %q: not a directory", path)
	}
	return nil
}
