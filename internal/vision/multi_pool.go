package vision

import (
	"errors"
	"log"
	"sync"
	"sync/atomic"

	"github.com/go-rod/rod"
)

// MultiPool orchestrates N independent *Pool instances (each its own Chromium process).
// This bypasses single-Chromium CDP screenshot serialization by spreading work across
// multiple browser processes inside one UIAI worker.
type MultiPool struct {
	mu      sync.Mutex
	pools   []*Pool
	owners  map[*rod.Page]*Pool
	rrCount uint64
}

func NewMultiPool(cfgs []PoolConfig) (*MultiPool, error) {
	if len(cfgs) == 0 {
		return nil, errors.New("multi pool requires at least one pool config")
	}
	mp := &MultiPool{pools: make([]*Pool, 0, len(cfgs)), owners: make(map[*rod.Page]*Pool)}
	for i, cfg := range cfgs {
		p, err := NewPoolWithConfig(cfg)
		if err != nil {
			mp.Close()
			return nil, err
		}
		log.Printf("[vision] multi-pool slot %d ready (max_pages=%d)", i, cfg.MaxPages)
		mp.pools = append(mp.pools, p)
	}
	return mp, nil
}

func (mp *MultiPool) nextPool(offset uint64) *Pool {
	if mp == nil || len(mp.pools) == 0 {
		return nil
	}
	return mp.pools[int(offset%uint64(len(mp.pools)))]
}

func (mp *MultiPool) GetPage() (*rod.Page, error) {
	if mp == nil || len(mp.pools) == 0 {
		return nil, errors.New("multi pool unavailable")
	}
	n := uint64(len(mp.pools))
	start := atomic.AddUint64(&mp.rrCount, 1)
	var lastErr error
	for i := uint64(0); i < n; i++ {
		pool := mp.nextPool(start + i)
		if pool == nil {
			continue
		}
		page, err := pool.GetPage()
		if err == nil {
			mp.mu.Lock()
			mp.owners[page] = pool
			mp.mu.Unlock()
			return page, nil
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, ErrQueueFull
}

func (mp *MultiPool) ReleasePage(page *rod.Page) {
	if page == nil || mp == nil {
		return
	}
	mp.mu.Lock()
	pool := mp.owners[page]
	delete(mp.owners, page)
	mp.mu.Unlock()
	if pool != nil {
		pool.ReleasePage(page)
	}
}

func (mp *MultiPool) ValidateNavigationURL(rawURL string) error {
	if mp == nil || len(mp.pools) == 0 {
		return errors.New("multi pool unavailable")
	}
	return mp.pools[0].ValidateNavigationURL(rawURL)
}

func (mp *MultiPool) IsBrowserAlive() bool {
	if mp == nil || len(mp.pools) == 0 {
		return false
	}
	for _, p := range mp.pools {
		if p.IsBrowserAlive() {
			return true
		}
	}
	return false
}

func (mp *MultiPool) MarkFailure() {
	if mp == nil {
		return
	}
	for _, p := range mp.pools {
		p.MarkFailure()
	}
}

func (mp *MultiPool) Reset() {
	if mp == nil {
		return
	}
	for _, p := range mp.pools {
		p.Reset()
	}
}

func (mp *MultiPool) Screenshot(opts ScreenshotOpts) (*ScreenshotResult, error) {
	if mp == nil || len(mp.pools) == 0 {
		return nil, errors.New("multi pool unavailable")
	}
	n := uint64(len(mp.pools))
	start := atomic.AddUint64(&mp.rrCount, 1)
	var lastErr error
	for i := uint64(0); i < n; i++ {
		pool := mp.nextPool(start + i)
		if pool == nil {
			continue
		}
		res, err := pool.Screenshot(opts)
		if err == nil {
			return res, nil
		}
		lastErr = err
	}
	return nil, lastErr
}

func (mp *MultiPool) Stats() map[string]any {
	stats := map[string]any{"multi_pool": true}
	if mp == nil {
		stats["browser_count"] = 0
		stats["browser_alive"] = false
		stats["browser_state"] = "dead"
		return stats
	}

	alive := 0
	maxPages := 0
	createdPages := 0
	availablePages := 0
	activePages := 0
	failCount := 0
	screenshotCount := int64(0)
	queueDepth := int64(0)
	queueMax := 0
	queueServed := int64(0)
	queueRejected := int64(0)
	children := make([]map[string]any, 0, len(mp.pools))

	for _, p := range mp.pools {
		child := p.Stats()
		children = append(children, child)
		if p.IsBrowserAlive() {
			alive++
		}
		maxPages += intAny(child["max_pages"])
		createdPages += intAny(child["created_pages"])
		availablePages += intAny(child["available_pages"])
		activePages += intAny(child["active_pages"])
		failCount += intAny(child["fail_count"])
		screenshotCount += int64Any(child["screenshot_count"])
		if q, ok := child["queue"].(map[string]any); ok {
			queueDepth += int64Any(q["depth"])
			queueMax += intAny(q["max"])
			queueServed += int64Any(q["served"])
			queueRejected += int64Any(q["rejected"])
		}
	}

	mp.mu.Lock()
	inUse := len(mp.owners)
	mp.mu.Unlock()
	state := "idle-off"
	deadChildren := 0
	for _, child := range children {
		if child["browser_state"] == "dead" {
			deadChildren++
		}
	}
	if alive > 0 {
		state = "running"
	} else if len(mp.pools) > 0 && deadChildren == len(mp.pools) {
		state = "dead"
	}
	stats["browser_count"] = len(mp.pools)
	stats["browser_alive"] = alive > 0
	stats["browser_state"] = state
	stats["alive_browsers"] = alive
	stats["browser_pid"] = nil
	stats["max_pages"] = maxPages
	stats["created_pages"] = createdPages
	stats["available_pages"] = availablePages
	stats["active_pages"] = activePages
	stats["sessions_in_use"] = inUse
	stats["fail_count"] = failCount
	stats["screenshot_count"] = screenshotCount
	stats["queue"] = map[string]any{"depth": queueDepth, "max": queueMax, "served": queueServed, "rejected": queueRejected}
	stats["pools"] = children
	return stats
}

func intAny(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	default:
		return 0
	}
}

func int64Any(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func (mp *MultiPool) Close() {
	if mp == nil {
		return
	}
	for _, p := range mp.pools {
		p.Close()
	}
}
