package observability

import "testing"

func TestClassRecovery(t *testing.T) {
	cases := map[string]struct {
		retryable bool
		first     string
	}{
		"selector_not_found": {false, "snapshot"},
		"url_not_allowed":    {false, "adjust_target"},
		"page_unavailable":   {false, "reopen_session"},
		"session_reaped":     {false, "reopen_session"},
		"pool_saturated":     {true, "wait_capacity"},
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
	if env.Code != "selector_not_found" || env.Retryable || len(env.Recover) == 0 {
		t.Fatalf("envelope missing recovery fields: %+v", env)
	}

	lost := NewErrorEnvelope(ErrorEvent{Class: "session_reaped"}, "gone", nil)
	if lost.Code != "session_reaped" || len(lost.StateLost) != 3 || lost.StateLost[0] != "cookies" {
		t.Fatalf("session loss is not machine-readable: %+v", lost)
	}

	unknown := NewErrorEnvelope(ErrorEvent{}, "boom", nil)
	if unknown.Code != "unknown" {
		t.Fatalf("missing stable fallback code: %+v", unknown)
	}
}
