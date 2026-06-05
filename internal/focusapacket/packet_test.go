package focusapacket

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSanitizeURLRedactsSecretQueryAndStripsFragment(t *testing.T) {
	got := SanitizeURL("https://example.com/path?token=abc&ok=yes&api_key=123&signature=sig#frag")
	if strings.Contains(got, "abc") || strings.Contains(got, "123") || strings.Contains(got, "sig#") || strings.Contains(got, "#frag") {
		t.Fatalf("secret or fragment leaked: %s", got)
	}
	for _, want := range []string{"token=REDACTED", "api_key=REDACTED", "signature=REDACTED", "ok=yes"} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %s in %s", want, got)
		}
	}
}

func TestNormalizeEnforcesSchemaLimitsAndRedaction(t *testing.T) {
	packet := ResearchDiagnosticsPacket{
		Mode:         "unknown",
		Goal:         strings.Repeat("g", MaxGoalChars+20),
		ScopeStatus:  "bad",
		TargetRefs:   []string{"browser:https://example.com/page?session=abc&plain=ok#frag"},
		EvidenceRefs: []string{"uiai-search:brave:hash:1"},
		Captures: []Capture{{
			Type:        "read",
			EvidenceRef: "uiai-browser:session=s1:read:1",
			TargetRef:   "https://example.com/a?jwt=secret#nope",
			Summary:     strings.Repeat("s", MaxCaptureSummaryChars+30),
		}},
		DiagnosticsSummary: &DiagnosticsSummary{ConsoleErrors: 1, FailedRequests: 2, TopFindings: []string{strings.Repeat("f", 400)}},
		ActiveObjectHints:  []ActiveObjectHint{{Kind: "url", Hint: "https://example.com/x?code=abc#frag"}},
		RecommendedFocusa: RecommendedFocusa{
			PreferredTool: "focusa_evidence_capture",
			ArgsPreview: map[string]any{
				"target_ref": "https://example.com/z?password=pw#frag",
				"token":      "must drop",
				"result":     strings.Repeat("r", 600),
			},
		},
		RecommendedNextAction: strings.Repeat("n", MaxRecommendedNextActionChars+20),
	}

	normalized := Normalize(packet)
	data, err := json.Marshal(normalized)
	if err != nil {
		t.Fatal(err)
	}
	encoded := string(data)
	if len(data) > DefaultMaxPacketBytes {
		t.Fatalf("packet over budget: %d", len(data))
	}
	if normalized.Schema != SchemaResearchDiagnosticsPacketV1 {
		t.Fatalf("schema mismatch: %s", normalized.Schema)
	}
	if normalized.Mode != ModeResearch || normalized.ScopeStatus != ScopeMissing {
		t.Fatalf("unexpected normalized mode/scope: %s/%s", normalized.Mode, normalized.ScopeStatus)
	}
	if len([]rune(normalized.Goal)) > MaxGoalChars || len([]rune(normalized.Captures[0].Summary)) > MaxCaptureSummaryChars {
		t.Fatalf("text fields not truncated")
	}
	for _, leaked := range []string{"secret", "must drop", "password=pw", "session=abc", "jwt=secret", "code=abc", "#frag", "#nope"} {
		if strings.Contains(encoded, leaked) {
			t.Fatalf("leaked %q in %s", leaked, encoded)
		}
	}
}

func TestEnforceBudgetTrimsOversizeCaptures(t *testing.T) {
	packet := ResearchDiagnosticsPacket{
		Schema:      SchemaResearchDiagnosticsPacketV1,
		Mode:        ModeResearch,
		Goal:        "budget test",
		ScopeStatus: ScopePresent,
		TargetRefs:  []string{"browser:https://example.com"},
		Captures: []Capture{
			{Type: "read", EvidenceRef: "one", TargetRef: "browser:https://example.com/1", Summary: strings.Repeat("a", MaxCaptureSummaryChars)},
			{Type: "read", EvidenceRef: "two", TargetRef: "browser:https://example.com/2", Summary: strings.Repeat("b", MaxCaptureSummaryChars)},
			{Type: "read", EvidenceRef: "three", TargetRef: "browser:https://example.com/3", Summary: strings.Repeat("c", MaxCaptureSummaryChars)},
		},
		RecommendedFocusa: RecommendedFocusa{PreferredTool: "focusa_evidence_capture"},
		Render:            RenderInfo{SummaryLine: strings.Repeat("render", 100)},
	}

	trimmed := EnforceBudget(packet, 900)
	data, err := json.Marshal(trimmed)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > 900 {
		t.Fatalf("packet over custom budget: %d", len(data))
	}
	if len(trimmed.Captures) >= len(packet.Captures) {
		t.Fatalf("expected captures trimmed, got %d", len(trimmed.Captures))
	}
}
