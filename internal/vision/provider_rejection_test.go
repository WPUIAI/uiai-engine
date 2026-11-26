package vision

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestProviderRejectionDetectedAndHinted proves the #152 acceptance: a
// provider page rejecting the automated browser itself raises the typed
// provider_browser_rejected class (never a bare click 500 or a silent
// oidc_redirect), enters the pending lease, and diagnostics carry the honest
// handoff guidance.
func TestProviderRejectionDetectedAndHinted(t *testing.T) {
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><h1>Couldn't sign you in</h1>
		<p>This browser or app may not be secure.</p>
		<p><a href="#">Try using a different browser</a></p>
		</body></html>`))
	}))
	defer site.Close()
	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := NewSessionManager(pool)
	defer manager.CloseAll()

	session, _, err := manager.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatal(err)
	}
	challenge, err := session.DetectAuthChallenge()
	if err != nil || challenge == nil {
		t.Fatalf("provider rejection not detected: %v %v", err, challenge)
	}
	if challenge.Class != ProviderBrowserRejectedClass {
		t.Fatalf("rejection class is %q, want provider_browser_rejected", challenge.Class)
	}
	if session.LeaseState() != LeaseAwaitingAuth {
		t.Fatalf("rejection did not enter the pending lease: %s", session.LeaseState())
	}

	diag := session.Diagnostics(100, "all", false)
	if diag.ProviderRejectionHint == "" {
		t.Fatal("diagnostics missing the provider rejection handoff hint")
	}
	if want := "browser_attach"; !contains(diag.ProviderRejectionHint, want) {
		t.Fatalf("hint does not route to the attach handoff: %q", diag.ProviderRejectionHint)
	}
}

func contains(haystack, needle string) bool {
	return len(needle) > 0 && len(haystack) >= len(needle) && indexOf(haystack, needle) >= 0
}

func indexOf(haystack, needle string) int {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
