package evidenceregistry

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const MaxProjectRefRunes = 512

type Manager struct {
	root   string
	mu     sync.Mutex
	syncMu sync.Mutex
	stores map[string]*Store
	events *registryEventHub
}

func NewManager(root string) (*Manager, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, ErrConfig
	}
	return &Manager{root: root, stores: make(map[string]*Store), events: newRegistryEventHub()}, nil
}

func (m *Manager) Project(ctx context.Context, projectRef string) (*Store, error) {
	return m.open(ctx, projectRef, false)
}

func (m *Manager) EnsureProject(ctx context.Context, projectRef string) (*Store, error) {
	return m.open(ctx, projectRef, true)
}

func (m *Manager) open(ctx context.Context, projectRef string, create bool) (*Store, error) {
	projectRef = strings.TrimSpace(projectRef)
	if m == nil || projectRef == "" || utf8.RuneCountInString(projectRef) > MaxProjectRefRunes {
		return nil, ErrConfig
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if store := m.stores[projectRef]; store != nil {
		return store, nil
	}
	digest := sha256.Sum256([]byte(projectRef))
	path := filepath.Join(m.root, hex.EncodeToString(digest[:]), "evidence-index.sqlite3")
	if !create {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return nil, ErrNotFound
		} else if err != nil {
			return nil, fmt.Errorf("resolve project registry: %w", err)
		}
	}
	store, err := Open(ctx, Config{Path: path, ProjectRef: projectRef})
	if err != nil {
		return nil, fmt.Errorf("open project registry: %w", err)
	}
	m.stores[projectRef] = store
	return store, nil
}

func (m *Manager) ProjectProjections(ctx context.Context) ([]ProjectProjection, error) {
	if m == nil {
		return nil, ErrConfig
	}
	entries, err := os.ReadDir(m.root)
	if os.IsNotExist(err) {
		return []ProjectProjection{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list project registries: %w", err)
	}
	projects := make([]ProjectProjection, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() || len(entry.Name()) != 64 {
			continue
		}
		path := filepath.Join(m.root, entry.Name(), "evidence-index.sqlite3")
		db, err := sql.Open("sqlite", "file:"+path+"?mode=ro")
		if err != nil {
			continue
		}
		var project ProjectProjection
		var observed int64
		err = db.QueryRowContext(ctx, `SELECT project_ref, project_id, display_name, fingerprint, workspace_kind, scope_safety, source_schema, source_revision, observed_at FROM project_projection LIMIT 1`).Scan(
			&project.ProjectRef, &project.ProjectID, &project.DisplayName, &project.Fingerprint, &project.WorkspaceKind, &project.ScopeSafety, &project.SourceSchema, &project.SourceRevision, &observed)
		_ = db.Close()
		if err == sql.ErrNoRows {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read project registry %q: %w", entry.Name(), err)
		}
		project.ObservedAt = time.Unix(0, observed).UTC()
		projects = append(projects, project)
	}
	sort.Slice(projects, func(i, j int) bool {
		if projects[i].DisplayName == projects[j].DisplayName {
			return projects[i].ProjectRef < projects[j].ProjectRef
		}
		return projects[i].DisplayName < projects[j].DisplayName
	})
	return projects, nil
}

func (m *Manager) Close() error {
	if m == nil {
		return nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	var first error
	for projectRef, store := range m.stores {
		if err := store.Close(); err != nil && first == nil {
			first = fmt.Errorf("close project registry %q: %w", projectRef, err)
		}
		delete(m.stores, projectRef)
	}
	return first
}
