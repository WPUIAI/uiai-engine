package evidenceaction

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	ReviewCaseSchema        = "uiai.review_case.v1"
	ReviewDecisionSchema    = "uiai.review_decision.v1"
	ReviewReceiptSchema     = "uiai.review_receipt.v1"
	ReviewDispositionSchema = "uiai.review_disposition.v1"
)

type ReviewPosture string

const (
	ReviewUnreviewed       ReviewPosture = "unreviewed"
	ReviewRequested        ReviewPosture = "review_requested"
	ReviewAssigned         ReviewPosture = "assigned"
	ReviewInReview         ReviewPosture = "in_review"
	ReviewAccepted         ReviewPosture = "accepted"
	ReviewRejected         ReviewPosture = "rejected"
	ReviewChangesRequested ReviewPosture = "changes_requested"
	ReviewDisputed         ReviewPosture = "disputed"
	ReviewExpired          ReviewPosture = "expired"
	ReviewLeaseExpired     ReviewPosture = "lease_expired"
	ReviewPartial          ReviewPosture = "partial"
	ReviewQuorumPending    ReviewPosture = "quorum_pending"
	ReviewReproofRequired  ReviewPosture = "reproof_required"
	ReviewPendingProof     ReviewPosture = "pending_proof"
	ReviewReturnedToModel  ReviewPosture = "returned_to_model"
	ReviewBlocked          ReviewPosture = "blocked"
)

// ReviewCase is a read projection of the canonical review authority. UIAI may
// render it and submit typed operations, but it never stores mutable review or
// completion state locally.
type ReviewCase struct {
	Schema                string        `json:"schema"`
	CaseRef               string        `json:"case_ref"`
	Scope                 ScopeBinding  `json:"scope"`
	ArtifactRef           string        `json:"artifact_ref"`
	ArtifactSHA256        string        `json:"artifact_sha256"`
	ArtifactRevision      uint64        `json:"artifact_revision"`
	WorkItemRef           string        `json:"work_item_ref"`
	AcceptanceAtomRefs    []string      `json:"acceptance_atom_refs,omitempty"`
	ReviewRequirementRefs []string      `json:"review_requirement_refs,omitempty"`
	ReviewerAssignmentRef string        `json:"reviewer_assignment_ref,omitempty"`
	ReviewerRef           string        `json:"reviewer_ref,omitempty"`
	ReviewerKind          string        `json:"reviewer_kind,omitempty"`
	HumanReviewMandated   bool          `json:"human_review_mandated"`
	AutonomousEligibility string        `json:"autonomous_eligibility,omitempty"`
	Posture               ReviewPosture `json:"posture"`
	DecisionRef           string        `json:"decision_ref,omitempty"`
	ReceiptRef            string        `json:"receipt_ref,omitempty"`
	ProofRefs             []string      `json:"proof_refs,omitempty"`
	ProofGaps             []string      `json:"proof_gaps,omitempty"`
	NextAction            string        `json:"next_action,omitempty"`
	ObservedAt            time.Time     `json:"observed_at"`
}

// ReviewDecisionRequest is the only browser/agent mutation payload. Its proof
// refs identify canonical evidence; they are not proof merely because a client
// supplied them. Focusa Completion Authority must validate them before an
// approval can be recorded.
type ReviewDecisionRequest struct {
	Schema                string         `json:"schema"`
	CaseRef               string         `json:"case_ref"`
	Scope                 ScopeBinding   `json:"scope"`
	ArtifactRef           string         `json:"artifact_ref"`
	ArtifactSHA256        string         `json:"artifact_sha256"`
	WorkItemRef           string         `json:"work_item_ref"`
	Decision              ReviewDecision `json:"decision"`
	ReviewerRef           string         `json:"reviewer_ref"`
	ReviewerAssignmentRef string         `json:"reviewer_assignment_ref"`
	ProofRefs             []string       `json:"proof_refs"`
	CitationRefs          []string       `json:"citation_refs,omitempty"`
	Reason                string         `json:"reason,omitempty"`
	IdempotencyKey        string         `json:"idempotency_key"`
	SubmittedAt           time.Time      `json:"submitted_at"`
}

// ReviewReceipt is a bounded receipt returned by the canonical authority. The
// Focusa receipt is retained as the authority reference; this object is a
// projection and not a second ledger.
type ReviewReceipt struct {
	Schema           string         `json:"schema"`
	ReceiptRef       string         `json:"receipt_ref"`
	CaseRef          string         `json:"case_ref"`
	DecisionRef      string         `json:"decision_ref"`
	Decision         ReviewDecision `json:"decision"`
	Posture          ReviewPosture  `json:"posture"`
	ReviewerRef      string         `json:"reviewer_ref"`
	ProofRefs        []string       `json:"proof_refs,omitempty"`
	ProofGaps        []string       `json:"proof_gaps,omitempty"`
	NextAction       string         `json:"next_action,omitempty"`
	FocusaReceiptRef string         `json:"focusa_receipt_ref,omitempty"`
	RecordedAt       time.Time      `json:"recorded_at"`
}

func NewReviewCaseRef(artifactRef, workItemRef, artifactSHA256 string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{artifactRef, workItemRef, artifactSHA256}, "\x00")))
	return "review-case:sha256:" + hex.EncodeToString(sum[:])
}

func (c ReviewCase) Validate() error {
	if c.Schema != ReviewCaseSchema {
		return fmt.Errorf("review case schema: %w", ErrInvalidReview)
	}
	if strings.TrimSpace(c.CaseRef) == "" || strings.TrimSpace(c.ArtifactRef) == "" || strings.TrimSpace(c.ArtifactSHA256) == "" {
		return fmt.Errorf("review case identity: %w", ErrInvalidReview)
	}
	if strings.TrimSpace(c.WorkItemRef) == "" || c.Scope.WorkItemRef != c.WorkItemRef {
		return fmt.Errorf("review case work item scope: %w", ErrScopeMismatch)
	}
	if c.Scope.ProjectRef == "" || c.Scope.WorkstreamRef == "" || c.Scope.WorksetRef == "" || c.Scope.CallGraphRef == "" || c.Scope.WorkpointRef == "" {
		return fmt.Errorf("review case incomplete scope: %w", ErrScopeMismatch)
	}
	if c.Posture == "" {
		return fmt.Errorf("review case posture: %w", ErrInvalidReview)
	}
	return nil
}

func (r ReviewDecisionRequest) Validate(c ReviewCase) error {
	if r.Schema != ReviewDecisionSchema {
		return fmt.Errorf("review decision schema: %w", ErrInvalidReview)
	}
	if err := c.Validate(); err != nil {
		return err
	}
	if r.CaseRef != c.CaseRef || r.ArtifactRef != c.ArtifactRef || r.ArtifactSHA256 != c.ArtifactSHA256 || r.WorkItemRef != c.WorkItemRef || r.Scope != c.Scope {
		return fmt.Errorf("review decision binding: %w", ErrScopeMismatch)
	}
	if strings.TrimSpace(r.ReviewerRef) == "" || strings.TrimSpace(r.ReviewerAssignmentRef) == "" || strings.TrimSpace(c.ReviewerAssignmentRef) == "" {
		return fmt.Errorf("reviewer assignment is required: %w", ErrReviewerNotAssigned)
	}
	if r.ReviewerAssignmentRef != c.ReviewerAssignmentRef || (c.ReviewerRef != "" && r.ReviewerRef != c.ReviewerRef) {
		return fmt.Errorf("reviewer assignment binding: %w", ErrScopeMismatch)
	}
	if strings.TrimSpace(r.IdempotencyKey) == "" {
		return fmt.Errorf("idempotency key is required: %w", ErrInvalidReview)
	}
	if r.Decision != DecisionApproved && r.Decision != DecisionRejected && r.Decision != DecisionChangesRequested {
		return fmt.Errorf("unsupported decision %q: %w", r.Decision, ErrInvalidReview)
	}
	if (r.Decision == DecisionRejected || r.Decision == DecisionChangesRequested) && strings.TrimSpace(r.Reason) == "" {
		return fmt.Errorf("rejection or changes request requires a reason: %w", ErrReasonRequired)
	}
	if r.Decision == DecisionApproved && (len(r.ProofRefs) == 0 || len(MissingProofRefs(c.AcceptanceAtomRefs, r.ProofRefs)) != 0) {
		return fmt.Errorf("approval proof is incomplete: %w", ErrProofMissing)
	}
	return nil
}

func MissingProofRefs(requiredAtoms, proofRefs []string) []string {
	missing := make([]string, 0)
	for _, atom := range requiredAtoms {
		atom = strings.TrimSpace(atom)
		if atom == "" {
			continue
		}
		covered := false
		for _, ref := range proofRefs {
			if strings.Contains(strings.ToLower(ref), strings.ToLower(atom)) {
				covered = true
				break
			}
		}
		if !covered {
			missing = append(missing, atom)
		}
	}
	return missing
}

var (
	ErrInvalidReview       = errors.New("invalid review contract")
	ErrScopeMismatch       = errors.New("review scope mismatch")
	ErrReviewerNotAssigned = errors.New("reviewer not assigned")
	ErrReasonRequired      = errors.New("review reason required")
	ErrProofMissing        = errors.New("required proof missing")
	ErrReviewAuthority     = errors.New("review authority unavailable")
)
