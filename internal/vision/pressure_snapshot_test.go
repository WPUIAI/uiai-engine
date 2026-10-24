package vision

import (
	"math"
	"testing"
	"time"

	"github.com/go-rod/rod"
)

func TestPSProcessTreeRSSSnapshots(t *testing.T) {
	for _, test := range []struct {
		name     string
		root     int
		snapshot string
		want     int64
	}{
		{"descendants only", 100, "100 1 10\n101 100 20\n102 101 30\n200 1 999\n", 60},
		{"unreadable parent still reaches child", 100, "100 1 10\n101 100 -\n102 101 7\n", 17},
		{"cycle and duplicate rows", 1, "1 2 10\n2 1 20\n1 2 10\n", 30},
		{"malformed and negative", 1, "PID PPID RSS\nbad\n1 0 10\n2 1 -5\n3 invalid 90\n", 10},
		{"missing root", 99, "1 0 10\n", 0},
		{"invalid root", 0, "1 0 10\n", 0},
		{"saturating sum", 1, "1 0 9223372036854775800\n2 1 99\n", math.MaxInt64},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := psTreeRSSKB(test.root, []byte(test.snapshot)); got != test.want {
				t.Fatalf("RSS=%d want=%d", got, test.want)
			}
		})
	}
}

func TestRSSPressureSnapshotDoesNotAdvertiseUnmeasuredProtection(t *testing.T) {
	if !rssMeasurementSupported() {
		t.Skip("unsupported platform has a separate declared posture")
	}
	previous := RecycleRSSBytes.Load()
	defer RecycleRSSBytes.Store(previous)
	RecycleRSSBytes.Store(64 * 1024 * 1024)
	pool := &Pool{browser: &rod.Browser{}, browserPID: 123}
	check := func(want string, active bool) {
		t.Helper()
		pool.mu.Lock()
		status := pool.rssPressureSnapshotLocked()
		pool.mu.Unlock()
		if status["measurement"] != want || status["protection_active"] != active {
			t.Fatalf("RSS posture=%v want %s active=%v", status, want, active)
		}
	}
	check("pending", false)
	pool.rssSamplePID, pool.rssSampleAt = 123, time.Now().UTC()
	check("unavailable", false)
	pool.rssSampleKB = 1024
	check("sampled", true)
	pool.browserPID = 456
	check("pending", false)
	RecycleRSSBytes.Store(0)
	check("disabled", false)
	RecycleRSSBytes.Store(64 * 1024 * 1024)
	pool.browser = nil
	check("no_browser", false)
}
