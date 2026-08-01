package browserprofile

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/base64"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/go-rod/rod"
)

// OpenRequest selects and launches a profile-scoped browser runtime.
type OpenRequest struct {
	URL              string `json:"url"`
	BrowserProfile   string `json:"browser_profile,omitempty"`
	BrowserMode      Mode   `json:"browser_mode,omitempty"`
	ProfileSelection string `json:"profile_selection,omitempty"`
}

// ManagedSession is an independently launched profile runtime and page.
type ManagedSession struct {
	ID        string          `json:"id"`
	URL       string          `json:"url"`
	Title     string          `json:"title"`
	CreatedAt time.Time       `json:"created_at"`
	LastUsed  time.Time       `json:"last_used"`
	Selection Selection       `json:"selection"`
	Profile   ResolvedProfile `json:"profile"`
	RuntimePID int            `json:"runtime_pid,omitempty"`

	runtime *Runtime
	page    *rod.Page
	lockKey string
}

// SessionSummary is the public manager view without runtime pointers.
type SessionSummary struct {
	ID         string    `json:"id"`
	URL        string    `json:"url"`
	Title      string    `json:"title"`
	CreatedAt  time.Time `json:"created_at"`
	LastUsed   time.Time `json:"last_used"`
	ProfileID  string    `json:"profile_id"`
	Mode       Mode      `json:"mode"`
	Engine     Engine    `json:"engine"`
	Digest     string    `json:"profile_digest"`
	RuntimePID int       `json:"runtime_pid,omitempty"`
	NetworkRoute string  `json:"network_route"`
	ChallengePolicy ChallengePolicy `json:"challenge_policy"`
}

// Manager owns profile-scoped runtimes and persistent-profile locks.
type Manager struct {
	mu       sync.RWMutex
	registry *Registry
	sessions map[string]*ManagedSession
	locks    map[string]string
}

func NewManager(registry *Registry) (*Manager, error) {
	if registry == nil {
		return nil, fmt.Errorf("browser profile registry is required")
	}
	return &Manager{
		registry: registry,
		sessions: make(map[string]*ManagedSession),
		locks: make(map[string]string),
	}, nil
}

func (m *Manager) Registry() *Registry { return m.registry }

func (m *Manager) Open(ctx context.Context, request OpenRequest) (*ManagedSession, error) {
	if request.URL == "" {
		return nil, fmt.Errorf("url is required")
	}
	profile, selection, err := m.registry.Select(request.URL, request.BrowserProfile, request.BrowserMode)
	if err != nil {
		return nil, err
	}

	lockKey := ""
	if profile.Storage.ExclusiveLock {
		lockKey = profile.Launch.UserDataDir
		if lockKey == "" {
			lockKey = profile.Storage.IsolationKey
		}
		if lockKey == "" {
			return nil, fmt.Errorf("exclusive profile %q has no lock identity", profile.ID)
		}
		m.mu.Lock()
		if owner, exists := m.locks[lockKey]; exists {
			m.mu.Unlock()
			return nil, fmt.Errorf("profile %q is locked by session %s", profile.ID, owner)
		}
		placeholder := "opening-" + newSessionID()
		m.locks[lockKey] = placeholder
		m.mu.Unlock()
		defer func() {
			if err != nil {
				m.mu.Lock()
				if m.locks[lockKey] == placeholder {
					delete(m.locks, lockKey)
				}
				m.mu.Unlock()
			}
		}()
	}

	runtime, err := Launch(ctx, profile)
	if err != nil {
		return nil, err
	}
	page, err := runtime.OpenPage(ctx, request.URL)
	if err != nil {
		runtime.Close()
		return nil, err
	}

	now := time.Now().UTC()
	session := &ManagedSession{
		ID: newSessionID(),
		URL: request.URL,
		CreatedAt: now,
		LastUsed: now,
		Selection: selection,
		Profile: profile,
		RuntimePID: runtime.PID,
		runtime: runtime,
		page: page,
		lockKey: lockKey,
	}
	if result, titleErr := page.Eval(`() => document.title`); titleErr == nil {
		session.Title = result.Value.Str()
	}

	m.mu.Lock()
	m.sessions[session.ID] = session
	if lockKey != "" {
		m.locks[lockKey] = session.ID
	}
	m.mu.Unlock()
	return session, nil
}

func (m *Manager) Get(id string) (*ManagedSession, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	return session, ok
}

func (m *Manager) List() []SessionSummary {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]SessionSummary, 0, len(m.sessions))
	for _, session := range m.sessions {
		out = append(out, sessionSummary(session))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out
}

func (m *Manager) Navigate(ctx context.Context, id, targetURL string) (*SessionSummary, error) {
	m.mu.RLock()
	session, ok := m.sessions[id]
	m.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("profile session %q not found", id)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := session.page.Timeout(30 * time.Second).Navigate(targetURL); err != nil {
		return nil, fmt.Errorf("navigate profile session %q: %w", id, err)
	}
	title := ""
	if result, err := session.page.Eval(`() => document.title`); err == nil {
		title = result.Value.Str()
	}
	m.mu.Lock()
	session.URL = targetURL
	session.Title = title
	session.LastUsed = time.Now().UTC()
	summary := sessionSummary(session)
	m.mu.Unlock()
	return &summary, nil
}

func (m *Manager) Close(id string) error {
	m.mu.Lock()
	session, ok := m.sessions[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("profile session %q not found", id)
	}
	delete(m.sessions, id)
	if session.lockKey != "" && m.locks[session.lockKey] == id {
		delete(m.locks, session.lockKey)
	}
	m.mu.Unlock()

	if session.page != nil {
		_ = session.page.Close()
	}
	if session.runtime != nil {
		session.runtime.Close()
	}
	return nil
}

func (m *Manager) CloseAll() {
	m.mu.RLock()
	ids := make([]string, 0, len(m.sessions))
	for id := range m.sessions {
		ids = append(ids, id)
	}
	m.mu.RUnlock()
	for _, id := range ids {
		_ = m.Close(id)
	}
}

func sessionSummary(session *ManagedSession) SessionSummary {
	return SessionSummary{
		ID: session.ID,
		URL: session.URL,
		Title: session.Title,
		CreatedAt: session.CreatedAt,
		LastUsed: session.LastUsed,
		ProfileID: session.Profile.ID,
		Mode: session.Profile.Mode,
		Engine: session.Profile.Engine,
		Digest: session.Profile.Digest,
		RuntimePID: session.RuntimePID,
		NetworkRoute: session.Profile.Network.Route,
		ChallengePolicy: session.Profile.Challenge.Policy,
	}
}

func newSessionID() string {
	var raw [9]byte
	if _, err := cryptorand.Read(raw[:]); err != nil {
		return fmt.Sprintf("bps-%x", time.Now().UnixNano())
	}
	return "bps-" + base64.RawURLEncoding.EncodeToString(raw[:])
}
