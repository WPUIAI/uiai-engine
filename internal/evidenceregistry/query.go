package evidenceregistry

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type listCursor struct {
	CapturedAt  int64  `json:"captured_at"`
	ArtifactRef string `json:"artifact_ref"`
	Revision    uint64 `json:"revision"`
}

func (s *Store) List(ctx context.Context, query Query) (Page, error) {
	if s == nil || s.db == nil {
		return Page{}, ErrIndexUnavailable
	}
	if query.ProjectRef == "" {
		query.ProjectRef = s.projectRef
	}
	if query.ProjectRef != s.projectRef || utf8.RuneCountInString(query.Text) > MaxQueryRunes {
		return Page{}, ErrInputInvalid
	}
	profile, pageSize, mediaPosture, err := resourcePage(query.ResourceProfile, query.PageSize, 50)
	if err != nil {
		return Page{}, err
	}
	query.PageSize = pageSize
	cursor, err := decodeCursor(query.Cursor)
	if err != nil {
		return Page{}, err
	}
	clauses := []string{"a.project_ref = ?"}
	args := []any{s.projectRef}
	join := ""
	text := strings.TrimSpace(query.Text)
	if text != "" {
		if strings.HasPrefix(text, "artifact:") {
			clauses = append(clauses, "a.artifact_ref = ?")
			args = append(args, text)
		} else if validSHA256(text) {
			clauses = append(clauses, "(a.manifest_sha256 = ? OR a.bundle_sha256 = ?)")
			args = append(args, text, text)
		} else {
			fts, err := safeFTSQuery(text)
			if err != nil {
				return Page{}, err
			}
			join = " JOIN artifact_search s ON s.artifact_ref = a.artifact_ref AND CAST(s.revision AS INTEGER) = a.revision "
			clauses = append(clauses, "artifact_search MATCH ?")
			args = append(args, fts)
		}
	}
	for value, column := range map[string]string{
		query.WorkstreamRef: "a.workstream_ref", query.WorksetRef: "a.workset_ref",
		query.WorkpointRef: "a.workpoint_ref", query.Verification: "a.verification",
		query.Access: "a.access_class", query.Closure: "a.closure_posture",
	} {
		if value != "" {
			clauses = append(clauses, column+" = ?")
			args = append(args, value)
		}
	}
	if query.WorkItemRef != "" {
		clauses = append(clauses, `EXISTS (SELECT 1 FROM work_item_binding wi WHERE wi.artifact_ref = a.artifact_ref AND wi.artifact_revision = a.revision AND wi.work_item_ref = ?)`)
		args = append(args, query.WorkItemRef)
	}
	if query.WorkItemType != "" {
		clauses = append(clauses, `EXISTS (SELECT 1 FROM work_item_binding wi WHERE wi.artifact_ref = a.artifact_ref AND wi.artifact_revision = a.revision AND wi.item_type = ?)`)
		args = append(args, query.WorkItemType)
	}
	if query.AcceptanceAtomRef != "" {
		clauses = append(clauses, `EXISTS (SELECT 1 FROM acceptance_binding ab WHERE ab.artifact_ref = a.artifact_ref AND ab.artifact_revision = a.revision AND ab.acceptance_atom_ref = ?)`)
		args = append(args, query.AcceptanceAtomRef)
	}
	if query.Cursor != "" {
		clauses = append(clauses, `(a.captured_at < ? OR (a.captured_at = ? AND (a.artifact_ref < ? OR (a.artifact_ref = ? AND a.revision < ?))))`)
		args = append(args, cursor.CapturedAt, cursor.CapturedAt, cursor.ArtifactRef, cursor.ArtifactRef, cursor.Revision)
	}
	args = append(args, query.PageSize+1)
	statement := artifactSelect + join + " WHERE " + strings.Join(clauses, " AND ") + " ORDER BY a.captured_at DESC, a.artifact_ref DESC, a.revision DESC LIMIT ?"
	rows, err := s.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return Page{}, fmt.Errorf("%w: list artifacts: %v", ErrIndexUnavailable, err)
	}
	defer rows.Close()
	items := make([]ArtifactRow, 0, query.PageSize+1)
	for rows.Next() {
		item, err := scanArtifact(rows)
		if err != nil {
			return Page{}, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return Page{}, fmt.Errorf("%w: iterate artifacts: %v", ErrIndexUnavailable, err)
	}
	page := Page{Schema: PageSchemaV1, ProjectRef: s.projectRef, Rows: items, PageSize: query.PageSize, TotalPosture: "omitted", ResourceProfile: profile, MediaPosture: mediaPosture, ObservedAt: s.now().UTC()}
	status, err := s.Status(ctx)
	if err != nil {
		return Page{}, err
	}
	page.IndexRevision, page.IndexState = status.Revision, status.State
	if len(page.Rows) > int(query.PageSize) {
		last := page.Rows[query.PageSize-1]
		page.NextCursor, err = encodeCursor(listCursor{CapturedAt: last.CapturedAt.UnixNano(), ArtifactRef: last.ArtifactRef, Revision: last.Revision})
		if err != nil {
			return Page{}, err
		}
		page.Rows = page.Rows[:query.PageSize]
	}
	return page, nil
}

func (s *Store) Resolve(ctx context.Context, artifactRef string, revision uint64) (ArtifactRow, error) {
	if s == nil || s.db == nil || strings.TrimSpace(artifactRef) == "" || revision == 0 {
		return ArtifactRow{}, ErrInputInvalid
	}
	row := s.db.QueryRowContext(ctx, artifactSelect+` WHERE a.project_ref = ? AND a.artifact_ref = ? AND a.revision = ?`, s.projectRef, artifactRef, revision)
	item, err := scanArtifact(row)
	if err == sql.ErrNoRows {
		return ArtifactRow{}, ErrNotFound
	}
	return item, err
}

func (s *Store) WorkItems(ctx context.Context, artifactRef string, revision uint64) ([]WorkItemSnapshot, error) {
	if s == nil || s.db == nil || strings.TrimSpace(artifactRef) == "" || revision == 0 {
		return nil, ErrInputInvalid
	}
	rows, err := s.db.QueryContext(ctx, `SELECT provider_surface, work_item_ref, item_id, item_type, title,
		description, description_ref, description_sha256, item_revision, item_digest, status_at_capture,
		parent_refs_json, dependency_refs_json, blocker_refs_json, acceptance_atom_refs_json,
		evidence_requirement_refs_json, review_requirement_refs_json, closure_posture
		FROM work_item_binding WHERE project_ref = ? AND artifact_ref = ? AND artifact_revision = ?
		ORDER BY work_item_ref`, s.projectRef, artifactRef, revision)
	if err != nil {
		return nil, fmt.Errorf("%w: query work items: %v", ErrIndexUnavailable, err)
	}
	defer rows.Close()
	items := make([]WorkItemSnapshot, 0)
	for rows.Next() {
		var item WorkItemSnapshot
		var parents, dependencies, blockers, atoms, evidenceRequirements, reviewRequirements string
		if err := rows.Scan(&item.ProviderSurface, &item.WorkItemRef, &item.ItemID, &item.ItemType, &item.Title,
			&item.Description, &item.DescriptionRef, &item.DescriptionSHA256, &item.Revision, &item.Digest,
			&item.StatusAtCapture, &parents, &dependencies, &blockers, &atoms, &evidenceRequirements,
			&reviewRequirements, &item.ClosurePosture); err != nil {
			return nil, fmt.Errorf("%w: scan work item: %v", ErrIndexCorrupt, err)
		}
		encoded := []string{parents, dependencies, blockers, atoms, evidenceRequirements, reviewRequirements}
		decoded := []*[]string{&item.ParentRefs, &item.DependencyRefs, &item.BlockerRefs, &item.AcceptanceAtomRefs, &item.EvidenceRequirementRefs, &item.ReviewRequirementRefs}
		for i := range encoded {
			if err := json.Unmarshal([]byte(encoded[i]), decoded[i]); err != nil {
				return nil, fmt.Errorf("%w: decode work item refs: %v", ErrIndexCorrupt, err)
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: iterate work items: %v", ErrIndexUnavailable, err)
	}
	if len(items) == 0 {
		if _, err := s.Resolve(ctx, artifactRef, revision); err != nil {
			return nil, err
		}
	}
	return items, nil
}

func (s *Store) Edges(ctx context.Context, objectRef string, direction EdgeDirection, relation RelationType, limit uint32) ([]Edge, error) {
	if s == nil || s.db == nil || strings.TrimSpace(objectRef) == "" {
		return nil, ErrInputInvalid
	}
	if limit == 0 {
		limit = 50
	}
	if limit > MaxPageSize || (direction != DirectionForward && direction != DirectionReverse) {
		return nil, ErrInputInvalid
	}
	column, order := "source_ref", "target_ref"
	if direction == DirectionReverse {
		column, order = "target_ref", "source_ref"
	}
	clauses := []string{"project_ref = ?", column + " = ?"}
	args := []any{s.projectRef, objectRef}
	if relation != "" {
		clauses = append(clauses, "relation = ?")
		args = append(args, relation)
	}
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, `SELECT source_ref, source_revision, target_ref, target_revision, relation, provenance_receipt, observed_at FROM artifact_edge WHERE `+strings.Join(clauses, " AND ")+` ORDER BY relation, `+order+` LIMIT ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: query edges: %v", ErrIndexUnavailable, err)
	}
	defer rows.Close()
	out := make([]Edge, 0, limit)
	for rows.Next() {
		var edge Edge
		var observed int64
		edge.Schema, edge.ProjectRef = EdgeSchemaV1, s.projectRef
		if err := rows.Scan(&edge.SourceRef, &edge.SourceRevision, &edge.TargetRef, &edge.TargetRevision, &edge.Relation, &edge.ProvenanceReceipt, &observed); err != nil {
			return nil, fmt.Errorf("%w: scan edge: %v", ErrIndexCorrupt, err)
		}
		edge.ObservedAt = time.Unix(0, observed).UTC()
		out = append(out, edge)
	}
	return out, rows.Err()
}

func (s *Store) Closure(ctx context.Context, workItemRef, completionCaseRef string) (ClosureProjection, error) {
	if s == nil || s.db == nil || strings.TrimSpace(workItemRef) == "" {
		return ClosureProjection{}, ErrInputInvalid
	}
	statement := `SELECT completion_case_ref, completion_contract_ref, required_atoms, accepted_atoms, blocked_atoms, stale_atoms, posture, completion_decision_ref, provider_close_receipt_ref, reopen_ref, settlement_posture, observed_at FROM closure_binding WHERE project_ref = ? AND work_item_ref = ?`
	args := []any{s.projectRef, workItemRef}
	if completionCaseRef != "" {
		statement += " AND completion_case_ref = ?"
		args = append(args, completionCaseRef)
	}
	statement += " ORDER BY observed_at DESC LIMIT 1"
	var projection ClosureProjection
	var observed int64
	projection.Schema, projection.ProjectRef, projection.WorkItemRef = ClosureSchemaV1, s.projectRef, workItemRef
	err := s.db.QueryRowContext(ctx, statement, args...).Scan(&projection.CompletionCaseRef, &projection.CompletionContractRef, &projection.RequiredAtoms, &projection.AcceptedAtoms, &projection.BlockedAtoms, &projection.StaleAtoms, &projection.Posture, &projection.CompletionDecisionRef, &projection.ProviderCloseReceiptRef, &projection.ReopenRef, &projection.SettlementPosture, &observed)
	if err == sql.ErrNoRows {
		return ClosureProjection{}, ErrNotFound
	}
	if err != nil {
		return ClosureProjection{}, fmt.Errorf("%w: query closure: %v", ErrIndexUnavailable, err)
	}
	projection.ObservedAt = time.Unix(0, observed).UTC()
	return projection, nil
}

const artifactSelect = `SELECT
	a.artifact_ref, a.revision, a.manifest_sha256, a.bundle_sha256, a.title, a.summary, a.kinds_json,
	a.project_ref, a.workstream_ref, a.workset_ref, a.callgraph_ref, a.workpoint_ref,
	COALESCE((SELECT wi.work_item_ref FROM work_item_binding wi WHERE wi.artifact_ref = a.artifact_ref AND wi.artifact_revision = a.revision ORDER BY wi.work_item_ref LIMIT 1), ''),
	COALESCE((SELECT wi.item_type FROM work_item_binding wi WHERE wi.artifact_ref = a.artifact_ref AND wi.artifact_revision = a.revision ORDER BY wi.work_item_ref LIMIT 1), ''),
	COALESCE((SELECT wi.title FROM work_item_binding wi WHERE wi.artifact_ref = a.artifact_ref AND wi.artifact_revision = a.revision ORDER BY wi.work_item_ref LIMIT 1), ''),
	(SELECT COUNT(*) FROM work_item_binding wi WHERE wi.artifact_ref = a.artifact_ref AND wi.artifact_revision = a.revision),
	(SELECT COUNT(*) FROM acceptance_binding ab WHERE ab.artifact_ref = a.artifact_ref AND ab.artifact_revision = a.revision),
	(SELECT COUNT(*) FROM acceptance_binding ab WHERE ab.artifact_ref = a.artifact_ref AND ab.artifact_revision = a.revision AND ab.state = 'accepted' AND ab.fresh = 1 AND ab.scope_matched = 1),
	a.verification, a.access_class, a.redaction_state, a.closure_posture, a.captured_at, a.freshness_observed_at, a.pwa_path
FROM artifact a`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanArtifact(row rowScanner) (ArtifactRow, error) {
	var item ArtifactRow
	var kinds string
	var capturedAt, freshness int64
	err := row.Scan(&item.ArtifactRef, &item.Revision, &item.ManifestSHA256, &item.BundleSHA256, &item.Title, &item.Summary, &kinds,
		&item.ProjectRef, &item.WorkstreamRef, &item.WorksetRef, &item.CallGraphRef, &item.WorkpointRef,
		&item.FirstWorkItemRef, &item.FirstWorkItemType, &item.FirstWorkItemTitle,
		&item.WorkItemCount, &item.AcceptanceTotal, &item.AcceptanceAccepted,
		&item.Verification, &item.Access, &item.Redaction, &item.Closure, &capturedAt, &freshness, &item.PWAPath)
	if err != nil {
		if err == sql.ErrNoRows {
			return ArtifactRow{}, err
		}
		return ArtifactRow{}, fmt.Errorf("%w: scan artifact: %v", ErrIndexCorrupt, err)
	}
	if err := json.Unmarshal([]byte(kinds), &item.Kinds); err != nil {
		return ArtifactRow{}, fmt.Errorf("%w: kinds: %v", ErrIndexCorrupt, err)
	}
	item.CapturedAt = time.Unix(0, capturedAt).UTC()
	item.FreshnessObservedAt = time.Unix(0, freshness).UTC()
	return item, nil
}

func safeFTSQuery(value string) (string, error) {
	fields := strings.Fields(value)
	if len(fields) == 0 {
		return "", ErrInputInvalid
	}
	parts := make([]string, 0, len(fields))
	for _, field := range fields {
		field = strings.ReplaceAll(field, `"`, `""`)
		parts = append(parts, `"`+field+`"*`)
	}
	return strings.Join(parts, " AND "), nil
}

func encodeCursor(cursor listCursor) (string, error) {
	body, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("%w: encode: %v", ErrCursorInvalid, err)
	}
	return base64.RawURLEncoding.EncodeToString(body), nil
}

func decodeCursor(value string) (listCursor, error) {
	if value == "" {
		return listCursor{}, nil
	}
	body, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return listCursor{}, ErrCursorInvalid
	}
	var cursor listCursor
	if json.Unmarshal(body, &cursor) != nil || cursor.CapturedAt <= 0 || cursor.ArtifactRef == "" || cursor.Revision == 0 || strconv.FormatUint(cursor.Revision, 10) == "" {
		return listCursor{}, ErrCursorInvalid
	}
	return cursor, nil
}
