package events

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// C-010-02: envelopes must be schema-valid focusa.stream_event.v1.
func TestEnvelopeSchemaFields(t *testing.T) {
	env := Emit("budget.paused", map[string]any{"organization_id": "org:test"}, []string{"workforce"}, map[string]any{"budget_id": "bud_x"})
	if env.Schema != SchemaVersion || env.Cursor == "" || env.Sequence <= 0 {
		t.Fatalf("bad envelope: %+v", env)
	}
	if env.Timestamp == "" || len(env.Invalidate) != 1 || env.Payload["budget_id"] != "bud_x" {
		t.Fatalf("fields missing: %+v", env)
	}
	raw, _ := json.Marshal(env)
	for _, want := range []string{`"schema":"focusa.stream_event.v1"`, `"event_type":"budget.paused"`, `"cursor":`} {
		if !strings.Contains(string(raw), want) {
			t.Fatalf("json missing %s: %s", want, raw)
		}
	}
}

func TestSequencesMonotonic(t *testing.T) {
	a := Emit("t.a", nil, nil, nil)
	b := Emit("t.b", nil, nil, nil)
	if b.Sequence <= a.Sequence || b.Cursor != itoa(b.Sequence) {
		t.Fatalf("sequence/cursor broken: %d %d %s", a.Sequence, b.Sequence, b.Cursor)
	}
}

func TestSinceReplayAndSubscribe(t *testing.T) {
	before := LatestSeq()
	Emit("replay.mark", nil, nil, nil)
	got := Since(before)
	if len(got) == 0 || got[len(got)-1].EventType != "replay.mark" {
		t.Fatalf("replay missing mark: %+v", got)
	}
	ch, cancel := Subscribe()
	defer cancel()
	go func() { Emit("sub.mark", nil, nil, nil) }()
	select {
	case e := <-ch:
		if e.EventType != "sub.mark" {
			t.Fatalf("wrong event %s", e.EventType)
		}
	case <-timeAfter(500):
		t.Fatal("subscribe fan-out timed out")
	}
}

func timeAfter(ms int) <-chan time.Time {
	return time.After(500 * time.Millisecond)
}
