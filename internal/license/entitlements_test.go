package license

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/auth"
)

func TestFromIdentity_NoIdentity_IsEvaluation(t *testing.T) {
	ent := FromIdentity(nil)
	if ent.Mode != "evaluation" {
		t.Fatalf("want evaluation, got %s", ent.Mode)
	}
	if !ent.Features[FeatureLocalSearch] {
		t.Fatalf("loopback eval must grant eval_allowed features")
	}
}

func TestFromIdentity_Operator_CommercialUse(t *testing.T) {
	id := &auth.Identity{Tier: "operator"}
	ent := FromIdentity(id)
	if ent.Mode != "licensed" {
		t.Fatalf("want licensed, got %s", ent.Mode)
	}
	if !ent.CommercialUse {
		t.Fatalf("operator tier must allow commercial use")
	}
	if !ent.Features[FeatureCritique] {
		t.Fatalf("operator must grant critique")
	}
}

func TestFeatureEnabled_LoopbackEvalAllowed(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/search", nil)
	r.RemoteAddr = "127.0.0.1:54321"
	if !FeatureEnabled(nil, FeatureLocalSearch, r) {
		t.Fatalf("loopback caller must be allowed for local_search")
	}
	if FeatureEnabled(nil, FeatureCritique, r) {
		t.Fatalf("loopback must NOT be allowed for critique (spec §6.6 license-gated)")
	}
}

func TestFeatureEnabled_RemoteLoopbackEval(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/critique", nil)
	r.RemoteAddr = "192.168.1.5:54321"
	// No identity (nil) AND remote (not loopback) → feature must NOT be allowed.
	if FeatureEnabled(nil, FeatureCritique, r) {
		t.Fatalf("unlicensed remote caller must not be allowed for critique")
	}
	// Licensed remote caller → must be allowed.
	if !FeatureEnabled(&auth.Identity{Tier: "operator"}, FeatureCritique, r) {
		t.Fatalf("licensed remote caller must be allowed for critique")
	}
	// Loopback remote caller (no identity) → must NOT be allowed (only eval-allowed).
	r2 := httptest.NewRequest("GET", "/api/critique", nil)
	r2.RemoteAddr = "127.0.0.1:1234"
	if FeatureEnabled(nil, FeatureCritique, r2) {
		t.Fatalf("loopback caller with no license must not be allowed for critique")
	}
}

func TestRequireFeature_WritesSpec_6_5_JSONEnvelope(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/critique", nil)
	r.RemoteAddr = "192.168.1.5:1234"
	w := httptest.NewRecorder()
	if RequireFeature(w, r, FeatureCritique) {
		t.Fatalf("RequireFeature should fail for unlicensed remote caller")
	}
	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("want 402, got %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), `"license_required"`) {
		t.Fatalf("body must contain license_required: %s", w.Body.String())
	}
	var env LicenseRequiredError
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil {
		t.Fatalf("body must be JSON: %v", err)
	}
	if env.Feature != FeatureCritique {
		t.Fatalf("want feature=%s, got %s", FeatureCritique, env.Feature)
	}
	if env.PurchaseURL == "" || env.DocsURL == "" {
		t.Fatalf("purchase_url and docs_url must be set")
	}
}

func TestRequireFeature_NoAuth_LoopbackEval_OK(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/search", nil)
	r.RemoteAddr = "127.0.0.1:8080"
	w := httptest.NewRecorder()
	if !RequireFeature(w, r, FeatureLocalSearch) {
		t.Fatalf("loopback eval should be allowed for eval_allowed features")
	}
}
