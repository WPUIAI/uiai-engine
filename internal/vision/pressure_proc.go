//go:build !windows

package vision

func treeRSSKB(pid int) int64 { return procTreeRSSKB(pid) }
