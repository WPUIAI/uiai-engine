package vision

import (
	"context"
	"fmt"
	"time"
)

// Eval runs JavaScript through the shared bounded evaluation boundary.
func (s *Session) Eval(js string) (string, *SnapResult, error) {
	return s.EvalContext(context.Background(), js)
}
func (s *Session) EvalContext(caller context.Context, js string) (string, *SnapResult, error) {
	return s.evaluateContext(caller, "eval", `() => { `+js+` }`, 30*time.Second)
}

func (s *Session) EvalAsync(js string, timeoutMs int) (string, *SnapResult, error) {
	return s.EvalAsyncContext(context.Background(), js, timeoutMs)
}
func (s *Session) EvalAsyncContext(caller context.Context, js string, timeoutMs int) (string, *SnapResult, error) {
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	if timeoutMs > 15000 {
		timeoutMs = 15000
	}
	wrapper := fmt.Sprintf(`async () => {
  const __timeout=new Promise((_,reject)=>setTimeout(()=>reject(new Error("eval_async timeout after %dms")),%d));
  const __work=(async()=>{ %s })();
  return await Promise.race([__work,__timeout]);
 }`, timeoutMs, timeoutMs, js)
	return s.evaluateContext(caller, "eval_async", wrapper, time.Duration(timeoutMs+1000)*time.Millisecond)
}

// Evaluation cancellation stops waiting, not arbitrary JavaScript already running
// in Chromium. A failed dispatch/result/observation never proves zero effects.
func (s *Session) evaluateContext(caller context.Context, operation, script string, budget time.Duration) (string, *SnapResult, error) {
	failure := func(phase string, err error, effects bool) error {
		return &SessionActionError{Operation: operation, Phase: phase, Cause: err, EffectsPossible: effects}
	}
	if err := caller.Err(); err != nil {
		return "", nil, failure("before_dispatch", err, false)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return "", nil, fmt.Errorf("session closed")
	}
	if err := caller.Err(); err != nil {
		return "", nil, failure("before_dispatch", err, false)
	}
	start := time.Now()
	s.touch()
	defer s.touch()
	ctx, cancel := sessionActionContext(s.page.GetContext(), caller, budget)
	defer cancel()
	page := s.page.Context(ctx)
	result, err := page.Eval(script)
	if err != nil {
		return "", nil, failure("dispatch", err, true)
	}
	value := ""
	if result != nil {
		value = result.Value.Str()
		if value == "" {
			raw, encodeErr := result.Value.MarshalJSON()
			if encodeErr != nil {
				return "", nil, failure("result", encodeErr, true)
			}
			value = string(raw)
		}
		value = sanitizeDiagnosticText(value)
	}
	settle := time.NewTimer(100 * time.Millisecond)
	defer settle.Stop()
	select {
	case <-ctx.Done():
		return "", nil, failure("settle", ctx.Err(), true)
	case <-settle.C:
	}
	s.URL = safeEvalStr(page, `() => window.location.href`)
	s.Title = safeEvalStr(page, `() => document.title`)
	if err := ctx.Err(); err != nil {
		return "", nil, failure("observe", err, true)
	}
	snapshot, err := s.snapWithPage(page, start)
	if err != nil {
		return "", nil, failure("snapshot", err, true)
	}
	return value, snapshot, nil
}
