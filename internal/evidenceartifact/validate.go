package evidenceartifact

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"path"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

var (
	ErrInvalidSchema    = errors.New("invalid evidence artifact schema")
	ErrInvalidIdentity  = errors.New("invalid evidence artifact identity")
	ErrInvalidScope     = errors.New("invalid evidence artifact scope")
	ErrInvalidAuthority = errors.New("invalid evidence artifact authority")
	ErrInvalidClaim     = errors.New("invalid evidence artifact claim")
	ErrInvalidAsset     = errors.New("invalid evidence artifact asset")
	ErrInvalidPolicy    = errors.New("invalid evidence artifact policy")
	ErrInvalidIntegrity = errors.New("invalid evidence artifact integrity")
	ErrLimitExceeded    = errors.New("evidence artifact limit exceeded")
)

func Validate(in Manifest) error {
	m := Normalize(in)
	raw, err := json.Marshal(m)
	if err != nil {
		return invalid(ErrInvalidIntegrity, "manifest JSON")
	}
	if len(raw) > MaxManifestBytes {
		return invalid(ErrLimitExceeded, "manifest bytes")
	}
	if m.Schema != SchemaManifestV1 {
		return ErrInvalidSchema
	}
	if err := validateIdentity(m); err != nil {
		return err
	}
	if err := validateScope(m.Scope); err != nil {
		return err
	}
	if err := validateAuthority(m.Authority); err != nil {
		return err
	}
	if err := validateClaims(m.Claims); err != nil {
		return err
	}
	if err := validateAssets(m.Assets); err != nil {
		return err
	}
	if err := validateProvenance(m.Provenance); err != nil {
		return err
	}
	if err := validateVerification(m.Verification); err != nil {
		return err
	}
	if err := validateSecurity(m.Security); err != nil {
		return err
	}
	if err := validateRefs(m.ReceiptRefs, 1, MaxReceiptRefs); err != nil {
		return invalid(ErrInvalidIntegrity, "receipt_refs")
	}
	if err := validatePolicy(m.Policy); err != nil {
		return err
	}
	if err := validateIntegrity(m.Integrity); err != nil {
		return err
	}
	return validateLinks(m.Links)
}

func validateIdentity(m Manifest) error {
	if !validRef(m.ArtifactID, true) || m.Revision == 0 || !validText(m.Title, MaxTitleRunes, true) || !validText(m.Summary, MaxSummaryRunes, false) {
		return ErrInvalidIdentity
	}
	if len(m.Kinds) == 0 || len(m.Kinds) > MaxKinds {
		return invalid(ErrLimitExceeded, "kinds")
	}
	for _, kind := range m.Kinds {
		if !validLabel(kind, 80, true) {
			return invalid(ErrInvalidIdentity, "kind")
		}
	}
	captured, ok := canonicalTime(m.CapturedAt, true)
	if !ok {
		return invalid(ErrInvalidIdentity, "captured_at")
	}
	created, ok := canonicalTime(m.CreatedAt, true)
	if !ok || created.Before(captured) {
		return invalid(ErrInvalidIdentity, "created_at")
	}
	return nil
}

func validateScope(s Scope) error {
	if s.Project.State != BindingMatched || !validRef(s.Project.ProjectRef, true) || !validLabel(s.Project.Fingerprint, 256, true) || !validRef(s.Project.WorkingSubpathRef, true) {
		return invalid(ErrInvalidScope, "project binding")
	}
	if s.Workstream.State != BindingMatched || !validRef(s.Workstream.WorkstreamRef, true) {
		return invalid(ErrInvalidScope, "workstream binding")
	}
	if s.Workset.State != BindingMatched || !validRef(s.Workset.WorksetRef, true) || s.Workset.Revision == 0 || !validSHA256(s.Workset.Digest) || !validRef(s.Workset.MembershipRef, true) {
		return invalid(ErrInvalidScope, "workset binding")
	}
	if err := validateRefs(s.Workset.RequirementRefs, 1, MaxRefsPerList); err != nil {
		return invalid(ErrInvalidScope, "workset requirement refs")
	}
	if err := validateRefs(s.Workset.DispositionRefs, 0, MaxRefsPerList); err != nil {
		return invalid(ErrInvalidScope, "workset disposition refs")
	}
	cg := s.CallGraph
	if cg.State != BindingMatched || !validRef(cg.DefinitionRef, true) || cg.DefinitionRevision == 0 || !validRef(cg.RunRef, true) || !validRef(cg.FrameRef, true) || !validRef(cg.NodeRef, true) || !validRef(cg.ItemRef, true) || cg.Attempt == 0 || cg.Generation == 0 {
		return invalid(ErrInvalidScope, "callgraph binding")
	}
	for _, ref := range []string{cg.PathRef, cg.ParentFrameRef, cg.JoinRef, cg.CompensationRef} {
		if !validRef(ref, false) {
			return invalid(ErrInvalidScope, "callgraph optional ref")
		}
	}
	wp := s.Workpoint
	if wp.State != BindingMatched || !validRef(wp.WorkpointRef, true) || wp.Revision == 0 || !validRef(wp.CheckpointRef, true) || !validRef(wp.CurrentActionIntentRef, true) {
		return invalid(ErrInvalidScope, "workpoint binding")
	}
	if err := validateAutonomy(s.Autonomy); err != nil {
		return err
	}
	if err := validateWorkItems(s.WorkItems); err != nil {
		return err
	}
	if !validRef(s.TrajectoryRef, false) {
		return invalid(ErrInvalidScope, "trajectory_ref")
	}
	for name, refs := range map[string][]string{
		"assignment_refs": s.AssignmentRefs,
		"operation_refs":  s.OperationRefs,
		"ontology_refs":   s.OntologyRefs,
		"rehydrate_refs":  s.RehydrateRefs,
	} {
		if err := validateRefs(refs, 0, MaxRefsPerList); err != nil {
			return invalid(ErrInvalidScope, name)
		}
	}
	return nil
}

func validateAutonomy(a AutonomyBinding) error {
	if !validLabel(a.Mode, 80, true) || !validLabel(a.RunStatus, 80, true) {
		return invalid(ErrInvalidScope, "autonomy mode/status")
	}
	for _, ref := range []string{a.PolicyRef, a.WorkLoopRef, a.RunRef, a.AgentTeamPlanRef, a.ExecutorAssignmentRef} {
		if !validRef(ref, true) {
			return invalid(ErrInvalidScope, "autonomy required ref")
		}
	}
	for _, ref := range []string{a.BudgetPolicyRef, a.ResourcePolicyRef, a.RetryPolicyRef, a.FailoverPolicyRef, a.CircuitBreakerPolicyRef, a.ReviewPostureRef, a.ClosurePostureRef, a.EventCursorRef} {
		if !validRef(ref, true) {
			return invalid(ErrInvalidScope, "autonomy policy/posture ref")
		}
	}
	if !validRef(a.CooldownPolicyRef, false) {
		return invalid(ErrInvalidScope, "autonomy cooldown ref")
	}
	for name, refs := range map[string][]string{
		"verifier_assignment_refs":   a.VerifierAssignmentRefs,
		"arbitrator_assignment_refs": a.ArbitratorAssignmentRefs,
		"capability_digest_refs":     a.CapabilityDigestRefs,
		"continuation_refs":          a.ContinuationRefs,
	} {
		if err := validateRefs(refs, 0, MaxRefsPerList); err != nil {
			return invalid(ErrInvalidScope, name)
		}
	}
	if len(a.CapabilityDigestRefs) == 0 || len(a.ContinuationRefs) == 0 {
		return invalid(ErrInvalidScope, "autonomy capability/continuation refs")
	}
	return nil
}

func validateWorkItems(items []WorkItemBinding) error {
	if len(items) == 0 {
		return invalid(ErrInvalidScope, "work_items")
	}
	if len(items) > MaxWorkItems {
		return invalid(ErrLimitExceeded, "work_items")
	}
	refs := make(map[string]struct{}, len(items))
	ids := make(map[string]struct{}, len(items))
	for _, item := range items {
		if !validLabel(item.ProviderSurface, 80, true) || !validRef(item.WorkItemRef, true) || !validLabel(item.ItemID, 256, true) || !validLabel(item.ItemType, 80, true) || !validText(item.Title, MaxTitleRunes, true) || !validText(item.Description, MaxDescriptionRunes, false) {
			return invalid(ErrInvalidScope, "work item identity")
		}
		if _, exists := refs[item.WorkItemRef]; exists {
			return invalid(ErrInvalidScope, "duplicate work item ref")
		}
		refs[item.WorkItemRef] = struct{}{}
		idKey := item.ProviderSurface + "\x00" + item.ItemID
		if _, exists := ids[idKey]; exists {
			return invalid(ErrInvalidScope, "duplicate provider item id")
		}
		ids[idKey] = struct{}{}
		if item.Description == "" && (!validRef(item.DescriptionRef, true) || !validSHA256(item.DescriptionSHA256)) {
			return invalid(ErrInvalidScope, "work item description")
		}
		if item.Description != "" && (!validRef(item.DescriptionRef, false) || (item.DescriptionSHA256 != "" && (!validSHA256(item.DescriptionSHA256) || item.DescriptionSHA256 != textSHA256(item.Description)))) {
			return invalid(ErrInvalidScope, "work item description metadata")
		}
		if !validLabel(item.Revision, 256, true) || !validSHA256(item.Digest) || !validLabel(item.StatusAtCapture, 80, true) || !validLabel(item.ClosurePosture, 80, true) {
			return invalid(ErrInvalidScope, "work item revision/status")
		}
		for _, list := range [][]string{item.ParentRefs, item.DependencyRefs, item.BlockerRefs, item.AcceptanceAtomRefs, item.EvidenceRequirementRefs, item.ReviewRequirementRefs} {
			if err := validateRefs(list, 0, MaxRefsPerList); err != nil {
				return invalid(ErrInvalidScope, "work item refs")
			}
		}
	}
	return nil
}

func validateAuthority(a Authority) error {
	for _, ref := range []string{a.ProducerRef, a.SourceAuthorityRef, a.EvidenceAuthorityRef, a.CompletionAuthorityRef, a.ReviewerPolicyRef} {
		if !validRef(ref, true) {
			return ErrInvalidAuthority
		}
	}
	switch a.Posture {
	case PostureCanonical, PostureAdvisory, PostureDegraded, PostureBlocked, PostureStale:
		return nil
	default:
		return ErrInvalidAuthority
	}
}

func validateClaims(claims []Claim) error {
	if len(claims) == 0 || len(claims) > MaxClaims {
		return invalid(ErrLimitExceeded, "claims")
	}
	seen := make(map[string]struct{}, len(claims))
	for _, claim := range claims {
		if !validRef(claim.ClaimID, true) || !validText(claim.Summary, MaxSummaryRunes, true) {
			return ErrInvalidClaim
		}
		if _, exists := seen[claim.ClaimID]; exists {
			return invalid(ErrInvalidClaim, "duplicate claim_id")
		}
		seen[claim.ClaimID] = struct{}{}
		switch claim.Status {
		case ClaimActual, ClaimPartial, ClaimSurrogate, ClaimBlocked, ClaimMissing:
		default:
			return ErrInvalidClaim
		}
		for _, refs := range [][]string{claim.AcceptanceAtomRefs, claim.EvidenceRefs, claim.ReviewRequirementRefs} {
			if err := validateRefs(refs, 0, MaxRefsPerList); err != nil {
				return ErrInvalidClaim
			}
		}
		if len(claim.EvidenceRefs) == 0 {
			return invalid(ErrInvalidClaim, "evidence refs")
		}
	}
	return nil
}

func validateAssets(assets []Asset) error {
	if len(assets) == 0 || len(assets) > MaxAssets {
		return invalid(ErrLimitExceeded, "assets")
	}
	seen := make(map[string]struct{}, len(assets))
	for _, asset := range assets {
		if !validRef(asset.AssetID, true) || !validLabel(asset.Kind, 80, true) || !validMediaType(asset.MediaType) || !validRelativePath(asset.Path) || !validSHA256(asset.SHA256) || asset.ByteSize <= 0 || !validRef(asset.SourceRef, true) {
			return ErrInvalidAsset
		}
		if _, exists := seen[asset.AssetID]; exists {
			return invalid(ErrInvalidAsset, "duplicate asset_id")
		}
		seen[asset.AssetID] = struct{}{}
		if _, ok := canonicalTime(asset.CapturedAt, false); !ok {
			return invalid(ErrInvalidAsset, "captured_at")
		}
		if err := validateRefs(asset.ClaimRefs, 1, MaxClaims); err != nil {
			return invalid(ErrInvalidAsset, "claim_refs")
		}
		switch asset.VerificationClass {
		case VerificationActual, VerificationBlockedClass, VerificationSurrogate, VerificationMissingNative:
		default:
			return ErrInvalidAsset
		}
		switch asset.RedactionState {
		case RedactionNone, RedactionRedacted, RedactionBlocked, RedactionPublicSafe:
		default:
			return ErrInvalidAsset
		}
		if strings.HasPrefix(asset.MediaType, "image/") && (asset.Width <= 0 || asset.Height <= 0 || !validText(asset.AltText, MaxSummaryRunes, true)) {
			return invalid(ErrInvalidAsset, "image metadata")
		}
		if strings.HasPrefix(asset.MediaType, "video/") && (asset.DurationMS <= 0 || !validRef(asset.TranscriptRef, true)) {
			return invalid(ErrInvalidAsset, "video metadata")
		}
	}
	return nil
}

func validateProvenance(p Provenance) error {
	for _, refs := range [][]string{p.SourceRefs, p.EnvironmentRefs, p.OmissionRefs} {
		if err := validateRefs(refs, 0, MaxRefsPerList); err != nil {
			return invalid(ErrInvalidIntegrity, "provenance refs")
		}
	}
	if len(p.SourceRefs) == 0 || len(p.Custody) == 0 || len(p.Custody) > MaxCustodyEvents {
		return invalid(ErrInvalidIntegrity, "provenance custody")
	}
	seen := make(map[string]struct{}, len(p.Custody))
	var previous time.Time
	for _, event := range p.Custody {
		if !validRef(event.EventID, true) || !validLabel(event.Action, 80, true) || !validRef(event.ActorRef, true) || !validRef(event.InstanceRef, true) {
			return invalid(ErrInvalidIntegrity, "custody event")
		}
		if _, exists := seen[event.EventID]; exists {
			return invalid(ErrInvalidIntegrity, "duplicate custody event")
		}
		seen[event.EventID] = struct{}{}
		if err := validateRefs(event.InputRefs, 0, MaxRefsPerList); err != nil {
			return invalid(ErrInvalidIntegrity, "custody input refs")
		}
		if err := validateRefs(event.OutputRefs, 1, MaxRefsPerList); err != nil {
			return invalid(ErrInvalidIntegrity, "custody output refs")
		}
		occurred, ok := canonicalTime(event.OccurredAt, true)
		if !ok || (!previous.IsZero() && occurred.Before(previous)) {
			return invalid(ErrInvalidIntegrity, "custody chronology")
		}
		previous = occurred
	}
	return nil
}

func validateVerification(v Verification) error {
	switch v.Status {
	case VerificationPending, VerificationPassed, VerificationFailed, VerificationBlocked, VerificationIndeterminate, VerificationDisputed:
	default:
		return invalid(ErrInvalidIntegrity, "verification status")
	}
	if !validRef(v.ReviewCaseRef, true) {
		return invalid(ErrInvalidIntegrity, "review_case_ref")
	}
	for _, refs := range [][]string{v.VerifierRefs, v.JudgeResultRefs, v.DecisionRefs} {
		if err := validateRefs(refs, 0, MaxRefsPerList); err != nil {
			return invalid(ErrInvalidIntegrity, "verification refs")
		}
	}
	if v.InformationSetHash != "" && !validSHA256(v.InformationSetHash) {
		return invalid(ErrInvalidIntegrity, "information_set_hash")
	}
	return nil
}

func validatePolicy(p Policy) error {
	switch p.AccessClass {
	case AccessLocal, AccessLAN, AccessTailnet, AccessPrivateTeam, AccessUnlisted, AccessPublicSafe:
	default:
		return ErrInvalidPolicy
	}
	switch p.RedactionState {
	case RedactionNone, RedactionRedacted, RedactionBlocked, RedactionPublicSafe:
	default:
		return ErrInvalidPolicy
	}
	switch p.RetentionClass {
	case RetentionEphemeral, RetentionProject, RetentionWorkstream, RetentionRelease, RetentionLegalHold, RetentionCustom:
	default:
		return ErrInvalidPolicy
	}
	if !validLabel(p.Audience, 160, true) {
		return ErrInvalidPolicy
	}
	if _, ok := canonicalTime(p.ExpiresAt, false); !ok {
		return invalid(ErrInvalidPolicy, "expires_at")
	}
	if err := validateRefs(p.PolicyRefs, 1, MaxPolicyRefs); err != nil {
		return ErrInvalidPolicy
	}
	if p.AccessClass == AccessPublicSafe && p.RedactionState != RedactionPublicSafe {
		return invalid(ErrInvalidPolicy, "public_safe redaction")
	}
	return nil
}

func validateIntegrity(i Integrity) error {
	if i.Algorithm != "sha256" || (i.ManifestSHA256 != "" && !validSHA256(i.ManifestSHA256)) || (i.BundleSHA256 != "" && !validSHA256(i.BundleSHA256)) {
		return ErrInvalidIntegrity
	}
	return nil
}

func validateLinks(l Links) error {
	if !validRelativePath(l.ManifestPath) || (l.PWAPath != "" && !validRelativePath(l.PWAPath)) || !validRef(l.SupersedesRef, false) {
		return invalid(ErrInvalidIntegrity, "links")
	}
	if err := validateRefs(l.RelatedRefs, 0, MaxRelatedRefs); err != nil {
		return invalid(ErrInvalidIntegrity, "related_refs")
	}
	return nil
}

func validateRefs(refs []string, min, max int) error {
	if len(refs) < min || len(refs) > max {
		return ErrLimitExceeded
	}
	for _, ref := range refs {
		if !validRef(ref, true) {
			return ErrInvalidIdentity
		}
	}
	return nil
}

func validRef(value string, required bool) bool {
	if value == "" {
		return !required
	}
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > MaxRefRunes || strings.HasPrefix(value, "/") || strings.HasPrefix(strings.ToLower(value), "file:") || strings.Contains(value, "\\") {
		return false
	}
	if len(value) >= 3 && ((value[0] >= 'A' && value[0] <= 'Z') || (value[0] >= 'a' && value[0] <= 'z')) && value[1] == ':' && value[2] == '/' {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func validText(value string, max int, required bool) bool {
	if value == "" {
		return !required
	}
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > max {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			return false
		}
	}
	return true
}

func validLabel(value string, max int, required bool) bool {
	return validText(value, max, required) && !strings.ContainsAny(value, "\r\n\t")
}

func validSHA256(value string) bool {
	if len(value) != 64 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil && value == strings.ToLower(value)
}

func validRelativePath(value string) bool {
	if !validRef(value, true) || strings.Contains(value, "://") {
		return false
	}
	clean := path.Clean(value)
	return clean == value && clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}

func validMediaType(value string) bool {
	if value == "" || strings.Contains(value, ";") {
		return false
	}
	parsed, params, err := mime.ParseMediaType(value)
	return err == nil && parsed == value && len(params) == 0 && strings.Contains(parsed, "/")
}

func canonicalTime(value string, required bool) (time.Time, bool) {
	if value == "" {
		return time.Time{}, !required
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || parsed.UTC().Format(time.RFC3339Nano) != value {
		return time.Time{}, false
	}
	return parsed, true
}

func invalid(base error, field string) error {
	return fmt.Errorf("%w: %s", base, field)
}
