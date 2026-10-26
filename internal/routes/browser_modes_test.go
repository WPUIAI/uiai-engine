package routes

import (
	"encoding/json"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/go-chi/chi/v5"
)

func TestBrowserModesSharedAcrossDiscoverySurfaces(t *testing.T) {
	router := chi.NewRouter()
	MountToolsDiscovery(router, &config.Config{})
	var previous any
	for _, path := range []string{"/agent-card", "/docs"} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("GET", path, nil))
		if response.Code != 200 {
			t.Fatal(response.Code)
		}
		var payload map[string]any
		if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
			t.Fatal(err)
		}
		card, ok := payload["browser_modes"].(map[string]any)
		if !ok {
			t.Fatalf("%s missing browser mode decision card", path)
		}
		if card["schema"] != "uiai.browser_modes.v1" || card["grants_authority"] != false || card["performs_upstream_probes"] != false {
			t.Fatal("mode card lacks discovery-only boundary")
		}
		if previous != nil && !reflect.DeepEqual(previous, card) {
			t.Fatal("discovery surfaces diverged from canonical mode card")
		}
		previous = card
	}
}
