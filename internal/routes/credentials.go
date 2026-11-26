package routes

import (
	"net/http"

	"github.com/WPUIAI/uiai-engine/internal/ai"
	"github.com/go-chi/chi/v5"
)

// CredentialInvalidator is the provider surface the revocation entry needs.
// Narrow interface keeps the route testable without a full provider.
type CredentialInvalidator interface {
	InvalidateCredentialCache()
}

// MountCredentialRevocationRoutes registers the broker revocation entry
// (#86): POST /api/admin/credentials/invalidate fences the cached
// credential snapshot immediately so a revoked key stops being used within
// the broker's SLA instead of at TTL expiry. The global authenticator
// middleware gates this path (it is not in any loopback-exempt list); the
// broker calls it from its revoke webhook or SSE handler.
func MountCredentialRevocationRoutes(r chi.Router, provider CredentialInvalidator) {
	r.Post("/credentials/invalidate", func(w http.ResponseWriter, _ *http.Request) {
		if provider == nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "credential cache unavailable", "code": "credential_revocation_unavailable"})
			return
		}
		provider.InvalidateCredentialCache()
		writeJSON(w, http.StatusOK, map[string]any{
			"invalidated": true,
			"code":        "credential_cache_invalidated",
			"posture":     "in-flight refreshes are fenced; the next use re-authenticates against the custody source",
		})
	})
}

var _ CredentialInvalidator = (*ai.Provider)(nil)
