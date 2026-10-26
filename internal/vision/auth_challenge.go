package vision

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/events"
	"github.com/google/uuid"
)

const (
	LeaseActive       = "ACTIVE"
	LeaseAwaitingAuth = "AWAITING_AUTH"
)

type AuthChallenge struct {
	Class       string    `json:"class"`
	URLHint     string    `json:"url_hint,omitempty"`
	SessionID   string    `json:"session_id"`
	ActorRef    string    `json:"actor,omitempty"`
	ResumeToken string    `json:"resume_token"`
	DetectedAt  time.Time `json:"detected_at"`
}

func (s *Session) LeaseState() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.authChallenge != nil {
		return LeaseAwaitingAuth
	}
	return LeaseActive
}

func (s *Session) CurrentAuthChallenge() *AuthChallenge {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.authChallenge == nil {
		return nil
	}
	copy := *s.authChallenge
	return &copy
}

// DetectAuthChallenge inspects only page metadata and visible challenge markers.
// It never reads credentials or challenge values.
func (s *Session) DetectAuthChallenge() (*AuthChallenge, error) {
	return s.DetectAuthChallengeContext(context.Background())
}

func (s *Session) DetectAuthChallengeContext(caller context.Context) (*AuthChallenge, error) {
	if err := caller.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	page := s.page
	s.mu.Unlock()
	if page == nil {
		return nil, fmt.Errorf("session page unavailable")
	}
	ctx, cancel := sessionActionContext(page.GetContext(), caller, sessionInputTimeout)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	result, err := page.Context(ctx).Eval(`() => {
		const url = location.href || '';
		const text = (document.body?.innerText || '').toLowerCase();
		const has = selector => Boolean(document.querySelector(selector));
		let cls = '';
		if (has('input[type="password"], form[action*="login" i]')) cls = 'credential_form';
		else if (has('[data-sitekey], [name*="captcha" i], [id*="captcha" i], [id*="turnstile" i], iframe[src*="recaptcha" i]')) cls = 'captcha';
		else if (text.includes('webauthn') || text.includes('security key') || text.includes('passkey')) cls = 'webauthn';
		else if (text.includes('3-d secure') || text.includes('3ds') || text.includes('strong customer authentication')) cls = 'payment_sca';
		else if (/[?&/](oauth|authorize|authorization|login)([/?=&]|$)/i.test(url)) cls = 'oidc_redirect';
		return JSON.stringify({class: cls, url_hint: url.split('#')[0].split('?')[0]});
	}`)
	if err != nil {
		return nil, err
	}
	var detected struct {
		Class   string `json:"class"`
		URLHint string `json:"url_hint"`
	}
	if err := json.Unmarshal([]byte(result.Value.Str()), &detected); err != nil {
		return nil, err
	}
	if strings.TrimSpace(detected.Class) == "" {
		return nil, nil
	}
	return s.raiseAuthChallenge(detected.Class, detected.URLHint), nil
}

func (s *Session) raiseAuthChallenge(class, urlHint string) *AuthChallenge {
	urlHint = redactURL(urlHint)
	s.mu.Lock()
	if s.authChallenge != nil && s.authChallenge.Class == class {
		challenge := *s.authChallenge
		s.mu.Unlock()
		return &challenge
	}
	challenge := &AuthChallenge{Class: class, URLHint: urlHint, SessionID: s.ID, ActorRef: s.ActorRef, ResumeToken: "resume_" + uuid.NewString(), DetectedAt: time.Now().UTC()}
	s.authChallenge = challenge
	if s.timer != nil {
		s.timer.Stop()
	}
	actor := s.ActorRef
	copy := *challenge
	s.mu.Unlock()
	events.Emit("auth_challenge_required", map[string]any{"session_id": s.ID, "actor": actor}, nil, map[string]any{
		"type": "auth_challenge_required", "class": class, "url_hint": urlHint, "session_id": s.ID, "actor": actor, "resume_token": challenge.ResumeToken,
	})
	return &copy
}

func (s *Session) ResolveAuthChallenge(resumeToken string) error {
	s.mu.Lock()
	if s.authChallenge == nil {
		s.mu.Unlock()
		return nil
	}
	if resumeToken == "" || resumeToken != s.authChallenge.ResumeToken {
		s.mu.Unlock()
		return fmt.Errorf("invalid auth challenge resume token")
	}
	challenge := *s.authChallenge
	s.authChallenge = nil
	s.LastUsed = time.Now()
	if s.timer != nil {
		s.timer.Reset(SessionTTL)
	}
	actor := s.ActorRef
	s.mu.Unlock()
	events.Emit("auth_challenge_resolved", map[string]any{"session_id": s.ID, "actor": actor}, nil, map[string]any{"type": "auth_challenge_resolved", "class": challenge.Class, "session_id": s.ID, "actor": actor})
	return nil
}

func (s *Session) CancelAuthChallenge() *AuthChallenge {
	s.mu.Lock()
	if s.authChallenge == nil {
		s.mu.Unlock()
		return nil
	}
	challenge := *s.authChallenge
	s.authChallenge = nil
	actor := s.ActorRef
	s.mu.Unlock()
	events.Emit("auth_challenge_aborted", map[string]any{"session_id": s.ID, "actor": actor}, nil, map[string]any{"type": "auth_challenge_aborted", "class": challenge.Class, "session_id": s.ID, "actor": actor})
	return &challenge
}
