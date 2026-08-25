package license

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/auth"
)

var premiumKeys = []string{
	FeaturePersonaStealth, FeatureAdaptiveRotation, FeatureSolverCoveragePro,
	FeatureConsensusReads, FeatureMeshWorkers, FeatureWebStateContinuity,
	FeatureTimeTravelExport,
}

func remoteReq() *http.Request {
	r := httptest.NewRequest("POST", "http://x/api/t", nil)
	r.RemoteAddr = "203.0.113.9:443" // non-loopback
	return r
}

// C-010-29: premium keys are never granted in evaluation mode.
func TestPremiumKeysDeniedInEvaluation(t *testing.T) {
	for _, key := range premiumKeys {
		if FeatureEnabled(nil, key, httptest.NewRequest("POST", "http://127.0.0.1/api/t", nil)) {
			t.Fatalf("%s must NOT be enabled for loopback evaluation", key)
		}
		id := &auth.Identity{} // authenticated but tierless → evaluation
		if FeatureEnabled(id, key, httptest.NewRequest("POST", "http://127.0.0.1/api/t", nil)) {
			t.Fatalf("%s must NOT be enabled for tierless identity", key)
		}
	}
}

// C-010-29: known licensed tiers carry the premium set; unknown tiers fail closed.
func TestPremiumKeysTrackTierGrants(t *testing.T) {
	ent := FromIdentity(&auth.Identity{Tier: "operator"})
	for _, key := range premiumKeys {
		if !ent.Features[key] {
			t.Fatalf("operator tier should grant %s", key)
		}
	}
	ghost := FromIdentity(&auth.Identity{Tier: "mystery"})
	for _, key := range premiumKeys {
		if ghost.Features[key] {
			t.Fatal("unknown tier must fail closed")
		}
	}
}

// C-010-29: remote unauthenticated callers receive the canonical 402 envelope.
func TestPremiumKeysDeniedRemoteAnonymous402(t *testing.T) {
	w := httptest.NewRecorder()
	r := remoteReq()
	if RequireFeature(w, r, FeatureConsensusReads) {
		t.Fatal("remote anonymous must not pass")
	}
	if w.Code != http.StatusPaymentRequired {
		t.Fatalf("want 402 got %d", w.Code)
	}
	want := "\"feature\":\"" + FeatureConsensusReads + "\""
	if !contains(w.Body.String(), want) {
		t.Fatalf("envelope missing feature key: %s", w.Body.String())
	}
}

func contains(hay, needle string) bool {
	return len(hay) >= len(needle) && (hay == needle || len(needle) == 0 || indexOf(hay, needle) >= 0)
}

func indexOf(hay, needle string) int {
	for i := 0; i+len(needle) <= len(hay); i++ {
		if hay[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
