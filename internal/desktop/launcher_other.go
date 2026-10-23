//go:build !darwin && !linux && !windows

package desktop

import "context"

type platformLauncher struct{}

func (platformLauncher) Open(context.Context, string) error { return ErrDesktopUnavailable }
