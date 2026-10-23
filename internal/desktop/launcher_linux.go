//go:build linux

package desktop

import (
	"context"
	"os"
	"os/exec"
)

type platformLauncher struct{}

func (platformLauncher) Open(ctx context.Context, activationURL string) error {
	if os.Getenv("DISPLAY") == "" && os.Getenv("WAYLAND_DISPLAY") == "" {
		return ErrDesktopUnavailable
	}
	if _, err := exec.LookPath("xdg-open"); err != nil {
		return ErrDesktopUnavailable
	}
	return exec.CommandContext(ctx, "xdg-open", activationURL).Run()
}
