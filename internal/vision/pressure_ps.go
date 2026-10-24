package vision

import (
	"math"
	"strconv"
	"strings"
)

// psTreeRSSKB consumes BSD ps's headerless PID, PPID, RSS (KiB) snapshot.
// A missing/exited process contributes zero; readable descendants still count.
func psTreeRSSKB(root int, output []byte) int64 {
	if root <= 0 {
		return 0
	}
	children := map[int][]int{}
	memory := map[int]int64{}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 3 {
			continue
		}
		pid, pidErr := strconv.Atoi(fields[0])
		parent, parentErr := strconv.Atoi(fields[1])
		if pidErr != nil || parentErr != nil || pid <= 0 || parent < 0 {
			continue
		}
		children[parent] = append(children[parent], pid)
		rss, err := strconv.ParseInt(fields[2], 10, 64)
		if err == nil && rss >= 0 {
			memory[pid] = rss
		}
	}
	seen := map[int]bool{}
	pending := []int{root}
	var total int64
	for len(pending) > 0 {
		pid := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if seen[pid] {
			continue
		}
		seen[pid] = true
		rss := memory[pid]
		if total > math.MaxInt64-rss {
			return math.MaxInt64
		}
		total += rss
		pending = append(pending, children[pid]...)
	}
	return total
}
