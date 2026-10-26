// Package focusaevidence defines immutable UIAI-to-Focusa evidence contracts.
// It performs no IO, verifies no signatures, grants no capabilities, and cannot
// mutate completion, provider or settlement state. Approval records supplied to
// validators must originate from a separately verified authority adapter.
package focusaevidence

import (
	"github.com/WPUIAI/uiai-engine/internal/evidencejudge"
	"time"
)

const (
	HandoffSchema          = "uiai.focusa_evidence_handoff.v1"
	OperationBindingSchema = "uiai.focusa_evidence_operation_binding.v1"
	ReconciliationSchema   = "uiai.focusa_evidence_reconciliation.v1"
	InvalidationSchema     = "uiai.focusa_evidence_invalidation.v1"
	MaxContractBytes       = 1 << 20
	MaxRefs                = 256
	MaxEffects             = 128
	MaxRefBytes            = 1024
)

type ScopeBinding = evidencejudge.ScopeBinding
type ArtifactBinding = evidencejudge.ArtifactBinding

type ConsumerRole string

const (
	ConsumerWorkpoint     ConsumerRole = "workpoint_evidence_link"
	ConsumerAcceptance    ConsumerRole = "acceptance_verification_candidate"
	ConsumerContext       ConsumerRole = "context_cognition_projection"
	ConsumerWorkset       ConsumerRole = "workset_callgraph_result_input"
	ConsumerSilentSession ConsumerRole = "silent_session_output"
	ConsumerDesktop       ConsumerRole = "desktop_view"
	ConsumerCockpit       ConsumerRole = "cockpit_view"
	ConsumerProvider      ConsumerRole = "provider_reconciliation_input"
	ConsumerFinalOutput   ConsumerRole = "agent_final_output_reference"
)

type OperationBinding struct {
	Schema                   string       `json:"schema"`
	OperationID              string       `json:"operation_id"`
	OperationVersion         string       `json:"operation_version"`
	RequestSchemaRef         string       `json:"request_schema_ref"`
	ResponseSchemaRef        string       `json:"response_schema_ref"`
	CapabilitySnapshotRef    string       `json:"capability_snapshot_ref"`
	CapabilitySnapshotSHA256 string       `json:"capability_snapshot_sha256"`
	AuthorityScope           ScopeBinding `json:"authority_scope"`
	Consumer                 ConsumerRole `json:"consumer_role"`
	ConfirmationRequired     bool         `json:"confirmation_required"`
	IdempotencyRequired      bool         `json:"idempotency_required"`
	ReceiptRequired          bool         `json:"receipt_required"`
	ApprovalEvidenceRefs     []string     `json:"approval_evidence_refs"`
	ObservedAt               time.Time    `json:"observed_at"`
	ExpiresAt                time.Time    `json:"expires_at"`
}

// OperationApproval is immutable evidence INPUT from a separately verified
// authority adapter, not a signature verifier or a self-issued approval switch.
// Its digest binds every operation/schema/capability/scope/requirement field.
type OperationApproval struct {
	BindingSHA256 string    `json:"binding_sha256"`
	EvidenceRefs  []string  `json:"evidence_refs"`
	ObservedAt    time.Time `json:"observed_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}

type OperationAssessment struct {
	State     string `json:"state"` // blocked or contract_bound; neither grants execution
	ErrorCode string `json:"error_code,omitempty"`
}

type Handoff struct {
	Schema                  string           `json:"schema"`
	HandoffID               string           `json:"handoff_id"`
	Artifact                ArtifactBinding  `json:"artifact"`
	JudgeResultRefs         []string         `json:"judge_result_refs"`
	AcceptanceAtomRefs      []string         `json:"acceptance_atom_refs"`
	ObservedAt              time.Time        `json:"observed_at"`
	ExpiresAt               time.Time        `json:"expires_at"`
	FreshnessPolicyRef      string           `json:"freshness_policy_ref"`
	RequestSHA256           string           `json:"request_sha256"` // normalized external operation request, never its payload
	IdempotencyKey          string           `json:"idempotency_key"`
	ConfirmationEvidenceRef string           `json:"confirmation_evidence_ref,omitempty"`
	Operation               OperationBinding `json:"operation"`
}

type State string

const (
	StateProposed            State = "proposed"
	StateLinked              State = "linked"
	StateVerificationPending State = "verification_pending"
	StateVerifiedCandidate   State = "verified_candidate"
	StateRejected            State = "rejected"
	StateStale               State = "stale"
	StateInvalidated         State = "invalidated"
	StateReopenRequested     State = "reopen_requested"
	StateReconciled          State = "reconciled"
	StateBlocked             State = "blocked"
	StateOutcomeUnknown      State = "outcome_unknown"
)

type Effect struct {
	EffectRef       string       `json:"effect_ref"`
	TargetRef       string       `json:"target_ref"`
	Kind            ConsumerRole `json:"kind"`
	ObservedVersion uint64       `json:"observed_version"`
	EvidenceRef     string       `json:"evidence_ref"`
}

type Reconciliation struct {
	Schema                 string       `json:"schema"`
	ReconciliationID       string       `json:"reconciliation_id"`
	HandoffRef             string       `json:"handoff_ref"`
	HandoffSHA256          string       `json:"handoff_sha256"`
	Scope                  ScopeBinding `json:"scope"`
	OperationBindingSHA256 string       `json:"operation_binding_sha256"`
	IdempotencyKey         string       `json:"idempotency_key"`
	State                  State        `json:"state"`
	RequestSHA256          string       `json:"request_sha256"`
	ResponseSHA256         string       `json:"response_sha256,omitempty"`
	ReceiptRefs            []string     `json:"receipt_refs"`
	EvidenceRefs           []string     `json:"evidence_refs"`
	AuthorityEvidenceRefs  []string     `json:"authority_evidence_refs"`
	ObservedVersion        uint64       `json:"observed_version"`
	Effects                []Effect     `json:"effects"`
	PartialEffects         []Effect     `json:"partial_effects"`
	TransportOutcome       string       `json:"transport_outcome"` // response, timeout, lost_response
	NextAction             string       `json:"next_action"`       // no_retry, await_authority_reconciliation, await_verification, retry_authorized
	ObservedAt             time.Time    `json:"observed_at"`
}

type InvalidationReason string

const (
	ReasonRetraction       InvalidationReason = "retraction"
	ReasonCorruption       InvalidationReason = "corruption"
	ReasonReinspection     InvalidationReason = "failed_reinspection"
	ReasonRevokedTrust     InvalidationReason = "revoked_trust"
	ReasonCompromisedTrust InvalidationReason = "compromised_trust"
	ReasonExpiry           InvalidationReason = "expired_evidence"
	ReasonLineage          InvalidationReason = "lineage_mismatch"
	ReasonAcceptance       InvalidationReason = "acceptance_change"
	ReasonContradiction    InvalidationReason = "newer_contradictory_artifact"
)

type Invalidation struct {
	Schema          string             `json:"schema"`
	InvalidationID  string             `json:"invalidation_id"`
	HandoffRef      string             `json:"handoff_ref"`
	HandoffSHA256   string             `json:"handoff_sha256"`
	Scope           ScopeBinding       `json:"scope"`
	ArtifactRef     string             `json:"artifact_ref"`
	ManifestSHA256  string             `json:"manifest_sha256"`
	Reason          InvalidationReason `json:"reason"`
	EvidenceRefs    []string           `json:"evidence_refs"`
	ObservedAt      time.Time          `json:"observed_at"`
	RequestedAction string             `json:"requested_action"` // revalidation_candidate or reopen_candidate; never an operation
	Operation       OperationBinding   `json:"operation"`
}
