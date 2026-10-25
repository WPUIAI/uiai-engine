package routes

import (
	"context"
	"errors"
	"math"
	"net/http"
	"sync"
	"time"
)

func retryableSearchStatus(status int) bool {
	return status == http.StatusTooManyRequests || status == http.StatusRequestTimeout || (status >= 500 && status < 600)
}

const searchBreakerThreshold = 2
const searchBreakerBaseDelay = 30 * time.Second
const searchBreakerMaxDelay = 5 * time.Minute

type searchBreakersKey struct{}
type searchCircuit struct {
	failures, opens int
	until           time.Time
	probe           bool
	generation      uint64
}
type searchAttempt struct {
	generation uint64
	probe      bool
}

// searchBreakers belongs to one mounted search router, not a tenant-global map.
// Only supported provider ids enter this bounded, value-free transient state.
type searchBreakers struct {
	mu       sync.Mutex
	now      func() time.Time
	circuits map[string]searchCircuit
}
type searchCircuitStatus struct {
	State               string `json:"state"`
	ConsecutiveFailures int    `json:"consecutive_failures"`
	RetryAfterSeconds   int    `json:"retry_after_seconds"`
	ProbeInFlight       bool   `json:"probe_in_flight"`
}
type searchCircuitOpenError struct{ retryAfter int }

func (*searchCircuitOpenError) Error() string { return "search provider circuit is cooling down" }
func newSearchBreakers(now func() time.Time) *searchBreakers {
	return &searchBreakers{now: now, circuits: make(map[string]searchCircuit)}
}
func searchBreakersFor(req *http.Request) *searchBreakers {
	b, _ := req.Context().Value(searchBreakersKey{}).(*searchBreakers)
	return b
}
func (b *searchBreakers) acquire(id string) (searchAttempt, int) {
	if b == nil {
		return searchAttempt{}, 0
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.circuits[id]
	now := b.now()
	if s.until.After(now) {
		return searchAttempt{}, int(math.Ceil(s.until.Sub(now).Seconds()))
	}
	if s.probe {
		return searchAttempt{}, 1
	}
	attempt := searchAttempt{generation: s.generation}
	if !s.until.IsZero() {
		s.probe = true
		attempt.probe = true
		b.circuits[id] = s
	}
	return attempt, 0
}
func (b *searchBreakers) complete(id string, attempt searchAttempt, err error) {
	if b == nil {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.circuits[id]
	if s.generation != attempt.generation {
		return
	} // An older response cannot reopen/clear a newer circuit.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		if attempt.probe {
			s.probe = false
			b.circuits[id] = s
		}
		return
	}
	var upstream *upstreamSearchError
	retryAfter := ""
	if errors.As(err, &upstream) {
		if !retryableSearchStatus(upstream.status) {
			b.circuits[id] = searchCircuit{generation: s.generation + 1}
			return
		}
		retryAfter = upstream.retryAfter
	} else if err == nil {
		b.circuits[id] = searchCircuit{generation: s.generation + 1}
		return
	} else if !attempt.probe {
		return
	}
	// A failed recovery probe also backs off on value-free transport/read errors;
	// cancellation above never consumes this recovery opportunity.
	s.failures = min(searchBreakerThreshold, s.failures+1)
	if s.failures >= searchBreakerThreshold || attempt.probe {
		s.opens = min(s.opens+1, 5)
		delay := min(searchBreakerBaseDelay*time.Duration(1<<uint(s.opens-1)), searchBreakerMaxDelay)
		now := b.now()
		s.until = now.Add(delay)
		if seconds, e := time.ParseDuration(retryAfter + "s"); e == nil && seconds > 0 {
			if floor := now.Add(seconds); floor.After(s.until) {
				s.until = floor
			}
		} else if date, e := http.ParseTime(retryAfter); e == nil && date.After(s.until) {
			s.until = date
		}
		s.probe = false
		s.generation++
	}
	b.circuits[id] = s
}
func (b *searchBreakers) status(id string) searchCircuitStatus {
	if b == nil {
		return searchCircuitStatus{State: "closed"}
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	s := b.circuits[id]
	now := b.now()
	status := searchCircuitStatus{State: "closed", ConsecutiveFailures: s.failures, ProbeInFlight: s.probe}
	if s.until.After(now) {
		status.State = "open"
		status.RetryAfterSeconds = int(math.Ceil(s.until.Sub(now).Seconds()))
	} else if !s.until.IsZero() {
		status.State = "half_open"
		if s.probe {
			status.RetryAfterSeconds = 1
		}
	}
	return status
}
