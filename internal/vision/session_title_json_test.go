package vision

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/secrettext"
)

func TestEscapedStructuredTitleIsWithheldAcrossMetadataFields(t *testing.T) {
	title := `{"password":"UIAI\u005fSYNTHETIC\u005fESCAPED\u005fTITLE","echo":"UIAI_SYNTHETIC_ESCAPED_TITLE"}`
	clean := redactPageFields(map[string]any{"title": title, "fields": []any{title, "ordinary field"}})
	if clean["title"] != secrettext.Marker || clean["fields"].([]any)[0] != secrettext.Marker {
		t.Fatal("encoded structured credential survived title or identical metadata alias")
	}
	if clean["fields"].([]any)[1] != "ordinary field" {
		t.Fatal("ordinary unrelated metadata changed")
	}
}

func TestPageTitleJSONCredentialsRedactedBeforeMetadata(t *testing.T) {
	const canary = "UIAI_SYNTHETIC_TITLE_0123456789"
	for _, title := range []string{
		`{"password":"` + canary + `","echo":"` + canary + `"}`,
		`Status {"password":["` + canary + `"],"echo":"` + canary + `"} suffix`,
		`{"echo":"` + canary + `","padding":"` + strings.Repeat("x", 1400) + `","password":"` + canary + `"}`,
	} {
		raw, err := json.Marshal(map[string]any{"title": title, "body": "ordinary body", "controls": []any{}})
		if err != nil {
			t.Fatal(err)
		}
		clean, outputs := redactPageProjection(string(raw), canary)
		encoded, err := json.Marshal([]any{clean, outputs})
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), canary) {
			t.Fatal("synthetic credential escaped title or output alias")
		}
		if !strings.Contains(clean, "REDACTED_SECRET") || len(clean) > 1003 {
			t.Fatal("title redaction marker or existing bound lost")
		}
	}
	raw := `{"title":"Ordinary page title","body":"ordinary body","controls":[]}`
	title, outputs := redactPageProjection(raw, "ordinary result")
	if title != "Ordinary page title" || len(outputs) != 1 || outputs[0] != "ordinary result" {
		t.Fatal("ordinary page metadata changed")
	}
	title, outputs = redactPageProjection("not JSON", "ordinary result")
	if !strings.Contains(title, "REDACTED_SECRET") || !strings.Contains(outputs[0], "REDACTED_SECRET") {
		t.Fatal("unavailable page privacy context did not fail closed")
	}
}
