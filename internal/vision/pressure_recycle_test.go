package vision

import (
	"os/exec"
	"testing"
	"time"
)

// TestPressureRecycleDoesNotDeadlockPool guards the 2026-09-11 OVH outage:
// scheduleDrainRestart used to call restartBrowser while holding p.mu, and
// restartBrowser locks p.mu again (sync.Mutex is not reentrant), wedging the
// pool forever after every zero-active RSS-pressure recycle.
//
// Regression signals (no real-Chromium dependency, no flake):
//  1. the recycle kills the stale fake browser process (restart path ran);
//  2. afterwards p.mu is releasable — with the old code the recycle
//     goroutine self-deadlocks holding p.mu forever.
func TestPressureRecycleDoesNotDeadlockPool(t *testing.T) {
	p := &Pool{}

	// Simulate a live managed browser process that the recycle must kill.
	cmd := exec.Command("sleep", "300")
	if err := cmd.Start(); err != nil {
		t.Fatalf("spawn fake browser: %v", err)
	}
	defer func() { _ = cmd.Process.Kill() }()

	p.mu.Lock()
	p.browserPID = cmd.Process.Pid
	p.mu.Unlock()

	p.scheduleDrainRestart("test_recycle")

	// restartBrowser kills the old PID before it takes p.mu for the drain
	// step, so on BOTH old and new code the kill must happen; on the old
	// code the goroutine then deadlocks holding p.mu forever.
	waitCh := make(chan error, 1)
	go func() { waitCh <- cmd.Wait() }()
	select {
	case <-waitCh:
		// killed and reaped — restart path reached the kill step
	case <-time.After(15 * time.Second):
		t.Fatal("recycle did not kill the stale browser process")
	}

	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if p.mu.TryLock() {
			p.mu.Unlock()
			return // mutex released — no deadlock
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("pool mutex still held after pressure recycle completed (deadlock regression)")
}