package captcha

import (
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// StatsTracker records captcha solve attempts for operational visibility.
type StatsTracker struct {
	mu      sync.Mutex
	stats   SolverStats
	logFile string
	enabled bool
}

// NewStatsTracker creates a tracker. If logFile is empty, logging is disabled.
func NewStatsTracker(cfg StatsConfig) *StatsTracker {
	st := &StatsTracker{
		stats: SolverStats{
			ByType: make(map[string]*TypeStats),
		},
		logFile: cfg.LogFile,
		enabled: cfg.Enabled,
	}
	if st.logFile != "" {
		dir := filepath.Dir(st.logFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("[captcha-stats] cannot create log dir %s: %v", dir, err)
		}
	}
	return st
}

// Record logs a solve attempt.
func (st *StatsTracker) Record(entry StatsEntry) {
	if !st.enabled {
		return
	}

	st.mu.Lock()
	defer st.mu.Unlock()

	st.stats.TotalAttempts++
	if entry.Solved {
		st.stats.TotalSolved++
	}
	if st.stats.TotalAttempts > 0 {
		st.stats.SuccessRate = float64(st.stats.TotalSolved) / float64(st.stats.TotalAttempts)
	}

	ts, ok := st.stats.ByType[entry.Type]
	if !ok {
		ts = &TypeStats{}
		st.stats.ByType[entry.Type] = ts
	}
	ts.Attempts++
	if entry.Solved {
		ts.Solved++
	}
	if ts.Attempts > 0 {
		ts.Rate = float64(ts.Solved) / float64(ts.Attempts)
	}

	// Append to JSONL log
	if st.logFile != "" {
		entry.Timestamp = time.Now()
		line, err := json.Marshal(entry)
		if err == nil {
			f, err := os.OpenFile(st.logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err == nil {
				f.Write(line)
				f.Write([]byte("\n"))
				f.Close()
			}
		}
	}
}

// Get returns a snapshot of the current stats.
func (st *StatsTracker) Get() *SolverStats {
	st.mu.Lock()
	defer st.mu.Unlock()
	// Copy
	s := SolverStats{
		TotalAttempts: st.stats.TotalAttempts,
		TotalSolved:   st.stats.TotalSolved,
		SuccessRate:   st.stats.SuccessRate,
		ByType:        make(map[string]*TypeStats),
	}
	for k, v := range st.stats.ByType {
		cp := *v
		s.ByType[k] = &cp
	}
	return &s
}
