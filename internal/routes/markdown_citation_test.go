package routes

import (
	"strings"
	"testing"
)

func TestMarkdownCitationRedactionUsesCanonicalBoundaries(t *testing.T) {
	raw := "[one](https://dummy-user:dummy-pass@one.test/?token=SYNTHETIC_TOKEN#SYNTHETIC_FRAGMENT)[two](https://two.test/path_(balanced)#PRIVATE_FRAGMENT)"
	want := "[one](https://one.test/?token=REDACTED)[two](https://two.test/path_(balanced))"
	if got := sanitizeMarkdownText(raw); got != want {
		t.Fatalf("Markdown URL boundaries corrupted: %q", got)
	}
	nested := map[string]any{"markdown": raw, "nested": []any{raw}}
	if strings.Contains(sanitizeMarkdownPayload(nested).(map[string]any)["markdown"].(string), "SYNTHETIC") {
		t.Fatal("Markdown payload lost URL protection")
	}
}
