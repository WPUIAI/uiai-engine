package focusaevidence

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/WPUIAI/uiai-engine/internal/secrettext"
	"io"
	"strings"
	"time"
)

var (
	ErrHandoffContractInvalid       = errors.New("focusa evidence handoff contract invalid")
	ErrLineageBindingMismatch       = errors.New("focusa evidence lineage binding mismatch")
	ErrArtifactBindingMismatch      = errors.New("focusa evidence artifact binding mismatch")
	ErrOperationUnapproved          = errors.New("focusa evidence operation unapproved")
	ErrCapabilityBindingStale       = errors.New("focusa evidence capability binding stale")
	ErrAuthorityScopeMismatch       = errors.New("focusa evidence authority scope mismatch")
	ErrReceiptRequired              = errors.New("focusa evidence receipt required")
	ErrFocusaOutcomeUnknown         = errors.New("focusa evidence outcome unknown")
	ErrFocusaReconciliationRequired = errors.New("focusa evidence reconciliation required")
	ErrEvidenceStale                = errors.New("focusa evidence stale")
	ErrEvidenceInvalidated          = errors.New("focusa evidence invalidated")
	ErrCompletionAuthorityViolation = errors.New("focusa evidence completion authority violation")
)

func ValidateOperationBinding(v OperationBinding) error {
	if v.Schema != OperationBindingSchema || !scopeOK(v.AuthorityScope) || !roleOK(v.Consumer) || !refsOK(v.ApprovalEvidenceRefs, true) ||
		!allRefs(v.OperationID, v.OperationVersion, v.RequestSchemaRef, v.ResponseSchemaRef, v.CapabilitySnapshotRef) || !shaOK(v.CapabilitySnapshotSHA256) ||
		!v.IdempotencyRequired || !v.ReceiptRequired || !windowOK(v.ObservedAt, v.ExpiresAt) {
		return ErrOperationUnapproved
	}
	return sizeOK(v)
}

// Against checks an exact separately supplied approval record. It is contract
// validation, NOT signature verification, revocation checking or execution admission.
func ValidateOperationBindingAgainst(v OperationBinding, a OperationApproval, scope ScopeBinding, now time.Time) error {
	if err := ValidateOperationBinding(v); err != nil {
		return err
	}
	if !scopeOK(scope) || v.AuthorityScope != scope {
		return ErrAuthorityScopeMismatch
	}
	if now.IsZero() || v.ObservedAt.After(now) || !now.Before(v.ExpiresAt) || !windowOK(a.ObservedAt, a.ExpiresAt) || a.ObservedAt.After(now) || !now.Before(a.ExpiresAt) {
		return ErrCapabilityBindingStale
	}
	digest, err := DigestOperationBinding(v)
	if err != nil {
		return err
	}
	if !shaOK(a.BindingSHA256) || a.BindingSHA256 != digest || !refsOK(a.EvidenceRefs, true) || !sameRefs(a.EvidenceRefs, v.ApprovalEvidenceRefs) {
		return ErrOperationUnapproved
	}
	return nil
}

// The only positive posture is contract_bound, never authorized or executable.
// Missing/unknown/widened/stale approval serializes as a bounded blocked projection.
func AssessOperation(v OperationBinding, a OperationApproval, scope ScopeBinding, now time.Time) OperationAssessment {
	if err := ValidateOperationBindingAgainst(v, a, scope, now); err != nil {
		code := "operation_unapproved"
		if errors.Is(err, ErrCapabilityBindingStale) {
			code = "capability_binding_stale"
		}
		if errors.Is(err, ErrAuthorityScopeMismatch) {
			code = "authority_scope_mismatch"
		}
		return OperationAssessment{State: "blocked", ErrorCode: code}
	}
	return OperationAssessment{State: "contract_bound"}
}

func ValidateHandoff(v Handoff) error {
	a := v.Artifact
	if v.Schema != HandoffSchema || !allRefs(v.HandoffID, a.ArtifactRef, v.FreshnessPolicyRef, v.IdempotencyKey) || !scopeOK(a.Scope) || a.Revision == 0 || a.Revision > 1<<53-1 || !shaOK(a.BundleSHA256) || !shaOK(a.ManifestSHA256) || !shaOK(v.RequestSHA256) || !windowOK(v.ObservedAt, v.ExpiresAt) {
		return ErrHandoffContractInvalid
	}
	for _, refs := range [][]string{a.AttestationRefs, a.TrustRefs, a.SecurityRefs, v.JudgeResultRefs, v.AcceptanceAtomRefs} {
		if !refsOK(refs, true) {
			return ErrHandoffContractInvalid
		}
	}
	if err := ValidateOperationBinding(v.Operation); err != nil {
		return err
	}
	if v.Operation.AuthorityScope != a.Scope {
		return ErrAuthorityScopeMismatch
	}
	if (v.Operation.ConfirmationRequired || v.ConfirmationEvidenceRef != "") && !refOK(v.ConfirmationEvidenceRef) {
		return ErrOperationUnapproved
	}
	if v.ObservedAt.Before(v.Operation.ObservedAt) || v.ExpiresAt.After(v.Operation.ExpiresAt) {
		return ErrCapabilityBindingStale
	}
	return sizeOK(v)
}

func ValidateHandoffAgainst(v Handoff, artifact ArtifactBinding, now time.Time) error {
	if err := ValidateHandoff(v); err != nil {
		return err
	}
	if v.Artifact.Scope != artifact.Scope {
		return ErrLineageBindingMismatch
	}
	a := v.Artifact
	if a.ArtifactRef != artifact.ArtifactRef || a.Revision != artifact.Revision || a.ManifestSHA256 != artifact.ManifestSHA256 || a.BundleSHA256 != artifact.BundleSHA256 || !sameRefs(a.AttestationRefs, artifact.AttestationRefs) || !sameRefs(a.TrustRefs, artifact.TrustRefs) || !sameRefs(a.SecurityRefs, artifact.SecurityRefs) {
		return ErrArtifactBindingMismatch
	}
	if now.IsZero() || v.ObservedAt.After(now) || !now.Before(v.ExpiresAt) {
		return ErrEvidenceStale
	}
	return nil
}

func ValidateReconciliation(v Reconciliation) error {
	if v.Schema != ReconciliationSchema || !allRefs(v.ReconciliationID, v.HandoffRef, v.IdempotencyKey) || !scopeOK(v.Scope) || !shaOK(v.HandoffSHA256) || !shaOK(v.OperationBindingSHA256) || !shaOK(v.RequestSHA256) || v.ObservedAt.IsZero() || v.ObservedVersion > 1<<53-1 || !stateOK(v.State) {
		return ErrHandoffContractInvalid
	}
	if !refsOK(v.ReceiptRefs, false) || !refsOK(v.EvidenceRefs, true) || !refsOK(v.AuthorityEvidenceRefs, false) || len(v.Effects)+len(v.PartialEffects) > MaxEffects {
		return ErrHandoffContractInvalid
	}
	switch v.TransportOutcome {
	case "timeout", "lost_response":
		if v.State != StateOutcomeUnknown || v.NextAction != "await_authority_reconciliation" || v.ResponseSHA256 != "" || len(v.Effects) > 0 {
			return ErrFocusaOutcomeUnknown
		}
	case "response":
		if !shaOK(v.ResponseSHA256) {
			return ErrHandoffContractInvalid
		}
	default:
		return ErrHandoffContractInvalid
	}
	switch v.NextAction {
	case "no_retry", "await_authority_reconciliation", "await_verification", "retry_authorized":
	default:
		return ErrHandoffContractInvalid
	}
	if v.State == StateOutcomeUnknown && v.NextAction != "await_authority_reconciliation" {
		return ErrFocusaOutcomeUnknown
	}
	authoritative := v.State == StateLinked || v.State == StateVerifiedCandidate || v.State == StateReconciled
	if len(v.Effects) > 0 && !authoritative {
		return ErrHandoffContractInvalid
	}
	if len(v.PartialEffects) > 0 && v.State != StateOutcomeUnknown && v.State != StateVerificationPending && v.State != StateReconciled {
		return ErrHandoffContractInvalid
	}
	if authoritative && len(v.ReceiptRefs) == 0 {
		return ErrReceiptRequired
	}
	if (v.State == StateVerifiedCandidate || v.State == StateReconciled) && (len(v.AuthorityEvidenceRefs) == 0 || v.ObservedVersion == 0) {
		return ErrFocusaReconciliationRequired
	}
	if v.NextAction == "retry_authorized" && v.State != StateReconciled {
		return ErrFocusaReconciliationRequired
	}
	if len(v.PartialEffects) > 0 && v.State == StateReconciled && len(v.AuthorityEvidenceRefs) == 0 {
		return ErrFocusaReconciliationRequired
	}
	seen := map[string]bool{}
	available := map[string]bool{}
	for _, refs := range [][]string{v.ReceiptRefs, v.EvidenceRefs, v.AuthorityEvidenceRefs} {
		for _, ref := range refs {
			available[ref] = true
		}
	}
	for _, group := range [][]Effect{v.Effects, v.PartialEffects} {
		for _, e := range group {
			if !allRefs(e.EffectRef, e.TargetRef, e.EvidenceRef) || !roleOK(e.Kind) || e.ObservedVersion == 0 || e.ObservedVersion > v.ObservedVersion || seen[e.EffectRef] || !available[e.EvidenceRef] {
				return ErrHandoffContractInvalid
			}
			seen[e.EffectRef] = true
		}
	}
	return sizeOK(v)
}

// Result observation is not new admission: a receipt may legitimately arrive
// after the handoff/operation expired. Its immutable bindings still must match.
func ValidateReconciliationAgainst(v Reconciliation, h Handoff, now time.Time) error {
	if err := ValidateHandoff(h); err != nil {
		return err
	}
	if err := ValidateReconciliation(v); err != nil {
		return err
	}
	hd, err := DigestHandoff(h)
	if err != nil {
		return err
	}
	od, err := DigestOperationBinding(h.Operation)
	if err != nil {
		return err
	}
	if v.Scope != h.Artifact.Scope {
		return ErrLineageBindingMismatch
	}
	if v.HandoffRef != h.HandoffID || v.HandoffSHA256 != hd || v.OperationBindingSHA256 != od || v.IdempotencyKey != h.IdempotencyKey || v.RequestSHA256 != h.RequestSHA256 {
		return ErrArtifactBindingMismatch
	}
	if now.IsZero() || v.ObservedAt.After(now) || v.ObservedAt.Before(h.ObservedAt) {
		return ErrHandoffContractInvalid
	}
	for _, group := range [][]Effect{v.Effects, v.PartialEffects} {
		for _, e := range group {
			if e.Kind != h.Operation.Consumer || !targetOK(e.TargetRef, e.Kind, h) {
				return ErrAuthorityScopeMismatch
			}
		}
	}
	return nil
}

func ValidateInvalidation(v Invalidation) error {
	if v.Schema != InvalidationSchema || !allRefs(v.InvalidationID, v.HandoffRef, v.ArtifactRef) || !shaOK(v.HandoffSHA256) || !shaOK(v.ManifestSHA256) || !scopeOK(v.Scope) || !refsOK(v.EvidenceRefs, true) || v.ObservedAt.IsZero() {
		return ErrHandoffContractInvalid
	}
	switch v.Reason {
	case ReasonRetraction, ReasonCorruption, ReasonReinspection, ReasonRevokedTrust, ReasonCompromisedTrust, ReasonExpiry, ReasonLineage, ReasonAcceptance, ReasonContradiction:
	default:
		return ErrHandoffContractInvalid
	}
	if v.RequestedAction != "revalidation_candidate" && v.RequestedAction != "reopen_candidate" {
		return ErrCompletionAuthorityViolation
	}
	if err := ValidateOperationBinding(v.Operation); err != nil {
		return err
	}
	if v.Operation.AuthorityScope != v.Scope {
		return ErrAuthorityScopeMismatch
	}
	return sizeOK(v)
}
func ValidateInvalidationAgainst(v Invalidation, h Handoff, now time.Time) error {
	if err := ValidateHandoff(h); err != nil {
		return err
	}
	if err := ValidateInvalidation(v); err != nil {
		return err
	}
	hd, err := DigestHandoff(h)
	if err != nil {
		return err
	}
	if v.Scope != h.Artifact.Scope {
		return ErrLineageBindingMismatch
	}
	if v.HandoffRef != h.HandoffID || v.HandoffSHA256 != hd || v.ArtifactRef != h.Artifact.ArtifactRef || v.ManifestSHA256 != h.Artifact.ManifestSHA256 {
		return ErrArtifactBindingMismatch
	}
	if now.IsZero() || v.ObservedAt.After(now) || v.ObservedAt.Before(h.ObservedAt) || (v.Reason == ReasonExpiry && v.ObservedAt.Before(h.ExpiresAt)) {
		return ErrHandoffContractInvalid
	}
	return nil
}

// ValidateRetryCandidateAgainst checks a proposal only; it grants no retry.
// Exact immutable bindings and authoritative reconciliation evidence are required.
func ValidateRetryCandidateAgainst(v Reconciliation, h Handoff, now time.Time) error {
	if err := ValidateReconciliationAgainst(v, h, now); err != nil {
		return err
	}
	switch v.State {
	case StateOutcomeUnknown:
		return ErrFocusaOutcomeUnknown
	case StateStale:
		return ErrEvidenceStale
	case StateInvalidated:
		return ErrEvidenceInvalidated
	}
	if v.State != StateReconciled || v.NextAction != "retry_authorized" {
		return ErrFocusaReconciliationRequired
	}
	return nil
}

func DecodeHandoff(body []byte) (Handoff, error) {
	var v Handoff
	if err := strictDecode(body, &v); err != nil {
		return Handoff{}, err
	}
	return v, ValidateHandoff(v)
}
func DecodeOperationBinding(body []byte) (OperationBinding, error) {
	var v OperationBinding
	if err := strictDecode(body, &v); err != nil {
		return OperationBinding{}, err
	}
	return v, ValidateOperationBinding(v)
}
func DecodeReconciliation(body []byte) (Reconciliation, error) {
	var v Reconciliation
	if err := strictDecode(body, &v); err != nil {
		return Reconciliation{}, err
	}
	return v, ValidateReconciliation(v)
}
func DecodeInvalidation(body []byte) (Invalidation, error) {
	var v Invalidation
	if err := strictDecode(body, &v); err != nil {
		return Invalidation{}, err
	}
	return v, ValidateInvalidation(v)
}

func DigestHandoff(v Handoff) (string, error) {
	if err := ValidateHandoff(v); err != nil {
		return "", err
	}
	return digest(v)
}
func DigestOperationBinding(v OperationBinding) (string, error) {
	if err := ValidateOperationBinding(v); err != nil {
		return "", err
	}
	return digest(v)
}
func DigestReconciliation(v Reconciliation) (string, error) {
	if err := ValidateReconciliation(v); err != nil {
		return "", err
	}
	return digest(v)
}

func strictDecode(body []byte, out any) error {
	if len(body) == 0 || len(body) > MaxContractBytes {
		return ErrHandoffContractInvalid
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.UseNumber()
	if err := inspectJSON(d, 0); err != nil {
		return err
	}
	if _, err := d.Token(); err != io.EOF {
		return ErrHandoffContractInvalid
	}
	d = json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil {
		return ErrHandoffContractInvalid
	}
	return nil
}

// One bounded token walk rejects duplicate keys, forbidden authority fields and
// deep inputs before encoding/json can silently overwrite a binding.
func inspectJSON(d *json.Decoder, depth int) error {
	if depth > 32 {
		return ErrHandoffContractInvalid
	}
	tok, err := d.Token()
	if err != nil {
		return ErrHandoffContractInvalid
	}
	delim, ok := tok.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return ErrHandoffContractInvalid
			}
			s, ok := key.(string)
			if !ok || seen[s] {
				return ErrHandoffContractInvalid
			}
			seen[s] = true
			normalized := strings.ToLower(strings.ReplaceAll(s, "_", ""))
			switch normalized {
			case "completed", "settled", "providerclosed", "taskstatus", "completiondecision", "settlement", "reopened", "reviewapproval":
				return ErrCompletionAuthorityViolation
			}
			for _, c := range s {
				if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_') {
					return ErrHandoffContractInvalid
				}
			}
			if err := inspectJSON(d, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err := inspectJSON(d, depth+1); err != nil {
				return err
			}
		}
	default:
		return ErrHandoffContractInvalid
	}
	closing, err := d.Token()
	if err != nil || (delim == '{' && closing != json.Delim('}')) || (delim == '[' && closing != json.Delim(']')) {
		return ErrHandoffContractInvalid
	}
	return nil
}

func scopeOK(s ScopeBinding) bool {
	return allRefs(s.ProjectRef, s.WorkstreamRef, s.WorksetRef, s.CallGraphRef, s.WorkpointRef, s.WorkItemRef)
}
func allRefs(values ...string) bool {
	for _, v := range values {
		if !refOK(v) {
			return false
		}
	}
	return true
}

// This contract uses opaque ASCII handles, not URLs or filesystem locators.
// Existing secrettext is the canonical credential-content guard, not a new detector.
func refOK(v string) bool {
	if len(v) == 0 || len(v) > MaxRefBytes || secrettext.Text(v) != v {
		return false
	}
	for i, c := range v {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || i > 0 && strings.ContainsRune("._:-#@+", c)) {
			return false
		}
	}
	return true
}
func refsOK(v []string, required bool) bool {
	if required && len(v) == 0 || len(v) > MaxRefs {
		return false
	}
	seen := map[string]bool{}
	for _, s := range v {
		if !refOK(s) || seen[s] {
			return false
		}
		seen[s] = true
	}
	return true
}
func sameRefs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := map[string]bool{}
	for _, s := range a {
		m[s] = true
	}
	for _, s := range b {
		if !m[s] {
			return false
		}
	}
	return true
}
func shaOK(s string) bool {
	if len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
func windowOK(start, end time.Time) bool {
	return !start.IsZero() && !end.IsZero() && start.Before(end)
}
func roleOK(v ConsumerRole) bool {
	switch v {
	case ConsumerWorkpoint, ConsumerAcceptance, ConsumerContext, ConsumerWorkset, ConsumerSilentSession, ConsumerDesktop, ConsumerCockpit, ConsumerProvider, ConsumerFinalOutput:
		return true
	}
	return false
}
func stateOK(v State) bool {
	switch v {
	case StateProposed, StateLinked, StateVerificationPending, StateVerifiedCandidate, StateRejected, StateStale, StateInvalidated, StateReopenRequested, StateReconciled, StateBlocked, StateOutcomeUnknown:
		return true
	}
	return false
}
func targetOK(ref string, role ConsumerRole, h Handoff) bool {
	s := h.Artifact.Scope
	switch role {
	case ConsumerWorkpoint:
		return ref == s.WorkpointRef
	case ConsumerAcceptance:
		for _, a := range h.AcceptanceAtomRefs {
			if ref == a {
				return true
			}
		}
		return false
	case ConsumerContext:
		return ref == s.ProjectRef || ref == s.WorkstreamRef
	case ConsumerWorkset:
		return ref == s.WorksetRef || ref == s.CallGraphRef
	case ConsumerProvider:
		return ref == s.WorkItemRef
	default:
		return ref == h.Artifact.ArtifactRef
	}
}
func sizeOK(v any) error {
	body, err := json.Marshal(v)
	if err != nil || len(body) > MaxContractBytes {
		return ErrHandoffContractInvalid
	}
	return nil
}

// Accepted contracts contain opaque ASCII strings and bounded integral versions.
// Sorting JSON object keys provides a cross-language deterministic digest; this
// hash is evidence identity and never an authority signature or a minted grant.
func digest(v any) (string, error) {
	body, err := json.Marshal(v)
	if err != nil || len(body) > MaxContractBytes {
		return "", ErrHandoffContractInvalid
	}
	var raw any
	d := json.NewDecoder(bytes.NewReader(body))
	d.UseNumber()
	if d.Decode(&raw) != nil {
		return "", ErrHandoffContractInvalid
	}
	body, err = json.Marshal(raw)
	if err != nil {
		return "", ErrHandoffContractInvalid
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:]), nil
}
