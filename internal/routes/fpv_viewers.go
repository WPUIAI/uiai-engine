package routes

import (
	"net/http"
	"sync"
	"time"
)

// Share values in fpvShares are immutable after Store. All changes use this
// owner so concurrent arrivals/control audit/revocation cannot overwrite each other.
var fpvShareUpdateMu sync.Mutex

func fpvUpdateShare(token string, change func(*fpvShare) bool) (*fpvShare, bool) {
	fpvShareUpdateMu.Lock()
	defer fpvShareUpdateMu.Unlock()
	value, ok := fpvShares.Load(token)
	if !ok {
		return nil, false
	}
	old := value.(*fpvShare)
	if old.Revoked || !time.Now().Before(old.ExpiresAt) {
		return nil, false
	}
	next := *old
	next.Audit = append([]fpvAuditLog(nil), old.Audit...)
	if !change(&next) {
		return nil, false
	}
	fpvShares.Store(token, &next)
	return &next, true
}

// Cookies deduplicate ordinary refresh/status polls, not authenticated people.
// Telemetry is advisory: a viewer event never proves rendered pixels or consent.
func fpvRecordViewer(w http.ResponseWriter, r *http.Request, token string, page bool) (*fpvShare, bool) {
	cookie, err := r.Cookie("uiai_fpv_seen")
	joined := err != nil || cookie.Value != "1"
	now := time.Now().UTC()
	entry, ok := fpvUpdateShare(token, func(next *fpvShare) bool {
		if page {
			if next.Views >= fpvAllowedViews(next) {
				return false
			}
			next.Views++
		}
		if joined {
			next.ViewerJoins++
			if next.ViewerSince.IsZero() {
				next.ViewerSince = now
			}
		}
		next.LastViewerAt = now
		return true
	})
	if !ok {
		return nil, false
	}
	if joined {
		http.SetCookie(w, &http.Cookie{Name: "uiai_fpv_seen", Value: "1", Path: "/m/" + token, Expires: entry.ExpiresAt, HttpOnly: true, Secure: r.TLS != nil, SameSite: http.SameSiteStrictMode})
		fpvRecordAuditEvent(token, entry.SessionID, fpvAuditLog{TS: now, Action: "fpv_viewer_joined", OK: true, Meta: map[string]any{"surface": "page", "viewer_join_count": entry.ViewerJoins, "observation": "request_accepted_not_render_proof"}})
	}
	if joined || page {
		fpvSaveRegistry()
	}
	return entry, true
}

func fpvViewerState(entry *fpvShare) map[string]any {
	return map[string]any{"viewer_since": entry.ViewerSince, "last_viewer_at": entry.LastViewerAt, "viewer_join_count": entry.ViewerJoins, "viewer_count": fpvViewerCount(entry.Token), "count_basis": "active_stream_connections", "identity_basis": "share_path_cookie_not_authenticated_person"}
}

func fpvRecordStreamViewer(token, sessionID, action string, count int) {
	if sessionID == "" {
		return
	}
	now := time.Now().UTC()
	if action == "fpv_viewer_joined" {
		_, _ = fpvUpdateShare(token, func(next *fpvShare) bool {
			if next.ViewerSince.IsZero() {
				next.ViewerSince = now
			}
			next.LastViewerAt = now
			return true
		})
	}
	fpvRecordAuditEvent(token, sessionID, fpvAuditLog{TS: now, Action: action, OK: true, Meta: map[string]any{"surface": "stream", "viewer_count": count}})
}
