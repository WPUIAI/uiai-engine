package routes

import (
	"testing"

	"github.com/philoveracity/uiai-engine/internal/vision"
)

func TestBrowserHealthPayloadUnavailableWhenPoolNil(t *testing.T) {
	payload := browserHealthPayload(nil)
	if got, want := payload["status"], "unavailable"; got != want {
		t.Fatalf("status=%v want=%v", got, want)
	}
	if got, want := browserHealthStatus(nil), 503; got != want {
		t.Fatalf("status code=%d want=%d", got, want)
	}
}

func TestBrowserHealthStandbyWhenPoolLazyIdle(t *testing.T) {
	pool := &vision.Pool{}
	payload := browserHealthPayload(pool)
	if got, want := payload["status"], "standby"; got != want {
		t.Fatalf("status=%v want=%v", got, want)
	}
	if got, want := browserHealthStatus(pool), 200; got != want {
		t.Fatalf("status code=%d want=%d", got, want)
	}
}
