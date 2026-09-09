package evidenceshare

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidenceaction"
)

// ReviewStateSchema is the offline projection of the canonical review
// posture embedded into the record envelope and portable zip. It is a
// bounded read-only projection of Focusa receipts; the live canonical state
// always comes from the review route and the Focusa ledger itself.
const ReviewStateSchema = "uiai.review_state.v1"

type ReviewState struct {
	Schema       string                             `json:"schema"`
	CaseRef      string                             `json:"case_ref"`
	ArtifactRef  string                             `json:"artifact_ref"`
	Posture      evidenceaction.ReviewPosture       `json:"posture"`
	Decision     evidenceaction.ReviewDecision      `json:"decision,omitempty"`
	AuthorityRef string                             `json:"authority_ref,omitempty"`
	Dispositions []evidenceaction.ReviewDisposition `json:"dispositions,omitempty"`
	NextAction   string                             `json:"next_action,omitempty"`
	GeneratedAt  time.Time                          `json:"generated_at"`
}

// WriteReviewState writes the current authority-backed review projection
// into the record package so the envelope and portable zip carry disposition
// state. chainDir is the engine-local append-only projection of Focusa
// receipts; the package only mirrors what that chain already recorded.
// Empty chains produce the explicit unreviewed posture — never an absent or
// silently-default file.
func WriteReviewState(root, id, chainDir, caseRef, artifactRef string) (ReviewState, error) {
	state := ReviewState{
		Schema:      ReviewStateSchema,
		CaseRef:     caseRef,
		ArtifactRef: artifactRef,
		Posture:     evidenceaction.ReviewUnreviewed,
		GeneratedAt: time.Now().UTC(),
	}
	dispositions, err := evidenceaction.ListReviewDispositions(chainDir, caseRef)
	if err != nil {
		return ReviewState{}, err
	}
	if len(dispositions) > 0 {
		latest := dispositions[len(dispositions)-1]
		state.Posture = latest.Posture
		state.Decision = latest.Decision
		state.AuthorityRef = latest.FocusaReceiptRef
		state.NextAction = dispositionNextAction(latest)
	}
	state.Dispositions = append(state.Dispositions, dispositions...)
	body, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return ReviewState{}, err
	}
	body = append(body, '\n')
	if err := os.WriteFile(filepath.Join(root, id, "review.json"), body, 0o600); err != nil {
		return ReviewState{}, fmt.Errorf("write review state: %w", err)
	}
	return state, nil
}

func dispositionNextAction(disposition evidenceaction.ReviewDisposition) string {
	if disposition.ReturnToModel || disposition.Decision != evidenceaction.DecisionApproved {
		return "repair the cited proof gaps and submit a new artifact revision"
	}
	return "request canonical completion evaluation"
}
