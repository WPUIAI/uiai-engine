package vision

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"
)

// #293: navigation that completed must never be reported as a navigation
// failure when only the capture fails, and a capture failure must strike
// the pool's browser degradation counter.

func TestCaptureFailureAfterNavigationWrapsHonestError(t *testing.T) {
	underlying := errors.New("{-32603 Internal error }")
	start := time.Now().Add(-150 * time.Millisecond)
	err := captureFailureAfterNavigation("https://docs.openclaw.ai/cli/doctor/sqlite-maintenance", start, underlying)

	if !errors.Is(err, ErrPageCaptureFailed) {
		t.Fatalf("error must wrap ErrPageCaptureFailed, got %v", err)
	}
	if !errors.Is(err, underlying) {
		t.Fatalf("error must keep the underlying CDP failure for errors.Is, got %v", err)
	}
	msg := err.Error()
	if !strings.Contains(msg, "navigation to https://docs.openclaw.ai/cli/doctor/sqlite-maintenance completed") {
		t.Fatalf("error must state the completed navigation, got %q", msg)
	}
	if !strings.Contains(msg, "page capture failed after 150ms") {
		t.Fatalf("error must carry the typed class and stage timing, got %q", msg)
	}
	if strings.Contains(msg, "navigation failed") {
		t.Fatalf("error must never read as a navigation failure, got %q", msg)
	}
}

// strikeCountingPool is a PoolSource that only counts MarkFailure strikes.
type strikeCountingPool struct {
	failures int
}

func (sp *strikeCountingPool) GetPage() (*rod.Page, error) { return nil, fmt.Errorf("unused") }
func (sp *strikeCountingPool) ReleasePage(*rod.Page)       {}
func (sp *strikeCountingPool) ValidateNavigationURL(string) error {
	return nil
}
func (sp *strikeCountingPool) IsBrowserAlive() bool { return true }
func (sp *strikeCountingPool) MarkFailure()         { sp.failures++ }
func (sp *strikeCountingPool) Reset()               {}
func (sp *strikeCountingPool) Screenshot(ScreenshotOpts) (*ScreenshotResult, error) {
	return nil, fmt.Errorf("unused")
}
func (sp *strikeCountingPool) Stats() map[string]any { return nil }
func (sp *strikeCountingPool) Close()                {}

// TestCaptureAfterNavigationStrikesPoolOnCaptureFailure injects the #293
// failure mechanics deterministically: a page bound to an already-canceled
// context cannot capture, which is the degraded-target capture failure the
// engine saw on docs.openclaw.ai while navigation still completed.
func TestCaptureAfterNavigationStrikesPoolOnCaptureFailure(t *testing.T) {
	if os.Getenv("UIAI_INPUT_TESTS") == "" {
		t.Skip("set UIAI_INPUT_TESTS=1 for browser capture regression")
	}
	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	page, err := pool.GetPage()
	if err != nil {
		t.Fatal(err)
	}
	defer pool.ReleasePage(page)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	canceled := page.Context(ctx)

	sp := &strikeCountingPool{}
	sess := &Session{ID: "capture-test", Width: 640, Height: 480, page: canceled, pool: sp}

	data, err := sess.captureAfterNavigation("https://example.test/", time.Now())
	if err == nil {
		t.Fatal("capture on a canceled target must fail")
	}
	if data != nil {
		t.Fatalf("no capture bytes expected, got %d", len(data))
	}
	if sp.failures != 1 {
		t.Fatalf("capture failure must strike the pool degradation counter once, got %d", sp.failures)
	}
	if !errors.Is(err, ErrPageCaptureFailed) {
		t.Fatalf("capture failure must wrap ErrPageCaptureFailed, got %v", err)
	}
	if !strings.Contains(err.Error(), "navigation to https://example.test/ completed") {
		t.Fatalf("capture failure must report the completed navigation, got %v", err)
	}
}
