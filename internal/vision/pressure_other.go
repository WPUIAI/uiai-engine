//go:build !linux && !darwin && !windows

package vision

// Unsupported platforms expose an unavailable measurement, never a healthy zero.
func treeRSSKB(int) int64 { return 0 }
