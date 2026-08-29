package evidenceartifact

const (
	SchemaManifestV1 = "uiai.evidence_artifact_manifest.v1"

	MaxManifestBytes    = 1024 * 1024
	MaxTitleRunes       = 240
	MaxDescriptionRunes = 8000
	MaxSummaryRunes     = 2000
	MaxRefRunes         = 512
	MaxKinds            = 32
	MaxRefsPerList      = 512
	MaxWorkItems        = 256
	MaxClaims           = 128
	MaxAssets           = 512
	MaxCustodyEvents    = 512
	MaxReceiptRefs      = 256
	MaxRelatedRefs      = 256
	MaxPolicyRefs       = 64
)

type Posture string

const (
	PostureCanonical Posture = "canonical"
	PostureAdvisory  Posture = "advisory"
	PostureDegraded  Posture = "degraded"
	PostureBlocked   Posture = "blocked"
	PostureStale     Posture = "stale"
)

type BindingState string

const (
	BindingMatched  BindingState = "matched"
	BindingMissing  BindingState = "missing"
	BindingStale    BindingState = "stale"
	BindingMismatch BindingState = "mismatch"
)

type ClaimStatus string

const (
	ClaimActual    ClaimStatus = "actual"
	ClaimPartial   ClaimStatus = "partial"
	ClaimSurrogate ClaimStatus = "surrogate"
	ClaimBlocked   ClaimStatus = "blocked"
	ClaimMissing   ClaimStatus = "missing"
)

type VerificationStatus string

const (
	VerificationPending       VerificationStatus = "pending"
	VerificationPassed        VerificationStatus = "passed"
	VerificationFailed        VerificationStatus = "failed"
	VerificationBlocked       VerificationStatus = "blocked"
	VerificationIndeterminate VerificationStatus = "indeterminate"
	VerificationDisputed      VerificationStatus = "disputed"
)

type VerificationClass string

const (
	VerificationActual        VerificationClass = "actual"
	VerificationBlockedClass  VerificationClass = "blocked"
	VerificationSurrogate     VerificationClass = "surrogate"
	VerificationMissingNative VerificationClass = "missing_native"
)

type AccessClass string

const (
	AccessLocal       AccessClass = "local"
	AccessLAN         AccessClass = "lan"
	AccessTailnet     AccessClass = "tailnet"
	AccessPrivateTeam AccessClass = "private_team"
	AccessUnlisted    AccessClass = "unlisted"
	AccessPublicSafe  AccessClass = "public_safe"
)

type RedactionState string

const (
	RedactionNone       RedactionState = "none"
	RedactionRedacted   RedactionState = "redacted"
	RedactionBlocked    RedactionState = "blocked"
	RedactionPublicSafe RedactionState = "public_safe"
)

type RetentionClass string

const (
	RetentionEphemeral  RetentionClass = "ephemeral"
	RetentionProject    RetentionClass = "project"
	RetentionWorkstream RetentionClass = "workstream"
	RetentionRelease    RetentionClass = "release"
	RetentionLegalHold  RetentionClass = "legal_hold"
	RetentionCustom     RetentionClass = "custom"
)

type Manifest struct {
	Schema       string       `json:"schema"`
	ArtifactID   string       `json:"artifact_id"`
	Revision     uint64       `json:"revision"`
	Title        string       `json:"title"`
	Summary      string       `json:"summary,omitempty"`
	Kinds        []string     `json:"kinds"`
	CapturedAt   string       `json:"captured_at"`
	CreatedAt    string       `json:"created_at"`
	Scope        Scope        `json:"scope"`
	Authority    Authority    `json:"authority"`
	Claims       []Claim      `json:"claims"`
	Assets       []Asset      `json:"assets"`
	Provenance   Provenance   `json:"provenance"`
	Verification Verification `json:"verification"`
	ReceiptRefs  []string     `json:"receipt_refs"`
	Policy       Policy       `json:"policy"`
	Integrity    Integrity    `json:"integrity"`
	Links        Links        `json:"links"`
}

type Scope struct {
	Project        ProjectBinding    `json:"project"`
	Workstream     WorkstreamBinding `json:"workstream"`
	Workset        WorksetBinding    `json:"workset"`
	CallGraph      CallGraphBinding  `json:"callgraph"`
	Workpoint      WorkpointBinding  `json:"workpoint"`
	Autonomy       AutonomyBinding   `json:"autonomy"`
	WorkItems      []WorkItemBinding `json:"work_items"`
	TrajectoryRef  string            `json:"trajectory_ref,omitempty"`
	AssignmentRefs []string          `json:"assignment_refs"`
	OperationRefs  []string          `json:"operation_refs"`
	OntologyRefs   []string          `json:"ontology_refs"`
	RehydrateRefs  []string          `json:"rehydrate_refs"`
}

type ProjectBinding struct {
	ProjectRef        string       `json:"project_ref"`
	Fingerprint       string       `json:"fingerprint"`
	WorkingSubpathRef string       `json:"working_subpath_ref"`
	State             BindingState `json:"state"`
}

type WorkstreamBinding struct {
	WorkstreamRef string       `json:"workstream_ref"`
	State         BindingState `json:"state"`
}

type WorksetBinding struct {
	WorksetRef      string       `json:"workset_ref"`
	Revision        uint64       `json:"revision"`
	Digest          string       `json:"digest"`
	MembershipRef   string       `json:"membership_ref"`
	RequirementRefs []string     `json:"requirement_refs"`
	DispositionRefs []string     `json:"disposition_refs"`
	State           BindingState `json:"state"`
}

type CallGraphBinding struct {
	DefinitionRef      string       `json:"definition_ref"`
	DefinitionRevision uint64       `json:"definition_revision"`
	RunRef             string       `json:"run_ref"`
	FrameRef           string       `json:"frame_ref"`
	NodeRef            string       `json:"node_ref"`
	ItemRef            string       `json:"item_ref"`
	PathRef            string       `json:"path_ref,omitempty"`
	Attempt            uint64       `json:"attempt"`
	Generation         uint64       `json:"generation"`
	Cycle              uint64       `json:"cycle"`
	ParentFrameRef     string       `json:"parent_frame_ref,omitempty"`
	JoinRef            string       `json:"join_ref,omitempty"`
	CompensationRef    string       `json:"compensation_ref,omitempty"`
	State              BindingState `json:"state"`
}

type WorkpointBinding struct {
	WorkpointRef           string       `json:"workpoint_ref"`
	Revision               uint64       `json:"revision"`
	CheckpointRef          string       `json:"checkpoint_ref"`
	CurrentActionIntentRef string       `json:"current_action_intent_ref"`
	State                  BindingState `json:"state"`
}

type AutonomyBinding struct {
	Mode                     string   `json:"mode"`
	PolicyRef                string   `json:"policy_ref"`
	WorkLoopRef              string   `json:"work_loop_ref"`
	RunRef                   string   `json:"run_ref"`
	RunStatus                string   `json:"run_status"`
	AgentTeamPlanRef         string   `json:"agent_team_plan_ref"`
	ExecutorAssignmentRef    string   `json:"executor_assignment_ref"`
	VerifierAssignmentRefs   []string `json:"verifier_assignment_refs"`
	ArbitratorAssignmentRefs []string `json:"arbitrator_assignment_refs"`
	CapabilityDigestRefs     []string `json:"capability_digest_refs"`
	BudgetPolicyRef          string   `json:"budget_policy_ref"`
	ResourcePolicyRef        string   `json:"resource_policy_ref"`
	RetryPolicyRef           string   `json:"retry_policy_ref"`
	FailoverPolicyRef        string   `json:"failover_policy_ref"`
	CooldownPolicyRef        string   `json:"cooldown_policy_ref"`
	CircuitBreakerPolicyRef  string   `json:"circuit_breaker_policy_ref"`
	ReviewPostureRef         string   `json:"review_posture_ref"`
	ClosurePostureRef        string   `json:"closure_posture_ref"`
	EventCursorRef           string   `json:"event_cursor_ref"`
	ContinuationRefs         []string `json:"continuation_refs"`
}

type WorkItemBinding struct {
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

type Authority struct {
	ProducerRef            string  `json:"producer_ref"`
	SourceAuthorityRef     string  `json:"source_authority_ref"`
	EvidenceAuthorityRef   string  `json:"evidence_authority_ref"`
	CompletionAuthorityRef string  `json:"completion_authority_ref"`
	ReviewerPolicyRef      string  `json:"reviewer_policy_ref"`
	Posture                Posture `json:"posture"`
}

type Claim struct {
	ClaimID               string      `json:"claim_id"`
	Summary               string      `json:"summary"`
	Status                ClaimStatus `json:"status"`
	AcceptanceAtomRefs    []string    `json:"acceptance_atom_refs"`
	EvidenceRefs          []string    `json:"evidence_refs"`
	ReviewRequirementRefs []string    `json:"review_requirement_refs"`
}

type Asset struct {
	AssetID           string            `json:"asset_id"`
	Kind              string            `json:"kind"`
	MediaType         string            `json:"media_type"`
	Path              string            `json:"path"`
	SHA256            string            `json:"sha256"`
	ByteSize          int64             `json:"byte_size"`
	Width             int               `json:"width,omitempty"`
	Height            int               `json:"height,omitempty"`
	DurationMS        int64             `json:"duration_ms,omitempty"`
	CapturedAt        string            `json:"captured_at,omitempty"`
	SourceRef         string            `json:"source_ref"`
	ClaimRefs         []string          `json:"claim_refs"`
	VerificationClass VerificationClass `json:"verification_class"`
	RedactionState    RedactionState    `json:"redaction_state"`
	AltText           string            `json:"alt_text,omitempty"`
	TranscriptRef     string            `json:"transcript_ref,omitempty"`
}

type Provenance struct {
	SourceRefs      []string       `json:"source_refs"`
	EnvironmentRefs []string       `json:"environment_refs"`
	OmissionRefs    []string       `json:"omission_refs"`
	Custody         []CustodyEvent `json:"custody"`
}

type CustodyEvent struct {
	EventID     string   `json:"event_id"`
	Action      string   `json:"action"`
	ActorRef    string   `json:"actor_ref"`
	InstanceRef string   `json:"instance_ref"`
	InputRefs   []string `json:"input_refs"`
	OutputRefs  []string `json:"output_refs"`
	OccurredAt  string   `json:"occurred_at"`
}

type Verification struct {
	Status             VerificationStatus `json:"status"`
	ReviewCaseRef      string             `json:"review_case_ref"`
	VerifierRefs       []string           `json:"verifier_refs"`
	JudgeResultRefs    []string           `json:"judge_result_refs"`
	DecisionRefs       []string           `json:"decision_refs"`
	InformationSetHash string             `json:"information_set_hash,omitempty"`
}

type Policy struct {
	AccessClass    AccessClass    `json:"access_class"`
	RedactionState RedactionState `json:"redaction_state"`
	Audience       string         `json:"audience"`
	RetentionClass RetentionClass `json:"retention_class"`
	ExpiresAt      string         `json:"expires_at,omitempty"`
	PolicyRefs     []string       `json:"policy_refs"`
}

type Integrity struct {
	Algorithm      string `json:"algorithm"`
	ManifestSHA256 string `json:"manifest_sha256,omitempty"`
	BundleSHA256   string `json:"bundle_sha256,omitempty"`
}

type Links struct {
	PWAPath       string   `json:"pwa_path,omitempty"`
	ManifestPath  string   `json:"manifest_path,omitempty"`
	RelatedRefs   []string `json:"related_refs"`
	SupersedesRef string   `json:"supersedes_ref,omitempty"`
}
