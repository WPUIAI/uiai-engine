// C-010-04: bounded deadlines with exclusive ownership of the client response.
package routes

import (
	"bytes"
	"context"
	"net/http"
	"sync"
	"time"
)

// WithDeadline buffers finite API responses until the handler completes. Only
// this middleware writes to the client; a timed-out handler cannot change it.
// Streaming endpoints must stay outside this middleware.
func WithDeadline(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()
			buffered := &deadlineResponseWriter{header: w.Header().Clone()}
			var handlerWriter http.ResponseWriter = buffered
			// Preserve cost injection without exposing the live writer through Unwrap.
			// A handler that ignores cancellation must not race cost trailers either.
			originalCost, _ := r.Context().Value(costKeyType{}).(*costHolder)
			var handlerCost *costHolder
			if originalCost != nil {
				handlerCost = &costHolder{start: originalCost.start, pages: originalCost.pages}
				ctx = context.WithValue(ctx, costKeyType{}, handlerCost)
				handlerWriter = &costWriter{ResponseWriter: buffered, holder: handlerCost}
			}
			done := make(chan any, 1)
			go func() {
				defer func() { done <- recover() }()
				next.ServeHTTP(handlerWriter, r.WithContext(ctx))
			}()
			select {
			case panicValue := <-done:
				if panicValue != nil {
					panic(panicValue)
				}
				if originalCost != nil {
					originalCost.pages = handlerCost.pages
				}
				for key := range w.Header() {
					delete(w.Header(), key)
				}
				for key, values := range buffered.header {
					w.Header()[key] = append([]string(nil), values...)
				}
				code := buffered.code
				if code == 0 {
					code = http.StatusOK
				}
				if buffered.body.Len() > 0 && w.Header().Get("Content-Type") == "" {
					w.Header().Set("Content-Type", http.DetectContentType(buffered.body.Bytes()))
				}
				w.WriteHeader(code)
				_, _ = w.Write(buffered.body.Bytes())
			case <-ctx.Done():
				buffered.mu.Lock()
				buffered.timedOut = true
				buffered.mu.Unlock()
				w.Header().Del("Content-Length")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusServiceUnavailable)
				_, _ = w.Write([]byte(`{"status":"degraded","error":"deadline_exceeded","deadline_ms":` + itoa64(d.Milliseconds()) + `,"retry":true,"recovery":"retry_once_then_backoff"}`))
			}
		})
	}
}

// Header belongs exclusively to the handler. On timeout no caller reads it.
// No Unwrap method exposes the real client writer to the downstream handler.
type deadlineResponseWriter struct {
	mu       sync.Mutex
	header   http.Header
	body     bytes.Buffer
	code     int
	timedOut bool
}

func (w *deadlineResponseWriter) Header() http.Header { return w.header }
func (w *deadlineResponseWriter) WriteHeader(code int) {
	if code < 100 || code > 999 {
		panic("invalid WriteHeader code")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if !w.timedOut && w.code == 0 {
		w.code = code
	}
}
func (w *deadlineResponseWriter) Write(data []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.timedOut {
		return 0, http.ErrHandlerTimeout
	}
	if w.code == 0 {
		w.code = http.StatusOK
	}
	return w.body.Write(data)
}
