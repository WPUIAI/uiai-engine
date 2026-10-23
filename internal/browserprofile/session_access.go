package browserprofile

import "github.com/go-rod/rod"

// SessionPage returns the live page and resolved profile for internal core
// adapters such as the existing CAPTCHA solver. The HTTP API never exposes the
// raw page pointer.
func (m *Manager) SessionPage(id string) (*rod.Page, ResolvedProfile, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	if !ok || session.page == nil {
		return nil, ResolvedProfile{}, false
	}
	return session.page, session.Profile, true
}
