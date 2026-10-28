package routes

import (
	"net/http"

	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

// A public view is not permission to observe a later authentication challenge.
// Revocation is permanent even if that challenge is subsequently resolved.
func fpvRevokePrivateShare(token string) {
	if _, changed := fpvUpdateShare(token, func(next *fpvShare) bool { next.Revoked = true; return true }); changed {
		fpvSaveRegistry()
	}
}

// Reuse the existing session challenge authority; streaming must not outlive a
// recognized transition simply because its HTTP request was admitted earlier.
func fpvPublicSessionAllowed(sm *vision.SessionManager, entry *fpvShare, sess *vision.Session) bool {
	if sm == nil || sess == nil || entry == nil {
		return false
	}
	current, valid := fpvEntry(entry.Token)
	if !valid || current.SessionID != entry.SessionID {
		return false
	}
	active, exists := sm.Get(entry.SessionID)
	if !exists || active != sess {
		return false
	}
	if sess.CurrentAuthChallenge() != nil {
		fpvRevokePrivateShare(entry.Token)
		return false
	}
	return true
}

func fpvPublicPrivacyGuard(sm *vision.SessionManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			token := chi.URLParam(req, "token")
			if token == "" {
				next.ServeHTTP(w, req)
				return
			} // Static viewer assets carry no session content.
			entry, valid := fpvEntry(token)
			if !valid {
				next.ServeHTTP(w, req)
				return
			} // Preserve route-specific expired/revoked responses.
			if sm == nil {
				writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "session manager unavailable"})
				return
			}
			sess, exists := sm.Get(entry.SessionID)
			if !exists {
				writeJSON(w, http.StatusGone, map[string]string{"error": "session closed"})
				return
			}
			if sess.CurrentAuthChallenge() == nil {
				if _, err := sess.DetectAuthChallengeContext(req.Context()); err != nil {
					writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "session privacy could not be verified", "error_class": "fpv_privacy_check_failed"})
					return
				}
			}
			if !fpvPublicSessionAllowed(sm, entry, sess) {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "public view is no longer authorized", "error_class": "fpv_private_session"})
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}
