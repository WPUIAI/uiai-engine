package evidenceregistry

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

func (s *Store) ReplaceProviderGraph(ctx context.Context, input ProviderGraphInput) (ProviderGraphResult, error) {
	if s == nil || s.db == nil || input.Project.ProjectRef != s.projectRef || strings.TrimSpace(input.Project.ProjectID) == "" || strings.TrimSpace(input.Project.DisplayName) == "" || strings.TrimSpace(input.Project.Fingerprint) == "" || input.Project.ScopeSafety != "safe" {
		return ProviderGraphResult{}, ErrInputInvalid
	}
	observed := input.Project.ObservedAt.UTC()
	if observed.IsZero() {
		observed = s.now().UTC()
	}
	graphDigest := providerGraphDigest(input)
	var currentDigest string
	var currentItems, currentRevision uint64
	digestErr := s.db.QueryRowContext(ctx, `SELECT graph_digest, item_count FROM provider_graph_state WHERE project_ref = ?`, s.projectRef).Scan(&currentDigest, &currentItems)
	if digestErr != nil && digestErr != sql.ErrNoRows {
		return ProviderGraphResult{}, fmt.Errorf("%w: read provider graph state: %v", ErrIndexUnavailable, digestErr)
	}
	if digestErr == nil && currentDigest == graphDigest {
		if err := s.db.QueryRowContext(ctx, `SELECT revision FROM registry_meta WHERE id = 1`).Scan(&currentRevision); err != nil {
			return ProviderGraphResult{}, fmt.Errorf("%w: read registry revision: %v", ErrIndexUnavailable, err)
		}
		return ProviderGraphResult{ProjectRef: s.projectRef, Items: currentItems, Revision: currentRevision, Changed: false}, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return ProviderGraphResult{}, fmt.Errorf("%w: begin provider graph: %v", ErrIndexUnavailable, err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO project_projection(project_ref, project_id, display_name, fingerprint, workspace_kind, scope_safety, source_schema, source_revision, observed_at)
		VALUES(?,?,?,?,?,?,?,?,?) ON CONFLICT(project_ref) DO UPDATE SET project_id=excluded.project_id, display_name=excluded.display_name,
		fingerprint=excluded.fingerprint, workspace_kind=excluded.workspace_kind, scope_safety=excluded.scope_safety,
		source_schema=excluded.source_schema, source_revision=excluded.source_revision, observed_at=excluded.observed_at`,
		input.Project.ProjectRef, input.Project.ProjectID, input.Project.DisplayName, input.Project.Fingerprint,
		input.Project.WorkspaceKind, input.Project.ScopeSafety, input.Project.SourceSchema, input.Project.SourceRevision, observed.UnixNano()); err != nil {
		return ProviderGraphResult{}, fmt.Errorf("%w: upsert project: %v", ErrIndexUnavailable, err)
	}
	for _, table := range []string{"work_item_edge", "provider_work_item", "provider_work_item_search"} {
		if _, err := tx.ExecContext(ctx, `DELETE FROM `+table+` WHERE project_ref = ?`, s.projectRef); err != nil {
			return ProviderGraphResult{}, fmt.Errorf("%w: clear %s: %v", ErrIndexUnavailable, table, err)
		}
	}
	seen := make(map[string]ProviderWorkItem, len(input.Items))
	for i := range input.Items {
		item := input.Items[i]
		if item.ProjectRef != s.projectRef || strings.TrimSpace(item.WorkItemRef) == "" || strings.TrimSpace(item.ItemID) == "" || strings.TrimSpace(item.Title) == "" || strings.TrimSpace(item.Revision) == "" || !validSHA256(item.Digest) || item.BindingState == "" || item.SourceAuthority == "" {
			return ProviderGraphResult{}, ErrInputInvalid
		}
		if _, duplicate := seen[item.WorkItemRef]; duplicate {
			return ProviderGraphResult{}, fmt.Errorf("%w: duplicate work item %q", ErrInputInvalid, item.WorkItemRef)
		}
		seen[item.WorkItemRef] = item
		parents, _ := json.Marshal(uniqueStrings(item.ParentRefs))
		dependencies, _ := json.Marshal(uniqueStrings(item.DependencyRefs))
		blockers, _ := json.Marshal(uniqueStrings(item.BlockerRefs))
		acceptance, _ := json.Marshal(uniqueStrings(item.AcceptanceRefs))
		specs, _ := json.Marshal(uniqueStrings(item.SpecRefs))
		if _, err := tx.ExecContext(ctx, `INSERT INTO provider_work_item VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			item.ProjectRef, item.WorkItemRef, item.ProviderSurface, item.ItemID, item.ItemType, item.Title, item.Description,
			item.Status, item.Priority, item.Revision, item.Digest, string(parents), string(dependencies), string(blockers),
			string(acceptance), string(specs), item.ExternalRef, item.SourceAuthority, item.BindingState, observed.UnixNano()); err != nil {
			return ProviderGraphResult{}, fmt.Errorf("%w: insert provider work item: %v", ErrIndexUnavailable, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO provider_work_item_search VALUES(?,?,?,?,?,?)`, item.ProjectRef, item.WorkItemRef, item.ItemID, item.Title, item.Description, item.ExternalRef); err != nil {
			return ProviderGraphResult{}, fmt.Errorf("%w: insert provider work item search: %v", ErrIndexUnavailable, err)
		}
	}
	for _, item := range seen {
		for relation, refs := range map[string][]string{"parent": item.ParentRefs, "depends_on": item.DependencyRefs, "blocked_by": item.BlockerRefs} {
			for _, target := range uniqueStrings(refs) {
				targetRevision := ""
				if linked, ok := seen[target]; ok {
					targetRevision = linked.Revision
				}
				if _, err := tx.ExecContext(ctx, `INSERT INTO work_item_edge VALUES(?,?,?,?,?,?,?,?)`, s.projectRef, item.WorkItemRef, target, relation, item.Revision, targetRevision, item.SourceAuthority, observed.UnixNano()); err != nil {
					return ProviderGraphResult{}, fmt.Errorf("%w: insert work item edge: %v", ErrIndexUnavailable, err)
				}
			}
		}
	}
	if err := reconcileProviderBindings(ctx, tx, s.projectRef); err != nil {
		return ProviderGraphResult{}, err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO provider_graph_state(project_ref, graph_digest, item_count, updated_at) VALUES(?,?,?,?) ON CONFLICT(project_ref) DO UPDATE SET graph_digest=excluded.graph_digest, item_count=excluded.item_count, updated_at=excluded.updated_at`, s.projectRef, graphDigest, len(input.Items), observed.UnixNano()); err != nil {
		return ProviderGraphResult{}, fmt.Errorf("%w: write provider graph state: %v", ErrIndexUnavailable, err)
	}
	var revision uint64
	if err := tx.QueryRowContext(ctx, `UPDATE registry_meta SET revision = revision + 1, updated_at = ? WHERE id = 1 RETURNING revision`, observed.UnixNano()).Scan(&revision); err != nil {
		return ProviderGraphResult{}, fmt.Errorf("%w: advance provider graph revision: %v", ErrIndexUnavailable, err)
	}
	if err := tx.Commit(); err != nil {
		return ProviderGraphResult{}, fmt.Errorf("%w: commit provider graph: %v", ErrIndexUnavailable, err)
	}
	return ProviderGraphResult{ProjectRef: s.projectRef, Items: uint64(len(input.Items)), Revision: revision, Changed: true}, nil
}

func providerGraphDigest(input ProviderGraphInput) string {
	project := input.Project
	project.ObservedAt = time.Time{}
	items := append([]ProviderWorkItem(nil), input.Items...)
	for i := range items {
		items[i].ObservedAt = time.Time{}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].WorkItemRef < items[j].WorkItemRef })
	body, _ := json.Marshal(struct {
		Project ProjectProjection  `json:"project"`
		Items   []ProviderWorkItem `json:"items"`
	}{Project: project, Items: items})
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

func reconcileProviderBindings(ctx context.Context, tx *sql.Tx, projectRef string) error {
	_, err := tx.ExecContext(ctx, `UPDATE provider_work_item SET binding_state = CASE
		WHEN EXISTS (SELECT 1 FROM work_item_binding b WHERE b.project_ref = provider_work_item.project_ref AND b.work_item_ref = provider_work_item.work_item_ref AND b.item_revision = provider_work_item.revision AND b.item_digest = provider_work_item.digest) THEN 'artifact_bound'
		WHEN EXISTS (SELECT 1 FROM work_item_binding b WHERE b.project_ref = provider_work_item.project_ref AND b.work_item_ref = provider_work_item.work_item_ref) THEN 'focusa_binding_stale'
		ELSE 'focusa_binding_pending' END WHERE project_ref = ?`, projectRef)
	if err != nil {
		return fmt.Errorf("%w: reconcile provider bindings: %v", ErrIndexUnavailable, err)
	}
	return nil
}

func (s *Store) ListProviderWorkItems(ctx context.Context, query ProviderWorkItemQuery) (ProviderWorkItemPage, error) {
	if s == nil || s.db == nil || (query.ProjectRef != "" && query.ProjectRef != s.projectRef) {
		return ProviderWorkItemPage{}, ErrInputInvalid
	}
	profile, limit, mediaPosture, err := resourcePage(query.ResourceProfile, query.Limit, 100)
	if err != nil {
		return ProviderWorkItemPage{}, err
	}
	query.Limit = limit
	cursor, err := decodeProviderCursor(query.Cursor)
	if err != nil {
		return ProviderWorkItemPage{}, err
	}
	clauses, args := []string{"project_ref = ?"}, []any{s.projectRef}
	if query.Status != "" {
		clauses, args = append(clauses, "status = ?"), append(args, query.Status)
	}
	if query.ItemType != "" {
		clauses, args = append(clauses, "item_type = ?"), append(args, query.ItemType)
	}
	if text := strings.TrimSpace(query.Text); text != "" {
		switch {
		case validSHA256(text):
			clauses, args = append(clauses, "digest = ?"), append(args, text)
		case strings.HasPrefix(text, "work-item:"):
			clauses, args = append(clauses, "work_item_ref = ?"), append(args, text)
		default:
			fts, ftsErr := safeFTSQuery(text)
			if ftsErr != nil {
				return ProviderWorkItemPage{}, ftsErr
			}
			clauses, args = append(clauses, `work_item_ref IN (SELECT work_item_ref FROM provider_work_item_search WHERE project_ref = ? AND provider_work_item_search MATCH ?)`), append(args, s.projectRef, fts)
		}
	}
	if query.Cursor != "" {
		clauses = append(clauses, `(priority > ? OR (priority = ? AND (item_type > ? OR (item_type = ? AND item_id > ?))))`)
		args = append(args, cursor.Priority, cursor.Priority, cursor.ItemType, cursor.ItemType, cursor.ItemID)
	}
	args = append(args, query.Limit+1)
	rows, err := s.db.QueryContext(ctx, `SELECT project_ref, work_item_ref, provider_surface, item_id, item_type, title, description,
		status, priority, revision, digest, parent_refs_json, dependency_refs_json, blocker_refs_json, acceptance_refs_json,
		spec_refs_json, external_ref, source_authority, binding_state, observed_at FROM provider_work_item WHERE `+strings.Join(clauses, " AND ")+` ORDER BY priority, item_type, item_id LIMIT ?`, args...)
	if err != nil {
		return ProviderWorkItemPage{}, fmt.Errorf("%w: list provider work items: %v", ErrIndexUnavailable, err)
	}
	defer rows.Close()
	items, err := scanProviderWorkItems(rows)
	if err != nil {
		return ProviderWorkItemPage{}, err
	}
	page := ProviderWorkItemPage{Schema: "uiai.evidence_provider_work_items.v1", ProjectRef: s.projectRef, WorkItems: items, PageSize: query.Limit, ResourceProfile: profile, MediaPosture: mediaPosture, ObservedAt: s.now().UTC()}
	if len(page.WorkItems) > int(query.Limit) {
		last := page.WorkItems[query.Limit-1]
		page.NextCursor, err = encodeProviderCursor(providerCursor{Priority: last.Priority, ItemType: last.ItemType, ItemID: last.ItemID})
		if err != nil {
			return ProviderWorkItemPage{}, err
		}
		page.WorkItems = page.WorkItems[:query.Limit]
	}
	if err := s.db.QueryRowContext(ctx, `SELECT revision FROM registry_meta WHERE id = 1`).Scan(&page.IndexRevision); err != nil {
		return ProviderWorkItemPage{}, fmt.Errorf("%w: read provider index revision: %v", ErrIndexUnavailable, err)
	}
	return page, nil
}

type providerCursor struct {
	Priority int    `json:"priority"`
	ItemType string `json:"item_type"`
	ItemID   string `json:"item_id"`
}

func encodeProviderCursor(cursor providerCursor) (string, error) {
	body, err := json.Marshal(cursor)
	if err != nil {
		return "", fmt.Errorf("%w: encode provider cursor", ErrCursorInvalid)
	}
	return base64.RawURLEncoding.EncodeToString(body), nil
}

func decodeProviderCursor(value string) (providerCursor, error) {
	if value == "" {
		return providerCursor{}, nil
	}
	body, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil || len(body) > 2048 {
		return providerCursor{}, ErrCursorInvalid
	}
	var cursor providerCursor
	if json.Unmarshal(body, &cursor) != nil || cursor.ItemType == "" || cursor.ItemID == "" {
		return providerCursor{}, ErrCursorInvalid
	}
	return cursor, nil
}

func (s *Store) ProviderWorkItemEdges(ctx context.Context, objectRef string, direction EdgeDirection, relation string, limit uint32) ([]ProviderWorkItemEdge, error) {
	if s == nil || s.db == nil || strings.TrimSpace(objectRef) == "" {
		return nil, ErrInputInvalid
	}
	if direction == "" {
		direction = DirectionForward
	}
	if (direction != DirectionForward && direction != DirectionReverse) || limit == 0 || limit > MaxPageSize {
		return nil, ErrInputInvalid
	}
	column := "source_ref"
	if direction == DirectionReverse {
		column = "target_ref"
	}
	args := []any{s.projectRef, objectRef}
	filter := ""
	if relation != "" {
		filter = " AND relation = ?"
		args = append(args, relation)
	}
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, `SELECT project_ref, source_ref, target_ref, relation, source_revision, target_revision, source_authority, observed_at FROM work_item_edge WHERE project_ref = ? AND `+column+` = ?`+filter+` ORDER BY relation, source_ref, target_ref LIMIT ?`, args...)
	if err != nil {
		return nil, fmt.Errorf("%w: list provider work item edges: %v", ErrIndexUnavailable, err)
	}
	defer rows.Close()
	out := make([]ProviderWorkItemEdge, 0)
	for rows.Next() {
		var edge ProviderWorkItemEdge
		var observed int64
		if err := rows.Scan(&edge.ProjectRef, &edge.SourceRef, &edge.TargetRef, &edge.Relation, &edge.SourceRevision, &edge.TargetRevision, &edge.SourceAuthority, &observed); err != nil {
			return nil, fmt.Errorf("%w: scan provider work item edge: %v", ErrIndexCorrupt, err)
		}
		edge.ObservedAt = time.Unix(0, observed).UTC()
		out = append(out, edge)
	}
	return out, rows.Err()
}

func scanProviderWorkItems(rows *sql.Rows) ([]ProviderWorkItem, error) {
	out := make([]ProviderWorkItem, 0)
	for rows.Next() {
		var item ProviderWorkItem
		var parents, dependencies, blockers, acceptance, specs string
		var observed int64
		if err := rows.Scan(&item.ProjectRef, &item.WorkItemRef, &item.ProviderSurface, &item.ItemID, &item.ItemType, &item.Title,
			&item.Description, &item.Status, &item.Priority, &item.Revision, &item.Digest, &parents, &dependencies, &blockers,
			&acceptance, &specs, &item.ExternalRef, &item.SourceAuthority, &item.BindingState, &observed); err != nil {
			return nil, fmt.Errorf("%w: scan provider work item: %v", ErrIndexCorrupt, err)
		}
		encoded := []string{parents, dependencies, blockers, acceptance, specs}
		decoded := []*[]string{&item.ParentRefs, &item.DependencyRefs, &item.BlockerRefs, &item.AcceptanceRefs, &item.SpecRefs}
		for i := range encoded {
			if err := json.Unmarshal([]byte(encoded[i]), decoded[i]); err != nil {
				return nil, fmt.Errorf("%w: decode provider work item refs: %v", ErrIndexCorrupt, err)
			}
		}
		item.ObservedAt = time.Unix(0, observed).UTC()
		out = append(out, item)
	}
	return out, rows.Err()
}
