package evidenceartifact

import (
	"errors"
	"testing"
)

func TestValidateRequiresEvidenceForRedactionClaims(t *testing.T) {
	manifest := testManifest()
	manifest.Security.RedactionRefs = nil
	if err := Validate(manifest); !errors.Is(err, ErrInvalidPolicy) {
		t.Fatalf("Validate() error = %v, want ErrInvalidPolicy", err)
	}
}

func TestValidateBindsAssetRedactionToManifestPolicy(t *testing.T) {
	tests := []struct {
		name        string
		policyState RedactionState
		assetState  RedactionState
		wantErr     bool
	}{
		{name: "public safe", policyState: RedactionPublicSafe, assetState: RedactionPublicSafe},
		{name: "public safe with blocked asset", policyState: RedactionPublicSafe, assetState: RedactionBlocked},
		{name: "public claim with merely redacted asset", policyState: RedactionPublicSafe, assetState: RedactionRedacted, wantErr: true},
		{name: "redacted", policyState: RedactionRedacted, assetState: RedactionRedacted},
		{name: "redacted with blocked asset", policyState: RedactionRedacted, assetState: RedactionBlocked},
		{name: "redacted policy with raw asset", policyState: RedactionRedacted, assetState: RedactionNone, wantErr: true},
		{name: "raw", policyState: RedactionNone, assetState: RedactionNone},
		{name: "raw with blocked asset", policyState: RedactionNone, assetState: RedactionBlocked},
		{name: "raw policy with public-safe asset", policyState: RedactionNone, assetState: RedactionPublicSafe, wantErr: true},
		{name: "blocked", policyState: RedactionBlocked, assetState: RedactionBlocked},
		{name: "blocked policy with public-safe asset", policyState: RedactionBlocked, assetState: RedactionPublicSafe, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest := testManifest()
			manifest.Policy.RedactionState = tt.policyState
			manifest.Assets[0].RedactionState = tt.assetState
			manifest.Security.RedactionRefs = []string{"redaction:proof"}
			err := Validate(manifest)
			if tt.wantErr && !errors.Is(err, ErrInvalidPolicy) {
				t.Fatalf("Validate() error = %v, want ErrInvalidPolicy", err)
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}
}
