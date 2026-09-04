package evidencejudge

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestAssignJudgeGoldenDeterministicAndInputImmutable(t *testing.T) {
	request := assignmentTestRequest()
	first := assignmentTestCapability(t, "judge:a", "principal:judge-a", "instance:judge-a")
	second := assignmentTestCapability(t, "judge:b", "principal:judge-b", "instance:judge-b")
	candidates := []JudgeCapability{second, first}
	beforeRequest := assignmentJSON(t, request)
	beforeCandidates := assignmentJSON(t, candidates)

	assignment, err := AssignJudge(request, candidates)
	if err != nil {
		t.Fatalf("AssignJudge() error = %v", err)
	}
	if assignment.JudgeIdentityRef != "judge:a" || assignment.Status != AssignmentAppointed {
		t.Fatalf("assignment = %#v", assignment)
	}
	if err := ValidateJudgeAssignmentAt(assignment, request, first, request.RequestedAt.Add(time.Minute)); err != nil {
		t.Fatalf("ValidateJudgeAssignmentAt() error = %v", err)
	}
	if assignmentJSON(t, request) != beforeRequest || assignmentJSON(t, candidates) != beforeCandidates {
		t.Fatal("AssignJudge() mutated input")
	}

	reversed := append([]JudgeCapability(nil), candidates...)
	slices.Reverse(reversed)
	for i := 0; i < 30; i++ {
		replay, err := AssignJudge(request, reversed)
		if err != nil || replay.AssignmentSHA256 != assignment.AssignmentSHA256 || replay.JudgeIdentityRef != assignment.JudgeIdentityRef {
			t.Fatalf("replay %d = %s/%s, %v", i, replay.JudgeIdentityRef, replay.AssignmentSHA256, err)
		}
	}

	fixture := struct {
		Capability JudgeCapability        `json:"capability"`
		Request    JudgeAssignmentRequest `json:"request"`
		Assignment JudgeAssignment        `json:"assignment"`
	}{first, request, assignment}
	body := append([]byte(assignmentJSON(t, fixture)), '\n')
	fixturePath := "testdata/assignment.golden.json"
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(fixturePath, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(body, want) {
		t.Fatalf("golden drift\n got: %s\nwant: %s", body, want)
	}
}

func TestAssignJudgeFailsClosedForIndependenceCorrelationAndAmbiguity(t *testing.T) {
	request := assignmentTestRequest()

	collision := assignmentTestCapability(t, "judge:collision", request.ExecutorPrincipalRef, "instance:judge")
	if assignment, err := AssignJudge(request, []JudgeCapability{collision}); !errors.Is(err, ErrJudgeIndependenceViolation) || assignment.Status != AssignmentBlocked {
		t.Fatalf("principal collision = %#v, %v", assignment, err)
	}

	instanceCollision := assignmentTestCapability(t, "judge:collision", "principal:judge", request.ExecutorInstanceRef)
	if _, err := AssignJudge(request, []JudgeCapability{instanceCollision}); !errors.Is(err, ErrJudgeIndependenceViolation) {
		t.Fatalf("instance collision error = %v", err)
	}

	correlated := assignmentTestCapability(t, "judge:correlated", "principal:judge", "instance:judge")
	correlated.CorrelationRefs = []string{"provider:executor"}
	correlated = assignmentSealCapability(t, correlated)
	if _, err := AssignJudge(request, []JudgeCapability{correlated}); !errors.Is(err, ErrJudgeIndependenceViolation) {
		t.Fatalf("correlation error = %v", err)
	}

	duplicate := assignmentTestCapability(t, "judge:duplicate", "principal:a", "instance:a")
	other := duplicate
	other.PrincipalRef = "principal:b"
	other.InstanceRef = "instance:b"
	other = assignmentSealCapability(t, other)
	if _, err := AssignJudge(request, []JudgeCapability{duplicate, other}); !errors.Is(err, ErrJudgeAssignmentInvalid) {
		t.Fatalf("duplicate identity error = %v", err)
	}
}

func TestAssignJudgeEnforcesCapabilityCalibrationLeaseAndBudget(t *testing.T) {
	request := assignmentTestRequest()
	base := assignmentTestCapability(t, "judge:a", "principal:judge-a", "instance:judge-a")

	missingModality := base
	missingModality.SupportedModalities = []Modality{ModalityBoundedText}
	missingModality = assignmentSealCapability(t, missingModality)
	if _, err := AssignJudge(request, []JudgeCapability{missingModality}); !errors.Is(err, ErrJudgeCapabilityMismatch) {
		t.Fatalf("modality error = %v", err)
	}

	failedCalibration := base
	failedCalibration.Calibration.Status = CalibrationFailed
	failedCalibration = assignmentSealCapability(t, failedCalibration)
	if _, err := AssignJudge(request, []JudgeCapability{failedCalibration}); !errors.Is(err, ErrJudgeCalibrationInvalid) {
		t.Fatalf("calibration error = %v", err)
	}

	expiresDuringLease := base
	expiresDuringLease.ValidUntil = request.RequestedAt.Add(10 * time.Minute)
	expiresDuringLease = assignmentSealCapability(t, expiresDuringLease)
	if _, err := AssignJudge(request, []JudgeCapability{expiresDuringLease}); !errors.Is(err, ErrJudgeExpired) {
		t.Fatalf("lease expiry error = %v", err)
	}

	lowBudget := base
	lowBudget.MaxBudget.MaxTokens = request.Budget.MaxTokens - 1
	lowBudget = assignmentSealCapability(t, lowBudget)
	if _, err := AssignJudge(request, []JudgeCapability{lowBudget}); !errors.Is(err, ErrJudgeBudgetExceeded) {
		t.Fatalf("budget error = %v", err)
	}

	blocked := base
	blocked.Status = CapabilityBlocked
	blocked = assignmentSealCapability(t, blocked)
	if _, err := AssignJudge(request, []JudgeCapability{blocked}); !errors.Is(err, ErrJudgeCandidateUnavailable) {
		t.Fatalf("blocked error = %v", err)
	}
}

func TestAssignJudgeDeclaresAutonomousIneligibleWithoutLease(t *testing.T) {
	request := assignmentTestRequest()
	request.HumanAuthorityRequired = true
	capability := assignmentTestCapability(t, "judge:a", "principal:judge-a", "instance:judge-a")
	assignment, err := AssignJudge(request, []JudgeCapability{capability})
	if !errors.Is(err, ErrJudgeAutonomousIneligible) || assignment.Status != AssignmentAutonomousIneligible {
		t.Fatalf("assignment = %#v, error = %v", assignment, err)
	}
	if assignment.Lease.Generation != 0 || assignment.JudgeIdentityRef != "" || assignment.CapabilityDigest != "" {
		t.Fatal("autonomous-ineligible result issued judge authority")
	}
}

func TestAssignmentContractsRejectUnsafeStaleAndTamperedData(t *testing.T) {
	request := assignmentTestRequest()
	capability := assignmentTestCapability(t, "judge:a", "principal:judge-a", "instance:judge-a")
	assignment, err := AssignJudge(request, []JudgeCapability{capability})
	if err != nil {
		t.Fatal(err)
	}

	unsafe := request
	unsafe.AssignmentAuthorityRef = "https://user:secret@example.test/authority?token=x#fragment"
	if err := ValidateJudgeAssignmentRequest(unsafe); !errors.Is(err, ErrJudgeAssignmentInvalid) {
		t.Fatalf("unsafe request error = %v", err)
	}

	staleNow := capability.ValidUntil
	if err := ValidateJudgeCapabilityAt(capability, staleNow); !errors.Is(err, ErrJudgeExpired) {
		t.Fatalf("stale capability error = %v", err)
	}

	tamperedCapability := capability
	tamperedCapability.ModelRef = "model:tampered"
	if err := VerifyJudgeCapabilityDigest(tamperedCapability); !errors.Is(err, ErrJudgeCapabilityInvalid) {
		t.Fatalf("tampered capability error = %v", err)
	}

	tamperedAssignment := assignment
	tamperedAssignment.Lease.Generation++
	if err := VerifyJudgeAssignmentDigest(tamperedAssignment); !errors.Is(err, ErrJudgeAssignmentMismatch) {
		t.Fatalf("tampered assignment error = %v", err)
	}

	if err := ValidateJudgeAssignmentAt(assignment, request, capability, assignment.Lease.ExpiresAt); !errors.Is(err, ErrJudgeExpired) {
		t.Fatalf("expired lease error = %v", err)
	}
}

func TestAssignmentCanonicalBytesAreOrderingStable(t *testing.T) {
	capability := assignmentTestCapability(t, "judge:a", "principal:judge-a", "instance:judge-a")
	first, err := CanonicalJudgeCapabilityBytes(capability)
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(capability.SupportedModalities)
	slices.Reverse(capability.PolicyRefs)
	slices.Reverse(capability.IndependenceRefs)
	capability = assignmentSealCapability(t, capability)
	second, err := CanonicalJudgeCapabilityBytes(capability)
	if err != nil {
		t.Fatal(err)
	}
	var left, right JudgeCapability
	if json.Unmarshal(first, &left) != nil || json.Unmarshal(second, &right) != nil {
		t.Fatal("canonical decode failed")
	}
	left.CapabilityDigest, right.CapabilityDigest = "", ""
	if !reflect.DeepEqual(left, right) {
		t.Fatal("ordering changed canonical capability")
	}
}

func assignmentTestRequest() JudgeAssignmentRequest {
	requested := time.Date(2026, 8, 30, 8, 0, 0, 0, time.UTC)
	return JudgeAssignmentRequest{
		Schema: JudgeAssignmentRequestSchema, RequestID: "assignment-request:epwa-t06c", IdempotencyRef: "idempotency:epwa-t06c",
		Artifact: ArtifactBinding{ArtifactRef: "artifact:epwa-001", Revision: 1,
			BundleSHA256: assignmentHex('b'), ManifestSHA256: assignmentHex('a'),
			Scope: ScopeBinding{ProjectRef: "project:uiai", WorkstreamRef: "workstream:epwa", WorksetRef: "workset:t06",
				CallGraphRef: "callgraph:epwa", WorkpointRef: "workpoint:t06", WorkItemRef: "work-item:t06c"},
			TrustRefs: []string{"trust:artifact"}, SecurityRefs: []string{"security:strict"}},
		ViewRef: "judge-view:epwa", ViewSHA256: assignmentHex('c'), InformationSetRef: "information-set:epwa",
		InformationSetSHA256: assignmentHex('d'), ExecutorIdentityRef: "agent:executor", ExecutorPrincipalRef: "principal:executor",
		ExecutorInstanceRef: "instance:executor", ExecutorCorrelationRefs: []string{"provider:executor"},
		VerifierPolicyRef: "verifier-policy:independent", VerifierPolicyRevision: "1",
		RequiredModalities: []Modality{ModalityStructuredData, ModalityStaticImage}, ResultDetail: "atom_decisions_with_citations",
		Budget:                 JudgeBudget{MaxTokens: 4000, MaxMediaBytes: 8 << 20, MaxSpendMicros: 100000, MaxDurationMS: 120000},
		AssignmentAuthorityRef: "authority:review-assignment", DisallowedCorrelationRefs: []string{"team:executor"},
		RequestedAt: requested, ExpiresAt: requested.Add(time.Hour), LeaseDurationMS: uint64((30 * time.Minute).Milliseconds()),
		MaxCandidates: 4, IndependenceRequired: true, AutonomousEligible: true,
	}
}

func assignmentTestCapability(t *testing.T, identity, principal, instance string) JudgeCapability {
	t.Helper()
	capability := JudgeCapability{
		Schema: JudgeCapabilitySchema, JudgeIdentityRef: identity, PrincipalRef: principal, InstanceRef: instance,
		HarnessRef: "harness:focusa-silent", ProviderRef: "provider:independent", ModelRef: "model:multimodal",
		SupportedModalities:   []Modality{ModalityStaticImage, ModalityStructuredData, ModalityBoundedText},
		SupportedResultDetail: []string{"atom_decisions_with_citations"},
		Calibration: JudgeCalibration{CorpusRef: "calibration-corpus:epwa", ResultRef: "calibration-result:passed", Revision: "1", Status: CalibrationPassed,
			ValidUntil: time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC)},
		IndependenceRefs: []string{"independence:separate-principal", "independence:separate-instance"},
		CorrelationRefs:  []string{"provider:independent"}, PolicyRefs: []string{"verifier-policy:independent"}, Status: CapabilityEligible,
		MaxBudget: JudgeBudget{MaxTokens: 8000, MaxMediaBytes: 16 << 20, MaxSpendMicros: 200000, MaxDurationMS: 240000},
		ValidFrom: time.Date(2026, 8, 30, 7, 0, 0, 0, time.UTC), ValidUntil: time.Date(2026, 8, 30, 10, 0, 0, 0, time.UTC),
	}
	return assignmentSealCapability(t, capability)
}

func assignmentSealCapability(t *testing.T, capability JudgeCapability) JudgeCapability {
	t.Helper()
	capability.CapabilityDigest = ""
	digest, err := ComputeJudgeCapabilityDigest(capability)
	if err != nil {
		t.Fatalf("ComputeJudgeCapabilityDigest() error = %v", err)
	}
	capability.CapabilityDigest = digest
	return capability
}

func assignmentHex(value byte) string { return strings.Repeat(string(value), 64) }

func assignmentJSON(t *testing.T, value any) string {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
