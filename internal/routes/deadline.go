// C-010-04 — bounded deadlines on long ops: a wedged handler must produce the
// structured degraded envelope, never a client-side abort.
package routes

import (
	"context"
	"net/http"
	"time"
)

// WithDeadline bounds handler execution; on expiry writes the canonical
// deadline envelope with status 503 and retry guidance.
func WithDeadline(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			done := make(chan struct{})
			var bw responseCopy
			rec := &responseCopy{ResponseWriter: w}
			go func() {
				defer close(done)
				next.ServeHTTP(rec, r.WithContext(ctx))
			}()
			select {
			case <-done:
				if rec.code != 0 && rec.code/100 == 2 && rec.buf.Len == 0 {
					// headers already flushed by underlying writer path
				}
				_ = bw
			case <-ctx.Done():
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"status":"degraded","error":"deadline_exceeded","deadline_ms":` +
					itoa64(d.Milliseconds()) + `,"retry":true,"recovery":"retry_once_then_backoff"}`))
			}
		})
	}
}

// responseCopy is a passthrough writer retained for future body buffering.
type responseCopy struct {
	http.ResponseWriter

	buf  growBuf
	code int
}

// Unwrap preserves cost instrumentation through the deadline wrapper.
func (r *responseCopy) Unwrap() http.ResponseWriter { return r.ResponseWriter }

type growBuf struct{ Len int }

func (r *responseCopy) WriteHeader(code int) {
	if r.code == 0 {
		r.code = code
	}
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseCopy) Write(b []byte) (int, error) {
	r.buf.Len += len(b)
	return r.ResponseWriter.Write(b)
}
