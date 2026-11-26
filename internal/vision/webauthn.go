package vision

import (
	"encoding/json"
	"fmt"
	"strings"
)

// #66: WebAuthn/passkey flows were invisible — the challenge often lives in
// a cross-origin iframe whose text the page classifier cannot read (the
// Bitwarden repro embeds webauthn-connector.html reading only "Awaiting
// security key"). The session now injects a new-document observer that wraps
// navigator.credentials.get/create, so the attempt is recorded before the
// headless environment rejects it; diagnostics surface the attempts with the
// honest completion guidance. Headless sessions cannot complete a passkey —
// detection and a typed path are what this slice delivers.

const webAuthnObserverJS = `(() => {
	if (window.__uiaiWebAuthnInstalled) return;
	window.__uiaiWebAuthnInstalled = true;
	window.__uiaiWebAuthnAttempts = [];
	const record = kind => {
		try {
			window.__uiaiWebAuthnAttempts.push({
				kind: kind,
				origin: location.origin,
				ts: Date.now(),
			});
			if (window.__uiaiWebAuthnAttempts.length > 20) window.__uiaiWebAuthnAttempts.shift();
		} catch (e) { /* recording must never break the page */ }
	};
	if (navigator.credentials) {
		const origGet = navigator.credentials.get.bind(navigator.credentials);
		navigator.credentials.get = function(opts) {
			if (opts && opts.publicKey) record('get');
			return origGet(opts);
		};
		const origCreate = navigator.credentials.create.bind(navigator.credentials);
		navigator.credentials.create = function(opts) {
			if (opts && opts.publicKey) record('create');
			return origCreate(opts);
		};
	}
})();`

// WebAuthnAttempt is one recorded navigator.credentials call.
type WebAuthnAttempt struct {
	Kind   string `json:"kind"`
	Origin string `json:"origin"`
	TS     int64  `json:"ts"`
}

// initWebAuthnObserver installs the credentials observer on every future
// document in this session's page. The remove func is not retained: the page
// lifetime bounds the observer, matching the other per-session listeners.
func (s *Session) initWebAuthnObserver() {
	if s.page == nil {
		return
	}
	if _, err := s.page.EvalOnNewDocument(webAuthnObserverJS); err != nil {
		// The observer is best-effort detection; failure is logged by the
		// caller's normal diagnostics rather than blocking session open.
		_ = err
	}
}

// WebAuthnAttempts reads the bounded in-page attempt list. Cross-origin
// iframe attempts land on the iframe's own window, so the primary-frame list
// records the page-level flow; iframe URLs are covered by the classifier.
func (s *Session) WebAuthnAttempts() []WebAuthnAttempt {
	s.mu.Lock()
	page := s.page
	s.mu.Unlock()
	if page == nil {
		return nil
	}
	// rod's Eval composes `function() { return (<js>).apply(this, arguments) }`,
	// so the script must be a function expression - not a bare expression and
	// not a `return` statement, both of which fail silently or loudly.
	result, err := page.Eval(`() => JSON.stringify(window.__uiaiWebAuthnAttempts || [])`)
	if err != nil || result == nil {
		return nil
	}
	var attempts []WebAuthnAttempt
	if jerr := json.Unmarshal([]byte(result.Value.Str()), &attempts); jerr != nil {
		return nil
	}
	return attempts
}

// WebAuthnSurfaceHint is the honest, actionable guidance for a detected
// passkey challenge: headless sessions cannot complete hardware/platform
// authenticator flows.
const WebAuthnSurfaceHint = "a hardware authenticator or platform passkey is required; a headless session cannot complete it - complete this challenge through the approved private human or broker path, or use an account recovery fallback"

// FormatWebAuthnHint renders the hint with the observed attempts.
func FormatWebAuthnHint(attempts []WebAuthnAttempt) string {
	if len(attempts) == 0 {
		return ""
	}
	kinds := make([]string, 0, len(attempts))
	for _, a := range attempts {
		kinds = append(kinds, a.Kind)
	}
	return fmt.Sprintf("webauthn challenge observed (%s): %s", strings.Join(kinds, ","), WebAuthnSurfaceHint)
}
