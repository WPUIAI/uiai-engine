package secrettext

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestFramedProjectionRetainsJSONRolesAndWholeProjectionContract(t *testing.T) {
	const canary = "UIAI_SYNTHETIC_FRAMED_0123456789"
	raw := `Uncaught {"password":["` + canary + `"],"echo":"` + canary + `"}` + "\n at " + canary
	prefix, value, suffix := DecodeFramedProjection(raw)
	if prefix != "Uncaught " || suffix != "\n at "+canary {
		t.Fatal("diagnostic framing changed")
	}
	if _, ok := value.(map[string]any); !ok {
		t.Fatal("JSON credential roles lost")
	}
	clean := Value([]any{prefix, value, suffix})
	encoded, err := json.Marshal(clean)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), canary) {
		t.Fatal("framed projection alias survived classification")
	}
	if got := DecodeProjection(raw); got != raw {
		t.Fatal("whole-JSON decoder accepted a framed message")
	}
	for _, ordinary := range []string{"ordinary exception", "Error: [invalid JSON]", `{"incomplete":`} {
		before, projection, after := DecodeFramedProjection(ordinary)
		if before != "" || after != "" || projection != ordinary {
			t.Fatal("malformed or ordinary text was rewritten")
		}
	}
}
