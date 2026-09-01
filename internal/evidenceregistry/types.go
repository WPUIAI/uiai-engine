package evidenceregistry

import (
	"errors"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
)

const (
	RegistrySchemaV1 = "uiai.evidence_registry.v1"
	PageSchemaV1     = "uiai.evidence_registry_page.v1"
	EdgeSchemaV1     = "uiai.evidence_registry_edge.v1"
	ClosureSchemaV1  = "uiai.evidence_closure_projection.v1"

	MaxPageSize   = 200
	MaxQueryRunes = 512
)

var (
	ErrConfig           = errors.New("evidence registry configuration invalid")
	ErrProjectMismatch  = errors.New("evidence registry project mismatch")
	ErrInputInvalid     = errors.New("evidence registry input invalid")
	ErrCursorInvalid    = errors.New("evidence registry cursor invalid")
	ErrNotFound         = errors.New("evidence registry artifact not found")
	ErrIndexCorrupt     = errors.New("evidence registry index corrupt")
	ErrIndexUnavailable = errors.New("evidence registry index unavailable")
)

type IndexState string

const (
	IndexReady      IndexState = "ready"
	IndexStale      IndexState = "stale_index"
	IndexCorrupt    IndexState = "corrupt_index"
	IndexRebuilding IndexState = "rebuilding"
	IndexDegraded   IndexState = "degraded"
)

type RelationType string

const (
	RelationWorkItem       RelationType = "artifact_work_item"
	RelationAcceptanceAtom RelationType = "artifact_acceptance_atom"
	RelationCompletionCase RelationType = "artifact_completion_case"
	RelationReceipt        RelationType = "artifact_receipt"
	RelationRelated        RelationType = "artifact_related"
	RelationSupersedes     RelationType = "artifact_supersedes"
)

type ClosurePosture string

const (
	ClosureIneligible ClosurePosture = "ineligible"
	ClosureBlocked    ClosurePosture = "blocked"
	ClosureStale      ClosurePosture = "stale"
	ClosureEligible   ClosurePosture = "eligible_for_closure"
	ClosureCompleted  ClosurePosture = "completed"
	ClosureReopened   ClosurePosture = "reopened"
)

type AcceptanceState string

const (
	AcceptanceMissing       AcceptanceState = "missing"
	AcceptancePending       AcceptanceState = "pending"
	AcceptanceAccepted      AcceptanceState = "accepted"
	AcceptanceRejected      AcceptanceState = "rejected"
	AcceptanceBlocked       AcceptanceState = "blocked"
	AcceptanceStale         AcceptanceState = "stale"
	AcceptanceIndeterminate AcceptanceState = "indeterminate"
)

type Config struct {
	Path        string
	ProjectRef  string
	BusyTimeout time.Duration
	Now         func() time.Time
}

type IndexInput struct {
	Manifest                evidenceartifact.Manifest
	ManifestSHA256          string
	CompletionCaseRef       string
	CompletionContractRef   string
	CompletionDecisionRef   string
	ProviderCloseReceiptRef string
	ReopenRef               string
	SettlementPosture       string
	Acceptances             []AcceptanceBinding
	ObservedAt              time.Time
}

type AcceptanceBinding struct {
	AcceptanceAtomRef string          `json:"acceptance_atom_ref"`
	Revision          string          `json:"revision"`
	State             AcceptanceState `json:"state"`
	VerifierClass     string          `json:"verifier_class,omitempty"`
	VerifierRefs      []string        `json:"verifier_refs,omitempty"`
	DecisionRef       string          `json:"decision_ref,omitempty"`
	ReceiptRef        string          `json:"receipt_ref,omitempty"`
	Fresh             bool            `json:"fresh"`
	ScopeMatched      bool            `json:"scope_matched"`
}

type IndexResult struct {
	ArtifactRef   string `json:"artifact_ref"`
	Revision      uint64 `json:"revision"`
	IndexRevision uint64 `json:"index_revision"`
	InputSHA256   string `json:"input_sha256"`
	Deduplicated  bool   `json:"deduplicated"`
}

type WorkItemSnapshot struct {
	ProviderSurface         string   `json:"provider_surface"`
	WorkItemRef             string   `json:"work_item_ref"`
	ItemID                  string   `json:"item_id"`
	ItemType                string   `json:"item_type"`
	Title                   string   `json:"title"`
	Description             string   `json:"description,omitempty"`
	DescriptionRef          string   `json:"description_ref,omitempty"`
	DescriptionSHA256       string   `json:"description_sha256,omitempty"`
	Revision                string   `json:"revision"`
	Digest                  string   `json:"digest"`
	StatusAtCapture         string   `json:"status_at_capture"`
	ParentRefs              []string `json:"parent_refs"`
	DependencyRefs          []string `json:"dependency_refs"`
	BlockerRefs             []string `json:"blocker_refs"`
	AcceptanceAtomRefs      []string `json:"acceptance_atom_refs"`
	EvidenceRequirementRefs []string `json:"evidence_requirement_refs"`
	ReviewRequirementRefs   []string `json:"review_requirement_refs"`
	ClosurePosture          string   `json:"closure_posture"`
}

type ArtifactRow struct {
	ArtifactRef         string                              `json:"artifact_ref"`
	Revision            uint64                              `json:"revision"`
	ManifestSHA256      string                              `json:"manifest_sha256"`
	BundleSHA256        string                              `json:"bundle_sha256,omitempty"`
	Title               string                              `json:"title"`
	Summary             string                              `json:"summary,omitempty"`
	Kinds               []string                            `json:"kinds,omitempty"`
	ProjectRef          string                              `json:"project_ref"`
	WorkstreamRef       string                              `json:"workstream_ref"`
	WorksetRef          string                              `json:"workset_ref"`
	CallGraphRef        string                              `json:"callgraph_ref"`
	WorkpointRef        string                              `json:"workpoint_ref"`
	FirstWorkItemRef    string                              `json:"first_work_item_ref,omitempty"`
	FirstWorkItemType   string                              `json:"first_work_item_type,omitempty"`
	FirstWorkItemTitle  string                              `json:"first_work_item_title,omitempty"`
	WorkItemCount       uint32                              `json:"work_item_count"`
	AcceptanceTotal     uint32                              `json:"acceptance_total"`
	AcceptanceAccepted  uint32                              `json:"acceptance_accepted"`
	Verification        evidenceartifact.VerificationStatus `json:"verification"`
	Access              evidenceartifact.AccessClass        `json:"access"`
	Redaction           evidenceartifact.RedactionState     `json:"redaction"`
	Closure             ClosurePosture                      `json:"closure"`
	CapturedAt          time.Time                           `json:"captured_at"`
	FreshnessObservedAt time.Time                           `json:"freshness_observed_at"`
	PWAPath             string                              `json:"pwa_path,omitempty"`
}

type Edge struct {
	Schema            string       `json:"schema"`
	ProjectRef        string       `json:"project_ref"`
	SourceRef         string       `json:"source_ref"`
	SourceRevision    string       `json:"source_revision,omitempty"`
	TargetRef         string       `json:"target_ref"`
	TargetRevision    string       `json:"target_revision,omitempty"`
	Relation          RelationType `json:"relation"`
	ProvenanceReceipt string       `json:"provenance_receipt,omitempty"`
	ObservedAt        time.Time    `json:"observed_at"`
}

type EdgeDirection string

const (
	DirectionForward EdgeDirection = "forward"
	DirectionReverse EdgeDirection = "reverse"
)

type Query struct {
	ProjectRef        string
	Text              string
	WorkstreamRef     string
	WorksetRef        string
	WorkpointRef      string
	WorkItemRef       string
	WorkItemType      string
	AcceptanceAtomRef string
	Verification      string
	Access            string
	Closure           string
	Cursor            string
	PageSize          uint32
	SortDescending    bool
}

type Page struct {
	Schema        string        `json:"schema"`
	ProjectRef    string        `json:"project_ref"`
	Rows          []ArtifactRow `json:"rows"`
	NextCursor    string        `json:"next_cursor,omitempty"`
	PageSize      uint32        `json:"page_size"`
	TotalPosture  string        `json:"total_posture"`
	TotalCount    uint64        `json:"total_count,omitempty"`
	IndexRevision uint64        `json:"index_revision"`
	IndexState    IndexState    `json:"index_state"`
	ObservedAt    time.Time     `json:"observed_at"`
}

type ClosureProjection struct {
	Schema                  string         `json:"schema"`
	ProjectRef              string         `json:"project_ref"`
	WorkItemRef             string         `json:"work_item_ref"`
	CompletionCaseRef       string         `json:"completion_case_ref,omitempty"`
	CompletionContractRef   string         `json:"completion_contract_ref,omitempty"`
	RequiredAtoms           uint32         `json:"required_atoms"`
	AcceptedAtoms           uint32         `json:"accepted_atoms"`
	BlockedAtoms            uint32         `json:"blocked_atoms"`
	StaleAtoms              uint32         `json:"stale_atoms"`
	Posture                 ClosurePosture `json:"posture"`
	CompletionDecisionRef   string         `json:"completion_decision_ref,omitempty"`
	ProviderCloseReceiptRef string         `json:"provider_close_receipt_ref,omitempty"`
	ReopenRef               string         `json:"reopen_ref,omitempty"`
	SettlementPosture       string         `json:"settlement_posture,omitempty"`
	ObservedAt              time.Time      `json:"observed_at"`
}

type IndexStatus struct {
	Schema          string     `json:"schema"`
	ProjectRef      string     `json:"project_ref"`
	State           IndexState `json:"state"`
	Revision        uint64     `json:"revision"`
	ArtifactCount   uint64     `json:"artifact_count"`
	EdgeCount       uint64     `json:"edge_count"`
	SourceCursor    string     `json:"source_cursor,omitempty"`
	RebuildCursor   string     `json:"rebuild_cursor,omitempty"`
	StaleReason     string     `json:"stale_reason,omitempty"`
	LastIntegrityAt time.Time  `json:"last_integrity_at,omitempty"`
}
