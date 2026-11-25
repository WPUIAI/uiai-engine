package vision

import (
	"fmt"
	"log"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/focusapacket"
	"github.com/go-rod/rod/lib/proto"
)

// maxPopupTargets bounds the per-session popup ring. A page that sprays
// popups must not grow the session unbounded.
const maxPopupTargets = 20

// PopupTarget describes one tab/window the session's page opened
// (window.open or target=_blank). The agent can list, switch to, and close
// these targets through the session's popup API (#301).
type PopupTarget struct {
	TargetID  string `json:"target_id"`
	URL       string `json:"url"`
	Title     string `json:"title,omitempty"`
	CreatedAt string `json:"created_at"`
}

// initPopupsListener tracks targets this session's pages open, via the
// browser-level target-creation event filtered on the opener id. The remove
// func is stored on the session and released on close.
func (s *Session) initPopupsListener() {
	if s.page == nil {
		return
	}
	s.originalTargetID = string(s.page.TargetID)
	browser := s.page.Browser()
	if browser == nil {
		return
	}
	// rod's EachEvent only dispatches events inside its returned wait
	// function; run that loop in a goroutine exactly like the proven
	// diagnostics pattern. Browser.WithCancel breaks browser-level event
	// delivery, so the listener rides the browser connection itself and
	// stays safe after session close via the s.page nil guard.
	go browser.EachEvent(func(e *proto.TargetTargetCreated) {
		info := e.TargetInfo
		s.mu.Lock()
		if s.page == nil {
			s.mu.Unlock()
			return
		}
		current := string(s.page.TargetID)
		s.mu.Unlock()
		if string(info.OpenerID) != current || info.TargetID == "" {
			return
		}
		s.mu.Lock()
		s.popups = append(s.popups, PopupTarget{
			TargetID:  string(info.TargetID),
			URL:       focusapacket.SanitizeURL(info.URL),
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		})
		if len(s.popups) > maxPopupTargets {
			s.popups = s.popups[len(s.popups)-maxPopupTargets:]
		}
		s.mu.Unlock()
		log.Printf("[session] popup opened for %s → %s (target %s)", s.ID, focusapacket.SanitizeURL(info.URL), info.TargetID)
	})()
}

// Popups returns a copy of this session's tracked popup targets, newest last.
func (s *Session) Popups() []PopupTarget {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]PopupTarget, len(s.popups))
	copy(out, s.popups)
	return out
}

// CurrentTargetID reports the target the session's page is currently bound
// to; the original target stays switchable through the same API.
func (s *Session) CurrentTargetID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return string(s.page.TargetID)
}

// ownsTarget reports whether targetID is one of this session's switchable
// targets: the original page or a tracked popup. Foreign session targets are
// never attachable through this API.
func (s *Session) ownsTarget(targetID string) bool {
	if targetID == "" || s.page == nil {
		return false
	}
	if targetID == s.originalTargetID {
		return true
	}
	for _, p := range s.popups {
		if p.TargetID == targetID {
			return true
		}
	}
	return false
}

// OriginalTargetID reports the session's original page target; the agent
// switches back to it with the same popup switch API.
func (s *Session) OriginalTargetID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.originalTargetID
}

// ClosePopup closes one popup target this session opened. Never the original
// page target: use session close for that.
func (s *Session) ClosePopup(targetID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return fmt.Errorf("session closed")
	}
	if targetID == s.originalTargetID {
		return fmt.Errorf("refusing to close the session's original target; close the session instead")
	}
	if !s.ownsTarget(targetID) {
		return fmt.Errorf("target %s is not a popup of this session", targetID)
	}
	if _, err := (proto.TargetCloseTarget{TargetID: proto.TargetTargetID(targetID)}).Call(s.page); err != nil {
		return fmt.Errorf("popup close failed: %w", err)
	}
	kept := s.popups[:0]
	for _, p := range s.popups {
		if p.TargetID != targetID {
			kept = append(kept, p)
		}
	}
	s.popups = kept
	return nil
}

// SwitchToPopup rebinds the session page to one of its own targets — a
// tracked popup, or the original page to switch back. The current page is
// never closed by switching, so the agent can always switch back. Returns a
// fresh capture of the switched page.
func (s *Session) SwitchToPopup(targetID string) (*SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}
	if !s.ownsTarget(targetID) {
		return nil, fmt.Errorf("target %s is not a popup of this session", targetID)
	}
	if string(s.page.TargetID) == targetID {
		return nil, fmt.Errorf("session is already bound to target %s", targetID)
	}
	newPage, err := s.page.Browser().PageFromTarget(proto.TargetTargetID(targetID))
	if err != nil {
		return nil, fmt.Errorf("popup attach failed: %w", err)
	}
	if err := newPage.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width: s.Width, Height: s.Height, DeviceScaleFactor: 1,
	}); err != nil {
		_, _ = proto.TargetCloseTarget{TargetID: proto.TargetTargetID(targetID)}.Call(s.page)
		return nil, fmt.Errorf("popup viewport failed: %w", err)
	}
	s.page = newPage
	start := time.Now()
	if info, ierr := newPage.Info(); ierr == nil {
		s.URL = focusapacket.SanitizeURL(info.URL)
		s.Title = safePageTitle(newPage)
	}
	s.NavCount++
	s.touch()
	return s.snapWithPage(newPage, start)
}
