package ai

import (
	"errors"
	"time"
)

// This is a local cache-age ceiling, not an upstream broker lease or proof of
// end-to-end revocation. Shorter configured TTLs still win; zero disables reuse.
const maxCredentialCacheAge = 60 * time.Second

var errCredentialCacheInvalidated = errors.New("credential cache invalidated during refresh")

func (p *Provider) credentialTime() time.Time {
	if p.now != nil {
		return p.now()
	}
	return time.Now()
}

// Caller holds keysMu. A backward clock jump must not extend credential reuse.
func (p *Provider) credentialCacheFreshLocked() bool {
	ttl := min(p.keysTTL, maxCredentialCacheAge)
	age := p.credentialTime().Sub(p.keys.fetchedAt)
	return ttl > 0 && !p.keys.fetchedAt.IsZero() && age >= 0 && age < ttl
}

// InvalidateCredentialCache invalidates the entire local settings snapshot and
// fences refreshes that began before this call. It does not revoke credentials
// upstream, erase immutable Go strings, or cancel already-started requests.
// Broker event/credential-reference binding remains a separate integration.
func (p *Provider) InvalidateCredentialCache() {
	p.keysMu.Lock()
	defer p.keysMu.Unlock()
	p.keysEpoch++
	p.keys = aiKeys{}
	p.modelsMu.Lock()
	p.models = nil
	p.modelsMu.Unlock()
}
