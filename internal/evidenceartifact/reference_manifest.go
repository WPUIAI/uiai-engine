package evidenceartifact

// Reference digest placeholders shared by the reference manifest fixture and
// digest scenario. They are literal placeholder hashes, not secrets.
const (
	ReferenceDigestA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	ReferenceDigestB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	ReferenceDigestC = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
)

// ReferenceEvidenceManifest returns the canonical, valid reference manifest
// used by producer tests and the CG-05 capture digest scenario. Single source
// of truth: tests must not maintain a private copy.
func ReferenceEvidenceManifest() Manifest {
	return Manifest{
		Schema:     SchemaManifestV1,
		ArtifactID: "artifact:epwa-001",
		Revision:   1,
		Title:      "Evidence artifact contract proof",
		Summary:    "Bound immutable evidence for autonomous review.",
		Kinds:      []string{"diagnostic", "structured_data"},
		CapturedAt: "2026-08-29T12:00:00Z",
		CreatedAt:  "2026-08-29T12:00:01Z",
		Scope: Scope{
			Project:    ProjectBinding{ProjectRef: "project:uiai-engine", Fingerprint: ReferenceDigestA, WorkingSubpathRef: "subpath:primary", State: BindingMatched},
			Workstream: WorkstreamBinding{WorkstreamRef: "workstream:epwa", State: BindingMatched},
			Workset: WorksetBinding{
				WorksetRef: "workset:epwa-t01", Revision: 1, Digest: ReferenceDigestA, MembershipRef: "membership:epwa-t01",
				RequirementRefs: []string{"requirement:manifest", "requirement:review"}, DispositionRefs: []string{"disposition:open"}, State: BindingMatched,
			},
			CallGraph: CallGraphBinding{
				DefinitionRef: "callgraph:epwa", DefinitionRevision: 1, RunRef: "callgraph-run:001", FrameRef: "frame:publish",
				NodeRef: "node:evidence", ItemRef: "callgraph-item:t01", PathRef: "path:executor", Attempt: 1, Generation: 1, Cycle: 0, State: BindingMatched,
			},
			Workpoint: WorkpointBinding{
				WorkpointRef: "workpoint:epwa-t01", Revision: 1, CheckpointRef: "checkpoint:001",
				CurrentActionIntentRef: "intent:publish-contract", State: BindingMatched,
			},
			Autonomy: AutonomyBinding{
				Mode: "bounded_autonomous", PolicyRef: "autonomy-policy:epwa", WorkLoopRef: "work-loop:epwa", RunRef: "autonomy-run:001",
				RunStatus: "executing", AgentTeamPlanRef: "agent-team:epwa", ExecutorAssignmentRef: "assignment:executor",
				VerifierAssignmentRefs: []string{"assignment:judge"}, CapabilityDigestRefs: []string{"capability-digest:executor"},
				BudgetPolicyRef: "budget:epwa", ResourcePolicyRef: "resource:normal", RetryPolicyRef: "retry:bounded",
				FailoverPolicyRef: "failover:independent", CooldownPolicyRef: "cooldown:default", CircuitBreakerPolicyRef: "circuit-breaker:epwa",
				ReviewPostureRef: "review-posture:pending", ClosurePostureRef: "closure-posture:blocked", EventCursorRef: "cursor:001",
				ContinuationRefs: []string{"continuation:workpoint"},
			},
			WorkItems: []WorkItemBinding{
				{
					ProviderSurface: "bd", WorkItemRef: "work-item:focusa-a1", ItemID: "focusa-a1", ItemType: "task",
					Title: "Implement artifact contract", Description: "Create immutable evidence manifest types.", Revision: "1", Digest: ReferenceDigestB,
					StatusAtCapture: "in_progress", AcceptanceAtomRefs: []string{"atom:types"}, EvidenceRequirementRefs: []string{"evidence:test"},
					ReviewRequirementRefs: []string{"review-requirement:llm"}, ClosurePosture: "review_pending",
				},
				{
					ProviderSurface: "br", WorkItemRef: "work-item:focusa-a2", ItemID: "focusa-a2", ItemType: "task",
					Title: "Verify artifact contract", Description: "Run contract and golden tests.", Revision: "2", Digest: ReferenceDigestC,
					StatusAtCapture: "open", DependencyRefs: []string{"work-item:focusa-a1"}, AcceptanceAtomRefs: []string{"atom:tests"},
					EvidenceRequirementRefs: []string{"evidence:go-test"}, ReviewRequirementRefs: []string{"review-requirement:llm"}, ClosurePosture: "evidence_missing",
				},
			},
			TrajectoryRef: "trajectory:epwa", AssignmentRefs: []string{"assignment:executor", "assignment:judge"},
			OperationRefs: []string{"operation:manifest-create"}, OntologyRefs: []string{"object:evidence-artifact"}, RehydrateRefs: []string{"rehydrate:epwa"},
		},
		Authority: Authority{
			ProducerRef: "agent:executor", SourceAuthorityRef: "authority:uiai", EvidenceAuthorityRef: "authority:evidence",
			CompletionAuthorityRef: "authority:completion", ReviewerPolicyRef: "review-policy:autonomous", Posture: PostureCanonical,
		},
		Claims: []Claim{
			{ClaimID: "claim:manifest-valid", Summary: "The manifest satisfies the frozen contract.", Status: ClaimActual, AcceptanceAtomRefs: []string{"atom:types"}, EvidenceRefs: []string{"asset:proof"}, ReviewRequirementRefs: []string{"review-requirement:llm"}},
		},
		Assets: []Asset{
			{AssetID: "asset:proof", Kind: "structured_data", MediaType: "application/json", Path: "assets/proof.json", SHA256: ReferenceDigestC, ByteSize: 512, CapturedAt: "2026-08-29T12:00:00Z", SourceRef: "source:go-test", ClaimRefs: []string{"claim:manifest-valid"}, VerificationClass: VerificationActual, RedactionState: RedactionPublicSafe},
		},
		Provenance: Provenance{
			SourceRefs: []string{"source:go-test"}, EnvironmentRefs: []string{"environment:ovh"}, OmissionRefs: []string{},
			Custody: []CustodyEvent{{EventID: "custody:1", Action: "captured", ActorRef: "agent:executor", InstanceRef: "instance:uiai", InputRefs: []string{"source:go-test"}, OutputRefs: []string{"asset:proof"}, OccurredAt: "2026-08-29T12:00:00Z"}},
		},
		Verification: Verification{Status: VerificationPending, ReviewCaseRef: "review-case:epwa-t01", VerifierRefs: []string{"agent:judge"}, JudgeResultRefs: []string{}, DecisionRefs: []string{}},
		Security:     Security{PolicyRef: StrictSecurityPolicyV1, InspectionReceiptRefs: []string{}, SanitizationRefs: []string{}, RedactionRefs: []string{"redaction:proof"}},
		ReceiptRefs:  []string{"receipt:capture"},
		Policy:       Policy{AccessClass: AccessPrivateTeam, RedactionState: RedactionPublicSafe, Audience: "project_reviewers", RetentionClass: RetentionWorkstream, PolicyRefs: []string{"policy:evidence"}},
		Integrity:    Integrity{Algorithm: "sha256", BundleSHA256: ReferenceDigestB},
		Links:        Links{PWAPath: "pwa/index.html", ManifestPath: "manifest.json", RelatedRefs: []string{"artifact:parent"}},
	}
}
