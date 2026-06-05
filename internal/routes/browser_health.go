package routes

import (
	"net/http"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/observability"
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

func agentPressureSummary(browserStats map[string]any) map[string]any {
	activePages, _ := browserStats["active_pages"].(int)
	availablePages, _ := browserStats["available_pages"].(int)
	maxPages, _ := browserStats["max_pages"].(int)
	failCount, _ := browserStats["fail_count"].(int64)
	if failCount == 0 {
		if v, ok := browserStats["fail_count"].(int); ok {
			failCount = int64(v)
		}
	}
	cacheSummary := map[string]any{"status": "unknown"}
	if cache, ok := browserStats["cache"].(map[string]any); ok {
		cacheSummary = cache
	}
	browserPressure := "normal"
	if maxPages > 0 && activePages >= maxPages {
		browserPressure = "saturated"
	} else if availablePages == 0 && activePages > 0 {
		browserPressure = "constrained"
	}
	return map[string]any{
		"packet": map[string]any{
			"schema":             "uiai.focusa_research_diagnostics_packet.v1",
			"surfaces":           []string{"search", "browser_read", "browser_diagnostics", "structured_errors"},
			"max_packet_bytes":   8192,
			"composition_status": "pi_extension_first",
		},
		"search": searchPressureSummary(),
		"browser": map[string]any{
			"pressure":        browserPressure,
			"active_pages":    activePages,
			"available_pages": availablePages,
			"max_pages":       maxPages,
			"fail_count":      failCount,
		},
		"cache":              cacheSummary,
		"errors":             map[string]any{"stored_count": observability.Count()},
		"recommended_action": "Use uiai_focusa_packet_build with bounded response metadata; close browser sessions when cleanup recommends; narrow diagnostics if browser pressure is constrained.",
	}
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
		"agent_pressure":      agentPressureSummary(stats),
		"actions": map[string]string{
			"open_session": "/api/session",
			"diagnostics":  "/api/session/{id}/diagnostics",
			"eval_async":   "/api/session/{id}/eval_async",
			"tool_search":  "/api/tools/search?q=diagnostics",
		},
	}
}
