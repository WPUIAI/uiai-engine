package evidenceshare

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidenceaction"
)

func TestWriteReviewStateUnreviewedIsExplicit(t *testing.T) {
	root := t.TempDir()
	chainDir := t.TempDir()
	id := "a1b2c3"
	if err := os.MkdirAll(filepath.Join(root, id), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	state, err := WriteReviewState(root, id, chainDir, "review-case:sha256:case", "artifact:"+id)
	if err != nil {
		t.Fatalf("WriteReviewState() error = %v", err)
	}
	if state.Posture != evidenceaction.ReviewUnreviewed || len(state.Dispositions) != 0 {
		t.Fatalf("state = %#v", state)
	}
}

func TestWriteReviewStateMirrorsChain(t *testing.T) {
	root := t.TempDir()
	chainDir := t.TempDir()
	id := "a1b2c3"
	if err := os.MkdirAll(filepath.Join(root, id), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	disposition := evidenceaction.ReviewDisposition{
		Schema: evidenceaction.ReviewDispositionSchema, DispositionRef: "disposition:1",
		CaseRef: "review-case:sha256:case", Decision: evidenceaction.DecisionRejected,
		Posture: evidenceaction.ReviewReturnedToModel, AuthorityRef: "focusa-direction-receipt:op-1",
		FocusaReceiptRef: "op-1", ReviewerRef: "user:reviewer", WorkItemRef: "work-item:item",
		Reason: "missing proof", ReturnToModel: true, RecordedAt: time.Now().UTC(),
	}
	if err := evidenceaction.AppendReviewDisposition(chainDir, disposition); err != nil {
		t.Fatalf("AppendReviewDisposition() error = %v", err)
	}
	state, err := WriteReviewState(root, id, chainDir, disposition.CaseRef, "artifact:"+id)
	if err != nil {
		t.Fatalf("WriteReviewState() error = %v", err)
	}
	if state.Posture != evidenceaction.ReviewReturnedToModel || state.Decision != evidenceaction.DecisionRejected {
		t.Fatalf("state = %#v", state)
	}
	if state.AuthorityRef != "op-1" || len(state.Dispositions) != 1 {
		t.Fatalf("state = %#v", state)
	}
}
