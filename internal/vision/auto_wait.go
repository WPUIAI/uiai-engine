package vision

import (
	"fmt"
	"time"
)

// AutoWait settles the current page after an action. It is intentionally bounded
// and does not replace explicit WaitFor for page-specific conditions.
func (s *Session) AutoWait(timeoutMs int) error {
	if timeoutMs <= 0 {
		return nil
	}
	if timeoutMs > 5000 {
		timeoutMs = 5000
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return fmt.Errorf("session closed")
	}
	time.Sleep(time.Duration(min(timeoutMs, 300)) * time.Millisecond)
	_ = s.page.Timeout(time.Duration(timeoutMs)*time.Millisecond).WaitDOMStable(100*time.Millisecond, 0.2)
	s.URL = safeEvalStr(s.page, `() => window.location.href`)
	s.Title = safeEvalStr(s.page, `() => document.title`)
	s.touch()
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
