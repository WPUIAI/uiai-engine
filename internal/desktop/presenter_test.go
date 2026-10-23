package desktop

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/desktopcontract"
)

type recordingLauncher struct {
	calls []string
	err   error
}

func (l *recordingLauncher) Open(_ context.Context, target string) error {
	l.calls = append(l.calls, target)
	return l.err
}

func request(key string) desktopcontract.DesktopPresentationRequest {
	return desktopcontract.DesktopPresentationRequest{
		Schema: desktopcontract.SchemaDesktopPresentationRequestV1,
		Mode:   "full", Reason: "operator_request", Focus: true, ExpiresInMS: 30_000,
		RequestedBy:    desktopcontract.ClientRef{ClientType: "pi", ClientID: "pi-test"},
		IdempotencyKey: key,
	}
}

func TestEnsureVisibleIsIdempotentAndDoesNotCreateSession(t *testing.T) {
	launcher := &recordingLauncher{}
	lookups := 0
	presenter := NewPresenter(func(id string) bool { lookups++; return id == "session-1" }, launcher)

	first, err := presenter.EnsureVisible(context.Background(), "session-1", request("request-1"))
	if err != nil {
		t.Fatal(err)
	}
	second, err := presenter.EnsureVisible(context.Background(), "session-1", request("request-1"))
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatalf("repeated request diverged: %#v != %#v", first, second)
	}
	if first.Status != "launching" {
		t.Fatalf("status = %q", first.Status)
	}
	if len(launcher.calls) != 1 {
		t.Fatalf("launcher calls = %d", len(launcher.calls))
	}
	if lookups != 1 {
		t.Fatalf("session lookups = %d", lookups)
	}
	if !strings.Contains(launcher.calls[0], "cockpit://live/session/session-1?") || strings.Contains(launcher.calls[0], "token") {
		t.Fatalf("unsafe activation URL: %s", launcher.calls[0])
	}
}

func TestEnsureVisibleReturnsTypedRecoveryWithoutLaunchingMissingSession(t *testing.T) {
	launcher := &recordingLauncher{}
	presenter := NewPresenter(func(string) bool { return false }, launcher)
	receipt, err := presenter.EnsureVisible(context.Background(), "missing-session", request("missing-1"))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "session_missing" || receipt.ReasonCode != "canonical_session_missing" {
		t.Fatalf("receipt = %#v", receipt)
	}
	if len(launcher.calls) != 0 {
		t.Fatal("missing session must not launch Cockpit")
	}
}

func TestHeadlessFallbackAndAck(t *testing.T) {
	launcher := &recordingLauncher{err: ErrDesktopUnavailable}
	presenter := NewPresenter(func(string) bool { return true }, launcher)
	receipt, err := presenter.EnsureVisible(context.Background(), "session-2", request("request-2"))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "desktop_unavailable" || receipt.ReasonCode != "fpv_recovery_available" {
		t.Fatalf("receipt = %#v", receipt)
	}

	launcher.err = nil
	visible, err := presenter.EnsureVisible(context.Background(), "session-3", request("request-3"))
	if err != nil {
		t.Fatal(err)
	}
	visible, err = presenter.Acknowledge(context.Background(), visible.PresentationID, Acknowledgement{Status: "visible", CockpitInstanceID: "cockpit-1"})
	if err != nil {
		t.Fatal(err)
	}
	status, err := presenter.Status(context.Background(), visible.PresentationID)
	if err != nil {
		t.Fatal(err)
	}
	if status.Status != "visible" || status.CockpitInstanceID != "cockpit-1" {
		t.Fatalf("status = %#v", status)
	}
}

func TestIdempotencyConflictAndExpiry(t *testing.T) {
	presenter := NewPresenter(func(string) bool { return true }, &recordingLauncher{})
	now := time.Date(2026, 8, 3, 0, 0, 0, 0, time.UTC)
	presenter.now = func() time.Time { return now }
	first, err := presenter.EnsureVisible(context.Background(), "session-4", request("same-key"))
	if err != nil {
		t.Fatal(err)
	}
	changed := request("same-key")
	changed.Mode = "pip"
	if _, err := presenter.EnsureVisible(context.Background(), "session-4", changed); !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("error = %v", err)
	}
	now = now.Add(31 * time.Second)
	expired, err := presenter.Status(context.Background(), first.PresentationID)
	if err != nil {
		t.Fatal(err)
	}
	if expired.Status != "expired" {
		t.Fatalf("status = %q", expired.Status)
	}
}
