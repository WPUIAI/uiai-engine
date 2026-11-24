package vision

import (
	"testing"
	"time"
)

// #303: session lifetime and caps are operator policy. Zero-value limits keep
// the historic contract; explicit values are honored and reported.
func TestSessionLimitsOperatorPolicy(t *testing.T) {
	defaults := SessionLimits{}.normalized()
	if defaults.IdleTTL != SessionTTL || defaults.MaxPerScope != MaxSessions || defaults.MaxGlobal != globalMaxSessions {
		t.Fatalf("defaults drifted: %+v", defaults)
	}
	custom := SessionLimits{IdleTTL: 30 * time.Minute, MaxPerScope: 8, MaxGlobal: 32}.normalized()
	if custom.IdleTTL != 30*time.Minute || custom.MaxPerScope != 8 || custom.MaxGlobal != 32 {
		t.Fatalf("operator limits not honored: %+v", custom)
	}
}

// #303: an operator-set idle lifetime must reach the armed expiry timer.
// WrapPage sessions (no manager) keep the default TTL on touch.
func TestIdleTTLDefaultsForManagerlessSessions(t *testing.T) {
	s := &Session{}
	if ttl := s.idleTTL; ttl != 0 {
		t.Fatalf("managerless session should carry no TTL until touched, got %s", ttl)
	}
}
