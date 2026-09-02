package evidencepwa

import "time"

const (
	ProjectionSchema = "uiai.evidence_pwa_projection.v1"

	MaxProjectionBytes = 1 << 20
	MaxSections        = 5
	MaxClaims          = 512
	MaxAssets          = 512
	MaxCitations       = 1024
	MaxTimelineEntries = 2048
	MaxWarnings        = 128
	MaxRefs            = 512
	MaxTitleRunes      = 240
	MaxSummaryRunes    = 4000
	MaxPageSize        = 200
)

type SectionID string

const (
	SectionOverview  SectionID = "overview"
	SectionEvidence  SectionID = "evidence"
	SectionTimeline  SectionID = "timeline"
	SectionInspect   SectionID = "inspect"
	SectionDeveloper SectionID = "developer"
)

type AvailabilityState string

const (
	AvailabilityLoading     AvailabilityState = "loading"
	AvailabilityReady       AvailabilityState = "ready"
	AvailabilityUnavailable AvailabilityState = "unavailable"
	AvailabilityBlocked     AvailabilityState = "blocked"
	AvailabilityCorrupt     AvailabilityState = "corrupt"
	AvailabilityStale       AvailabilityState = "stale"
	AvailabilityRedacted    AvailabilityState = "redacted"
	AvailabilityDegraded    AvailabilityState = "degraded"
)

type AccessPosture string

const (
	AccessLocalhost       AccessPosture = "localhost"
	AccessLAN             AccessPosture = "lan"
	AccessTailnet         AccessPosture = "tailnet"
	AccessPrivate         AccessPosture = "private"
	AccessUnlisted        AccessPosture = "unlisted"
	AccessPublicSafe      AccessPosture = "public_safe"
	AccessOfflineSnapshot AccessPosture = "offline_snapshot"
)

type RedactionState string

const (
	RedactionNotRequired      RedactionState = "not_required"
	RedactionApplied          RedactionState = "applied"
	RedactionPartiallyApplied RedactionState = "partially_applied"
	RedactionBlocked          RedactionState = "blocked"
	RedactionUnknown          RedactionState = "unknown"
)

type InteractionMode string

const (
	InteractionReadOnly             InteractionMode = "read_only"
	InteractionAuthenticatedHandoff InteractionMode = "authenticated_handoff"
)

type ScopeBinding struct {
	ProjectRef    string               `json:"project_ref"`
	WorkstreamRef string               `json:"workstream_ref"`
	WorksetRef    string               `json:"workset_ref"`
	CallGraphRef  string               `json:"callgraph_ref"`
	WorkpointRef  string               `json:"workpoint_ref"`
	WorkItemRef   string               `json:"work_item_ref"`
	WorkItems     []WorkItemProjection `json:"work_items,omitempty"`
}

type ArtifactBinding struct {
	ArtifactRef    string       `json:"artifact_ref"`
	Revision       uint64       `json:"revision"`
	ManifestSHA256 string       `json:"manifest_sha256"`
	BundleSHA256   string       `json:"bundle_sha256"`
	Scope          ScopeBinding `json:"scope"`
}

type Section struct {
	ID      SectionID `json:"id"`
	Heading string    `json:"heading"`
}

type Claim struct {
	ClaimID     string   `json:"claim_id"`
	Statement   string   `json:"statement"`
	Posture     string   `json:"posture"`
	CitationIDs []string `json:"citation_ids,omitempty"`
	AssetIDs    []string `json:"asset_ids,omitempty"`
}

type Asset struct {
	AssetID  string `json:"asset_id"`
	Ref      string `json:"ref"`
	SHA256   string `json:"sha256"`
	MIME     string `json:"mime"`
	Modality string `json:"modality"`
	Bytes    uint64 `json:"bytes"`
}

type Citation struct {
	CitationID string `json:"citation_id"`
	SourceRef  string `json:"source_ref"`
	SHA256     string `json:"sha256"`
	Locator    string `json:"locator"`
}

type TimelineEntry struct {
	EntryID    string    `json:"entry_id"`
	OccurredAt time.Time `json:"occurred_at"`
	EventType  string    `json:"event_type"`
	Refs       []string  `json:"refs,omitempty"`
}

type Redaction struct {
	State              RedactionState `json:"state"`
	EvidenceRef        string         `json:"evidence_ref,omitempty"`
	PrivateRefCount    uint64         `json:"private_ref_count"`
	SecretFindingCount uint64         `json:"secret_finding_count"`
	PIIFindingCount    uint64         `json:"pii_finding_count"`
}

type Warning struct {
	Code         string   `json:"code"`
	Message      string   `json:"message"`
	EvidenceRefs []string `json:"evidence_refs,omitempty"`
}

type RelativeLink struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

type PageInfo struct {
	Cursor     string `json:"cursor,omitempty"`
	NextCursor string `json:"next_cursor,omitempty"`
	PageSize   uint32 `json:"page_size"`
	TotalCount uint64 `json:"total_count"`
}

type Projection struct {
	Schema              string            `json:"schema"`
	ProjectionID        string            `json:"projection_id"`
	Artifact            ArtifactBinding   `json:"artifact"`
	Title               string            `json:"title"`
	Summary             string            `json:"summary"`
	Sections            []Section         `json:"sections"`
	Claims              []Claim           `json:"claims,omitempty"`
	Assets              []Asset           `json:"assets,omitempty"`
	Citations           []Citation        `json:"citations,omitempty"`
	Timeline            []TimelineEntry   `json:"timeline,omitempty"`
	InspectionRefs      []string          `json:"inspection_refs,omitempty"`
	SecurityRefs        []string          `json:"security_refs,omitempty"`
	CustodyRefs         []string          `json:"custody_refs,omitempty"`
	AttestationRefs     []string          `json:"attestation_refs,omitempty"`
	TrustRefs           []string          `json:"trust_refs,omitempty"`
	OmissionRefs        []string          `json:"omission_refs,omitempty"`
	RelatedArtifactRefs []string          `json:"related_artifact_refs,omitempty"`
	ReceiptRefs         []string          `json:"receipt_refs,omitempty"`
	Availability        AvailabilityState `json:"availability"`
	Access              AccessPosture     `json:"access"`
	Redaction           Redaction         `json:"redaction"`
	FederationPosture   string            `json:"federation_posture"`
	FreshnessObservedAt time.Time         `json:"freshness_observed_at"`
	Warnings            []Warning         `json:"warnings,omitempty"`
	Page                PageInfo          `json:"page"`
	Links               []RelativeLink    `json:"links,omitempty"`
	Interaction         InteractionMode   `json:"interaction"`
	HandoffRef          string            `json:"handoff_ref,omitempty"`
}
