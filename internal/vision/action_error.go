package vision

import "fmt"

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
		return fmt.Sprintf("%s outcome uncertain during %s: %v", e.Operation, e.Phase, e.Cause)
	}
	return fmt.Sprintf("%s failed during %s: %v", e.Operation, e.Phase, e.Cause)
}
func (e *SessionActionError) Unwrap() error { return e.Cause }
