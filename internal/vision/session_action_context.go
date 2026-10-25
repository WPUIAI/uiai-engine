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
