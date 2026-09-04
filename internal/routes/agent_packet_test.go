package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/focusapacket"
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

func TestBuildResearchPacketFromSourceMarkdownResponse(t *testing.T) {
	packet := buildResearchPacketFromResponses(researchPacketRequest{
		Goal: "source markdown packet",
		Responses: []map[string]any{{
			"title": "Example Source",
			"focusa": map[string]any{
				"target_ref":   "source-markdown:https://example.test/path?token=secret#frag",
				"evidence_ref": "uiai-source-markdown:sha256:abcdef1234567890",
				"summary":      "Converted Example Source to Markdown",
			},
		}},
	})
	if len(packet.Captures) != 1 || packet.Captures[0].Type != "source_markdown" {
		t.Fatalf("unexpected captures: %+v", packet.Captures)
	}
	if packet.RecommendedFocusa.PreferredTool != "focusa_evidence_capture" {
		t.Fatalf("unexpected preferred tool: %+v", packet.RecommendedFocusa)
	}
	data, _ := json.Marshal(packet)
	if strings.Contains(string(data), "secret") || strings.Contains(string(data), "#frag") {
		t.Fatalf("packet leaked secret/fragment: %s", string(data))
	}
}

func TestAgentResearchPacketEndpointUsesCompleteBodyScope(t *testing.T) {
	r := chi.NewRouter()
	MountAgentPacketRoutes(r, &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}})
	scope := completeEvidenceScope()
	body, err := json.Marshal(researchPacketRequest{
		Goal:      "body-scoped endpoint packet",
		Responses: []map[string]any{{"focusa": map[string]any{"target_ref": "browser:https://example.test", "evidence_ref": "uiai-search:brave:body:1", "summary": "Search result"}}},
		FocusaScope: &focusapacket.FocusaScope{
			ProjectRef: scope.ProjectRef, ProjectRoot: "/private/source/root", WorkstreamRef: scope.WorkstreamRef,
			WorksetRef: scope.WorksetRef, CallGraphRef: scope.CallGraphRef, WorkpointID: scope.WorkpointRef,
			WorkItemRef: scope.WorkItemRef, WorkItems: scope.WorkItems, ContinuityID: scope.ContinuityRef,
			EvidenceRef: "uiai-agent-packet-body-scope",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "https://evidence.example/research-packet", strings.NewReader(string(body)))
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var packet map[string]any
	if err := json.NewDecoder(res.Body).Decode(&packet); err != nil {
		t.Fatal(err)
	}
	if packet["schema"] != "uiai.artifact_result.v2" || packet["artifact_schema"] != "uiai.focusa_research_diagnostics_packet.v1" || packet["delivery_state"] != "ready" {
		t.Fatalf("unexpected packet delivery envelope: %#v", packet)
	}
	delivery := packet["epwa_delivery"].(map[string]any)
	deliveryScope := delivery["scope"].(map[string]any)
	for key, want := range map[string]string{
		"project_ref": scope.ProjectRef, "workstream_ref": scope.WorkstreamRef, "workset_ref": scope.WorksetRef,
		"callgraph_ref": scope.CallGraphRef, "workpoint_ref": scope.WorkpointRef, "work_item_ref": scope.WorkItemRef,
		"continuity_ref": scope.ContinuityRef,
	} {
		if deliveryScope[key] != want {
			t.Fatalf("delivery scope %s=%v want %q: %#v", key, deliveryScope[key], want, deliveryScope)
		}
	}
	packetScope := packet["focusa_scope"].(map[string]any)
	if packetScope["project_ref"] != scope.ProjectRef || packetScope["project_root"] != nil || len(packetScope["work_items"].([]any)) != len(scope.WorkItems) {
		t.Fatalf("packet scope was not public-safe and complete: %#v", packetScope)
	}
}

func TestAgentResearchPacketEndpoint(t *testing.T) {
	r := chi.NewRouter()
	MountAgentPacketRoutes(r, &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}})
	body := `{"goal":"endpoint packet","responses":[{"focusa":{"target_ref":"browser:https://example.test","evidence_ref":"uiai-search:brave:abc:1","summary":"Search result"}}]}`
	req := httptest.NewRequest(http.MethodPost, "https://evidence.example/research-packet", strings.NewReader(body))
	scope := completeEvidenceScope()
	req.Header.Set("X-UIAI-Project-Ref", scope.ProjectRef)
	req.Header.Set("X-UIAI-Workstream-Ref", scope.WorkstreamRef)
	req.Header.Set("X-UIAI-Workset-Ref", scope.WorksetRef)
	req.Header.Set("X-UIAI-CallGraph-Ref", scope.CallGraphRef)
	req.Header.Set("X-UIAI-Workpoint-Ref", scope.WorkpointRef)
	req.Header.Set("X-UIAI-Work-Item-Ref", scope.WorkItemRef)
	req.Header.Set("X-UIAI-Continuity-Ref", scope.ContinuityRef)
	workItems, _ := json.Marshal(scope.WorkItems)
	req.Header.Set("X-UIAI-Work-Items", string(workItems))
	res := httptest.NewRecorder()
	r.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	var packet map[string]any
	if err := json.NewDecoder(res.Body).Decode(&packet); err != nil {
		t.Fatal(err)
	}
	if packet["schema"] != "uiai.artifact_result.v2" || packet["artifact_schema"] != "uiai.focusa_research_diagnostics_packet.v1" || packet["delivery_state"] != "ready" {
		t.Fatalf("unexpected packet delivery envelope: %#v", packet)
	}
	captures := packet["captures"].([]any)
	if len(captures) != 1 {
		t.Fatalf("captures=%#v", captures)
	}
}
