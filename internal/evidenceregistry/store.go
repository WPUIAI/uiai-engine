package evidenceregistry

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const defaultBusyTimeout = 5 * time.Second

type Store struct {
	db         *sql.DB
	projectRef string
	now        func() time.Time
}

func Open(ctx context.Context, cfg Config) (*Store, error) {
	cfg.Path = strings.TrimSpace(cfg.Path)
	cfg.ProjectRef = strings.TrimSpace(cfg.ProjectRef)
	if cfg.Path == "" || cfg.ProjectRef == "" {
		return nil, ErrConfig
	}
	if cfg.BusyTimeout == 0 {
		cfg.BusyTimeout = defaultBusyTimeout
	}
	if cfg.BusyTimeout < 0 || cfg.BusyTimeout > time.Minute {
		return nil, ErrConfig
	}
	if cfg.Now == nil {
		cfg.Now = func() time.Time { return time.Now().UTC() }
	}
	absolute, err := filepath.Abs(cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("%w: database path: %v", ErrConfig, err)
	}
	if err := os.MkdirAll(filepath.Dir(absolute), 0o750); err != nil {
		return nil, fmt.Errorf("%w: create database directory: %v", ErrIndexUnavailable, err)
	}
	dsn := registryDSN(absolute, cfg.BusyTimeout)
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("%w: open database: %v", ErrIndexUnavailable, err)
	}
	db.SetMaxOpenConns(8)
	db.SetMaxIdleConns(4)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("%w: ping database: %v", ErrIndexUnavailable, err)
	}
	now := cfg.Now().UTC()
	if err := migrate(ctx, db, cfg.ProjectRef, now); err != nil {
		db.Close()
		return nil, err
	}
	store := &Store{db: db, projectRef: cfg.ProjectRef, now: cfg.Now}
	if err := store.IntegrityCheck(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return store, nil
}

func registryDSN(path string, busyTimeout time.Duration) string {
	uriPath := filepath.ToSlash(path)
	if filepath.VolumeName(path) != "" {
		uriPath = "/" + uriPath
	}
	uri := url.URL{Scheme: "file", Path: uriPath}
	query := url.Values{}
	query.Set("_defensive", "1")
	query.Set("_journal_mode", "WAL")
	query.Set("_foreign_keys", "1")
	query.Set("_busy_timeout", fmt.Sprintf("%d", busyTimeout.Milliseconds()))
	query.Set("_synchronous", "NORMAL")
	query.Set("_txlock", "immediate")
	query.Set("_dqs", "0")
	uri.RawQuery = query.Encode()
	return uri.String()
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) ProjectRef() string {
	if s == nil {
		return ""
	}
	return s.projectRef
}

func (s *Store) IntegrityCheck(ctx context.Context) error {
	if s == nil || s.db == nil {
		return ErrIndexUnavailable
	}
	var result string
	if err := s.db.QueryRowContext(ctx, `PRAGMA quick_check`).Scan(&result); err != nil {
		return fmt.Errorf("%w: quick check: %v", ErrIndexUnavailable, err)
	}
	if result != "ok" {
		_, updateErr := s.db.ExecContext(ctx, `UPDATE registry_meta SET state = ?, stale_reason = ?, updated_at = ? WHERE id = 1`, IndexCorrupt, result, s.now().UTC().UnixNano())
		if updateErr != nil {
			return fmt.Errorf("%w: %s; status update: %v", ErrIndexCorrupt, result, updateErr)
		}
		return fmt.Errorf("%w: %s", ErrIndexCorrupt, result)
	}
	_, err := s.db.ExecContext(ctx, `UPDATE registry_meta SET last_integrity_at = ?, updated_at = ? WHERE id = 1`, s.now().UTC().UnixNano(), s.now().UTC().UnixNano())
	if err != nil {
		return fmt.Errorf("%w: integrity receipt: %v", ErrIndexUnavailable, err)
	}
	return nil
}

func (s *Store) Status(ctx context.Context) (IndexStatus, error) {
	if s == nil || s.db == nil {
		return IndexStatus{}, ErrIndexUnavailable
	}
	var status IndexStatus
	status.Schema = RegistrySchemaV1
	var state string
	var integrityAt int64
	if err := s.db.QueryRowContext(ctx, `SELECT project_ref, state, revision, source_cursor, rebuild_cursor, stale_reason, last_integrity_at FROM registry_meta WHERE id = 1`).Scan(
		&status.ProjectRef, &state, &status.Revision, &status.SourceCursor, &status.RebuildCursor, &status.StaleReason, &integrityAt,
	); err != nil {
		return IndexStatus{}, fmt.Errorf("%w: read status: %v", ErrIndexCorrupt, err)
	}
	status.State = IndexState(state)
	if integrityAt > 0 {
		status.LastIntegrityAt = time.Unix(0, integrityAt).UTC()
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM artifact WHERE project_ref = ?`, s.projectRef).Scan(&status.ArtifactCount); err != nil {
		return IndexStatus{}, fmt.Errorf("%w: count artifacts: %v", ErrIndexUnavailable, err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM artifact_edge WHERE project_ref = ?`, s.projectRef).Scan(&status.EdgeCount); err != nil {
		return IndexStatus{}, fmt.Errorf("%w: count edges: %v", ErrIndexUnavailable, err)
	}
	return status, nil
}
