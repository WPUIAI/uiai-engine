package vision

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"testing"
	"time"
)

func TestPacingInstantPreservesDefaultShape(t *testing.T) {
	session := &Session{}
	if err := session.SetPace(PaceConfig{Mode: "instant"}); err != nil {
		t.Fatal(err)
	}
	if session.pacer != nil || session.Pacing() != nil {
		t.Fatal("instant created pacing state")
	}
	before, _ := json.Marshal(DiagnosticsSnapshot{})
	after, _ := json.Marshal(DiagnosticsSnapshot{Pacing: session.Pacing()})
	if string(before) != string(after) {
		t.Fatal("instant changed diagnostics bytes")
	}
	if err := session.waitPaceLocked(context.Background(), "click"); err != nil {
		t.Fatal(err)
	}
}

func TestPacingSeededNonuniformBoundedDelays(t *testing.T) {
	seed := int64(47)
	config := PaceConfig{Mode: "human", Seed: &seed}
	first, err := newSessionPacer(config)
	if err != nil {
		t.Fatal(err)
	}
	second, _ := newSessionPacer(config)
	distinct := map[time.Duration]bool{}
	for i := 0; i < 200; i++ {
		delay := first.actionDelay("click")
		if delay != second.actionDelay("click") {
			t.Fatal("seed not deterministic")
		}
		if delay < 50*time.Millisecond || delay > 2*time.Second {
			t.Fatal("click dwell out of bounds")
		}
		distinct[delay] = true
		key := first.keyDelay()
		if key != second.keyDelay() {
			t.Fatal("key seed not deterministic")
		}
		if key < time.Second/16 || key > time.Second/8 {
			t.Fatal("key cadence out of range")
		}
	}
	if len(distinct) < 190 {
		t.Fatal("jitter collapsed toward constant intervals")
	}
}

func TestPacingInvalidConfigPreservesProfile(t *testing.T) {
	session := &Session{}
	if err := session.SetPace(PaceConfig{Mode: "human"}); err != nil {
		t.Fatal(err)
	}
	original := session.pacer
	for _, config := range []PaceConfig{{Mode: "unknown"}, {Mode: "instant", MedianMS: 1}, {Mode: "custom", MedianMS: -1}, {Mode: "custom", MedianMS: 2001}, {Mode: "custom", CPSMin: math.NaN()}, {Mode: "custom", CPSMax: math.Inf(1)}, {Mode: "custom", CPSMin: .1}, {Mode: "custom", CPSMin: 20, CPSMax: 10}} {
		if err := session.SetPace(config); err == nil {
			t.Fatal("invalid profile accepted")
		}
		if session.pacer != original {
			t.Fatal("invalid config replaced profile")
		}
	}
}

func TestPacingCancellationAndAuthPause(t *testing.T) {
	session := &Session{}
	if err := session.SetPace(PaceConfig{Mode: "human"}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()
	if err := session.waitPaceLocked(ctx, "click"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatal("wait lost deadline")
	}
	if session.Pacing().CanceledWaits != 1 {
		t.Fatal("canceled wait unreported")
	}
	session.raiseAuthChallenge("credential_form", "https://example.test/login")
	if err := session.waitPaceLocked(context.Background(), "click"); err == nil {
		t.Fatal("pacing bypassed auth pause")
	}
	if err := session.SetPace(PaceConfig{Mode: "instant"}); err == nil {
		t.Fatal("profile change bypassed auth pause")
	}
}
