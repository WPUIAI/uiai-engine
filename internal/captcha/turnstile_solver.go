package captcha

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/vision"
)

// #70: Turnstile managed challenges reach the engine but had no solve path
// (the dispatch answered "unsupported captcha type: turnstile"). The honest
// strategy: Turnstile decides server-side from the browser environment; the
// solver clicks the widget once (the native click lands inside the
// cross-origin challenge iframe), then waits a bounded time for the token to
// appear in the parent page or for the interstitial to clear. Unsolved is
// reported unsolved with the observed state — no fabricated success.

const (
	defaultTurnstileWaitSeconds = 20
	turnstilePollInterval       = 800 * time.Millisecond
)

// SolveTurnstile attempts a Cloudflare Turnstile challenge in the session.
func (s *Solver) SolveTurnstile(ctx context.Context, sess *vision.Session, cfg SolveConfig) *SolveResponse {
	start := time.Now()
	resp := &SolveResponse{Type: "turnstile", Method: "turnstile_token_wait"}

	probe, _, perr := sess.Eval(`return JSON.stringify({
		widget: !!document.querySelector('.cf-turnstile, [class*="cf-turnstile"]'),
		interstitial: (document.body?.innerText || '').toLowerCase().includes('just a moment') || (document.title || '').toLowerCase().includes('just a moment'),
	})`)
	widget, interstitial := false, false
	if perr == nil {
		var detected struct {
			Widget       bool `json:"widget"`
			Interstitial bool `json:"interstitial"`
		}
		if jerr := json.Unmarshal([]byte(probe), &detected); jerr == nil {
			widget, interstitial = detected.Widget, detected.Interstitial
		}
	}
	if !widget && !interstitial {
		resp.Error = "no turnstile challenge detected on the page"
		resp.DurationMs = time.Since(start).Milliseconds()
		return resp
	}
	resp.Attempts = 1

	// One click attempt at the widget container: the native click dispatches
	// at page-absolute coordinates and reaches the cross-origin challenge
	// iframe. Widgets that auto-run are unaffected; failure to click is not
	// fatal because managed challenges frequently complete without it.
	if widget {
		_, _ = sess.ClickContext(ctx, ".cf-turnstile, [class*=\"cf-turnstile\"]")
		resp.Attempts = 2
	}

	wait := defaultTurnstileWaitSeconds
	if cfg.TurnstileWaitSeconds > 0 {
		wait = cfg.TurnstileWaitSeconds
	}
	deadline := time.Now().Add(time.Duration(wait) * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			resp.Error = "solve context cancelled while waiting for the turnstile token"
			resp.DurationMs = time.Since(start).Milliseconds()
			return resp
		default:
		}
		state, _, serr := sess.Eval(`return JSON.stringify({
			token: (document.querySelector('input[name="cf-turnstile-response"]') || {}).value || '',
			stillInterstitial: (document.body?.innerText || '').toLowerCase().includes('just a moment'),
		})`)
		if serr == nil {
			var observed struct {
				Token             string `json:"token"`
				StillInterstitial bool   `json:"stillInterstitial"`
			}
			if jerr := json.Unmarshal([]byte(state), &observed); jerr == nil {
				if strings.TrimSpace(observed.Token) != "" {
					// Completion is the signal. The token value itself never
					// leaves the page: the engine's read redaction withholds
					// it from model-visible surfaces, and the page's own form
					// submits it. The solver reports solved without echoing a
					// capability, including when redaction already replaced
					// the value with a withheld marker.
					resp.Solved = true
					resp.Token = ""
					resp.DurationMs = time.Since(start).Milliseconds()
					return resp
				}
				if interstitial && !observed.StillInterstitial {
					// The challenge interstitial cleared without a parent-page
					// token: navigation or reload completed the check.
					resp.Solved = true
					resp.Method = "turnstile_interstitial_cleared"
					resp.DurationMs = time.Since(start).Milliseconds()
					return resp
				}
			}
		}
		time.Sleep(turnstilePollInterval)
	}
	resp.Error = "turnstile token never appeared within the bounded wait; the browser environment was not accepted"
	resp.DurationMs = time.Since(start).Milliseconds()
	return resp
}
