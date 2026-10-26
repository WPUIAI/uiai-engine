package routes

import (
	"net/http"
	"strconv"
	"strings"
	"time"
)

const maxSearchQuotaWindows = 8
const maxSearchQuotaNumber = (1 << 53) - 1
const maxSearchQuotaDelay = 366 * 24 * 60 * 60
const searchQuotaObservationTTL = 5 * time.Minute

type searchQuotaWindow struct {
	Limit             uint64 `json:"limit_as_observed"`
	WindowSeconds     uint64 `json:"window_seconds"`
	Remaining         uint64 `json:"remaining_as_observed"`
	ResetAfterSeconds uint64 `json:"reset_after_seconds_as_observed"`
	Unlimited         bool   `json:"unlimited_as_observed"`
	ResetElapsed      bool   `json:"reported_reset_elapsed"`
}
type searchQuotaObservation struct {
	receivedAt time.Time
	status     int
	windows    []searchQuotaWindow
}
type searchQuotaProjection struct {
	Schema                 string              `json:"schema"`
	Status                 string              `json:"status"`
	Source                 string              `json:"source,omitempty"`
	Scope                  string              `json:"scope"`
	Authority              string              `json:"authority"`
	AccountBinding         string              `json:"account_binding"`
	CurrentRemainingStatus string              `json:"current_remaining_status"`
	ReceivedAt             *time.Time          `json:"received_at,omitempty"`
	ResponseStatus         int                 `json:"response_status,omitempty"`
	Stale                  bool                `json:"stale"`
	ClockUncertain         bool                `json:"clock_uncertain"`
	Windows                []searchQuotaWindow `json:"windows,omitempty"`
}

func searchQuotaHeader(h http.Header, key string) (string, bool) {
	values := h.Values(key)
	size := 0
	for _, v := range values {
		size += len(v) + 1
		if size > 1024 {
			return "", false
		}
	}
	raw := strings.Join(values, ",")
	return raw, strings.TrimSpace(raw) != ""
}
func searchQuotaNumber(raw string) (uint64, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0, false
	}
	for _, c := range raw {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	n, err := strconv.ParseUint(raw, 10, 53)
	return n, err == nil && n <= maxSearchQuotaNumber
}

// The documented Brave four-header contract is parsed as coherent numeric
// dimensions. Unknown/malformed/oversize fields never become telemetry strings.
func parseBraveSearchQuota(h http.Header, status int, received time.Time) (searchQuotaObservation, bool) {
	keys := []string{"X-RateLimit-Limit", "X-RateLimit-Policy", "X-RateLimit-Remaining", "X-RateLimit-Reset"}
	fields := make([][]string, len(keys))
	for i, key := range keys {
		raw, ok := searchQuotaHeader(h, key)
		if !ok {
			return searchQuotaObservation{}, false
		}
		fields[i] = strings.Split(raw, ",")
		if len(fields[i]) == 0 || len(fields[i]) > maxSearchQuotaWindows || i > 0 && len(fields[i]) != len(fields[0]) {
			return searchQuotaObservation{}, false
		}
	}
	out := searchQuotaObservation{receivedAt: received, status: status}
	for i := range fields[0] {
		limit, ok := searchQuotaNumber(fields[0][i])
		if !ok {
			return searchQuotaObservation{}, false
		}
		parts := strings.Split(strings.TrimSpace(fields[1][i]), ";")
		if len(parts) != 2 {
			return searchQuotaObservation{}, false
		}
		policyLimit, ok := searchQuotaNumber(parts[0])
		if !ok || policyLimit != limit {
			return searchQuotaObservation{}, false
		}
		policy := strings.TrimSpace(parts[1])
		if !strings.HasPrefix(policy, "w=") {
			return searchQuotaObservation{}, false
		}
		window, ok := searchQuotaNumber(strings.TrimPrefix(policy, "w="))
		if !ok || window == 0 || window > maxSearchQuotaDelay {
			return searchQuotaObservation{}, false
		}
		remaining, ok := searchQuotaNumber(fields[2][i])
		if !ok || limit > 0 && remaining > limit {
			return searchQuotaObservation{}, false
		}
		reset, ok := searchQuotaNumber(fields[3][i])
		if !ok || reset > maxSearchQuotaDelay {
			return searchQuotaObservation{}, false
		}
		out.windows = append(out.windows, searchQuotaWindow{Limit: limit, WindowSeconds: window, Remaining: remaining, ResetAfterSeconds: reset, Unlimited: limit == 0})
	}
	return out, true
}
