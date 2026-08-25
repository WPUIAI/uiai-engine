// C-010-05 cost telemetry — every JSON envelope carries a bounded cost object
// and every response carries cost headers, so agents self-regulate on measured spend.
package routes

import (
	"context"
	"net/http"
	"time"
)

type costKeyType struct{}

type costHolder struct {
	start time.Time
	pages int
}

type costWriter struct {
	http.ResponseWriter
	holder *costHolder
	bytes  int
	wrote  bool
}

func (c *costWriter) Write(b []byte) (int, error) {
	c.wrote = true
	n, err := c.ResponseWriter.Write(b)
	c.bytes += n
	return n, err
}

func (c *costWriter) WriteHeader(code int) {
	if !c.wrote {
		c.wrote = true
	}
	c.ResponseWriter.WriteHeader(code)
}

// CostMiddleware records duration + bytes-out and stamps X-UIAI-Cost-* headers.
func CostMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		holder := &costHolder{start: time.Now()}
		cw := &costWriter{ResponseWriter: w, holder: holder}
		r = r.WithContext(context.WithValue(r.Context(), costKeyType{}, holder))
		next.ServeHTTP(cw, r)
		// C-010-05: emit as HTTP trailers (valid after body) so handlers are
		// never blocked from flushing early.
		h := cw.Header()
		h.Set(http.TrailerPrefix+"X-UIAI-Cost-Ms", itoa64(time.Since(holder.start).Milliseconds()))
		h.Set(http.TrailerPrefix+"X-UIAI-Cost-Bytes", itoa64(int64(cw.bytes)))
		h.Set(http.TrailerPrefix+"X-UIAI-Cost-Pages", itoa64(int64(holder.pages)))
	})
}

// CostTouchPage records page interactions for the current request cost.
func CostTouchPage(r *http.Request, n int) {
	if h, ok := r.Context().Value(costKeyType{}).(*costHolder); ok && h != nil {
		h.pages += n
	}
}

// InjectCost adds the cost object to top-level map envelopes when the writer
// is cost-instrumented (C-010-05). Safe no-op otherwise.
func InjectCost(data any, w http.ResponseWriter) any {
	cw, ok := w.(*costWriter)
	if !ok || cw == nil || cw.holder == nil {
		return data
	}
	m, ok := data.(map[string]any)
	if !ok {
		return data
	}
	out := make(map[string]any, len(m)+1)
	for k, v := range m {
		out[k] = v
	}
	out["cost"] = map[string]any{
		"duration_ms":   time.Since(cw.holder.start).Milliseconds(),
		"pages_touched": cw.holder.pages,
	}
	return out
}

func itoa64(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
