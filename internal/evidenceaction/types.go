package evidenceaction

import (
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidencejudge"
)

const (
	ActionProposalSchema       = "uiai.evidence_action_proposal.v1"
	ActionPreviewSchema        = "uiai.evidence_action_preview.v1"
	ActionResultSchema         = "uiai.evidence_action_result.v1"
	ReviewThreadSchema         = "uiai.evidence_review_thread.v1"
	ActionReconciliationSchema = "uiai.evidence_action_reconciliation.v1"

	MaxContractBytes = 1 << 20
	MaxRefs          = 256
	MaxEffects       = 128
	MaxEntries       = 1024
	MaxMessageRunes  = 4000
)

type ScopeBinding = evidencejudge.ScopeBinding

type ActionType string

const (
	ActionInspect      ActionType = "inspect"
	ActionLink         ActionType = "link"
	ActionCapture      ActionType = "capture"
	ActionReproof      ActionType = "reproof"
	ActionFollowUp     ActionType = "follow_up"
	ActionAdjudication ActionType = "adjudication"
	ActionShare        ActionType = "share"
	ActionExport       ActionType = "export"
)

type SideEffectClass string

const (
	EffectReadOnly         SideEffectClass = "read_only"
	EffectEvidenceAppend   SideEffectClass = "evidence_append"
	EffectFocusaHandoff    SideEffectClass = "focusa_handoff"
	EffectExternalMutation SideEffectClass = "external_mutation"
)

type SourceTrust string

const (
	SourceVerified          SourceTrust = "verified"
	SourceImportedUntrusted SourceTrust = "imported_untrusted"
)

type AutonomousEligibility string

const (
	AutonomousEligible   AutonomousEligibility = "autonomous_eligible"
	AutonomousIneligible AutonomousEligibility = "autonomous_ineligible"
)

type ActionProposal struct {
	Schema                   string                `json:"schema"`
	ProposalID               string                `json:"proposal_id"`
	Scope                    ScopeBinding          `json:"scope"`
	ArtifactRef              string                `json:"artifact_ref"`
	ArtifactSHA256           string                `json:"artifact_sha256"`
	TargetAcceptanceAtomRefs []string              `json:"target_acceptance_atom_refs"`
	CapabilitySnapshotRef    string                `json:"capability_snapshot_ref"`
	CapabilitySnapshotSHA256 string                `json:"capability_snapshot_sha256"`
	OperationRef             string                `json:"operation_ref"`
	OperationVersion         string                `json:"operation_version"`
	Action                   ActionType            `json:"action"`
	ActorRef                 string                `json:"actor_ref"`
	DelegationRef            string                `json:"delegation_ref"`
	IdempotencyKey           string                `json:"idempotency_key"`
	ExpectedStateRef         string                `json:"expected_state_ref"`
	ExpectedStateVersion     uint64                `json:"expected_state_version"`
	ExpiresAt                time.Time             `json:"expires_at"`
	SideEffect               SideEffectClass       `json:"side_effect"`
	SourceTrust              SourceTrust           `json:"source_trust"`
	HumanReviewMandated      bool                  `json:"human_review_mandated"`
	AutonomousEligibility    AutonomousEligibility `json:"autonomous_eligibility"`
}

type OperationApproval struct {
	RegistryRef              string    `json:"registry_ref"`
	RegistrySHA256           string    `json:"registry_sha256"`
	OperationRef             string    `json:"operation_ref"`
	OperationVersion         string    `json:"operation_version"`
	CapabilitySnapshotRef    string    `json:"capability_snapshot_ref"`
	CapabilitySnapshotSHA256 string    `json:"capability_snapshot_sha256"`
	Approved                 bool      `json:"approved"`
	ExpiresAt                time.Time `json:"expires_at"`
}

type ExpectedEffect struct {
	EffectRef string `json:"effect_ref"`
	TargetRef string `json:"target_ref"`
	Kind      string `json:"kind"`
}

type RiskClass string

const (
	RiskLow      RiskClass = "low"
	RiskModerate RiskClass = "moderate"
	RiskHigh     RiskClass = "high"
)

type ActionPreview struct {
	Schema                   string           `json:"schema"`
	PreviewID                string           `json:"preview_id"`
	ProposalRef              string           `json:"proposal_ref"`
	ProposalSHA256           string           `json:"proposal_sha256"`
	NormalizedRequestSHA256  string           `json:"normalized_request_sha256"`
	CapabilitySnapshotRef    string           `json:"capability_snapshot_ref"`
	CapabilitySnapshotSHA256 string           `json:"capability_snapshot_sha256"`
	TargetRefs               []string         `json:"target_refs"`
	ExpectedEffects          []ExpectedEffect `json:"expected_effects"`
	ExpectedStateVersion     uint64           `json:"expected_state_version"`
	Risk                     RiskClass        `json:"risk"`
	ConfirmationRequired     bool             `json:"confirmation_required"`
	ExpiresAt                time.Time        `json:"expires_at"`
	AntiReplayNonce          string           `json:"anti_replay_nonce"`
}

type ActionConfirmation struct {
	PreviewRef    string    `json:"preview_ref"`
	PreviewSHA256 string    `json:"preview_sha256"`
	Nonce         string    `json:"nonce"`
	ActorRef      string    `json:"actor_ref"`
	ConfirmedAt   time.Time `json:"confirmed_at"`
}

type ActionStatus string

const (
	StatusSucceeded        ActionStatus = "succeeded"
	StatusRejected         ActionStatus = "rejected"
	StatusBlocked          ActionStatus = "blocked"
	StatusPartiallyApplied ActionStatus = "partially_applied"
	StatusOutcomeUnknown   ActionStatus = "outcome_unknown"
)

type AppliedEffect struct {
	EffectRef       string `json:"effect_ref"`
	TargetRef       string `json:"target_ref"`
	ObservedVersion uint64 `json:"observed_version"`
	EvidenceRef     string `json:"evidence_ref"`
}

type Compensation struct {
	EffectRef       string `json:"effect_ref"`
	CompensationRef string `json:"compensation_ref"`
	Status          string `json:"status"`
	EvidenceRef     string `json:"evidence_ref,omitempty"`
}

type ActionResult struct {
	Schema                string          `json:"schema"`
	ResultID              string          `json:"result_id"`
	ProposalRef           string          `json:"proposal_ref"`
	ProposalSHA256        string          `json:"proposal_sha256"`
	PreviewRef            string          `json:"preview_ref,omitempty"`
	PreviewSHA256         string          `json:"preview_sha256,omitempty"`
	IdempotencyKey        string          `json:"idempotency_key"`
	Status                ActionStatus    `json:"status"`
	AppliedEffects        []AppliedEffect `json:"applied_effects,omitempty"`
	Compensations         []Compensation  `json:"compensations,omitempty"`
	ProviderReceiptRefs   []string        `json:"provider_receipt_refs,omitempty"`
	AuthoritativeStateRef string          `json:"authoritative_state_ref,omitempty"`
	ObservedStateVersion  uint64          `json:"observed_state_version"`
	ObservedAt            time.Time       `json:"observed_at"`
	ErrorCodes            []string        `json:"error_codes,omitempty"`
}

type ReconciliationState string

const (
	ReconciliationPending    ReconciliationState = "pending"
	ReconciliationConsistent ReconciliationState = "consistent"
	ReconciliationConflict   ReconciliationState = "conflict"
	ReconciliationBlocked    ReconciliationState = "blocked"
)

type ActionReconciliation struct {
	Schema                      string              `json:"schema"`
	ReconciliationID            string              `json:"reconciliation_id"`
	ResultRef                   string              `json:"result_ref"`
	ResultSHA256                string              `json:"result_sha256"`
	IdempotencyKey              string              `json:"idempotency_key"`
	AuthoritativeInspectionRefs []string            `json:"authoritative_inspection_refs"`
	State                       ReconciliationState `json:"state"`
	RetryPermitted              bool                `json:"retry_permitted"`
	RetryReasonCode             string              `json:"retry_reason_code,omitempty"`
	ReconciledAt                time.Time           `json:"reconciled_at"`
}

type ReviewEntryKind string

const (
	EntryComment            ReviewEntryKind = "comment"
	EntryAnnotation         ReviewEntryKind = "annotation"
	EntrySuggestion         ReviewEntryKind = "suggestion"
	EntryDecision           ReviewEntryKind = "decision"
	EntrySupersession       ReviewEntryKind = "supersession"
	EntryNotificationIntent ReviewEntryKind = "notification_intent"
)

type ReviewDecision string

const (
	DecisionNone             ReviewDecision = "none"
	DecisionApproved         ReviewDecision = "approved"
	DecisionChangesRequested ReviewDecision = "changes_requested"
	DecisionRejected         ReviewDecision = "rejected"
	DecisionBlocked          ReviewDecision = "blocked"
)

type ReviewEntry struct {
	EntryID       string          `json:"entry_id"`
	Revision      uint64          `json:"revision"`
	Kind          ReviewEntryKind `json:"kind"`
	Decision      ReviewDecision  `json:"decision"`
	Message       string          `json:"message"`
	ItemRef       string          `json:"item_ref"`
	AtomRefs      []string        `json:"atom_refs"`
	ArtifactRef   string          `json:"artifact_ref"`
	CitationRefs  []string        `json:"citation_refs,omitempty"`
	ActorRef      string          `json:"actor_ref"`
	DelegationRef string          `json:"delegation_ref"`
	OccurredAt    time.Time       `json:"occurred_at"`
	ProvenanceRef string          `json:"provenance_ref"`
	SupersedesRef string          `json:"supersedes_ref,omitempty"`
	Imported      bool            `json:"imported"`
	SourceTrust   SourceTrust     `json:"source_trust"`
}

type ReviewThread struct {
	Schema                string                `json:"schema"`
	ThreadID              string                `json:"thread_id"`
	Scope                 ScopeBinding          `json:"scope"`
	ArtifactRef           string                `json:"artifact_ref"`
	ArtifactSHA256        string                `json:"artifact_sha256"`
	AtomRefs              []string              `json:"atom_refs"`
	Revision              uint64                `json:"revision"`
	Entries               []ReviewEntry         `json:"entries"`
	SourceTrust           SourceTrust           `json:"source_trust"`
	HumanReviewMandated   bool                  `json:"human_review_mandated"`
	AutonomousEligibility AutonomousEligibility `json:"autonomous_eligibility"`
}
