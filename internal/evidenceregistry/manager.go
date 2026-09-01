package evidenceregistry

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"unicode/utf8"
)

const MaxProjectRefRunes = 512

type Manager struct {
	root   string
	mu     sync.Mutex
	stores map[string]*Store
}

func NewManager(root string) (*Manager, error) {
	root = strings.TrimSpace(root)
	if root == "" {
		return nil, ErrConfig
	}
	return &Manager{root: root, stores: make(map[string]*Store)}, nil
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
