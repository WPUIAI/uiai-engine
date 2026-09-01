package evidenceregistry

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const schemaVersion = 1

var schemaV1 = []string{
	`CREATE TABLE IF NOT EXISTS registry_meta (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		project_ref TEXT NOT NULL,
		schema_name TEXT NOT NULL,
		state TEXT NOT NULL,
		revision INTEGER NOT NULL DEFAULT 0,
		source_cursor TEXT NOT NULL DEFAULT '',
		rebuild_cursor TEXT NOT NULL DEFAULT '',
		stale_reason TEXT NOT NULL DEFAULT '',
		last_integrity_at INTEGER NOT NULL DEFAULT 0,
		updated_at INTEGER NOT NULL
	) STRICT`,
	`CREATE TABLE IF NOT EXISTS artifact (
		artifact_ref TEXT NOT NULL,
		revision INTEGER NOT NULL CHECK (revision > 0),
		manifest_sha256 TEXT NOT NULL,
		index_input_sha256 TEXT NOT NULL,
		bundle_sha256 TEXT NOT NULL DEFAULT '',
		title TEXT NOT NULL,
		summary TEXT NOT NULL DEFAULT '',
		kinds_json TEXT NOT NULL DEFAULT '[]',
		project_ref TEXT NOT NULL,
		workstream_ref TEXT NOT NULL,
		workset_ref TEXT NOT NULL,
		callgraph_ref TEXT NOT NULL,
		workpoint_ref TEXT NOT NULL,
		verification TEXT NOT NULL,
		access_class TEXT NOT NULL,
		redaction_state TEXT NOT NULL,
		closure_posture TEXT NOT NULL,
		captured_at INTEGER NOT NULL,
		freshness_observed_at INTEGER NOT NULL,
		pwa_path TEXT NOT NULL DEFAULT '',
		indexed_at INTEGER NOT NULL,
		PRIMARY KEY (artifact_ref, revision),
		CHECK (project_ref <> '')
	) STRICT`,
	`CREATE INDEX IF NOT EXISTS artifact_project_captured ON artifact(project_ref, captured_at DESC, artifact_ref DESC, revision DESC)`,
	`CREATE INDEX IF NOT EXISTS artifact_scope ON artifact(project_ref, workstream_ref, workset_ref, workpoint_ref)`,
	`CREATE INDEX IF NOT EXISTS artifact_posture ON artifact(project_ref, closure_posture, verification, access_class)`,
	`CREATE TABLE IF NOT EXISTS work_item_binding (
		artifact_ref TEXT NOT NULL,
		artifact_revision INTEGER NOT NULL,
		project_ref TEXT NOT NULL,
		work_item_ref TEXT NOT NULL,
		provider_surface TEXT NOT NULL DEFAULT '',
		item_id TEXT NOT NULL DEFAULT '',
		item_type TEXT NOT NULL DEFAULT '',
		title TEXT NOT NULL DEFAULT '',
		description TEXT NOT NULL DEFAULT '',
		description_ref TEXT NOT NULL DEFAULT '',
		description_sha256 TEXT NOT NULL DEFAULT '',
		item_revision TEXT NOT NULL DEFAULT '',
		item_digest TEXT NOT NULL DEFAULT '',
		status_at_capture TEXT NOT NULL DEFAULT '',
		parent_refs_json TEXT NOT NULL DEFAULT '[]',
		dependency_refs_json TEXT NOT NULL DEFAULT '[]',
		blocker_refs_json TEXT NOT NULL DEFAULT '[]',
		acceptance_atom_refs_json TEXT NOT NULL DEFAULT '[]',
		evidence_requirement_refs_json TEXT NOT NULL DEFAULT '[]',
		review_requirement_refs_json TEXT NOT NULL DEFAULT '[]',
		closure_posture TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (artifact_ref, artifact_revision, work_item_ref),
		FOREIGN KEY (artifact_ref, artifact_revision) REFERENCES artifact(artifact_ref, revision) ON DELETE CASCADE
	) STRICT`,
	`CREATE INDEX IF NOT EXISTS work_item_reverse ON work_item_binding(project_ref, work_item_ref, artifact_ref, artifact_revision)`,
	`CREATE INDEX IF NOT EXISTS work_item_descriptor ON work_item_binding(project_ref, item_type, title, work_item_ref)`,
	`CREATE TABLE IF NOT EXISTS acceptance_binding (
		artifact_ref TEXT NOT NULL,
		artifact_revision INTEGER NOT NULL,
		project_ref TEXT NOT NULL,
		work_item_ref TEXT NOT NULL DEFAULT '',
		acceptance_atom_ref TEXT NOT NULL,
		atom_revision TEXT NOT NULL DEFAULT '',
		state TEXT NOT NULL,
		verifier_class TEXT NOT NULL DEFAULT '',
		verifier_refs_json TEXT NOT NULL DEFAULT '[]',
		decision_ref TEXT NOT NULL DEFAULT '',
		receipt_ref TEXT NOT NULL DEFAULT '',
		fresh INTEGER NOT NULL CHECK (fresh IN (0,1)),
		scope_matched INTEGER NOT NULL CHECK (scope_matched IN (0,1)),
		PRIMARY KEY (artifact_ref, artifact_revision, acceptance_atom_ref),
		FOREIGN KEY (artifact_ref, artifact_revision) REFERENCES artifact(artifact_ref, revision) ON DELETE CASCADE
	) STRICT`,
	`CREATE INDEX IF NOT EXISTS acceptance_reverse ON acceptance_binding(project_ref, acceptance_atom_ref, state, artifact_ref, artifact_revision)`,
	`CREATE TABLE IF NOT EXISTS artifact_edge (
		project_ref TEXT NOT NULL,
		source_ref TEXT NOT NULL,
		source_revision TEXT NOT NULL DEFAULT '',
		target_ref TEXT NOT NULL,
		target_revision TEXT NOT NULL DEFAULT '',
		relation TEXT NOT NULL,
		provenance_receipt TEXT NOT NULL DEFAULT '',
		observed_at INTEGER NOT NULL,
		PRIMARY KEY (project_ref, source_ref, source_revision, target_ref, target_revision, relation)
	) STRICT`,
	`CREATE INDEX IF NOT EXISTS edge_forward ON artifact_edge(project_ref, source_ref, relation, target_ref)`,
	`CREATE INDEX IF NOT EXISTS edge_reverse ON artifact_edge(project_ref, target_ref, relation, source_ref)`,
	`CREATE TABLE IF NOT EXISTS closure_binding (
		project_ref TEXT NOT NULL,
		work_item_ref TEXT NOT NULL,
		completion_case_ref TEXT NOT NULL DEFAULT '',
		completion_contract_ref TEXT NOT NULL DEFAULT '',
		required_atoms INTEGER NOT NULL DEFAULT 0,
		accepted_atoms INTEGER NOT NULL DEFAULT 0,
		blocked_atoms INTEGER NOT NULL DEFAULT 0,
		stale_atoms INTEGER NOT NULL DEFAULT 0,
		posture TEXT NOT NULL,
		completion_decision_ref TEXT NOT NULL DEFAULT '',
		provider_close_receipt_ref TEXT NOT NULL DEFAULT '',
		reopen_ref TEXT NOT NULL DEFAULT '',
		settlement_posture TEXT NOT NULL DEFAULT '',
		observed_at INTEGER NOT NULL,
		PRIMARY KEY (project_ref, work_item_ref, completion_case_ref)
	) STRICT`,
	`CREATE INDEX IF NOT EXISTS closure_posture_idx ON closure_binding(project_ref, posture, work_item_ref)`,
	`CREATE TABLE IF NOT EXISTS collection (
		project_ref TEXT NOT NULL,
		collection_ref TEXT NOT NULL,
		descriptor TEXT NOT NULL,
		kind TEXT NOT NULL,
		query_json TEXT NOT NULL DEFAULT '{}',
		created_at INTEGER NOT NULL,
		PRIMARY KEY (project_ref, collection_ref)
	) STRICT`,
	`CREATE TABLE IF NOT EXISTS collection_member (
		project_ref TEXT NOT NULL,
		collection_ref TEXT NOT NULL,
		artifact_ref TEXT NOT NULL,
		artifact_revision INTEGER NOT NULL,
		ordinal INTEGER NOT NULL DEFAULT 0,
		PRIMARY KEY (project_ref, collection_ref, artifact_ref, artifact_revision),
		FOREIGN KEY (project_ref, collection_ref) REFERENCES collection(project_ref, collection_ref) ON DELETE CASCADE,
		FOREIGN KEY (artifact_ref, artifact_revision) REFERENCES artifact(artifact_ref, revision) ON DELETE CASCADE
	) STRICT`,
	`CREATE VIRTUAL TABLE IF NOT EXISTS artifact_search USING fts5(
		artifact_ref UNINDEXED,
		revision UNINDEXED,
		title,
		summary,
		kinds,
		work_items,
		acceptance_atoms,
		tokenize = 'unicode61 remove_diacritics 2'
	)`,
}

func migrate(ctx context.Context, db *sql.DB, projectRef string, now time.Time) error {
	var current int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&current); err != nil {
		return fmt.Errorf("%w: read schema version: %v", ErrIndexUnavailable, err)
	}
	if current > schemaVersion {
		return fmt.Errorf("%w: schema version %d newer than %d", ErrIndexCorrupt, current, schemaVersion)
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: begin migration: %v", ErrIndexUnavailable, err)
	}
	defer tx.Rollback()
	if current == 0 {
		for _, statement := range schemaV1 {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("%w: create schema: %v", ErrIndexUnavailable, err)
			}
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO registry_meta(id, project_ref, schema_name, state, updated_at) VALUES(1, ?, ?, ?, ?)`, projectRef, RegistrySchemaV1, IndexReady, now.UnixNano()); err != nil {
			return fmt.Errorf("%w: initialize metadata: %v", ErrIndexUnavailable, err)
		}
		if _, err := tx.ExecContext(ctx, `PRAGMA user_version = 1`); err != nil {
			return fmt.Errorf("%w: stamp schema: %v", ErrIndexUnavailable, err)
		}
	}
	var storedProject, schemaName string
	if err := tx.QueryRowContext(ctx, `SELECT project_ref, schema_name FROM registry_meta WHERE id = 1`).Scan(&storedProject, &schemaName); err != nil {
		return fmt.Errorf("%w: read metadata: %v", ErrIndexCorrupt, err)
	}
	if storedProject != projectRef {
		return fmt.Errorf("%w: database=%q requested=%q", ErrProjectMismatch, storedProject, projectRef)
	}
	if schemaName != RegistrySchemaV1 {
		return fmt.Errorf("%w: schema identity %q", ErrIndexCorrupt, schemaName)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: commit migration: %v", ErrIndexUnavailable, err)
	}
	return nil
}
