package routes

import (
	"net/http"
	"runtime"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/config"
	"github.com/philoveracity/uiai-engine/internal/storage"
)

var startTime = time.Now()

func MountAdminReal(r chi.Router, cfg *config.Config, usage *storage.UsageStore) {
	r.Get("/services", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{
			"uiai-engine": map[string]any{
				"id": "uiai-engine", "name": "UIAI Engine", "running": true,
				"port": cfg.Server.Port, "health": "healthy", "uptime": time.Since(startTime).String(),
			},
		})
	})

	r.Get("/resources", func(w http.ResponseWriter, req *http.Request) {
		var m runtime.MemStats
		runtime.ReadMemStats(&m)
		writeJSON(w, 200, map[string]any{
			"memory_alloc_mb":  float64(m.Alloc) / 1024 / 1024,
			"memory_sys_mb":    float64(m.Sys) / 1024 / 1024,
			"goroutines":       runtime.NumGoroutine(),
			"uptime":           time.Since(startTime).String(),
			"go_version":       runtime.Version(),
		})
	})

	r.Get("/usage", func(w http.ResponseWriter, req *http.Request) {
		records := usage.All()
		writeJSON(w, 200, map[string]any{"total": len(records)})
	})

	r.Get("/dashboard", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{
			"uptime":     time.Since(startTime).String(),
			"requests":   len(usage.All()),
			"status":     "healthy",
		})
	})

	r.Get("/keys", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{"keys": []any{}, "count": 0})
	})

	r.Get("/tokens", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{"tokens": []any{}, "count": 0})
	})

	r.Get("/recent-usage", func(w http.ResponseWriter, req *http.Request) {
		all := usage.All()
		n := 20
		if len(all) < n {
			n = len(all)
		}
		writeJSON(w, 200, map[string]any{"records": all[len(all)-n:], "count": n})
	})
}
