package routes

import (
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestWriteSessionErrorCallerInputClassesIs4xx covers #226: caller-input
// failures must surface as 4xx so retry/alert logic does not treat them as
// engine outages. The structured envelope (id/class/message/next action)
// stays unchanged.
func TestWriteSessionErrorCallerInputClassesIs4xx(t *testing.T) {
	cases := []struct {
		class  string
		status int
	}{
		{"selector_not_found", 404},
		{"unknown_key", 400},
		{"url_not_allowed", 400},
		{"session_capacity", 429},
		{"timeout", 500},              // server-side condition stays 5xx
		{"screenshot_failed", 500},    // server-side condition stays 5xx
		{"page_unavailable", 500},     // server-side condition stays 5xx
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		err := fmt.Errorf("synthetic %s failure", c.class)
		writeSessionError(w, 500, c.class, err, nil)
		if w.Code != c.status {
			t.Fatalf("class %s: got status %d want %d", c.class, w.Code, c.status)
		}
		body := w.Body.String()
		if !strings.Contains(body, `"error_class":"`+c.class+`"`) {
			t.Fatalf("class %s: envelope lost error_class: %s", c.class, body)
		}
	}
}

// TestClassifySessionErrorCallerInput covers the classifier additions that
// feed the 4xx remap.
func TestClassifySessionErrorCallerInput(t *testing.T) {
	if got := classifySessionError(errors.New(`unknown key: super (supported: Enter, Tab)`)); got != "unknown_key" {
		t.Fatalf("unknown key classified as %s", got)
	}
	if got := classifySessionError(errors.New("max sessions for scope host:loopback reached (4)")); got != "session_capacity" {
		t.Fatalf("capacity classified as %s", got)
	}
}
