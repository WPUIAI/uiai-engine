package epwadelivery

import "time"

const (
	Schema      = "uiai.epwa_delivery.v1"
	TruthNotice = "Artifact existence and EPWA delivery do not establish review, verification, completion, provider closure, settlement, or legal admissibility."
)

type State string
type ScopePosture string

type AccessPosture string

const (
	StateReady            State = "ready"
	StateBlocked          State = "blocked"
	StatePendingReconcile State = "pending_reconcile"
	StateUnavailable      State = "unavailable"
	StateCorrupt          State = "corrupt"
	StateStale            State = "stale"
	StateRedacted         State = "redacted"
	StateDegraded         State = "degraded"

	ScopeComplete ScopePosture = "complete"
	ScopeBlocked  ScopePosture = "blocked"

	AccessPublicSafe AccessPosture = "public_safe"
	AccessPrivate    AccessPosture = "private"
	AccessUnlisted   AccessPosture = "unlisted"
	AccessRedacted   AccessPosture = "redacted"
)

type ArtifactBinding struct {
	ArtifactRef    string `json:"artifact_ref"`
	Revision       uint64 `json:"revision"`
	ManifestSHA256 string `json:"manifest_sha256"`
	OutputSHA256   string `json:"output_sha256"`
}

type EPWABinding struct {
	PackageID        string        `json:"package_id"`
	ProjectionRef    string        `json:"projection_ref,omitempty"`
	ProjectionSHA256 string        `json:"projection_sha256,omitempty"`
	PackageRef       string        `json:"package_ref"`
	PackageSHA256    string        `json:"package_sha256"`
	RecordURL        string        `json:"record_url"`
	PortableURL      string        `json:"portable_url"`
	Access           AccessPosture `json:"access"`
}

type ScopeBinding struct {
	Posture       ScopePosture `json:"posture"`
	ProjectRef    string       `json:"project_ref,omitempty"`
	WorkstreamRef string       `json:"workstream_ref,omitempty"`
	WorksetRef    string       `json:"workset_ref,omitempty"`
	CallGraphRef  string       `json:"callgraph_ref,omitempty"`
	WorkpointRef  string       `json:"workpoint_ref,omitempty"`
	WorkItemRef   string       `json:"work_item_ref,omitempty"`
	ContinuityRef string       `json:"continuity_ref,omitempty"`
}

type Delivery struct {
	Schema         string          `json:"schema"`
	DeliveryID     string          `json:"delivery_id"`
	Revision       uint64          `json:"revision"`
	Producer       ProducerID      `json:"producer"`
	Artifact       ArtifactBinding `json:"artifact"`
	EPWA           EPWABinding     `json:"epwa"`
	Scope          ScopeBinding    `json:"scope"`
	State          State           `json:"state"`
	IdempotencyKey string          `json:"idempotency_key"`
	CreatedAt      time.Time       `json:"created_at"`
	ObservedAt     time.Time       `json:"observed_at"`
	RecoveryRef    string          `json:"recovery_ref,omitempty"`
	TruthNotice    string          `json:"truth_notice"`
}

type Input struct {
	Producer       ProducerID
	Artifact       ArtifactBinding
	EPWA           EPWABinding
	Scope          ScopeBinding
	State          State
	IdempotencyKey string
	CreatedAt      time.Time
	ObservedAt     time.Time
	RecoveryRef    string
}
