package evidenceaction

import (
	"errors"
	"testing"
)

func reviewCaseFixture() ReviewCase {
	return ReviewCase{
		Schema:                ReviewCaseSchema,
		CaseRef:               "review-case:sha256:case",
		Scope:                 ScopeBinding{ProjectRef: "project:p", WorkstreamRef: "workstream:w", WorksetRef: "workset:s", CallGraphRef: "callgraph:g", WorkpointRef: "workpoint:wp", WorkItemRef: "work-item:item"},
		ArtifactRef:           "artifact:a",
		ArtifactSHA256:        "aabbcc",
		ArtifactRevision:      1,
		WorkItemRef:           "work-item:item",
		AcceptanceAtomRefs:    []string{"atom:capture", "atom:verify"},
		ReviewRequirementRefs: []string{"review:req"},
		ReviewerAssignmentRef: "assignment:1",
		ReviewerRef:           "reviewer:operator",
		Posture:               ReviewAssigned,
	}
}

func TestReviewDecisionRequiresProofForApproval(t *testing.T) {
	c := reviewCaseFixture()
	r := ReviewDecisionRequest{
		Schema: ReviewDecisionSchema, CaseRef: c.CaseRef, Scope: c.Scope,
		ArtifactRef: c.ArtifactRef, ArtifactSHA256: c.ArtifactSHA256, WorkItemRef: c.WorkItemRef,
		Decision: DecisionApproved, ReviewerRef: c.ReviewerRef, ReviewerAssignmentRef: c.ReviewerAssignmentRef,
		ProofRefs: []string{"evidence:atom:capture"}, IdempotencyKey: "review-1",
	}
	if err := r.Validate(c); !errors.Is(err, ErrProofMissing) {
		t.Fatalf("Validate() error = %v, want ErrProofMissing", err)
	}
}

func TestReviewDecisionRejectRequiresReason(t *testing.T) {
	c := reviewCaseFixture()
	r := ReviewDecisionRequest{
		Schema: ReviewDecisionSchema, CaseRef: c.CaseRef, Scope: c.Scope,
		ArtifactRef: c.ArtifactRef, ArtifactSHA256: c.ArtifactSHA256, WorkItemRef: c.WorkItemRef,
		Decision: DecisionRejected, ReviewerRef: c.ReviewerRef, ReviewerAssignmentRef: c.ReviewerAssignmentRef,
		IdempotencyKey: "review-1",
	}
	if err := r.Validate(c); !errors.Is(err, ErrReasonRequired) {
		t.Fatalf("Validate() error = %v, want ErrReasonRequired", err)
	}
}

func TestReviewDecisionAcceptsCompleteProof(t *testing.T) {
	c := reviewCaseFixture()
	r := ReviewDecisionRequest{
		Schema: ReviewDecisionSchema, CaseRef: c.CaseRef, Scope: c.Scope,
		ArtifactRef: c.ArtifactRef, ArtifactSHA256: c.ArtifactSHA256, WorkItemRef: c.WorkItemRef,
		Decision: DecisionApproved, ReviewerRef: c.ReviewerRef, ReviewerAssignmentRef: c.ReviewerAssignmentRef,
		ProofRefs: []string{"evidence:atom:capture:receipt", "evidence:atom:verify:receipt"}, IdempotencyKey: "review-1",
	}
	if err := r.Validate(c); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestReviewDecisionRejectsBindingDrift(t *testing.T) {
	c := reviewCaseFixture()
	r := ReviewDecisionRequest{
		Schema: ReviewDecisionSchema, CaseRef: c.CaseRef, Scope: c.Scope,
		ArtifactRef: c.ArtifactRef, ArtifactSHA256: "different", WorkItemRef: c.WorkItemRef,
		Decision: DecisionRejected, ReviewerRef: c.ReviewerRef, ReviewerAssignmentRef: c.ReviewerAssignmentRef,
		Reason: "missing proof", IdempotencyKey: "review-1",
	}
	if err := r.Validate(c); !errors.Is(err, ErrScopeMismatch) {
		t.Fatalf("Validate() error = %v, want ErrScopeMismatch", err)
	}
}

func TestMissingProofRefsIsDeterministic(t *testing.T) {
	got := MissingProofRefs([]string{"atom:one", "atom:two"}, []string{"receipt:atom:one"})
	if len(got) != 1 || got[0] != "atom:two" {
		t.Fatalf("MissingProofRefs() = %#v, want [atom:two]", got)
	}
}
