package vision

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestEvaluationCallerCancellationDoesNotReplayOrPoisonSession(t *testing.T) {
	_, manager, url := storageIsolationFixture(t, 1)
	t.Cleanup(manager.CloseAll)
	for _, method := range []struct {
		name string
		run  func(*Session, context.Context, string) (string, *SnapResult, error)
	}{
		{"eval", (*Session).EvalContext},
		{"eval_async", func(s *Session, c context.Context, j string) (string, *SnapResult, error) {
			return s.EvalAsyncContext(c, j, 5000)
		}},
	} {
		t.Run(method.name, func(t *testing.T) {
			session, _, err := manager.Open(url, 640, 480)
			if err != nil {
				t.Fatal(err)
			}
			defer manager.Close(session.ID)
			caller, cancel := context.WithCancel(context.Background())
			cancel()
			_, snapshot, err := method.run(session, caller, `window.effects=1;return true;`)
			var actionError *SessionActionError
			if !errors.Is(err, context.Canceled) || !errors.As(err, &actionError) || actionError.EffectsPossible || snapshot != nil {
				t.Fatal("pre-dispatch cancellation lost its no-effect posture")
			}
			caller, cancel = context.WithCancel(context.Background())
			defer cancel()
			done := make(chan error, 1)
			go func() {
				_, _, err := method.run(session, caller, `window.effects=(window.effects||0)+1;return new Promise(()=>{});`)
				done <- err
			}()
			deadline := time.Now().Add(2 * time.Second)
			dispatched := false
			for time.Now().Before(deadline) {
				value, err := session.page.Timeout(time.Second).Eval(`() => window.effects||0`)
				if err != nil {
					t.Fatal(err)
				}
				if value.Value.Num() == 1 {
					dispatched = true
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			if !dispatched {
				t.Fatal("fixture evaluation never dispatched")
			}
			cancel()
			select {
			case err := <-done:
				if !errors.Is(err, context.Canceled) || !errors.As(err, &actionError) || !actionError.EffectsPossible {
					t.Fatalf("post-dispatch cancellation lost effect uncertainty: %v", err)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("caller cancellation did not release the action")
			}
			if session.page.GetContext().Err() != nil {
				t.Fatal("caller cancellation poisoned shared page context")
			}
			result, snapshot, err := session.Eval("return window.effects")
			if err != nil || result != "1" || snapshot == nil {
				t.Fatal("cancelled evaluation was replayed or session was unusable")
			}
		})
	}
}

func TestEvaluationNavigationInvalidationIsTypedAndRecoverable(t *testing.T) {
	_, manager, url := storageIsolationFixture(t, 1)
	t.Cleanup(manager.CloseAll)
	for _, method := range []struct {
		name string
		run  func(*Session, string) (string, *SnapResult, error)
	}{
		{"eval", (*Session).Eval},
		{"eval_async", func(s *Session, j string) (string, *SnapResult, error) { return s.EvalAsync(j, 3000) }},
	} {
		t.Run(method.name, func(t *testing.T) {
			session, _, err := manager.Open(url, 640, 480)
			if err != nil {
				t.Fatal(err)
			}
			defer manager.Close(session.ID)
			_, snapshot, err := method.run(session, `setTimeout(()=>location.assign('/replacement'),50);return new Promise(()=>{});`)
			var actionError *SessionActionError
			if err == nil || snapshot != nil || !errors.As(err, &actionError) || !actionError.EffectsPossible {
				t.Fatal("document replacement was presented as successful evaluation")
			}
			if actionError.CauseClass() != "execution_context_changed" {
				t.Fatalf("navigation clue lost: %s", actionError.CauseClass())
			}
			if _, err := session.Snapshot(SnapshotOptions{Compact: true}); err != nil {
				t.Fatal(err)
			}
			if _, err := session.Screenshot("jpeg", 60); err != nil {
				t.Fatal(err)
			}
			result, snapshot, err := session.Eval("return location.pathname")
			if err != nil || snapshot == nil || result != "/replacement" {
				t.Fatal("navigation left evaluation unusable")
			}
		})
	}
}
