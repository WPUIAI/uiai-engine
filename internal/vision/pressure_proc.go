//go:build linux

package vision

func treeRSSKB(pid int) int64 { return procTreeRSSKB(pid) }
