package focusapacket

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSemanticTextRedactedBeforePacketProjection(t *testing.T) {
	const canary = "UIAI_SYNTHETIC_PACKET_TOKEN_0123456789"
	captures := sanitizeCaptures([]Capture{{Type: "read", Title: "API key " + canary, Summary: "Bearer token\n" + canary}})
	summary := sanitizeDiagnosticsSummary(&DiagnosticsSummary{TopFindings: []string{"API key\n" + canary}})
	for _, value := range []any{captures, summary} {
		encoded, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), canary) {
			t.Fatal("synthetic token escaped packet projection")
		}
	}
}
