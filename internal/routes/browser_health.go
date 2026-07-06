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
func MountBrowserHealth(r chi.Router, pool vision.PoolSource) {
	r.Get("/browser", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, browserHealthStatus(pool), browserHealthPayload(pool))
	})
}

func agentPressureSummary(browserStats map[string]any) map[string]any {
	activePages := intFromAny(browserStats["active_pages"])
	availablePages := intFromAny(browserStats["available_pages"])
	maxPages := intFromAny(browserStats["max_pages"])
	failCount := int64FromAny(browserStats["fail_count"])
	queue := mapFromAny(browserStats["queue"])
	queueDepth := intFromAny(queue["depth"])
	queueRejected := intFromAny(queue["rejected"])
	queueP95WaitMs := intFromAny(queue["p95_wait_ms"])

	cacheSummary := map[string]any{"status": "unknown", "pressure": "unknown"}
	if cache, ok := browserStats["cache"].(map[string]any); ok {
		cacheSummary = cache
	}
	cachePressure := cachePressureLevel(cacheSummary)
	cacheSummary["pressure"] = cachePressure

	searchSummary := searchPressureSummary()
	searchPressure := stringFromAny(searchSummary["pressure"], "unknown")
	currentCapacity := currentBrowserCapacity(activePages, availablePages, maxPages, queueDepth)
	historicalPressure := historicalBrowserPressure(queue, failCount)
	browserPressure := browserPressureLevel(activePages, maxPages, failCount, queueDepth, queueRejected, queueP95WaitMs)
	errorPressure := errorPressureLevel(observability.Count())
	overallPressure := maxPressureLevel(browserPressure, searchPressure, cachePressure, errorPressure)
	actions := pressureRecommendedActions(overallPressure, browserPressure, searchPressure, cachePressure, errorPressure)

	return map[string]any{
		"schema":              "uiai.agent_pressure.v1",
		"telemetry_class":     "noncanonical_operational",
		"authority":           "advisory_only",
		"overall_pressure":    overallPressure,
		"recommended_actions": actions,
		"packet": map[string]any{
			"schema":             "uiai.focusa_research_diagnostics_packet.v1",
			"surfaces":           []string{"search", "source_markdown", "browser_read", "browser_snapshot", "browser_diagnostics", "structured_errors", "screenshot", "share"},
			"max_packet_bytes":   8192,
			"composition_status": "pi_http_mcp_cli_available",
			"authority":          "proposal_only_until_focusa_capture",
		},
		"search":              searchSummary,
		"current_capacity":    currentCapacity,
		"historical_pressure": historicalPressure,
		"browser": map[string]any{
			"pressure":            browserPressure,
			"current_capacity":    currentCapacity,
			"historical_pressure": historicalPressure,
			"active_pages":        activePages,
			"available_pages":     availablePages,
			"max_pages":           maxPages,
			"fail_count":          failCount,
			"queue_depth":         queueDepth,
			"queue_rejected":      queueRejected,
			"queue_p95_ms":        queueP95WaitMs,
		},
		"cache":               cacheSummary,
		"errors":              map[string]any{"pressure": errorPressure, "stored_count": observability.Count()},
		"recommended_action":  actions[0],
		"focusa_routing_hint": "Use as operational telemetry only; capture/link evidence through Focusa tools before treating results as durable cognition.",
	}
}

func intFromAny(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	default:
		return 0
	}
}

func int64FromAny(value any) int64 {
	switch v := value.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	default:
		return 0
	}
}

func mapFromAny(value any) map[string]any {
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func stringFromAny(value any, fallback string) string {
	if s, ok := value.(string); ok && s != "" {
		return s
	}
	return fallback
}

func currentBrowserCapacity(activePages, availablePages, maxPages, queueDepth int) map[string]any {
	remainingSlots := 0
	if maxPages > activePages {
		remainingSlots = maxPages - activePages
	}
	return map[string]any{
		"active_pages":         activePages,
		"available_idle_pages": availablePages,
		"max_pages":            maxPages,
		"remaining_page_slots": remainingSlots,
		"queue_depth":          queueDepth,
		"capacity_available":   remainingSlots > 0 && queueDepth == 0,
		"status":               map[bool]string{true: "available", false: "busy"}[remainingSlots > 0 && queueDepth == 0],
	}
}

func historicalBrowserPressure(queue map[string]any, failCount int64) map[string]any {
	return map[string]any{
		"queue_served":      intFromAny(queue["served"]),
		"queue_rejected":    intFromAny(queue["rejected"]),
		"queue_avg_wait_ms": intFromAny(queue["avg_wait_ms"]),
		"queue_p95_wait_ms": intFromAny(queue["p95_wait_ms"]),
		"queue_p99_wait_ms": intFromAny(queue["p99_wait_ms"]),
		"queue_max_wait_ms": intFromAny(queue["max_wait_ms"]),
		"fail_count":        failCount,
		"note":              "historical pressure is retained for detail; current_capacity controls immediate availability",
	}
}

func browserPressureLevel(activePages, maxPages int, failCount int64, queueDepth, queueRejected, queueP95WaitMs int) string {
	if maxPages > 0 && activePages >= maxPages || queueRejected > 0 || queueP95WaitMs >= 5000 {
		return "saturated"
	}
	if queueDepth > 0 || failCount >= 3 {
		return "constrained"
	}
	return "normal"
}

func cachePressureLevel(cache map[string]any) string {
	if stringFromAny(cache["status"], "") == "disabled" {
		return "disabled"
	}
	entries := intFromAny(cache["entries"])
	maxEntries := intFromAny(cache["max_entries"])
	if maxEntries > 0 && entries >= maxEntries {
		return "saturated"
	}
	return "normal"
}

func errorPressureLevel(storedCount int) string {
	if storedCount >= 100 {
		return "saturated"
	}
	if storedCount >= 20 {
		return "constrained"
	}
	return "normal"
}

func maxPressureLevel(levels ...string) string {
	rank := map[string]int{"unknown": 0, "disabled": 0, "normal": 1, "degraded": 2, "constrained": 3, "saturated": 4}
	best := "normal"
	for _, level := range levels {
		if rank[level] > rank[best] {
			best = level
		}
	}
	return best
}

func pressureRecommendedActions(overall, browser, search, cache, errors string) []string {
	actions := []string{"Use bounded packet/source/read metadata and capture durable evidence through Focusa only after scope is verified."}
	if browser == "saturated" || browser == "constrained" {
		actions = append(actions, "Close unused browser sessions, reduce concurrent page work, and prefer browser_read/diagnostics over screenshots.")
	}
	if search == "degraded" || search == "constrained" || search == "saturated" {
		actions = append(actions, "Check /api/search/providers and use cached/known URLs or Source-to-Markdown when live search is degraded.")
	}
	if cache == "saturated" {
		actions = append(actions, "Reduce repeated screenshot/source captures or let cache entries expire before stress workflows.")
	}
	if errors == "constrained" || errors == "saturated" {
		actions = append(actions, "Inspect /api/errors and browser diagnostics before retrying action loops.")
	}
	if overall == "normal" {
		actions = append(actions, "Proceed with normal browser/search/source workflow and close sessions when cleanup recommends.")
	}
	return actions
}

func browserHealthStatus(pool vision.PoolSource) int {
	if pool == nil {
		return http.StatusServiceUnavailable
	}
	stats := pool.Stats()
	if state, _ := stats["browser_state"].(string); state == "dead" {
		return http.StatusServiceUnavailable
	}
	return http.StatusOK
}

func browserHealthPayload(pool vision.PoolSource) map[string]any {
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
	queue := mapFromAny(stats["queue"])
	activePages := intFromAny(stats["active_pages"])
	availablePages := intFromAny(stats["available_pages"])
	maxPages := intFromAny(stats["max_pages"])
	failCount := int64FromAny(stats["fail_count"])
	currentCapacity := currentBrowserCapacity(activePages, availablePages, maxPages, intFromAny(queue["depth"]))
	historicalPressure := historicalBrowserPressure(queue, failCount)
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
		"multi_pool":          stats["multi_pool"],
		"browser_count":       stats["browser_count"],
		"alive_browsers":      stats["alive_browsers"],
		"pools":               stats["pools"],
		"current_capacity":    currentCapacity,
		"historical_pressure": historicalPressure,
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
