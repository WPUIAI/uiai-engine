package routes

import (
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/vision"
)

func TestBrowserHealthPayloadUnavailableWhenPoolNil(t *testing.T) {
	payload := browserHealthPayload(nil, true, nil)
	if got, want := payload["status"], "unavailable"; got != want {
		t.Fatalf("status=%v want=%v", got, want)
	}
	if got, want := browserHealthStatus(nil, true, nil), 503; got != want {
		t.Fatalf("status code=%d want=%d", got, want)
	}
}

func TestBrowserHealthStandbyWhenPoolLazyIdle(t *testing.T) {
	pool := &vision.Pool{}
	payload := browserHealthPayload(pool, true, nil)
	if got, want := payload["status"], "standby"; got != want {
		t.Fatalf("status=%v want=%v", got, want)
	}
	if got, want := browserHealthStatus(pool, true, nil), 200; got != want {
		t.Fatalf("status code=%d want=%d", got, want)
	}
}

func TestBrowserHealthDisabledIsTruthfulAndHealthy(t *testing.T) {
	payload := browserHealthPayload(nil, false, nil)
	if got, want := payload["status"], "disabled"; got != want {
		t.Fatalf("status=%v want=%v", got, want)
	}
	if got, want := payload["vision_enabled"], false; got != want {
		t.Fatalf("vision_enabled=%v want=%v", got, want)
	}
	if got, want := browserHealthStatus(nil, false, nil), 200; got != want {
		t.Fatalf("status code=%d want=%d", got, want)
	}
}

func TestBrowserHealthPayloadIncludesAgentPressure(t *testing.T) {
	pool := &vision.Pool{}
	payload := browserHealthPayload(pool, true, nil)
	pressure, ok := payload["agent_pressure"].(map[string]any)
	if !ok {
		t.Fatalf("expected agent_pressure map: %#v", payload)
	}
	if pressure["schema"] != "uiai.agent_pressure.v1" || pressure["telemetry_class"] != "noncanonical_operational" || pressure["authority"] != "advisory_only" {
		t.Fatalf("missing authority/telemetry contract: %#v", pressure)
	}
	packet := pressure["packet"].(map[string]any)
	if packet["schema"] != "uiai.focusa_research_diagnostics_packet.v1" || packet["authority"] != "proposal_only_until_focusa_capture" {
		t.Fatalf("missing packet pressure schema: %#v", packet)
	}
	search := pressure["search"].(map[string]any)
	if search["packet_surface"] != "search" || search["pressure"] == "" {
		t.Fatalf("missing search pressure surface: %#v", search)
	}
	browser := pressure["browser"].(map[string]any)
	if browser["pressure"] == "" {
		t.Fatalf("missing browser pressure: %#v", browser)
	}
	capacity := pressure["current_capacity"].(map[string]any)
	if capacity["capacity_available"] == nil || capacity["remaining_page_slots"] == nil {
		t.Fatalf("missing current capacity split: %#v", pressure)
	}
	history := pressure["historical_pressure"].(map[string]any)
	if history["queue_p95_wait_ms"] == nil || history["note"] == "" {
		t.Fatalf("missing historical pressure split: %#v", pressure)
	}
	if pressure["overall_pressure"] == "" || pressure["recommended_action"] == "" {
		t.Fatalf("missing recommended action: %#v", pressure)
	}
	actions := pressure["recommended_actions"].([]string)
	if len(actions) == 0 {
		t.Fatalf("missing recommended_actions: %#v", pressure)
	}
}

func TestAgentPressureSummaryDoesNotLookSaturatedWhenCapacityAvailable(t *testing.T) {
	pressure := agentPressureSummary(map[string]any{
		"active_pages":    1,
		"available_pages": 0,
		"max_pages":       2,
		"fail_count":      int64(0),
		"queue":           map[string]any{"depth": 0, "rejected": 0, "p95_wait_ms": 0},
	}, nil)
	browser := pressure["browser"].(map[string]any)
	capacity := pressure["current_capacity"].(map[string]any)
	if browser["pressure"] == "saturated" || browser["pressure"] == "constrained" {
		t.Fatalf("capacity-available health must not look saturated/constrained: %#v", pressure)
	}
	if capacity["capacity_available"] != true || capacity["remaining_page_slots"] != 1 {
		t.Fatalf("expected explicit available capacity: %#v", capacity)
	}
}

func TestAgentPressureSummaryIgnoresHistoricalOverloadWhenCapacityReturns(t *testing.T) {
	pressure := agentPressureSummary(map[string]any{
		"active_pages":    0,
		"available_pages": 0,
		"max_pages":       1,
		"fail_count":      int64(7),
		"queue": map[string]any{
			"depth":       0,
			"served":      9,
			"rejected":    14,
			"p95_wait_ms": 24590,
		},
	}, nil)

	browser := pressure["browser"].(map[string]any)
	capacity := pressure["current_capacity"].(map[string]any)
	history := pressure["historical_pressure"].(map[string]any)
	if got, want := browser["pressure"], "normal"; got != want {
		t.Fatalf("browser pressure=%v want=%v: %#v", got, want, pressure)
	}
	if got := pressure["overall_pressure"]; got == "saturated" {
		t.Fatalf("historical browser overload must not saturate overall pressure: %#v", pressure)
	}
	if capacity["capacity_available"] != true || capacity["remaining_page_slots"] != 1 {
		t.Fatalf("expected recovered current capacity: %#v", capacity)
	}
	if history["queue_rejected"] != 14 || history["queue_p95_wait_ms"] != 24590 || history["fail_count"] != int64(7) {
		t.Fatalf("historical overload must remain visible: %#v", history)
	}
}

func TestAgentPressureSummaryCurrentQueueIsConstrained(t *testing.T) {
	pressure := agentPressureSummary(map[string]any{
		"active_pages":    0,
		"available_pages": 0,
		"max_pages":       2,
		"fail_count":      int64(0),
		"queue":           map[string]any{"depth": 1, "rejected": 0, "p95_wait_ms": 0},
	}, nil)
	browser := pressure["browser"].(map[string]any)
	if got, want := browser["pressure"], "constrained"; got != want {
		t.Fatalf("browser pressure=%v want=%v: %#v", got, want, pressure)
	}
}

func TestAgentPressureSummaryClassifiesSaturatedBrowser(t *testing.T) {
	pressure := agentPressureSummary(map[string]any{
		"active_pages":    2,
		"available_pages": 0,
		"max_pages":       2,
		"fail_count":      int64(0),
		"queue": map[string]any{
			"depth":       1,
			"rejected":    1,
			"p95_wait_ms": 6000,
		},
		"cache": map[string]any{"status": "ready", "entries": 10, "max_entries": 10},
	}, nil)
	browser := pressure["browser"].(map[string]any)
	if browser["pressure"] != "saturated" || pressure["overall_pressure"] != "saturated" {
		t.Fatalf("expected saturated pressure: %#v", pressure)
	}
	actions := pressure["recommended_actions"].([]string)
	found := false
	for _, action := range actions {
		if action == "Close unused browser sessions, reduce concurrent page work, and prefer browser_read/diagnostics over screenshots." {
			found = true
		}
	}
	if !found {
		t.Fatalf("missing browser recovery action: %#v", actions)
	}
}
