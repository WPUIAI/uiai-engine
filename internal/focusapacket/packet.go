package focusapacket

import (
	"encoding/json"
	"net/url"
	"strings"
)

const (
	SchemaResearchDiagnosticsPacketV1 = "uiai.focusa_research_diagnostics_packet.v1"
	DefaultMaxPacketBytes             = 8 * 1024
	MaxCaptureSummaryChars            = 500
	MaxGoalChars                      = 240
	MaxRecommendedNextActionChars     = 240
	MaxDiagnosticsSummaryBytes        = 2 * 1024
	MaxArgsPreviewBytes               = 2 * 1024
	MaxCaptures                       = 8
	MaxTargetRefs                     = 16
	MaxEvidenceRefs                   = 32
	MaxActiveObjectHints              = 16
)

var secretQueryKeys = []string{
	"token", "key", "secret", "auth", "session", "password", "passwd", "code", "sig", "signature", "jwt",
	"credential", "authorization", "cookie", "api_key", "apikey", "access_token", "refresh_token",
}

type PacketMode string

const (
	ModeResearch PacketMode = "research"
	ModeDiagnose PacketMode = "diagnose"
	ModeProof    PacketMode = "proof"
)

type ScopeStatus string

const (
	ScopePresent           ScopeStatus = "present"
	ScopeMissing           ScopeStatus = "missing"
	ScopePartial           ScopeStatus = "partial"
	ScopeMismatchCandidate ScopeStatus = "mismatch_candidate"
)

type FocusaScope struct {
	ProjectRoot  string `json:"project_root,omitempty"`
	ContinuityID string `json:"continuity_id,omitempty"`
	WorkpointID  string `json:"workpoint_id,omitempty"`
	EvidenceRef  string `json:"evidence_ref,omitempty"`
}

type Capture struct {
	Type        string `json:"type"`
	EvidenceRef string `json:"evidence_ref"`
	TargetRef   string `json:"target_ref"`
	Title       string `json:"title,omitempty"`
	Summary     string `json:"summary"`
}

type DiagnosticsSummary struct {
	ConsoleErrors  int      `json:"console_errors"`
	FailedRequests int      `json:"failed_requests"`
	TopFindings    []string `json:"top_findings"`
}

type ActiveObjectHint struct {
	Kind string `json:"kind"`
	Hint string `json:"hint"`
}

type RecommendedFocusa struct {
	PreferredTool string         `json:"preferred_tool"`
	FallbackTool  string         `json:"fallback_tool,omitempty"`
	ArgsPreview   map[string]any `json:"args_preview,omitempty"`
	NextTools     []string       `json:"next_tools,omitempty"`
}

type RenderInfo struct {
	SummaryLine       string `json:"summary_line"`
	ExpandableJSONRef string `json:"expandable_json_ref,omitempty"`
}

type Cleanup struct {
	SessionID         string `json:"session_id,omitempty"`
	RecommendedAction string `json:"recommended_action"`
	Tool              string `json:"tool,omitempty"`
}

type ResearchDiagnosticsPacket struct {
	Schema                string              `json:"schema"`
	Mode                  PacketMode          `json:"mode"`
	Goal                  string              `json:"goal"`
	ScopeStatus           ScopeStatus         `json:"scope_status"`
	FocusaScope           *FocusaScope        `json:"focusa_scope,omitempty"`
	TargetRefs            []string            `json:"target_refs"`
	EvidenceRefs          []string            `json:"evidence_refs"`
	Captures              []Capture           `json:"captures"`
	DiagnosticsSummary    *DiagnosticsSummary `json:"diagnostics_summary,omitempty"`
	ActiveObjectHints     []ActiveObjectHint  `json:"active_object_hints,omitempty"`
	RecommendedFocusa     RecommendedFocusa   `json:"recommended_focusa"`
	RecommendedNextAction string              `json:"recommended_next_action"`
	Render                RenderInfo          `json:"render"`
	HeadlessNextAction    string              `json:"headless_next_action"`
	Cleanup               *Cleanup            `json:"cleanup,omitempty"`
}

func Normalize(packet ResearchDiagnosticsPacket) ResearchDiagnosticsPacket {
	if strings.TrimSpace(packet.Schema) == "" {
		packet.Schema = SchemaResearchDiagnosticsPacketV1
	}
	packet.Mode = normalizeMode(packet.Mode)
	packet.ScopeStatus = normalizeScopeStatus(packet.ScopeStatus)
	packet.Goal = Truncate(packet.Goal, MaxGoalChars)
	packet.RecommendedNextAction = Truncate(packet.RecommendedNextAction, MaxRecommendedNextActionChars)
	packet.HeadlessNextAction = Truncate(firstNonEmpty(packet.HeadlessNextAction, packet.RecommendedNextAction), MaxRecommendedNextActionChars)
	packet.TargetRefs = sanitizeRefs(limitStrings(packet.TargetRefs, MaxTargetRefs))
	packet.EvidenceRefs = limitStrings(packet.EvidenceRefs, MaxEvidenceRefs)
	packet.Captures = sanitizeCaptures(packet.Captures)
	packet.DiagnosticsSummary = sanitizeDiagnosticsSummary(packet.DiagnosticsSummary)
	packet.ActiveObjectHints = sanitizeActiveObjectHints(packet.ActiveObjectHints)
	packet.RecommendedFocusa = sanitizeRecommendedFocusa(packet.RecommendedFocusa)
	packet.Render.SummaryLine = Truncate(packet.Render.SummaryLine, 500)
	packet.Render.ExpandableJSONRef = Truncate(packet.Render.ExpandableJSONRef, 240)
	if packet.Cleanup != nil {
		packet.Cleanup.SessionID = Truncate(packet.Cleanup.SessionID, 120)
		packet.Cleanup.RecommendedAction = Truncate(packet.Cleanup.RecommendedAction, 80)
		packet.Cleanup.Tool = Truncate(packet.Cleanup.Tool, 80)
	}
	return EnforceBudget(packet, DefaultMaxPacketBytes)
}

func EnforceBudget(packet ResearchDiagnosticsPacket, maxBytes int) ResearchDiagnosticsPacket {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxPacketBytes
	}
	for packetSize(packet) > maxBytes {
		switched := false
		if packet.DiagnosticsSummary != nil && len(packet.DiagnosticsSummary.TopFindings) > 0 {
			packet.DiagnosticsSummary.TopFindings = packet.DiagnosticsSummary.TopFindings[:len(packet.DiagnosticsSummary.TopFindings)-1]
			switched = true
		} else if len(packet.ActiveObjectHints) > 0 {
			packet.ActiveObjectHints = packet.ActiveObjectHints[:len(packet.ActiveObjectHints)-1]
			switched = true
		} else if len(packet.Captures) > 0 {
			packet.Captures = packet.Captures[:len(packet.Captures)-1]
			switched = true
		} else if len(packet.EvidenceRefs) > 0 {
			packet.EvidenceRefs = packet.EvidenceRefs[:len(packet.EvidenceRefs)-1]
			switched = true
		} else if len(packet.TargetRefs) > 0 {
			packet.TargetRefs = packet.TargetRefs[:len(packet.TargetRefs)-1]
			switched = true
		} else if packet.RecommendedFocusa.ArgsPreview != nil {
			packet.RecommendedFocusa.ArgsPreview = nil
			switched = true
		}
		if !switched {
			packet.Goal = Truncate(packet.Goal, min(MaxGoalChars, 80))
			packet.RecommendedNextAction = Truncate(packet.RecommendedNextAction, min(MaxRecommendedNextActionChars, 80))
			packet.HeadlessNextAction = Truncate(packet.HeadlessNextAction, 80)
			packet.Render.SummaryLine = Truncate(packet.Render.SummaryLine, 120)
			break
		}
	}
	return packet
}

func SanitizeURL(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return Truncate(raw, 2048)
	}
	u.Fragment = ""
	q := u.Query()
	for key := range q {
		if IsSecretQueryKey(key) {
			q.Set(key, "REDACTED")
		}
	}
	u.RawQuery = q.Encode()
	return Truncate(u.String(), 2048)
}

func IsSecretQueryKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	for _, part := range secretQueryKeys {
		if strings.Contains(key, part) {
			return true
		}
	}
	return false
}

func Truncate(value string, maxChars int) string {
	value = strings.TrimSpace(value)
	if maxChars <= 0 {
		return ""
	}
	runes := []rune(value)
	if len(runes) <= maxChars {
		return value
	}
	if maxChars <= 1 {
		return string(runes[:maxChars])
	}
	return string(runes[:maxChars-1]) + "…"
}

func sanitizeTargetRef(ref string) string {
	ref = strings.TrimSpace(ref)
	if strings.HasPrefix(ref, "browser:http://") || strings.HasPrefix(ref, "browser:https://") {
		return "browser:" + SanitizeURL(strings.TrimPrefix(ref, "browser:"))
	}
	if strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://") {
		return SanitizeURL(ref)
	}
	return Truncate(ref, 500)
}

func sanitizeRefs(refs []string) []string {
	out := make([]string, 0, len(refs))
	for _, ref := range refs {
		if sanitized := sanitizeTargetRef(ref); sanitized != "" {
			out = append(out, sanitized)
		}
	}
	return out
}

func sanitizeCaptures(captures []Capture) []Capture {
	if len(captures) > MaxCaptures {
		captures = captures[:MaxCaptures]
	}
	out := make([]Capture, 0, len(captures))
	for _, capture := range captures {
		capture.Type = Truncate(capture.Type, 40)
		capture.EvidenceRef = Truncate(capture.EvidenceRef, 240)
		capture.TargetRef = sanitizeTargetRef(capture.TargetRef)
		capture.Title = Truncate(capture.Title, 200)
		capture.Summary = Truncate(capture.Summary, MaxCaptureSummaryChars)
		if capture.Type != "" || capture.EvidenceRef != "" || capture.TargetRef != "" || capture.Summary != "" {
			out = append(out, capture)
		}
	}
	return out
}

func sanitizeDiagnosticsSummary(summary *DiagnosticsSummary) *DiagnosticsSummary {
	if summary == nil {
		return nil
	}
	out := *summary
	out.TopFindings = limitStrings(out.TopFindings, 12)
	for i := range out.TopFindings {
		out.TopFindings[i] = Truncate(out.TopFindings[i], 240)
	}
	for packetSize(out) > MaxDiagnosticsSummaryBytes && len(out.TopFindings) > 0 {
		out.TopFindings = out.TopFindings[:len(out.TopFindings)-1]
	}
	return &out
}

func sanitizeActiveObjectHints(hints []ActiveObjectHint) []ActiveObjectHint {
	if len(hints) > MaxActiveObjectHints {
		hints = hints[:MaxActiveObjectHints]
	}
	out := make([]ActiveObjectHint, 0, len(hints))
	for _, hint := range hints {
		hint.Kind = Truncate(hint.Kind, 40)
		hint.Hint = sanitizeTargetRef(hint.Hint)
		if hint.Kind != "" || hint.Hint != "" {
			out = append(out, hint)
		}
	}
	return out
}

func sanitizeRecommendedFocusa(in RecommendedFocusa) RecommendedFocusa {
	in.PreferredTool = Truncate(in.PreferredTool, 80)
	in.FallbackTool = Truncate(in.FallbackTool, 80)
	in.NextTools = limitStrings(in.NextTools, 8)
	for i := range in.NextTools {
		in.NextTools[i] = Truncate(in.NextTools[i], 80)
	}
	in.ArgsPreview = sanitizeArgsPreview(in.ArgsPreview)
	return in
}

func sanitizeArgsPreview(args map[string]any) map[string]any {
	if len(args) == 0 {
		return nil
	}
	out := map[string]any{}
	for key, value := range args {
		key = Truncate(key, 80)
		if key == "" || IsSecretQueryKey(key) || strings.Contains(strings.ToLower(key), "cookie") || strings.Contains(strings.ToLower(key), "authorization") {
			continue
		}
		out[key] = sanitizeArgValue(key, value)
	}
	if len(out) == 0 {
		return nil
	}
	for packetSize(out) > MaxArgsPreviewBytes && len(out) > 0 {
		for key := range out {
			delete(out, key)
			break
		}
	}
	return out
}

func sanitizeArgValue(key string, value any) any {
	switch v := value.(type) {
	case string:
		if strings.Contains(strings.ToLower(key), "url") || strings.HasPrefix(strings.ToLower(v), "http://") || strings.HasPrefix(strings.ToLower(v), "https://") || strings.HasPrefix(strings.ToLower(v), "browser:http") {
			return sanitizeTargetRef(v)
		}
		return Truncate(v, 500)
	case bool, int, int64, uint64, float64:
		return v
	case []string:
		return limitStrings(v, 16)
	case []any:
		out := make([]any, 0, min(len(v), 16))
		for i, item := range v {
			if i >= 16 {
				break
			}
			out = append(out, sanitizeArgValue(key, item))
		}
		return out
	case map[string]any:
		return sanitizeArgsPreview(v)
	default:
		return Truncate(toJSONPreview(v), 500)
	}
}

func limitStrings(values []string, max int) []string {
	if max < 0 {
		max = 0
	}
	if len(values) > max {
		values = values[:max]
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = Truncate(value, 500)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func normalizeMode(mode PacketMode) PacketMode {
	switch mode {
	case ModeResearch, ModeDiagnose, ModeProof:
		return mode
	default:
		return ModeResearch
	}
}

func normalizeScopeStatus(status ScopeStatus) ScopeStatus {
	switch status {
	case ScopePresent, ScopeMissing, ScopePartial, ScopeMismatchCandidate:
		return status
	default:
		return ScopeMissing
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func packetSize(value any) int {
	data, err := json.Marshal(value)
	if err != nil {
		return DefaultMaxPacketBytes + 1
	}
	return len(data)
}

func toJSONPreview(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
