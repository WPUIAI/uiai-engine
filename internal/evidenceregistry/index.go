package evidenceregistry

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
)

func (s *Store) Index(ctx context.Context, input IndexInput) (IndexResult, error) {
	if s == nil || s.db == nil {
		return IndexResult{}, ErrIndexUnavailable
	}
	if err := validateIndexInput(s.projectRef, input); err != nil {
		return IndexResult{}, err
	}
	if input.ObservedAt.IsZero() {
		input.ObservedAt = s.now().UTC()
	} else {
		input.ObservedAt = input.ObservedAt.UTC()
	}
	inputDigest, err := digestIndexInput(input)
	if err != nil {
		return IndexResult{}, err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return IndexResult{}, fmt.Errorf("%w: begin index transaction: %v", ErrIndexUnavailable, err)
	}
	defer tx.Rollback()
	var existingManifest, existingInput string
	err = tx.QueryRowContext(ctx, `SELECT manifest_sha256, index_input_sha256 FROM artifact WHERE artifact_ref = ? AND revision = ?`, input.Manifest.ArtifactID, input.Manifest.Revision).Scan(&existingManifest, &existingInput)
	if err == nil {
		if existingManifest != input.ManifestSHA256 {
			return IndexResult{}, fmt.Errorf("%w: immutable identity digest conflict", ErrInputInvalid)
		}
		if existingInput == inputDigest {
			var indexRevision uint64
			if err := tx.QueryRowContext(ctx, `SELECT revision FROM registry_meta WHERE id = 1`).Scan(&indexRevision); err != nil {
				return IndexResult{}, fmt.Errorf("%w: read index revision: %v", ErrIndexCorrupt, err)
			}
			return IndexResult{ArtifactRef: input.Manifest.ArtifactID, Revision: input.Manifest.Revision, IndexRevision: indexRevision, InputSHA256: inputDigest, Deduplicated: true}, nil
		}
	} else if err != sql.ErrNoRows {
		return IndexResult{}, fmt.Errorf("%w: resolve artifact identity: %v", ErrIndexUnavailable, err)
	}

	acceptances, err := completeAcceptances(input)
	if err != nil {
		return IndexResult{}, err
	}
	closures := buildClosures(input, acceptances)
	artifactClosure := aggregateClosure(closures)
	capturedAt, err := time.Parse(time.RFC3339Nano, input.Manifest.CapturedAt)
	if err != nil {
		return IndexResult{}, fmt.Errorf("%w: captured_at: %v", ErrInputInvalid, err)
	}
	kindsJSON, _ := json.Marshal(input.Manifest.Kinds)
	if _, err := tx.ExecContext(ctx, `INSERT INTO artifact(
		artifact_ref, revision, manifest_sha256, index_input_sha256, bundle_sha256, title, summary, kinds_json,
		project_ref, workstream_ref, workset_ref, callgraph_ref, workpoint_ref, verification, access_class,
		redaction_state, closure_posture, captured_at, freshness_observed_at, pwa_path, indexed_at
	) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
	ON CONFLICT(artifact_ref, revision) DO UPDATE SET
		index_input_sha256=excluded.index_input_sha256, bundle_sha256=excluded.bundle_sha256,
		title=excluded.title, summary=excluded.summary, kinds_json=excluded.kinds_json,
		workstream_ref=excluded.workstream_ref, workset_ref=excluded.workset_ref,
		callgraph_ref=excluded.callgraph_ref, workpoint_ref=excluded.workpoint_ref,
		verification=excluded.verification, access_class=excluded.access_class,
		redaction_state=excluded.redaction_state, closure_posture=excluded.closure_posture,
		freshness_observed_at=excluded.freshness_observed_at, pwa_path=excluded.pwa_path, indexed_at=excluded.indexed_at`,
		input.Manifest.ArtifactID, input.Manifest.Revision, input.ManifestSHA256, inputDigest,
		input.Manifest.Integrity.BundleSHA256, input.Manifest.Title, input.Manifest.Summary, string(kindsJSON),
		s.projectRef, input.Manifest.Scope.Workstream.WorkstreamRef, input.Manifest.Scope.Workset.WorksetRef,
		callGraphRef(input.Manifest.Scope.CallGraph), input.Manifest.Scope.Workpoint.WorkpointRef,
		input.Manifest.Verification.Status, input.Manifest.Policy.AccessClass, input.Manifest.Policy.RedactionState,
		artifactClosure, capturedAt.UTC().UnixNano(), input.ObservedAt.UnixNano(), input.Manifest.Links.PWAPath, input.ObservedAt.UnixNano(),
	); err != nil {
		return IndexResult{}, fmt.Errorf("%w: upsert artifact: %v", ErrIndexUnavailable, err)
	}
	if err := replaceBindings(ctx, tx, input, acceptances, closures); err != nil {
		return IndexResult{}, err
	}
	if err := replaceSearchDocument(ctx, tx, input, acceptances); err != nil {
		return IndexResult{}, err
	}
	var indexRevision uint64
	if err := tx.QueryRowContext(ctx, `UPDATE registry_meta SET revision = revision + 1, state = ?, stale_reason = '', updated_at = ? WHERE id = 1 RETURNING revision`, IndexReady, input.ObservedAt.UnixNano()).Scan(&indexRevision); err != nil {
		return IndexResult{}, fmt.Errorf("%w: advance index revision: %v", ErrIndexUnavailable, err)
	}
	if err := tx.Commit(); err != nil {
		return IndexResult{}, fmt.Errorf("%w: commit index: %v", ErrIndexUnavailable, err)
	}
	return IndexResult{ArtifactRef: input.Manifest.ArtifactID, Revision: input.Manifest.Revision, IndexRevision: indexRevision, InputSHA256: inputDigest}, nil
}

func validateIndexInput(projectRef string, input IndexInput) error {
	if evidenceartifact.Validate(input.Manifest) != nil || input.Manifest.Scope.Project.ProjectRef != projectRef || !validSHA256(input.ManifestSHA256) {
		return ErrInputInvalid
	}
	if input.Manifest.Integrity.ManifestSHA256 != "" && input.Manifest.Integrity.ManifestSHA256 != input.ManifestSHA256 {
		return ErrInputInvalid
	}
	return nil
}

func digestIndexInput(input IndexInput) (string, error) {
	input.ObservedAt = time.Time{}
	body, err := json.Marshal(input)
	if err != nil {
		return "", fmt.Errorf("%w: encode input: %v", ErrInputInvalid, err)
	}
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:]), nil
}

func completeAcceptances(input IndexInput) ([]AcceptanceBinding, error) {
	required := map[string]struct{}{}
	for _, item := range input.Manifest.Scope.WorkItems {
		for _, ref := range item.AcceptanceAtomRefs {
			required[ref] = struct{}{}
		}
	}
	for _, claim := range input.Manifest.Claims {
		for _, ref := range claim.AcceptanceAtomRefs {
			required[ref] = struct{}{}
		}
	}
	provided := make(map[string]AcceptanceBinding, len(input.Acceptances))
	for _, binding := range input.Acceptances {
		if strings.TrimSpace(binding.AcceptanceAtomRef) == "" || !validAcceptanceState(binding.State) {
			return nil, ErrInputInvalid
		}
		if _, ok := required[binding.AcceptanceAtomRef]; !ok {
			return nil, fmt.Errorf("%w: acceptance %q not required by artifact", ErrInputInvalid, binding.AcceptanceAtomRef)
		}
		if _, duplicate := provided[binding.AcceptanceAtomRef]; duplicate {
			return nil, fmt.Errorf("%w: duplicate acceptance %q", ErrInputInvalid, binding.AcceptanceAtomRef)
		}
		provided[binding.AcceptanceAtomRef] = binding
	}
	refs := make([]string, 0, len(required))
	for ref := range required {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	out := make([]AcceptanceBinding, 0, len(refs))
	for _, ref := range refs {
		binding, ok := provided[ref]
		if !ok {
			binding = AcceptanceBinding{AcceptanceAtomRef: ref, State: AcceptanceMissing}
		}
		out = append(out, binding)
	}
	return out, nil
}

func replaceBindings(ctx context.Context, tx *sql.Tx, input IndexInput, acceptances []AcceptanceBinding, closures []ClosureProjection) error {
	artifactRef, revision := input.Manifest.ArtifactID, input.Manifest.Revision
	for _, table := range []string{"work_item_binding", "acceptance_binding"} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE artifact_ref = ? AND artifact_revision = ?`, artifactRef, revision); err != nil {
			return fmt.Errorf("%w: clear %s: %v", ErrIndexUnavailable, table, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM artifact_edge WHERE project_ref = ? AND source_ref = ? AND source_revision = ?`, input.Manifest.Scope.Project.ProjectRef, artifactRef, strconv.FormatUint(revision, 10)); err != nil {
		return fmt.Errorf("%w: clear edges: %v", ErrIndexUnavailable, err)
	}
	for _, item := range input.Manifest.Scope.WorkItems {
		parents, _ := json.Marshal(item.ParentRefs)
		dependencies, _ := json.Marshal(item.DependencyRefs)
		blockers, _ := json.Marshal(item.BlockerRefs)
		atoms, _ := json.Marshal(item.AcceptanceAtomRefs)
		evidenceRequirements, _ := json.Marshal(item.EvidenceRequirementRefs)
		reviewRequirements, _ := json.Marshal(item.ReviewRequirementRefs)
		if _, err := tx.ExecContext(ctx, `INSERT INTO work_item_binding(
			artifact_ref, artifact_revision, project_ref, work_item_ref, provider_surface, item_id,
			item_type, title, description, description_ref, description_sha256, item_revision, item_digest,
			status_at_capture, parent_refs_json, dependency_refs_json, blocker_refs_json,
			acceptance_atom_refs_json, evidence_requirement_refs_json, review_requirement_refs_json, closure_posture
		) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			artifactRef, revision, input.Manifest.Scope.Project.ProjectRef, item.WorkItemRef, item.ProviderSurface,
			item.ItemID, item.ItemType, item.Title, item.Description, item.DescriptionRef, item.DescriptionSHA256,
			item.Revision, item.Digest, item.StatusAtCapture, string(parents), string(dependencies), string(blockers),
			string(atoms), string(evidenceRequirements), string(reviewRequirements), item.ClosurePosture); err != nil {
			return fmt.Errorf("%w: work item: %v", ErrIndexUnavailable, err)
		}
		if err := insertEdge(ctx, tx, input, item.WorkItemRef, item.Revision, RelationWorkItem, ""); err != nil {
			return err
		}
	}
	for _, binding := range acceptances {
		verifiers, _ := json.Marshal(binding.VerifierRefs)
		if _, err := tx.ExecContext(ctx, `INSERT INTO acceptance_binding VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?)`, artifactRef, revision, input.Manifest.Scope.Project.ProjectRef, workItemForAtom(input.Manifest, binding.AcceptanceAtomRef), binding.AcceptanceAtomRef, binding.Revision, binding.State, binding.VerifierClass, string(verifiers), binding.DecisionRef, binding.ReceiptRef, boolInt(binding.Fresh), boolInt(binding.ScopeMatched)); err != nil {
			return fmt.Errorf("%w: acceptance: %v", ErrIndexUnavailable, err)
		}
		if err := insertEdge(ctx, tx, input, binding.AcceptanceAtomRef, binding.Revision, RelationAcceptanceAtom, binding.ReceiptRef); err != nil {
			return err
		}
	}
	if input.CompletionCaseRef != "" {
		if err := insertEdge(ctx, tx, input, input.CompletionCaseRef, "", RelationCompletionCase, input.ProviderCloseReceiptRef); err != nil {
			return err
		}
	}
	for _, receipt := range input.Manifest.ReceiptRefs {
		if err := insertEdge(ctx, tx, input, receipt, "", RelationReceipt, receipt); err != nil {
			return err
		}
	}
	for _, related := range input.Manifest.Links.RelatedRefs {
		if err := insertEdge(ctx, tx, input, related, "", RelationRelated, ""); err != nil {
			return err
		}
	}
	if input.Manifest.Links.SupersedesRef != "" {
		if err := insertEdge(ctx, tx, input, input.Manifest.Links.SupersedesRef, "", RelationSupersedes, ""); err != nil {
			return err
		}
	}
	for _, closure := range closures {
		if _, err := tx.ExecContext(ctx, `INSERT INTO closure_binding(
			project_ref, work_item_ref, completion_case_ref, completion_contract_ref, required_atoms,
			accepted_atoms, blocked_atoms, stale_atoms, posture, completion_decision_ref,
			provider_close_receipt_ref, reopen_ref, settlement_posture, observed_at
		) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?)
			ON CONFLICT(project_ref, work_item_ref, completion_case_ref) DO UPDATE SET
			completion_contract_ref=excluded.completion_contract_ref, required_atoms=excluded.required_atoms,
			accepted_atoms=excluded.accepted_atoms, blocked_atoms=excluded.blocked_atoms, stale_atoms=excluded.stale_atoms,
			posture=excluded.posture, completion_decision_ref=excluded.completion_decision_ref,
			provider_close_receipt_ref=excluded.provider_close_receipt_ref, reopen_ref=excluded.reopen_ref,
			settlement_posture=excluded.settlement_posture, observed_at=excluded.observed_at`,
			closure.ProjectRef, closure.WorkItemRef, closure.CompletionCaseRef, closure.CompletionContractRef,
			closure.RequiredAtoms, closure.AcceptedAtoms, closure.BlockedAtoms, closure.StaleAtoms, closure.Posture,
			closure.CompletionDecisionRef, closure.ProviderCloseReceiptRef, closure.ReopenRef, closure.SettlementPosture, closure.ObservedAt.UnixNano()); err != nil {
			return fmt.Errorf("%w: closure: %v", ErrIndexUnavailable, err)
		}
	}
	if err := reconcileProviderBindings(ctx, tx, input.Manifest.Scope.Project.ProjectRef); err != nil {
		return err
	}
	return nil
}

func insertEdge(ctx context.Context, tx *sql.Tx, input IndexInput, targetRef, targetRevision string, relation RelationType, receipt string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO artifact_edge VALUES(?,?,?,?,?,?,?,?)`, input.Manifest.Scope.Project.ProjectRef, input.Manifest.ArtifactID, strconv.FormatUint(input.Manifest.Revision, 10), targetRef, targetRevision, relation, receipt, input.ObservedAt.UnixNano())
	if err != nil {
		return fmt.Errorf("%w: edge %s: %v", ErrIndexUnavailable, relation, err)
	}
	return nil
}

func replaceSearchDocument(ctx context.Context, tx *sql.Tx, input IndexInput, acceptances []AcceptanceBinding) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM artifact_search WHERE artifact_ref = ? AND revision = ?`, input.Manifest.ArtifactID, input.Manifest.Revision); err != nil {
		return fmt.Errorf("%w: clear search document: %v", ErrIndexUnavailable, err)
	}
	items := make([]string, 0, len(input.Manifest.Scope.WorkItems)*12)
	for _, item := range input.Manifest.Scope.WorkItems {
		items = append(items, item.WorkItemRef, item.ItemID, item.ItemType, item.Title, item.Description, item.DescriptionRef,
			strings.Join(item.ParentRefs, " "), strings.Join(item.DependencyRefs, " "), strings.Join(item.BlockerRefs, " "),
			strings.Join(item.EvidenceRequirementRefs, " "), strings.Join(item.ReviewRequirementRefs, " "))
	}
	atoms := make([]string, 0, len(acceptances))
	for _, binding := range acceptances {
		atoms = append(atoms, binding.AcceptanceAtomRef)
	}
	_, err := tx.ExecContext(ctx, `INSERT INTO artifact_search VALUES(?,?,?,?,?,?,?)`, input.Manifest.ArtifactID, input.Manifest.Revision, input.Manifest.Title, input.Manifest.Summary, strings.Join(input.Manifest.Kinds, " "), strings.Join(items, " "), strings.Join(atoms, " "))
	if err != nil {
		return fmt.Errorf("%w: search document: %v", ErrIndexUnavailable, err)
	}
	return nil
}

func buildClosures(input IndexInput, bindings []AcceptanceBinding) []ClosureProjection {
	byRef := make(map[string]AcceptanceBinding, len(bindings))
	for _, binding := range bindings {
		byRef[binding.AcceptanceAtomRef] = binding
	}
	out := make([]ClosureProjection, 0, len(input.Manifest.Scope.WorkItems))
	for _, item := range input.Manifest.Scope.WorkItems {
		projection := ClosureProjection{Schema: ClosureSchemaV1, ProjectRef: input.Manifest.Scope.Project.ProjectRef, WorkItemRef: item.WorkItemRef, CompletionCaseRef: input.CompletionCaseRef, CompletionContractRef: input.CompletionContractRef, CompletionDecisionRef: input.CompletionDecisionRef, ProviderCloseReceiptRef: input.ProviderCloseReceiptRef, ReopenRef: input.ReopenRef, SettlementPosture: input.SettlementPosture, ObservedAt: input.ObservedAt}
		for _, atomRef := range uniqueStrings(item.AcceptanceAtomRefs) {
			projection.RequiredAtoms++
			binding := byRef[atomRef]
			switch binding.State {
			case AcceptanceAccepted:
				if binding.Fresh && binding.ScopeMatched {
					projection.AcceptedAtoms++
				} else {
					projection.StaleAtoms++
				}
			case AcceptanceStale:
				projection.StaleAtoms++
			default:
				projection.BlockedAtoms++
			}
		}
		projection.Posture = closurePosture(projection)
		out = append(out, projection)
	}
	return out
}

func closurePosture(projection ClosureProjection) ClosurePosture {
	switch {
	case projection.ReopenRef != "":
		return ClosureReopened
	case projection.CompletionDecisionRef != "" || projection.ProviderCloseReceiptRef != "":
		return ClosureCompleted
	case projection.RequiredAtoms == 0:
		return ClosureIneligible
	case projection.StaleAtoms > 0:
		return ClosureStale
	case projection.BlockedAtoms > 0:
		return ClosureBlocked
	case projection.AcceptedAtoms == projection.RequiredAtoms:
		return ClosureEligible
	default:
		return ClosureBlocked
	}
}

func aggregateClosure(closures []ClosureProjection) ClosurePosture {
	if len(closures) == 0 {
		return ClosureIneligible
	}
	allCompleted := true
	for _, closure := range closures {
		if closure.Posture == ClosureReopened {
			return ClosureReopened
		}
		if closure.Posture == ClosureStale {
			return ClosureStale
		}
		if closure.Posture == ClosureBlocked {
			return ClosureBlocked
		}
		if closure.Posture == ClosureIneligible {
			return ClosureIneligible
		}
		allCompleted = allCompleted && closure.Posture == ClosureCompleted
	}
	if allCompleted {
		return ClosureCompleted
	}
	return ClosureEligible
}

func workItemForAtom(manifest evidenceartifact.Manifest, atomRef string) string {
	for _, item := range manifest.Scope.WorkItems {
		for _, ref := range item.AcceptanceAtomRefs {
			if ref == atomRef {
				return item.WorkItemRef
			}
		}
	}
	return ""
}

func callGraphRef(binding evidenceartifact.CallGraphBinding) string {
	for _, ref := range []string{binding.FrameRef, binding.RunRef, binding.DefinitionRef} {
		if ref != "" {
			return ref
		}
	}
	return ""
}

func validSHA256(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && hex.EncodeToString(decoded) == value
}

func validAcceptanceState(value AcceptanceState) bool {
	switch value {
	case AcceptanceMissing, AcceptancePending, AcceptanceAccepted, AcceptanceRejected, AcceptanceBlocked, AcceptanceStale, AcceptanceIndeterminate:
		return true
	default:
		return false
	}
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
