package observability

import "testing"

func TestClassRecovery(t *testing.T) {
	cases := map[string]struct {
		retryable bool
		first     string
	}{
		"selector_not_found": {false, "snapshot"},
		"url_not_allowed":    {false, "adjust_target"},
		"page_unavailable":   {true, "reopen_session"},
		"timeout":            {true, "retry"},
		"unknown":            {true, "retry"},
	}
	for class, want := range cases {
		retryable, recover := classRecovery(class)
		if retryable != want.retryable {
			t.Errorf("%s: retryable=%v want %v", class, retryable, want.retryable)
		}
		if len(recover) == 0 || recover[0] != want.first {
			t.Errorf("%s: recover=%v want first %q", class, recover, want.first)
		}
	}
	env := NewErrorEnvelope(ErrorEvent{Class: "selector_not_found"}, "boom", nil)
	if env.Retryable || len(env.Recover) == 0 {
		t.Fatalf("envelope missing recovery fields: %+v", env)
	}
}
