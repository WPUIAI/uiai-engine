//go:build darwin

package desktop

import (
	"context"
	"os/exec"
)

type platformLauncher struct{}

func (platformLauncher) Open(ctx context.Context, activationURL string) error {
	return exec.CommandContext(ctx, "open", activationURL).Run()
}
