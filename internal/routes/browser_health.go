package routes

import (
	"net/http"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

// MountBrowserHealth exposes browser-specific readiness and metrics without
// requiring agents to inspect logs or load full tool definitions.
func MountBrowserHealth(r chi.Router, pool *vision.Pool) {
	r.Get("/browser", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, browserHealthStatus(pool), browserHealthPayload(pool))
	})
}

func browserHealthStatus(pool *vision.Pool) int {
	if pool == nil {
		return http.StatusServiceUnavailable
	}
	stats := pool.Stats()
	if state, _ := stats["browser_state"].(string); state == "dead" {
		return http.StatusServiceUnavailable
	}
	return http.StatusOK
}

func browserHealthPayload(pool *vision.Pool) map[string]any {
	if pool == nil {
		return map[string]any{
			"status":       "unavailable",
			"service":      "uiai-browser",
			"generated_at": time.Now().UTC().Format(time.RFC3339),
			"error":        "vision pool not initialized",
		}
	}
	stats := pool.Stats()
	status := "healthy"
	if state, _ := stats["browser_state"].(string); state == "idle-off" {
		status = "standby"
	} else if state == "dead" {
		status = "unavailable"
	}
	return map[string]any{
		"status":              status,
		"service":             "uiai-browser",
		"generated_at":        time.Now().UTC().Format(time.RFC3339),
		"browser_alive":       stats["browser_alive"],
		"browser_state":       stats["browser_state"],
		"browser_pid":         stats["browser_pid"],
		"max_pages":           stats["max_pages"],
		"created_pages":       stats["created_pages"],
		"available_pages":     stats["available_pages"],
		"active_pages":        stats["active_pages"],
		"screenshot_count":    stats["screenshot_count"],
		"fail_count":          stats["fail_count"],
		"queue":               stats["queue"],
		"diagnostics_enabled": true,
		"eval_async_enabled":  true,
		"actions": map[string]string{
			"open_session": "/api/session",
			"diagnostics":  "/api/session/{id}/diagnostics",
			"eval_async":   "/api/session/{id}/eval_async",
			"tool_search":  "/api/tools/search?q=diagnostics",
		},
	}
}
