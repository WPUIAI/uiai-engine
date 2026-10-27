package secrettext

import "testing"

func TestClassifiedStringAliasesAreWhollyWithheldBeforeInnerReplacement(t *testing.T) {
	const canary = "UIAI_SYNTHETIC_ENCODED_CONTAINER"
	composite := `{"password":"UIAI\u005fSYNTHETIC\u005fENCODED\u005fCONTAINER","echo":"` + canary + `"}`
	input := map[string]any{"secret": composite, "echo": composite, "context": map[string]any{"password": canary}, "ordinary": "ordinary text"}
	clean := Value(input).(map[string]any)
	if clean["secret"] != Marker || clean["echo"] != Marker {
		t.Fatal("a wholly classified string was only partially redacted")
	}
	if clean["ordinary"] != "ordinary text" || input["echo"] != composite {
		t.Fatal("ordinary value or original projection changed")
	}
}
