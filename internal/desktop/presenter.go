package desktop

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/desktopcontract"
)

var (
	ErrPresentationNotFound = errors.New("presentation not found")
	ErrIdempotencyConflict  = errors.New("idempotency key already represents a different presentation request")
)

type SessionLookup func(string) bool

type Acknowledgement struct {
	Status            string `json:"status"`
	CockpitInstanceID string `json:"cockpit_instance_id,omitempty"`
	ReasonCode        string `json:"reason_code,omitempty"`
}

type DesktopPresenter interface {
	EnsureVisible(context.Context, string, desktopcontract.DesktopPresentationRequest) (desktopcontract.DesktopPresentationReceipt, error)
	Status(context.Context, string) (desktopcontract.DesktopPresentationReceipt, error)
	Acknowledge(context.Context, string, Acknowledgement) (desktopcontract.DesktopPresentationReceipt, error)
}

const maxPresentationReceipts = 1024

type presentationEntry struct {
	receipt           desktopcontract.DesktopPresentationReceipt
	requestKey        string
	idempotencyMapKey string
	expiresAt         time.Time
}

type Presenter struct {
	mu            sync.Mutex
	launcher      Launcher
	sessionExists SessionLookup
	now           func() time.Time
	byID          map[string]*presentationEntry
	byRequest     map[string]*presentationEntry
}

func NewPresenter(sessionExists SessionLookup, launcher Launcher) *Presenter {
	if launcher == nil {
		launcher = NewPlatformLauncher()
	}
	return &Presenter{
		launcher: launcher, sessionExists: sessionExists, now: time.Now,
		byID: make(map[string]*presentationEntry), byRequest: make(map[string]*presentationEntry),
	}
}

func (p *Presenter) EnsureVisible(ctx context.Context, sessionID string, req desktopcontract.DesktopPresentationRequest) (desktopcontract.DesktopPresentationReceipt, error) {
	if err := desktopcontract.ValidateOpaqueRef(sessionID); err != nil {
		return desktopcontract.DesktopPresentationReceipt{}, fmt.Errorf("session_id: %w", err)
	}
	if err := desktopcontract.ValidatePresentationRequest(req); err != nil {
		return desktopcontract.DesktopPresentationReceipt{}, err
	}
	requestKey := sessionID + ":" + req.IdempotencyKey
	requestFingerprint := fmt.Sprintf("%s|%s|%s|%t|%d|%s", sessionID, req.Mode, req.Reason, req.Focus, req.ExpiresInMS, req.RequestedBy.ClientType)

	p.mu.Lock()
	p.pruneLocked()
	if existing := p.byRequest[requestKey]; existing != nil {
		if existing.requestKey != requestFingerprint {
			p.mu.Unlock()
			return desktopcontract.DesktopPresentationReceipt{}, ErrIdempotencyConflict
		}
		receipt := p.refreshLocked(existing)
		p.mu.Unlock()
		return receipt, nil
	}

	now := p.now().UTC()
	expiresAt := now.Add(time.Duration(req.ExpiresInMS) * time.Millisecond)
	receipt := desktopcontract.DesktopPresentationReceipt{
		Schema:         desktopcontract.SchemaDesktopPresentationReceiptV1,
		PresentationID: opaqueID("presentation"), SessionID: sessionID,
		Status: "requested", HandoffRef: opaqueID("handoff"),
		CreatedAt: now.Format(time.RFC3339Nano), ExpiresAt: expiresAt.Format(time.RFC3339Nano),
	}
	entry := &presentationEntry{receipt: receipt, requestKey: requestFingerprint, idempotencyMapKey: requestKey, expiresAt: expiresAt}
	p.byID[receipt.PresentationID] = entry
	p.byRequest[requestKey] = entry

	if req.ScopeRef != nil && req.ScopeRef.AuthorityState != "verified" {
		entry.receipt.Status = "blocked_scope"
		entry.receipt.ReasonCode = "scope_not_verified"
		receipt = entry.receipt
		p.mu.Unlock()
		return receipt, nil
	}
	if p.sessionExists == nil || !p.sessionExists(sessionID) {
		entry.receipt.Status = "session_missing"
		entry.receipt.ReasonCode = "canonical_session_missing"
		receipt = entry.receipt
		p.mu.Unlock()
		return receipt, nil
	}
	entry.receipt.Status = "launching"
	receipt = entry.receipt
	p.mu.Unlock()

	activationURL := presentationURL(sessionID, entry.receipt.HandoffRef, req)
	if err := p.launcher.Open(ctx, activationURL); err != nil {
		p.mu.Lock()
		if errors.Is(err, ErrDesktopUnavailable) {
			entry.receipt.Status = "desktop_unavailable"
			entry.receipt.ReasonCode = "fpv_recovery_available"
		} else {
			entry.receipt.Status = "failed"
			entry.receipt.ReasonCode = "cockpit_launch_failed"
		}
		receipt = entry.receipt
		p.mu.Unlock()
	}
	return receipt, nil
}

func (p *Presenter) Status(_ context.Context, presentationID string) (desktopcontract.DesktopPresentationReceipt, error) {
	if err := desktopcontract.ValidateOpaqueRef(presentationID); err != nil {
		return desktopcontract.DesktopPresentationReceipt{}, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	entry := p.byID[presentationID]
	if entry == nil {
		return desktopcontract.DesktopPresentationReceipt{}, ErrPresentationNotFound
	}
	return p.refreshLocked(entry), nil
}

func (p *Presenter) Acknowledge(_ context.Context, presentationID string, ack Acknowledgement) (desktopcontract.DesktopPresentationReceipt, error) {
	if err := desktopcontract.ValidateOpaqueRef(presentationID); err != nil {
		return desktopcontract.DesktopPresentationReceipt{}, err
	}
	if ack.Status != "attaching" && ack.Status != "visible" && ack.Status != "focused" && ack.Status != "attach_failed" && ack.Status != "incompatible" {
		return desktopcontract.DesktopPresentationReceipt{}, fmt.Errorf("unsupported acknowledgement status %q", ack.Status)
	}
	if ack.CockpitInstanceID != "" {
		if err := desktopcontract.ValidateOpaqueRef(ack.CockpitInstanceID); err != nil {
			return desktopcontract.DesktopPresentationReceipt{}, fmt.Errorf("cockpit_instance_id: %w", err)
		}
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	entry := p.byID[presentationID]
	if entry == nil {
		return desktopcontract.DesktopPresentationReceipt{}, ErrPresentationNotFound
	}
	if p.now().After(entry.expiresAt) {
		return p.refreshLocked(entry), nil
	}
	entry.receipt.Status = ack.Status
	entry.receipt.CockpitInstanceID = ack.CockpitInstanceID
	entry.receipt.ReasonCode = ack.ReasonCode
	return entry.receipt, nil
}

func (p *Presenter) pruneLocked() {
	now := p.now()
	for id, entry := range p.byID {
		if now.After(entry.expiresAt.Add(time.Minute)) {
			delete(p.byID, id)
			delete(p.byRequest, entry.idempotencyMapKey)
		}
	}
	for len(p.byID) >= maxPresentationReceipts {
		var oldestID string
		var oldest *presentationEntry
		for id, entry := range p.byID {
			if oldest == nil || entry.receipt.CreatedAt < oldest.receipt.CreatedAt {
				oldestID, oldest = id, entry
			}
		}
		if oldest == nil {
			break
		}
		delete(p.byID, oldestID)
		delete(p.byRequest, oldest.idempotencyMapKey)
	}
}

func (p *Presenter) refreshLocked(entry *presentationEntry) desktopcontract.DesktopPresentationReceipt {
	if p.now().After(entry.expiresAt) {
		switch entry.receipt.Status {
		case "requested", "resolving_session", "resolving_cockpit", "launching", "attaching":
			entry.receipt.Status = "expired"
			entry.receipt.ReasonCode = "presentation_expired"
		}
	}
	return entry.receipt
}

func opaqueID(prefix string) string {
	var value [16]byte
	if _, err := rand.Read(value[:]); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(value[:])
}

func presentationURL(sessionID, handoffRef string, req desktopcontract.DesktopPresentationRequest) string {
	query := url.Values{}
	query.Set("handoff", handoffRef)
	query.Set("mode", req.Mode)
	if req.Focus {
		query.Set("focus", "1")
	}
	return "cockpit://live/session/" + url.PathEscape(sessionID) + "?" + query.Encode()
}
