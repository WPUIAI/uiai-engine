package vision

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
)

type stalledScreenshotClient struct{ events chan *cdp.Event }

func (c *stalledScreenshotClient) Event() <-chan *cdp.Event { return c.events }
func (c *stalledScreenshotClient) Call(ctx context.Context, _ string, method string, _ interface{}) ([]byte, error) {
	switch method {
	case "Target.attachToTarget":
		return []byte(`{"sessionId":"synthetic-session"}`), nil
	case "Page.captureScreenshot":
		if _, ok := ctx.Deadline(); !ok {
			return nil, fmt.Errorf("screenshot transport received no deadline")
		}
		<-ctx.Done()
		return nil, ctx.Err()
	default:
		return []byte(`{}`), nil
	}
}

func TestSessionActionSnapshotDeadlineDoesNotPoisonPage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	browser := rod.New().NoDefaultDevice().Context(ctx).Client(&stalledScreenshotClient{events: make(chan *cdp.Event)})
	if err := browser.Connect(); err != nil {
		t.Fatal(err)
	}
	page, err := browser.PageFromTarget("synthetic-target")
	if err != nil {
		t.Fatal(err)
	}
	session := &Session{page: page}
	start := time.Now()
	session.mu.Lock()
	_, err = session.snap(start)
	session.mu.Unlock()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("snapshot lacks bounded cancellation: %v", err)
	}
	if time.Since(start) > 5*time.Second {
		t.Fatal("snapshot exceeded bounded recovery window")
	}
	if page.GetContext().Err() != nil {
		t.Fatal("action snapshot cancelled the shared session context")
	}
	if !session.mu.TryLock() {
		t.Fatal("snapshot left the session action lock held")
	}
	session.mu.Unlock()
}

func TestFillAllowsSlowFocusWithoutLeakingLookupDeadline(t *testing.T) {
	_, manager, url := storageIsolationFixture(t, 1)
	t.Cleanup(manager.CloseAll)
	session, _, err := manager.Open(url, 640, 480)
	if err != nil {
		t.Fatal(err)
	}
	_, err = session.page.Timeout(2 * time.Second).Eval(`() => {
  document.body.innerHTML='<input id="slow-focus">';
  document.querySelector('input').addEventListener('focus',()=>{
   const until=performance.now()+450;while(performance.now()<until){}
  },{once:true});
 }`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = session.Fill("#slow-focus", "synthetic-input"); err != nil {
		t.Fatalf("fill inherited the short lookup timeout: %v", err)
	}
	value, err := session.page.Timeout(2 * time.Second).Eval(`() => document.querySelector('input').value`)
	if err != nil || value.Value.Str() != "synthetic-input" {
		t.Fatal("fill did not update the intended input")
	}
}

func TestFillTimeoutReleasesSessionForDiagnosticsAndClose(t *testing.T) {
	_, manager, url := storageIsolationFixture(t, 1)
	t.Cleanup(manager.CloseAll)
	session, _, err := manager.Open(url, 640, 480)
	if err != nil {
		t.Fatal(err)
	}
	_, err = session.page.Timeout(2 * time.Second).Eval(`() => {
  document.body.innerHTML='<input id="blocked-focus">';
  document.querySelector('input').addEventListener('focus',()=>{
   const until=performance.now()+6500;while(performance.now()<until){}
  },{once:true});
 }`)
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_, err = session.Fill("#blocked-focus", "synthetic-input")
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("fill did not report its timeout: %v", err)
	}
	if time.Since(start) > 8*time.Second {
		t.Fatal("fill timeout exceeded action budget")
	}
	if session.page.GetContext().Err() != nil {
		t.Fatal("fill poisoned the session page context")
	}
	_ = session.Diagnostics(10, "all", false)
	result, snapshot, evalErr := session.Eval("return 'session-alive'")
	if evalErr != nil || result != "session-alive" || snapshot == nil {
		t.Fatalf("eval remained poisoned after fill timeout: %v", evalErr)
	}
	closed := make(chan error, 1)
	go func() { closed <- manager.Close(session.ID) }()
	select {
	case err := <-closed:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("close remained wedged after a timed-out fill")
	}
	if len(manager.List()) != 0 {
		t.Fatal("closed session remained active")
	}
}
