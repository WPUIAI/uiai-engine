package routes

import (
	"encoding/json"
	"testing"
)

func TestDeclaredPressureSchemaAcceptsRuntimeLevels(t *testing.T) {
	data, err := openapiEmbedFS.ReadFile("schemas_embed/uiai.agent_pressure.v1.schema.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Properties struct {
			Overall struct {
				Enum []string `json:"enum"`
			} `json:"overall_pressure"`
		} `json:"properties"`
	}
	if err = json.Unmarshal(data, &schema); err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{}
	for _, state := range schema.Properties.Overall.Enum {
		allowed[state] = true
	}
	runtimeLevels := []string{browserPressureLevel(0, 2, 0), browserPressureLevel(1, 2, 1), browserPressureLevel(2, 2, 0), maxPressureLevel("normal", "degraded")}
	for _, state := range runtimeLevels {
		if !allowed[state] {
			t.Errorf("runtime pressure level %q is rejected by its published schema", state)
		}
	}
}
