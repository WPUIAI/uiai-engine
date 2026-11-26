package vision

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// #66: WebAuthn attempts are recorded by the injected observer even when the
// challenge itself lives in a cross-origin iframe; the classifier raises the
// webauthn class and diagnostics carry the honest completion hint.
func TestWebAuthnAttemptDetectionAndHint(t *testing.T) {
	if os.Getenv("UIAI_INPUT_TESTS") == "" {
		t.Skip("set UIAI_INPUT_TESTS=1 for browser webauthn regression")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `<!doctype html><html><body>
		<button id="go">Continue</button>
		<script>
		document.getElementById('go').addEventListener('click', () => {
			navigator.credentials.get({ publicKey: {
				challenge: new Uint8Array(16),
				rpId: location.hostname,
				userVerification: 'preferred',
			}}).catch(() => {});
		});
		</script>
		</body></html>`)
	}))
	defer site.Close()

	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	sm := NewSessionManager(pool)
	defer sm.CloseAll()

	session, _, err := sm.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	// Before the flow: no attempts, no class.
	if len(session.WebAuthnAttempts()) != 0 {
		t.Fatal("attempts recorded before any credentials call")
	}
	if ch, _ := session.DetectAuthChallenge(); ch != nil && ch.Class == "webauthn" {
		t.Fatal("webauthn class raised before any credentials call")
	}

	// Trigger the flow: the headless environment rejects the call, but the
	// observer records the attempt first.
	if _, _, err := session.Eval(`document.getElementById('go').click(); return 'clicked';`); err != nil {
		t.Fatalf("trigger: %v", err)
	}
	attempts := session.WebAuthnAttempts()
	if len(attempts) != 1 || attempts[0].Kind != "get" {
		t.Fatalf("credentials.get attempt not recorded: %+v", attempts)
	}

	// The classifier raises the webauthn class from the attempt ring.
	ch, _ := session.DetectAuthChallenge()
	if ch == nil || ch.Class != "webauthn" {
		t.Fatalf("webauthn class not raised: %+v", ch)
	}

	// Diagnostics carry the honest hint naming the completion path.
	snap := session.DiagnosticsWithOptions(DiagnosticsOptions{Limit: 10, Format: "full"})
	if !strings.Contains(snap.WebAuthnHint, "webauthn challenge observed (get)") ||
		!strings.Contains(snap.WebAuthnHint, "cannot complete") {
		t.Fatalf("diagnostics hint missing or dishonest: %q", snap.WebAuthnHint)
	}
}
