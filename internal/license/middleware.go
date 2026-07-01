// Package license — chi middleware helper for gating entire route subtrees per spec §6.6.
//
// Usage:
//
//	import "github.com/WPUIAI/uiai-engine/internal/license"
//
//	r.Route("/api/critique", func(r chi.Router) {
//	    r.Use(license.RequireFeatureMiddleware(license.FeatureCritique))
//	    routes.MountCritiqueReal(r, ...)
//	})
//
// On fail: writes the spec §6.5 / §11 JSON envelope and returns 402.
// On pass: handler chain continues.
package license

import (
	"net/http"
)

// RequireFeatureMiddleware returns a chi middleware that calls RequireFeature and
// short-circuits the request when the feature is gated.
func RequireFeatureMiddleware(feature string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !RequireFeature(w, r, feature) {
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
