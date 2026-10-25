package vision

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestEvaluationFailureCannotReportSuccess(t *testing.T) {
	_, manager, url := storageIsolationFixture(t, 1)
	t.Cleanup(manager.CloseAll)
	for _, method := range []struct {
		name string
		run  func(*Session, string) (string, *SnapResult, error)
	}{
		{"eval", (*Session).Eval},
		{"eval_async", func(s *Session, js string) (string, *SnapResult, error) { return s.EvalAsync(js, 1000) }},
	} {
		t.Run(method.name, func(t *testing.T) {
			session, _, err := manager.Open(url, 640, 480)
			if err != nil {
				t.Fatal(err)
			}
			defer manager.Close(session.ID)
			_, snapshot, err := method.run(session, `window.evalEffects=(window.evalEffects||0)+1;throw new Error('synthetic-private-eval-token');`)
			if err == nil || snapshot != nil {
				t.Fatal("JavaScript exception was converted into a success result")
			}
			var actionError *SessionActionError
			if !errors.As(err, &actionError) || !actionError.EffectsPossible {
				t.Fatal("evaluation error lost possible-effect posture")
			}
			if strings.Contains(err.Error(), "synthetic-private-eval-token") {
				t.Fatal("public error echoed submitted JavaScript/exception content")
			}
			result, readErr := session.page.Timeout(2 * time.Second).Eval(`() => window.evalEffects`)
			if readErr != nil || result.Value.Num() != 1 {
				t.Fatal("failed evaluation was replayed or not dispatched")
			}
			if session.page.GetContext().Err() != nil {
				t.Fatal("evaluation error poisoned the session context")
			}
		})
	}
}
