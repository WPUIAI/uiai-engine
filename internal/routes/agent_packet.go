package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/focusapacket"
	"github.com/go-chi/chi/v5"
)

type researchPacketRequest struct {
	Mode                  string                    `json:"mode"`
	Goal                  string                    `json:"goal"`
	Responses             []map[string]any          `json:"responses"`
	FocusaScope           *focusapacket.FocusaScope `json:"focusa_scope"`
	RecommendedNextAction string                    `json:"recommended_next_action"`
	CleanupSessionID      string                    `json:"cleanup_session_id"`
	ExpandableJSONRef     string                    `json:"expandable_json_ref"`
}

// MountAgentPacketRoutes composes bounded agent packet proposals from existing UIAI response metadata.
// Schema: uiai.focusa_research_diagnostics_packet.v1
func MountAgentPacketRoutes(r chi.Router, cfg *config.Config) {
	r.Post("/research-packet", func(w http.ResponseWriter, req *http.Request) {
		var body researchPacketRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "invalid_json", "message": err.Error()})
			return
		}
		if strings.TrimSpace(body.Goal) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]any{"error": "goal_required", "message": "goal is required"})
			return
		}
		packet := buildResearchPacketFromResponses(body)
		scope := evidenceScopeFromRequest(req)
		if body.FocusaScope != nil {
			if scope.ProjectRef == "" {
				scope.ProjectRef = body.FocusaScope.ProjectRef
				if scope.ProjectRef == "" {
					scope.ProjectRef = body.FocusaScope.ProjectRoot
				}
			}
			if scope.WorkstreamRef == "" {
				scope.WorkstreamRef = body.FocusaScope.WorkstreamRef
			}
			if scope.WorksetRef == "" {
				scope.WorksetRef = body.FocusaScope.WorksetRef
			}
			if scope.CallGraphRef == "" {
				scope.CallGraphRef = body.FocusaScope.CallGraphRef
			}
			if scope.WorkpointRef == "" {
				scope.WorkpointRef = body.FocusaScope.WorkpointID
			}
			if scope.WorkItemRef == "" {
				scope.WorkItemRef = body.FocusaScope.WorkItemRef
			}
			if len(scope.WorkItems) == 0 {
				scope.WorkItems = append(scope.WorkItems, body.FocusaScope.WorkItems...)
			}
			if scope.ContinuityRef == "" {
				scope.ContinuityRef = body.FocusaScope.ContinuityID
			}
		}
		writeJSONArtifactEPWA(w, req, cfg, scope, "", "Focusa research and diagnostics packet", "research_packet", packet, http.StatusOK)
	})
}

func buildResearchPacketFromResponses(req researchPacketRequest) focusapacket.ResearchDiagnosticsPacket {
	captures := make([]focusapacket.Capture, 0, len(req.Responses))
	for _, response := range req.Responses {
		if capture, ok := captureFromFocusaResponse(response); ok {
			captures = append(captures, capture)
		}
	}
	targetRefs := make([]string, 0, len(captures))
	evidenceRefs := make([]string, 0, len(captures))
	for _, capture := range captures {
		if capture.TargetRef != "" {
			targetRefs = append(targetRefs, capture.TargetRef)
		}
		if capture.EvidenceRef != "" {
			evidenceRefs = append(evidenceRefs, capture.EvidenceRef)
		}
	}
	preferredTool := packetPreferredFocusaTool(captures)
	nextAction := req.RecommendedNextAction
	if strings.TrimSpace(nextAction) == "" {
		nextAction = "Call focusa_evidence_capture or focusa_browser_diagnostics_intake with recommended_focusa.args_preview."
	}
	packetScope := packetVisibleScope(req.FocusaScope)
	packet := focusapacket.ResearchDiagnosticsPacket{
		Schema:             focusapacket.SchemaResearchDiagnosticsPacketV1,
		Mode:               packetMode(req.Mode),
		Goal:               req.Goal,
		ScopeStatus:        packetScopeStatus(packetScope),
		FocusaScope:        packetScope,
		TargetRefs:         targetRefs,
		EvidenceRefs:       evidenceRefs,
		Captures:           captures,
		DiagnosticsSummary: packetDiagnosticsSummary(req.Responses),
		ActiveObjectHints:  packetActiveObjectHints(targetRefs),
		RecommendedFocusa: focusapacket.RecommendedFocusa{
			PreferredTool: preferredTool,
			FallbackTool:  "focusa_evidence_capture",
			ArgsPreview:   packetArgsPreview(captures, targetRefs, evidenceRefs, req.Goal),
			NextTools:     packetNextTools(preferredTool),
		},
		RecommendedNextAction: nextAction,
		Render: focusapacket.RenderInfo{
			SummaryLine:       fmt.Sprintf("UIAI packet evidence=%d scope=%s tool=%s next=%s", len(evidenceRefs), packetScopeStatus(req.FocusaScope), preferredTool, nextAction),
			ExpandableJSONRef: req.ExpandableJSONRef,
		},
		HeadlessNextAction: nextAction,
	}
	if strings.TrimSpace(req.CleanupSessionID) != "" {
		packet.Cleanup = &focusapacket.Cleanup{SessionID: req.CleanupSessionID, RecommendedAction: "close_when_done", Tool: "uiai_browser_close"}
	}
	return focusapacket.Normalize(packet)
}

func captureFromFocusaResponse(response map[string]any) (focusapacket.Capture, bool) {
	focusa, ok := response["focusa"].(map[string]any)
	if !ok {
		focusa, ok = response["focusa_evidence"].(map[string]any)
	}
	if !ok || len(focusa) == 0 {
		return focusapacket.Capture{}, false
	}
	evidenceRef := stringValue(focusa["evidence_ref"])
	targetRef := sanitizePacketTargetRef(stringValue(focusa["target_ref"]))
	typ := packetCaptureType(evidenceRef)
	title := stringValue(response["title"])
	if title == "" {
		if results, ok := response["results"].([]any); ok && len(results) > 0 {
			if first, ok := results[0].(map[string]any); ok {
				title = stringValue(first["title"])
			}
		}
	}
	return focusapacket.Capture{Type: typ, EvidenceRef: evidenceRef, TargetRef: targetRef, Title: title, Summary: firstString([]string{stringValue(focusa["summary"]), stringValue(focusa["result"]), stringValue(response["message"])}, "UIAI evidence capture")}, true
}

func sanitizePacketTargetRef(target string) string {
	if strings.HasPrefix(target, "source-markdown:") {
		return "source-markdown:" + sanitizeMarkdownURLForFocusa(strings.TrimPrefix(target, "source-markdown:"))
	}
	return target
}

func packetCaptureType(evidenceRef string) string {
	switch {
	case strings.Contains(evidenceRef, "diagnostics"):
		return "diagnostics"
	case strings.Contains(evidenceRef, "uiai-error"):
		return "error"
	case strings.Contains(evidenceRef, "uiai-source-markdown"):
		return "source_markdown"
	case strings.Contains(evidenceRef, ":read:"):
		return "read"
	case strings.Contains(evidenceRef, ":snapshot:"):
		return "snapshot"
	case strings.Contains(evidenceRef, "uiai-screenshot"):
		return "screenshot"
	case strings.Contains(evidenceRef, "uiai-share"):
		return "share"
	default:
		return "search"
	}
}

func packetMode(mode string) focusapacket.PacketMode {
	switch mode {
	case string(focusapacket.ModeDiagnose):
		return focusapacket.ModeDiagnose
	case string(focusapacket.ModeProof):
		return focusapacket.ModeProof
	default:
		return focusapacket.ModeResearch
	}
}

func packetVisibleScope(scope *focusapacket.FocusaScope) *focusapacket.FocusaScope {
	if scope == nil {
		return nil
	}
	visible := *scope
	visible.WorkItems = nil
	if visible.ProjectRef != "" {
		visible.ProjectRoot = ""
	}
	return &visible
}

func packetScopeStatus(scope *focusapacket.FocusaScope) focusapacket.ScopeStatus {
	if scope == nil {
		return focusapacket.ScopeMissing
	}
	if (scope.ProjectRef != "" || scope.ProjectRoot != "") && scope.ContinuityID != "" {
		return focusapacket.ScopePresent
	}
	return focusapacket.ScopePartial
}

func packetPreferredFocusaTool(captures []focusapacket.Capture) string {
	for _, capture := range captures {
		if capture.Type == "diagnostics" || capture.Type == "error" {
			return "focusa_browser_diagnostics_intake"
		}
	}
	return "focusa_evidence_capture"
}

func packetNextTools(preferred string) []string {
	if preferred == "focusa_browser_diagnostics_intake" {
		return []string{"focusa_browser_diagnostics_intake", "focusa_evidence_capture", "focusa_active_object_resolve", "focusa_predict_record"}
	}
	return []string{"focusa_evidence_capture", "focusa_active_object_resolve", "focusa_predict_record"}
}

func packetArgsPreview(captures []focusapacket.Capture, targetRefs, evidenceRefs []string, goal string) map[string]any {
	result := goal
	if len(captures) > 0 && captures[0].Summary != "" {
		result = captures[0].Summary
	}
	return map[string]any{"target_ref": firstString(targetRefs, "uiai:packet"), "result": result, "evidence_ref": firstString(evidenceRefs, "uiai-packet:manual"), "attach_to_workpoint": false}
}

func packetActiveObjectHints(targetRefs []string) []focusapacket.ActiveObjectHint {
	hints := make([]focusapacket.ActiveObjectHint, 0, len(targetRefs))
	for _, target := range targetRefs {
		kind := "source"
		if strings.HasPrefix(target, "browser:") {
			kind = "url"
		}
		hints = append(hints, focusapacket.ActiveObjectHint{Kind: kind, Hint: target})
	}
	return hints
}

func packetDiagnosticsSummary(responses []map[string]any) *focusapacket.DiagnosticsSummary {
	var out focusapacket.DiagnosticsSummary
	for _, response := range responses {
		summary, ok := response["summary"].(map[string]any)
		if !ok {
			summary, _ = response["diagnostics_summary"].(map[string]any)
		}
		if len(summary) == 0 {
			continue
		}
		out.ConsoleErrors += intValue(summary["console_errors"])
		out.FailedRequests += intValue(summary["failed_requests"])
		if exceptions := intValue(summary["exceptions"]); exceptions > 0 {
			out.TopFindings = append(out.TopFindings, fmt.Sprintf("exceptions=%d", exceptions))
		}
		if http4xx := intValue(summary["http_4xx"]); http4xx > 0 {
			out.TopFindings = append(out.TopFindings, fmt.Sprintf("http_4xx=%d", http4xx))
		}
		if http5xx := intValue(summary["http_5xx"]); http5xx > 0 {
			out.TopFindings = append(out.TopFindings, fmt.Sprintf("http_5xx=%d", http5xx))
		}
	}
	if out.ConsoleErrors == 0 && out.FailedRequests == 0 && len(out.TopFindings) == 0 {
		return nil
	}
	return &out
}

func stringValue(value any) string {
	if s, ok := value.(string); ok {
		return s
	}
	return ""
}

func intValue(value any) int {
	switch v := value.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	default:
		return 0
	}
}

func firstString(values []string, fallback string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return fallback
}
