package vision

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestClickCancellationBeforeDispatchHasNoEffect(t *testing.T) {
	_, manager, url := storageIsolationFixture(t, 1)
	t.Cleanup(manager.CloseAll)
	session, _, err := manager.Open(url, 640, 480)
	if err != nil {
		t.Fatal(err)
	}
	_, err = session.page.Timeout(2 * time.Second).Eval(`() => {window.clicks=0;document.body.innerHTML='<button id="cancelled">Cancel fixture</button>';document.querySelector('button').onclick=()=>window.clicks++;}`)
	if err != nil {
		t.Fatal(err)
	}
	caller, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = session.ClickContext(caller, "#cancelled")
	var actionError *SessionActionError
	if !errors.Is(err, context.Canceled) || !errors.As(err, &actionError) || actionError.EffectsPossible {
		t.Fatalf("caller cancellation lost its pre-dispatch posture: %v", err)
	}
	result, err := session.page.Timeout(2 * time.Second).Eval(`() => window.clicks`)
	if err != nil || result.Value.Num() != 0 {
		t.Fatal("cancelled click was dispatched")
	}
	if session.page.GetContext().Err() != nil {
		t.Fatal("caller cancellation poisoned the session context")
	}
}

func TestClickTimeoutReportsUncertainEffectsWithoutReplay(t *testing.T) {
	_, manager, url := storageIsolationFixture(t, 1)
	t.Cleanup(manager.CloseAll)
	session, _, err := manager.Open(url, 640, 480)
	if err != nil {
		t.Fatal(err)
	}
	_, err = session.page.Timeout(2 * time.Second).Eval(`() => {
  window.clicks=0;document.body.innerHTML='<button id="slow">Slow fixture</button>';
  document.querySelector('button').onclick=()=>{window.clicks++;const until=performance.now()+6500;while(performance.now()<until){}};
 }`)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_, err = session.Click("#slow")
	var actionError *SessionActionError
	if !errors.As(err, &actionError) || !actionError.EffectsPossible || actionError.Phase != "dispatch" {
		t.Fatalf("dispatched timeout omitted uncertain-effect posture: %v", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > 7*time.Second {
		t.Fatal("click did not respect its bounded deadline")
	}
	result, err := session.page.Timeout(3 * time.Second).Eval(`() => window.clicks`)
	if err != nil || result.Value.Num() != 1 {
		t.Fatal("uncertain click was automatically replayed or never dispatched")
	}
	if session.page.GetContext().Err() != nil {
		t.Fatal("action timeout poisoned the shared session context")
	}
}
