package routes

import (
	"encoding/json"
	"testing"
)

func TestBrowserModesRegistryBoundsAndSafety(t *testing.T) {
	for _, tools := range [][]map[string]any{nil, openAITools()} {
		card := browserModeDecisionCard(tools)
		encoded, err := json.Marshal(card)
		if err != nil {
			t.Fatal(err)
		}
		if len(encoded) > 8192 {
			t.Fatalf("mode card exceeds bounded bootstrap size: %d", len(encoded))
		}
		modes := card["modes"].([]map[string]any)
		ids := map[string]bool{}
		registered := map[string]bool{}
		for _, tool := range tools {
			registered[tool["name"].(string)] = true
		}
		for _, mode := range modes {
			id := mode["id"].(string)
			if ids[id] {
				t.Fatal("duplicate mode")
			}
			ids[id] = true
			if mode["status"] == "registered_tools" {
				for _, name := range mode["tools"].([]string) {
					if !registered[name] {
						t.Fatal("mode invented unregistered tool")
					}
				}
			}
			for _, field := range []string{"cost", "persistence", "security", "best_use"} {
				if mode[field] == "" || mode[field] == nil {
					t.Fatalf("mode %s lacks %s", id, field)
				}
			}
		}
		for _, rule := range card["state_routing"].([]map[string]string) {
			if !ids[rule["mode"]] {
				t.Fatal("routing references missing mode")
			}
		}
		security := card["security"].(map[string]any)
		for _, key := range []string{"recovery_code_automation", "nonrenewable_automation", "public_demo_credentials", "raw_secret_values_in_model"} {
			if security[key] != "forbidden" {
				t.Fatal("unsafe privileged-mode guidance")
			}
		}
		if security["unknown_renewability"] != "blocked" || security["transport_is_authority"] != false {
			t.Fatal("mode card minted authority")
		}
	}
}
