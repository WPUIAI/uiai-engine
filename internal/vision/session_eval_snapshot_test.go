package vision

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/cdp"
)

type evaluationScreenshotClient struct{ stalledScreenshotClient }

func (c *evaluationScreenshotClient) Call(ctx context.Context, session, method string, params interface{}) ([]byte, error) {
	switch method {
	case "Runtime.evaluate":
		return []byte(`{"result":{"type":"object","objectId":"synthetic-window"}}`), nil
	case "Runtime.callFunctionOn":
		return []byte(`{"result":{"type":"string","value":"synthetic-result"}}`), nil
	}
	return c.stalledScreenshotClient.Call(ctx, session, method, params)
}

func TestEvaluationScreenshotFailureIsBoundedAndNotSwallowed(t *testing.T) {
	for _, method := range []struct {
		name string
		run  func(*Session, string) (string, *SnapResult, error)
	}{
		{"eval", (*Session).Eval},
		{"eval_async", func(s *Session, j string) (string, *SnapResult, error) { return s.EvalAsync(j, 5000) }},
	} {
		t.Run(method.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client := &evaluationScreenshotClient{stalledScreenshotClient{events: make(chan *cdp.Event)}}
			browser := rod.New().NoDefaultDevice().Context(ctx).Client(client)
			if err := browser.Connect(); err != nil {
				t.Fatal(err)
			}
			page, err := browser.PageFromTarget("synthetic-target")
			if err != nil {
				t.Fatal(err)
			}
			session := &Session{page: page}
			start := time.Now()
			_, snapshot, err := method.run(session, "return true")
			var actionError *SessionActionError
			if !errors.Is(err, context.DeadlineExceeded) || !errors.As(err, &actionError) || actionError.Phase != "snapshot" || snapshot != nil {
				t.Fatalf("capture failure lost its typed bounded error: %v", err)
			}
			if time.Since(start) > 5*time.Second {
				t.Fatal("evaluation screenshot exceeded shared capture deadline")
			}
			if page.GetContext().Err() != nil {
				t.Fatal("capture failure cancelled shared page context")
			}
			if !session.mu.TryLock() {
				t.Fatal("evaluation capture left action lock held")
			}
			session.mu.Unlock()
		})
	}
}
