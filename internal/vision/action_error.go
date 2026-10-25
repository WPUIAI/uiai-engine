package vision

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-rod/rod"
)

// SessionActionError reports an operation stage without claiming that a failed
// response proves the operation had no effect. It is telemetry, not authority.
type SessionActionError struct {
	Operation       string
	Phase           string
	Cause           error
	EffectsPossible bool
}

func (e *SessionActionError) Error() string {
	if e.EffectsPossible {
		return fmt.Sprintf("%s outcome uncertain during %s: %v", e.Operation, e.Phase, e.CauseClass())
	}
	return fmt.Sprintf("%s failed during %s: %v", e.Operation, e.Phase, e.CauseClass())
}
func (e *SessionActionError) Unwrap() error { return e.Cause }

// CauseClass keeps navigation/cancellation clues without echoing submitted
// JavaScript, exception values, credentials or protocol parameters to callers.
func (e *SessionActionError) CauseClass() string {
	if errors.Is(e.Cause, context.Canceled) {
		return "operation_cancelled"
	}
	if errors.Is(e.Cause, context.DeadlineExceeded) {
		return "deadline_exceeded"
	}
	var javascriptError *rod.EvalError
	if errors.As(e.Cause, &javascriptError) {
		return "javascript_exception"
	}
	if e.Cause != nil {
		message := strings.ToLower(e.Cause.Error())
		for _, fragment := range []string{"execution context was destroyed", "cannot find context with specified id", "cannot find context with id", "inspected target navigated or closed"} {
			if strings.Contains(message, fragment) {
				return "execution_context_changed"
			}
		}
	}
	return "browser_action_failed"
}
