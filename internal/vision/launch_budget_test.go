package vision

import (
	"testing"
	"time"
)

// #292: the pool launch budget must stay strictly under the /api/session route
// deadline (routes.WithDeadline(45s)). When they were both 45s, a wedged
// chromium launch consumed the entire route budget and callers received a
// bare deadline 503 with no cause. The 10s headroom lets the route answer
// with the typed browser_launch_failed refusal instead.
func TestLaunchTimeoutStaysUnderRouteDeadline(t *testing.T) {
	const routeDeadline = 45 * time.Second
	if LaunchTimeout >= routeDeadline {
		t.Fatalf("LaunchTimeout %s must stay strictly under the %s route deadline", LaunchTimeout, routeDeadline)
	}
	if routeDeadline-LaunchTimeout < 5*time.Second {
		t.Fatalf("only %s of headroom to write the typed refusal; want >= 5s", routeDeadline-LaunchTimeout)
	}
}
