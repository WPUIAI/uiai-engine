// C-010-02 SSE surface: streams focusa.stream_event.v1 envelopes to
// Focusa consumers with cursor replay and heartbeats.
package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	fevents "github.com/WPUIAI/uiai-engine/internal/events"
	"github.com/go-chi/chi/v5"
)

func MountFocusaEvents(r chi.Router) {
	r.Get("/api/focusa-events", func(w http.ResponseWriter, req *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		since := int64(0)
		if v := req.URL.Query().Get("since"); v != "" {
			if n, err := strconv.ParseInt(v, 10, 64); err == nil {
				since = n
			}
		}
		writeEvent := func(env fevents.Envelope) bool {
			data, err := json.Marshal(env)
			if err != nil {
				return true
			}
			_, werr := fmt.Fprintf(w, "event: focusa_event\nid: %s\ndata: %s\n\n", env.Cursor, data)
			if werr != nil {
				return false
			}
			flusher.Flush()
			return true
		}

		for _, env := range fevents.Since(since) {
			if !writeEvent(env) {
				return
			}
		}

		ch, cancel := fevents.Subscribe()
		defer cancel()
		heartbeat := time.NewTicker(15 * time.Second)
		defer heartbeat.Stop()
		for {
			select {
			case <-req.Context().Done():
				return
			case <-heartbeat.C:
				if _, err := fmt.Fprint(w, ": hb\n\n"); err != nil {
					return
				}
				flusher.Flush()
			case env := <-ch:
				if !writeEvent(env) {
					return
				}
			}
		}
	})
}
