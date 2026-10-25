package observability

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSemanticTextRedactedBeforeErrorPersistence(t *testing.T) {
	const canary = "UIAI_SYNTHETIC_ERROR_TOKEN_0123456789"
	store := &ErrorStore{}
	stored := store.Record(ErrorEvent{Message: "API key\n" + canary, Context: map[string]any{"details": "Bearer token\n" + canary}, SuggestedNextAction: "token=" + canary})
	for _, value := range []any{stored, store.Recent(10, "", "")} {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), canary) {
			t.Fatal("synthetic token persisted into error projection")
		}
	}
}

func TestEmbeddedOAuthURLRedactedBeforeErrorPersistence(t *testing.T) {
	const canary = "UIAI_SYNTHETIC_LOGGED_CALLBACK_0123456789"
	input := "navigation failed https://example.test/callback?code=" + canary
	store := &ErrorStore{}
	event := store.Record(ErrorEvent{Message: input, Context: map[string]any{"detail": input}})
	encoded, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), canary) {
		t.Fatal("embedded callback persisted before URL sanitization")
	}
}
