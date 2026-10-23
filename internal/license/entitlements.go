// Package license implements the UIAI Engine feature entitlement helper per spec §6.5.
//
// The helper exposes a single source of truth for "is this feature enabled for the
// caller" decisions, with three entry points:
//
//   - FromIdentity(id):  pure struct — no I/O — derive Entitlements from an auth.Identity.
//   - IsEvaluation(id, r):  is the caller in evaluation mode (loopback + no license)?
//   - FeatureEnabled(id, feature, r):  is the named feature enabled for the caller?
//   - RequireFeature(w, r, feature):  short-circuit guard; on failure, writes the
//     spec §6.5 JSON error envelope (license_required) and returns false. On success
//     returns true and the handler continues.
//
// Status code policy (spec §6.5):
//
//	401 Unauthorized — no auth identity at all (loopback callers may bypass)
//	402 Payment Required — license_required (recommended for missing/invalid licenses)
//	403 Forbidden — auth identity present but feature not enabled for the tier
//
// Routes should call RequireFeature early and bail out on false. The JSON envelope
// matches spec §11 shape (license_required, feature, message, purchase_url, docs_url).
package license

import (
	"encoding/json"
	"net"
	"net/http"
	"strings"

	"github.com/WPUIAI/uiai-engine/internal/auth"
)

// Entitlements describes what the caller is allowed to do, derived from auth.Identity.
type Entitlements struct {
	// Mode is "evaluation" (no license / loopback eval) or "licensed" (real key or token).
	Mode string `json:"mode"`
	// Tier is the license tier (e.g. "operator", "founders-forge", "team", "commercial").
	// Empty when in evaluation mode.
	Tier string `json:"tier,omitempty"`
	// CommercialUse indicates whether the current entitlements permit commercial use.
	CommercialUse bool `json:"commercial_use"`
	// Features is the set of feature keys enabled by the current license.
	Features map[string]bool `json:"features"`
	// Product is "uiai-engine" or empty.
	Product string `json:"product,omitempty"`
}

// Feature keys — mirror the spec §6.6 license-gated route list and §6.5 eval-allowed list.
// These strings are the public contract for what a license may unlock.
const (
	FeatureEvalAllowed = "eval_allowed" // sentinel — loopback eval grants this

	// Spec §6.6 — license-gated routes
	FeatureRemoteAPIAccess = "remote_api_access"
	FeatureCritique        = "critique"
	FeatureUIReverse       = "ui_reverse"
	FeatureStyleEnhance    = "style_enhance"
	FeatureLayoutCompare   = "layout_compare"
	FeatureSectionDetect   = "section_detect"
	FeatureCopilot         = "copilot"
	FeatureMediaAccess     = "media_access"
	FeatureShareAccess     = "share_access"
	FeatureReferenceAccess = "reference_access"
	FeatureDesignSystem    = "design_system"
	FeatureContentMap      = "content_map"
	FeatureBlockRecipes    = "block_recipes"
	FeatureComparison      = "comparison"
	FeatureMigration       = "migration"

	// UIAI-ENGINE-010 premium pillars (C-010-29 licensing closure)
	FeaturePersonaStealth    = "persona_stealth"     // C-010-14/30 — identity personas + stealth hardening
	FeatureAdaptiveRotation  = "adaptive_rotation"   // C-010-31 — rotate-on-flag intelligence
	FeatureSolverCoveragePro = "solver_coverage_pro" // C-010-32 — extended provider/type matrix
	FeatureConsensusReads    = "consensus_reads"     // C-010-18 — N>1 persona diff/trust
	FeatureMeshWorkers       = "mesh_workers"        // C-010-19/25 — remote tailnet/container workers
	FeatureWebStateContinuity = "web_state_continuity" // C-010-07/23 — checkpoints beyond single session
	FeatureTimeTravelExport  = "time_travel_export"  // C-010-28 — moment-bundle export

	// Spec §6.5 eval-allowed (loopback only) features
	FeatureLocalSession    = "local_session"
	FeatureLocalSearch     = "local_search"
	FeatureLocalMarkdown   = "local_markdown"
	FeatureLocalResearch   = "local_research_packet"
	FeatureLocalScreenshot = "local_screenshot"
)

// EvalAllowedFeatures is the set of feature keys that loopback callers can use without a
// license (spec §6.3 Evaluation Mode). Non-loopback callers always need a license.
var EvalAllowedFeatures = map[string]bool{
	FeatureLocalSession:    true,
	FeatureLocalSearch:     true,
	FeatureLocalMarkdown:   true,
	FeatureLocalResearch:   true,
	FeatureLocalScreenshot: true,
}

// LicenseRequiredError matches spec §6.5 / §11 license_required JSON envelope.
type LicenseRequiredError struct {
	Feature     string `json:"feature"`
	Error       string `json:"error"`
	Message     string `json:"message"`
	PurchaseURL string `json:"purchase_url"`
	DocsURL     string `json:"docs_url"`
}

const (
	purchaseURLBase = "https://engine.focusa.dev"
	docsURLBase     = "https://install.focusa.dev/license"
)

// FromIdentity derives Entitlements from an auth.Identity. Pure function — no I/O, no
// request inspection. Use FeatureEnabled for the request-aware check.
func FromIdentity(id *auth.Identity) Entitlements {
	if id == nil {
		// No auth identity at all. Caller is presumed to be loopback / evaluation.
		return Entitlements{Mode: "evaluation", Features: copyBoolMap(EvalAllowedFeatures)}
	}
	tier := strings.ToLower(strings.TrimSpace(id.Tier))
	if tier == "" || strings.EqualFold(tier, "evaluation") {
		return Entitlements{Mode: "evaluation", Features: copyBoolMap(EvalAllowedFeatures)}
	}
	// Map tier to default feature set. Real licenses carry explicit feature lists; this
	// mapping is a best-effort default so an identity without explicit features still gets
	// the tier-appropriate entitlements.
	features := tierFeatures(tier)
	return Entitlements{
		Mode:          "licensed",
		Tier:          tier,
		CommercialUse: commercialUseForTier(tier),
		Features:      features,
		Product:       "uiai-engine",
	}
}

func tierFeatures(tier string) map[string]bool {
	// Spec 152 / ENDPOINT_AUTH_MATRIX §3: tier-derived grants are legacy.
	// All known licensed tiers currently receive the same broad set; explicit
	// authority-issued claims (product/features/limits) will replace this.
	// Unknown tiers fail closed (no features) per spec 152e.
	known := map[string]bool{"operator": true, "founders-forge": true, "team": true, "commercial": true, "enterprise": true, "internal": false, "pro": false}
	if _, ok := known[tier]; !ok {
		return map[string]bool{}
	}
	// internal/pro are legacy token-derived tiers — must not grant product features by themselves.
	if tier == "internal" || tier == "pro" {
		return map[string]bool{}
	}
	all := []string{
		FeatureRemoteAPIAccess, FeatureCritique, FeatureUIReverse, FeatureStyleEnhance,
		FeatureLayoutCompare, FeatureSectionDetect, FeatureCopilot, FeatureMediaAccess,
		FeatureShareAccess, FeatureReferenceAccess, FeatureDesignSystem, FeatureContentMap,
		FeatureBlockRecipes, FeatureComparison, FeatureMigration,

		// UIAI-ENGINE-010 premium pillars (C-010-29). Granted to known licensed
		// tiers by default; authority-issued explicit feature lists override.
		FeaturePersonaStealth, FeatureAdaptiveRotation, FeatureSolverCoveragePro,
		FeatureConsensusReads, FeatureMeshWorkers, FeatureWebStateContinuity,
		FeatureTimeTravelExport,
	}
	features := make(map[string]bool, len(all))
	for _, f := range all {
		features[f] = true
	}
	return features
}

func commercialUseForTier(tier string) bool {
	switch tier {
	case "operator", "founders-forge", "team", "commercial", "enterprise":
		return true
	default:
		return false
	}
}

func copyBoolMap(src map[string]bool) map[string]bool {
	out := make(map[string]bool, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

// isLoopbackRequest reports whether the request originates from loopback (per spec §6.5
// eval-allowed semantics: 127.0.0.0/8, ::1, or Unix sockets).
func isLoopbackRequest(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.RemoteAddr == "" {
		return true
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx >= 0 {
		host = host[:idx]
	}
	host = strings.Trim(host, "[]")
	if host == "::1" || host == "127.0.0.1" || host == "localhost" || host == "" {
		return true
	}
	// Last-resort parse via net.IP — handles IPv6 and bracketed forms
	ip := netParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

// netParseIP is a thin indirection so this file does not import "net" twice when
// extended; it just delegates to net.ParseIP.
func netParseIP(s string) net.IP { return net.ParseIP(s) }

// IsEvaluation reports whether the caller is in evaluation mode.
// Per spec §6.5: loopback callers with no license are eval-allowed.
func IsEvaluation(id *auth.Identity, r *http.Request) bool {
	ent := FromIdentity(id)
	if ent.Mode == "evaluation" {
		return true
	}
	// A licensed loopback caller may still want eval-like behavior (testing in eval mode).
	// The current model says eval == no commercial license, so this returns true only when
	// the license is missing or in evaluation mode.
	return false
}

// FeatureEnabled reports whether the named feature is enabled for the caller.
// Per spec §6.5:
//
//	if request is loopback and feature is eval-allowed:
//	    allow
//	else if auth identity exists and tier permits feature:
//	    allow
//	else:
//	    return 402 or 403 license_required
func FeatureEnabled(id *auth.Identity, feature string, r *http.Request) bool {
	ent := FromIdentity(id)
	// 1. Loopback + eval-allowed feature
	if isLoopbackRequest(r) {
		if EvalAllowedFeatures[feature] {
			return true
		}
	}
	// 2. Licensed caller with feature in their tier's allow-list
	if ent.Mode == "licensed" {
		if ent.Features[feature] {
			return true
		}
	}
	return false
}

// RequireFeature is the short-circuit guard for route handlers.
// Returns true when the feature is enabled. Returns false (and writes the spec §6.5
// JSON envelope) when the feature is gated.
//
// Status code policy:
//   - 402 Payment Required — license_required (spec §6.5 recommended default)
//   - 401 Unauthorized — reserved for explicit "login required" auth gates
//   - 403 Forbidden — reserved for future per-tier blocks
//
// Callers should `return` immediately on false.
func RequireFeature(w http.ResponseWriter, r *http.Request, feature string) bool {
	id := auth.FromContext(r.Context())
	if FeatureEnabled(id, feature, r) {
		return true
	}
	// Per spec §6.5: when the feature is gated, default to 402 license_required.
	// The "no auth at all" case still maps to 402 here because the gating reason is
	// licensing, not authentication.
	writeLicenseRequired(w, feature, http.StatusPaymentRequired,
		"This UIAI Engine feature requires an Operator or Commercial license.",
	)
	return false
}

// isLoopbackEvalAllowed is a small helper used by RequireFeature to decide between
// 401 (no auth at all) and 402 (auth present, feature gated).
func isLoopbackEvalAllowed(r *http.Request, feature string) bool {
	return isLoopbackRequest(r) && EvalAllowedFeatures[feature]
}

// writeLicenseRequired writes the spec §6.5 / §11 JSON envelope and the appropriate status.
func writeLicenseRequired(w http.ResponseWriter, feature string, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(LicenseRequiredError{
		Feature:     feature,
		Error:       "license_required",
		Message:     message,
		PurchaseURL: purchaseURLBase,
		DocsURL:     docsURLBase,
	})
}
