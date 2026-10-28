package vision

import (
	"context"
	"fmt"
)

// CurrentOriginContext reads browser-managed target metadata rather than the
// cached URL, which may lag redirects and page-initiated navigation. Reuse the
// authentication-storage origin normalizer; no path or credential value escapes.
func (s *Session) CurrentOriginContext(caller context.Context) (string, error) {
	if err := caller.Err(); err != nil {
		return "", err
	}
	s.mu.Lock()
	page := s.page
	s.mu.Unlock()
	if page == nil {
		return "", fmt.Errorf("session page unavailable")
	}
	ctx, cancel := sessionActionContext(page.GetContext(), caller, sessionInputTimeout)
	defer cancel()
	info, err := page.Context(ctx).Info()
	if err != nil {
		return "", fmt.Errorf("session origin could not be verified")
	}
	origin := storageOrigin(info.URL)
	if origin == "" {
		return "", fmt.Errorf("session has no HTTP(S) origin")
	}
	return origin, nil
}
