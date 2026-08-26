package vision

import (
	"github.com/WPUIAI/uiai-engine/internal/events"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// C-010-11 — pressure-based recycle thresholds.
// Chrome trees bloat well before screenshot counts trip; recycle on resident
// memory too. Default 1.5 GiB, overridable via UIAI_RECYCLE_RSS_MB.
var (
	RecycleRSSBytes atomic.Int64
	recycleRSSOnce  sync.Once
)

func init() { initRecycleRSS() }

func initRecycleRSS() {
	mb := int64(1536)
	if v := strings.TrimSpace(os.Getenv("UIAI_RECYCLE_RSS_MB")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n > 64 {
			mb = n
		}
	}
	RecycleRSSBytes.Store(mb * 1024)
}

// shouldRecycleByPressure reports whether the browser tree exceeded the RSS budget.
func (p *Pool) shouldRecycleByPressure() bool {
	limit := RecycleRSSBytes.Load()
	if limit <= 0 || p.browserPID <= 0 {
		return false
	}
	rss := treeRSSKB(p.browserPID)
	return rss > limit*1024 // KB compare
}

// treeRSSKB sums resident memory of pid and all descendants via procfs.
func treeRSSKB(pid int) int64 {
	seen := map[int]bool{}
	var walk func(int) int64
	walk = func(p int) int64 {
		if seen[p] {
			return 0
		}
		seen[p] = true
		var total int64
		if data, err := os.ReadFile("/proc/" + itoa(p) + "/status"); err == nil {
			for _, line := range strings.Split(string(data), "\n") {
				if strings.HasPrefix(line, "VmRSS:") {
					fields := strings.Fields(line)
					if len(fields) >= 2 {
						if kb, err := strconv.ParseInt(fields[1], 10, 64); err == nil {
							total += kb
						}
					}
					break
				}
			}
		}
		if entries, err := os.ReadDir("/proc/" + itoa(p) + "/task"); err == nil {
			for _, e := range entries {
				ch, err := os.ReadFile("/proc/" + itoa(p) + "/task/" + e.Name() + "/children")
				if err != nil {
					continue
				}
				for _, f := range strings.Fields(string(ch)) {
					if cpid, err := strconv.Atoi(f); err == nil {
						total += walk(cpid)
					}
				}
			}
		}
		return total
	}
	return walk(pid)
}

func itoa(i int) string { return strconv.Itoa(i) }

// StartPressureRecycler ticks and recycles on memory pressure (C-010-11).
func (p *Pool) StartPressureRecycler(every time.Duration) {
	go func() {
		t := time.NewTicker(every)
		defer t.Stop()
		for range t.C {
			p.mu.Lock()
			alive := p.browser != nil && p.browserPID > 0
			p.mu.Unlock()
			if !alive {
				continue
			}
			if p.shouldRecycleByPressure() {
				log.Printf("[vision] Pressure recycle triggered (tree RSS above budget)")
				p.scheduleDrainRestart("rss_pressure")
			}
		}
	}()
}

// scheduleDrainRestart waits for active pages to drain, then restarts Chrome.
func (p *Pool) scheduleDrainRestart(reason string) {
	events.Emit("browser.restarted", map[string]any{"reason": reason}, []string{"browser_fleet"}, nil)
	go func() {
		for i := 0; i < 30; i++ {
			p.mu.Lock()
			active := p.active
			p.mu.Unlock()
			if active == 0 {
				break
			}
			time.Sleep(time.Second)
		}
		p.mu.Lock()
		if p.active == 0 {
			log.Printf("[vision] Recycling Chrome (%s)", reason)
			p.restartBrowser()
		}
		p.mu.Unlock()
	}()
}
