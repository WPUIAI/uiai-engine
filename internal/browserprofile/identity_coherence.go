package browserprofile

import (
	"fmt"
	"runtime"
	"strings"
)

// userAgentPlatformCoherent reports whether an advertised user-agent's
// platform token matches the host OS (#152). The proven rejection input was
// a Macintosh UA on a Windows host - providers read the mismatch between
// advertised platform and real environment. No UA override is always
// coherent: the browser then advertises its own true identity.
//
// This is a coherence guard, not an anti-fingerprinting tool (#152 is
// deliberately provider-compliant): random_user_agent rotation across
// operating systems stays an operator choice, but it must be a visible
// choice, not a silent mismatch.
func userAgentPlatformCoherent(ua string, goos string) bool {
	ua = strings.TrimSpace(ua)
	if ua == "" {
		return true
	}
	lower := strings.ToLower(ua)
	advertised := ""
	switch {
	case strings.Contains(lower, "macintosh") || strings.Contains(lower, "mac os x"):
		advertised = "darwin"
	case strings.Contains(lower, "windows nt") || strings.Contains(lower, "windows phone"):
		advertised = "windows"
	case strings.Contains(lower, "x11") || strings.Contains(lower, "linux"):
		advertised = "linux"
	case strings.Contains(lower, "android"):
		advertised = "android"
	case strings.Contains(lower, "iphone") || strings.Contains(lower, "ipad"):
		advertised = "ios"
	default:
		// Unknown or synthetic platform tokens are not flagged here; the
		// known cross-OS mismatch class is the one with provider evidence.
		return true
	}
	if goos == "" {
		goos = runtime.GOOS
	}
	switch goos {
	case "darwin":
		return advertised == "darwin"
	case "windows":
		return advertised == "windows"
	case "linux":
		return advertised == "linux" || advertised == "android"
	default:
		return true
	}
}

// identityCoherenceWarning returns the typed warning for an incoherent UA
// override, or "" when the override is coherent.
func identityCoherenceWarning(ua string) string {
	if userAgentPlatformCoherent(ua, runtime.GOOS) {
		return ""
	}
	return fmt.Sprintf("profile advertises a cross-platform user agent that does not match this host (%s on %s): identity providers reject such mismatches as automated contexts - use a coherent UA or none", ua, runtime.GOOS)
}
