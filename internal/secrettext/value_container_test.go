package secrettext

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSensitiveContainersAndNumericAliasesAreWithheld(t *testing.T) {
	const canary = "UIAI_SYNTHETIC_CONTAINER_VALUE_0123456789"
	for name, input := range map[string]map[string]any{
		"array":       {"password": []any{canary}, "echo": canary},
		"object":      {"api_key": map[string]any{"primary": canary}, "echo": canary},
		"nested":      {"authorization": []any{map[string]any{"values": []any{canary}}}, "echo": canary},
		"numeric":     {"password": float64(934567891), "echo": float64(934567891), "string_echo": "934567891"},
		"integer":     {"password": int64(934567891), "echo": int64(934567891)},
		"json_number": {"password": json.Number("934567891"), "echo": json.Number("934567891")},
	} {
		t.Run(name, func(t *testing.T) {
			before, _ := json.Marshal(input)
			out := Value(input)
			encoded, err := json.Marshal(out)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(encoded), canary) || strings.Contains(string(encoded), "934567891") {
				t.Fatal("sensitive structured value or cross-field alias escaped redaction")
			}
			if !strings.Contains(string(encoded), Marker) {
				t.Fatal("missing redaction marker")
			}
			after, _ := json.Marshal(input)
			if string(before) != string(after) {
				t.Fatal("input projection was mutated")
			}
			repeated, _ := json.Marshal(Value(out))
			if string(repeated) != string(encoded) {
				t.Fatal("structured redaction is not idempotent")
			}
		})
	}
}

func TestOrdinaryStructuredValuesRemainTyped(t *testing.T) {
	input := map[string]any{"name": "public", "values": []any{"article", float64(42), true, nil}, "empty": map[string]any{}}
	before, _ := json.Marshal(input)
	after, _ := json.Marshal(Value(input))
	if string(before) != string(after) {
		t.Fatal("ordinary structured content was unnecessarily changed")
	}
}
