package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/auth"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/evidenceaction"
	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
	"github.com/WPUIAI/uiai-engine/internal/evidenceregistry"
	"github.com/WPUIAI/uiai-engine/internal/evidenceshare"
	"github.com/go-chi/chi/v5"
)

const (
	completionClaimSchema = "focusa.completion_claim.v1"
	maxReviewBodyBytes    = 256 * 1024
)

// reviewDispositionDir holds the append-only, engine-local projections of
// canonical Focusa disposition receipts. It is a cache of authority output,
// never a second authority; every entry carries the Focusa receipt reference.
func reviewDispositionDir(cfg *config.Config) string {
	if dir := os.Getenv("UIAI_EVIDENCE_REVIEW_DIR"); dir != "" {
		return dir
	}
	if cfg != nil && cfg.Storage.DataDir != "" {
		return filepath.Join(cfg.Storage.DataDir, "epwa-review")
	}
	return filepath.Join(screenshotStoreDir(), "epwa-review")
}

type completionClaim struct {
	Schema          string   `json:"schema"`
	WorkItemID      string   `json:"work_item_id"`
	AcceptanceAtoms []string `json:"acceptance_atoms"`
	EvidenceRefs    []string `json:"evidence_refs"`
	Receipts        []string `json:"receipts"`
	ClaimText       string   `json:"claim_text"`
}

type completionVerdict struct {
	Allow          bool     `json:"allow"`
	UncoveredAtoms []string `json:"uncovered_atoms"`
	OverclaimRisks []string `json:"overclaim_risks"`
	Reasons        []string `json:"reasons"`
}

type focusaReviewClient struct {
	baseURL string
	client  *http.Client
}

func newFocusaReviewClient(cfg *config.Config) *focusaReviewClient {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.EvidenceRegistry.FocusaURL), "/")
	if baseURL == "" {
		return nil
	}
	return &focusaReviewClient{baseURL: baseURL, client: evidenceregistry.NewFocusaHTTPClient(cfg.EvidenceRegistry.FocusaTokenFile)}
}

func (c *focusaReviewClient) postJSON(ctx context.Context, path string, input any, output any) error {
	if c == nil || c.client == nil || c.baseURL == "" {
		return evidenceaction.ErrReviewAuthority
	}
	body, err := json.Marshal(input)
	if err != nil {
		return fmt.Errorf("marshal Focusa request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build Focusa request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("Focusa request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("Focusa request returned HTTP %d", resp.StatusCode)
	}
	if output == nil {
		return nil
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxReviewBodyBytes)).Decode(output); err != nil {
		return fmt.Errorf("decode Focusa response: %w", err)
	}
	return nil
}

func (c *focusaReviewClient) evaluate(ctx context.Context, claim completionClaim) (completionVerdict, error) {
	var response struct {
		Verdict completionVerdict `json:"verdict"`
	}
	if err := c.postJSON(ctx, "/v1/completion-claims/evaluate", claim, &response); err != nil {
		return completionVerdict{}, err
	}
	return response.Verdict, nil
}

func (c *focusaReviewClient) record(ctx context.Context, req evidenceaction.ReviewDecisionRequest, returnToModel bool) (evidenceaction.ReviewReceipt, error) {
	decisionRef := req.CaseRef + "#" + req.IdempotencyKey
	feedback, err := json.Marshal(map[string]any{
		"schema":                  evidenceaction.ReviewDecisionSchema,
		"case_ref":                req.CaseRef,
		"scope":                   req.Scope,
		"artifact_ref":            req.ArtifactRef,
		"artifact_sha256":         req.ArtifactSHA256,
		"work_item_ref":           req.WorkItemRef,
		"reviewer_ref":            req.ReviewerRef,
		"reviewer_assignment_ref": req.ReviewerAssignmentRef,
		"proof_refs":              req.ProofRefs,
		"citation_refs":           req.CitationRefs,
		"reason":                  req.Reason,
		"return_to_model":         returnToModel,
	})
	if err != nil {
		return evidenceaction.ReviewReceipt{}, fmt.Errorf("marshal review feedback: %w", err)
	}
	operation := map[string]any{
		"operation":    "review_decision",
		"decision_ref": decisionRef,
		"outcome":      string(req.Decision),
		"feedback":     string(feedback),
	}
	var response struct {
		Receipt struct {
			OperationID string `json:"operation_id"`
			RecordedAt  string `json:"recorded_at"`
		} `json:"receipt"`
	}
	if err := c.postJSON(ctx, "/v1/direction/operations", operation, &response); err != nil {
		return evidenceaction.ReviewReceipt{}, err
	}
	if strings.TrimSpace(response.Receipt.OperationID) == "" {
		return evidenceaction.ReviewReceipt{}, errors.New("Focusa review response omitted receipt")
	}
	recordedAt, _ := time.Parse(time.RFC3339Nano, response.Receipt.RecordedAt)
	posture := evidenceaction.ReviewAccepted
	nextAction := "request canonical completion evaluation"
	if returnToModel || req.Decision == evidenceaction.DecisionRejected || req.Decision == evidenceaction.DecisionChangesRequested {
		posture = evidenceaction.ReviewReturnedToModel
		nextAction = "repair the cited proof gaps and submit a new artifact revision"
	}
	return evidenceaction.ReviewReceipt{
		Schema:           evidenceaction.ReviewReceiptSchema,
		ReceiptRef:       "focusa-direction-receipt:" + response.Receipt.OperationID,
		FocusaReceiptRef: response.Receipt.OperationID,
		CaseRef:          req.CaseRef,
		DecisionRef:      decisionRef,
		Decision:         req.Decision,
		Posture:          posture,
		ReviewerRef:      req.ReviewerRef,
		ProofRefs:        append([]string(nil), req.ProofRefs...),
		NextAction:       nextAction,
		RecordedAt:       recordedAt,
	}, nil
}

func mountEvidenceReview(r chi.Router, cfg *config.Config) {
	r.Get("/share/{id}/review", func(w http.ResponseWriter, req *http.Request) {
		caseProjection, err := loadReviewCase(cfg, chi.URLParam(req, "id"))
		if err != nil {
			writeReviewError(w, req, err)
			return
		}
		dispositions, listErr := evidenceaction.ListReviewDispositions(reviewDispositionDir(cfg), caseProjection.CaseRef)
		if listErr != nil {
			writeReviewError(w, req, listErr)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"schema":       evidenceaction.ReviewCaseSchema,
			"review_case":  caseProjection,
			"dispositions": dispositions,
			"decision_url": "./review/decision",
			"mutation":     "canonical Focusa authority only",
		})
	})

	r.Post("/share/{id}/review/decision", func(w http.ResponseWriter, req *http.Request) {
		caseProjection, err := loadReviewCase(cfg, chi.URLParam(req, "id"))
		if err != nil {
			writeReviewError(w, req, err)
			return
		}
		identity := auth.FromContext(req.Context())
		reviewerRef := authenticatedReviewerRef(identity)
		if reviewerRef == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]any{"state": "blocked", "code": "reviewer_authentication_required", "message": "an authenticated appointed reviewer is required"})
			return
		}
		var decision evidenceaction.ReviewDecisionRequest
		decoder := json.NewDecoder(io.LimitReader(req.Body, maxReviewBodyBytes))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&decision); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"code": "invalid_review_decision", "message": "the review decision payload is invalid"})
			return
		}
		decision.ReviewerRef = reviewerRef
		if caseProjection.ReviewerRef != "" && caseProjection.ReviewerRef != reviewerRef {
			writeJSON(w, http.StatusForbidden, map[string]string{"code": "reviewer_not_assigned", "message": "the authenticated reviewer is not the appointed reviewer for this case"})
			return
		}
		if err := decision.Validate(caseProjection); err != nil {
			writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"state": "blocked", "code": errorCode(err), "message": err.Error()})
			return
		}
		client := newFocusaReviewClient(cfg)
		if client == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"state": "blocked", "code": "review_authority_unavailable", "message": "canonical Focusa review authority is not configured"})
			return
		}

		proofGaps := evidenceaction.MissingProofRefs(caseProjection.AcceptanceAtomRefs, decision.ProofRefs)
		if decision.Decision == evidenceaction.DecisionApproved && len(proofGaps) == 0 {
			verdict, evalErr := client.evaluate(req.Context(), completionClaim{
				Schema: completionClaimSchema, WorkItemID: decision.WorkItemRef,
				AcceptanceAtoms: caseProjection.AcceptanceAtomRefs, EvidenceRefs: decision.ProofRefs,
				Receipts: decision.ProofRefs, ClaimText: "EPWA review approval for " + decision.CaseRef,
			})
			if evalErr != nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]any{"state": "blocked", "code": "completion_authority_unavailable", "message": "canonical completion evaluation could not be reconciled"})
				return
			}
			proofGaps = append(proofGaps, verdict.UncoveredAtoms...)
			if !verdict.Allow || len(proofGaps) != 0 {
				decision.Decision = evidenceaction.DecisionChangesRequested
				decision.Reason = "truthful completion proof is incomplete: " + strings.Join(proofGaps, ", ")
			}
		}
		returnToModel := decision.Decision != evidenceaction.DecisionApproved
		if len(proofGaps) != 0 && strings.TrimSpace(decision.Reason) == "" {
			decision.Decision = evidenceaction.DecisionChangesRequested
			decision.Reason = "truthful completion proof is incomplete: " + strings.Join(proofGaps, ", ")
			returnToModel = true
		}
		receipt, recordErr := client.record(req.Context(), decision, returnToModel)
		if recordErr != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{"state": "blocked", "code": "review_receipt_unavailable", "message": "canonical Focusa review receipt could not be committed"})
			return
		}
		receipt.ProofGaps = proofGaps
		disposition := evidenceaction.ReviewDisposition{
			Schema: evidenceaction.ReviewDispositionSchema, DispositionRef: receipt.DecisionRef,
			CaseRef: receipt.CaseRef, Decision: receipt.Decision, Posture: receipt.Posture,
			AuthorityRef: receipt.FocusaReceiptRef, FocusaReceiptRef: receipt.FocusaReceiptRef,
			ReviewerRef: receipt.ReviewerRef, ReviewerAssignmentRef: decision.ReviewerAssignmentRef,
			WorkItemRef: decision.WorkItemRef, Reason: decision.Reason,
			ProofGaps: proofGaps, ReturnToModel: returnToModel, RecordedAt: receipt.RecordedAt,
		}
		if previous, found, latestErr := evidenceaction.LatestReviewDisposition(reviewDispositionDir(cfg), disposition.CaseRef); latestErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"state": "blocked", "code": "disposition_chain_unavailable", "message": latestErr.Error()})
			return
		} else if found {
			disposition.SupersedesRef = previous.DispositionRef
		}
		if appendErr := evidenceaction.AppendReviewDisposition(reviewDispositionDir(cfg), disposition); appendErr != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]any{"state": "blocked", "code": "disposition_append_failed", "message": appendErr.Error()})
			return
		}
		envelopeRefresh := "ok"
		if _, stateErr := evidenceshare.WriteReviewState(evidenceShareDir(cfg), chi.URLParam(req, "id"), reviewDispositionDir(cfg), caseProjection.CaseRef, caseProjection.ArtifactRef); stateErr == nil {
			if _, zipErr := evidenceshare.EnsurePortableArchive(evidenceShareDir(cfg), chi.URLParam(req, "id")); zipErr != nil && !errors.Is(zipErr, evidenceshare.ErrInvalidInput) {
				envelopeRefresh = "zip_refresh_failed: " + zipErr.Error()
			}
		} else {
			envelopeRefresh = "state_write_failed: " + stateErr.Error()
		}
		if returnToModel {
			writeJSON(w, http.StatusConflict, map[string]any{"state": "returned_to_model", "review_receipt": receipt, "proof_gaps": proofGaps, "next_action": receipt.NextAction, "envelope_refresh": envelopeRefresh})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"state": "review_accepted", "review_receipt": receipt, "next_action": receipt.NextAction, "envelope_refresh": envelopeRefresh})
	})
}

func loadReviewCase(cfg *config.Config, id string) (evidenceaction.ReviewCase, error) {
	if !validShareID(id) {
		return evidenceaction.ReviewCase{}, os.ErrNotExist
	}
	manifestPath := filepath.Join(evidenceShareDir(cfg), id, "artifact.json")
	body, err := os.ReadFile(manifestPath)
	if err != nil {
		return evidenceaction.ReviewCase{}, err
	}
	var manifest evidenceshare.Manifest
	if err := json.Unmarshal(body, &manifest); err != nil || manifest.Schema != evidenceshare.Schema {
		return evidenceaction.ReviewCase{}, fmt.Errorf("review requires a screenshot EPWA manifest: %w", evidenceaction.ErrInvalidReview)
	}
	var item *evidencepwa.WorkItemProjection
	for index := range manifest.Scope.WorkItems {
		candidate := &manifest.Scope.WorkItems[index]
		if candidate.WorkItemRef == manifest.Scope.WorkItemRef {
			item = candidate
			break
		}
	}
	if item == nil {
		return evidenceaction.ReviewCase{}, fmt.Errorf("review work item missing: %w", evidenceaction.ErrScopeMismatch)
	}
	scope := evidenceaction.ScopeBinding{ProjectRef: manifest.Scope.ProjectRef, WorkstreamRef: manifest.Scope.WorkstreamRef, WorksetRef: manifest.Scope.WorksetRef, CallGraphRef: manifest.Scope.CallGraphRef, WorkpointRef: manifest.Scope.WorkpointRef, WorkItemRef: manifest.Scope.WorkItemRef}
	posture := evidenceaction.ReviewUnreviewed
	if len(item.Authority.ReviewRequirementRefs) > 0 {
		posture = evidenceaction.ReviewRequested
	}
	caseProjection := evidenceaction.ReviewCase{
		Schema: evidenceaction.ReviewCaseSchema, CaseRef: evidenceaction.NewReviewCaseRef(manifest.ArtifactRef, manifest.Scope.WorkItemRef, manifest.ArtifactSHA256),
		Scope: scope, ArtifactRef: manifest.ArtifactRef, ArtifactSHA256: manifest.ArtifactSHA256,
		WorkItemRef: manifest.Scope.WorkItemRef, AcceptanceAtomRefs: append([]string(nil), item.Authority.AcceptanceAtomRefs...),
		ReviewRequirementRefs: append([]string(nil), item.Authority.ReviewRequirementRefs...),
		ReviewerAssignmentRef: item.Authority.ReviewerAssignmentRef, ReviewerRef: item.Authority.ReviewerRef,
		ReviewerKind: item.Authority.ReviewerKind, Posture: posture, ObservedAt: time.Now().UTC(),
	}
	if caseProjection.ReviewerAssignmentRef == "" && len(caseProjection.ReviewRequirementRefs) > 0 {
		caseProjection.NextAction = "canonical reviewer assignment required"
	}
	caseProjection.ProofGaps = evidenceaction.MissingProofRefs(caseProjection.AcceptanceAtomRefs, nil)
	if err := caseProjection.Validate(); err != nil {
		return evidenceaction.ReviewCase{}, err
	}
	return caseProjection, nil
}

func authenticatedReviewerRef(identity *auth.Identity) string {
	if identity == nil {
		return ""
	}
	if identity.UserID != "" {
		return "user:" + identity.UserID
	}
	if identity.ClientID != "" {
		return "client:" + identity.ClientID
	}
	if identity.LicenseID != 0 {
		return fmt.Sprintf("license:%d", identity.LicenseID)
	}
	return ""
}

func errorCode(err error) string {
	switch {
	case errors.Is(err, evidenceaction.ErrScopeMismatch):
		return "scope_mismatch"
	case errors.Is(err, evidenceaction.ErrReviewerNotAssigned):
		return "reviewer_not_assigned"
	case errors.Is(err, evidenceaction.ErrReasonRequired):
		return "reason_required"
	case errors.Is(err, evidenceaction.ErrProofMissing):
		return "evidence_missing"
	default:
		return "invalid_review"
	}
}

func writeReviewError(w http.ResponseWriter, req *http.Request, err error) {
	if errors.Is(err, os.ErrNotExist) {
		writeJSON(w, http.StatusNotFound, map[string]string{"code": "review_case_not_found", "message": "the immutable EPWA review case was not found"})
		return
	}
	writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"code": errorCode(err), "message": err.Error()})
}
