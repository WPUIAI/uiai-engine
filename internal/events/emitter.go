// Package events — C-010-02: native focusa.stream_event.v1 citizenship.
// The engine emits schema-valid envelopes so Focusa consumers (notification
// center, audit ledger, widgets) can ingest engine telemetry unmodified.
package events

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

const SchemaVersion = "focusa.stream_event.v1"

type Envelope struct {
	Schema        string         `json:"schema"`
	EventID       string         `json:"event_id"`
	Cursor        string         `json:"cursor"`
	Timestamp     string         `json:"timestamp"`
	EventType     string         `json:"event_type"`
	SchemaVersion string         `json:"schema_version"`
	Sequence      int64          `json:"sequence"`
	Scope         map[string]any `json:"scope"`
	Invalidate    []string       `json:"invalidate"`
	Payload       map[string]any `json:"payload"`
}

var (
	mu     sync.Mutex
	seq    int64
	ring   = make([]Envelope, 0, 256)
	subs   = map[int]chan Envelope{}
	subSeq int
)

// Emit records an event and fans it out to subscribers. Never blocks.
func Emit(eventType string, scope map[string]any, invalidate []string, payload map[string]any) Envelope {
	mu.Lock()
	defer mu.Unlock()
	seq++
	env := Envelope{
		Schema:        SchemaVersion,
		EventID:       "evt_" + uuid.NewString()[:13],
		Cursor:        itoa(seq),
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		EventType:     eventType,
		SchemaVersion: "1",
		Sequence:      seq,
		Scope:         scope,
		Invalidate:    invalidate,
		Payload:       payload,
	}
	ring = append(ring, env)
	if len(ring) > 256 {
		ring = ring[len(ring)-256:]
	}
	for _, ch := range subs {
		select {
		case ch <- env:
		default: // slow subscriber drops; cursor replay covers it
		}
	}
	return env
}

// Since returns events after the given sequence (cursor replay).
func Since(after int64) []Envelope {
	mu.Lock()
	defer mu.Unlock()
	out := []Envelope{}
	for _, e := range ring {
		if e.Sequence > after {
			out = append(out, e)
		}
	}
	return out
}

// Subscribe registers a bounded channel for live fan-out; returns cancel fn.
func Subscribe() (<-chan Envelope, func()) {
	mu.Lock()
	defer mu.Unlock()
	subSeq++
	id := subSeq
	ch := make(chan Envelope, 64)
	subs[id] = ch
	return ch, func() {
		mu.Lock()
		delete(subs, id)
		mu.Unlock()
	}
}

// LatestSeq exposes the current cursor for replay bootstrap.
func LatestSeq() int64 {
	mu.Lock()
	defer mu.Unlock()
	return seq
}

func itoa(v int64) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
