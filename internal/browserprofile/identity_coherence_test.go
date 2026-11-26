package browserprofile

import "testing"

// TestUserAgentPlatformCoherent covers the #152 incident exactly: a
// Macintosh-advertised UA on a Windows host is the proven rejection input
// and must fail coherence, while same-platform and empty (true-identity)
// overrides stay coherent.
func TestUserAgentPlatformCoherent(t *testing.T) {
	incidentUA := "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36"
	if userAgentPlatformCoherent(incidentUA, "windows") {
		t.Fatal("the incident's macOS UA on a Windows host passed coherence")
	}
	if !userAgentPlatformCoherent(incidentUA, "darwin") {
		t.Fatal("same-platform macOS pairing unexpectedly incoherent")
	}
	if !userAgentPlatformCoherent("", "windows") {
		t.Fatal("empty override (true browser identity) must be coherent")
	}
	windowsUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"
	if !userAgentPlatformCoherent(windowsUA, "windows") {
		t.Fatal("coherent Windows pairing failed")
	}
	if userAgentPlatformCoherent(windowsUA, "darwin") {
		t.Fatal("Windows UA on darwin host passed coherence")
	}
	linuxUA := "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36"
	if !userAgentPlatformCoherent(linuxUA, "linux") {
		t.Fatal("coherent Linux pairing failed")
	}
	if userAgentPlatformCoherent(incidentUA, "") == false && identityCoherenceWarning(incidentUA) == "" {
		t.Fatal("incoherent override produced no warning")
	}
}
