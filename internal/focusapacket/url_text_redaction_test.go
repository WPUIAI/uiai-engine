package focusapacket

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestOAuthURLsEmbeddedInTelemetry(t *testing.T) {
	const canary = "UIAI_SYNTHETIC_CALLBACK_0123456789"
	for _, input := range []string{
		"navigation failed https://example.test/callback?code=" + canary + "&ok=1",
		"request https://example.test/callback?%63ode=" + canary + "&CODE=" + canary,
		"redirect https://example.test/#access_token=" + canary,
		"nested https://example.test/?redirect_uri=https%3A%2F%2Fexample.test%2Fcb%3Fcode%3D" + canary,
	} {
		out := SanitizeText(input)
		if strings.Contains(out, canary) {
			t.Fatal("synthetic OAuth value escaped embedded URL sanitization")
		}
		if !strings.Contains(out, "https://example.test/") {
			t.Fatal("useful origin/path lost")
		}
	}
}

func TestPacketFreeTextUsesURLSanitizerBeforeSerialization(t *testing.T) {
	const canary = "UIAI_SYNTHETIC_PACKET_CALLBACK_0123456789"
	input := "Inspect https://example.test/callback?code=" + canary
	packet := Normalize(ResearchDiagnosticsPacket{Goal: input, RecommendedNextAction: input, HeadlessNextAction: input, Render: RenderInfo{SummaryLine: input}})
	encoded, err := json.Marshal(packet)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), canary) {
		t.Fatal("callback escaped packet free-text boundary")
	}
}
