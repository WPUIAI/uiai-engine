package routes

import (
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/vision"
)

func TestBrowserHealthPayloadUnavailableWhenPoolNil(t *testing.T) {
	payload := browserHealthPayload(nil)
	if got, want := payload["status"], "unavailable"; got != want {
		t.Fatalf("status=%v want=%v", got, want)
	}
	if got, want := browserHealthStatus(nil), 503; got != want {
		t.Fatalf("status code=%d want=%d", got, want)
	}
}

func TestBrowserHealthStandbyWhenPoolLazyIdle(t *testing.T) {
	pool := &vision.Pool{}
	payload := browserHealthPayload(pool)
	if got, want := payload["status"], "standby"; got != want {
		t.Fatalf("status=%v want=%v", got, want)
	}
	if got, want := browserHealthStatus(pool), 200; got != want {
		t.Fatalf("status code=%d want=%d", got, want)
	}
}

func TestBrowserHealthPayloadIncludesAgentPressure(t *testing.T) {
	pool := &vision.Pool{}
	payload := browserHealthPayload(pool)
	pressure, ok := payload["agent_pressure"].(map[string]any)
	if !ok {
		t.Fatalf("expected agent_pressure map: %#v", payload)
	}
	packet := pressure["packet"].(map[string]any)
	if packet["schema"] != "uiai.focusa_research_diagnostics_packet.v1" {
		t.Fatalf("missing packet pressure schema: %#v", packet)
	}
	search := pressure["search"].(map[string]any)
	if search["packet_surface"] != "search" {
		t.Fatalf("missing search pressure surface: %#v", search)
	}
	browser := pressure["browser"].(map[string]any)
	if browser["pressure"] == "" {
		t.Fatalf("missing browser pressure: %#v", browser)
	}
	if pressure["recommended_action"] == "" {
		t.Fatalf("missing recommended action: %#v", pressure)
	}
}
