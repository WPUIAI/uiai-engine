package routes

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

// SSE-based real-time streaming (no external WebSocket dependency needed).
// Uses Server-Sent Events which work over standard HTTP and are natively
// supported by browsers via EventSource.

type eventBus struct {
	clients sync.Map // clientID → chan []byte
	counter int64
	mu      sync.Mutex
}

var bus = &eventBus{}

// Subscribe returns a channel that receives JSON-encoded events.
func (b *eventBus) Subscribe() (string, chan []byte) {
	b.mu.Lock()
	b.counter++
	id := time.Now().Format("20060102150405") + "-" + string(rune('A'+b.counter%26))
	b.mu.Unlock()

	ch := make(chan []byte, 100)
	b.clients.Store(id, ch)
	return id, ch
}

func (b *eventBus) Unsubscribe(id string) {
	if v, ok := b.clients.LoadAndDelete(id); ok {
		close(v.(chan []byte))
	}
}

// Publish sends an event to all subscribers.
func (b *eventBus) Publish(eventType string, data any) {
	payload, _ := json.Marshal(map[string]any{
		"type":      eventType,
		"data":      data,
		"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
	})

	b.clients.Range(func(key, value any) bool {
		ch := value.(chan []byte)
		select {
		case ch <- payload:
		default:
			// Client too slow, drop message
			log.Printf("[sse] dropping message for slow client %s", key)
		}
		return true
	})
}

// PublishEvent is a package-level helper to emit events from any route handler.
func PublishEvent(eventType string, data any) {
	bus.Publish(eventType, data)
}

// MountSSEStream registers the SSE endpoint for real-time updates.
func MountSSEStream(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			writeJSON(w, 500, map[string]string{"error": "streaming not supported"})
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.Header().Set("Access-Control-Allow-Origin", "*")

		clientID, ch := bus.Subscribe()
		defer bus.Unsubscribe(clientID)

		log.Printf("[sse] client %s connected", clientID)

		// Send initial connected event
		w.Write([]byte("event: connected\ndata: {\"clientId\":\"" + clientID + "\"}\n\n"))
		flusher.Flush()

		// Heartbeat ticker
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case msg, ok := <-ch:
				if !ok {
					return
				}
				w.Write([]byte("event: message\ndata: "))
				w.Write(msg)
				w.Write([]byte("\n\n"))
				flusher.Flush()

			case <-ticker.C:
				w.Write([]byte("event: heartbeat\ndata: {\"time\":\"" + time.Now().UTC().Format(time.RFC3339) + "\"}\n\n"))
				flusher.Flush()

			case <-r.Context().Done():
				log.Printf("[sse] client %s disconnected", clientID)
				return
			}
		}
	}
}
