package routes

import (
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

func TestBrowserHealthPreservesRSSMeasurementPosture(t *testing.T) {
	for _, state := range []string{"pending", "unavailable", "sampled", "disabled", "unsupported"} {
		t.Run(state, func(t *testing.T) {
			active := state == "sampled"
			summary := agentPressureSummary(map[string]any{"max_pages": 2, "rss_pressure": map[string]any{"measurement": state, "protection_active": active}}, nil, &config.Config{})
			browser := summary["browser"].(map[string]any)
			rss := browser["rss_pressure"].(map[string]any)
			if rss["measurement"] != state || rss["protection_active"] != active {
				t.Fatal("browser health changed the memory-protection posture")
			}
		})
	}
	summary := agentPressureSummary(map[string]any{"max_pages": 2}, nil, &config.Config{})
	rss := summary["browser"].(map[string]any)["rss_pressure"].(map[string]any)
	if rss["measurement"] != "unavailable" || rss["protection_active"] != false {
		t.Fatal("missing memory observations were treated as active protection")
	}
}
