package evidenceaction

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ReviewDisposition is a bounded read-only projection of one canonical
// Focusa disposition receipt. The engine never creates review authority
// here: every appended entry must carry the Focusa receipt reference that
// produced it, and the append-only file preserves the full supersede chain.
type ReviewDisposition struct {
	Schema                string         `json:"schema"`
	DispositionRef        string         `json:"disposition_ref"`
	CaseRef               string         `json:"case_ref"`
	Decision              ReviewDecision `json:"decision"`
	Posture               ReviewPosture  `json:"posture"`
	AuthorityRef          string         `json:"authority_ref"`
	FocusaReceiptRef      string         `json:"focusa_receipt_ref"`
	ReviewerRef           string         `json:"reviewer_ref"`
	ReviewerAssignmentRef string         `json:"reviewer_assignment_ref,omitempty"`
	WorkItemRef           string         `json:"work_item_ref"`
	Reason                string         `json:"reason,omitempty"`
	ProofGaps             []string       `json:"proof_gaps,omitempty"`
	SupersedesRef         string         `json:"supersedes_ref,omitempty"`
	ReturnToModel         bool           `json:"return_to_model"`
	RecordedAt            time.Time      `json:"recorded_at"`
}

func (d ReviewDisposition) Validate() error {
	if d.Schema != ReviewDispositionSchema {
		return fmt.Errorf("disposition schema: %w", ErrInvalidReview)
	}
	if strings.TrimSpace(d.DispositionRef) == "" || strings.TrimSpace(d.CaseRef) == "" {
		return fmt.Errorf("disposition identity: %w", ErrInvalidReview)
	}
	if d.Decision != DecisionApproved && d.Decision != DecisionRejected && d.Decision != DecisionChangesRequested && d.Decision != DecisionBlocked {
		return fmt.Errorf("disposition decision %q: %w", d.Decision, ErrInvalidReview)
	}
	if strings.TrimSpace(d.AuthorityRef) == "" || strings.TrimSpace(d.FocusaReceiptRef) == "" {
		return fmt.Errorf("disposition authority receipt is required: %w", ErrReviewAuthority)
	}
	if (d.Decision == DecisionRejected || d.Decision == DecisionChangesRequested) && strings.TrimSpace(d.Reason) == "" {
		return fmt.Errorf("disposition rejection requires a reason: %w", ErrReasonRequired)
	}
	if d.Posture == "" {
		return fmt.Errorf("disposition posture: %w", ErrInvalidReview)
	}
	return nil
}

func dispositionFilePath(dir, caseRef string) string {
	sum := sha256.Sum256([]byte(caseRef))
	return filepath.Join(dir, fmt.Sprintf("%x.jsonl", sum[:]))
}

// AppendReviewDisposition appends one validated disposition to the
// append-only chain for its case. When a chain already exists, the new entry
// must explicitly supersede the latest entry; nothing is ever rewritten or
// deleted. Unknown postures and silent defaults fail closed.
func AppendReviewDisposition(dir string, disposition ReviewDisposition) error {
	if err := disposition.Validate(); err != nil {
		return err
	}
	if disposition.SupersedesRef == "" {
		if latest, found, err := LatestReviewDisposition(dir, disposition.CaseRef); err != nil {
			return err
		} else if found {
			return fmt.Errorf("disposition must supersede %q: %w", latest.DispositionRef, ErrScopeMismatch)
		}
	} else {
		latest, found, err := LatestReviewDisposition(dir, disposition.CaseRef)
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("supersede target %q not in chain: %w", disposition.SupersedesRef, ErrScopeMismatch)
		}
		if latest.DispositionRef != disposition.SupersedesRef {
			return fmt.Errorf("disposition must supersede latest %q: %w", latest.DispositionRef, ErrScopeMismatch)
		}
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("prepare disposition store: %w", err)
	}
	body, err := json.Marshal(disposition)
	if err != nil {
		return fmt.Errorf("encode disposition: %w", err)
	}
	file := dispositionFilePath(dir, disposition.CaseRef)
	handle, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("open disposition chain: %w", err)
	}
	defer handle.Close()
	if _, err := handle.Write(append(body, '\n')); err != nil {
		return fmt.Errorf("append disposition: %w", err)
	}
	return nil
}

// LatestReviewDisposition returns the newest disposition in the chain, or
// found=false when the case has no local projection. Truncated entries fail
// closed instead of being skipped.
func LatestReviewDisposition(dir, caseRef string) (ReviewDisposition, bool, error) {
	body, err := os.ReadFile(dispositionFilePath(dir, caseRef))
	if err != nil {
		if os.IsNotExist(err) {
			return ReviewDisposition{}, false, nil
		}
		return ReviewDisposition{}, false, fmt.Errorf("read disposition chain: %w", err)
	}
	lines := strings.Split(strings.TrimRight(string(body), "\n"), "\n")
	var latest ReviewDisposition
	for index, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var entry ReviewDisposition
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return ReviewDisposition{}, false, fmt.Errorf("corrupt disposition entry %d: %w", index+1, err)
		}
		latest = entry
	}
	if latest.Schema == "" {
		return ReviewDisposition{}, false, nil
	}
	return latest, true, nil
}

// ListReviewDispositions returns every entry in chain order; the full
// supersede history stays inspectable on the record surface.
func ListReviewDispositions(dir, caseRef string) ([]ReviewDisposition, error) {
	body, err := os.ReadFile(dispositionFilePath(dir, caseRef))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read disposition chain: %w", err)
	}
	lines := strings.Split(strings.TrimRight(string(body), "\n"), "\n")
	entries := make([]ReviewDisposition, 0, len(lines))
	for index, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var entry ReviewDisposition
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			return nil, fmt.Errorf("corrupt disposition entry %d: %w", index+1, err)
		}
		entries = append(entries, entry)
	}
	return entries, nil
}
