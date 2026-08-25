package ratelimit

import (
	"fmt"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

type window struct {
	count   int
	resetAt time.Time
}

// Limiter is process-health rate limiting — classified as runtime-infra per Spec 104 §5.5.5.
// Windows are keyed by (key,tier) only; two projects sharing the same key string share windows.
// This is intentional process-protection, NOT project-isolated authority. Per-scope keying is
// tracked via bead uiai-engine-fg1 (child of h6s); until then callers must not rely on
// rate-limit isolation across scopes. Documented bleed + mitigation in docs/ratelimit.md.
type Limiter struct {
	tiers  map[string]config.TierLimit
	hourly sync.Map // key → *window
	daily  sync.Map // key → *window
}

func New(cfg *config.Config) *Limiter {
	l := &Limiter{tiers: cfg.RateLimits.Tiers}
	go l.cleanLoop()
	return l
}

// Check returns nil if allowed, error with retry info if rate limited.
func (l *Limiter) Check(key, tier string) error {
	limits, ok := l.tiers[tier]
	if !ok {
		limits = config.TierLimit{PerHour: 10, PerDay: 100} // default to free
	}

	now := time.Now()

	// Hourly
	hk := "h:" + key
	hw := l.getOrCreate(&l.hourly, hk, now, time.Hour)
	if hw.count >= limits.PerHour {
		return fmt.Errorf("rate limit exceeded: %d/%d per hour (resets %s)", hw.count, limits.PerHour, hw.resetAt.Format(time.RFC3339))
	}
	hw.count++

	// Daily
	dk := "d:" + key
	dw := l.getOrCreate(&l.daily, dk, now, 24*time.Hour)
	if dw.count >= limits.PerDay {
		return fmt.Errorf("rate limit exceeded: %d/%d per day (resets %s)", dw.count, limits.PerDay, dw.resetAt.Format(time.RFC3339))
	}
	dw.count++

	return nil
}

// Status returns current usage for a key.
func (l *Limiter) Status(key, tier string) (hourUsed, hourLimit, dayUsed, dayLimit int) {
	limits, ok := l.tiers[tier]
	if !ok {
		limits = config.TierLimit{PerHour: 10, PerDay: 100}
	}
	if v, ok := l.hourly.Load("h:" + key); ok {
		hourUsed = v.(*window).count
	}
	if v, ok := l.daily.Load("d:" + key); ok {
		dayUsed = v.(*window).count
	}
	return hourUsed, limits.PerHour, dayUsed, limits.PerDay
}

func (l *Limiter) getOrCreate(m *sync.Map, key string, now time.Time, duration time.Duration) *window {
	v, loaded := m.Load(key)
	if loaded {
		w := v.(*window)
		if now.After(w.resetAt) {
			w.count = 0
			w.resetAt = now.Add(duration)
		}
		return w
	}
	w := &window{count: 0, resetAt: now.Add(duration)}
	actual, _ := m.LoadOrStore(key, w)
	return actual.(*window)
}

func (l *Limiter) cleanLoop() {
	for {
		time.Sleep(5 * time.Minute)
		now := time.Now()
		for _, m := range []*sync.Map{&l.hourly, &l.daily} {
			m.Range(func(key, value any) bool {
				if w, ok := value.(*window); ok && now.After(w.resetAt) {
					m.Delete(key)
				}
				return true
			})
		}
	}
}
