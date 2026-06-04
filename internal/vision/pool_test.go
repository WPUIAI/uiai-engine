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

func TestValidateNavigationURLBlocksUnsafeSchemes(t *testing.T) {
	p, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"file:///etc/passwd", "data:text/html,test", "ftp://example.com/file"} {
		if err := p.ValidateNavigationURL(raw); err == nil {
			t.Fatalf("expected unsafe scheme blocked for %s", raw)
		}
	}
}

func TestValidateNavigationURLBlocksPrivateWhenConfigured(t *testing.T) {
	p, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: false})
	if err != nil {
		t.Fatal(err)
	}
	for _, raw := range []string{"http://localhost:7456", "http://127.0.0.1:7456", "http://10.0.0.5"} {
		if err := p.ValidateNavigationURL(raw); err == nil {
			t.Fatalf("expected private URL blocked for %s", raw)
		}
	}
	if err := p.ValidateNavigationURL("https://example.com"); err != nil {
		t.Fatalf("expected public https URL allowed: %v", err)
	}
}

func TestValidateNavigationURLAllowsPrivateWhenConfigured(t *testing.T) {
	p, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.ValidateNavigationURL("http://127.0.0.1:7456"); err != nil {
		t.Fatalf("expected private URL allowed with AllowPrivateURLs: %v", err)
	}
}
