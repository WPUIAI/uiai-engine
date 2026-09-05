package vision

import (
	"os"
	"strconv"
	"testing"
)

func TestTreeRSSKBSelfPositive(t *testing.T) {
	pid := os.Getpid()
	if kb := treeRSSKB(pid); kb <= 0 {
		t.Fatalf("treeRSSKB(self)=%d want >0", kb)
	}
}

func TestShouldRecycleByPressureThreshold(t *testing.T) {
	p := &Pool{browserPID: os.Getpid()}
	old := RecycleRSSBytes.Load()
	defer RecycleRSSBytes.Store(old)
	RecycleRSSBytes.Store(1) // 1 byte budget → self always exceeds
	if !p.shouldRecycleByPressure() {
		t.Fatal("expected pressure recycle at 1KB budget")
	}
	measured := treeRSSKB(os.Getpid())
	if measured <= 2 {
		t.Fatalf("invalid process memory sample: %d", measured)
	}
	RecycleRSSBytes.Store(measured * 1024 / 2)
	if !p.shouldRecycleByPressure() {
		t.Fatal("RSS sample and byte budget use inconsistent units")
	}
	RecycleRSSBytes.Store(1 << 40) // absurd budget
	if p.shouldRecycleByPressure() {
		t.Fatal("unexpected recycle at huge budget")
	}
}

func TestEnvOverrideParses(t *testing.T) {
	t.Setenv("UIAI_RECYCLE_RSS_MB", "2048")
	initRecycleRSS()
	if got := RecycleRSSBytes.Load(); got != 2048*1024*1024 {
		t.Fatalf("budget=%d want %d", got, 2048*1024)
	}
	_ = strconv.Itoa // keep strconv referenced
}
