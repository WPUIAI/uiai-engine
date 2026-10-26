package vision

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SequenceOperation deliberately excludes navigation, authentication, arbitrary
// JS and control/permission changes. Values never enter progress telemetry.
type SequenceOperation struct {
	Action   string `json:"action"`
	Selector string `json:"selector"`
	Text     string `json:"text,omitempty"`
}

type SequenceResult struct {
	Completed       int   `json:"completed"`
	Attempted       int   `json:"attempted"`
	EffectsPossible bool  `json:"effects_possible"`
	DurationMS      int64 `json:"duration_ms"`
}

// Sequence is bounded and cancelable, not an atomic transaction or rollback.
// Concurrent callers are not granted an exclusive actuator lease by batching.
func (s *Session) Sequence(caller context.Context, operations []SequenceOperation, budget time.Duration) (*SnapResult, SequenceResult, error) {
	result := SequenceResult{}
	started := time.Now()
	finish := func(snapshot *SnapResult, err error) (*SnapResult, SequenceResult, error) {
		result.DurationMS = time.Since(started).Milliseconds()
		return snapshot, result, err
	}
	if len(operations) < 1 || len(operations) > 32 || budget < time.Millisecond || budget > 10*time.Second {
		return finish(nil, fmt.Errorf("sequence requires 1..32 operations and 1..10000ms budget"))
	}
	for _, op := range operations {
		if op.Action != "type" && op.Action != "fill" && op.Action != "click" {
			return finish(nil, fmt.Errorf("unsupported sequence action"))
		}
		if strings.TrimSpace(op.Selector) == "" || strings.HasPrefix(op.Selector, "@") || strings.HasPrefix(op.Selector, "text=") || strings.HasPrefix(op.Selector, "text/") || strings.HasPrefix(op.Selector, "role=") {
			return finish(nil, fmt.Errorf("sequence v1 requires explicit CSS selectors"))
		}
	}
	ctx, cancel := context.WithTimeout(caller, budget)
	defer cancel()
	var snapshot *SnapResult
	for _, op := range operations {
		if err := ctx.Err(); err != nil {
			return finish(nil, err)
		}
		if s.LeaseState() == LeaseAwaitingAuth {
			return finish(nil, fmt.Errorf("auth challenge pending"))
		}
		result.Attempted++
		result.EffectsPossible = true
		var err error
		if op.Action == "click" {
			snapshot, err = s.ClickContext(ctx, op.Selector)
		} else {
			snapshot, err = s.TypeContext(ctx, op.Selector, op.Text)
		}
		if err != nil {
			return finish(nil, err)
		}
		result.Completed++
		if err := ctx.Err(); err != nil {
			return finish(nil, err)
		}
		if _, err := s.DetectAuthChallengeContext(ctx); err != nil {
			return finish(nil, err)
		}
		if s.LeaseState() == LeaseAwaitingAuth {
			return finish(nil, fmt.Errorf("auth challenge pending"))
		}
	}
	return finish(snapshot, nil)
}
