package evidenceaction

import (
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func dispositionFixture(dispositionRef string, supersedes string) ReviewDisposition {
	return ReviewDisposition{
		Schema:                ReviewDispositionSchema,
		DispositionRef:        dispositionRef,
		CaseRef:               "review-case:sha256:case",
		Decision:              DecisionChangesRequested,
		Posture:               ReviewReturnedToModel,
		AuthorityRef:          "focusa-direction-receipt:op-1",
		FocusaReceiptRef:      "op-1",
		ReviewerRef:           "user:reviewer",
		ReviewerAssignmentRef: "assignment:1",
		WorkItemRef:           "work-item:item",
		Reason:                "missing proof",
		ProofGaps:             []string{"atom:verify"},
		SupersedesRef:         supersedes,
		ReturnToModel:         true,
		RecordedAt:            time.Now().UTC(),
	}
}

func TestAppendReviewDispositionBuildsChain(t *testing.T) {
	dir := t.TempDir()
	first := dispositionFixture("disposition:1", "")
	if err := AppendReviewDisposition(dir, first); err != nil {
		t.Fatalf("AppendReviewDisposition() error = %v", err)
	}
	second := dispositionFixture("disposition:2", "disposition:1")
	second.Decision = DecisionApproved
	second.Posture = ReviewAccepted
	second.Reason = ""
	second.ProofGaps = nil
	second.ReturnToModel = false
	if err := AppendReviewDisposition(dir, second); err != nil {
		t.Fatalf("AppendReviewDisposition(second) error = %v", err)
	}
	latest, found, err := LatestReviewDisposition(dir, second.CaseRef)
	if err != nil || !found {
		t.Fatalf("LatestReviewDisposition() found=%v error=%v", found, err)
	}
	if latest.DispositionRef != "disposition:2" || latest.SupersedesRef != "disposition:1" {
		t.Fatalf("latest = %#v", latest)
	}
	entries, err := ListReviewDispositions(dir, second.CaseRef)
	if err != nil || len(entries) != 2 {
		t.Fatalf("ListReviewDispositions() = %d entries, error %v", len(entries), err)
	}
}

func TestAppendReviewDispositionFailsClosed(t *testing.T) {
	dir := t.TempDir()
	first := dispositionFixture("disposition:1", "")
	if err := AppendReviewDisposition(dir, first); err != nil {
		t.Fatalf("AppendReviewDisposition() error = %v", err)
	}
	second := dispositionFixture("disposition:2", "")
	if err := AppendReviewDisposition(dir, second); !errors.Is(err, ErrScopeMismatch) {
		t.Fatalf("missing supersedes error = %v, want ErrScopeMismatch", err)
	}
	stale := dispositionFixture("disposition:2", "disposition:stale")
	if err := AppendReviewDisposition(dir, stale); !errors.Is(err, ErrScopeMismatch) {
		t.Fatalf("wrong supersedes error = %v, want ErrScopeMismatch", err)
	}
	missingReceipt := dispositionFixture("disposition:9", "")
	missingReceipt.FocusaReceiptRef = ""
	if err := AppendReviewDisposition(dir, missingReceipt); !errors.Is(err, ErrReviewAuthority) {
		t.Fatalf("missing authority error = %v, want ErrReviewAuthority", err)
	}
	missingReason := dispositionFixture("disposition:9", "")
	missingReason.Reason = ""
	missingReason.FocusaReceiptRef = "op-9"
	if err := AppendReviewDisposition(dir, missingReason); !errors.Is(err, ErrReasonRequired) {
		t.Fatalf("missing reason error = %v, want ErrReasonRequired", err)
	}
	staleSupersedes := dispositionFixture("disposition:2", "disposition:1")
	staleSupersedes.CaseRef = "review-case:sha256:unknown-chain"
	if err := AppendReviewDisposition(dir, staleSupersedes); !errors.Is(err, ErrScopeMismatch) {
		t.Fatalf("supersede on unknown chain error = %v, want ErrScopeMismatch", err)
	}
	if _, found, err := LatestReviewDisposition(filepath.Join(dir), "review-case:sha256:case"); err != nil || !found {
		t.Fatalf("chain must remain intact after rejected appends, found=%v err=%v", found, err)
	}
}

func TestLatestReviewDispositionEmptyCase(t *testing.T) {
	dir := t.TempDir()
	_, found, err := LatestReviewDisposition(dir, "review-case:sha256:none")
	if err != nil || found {
		t.Fatalf("LatestReviewDisposition() found=%v error=%v, want false/nil", found, err)
	}
}
