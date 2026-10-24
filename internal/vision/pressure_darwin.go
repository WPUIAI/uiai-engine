//go:build darwin

package vision

import (
	"context"
	"log"
	"os/exec"
	"time"
)

// macOS has no Linux procfs. BSD ps returns a bounded best-effort process-tree
// snapshot; RSS is KiB, matching the other platform adapters.
func treeRSSKB(pid int) int64 {
	if pid <= 0 {
		return 0
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, "/bin/ps", "-axo", "pid=,ppid=,rss=").Output()
	if err != nil {
		log.Printf("[vision] macOS RSS sample unavailable: %v", err)
		return 0
	}
	return psTreeRSSKB(pid, output)
}
