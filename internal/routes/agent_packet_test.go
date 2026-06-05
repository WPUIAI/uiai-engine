package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

func TestBuildResearchPacketFromResponses(t *testing.T) {
	packet := buildResearchPacketFromResponses(researchPacketRequest{
		Mode: "proof",
		Goal: "prove packet",
		Responses: []map[string]any{
			{"focusa": map[string]any{"target_ref": "browser:https://example.test/path?token=secret#frag", "evidence_ref": "uiai-browser:session=s:read:1", "summary": strings.Repeat("read ", 200)}},
			{"focusa": map[string]any{"target_ref": "browser:https://example.test/path", "evidence_ref": "uiai-diagnostics:session=s:seq=2", "summary": "diag"}, "summary": map[string]any{"console_errors": float64(1), "failed_requests": float64(2), "exceptions": float64(3)}},
		},
		CleanupSessionID: "s",
	})
	if packet.Schema != "uiai.focusa_research_diagnostics_packet.v1" || packet.Mode != "proof" {
		t.Fatalf("unexpected packet identity: %+v", packet)
	}
	if len(packet.Captures) != 2 || packet.Captures[0].Type != "read" || packet.Captures[1].Type != "diagnostics" {
		t.Fatalf("unexpected captures: %+v", packet.Captures)
	}
	if packet.RecommendedFocusa.PreferredTool != "focusa_browser_diagnostics_intake" {
		t.Fatalf("unexpected preferred tool: %+v", packet.RecommendedFocusa)
	}
	if packet.DiagnosticsSummary == nil || packet.DiagnosticsSummary.ConsoleErrors != 1 || packet.DiagnosticsSummary.FailedRequests != 2 {
		t.Fatalf("missing diagnostics summary: %+v", packet.DiagnosticsSummary)
	}
	if packet.Cleanup == nil || packet.Cleanup.Tool != "uiai_browser_close" {
		t.Fatalf("missing cleanup: %+v", packet.Cleanup)
	}
	data, _ := json.Marshal(packet)
	if len(data) > 8192 {
		t.Fatalf("packet too large: %d", len(data))
	}
	if strings.Contains(string(data), "secret") || strings.Contains(string(data), "#frag") {
		t.Fatalf("packet leaked secret/fragment: %s", string(data))
	}
}

func TestAgentResearchPacketEndpoint(t *testing.T) {
	r := chi.NewRouter()
	MountAgentPacketRoutes(r)
	body := `{"goal":"endpoint packet","responses":[{"focusa":{"target_ref":"browser:https://example.test","evidence_ref":"uiai-search:brave:abc:1","summary":"Search result"}}]}`
	req := httptest.NewRequest(http.MethodPost, "/research-packet", strings.NewReader(body))
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var packet map[string]any
	if err := json.NewDecoder(res.Body).Decode(&packet); err != nil {
		t.Fatal(err)
	}
	if packet["schema"] != "uiai.focusa_research_diagnostics_packet.v1" {
		t.Fatalf("unexpected packet: %#v", packet)
	}
	captures := packet["captures"].([]any)
	if len(captures) != 1 {
		t.Fatalf("captures=%#v", captures)
	}
}
