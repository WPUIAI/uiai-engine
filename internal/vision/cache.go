package vision

import (
	"crypto/sha256"
	"fmt"
	"sync"
	"time"
)

const (
	DefaultCacheTTL     = 5 * time.Minute
	DefaultCacheMaxSize = 50 * 1024 * 1024 // 50MB
)

type cacheEntry struct {
	data      []byte
	domReport string
	width     int
	height    int
	format    string
	createdAt time.Time
	size      int // len(data)
}

type screenshotCache struct {
	mu      sync.Mutex
	entries map[string]*cacheEntry
	order   []string // oldest first for eviction
	ttl     time.Duration
	maxSize int
	curSize int
	hits    int64
	misses  int64
}

func newScreenshotCache(ttl time.Duration, maxSize int) *screenshotCache {
	if ttl <= 0 {
		ttl = DefaultCacheTTL
	}
	if maxSize <= 0 {
		maxSize = DefaultCacheMaxSize
	}
	return &screenshotCache{
		entries: make(map[string]*cacheEntry),
		ttl:     ttl,
		maxSize: maxSize,
	}
}

// cacheKey generates a deterministic key from screenshot options.
func cacheKey(opts ScreenshotOpts) string {
	raw := fmt.Sprintf("%s|%d|%d|%v|%s|%d|%s",
		opts.URL, opts.Width, opts.Height, opts.FullPage, opts.Format, opts.Quality, opts.Cookies)
	h := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", h[:16]) // 32 hex chars
}

// get returns a cached result if available and not expired.
func (c *screenshotCache) get(key string) *ScreenshotResult {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok {
		c.misses++
		return nil
	}

	if time.Since(entry.createdAt) > c.ttl {
		// Expired — remove it
		c.evictLocked(key)
		c.misses++
		return nil
	}

	c.hits++
	return &ScreenshotResult{
		Data:      entry.data,
		Width:     entry.width,
		Height:    entry.height,
		Format:    entry.format,
		Duration:  0, // cache hit = 0 duration
		DOMReport: entry.domReport,
	}
}

// put stores a screenshot result in the cache.
func (c *screenshotCache) put(key string, result *ScreenshotResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Don't cache if result is too large (>10MB single entry)
	if len(result.Data) > 10*1024*1024 {
		return
	}

	// Evict expired entries first
	c.evictExpiredLocked()

	// Evict oldest entries until we have room
	for c.curSize+len(result.Data) > c.maxSize && len(c.order) > 0 {
		c.evictLocked(c.order[0])
	}

	// If entry already exists, update it
	if old, ok := c.entries[key]; ok {
		c.curSize -= old.size
		// Remove from order list
		for i, k := range c.order {
			if k == key {
				c.order = append(c.order[:i], c.order[i+1:]...)
				break
			}
		}
	}

	entry := &cacheEntry{
		data:      result.Data,
		domReport: result.DOMReport,
		width:     result.Width,
		height:    result.Height,
		format:    result.Format,
		createdAt: time.Now(),
		size:      len(result.Data),
	}
	c.entries[key] = entry
	c.order = append(c.order, key)
	c.curSize += entry.size
}

func (c *screenshotCache) evictLocked(key string) {
	if entry, ok := c.entries[key]; ok {
		c.curSize -= entry.size
		delete(c.entries, key)
		for i, k := range c.order {
			if k == key {
				c.order = append(c.order[:i], c.order[i+1:]...)
				break
			}
		}
	}
}

func (c *screenshotCache) evictExpiredLocked() {
	now := time.Now()
	for len(c.order) > 0 {
		key := c.order[0]
		entry, ok := c.entries[key]
		if !ok {
			c.order = c.order[1:]
			continue
		}
		if now.Sub(entry.createdAt) > c.ttl {
			c.evictLocked(key)
		} else {
			break // entries are in order, so first non-expired means rest are fresh
		}
	}
}

// stats returns cache statistics.
func (c *screenshotCache) stats() map[string]any {
	c.mu.Lock()
	defer c.mu.Unlock()
	return map[string]any{
		"entries":  len(c.entries),
		"size_mb":  float64(c.curSize) / (1024 * 1024),
		"max_mb":   float64(c.maxSize) / (1024 * 1024),
		"hits":     c.hits,
		"misses":   c.misses,
		"hit_rate": hitRate(c.hits, c.misses),
	}
}

func hitRate(hits, misses int64) float64 {
	total := hits + misses
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total)
}
