package vision

import (
	"context"
	"time"
)

// sessionActionContext preserves the session lifetime and the caller's explicit
// cancellation without ever cancelling the shared page context.
func sessionActionContext(session, caller context.Context, budget time.Duration) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(session, budget)
	stop := context.AfterFunc(caller, cancel)
	return ctx, func() { stop(); cancel() }
}

// waitSessionDelay is the shared cancellable delay for paced and settle waits.
func waitSessionDelay(ctx context.Context, delay time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return ctx.Err()
	}
}
