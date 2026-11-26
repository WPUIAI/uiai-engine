package vision

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/launcher"
)

// startOwnerBrowser launches a standalone "owner browser" with a fixed CDP
// debugger port, simulating the human's already-running authenticated
// Chrome (#143). The test never attaches to anything it did not launch.
func startOwnerBrowser(t *testing.T, port int) (*launcher.Launcher, string) {
	t.Helper()
	portStr := strconv.Itoa(port)
	l := launcher.New().
		Headless(true).
		Set("remote-debugging-port", portStr).
		Set("remote-debugging-address", "127.0.0.1")
	wsURL, err := l.Launch()
	if err != nil {
		t.Skipf("owner browser launch unavailable in this environment: %v", err)
	}
	return l, wsURL
}

func ownerDebuggerAlive(t *testing.T, port int) bool {
	t.Helper()
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/json/version", port))
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}

// TestAttachPoolAdoptsOwnerProfile proves the #143 acceptance path: a
// session attached to a running browser inherits the owner profile (state
// persists across attached tabs), all native actions work on the attached
// page, releasing a session closes only the tab, and closing the attach
// source never shuts down the owner's browser.
func TestAttachPoolAdoptsOwnerProfile(t *testing.T) {
	if os.Getenv("UIAI_INPUT_TESTS") == "" {
		t.Skip("set UIAI_INPUT_TESTS=1 for attach browser tests")
	}
	const ownerPort = 19301
	l, _ := startOwnerBrowser(t, ownerPort)
	defer func() {
		if pid := l.PID(); pid > 0 {
			_ = exec.Command("kill", "-9", strconv.Itoa(pid)).Run() // #nosec G204 -- test-owned launcher PID
		}
		l.Cleanup()
	}()

	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><body><input id="field"><span id="echo"></span>
		<script>document.getElementById('field').addEventListener('input', () => { document.getElementById('echo').textContent = 'typed:' + document.getElementById('field').value; });</script>
		</body></html>`))
	}))
	defer site.Close()

	attach := NewAttachPool("owner-test", fmt.Sprintf("http://127.0.0.1:%d", ownerPort), true)

	if err := attach.ValidateNavigationURL(site.URL); err != nil {
		t.Fatalf("private fixture rejected though endpoint allows private: %v", err)
	}

	manager := NewSessionManagerWithPoolsOptions(attach, SessionLimits{IdleTTL: time.Minute, MaxPerScope: 4, MaxGlobal: 8})

	// Session one: write profile state, act on the page, close (tab release).
	sessA, snapA, err := manager.OpenAttached(attach, site.URL, 640, 480, DefaultHostScope())
	if err != nil {
		t.Fatalf("attached open failed: %v", err)
	}
	if snapA == nil || len(snapA.Screenshot) == 0 {
		t.Fatalf("attached session missing native snapshot/screenshot")
	}
	if _, _, err := sessA.Eval(`localStorage.setItem('attach-adoption', 'persisted'); return localStorage.getItem('attach-adoption');`); err != nil {
		t.Fatalf("profile state write failed: %v", err)
	}
	if _, err := sessA.Fill("#field", "adopted"); err != nil {
		t.Fatalf("native fill failed on attached page: %v", err)
	}
	echo, _, err := sessA.Eval(`return document.getElementById('echo').textContent;`)
	if err != nil || echo != "typed:adopted" {
		t.Fatalf("attached fill did not deliver input: %v %q", err, echo)
	}
	if err := manager.Close(sessA.ID); err != nil {
		t.Fatalf("attached session close failed: %v", err)
	}

	// Session two in a fresh tab: the owner profile persisted across tabs.
	sessB, _, err := manager.OpenAttached(attach, site.URL, 640, 480, DefaultHostScope())
	if err != nil {
		t.Fatalf("second attached open failed: %v", err)
	}
	value, _, err := sessB.Eval(`return localStorage.getItem('attach-adoption');`)
	if err != nil || value != "persisted" {
		t.Fatalf("owner profile state did not persist across attached tabs: %v %q", err, value)
	}
	if err := manager.Close(sessB.ID); err != nil {
		t.Fatal(err)
	}

	// Closing the attach source disconnects only: the owner browser lives.
	attach.Close()
	if !ownerDebuggerAlive(t, ownerPort) {
		t.Fatal("attach Close() shut down the owner browser - the engine must never own an attached browser's lifecycle")
	}

	// Reconnect after Close still works (lazy reconnect by design).
	sessC, _, err := manager.OpenAttached(attach, site.URL, 640, 480, DefaultHostScope())
	if err != nil {
		t.Fatalf("reconnect attach failed after source close: %v", err)
	}
	if err := manager.Close(sessC.ID); err != nil {
		t.Fatal(err)
	}
	if !attach.IsBrowserAlive() {
		t.Fatal("attached browser reported dead while its debugger endpoint answers")
	}
}

// TestAttachPoolValidationGuardProves the fail-safe URL policy: private
// targets stay blocked unless the endpoint opts in, and dangerous schemes
// are always refused.
func TestAttachPoolValidationGuard(t *testing.T) {
	attach := NewAttachPool("strict", "http://127.0.0.1:19999", false)
	if err := attach.ValidateNavigationURL("http://127.0.0.1:18099/x"); err == nil {
		t.Fatal("private target accepted on an endpoint that did not opt in")
	}
	if err := attach.ValidateNavigationURL("file:///etc/passwd"); err == nil {
		t.Fatal("dangerous scheme accepted")
	}
	if err := attach.ValidateNavigationURL("https://example.test/"); err != nil {
		t.Fatalf("public https target rejected: %v", err)
	}
	// Unknown endpoints never connect - the unreachable debugger surfaces
	// as a typed attach error, not a silent success.
	_, err := attach.GetPage()
	if err == nil {
		t.Fatal("unreachable attach endpoint returned a page")
	}
}
