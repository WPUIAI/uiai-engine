package evidencederivative

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"testing"
)

func TestNonVisualRenderingProfileDigestsAreFrozen(t *testing.T) {
	cases := []struct {
		body string
		want string
	}{
		{"uiai.evidence.portable-data-v1\n", PortableDataProfileSHA256},
		{"uiai.evidence.archive-store-v1\n", ArchiveStoreProfileSHA256},
	}
	for _, test := range cases {
		digest := sha256.Sum256([]byte(test.body))
		if got := fmt.Sprintf("%x", digest); got != test.want {
			t.Fatalf("profile digest = %s, want %s", got, test.want)
		}
	}
}

func TestDataRequestAllowsTruthfulNoFontProfile(t *testing.T) {
	request, _, _ := contracts()
	request.DerivativeType = DerivativeJSON
	request.AccessibilityTarget = AccessibilityNotApplicable
	request.Rendering = PortableDataRenderingProfile()
	if err := ValidateRequest(request); err != nil {
		t.Fatal(err)
	}
}

func TestCanonicalJSONRejectsFalseRenderingProfile(t *testing.T) {
	request, projection, manifest := portableFixture(t)
	request.DerivativeType = DerivativeJSON
	request.Rendering.ProfileRef = "rendering:unimplemented"
	if _, err := RenderProjectionJSON(request, projection, manifest.Renderer, manifest.ViewerMatrix, manifest.Licenses, "receipt:false-profile", manifest.CreatedAt); !errors.Is(err, ErrDerivativeContractInvalid) {
		t.Fatalf("unimplemented requested profile accepted: %v", err)
	}
}
