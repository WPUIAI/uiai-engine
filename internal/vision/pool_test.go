package vision

import (
	"sync/atomic"
	"testing"
)

func TestPoolStatsQueueMetrics(t *testing.T) {
	p := &Pool{maxPages: 4, queueMax: 8}
	atomic.StoreInt64(&p.queued, 2)
	atomic.StoreInt64(&p.queueDone, 4)
	atomic.StoreInt64(&p.queueDrop, 1)
	atomic.StoreInt64(&p.queueWaitTotalMs, 100)
	atomic.StoreInt64(&p.queueWaitMaxMs, 40)
	p.queueWaitSamples = []int64{10, 20, 30, 40}

	stats := p.Stats()
	queue, ok := stats["queue"].(map[string]any)
	if !ok {
		t.Fatalf("queue stats missing: %#v", stats["queue"])
	}
	if got, want := queue["depth"], int64(2); got != want {
		t.Fatalf("depth=%v want=%v", got, want)
	}
	if got, want := queue["served"], int64(4); got != want {
		t.Fatalf("served=%v want=%v", got, want)
	}
	if got, want := queue["rejected"], int64(1); got != want {
		t.Fatalf("rejected=%v want=%v", got, want)
	}
	if got, want := queue["avg_wait_ms"], int64(25); got != want {
		t.Fatalf("avg_wait_ms=%v want=%v", got, want)
	}
	if got, want := queue["p95_wait_ms"], int64(40); got != want {
		t.Fatalf("p95_wait_ms=%v want=%v", got, want)
	}
	if got, want := queue["p99_wait_ms"], int64(40); got != want {
		t.Fatalf("p99_wait_ms=%v want=%v", got, want)
	}
	if got, want := queue["max_wait_ms"], int64(40); got != want {
		t.Fatalf("max_wait_ms=%v want=%v", got, want)
	}
}

func TestQueueWaitSamplesAreBounded(t *testing.T) {
	p := &Pool{}
	for i := int64(0); i < 600; i++ {
		p.recordQueueWaitSampleLocked(i)
	}
	if got, want := len(p.queueWaitSamples), 512; got != want {
		t.Fatalf("sample len=%d want=%d", got, want)
	}
	if got, want := p.queueWaitSamples[0], int64(88); got != want {
		t.Fatalf("first retained sample=%d want=%d", got, want)
	}
}
