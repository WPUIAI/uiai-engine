package vision

import (
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// PageTargetInfo is the reconciler's minimal view of a live browser target.
type PageTargetInfo struct {
	ID  string
	URL string
}

// TargetSource is implemented by pools that can enumerate and close raw
// browser targets. *Pool and *MultiPool implement it; the reconciler degrades
// gracefully when a PoolSource does not.
type TargetSource interface {
	PageTargets() []PageTargetInfo
	CloseTarget(targetID string) error
}

// Reconciler periodically diffs the session registry against ground truth
// from Chrome itself (Spec 76 pillar 3 / issue #45):
//
//   - orphan page targets (no owning session, non-blank URL) are force-closed;
//   - registry sessions whose target vanished are reaped from the registry.
//
// It exists so leak-class bugs can never accumulate silently again: every
// cycle converts divergence into either self-healing or a visible counter.
type Reconciler struct {
	sm       *SessionManager
	interval time.Duration

	leakedTotal uint64 // orphan targets force-closed since process start
	reapedTotal uint64 // dead sessions removed from the registry
	lastRun     atomic.Value
	lastOrphans int64
	lastReaped  int64
	stopCh      chan struct{}
	stopOnce    sync.Once
	running     atomic.Bool
}

// StartReconciler attaches a reconciler to the manager and starts its loop.
// Safe to call once per manager; subsequent calls are no-ops.
func (sm *SessionManager) StartReconciler(interval time.Duration) {
	if sm == nil || interval <= 0 {
		return
	}
	sm.mu.Lock()
	if sm.reconciler != nil {
		sm.mu.Unlock()
		return
	}
	r := &Reconciler{sm: sm, interval: interval, stopCh: make(chan struct{})}
	sm.reconciler = r
	sm.mu.Unlock()
	r.running.Store(true)
	go r.loop()
}

// StopReconciler halts the loop if running.
func (sm *SessionManager) StopReconciler() {
	if sm == nil {
		return
	}
	sm.mu.RLock()
	r := sm.reconciler
	sm.mu.RUnlock()
	if r != nil {
		r.Stop()
	}
}

func (r *Reconciler) loop() {
	t := time.NewTicker(r.interval)
	defer t.Stop()
	for {
		select {
		case <-r.stopCh:
			return
		case <-t.C:
			r.ReconcileOnce()
		}
	}
}

// Stop terminates the loop.
func (r *Reconciler) Stop() { r.stopOnce.Do(func() { close(r.stopCh) }) }

// Snapshot returns counters for health/observability surfaces.
func (r *Reconciler) Snapshot() map[string]any {
	last, _ := r.lastRun.Load().(time.Time)
	return map[string]any{
		"running":               r.running.Load(),
		"interval_s":            int(r.interval.Seconds()),
		"pages_leaked_total":    atomic.LoadUint64(&r.leakedTotal),
		"reaped_sessions_total": atomic.LoadUint64(&r.reapedTotal),
		"last_run":              last.UTC().Format(time.RFC3339),
		"last_orphans_closed":   atomic.LoadInt64(&r.lastOrphans),
		"last_reaped":           atomic.LoadInt64(&r.lastReaped),
	}
}

// ReconcileOnce performs one diff-and-heal pass.
func (r *Reconciler) ReconcileOnce() {
	if r.sm == nil || r.sm.pool == nil {
		return
	}
	ts, ok := r.sm.pool.(TargetSource)
	if !ok {
		return // pool cannot enumerate targets (e.g. tests with fakes)
	}
	targets := ts.PageTargets()

	live := make(map[string]bool, len(targets))
	for _, t := range targets {
		live[t.ID] = true
	}

	// Session target-ID set (skip sessions whose lookup failed this pass).
	sessions := r.sm.List()
	sessionTargets := make(map[string]bool, len(sessions))
	for _, s := range sessions {
		if tid := s.PageTargetID(); tid != "" {
			sessionTargets[tid] = true
		}
	}

	// Orphan targets: real pages owned by nobody. Blank URLs belong to the
	// warm/idle pool channel and are intentionally spared.
	orphans := DiffOrphans(targets, sessionTargets)
	closed := 0
	for _, o := range orphans {
		if err := ts.CloseTarget(o.ID); err != nil {
			log.Printf("[reconciler] close orphan %s (%s) failed: %v", o.ID, o.URL, err)
			continue
		}
		closed++
	}
	atomic.AddUint64(&r.leakedTotal, uint64(closed))

	// Dead sessions: registered but their target no longer exists. The page
	// died underneath us (browser restart race, crash); reap the record so
	// agents get clean already_closed semantics instead of ghost 404s.
	reaped := 0
	for _, s := range sessions {
		tid := s.PageTargetID()
		if tid == "" {
			continue // indeterminate this pass; never kill on unknown
		}
		if !live[tid] {
			if err := r.sm.Close(s.ID); err == nil {
				reaped++
				log.Printf("[reconciler] reaped dead session %s (target gone)", s.ID)
			}
		}
	}

	now := time.Now()
	r.lastRun.Store(now)
	atomic.StoreInt64(&r.lastOrphans, int64(closed))
	atomic.StoreInt64(&r.lastReaped, int64(reaped))
	if closed > 0 || reaped > 0 {
		log.Printf("[reconciler] pass: closed=%d orphans, reaped=%d dead sessions", closed, reaped)
	}
}

// DiffOrphans returns page targets that are neither blank-pool pages nor
// owned by a live session. Pure function; unit-tested.
func DiffOrphans(targets []PageTargetInfo, sessionTargetIDs map[string]bool) []PageTargetInfo {
	out := []PageTargetInfo{}
	for _, t := range targets {
		u := strings.TrimSpace(t.URL)
		if u == "" || u == "about:blank" {
			continue
		}
		if t.ID == "" || sessionTargetIDs[t.ID] {
			continue
		}
		out = append(out, t)
	}
	return out
}
