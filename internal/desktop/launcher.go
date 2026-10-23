package desktop

import (
	"context"
	"errors"
)

var ErrDesktopUnavailable = errors.New("desktop unavailable")

type Launcher interface {
	Open(context.Context, string) error
}

func NewPlatformLauncher() Launcher { return platformLauncher{} }
