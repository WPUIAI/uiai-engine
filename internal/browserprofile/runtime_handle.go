package browserprofile

import (
	"context"

	"github.com/go-rod/rod"
)

// RuntimeHandle is the lifecycle surface required by profile sessions.
type RuntimeHandle interface {
	OpenPage(ctx context.Context, targetURL string) (*rod.Page, error)
	Close()
	RuntimePID() int
}

// LauncherFunc enables the manager to route profiles through the core Chromium
// launcher, the existing local-IP/proxy pool, or future engine adapters.
type LauncherFunc func(ctx context.Context, profile ResolvedProfile) (RuntimeHandle, error)

func (r *Runtime) RuntimePID() int {
	if r == nil {
		return 0
	}
	return r.PID
}
