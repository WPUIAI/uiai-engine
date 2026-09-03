package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/vision"
)

const (
	fpvSharePolicyVersion = "uiai.fpv_share_policy.v2"
	fpvDefaultTTLMinutes  = 15
	fpvMaximumTTLMinutes  = 60
	fpvDefaultMaxViews    = 25
	fpvMaximumMaxViews    = 100
)

var errFPVSensitiveOrigin = errors.New("public FPV sharing denied for sensitive origin")
var errFPVControlsDenied = errors.New("control shares require governed confirmation")
var errFPVPolicyInvalid = errors.New("FPV share policy invalid")
var errFPVShareUnavailable = errors.New("FPV share unavailable")
var errFPVViewLimit = errors.New("FPV share view limit reached")
var errFPVRegistryPersistence = errors.New("FPV share registry persistence failed")

func fpvShareOrigin(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return "", errFPVSensitiveOrigin
	}
	host := strings.ToLower(parsed.Hostname())
	path := strings.ToLower(parsed.EscapedPath())
	for _, label := range strings.Split(host, ".") {
		for _, sensitive := range []string{"accounts", "auth", "login", "pay", "checkout", "billing", "privacy", "health", "patient", "medical", "mychart", "wallet"} {
			if label == sensitive || strings.HasPrefix(label, sensitive+"-") || strings.HasPrefix(label, sensitive+"portal") {
				return "", errFPVSensitiveOrigin
			}
		}
	}
	if strings.HasPrefix(host, "dashboard.stripe.") || strings.HasPrefix(host, "paypal.") {
		return "", errFPVSensitiveOrigin
	}
	for _, segment := range strings.Split(path, "/") {
		switch segment {
		case "login", "signin", "sign-in", "oauth", "authorize", "authorization", "auth", "account", "privacy", "payment", "payments", "billing", "checkout", "health", "patient", "medical", "bank", "wallet":
			return "", errFPVSensitiveOrigin
		}
	}
	for key := range parsed.Query() {
		switch strings.ToLower(key) {
		case "code", "token", "access_token", "refresh_token", "session", "password", "otp", "client_secret":
			return "", errFPVSensitiveOrigin
		}
	}
	return strings.ToLower(parsed.Scheme) + "://" + strings.ToLower(parsed.Host), nil
}

func fpvSessionForShare(sm *vision.SessionManager, entry *fpvShare) (*vision.Session, bool) {
	if sm == nil || entry == nil || entry.Revoked || entry.PolicyVersion != fpvSharePolicyVersion {
		return nil, false
	}
	session, found := sm.Get(entry.SessionID)
	if !found {
		return nil, false
	}
	origin, err := fpvShareOrigin(session.URL)
	if err != nil || origin != entry.Origin {
		return nil, false
	}
	return session, true
}

func fpvConsumeView(token string, sm *vision.SessionManager) (*fpvShare, error) {
	return fpvConsumeViewIf(token, func(entry *fpvShare) bool {
		_, ok := fpvSessionForShare(sm, entry)
		return ok
	})
}

func fpvConsumeViewIf(token string, sessionAvailable func(*fpvShare) bool) (*fpvShare, error) {
	fpvShareMu.Lock()
	value, found := fpvShares.Load(token)
	if !found {
		fpvShareMu.Unlock()
		return nil, errFPVShareUnavailable
	}
	entry := value.(*fpvShare)
	if !fpvShareUsable(entry, time.Now().UTC()) || entry.Views >= fpvAllowedViews(entry) {
		limited := fpvShareUsable(entry, time.Now().UTC()) && entry.Views >= fpvAllowedViews(entry)
		fpvShareMu.Unlock()
		if limited {
			return nil, errFPVViewLimit
		}
		return nil, errFPVShareUnavailable
	}
	if sessionAvailable == nil || !sessionAvailable(entry) {
		updated := cloneFPVShare(entry)
		updated.Revoked = true
		fpvShares.Store(updated.Token, updated)
		fpvShareMu.Unlock()
		if err := fpvSaveRegistry(); err != nil {
			return nil, errFPVRegistryPersistence
		}
		return nil, errFPVShareUnavailable
	}
	updated := cloneFPVShare(entry)
	updated.Views++
	fpvShares.Store(updated.Token, updated)
	fpvShareMu.Unlock()
	if err := fpvSaveRegistry(); err != nil {
		fpvShareMu.Lock()
		updated.Revoked = true
		fpvShares.Store(updated.Token, updated)
		fpvShareMu.Unlock()
		return nil, errFPVRegistryPersistence
	}
	return cloneFPVShare(updated), nil
}

func fpvRevokeToken(token string) (*fpvShare, bool, error) {
	fpvShareMu.Lock()
	value, found := fpvShares.Load(token)
	if !found {
		fpvShareMu.Unlock()
		return nil, false, nil
	}
	updated := cloneFPVShare(value.(*fpvShare))
	updated.Revoked = true
	fpvShares.Store(updated.Token, updated)
	fpvShareMu.Unlock()
	if err := fpvSaveRegistry(); err != nil {
		return cloneFPVShare(updated), true, errFPVRegistryPersistence
	}
	return cloneFPVShare(updated), true, nil
}

func fpvAppendShareAudit(token string, log fpvAuditLog) (*fpvShare, bool) {
	fpvShareMu.Lock()
	defer fpvShareMu.Unlock()
	value, found := fpvShares.Load(token)
	if !found {
		return nil, false
	}
	updated := cloneFPVShare(value.(*fpvShare))
	updated.Audit = append(updated.Audit, log)
	if len(updated.Audit) > 100 {
		updated.Audit = updated.Audit[len(updated.Audit)-100:]
	}
	fpvShares.Store(updated.Token, updated)
	return cloneFPVShare(updated), true
}

func fpvRevokeSessionShares(sessionID string) (int, error) {
	fpvShareMu.Lock()
	revoked := 0
	fpvShares.Range(func(_, value any) bool {
		entry, ok := value.(*fpvShare)
		if ok && entry != nil && entry.SessionID == sessionID && !entry.Revoked {
			updated := cloneFPVShare(entry)
			updated.Revoked = true
			fpvShares.Store(updated.Token, updated)
			revoked++
		}
		return true
	})
	fpvShareMu.Unlock()
	if revoked > 0 {
		if err := fpvSaveRegistry(); err != nil {
			return revoked, errFPVRegistryPersistence
		}
	}
	return revoked, nil
}

func fpvShareUsable(entry *fpvShare, now time.Time) bool {
	return entry != nil && entry.PolicyVersion == fpvSharePolicyVersion && entry.Origin != "" && entry.ConsentRef != "" && !entry.Controls &&
		entry.MaxViews > 0 && entry.MaxViews <= fpvMaximumMaxViews && now.Before(entry.ExpiresAt) && !entry.Revoked
}

func cloneFPVShare(entry *fpvShare) *fpvShare {
	if entry == nil {
		return nil
	}
	clone := *entry
	clone.Audit = append([]fpvAuditLog(nil), entry.Audit...)
	return &clone
}

func fpvTokenRef(token string) string {
	digest := sha256.Sum256([]byte(token))
	return "fpv-share:sha256:" + hex.EncodeToString(digest[:])
}
