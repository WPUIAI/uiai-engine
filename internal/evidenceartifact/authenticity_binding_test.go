package evidenceartifact

import (
	"errors"
	"testing"
)

// This proves binding of supplied references/revisions, not independent
// authentication of an account or the current state of an external authority.
func TestAttestationBindsCaptureIdentityAndStateSnapshot(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Manifest)
	}{
		{"target", func(m *Manifest) { m.Capture.TargetRef = "target:other" }},
		{"account", func(m *Manifest) { m.Capture.AccountRef = "account:other" }},
		{"runtime", func(m *Manifest) { m.Capture.RuntimeRef = "runtime:other" }},
		{"workset_revision", func(m *Manifest) { m.Scope.Workset.Revision++ }},
		{"workpoint_revision", func(m *Manifest) { m.Scope.Workpoint.Revision++ }},
		{"graph_generation", func(m *Manifest) { m.Scope.CallGraph.Generation++ }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			manifest, err := AssembleCapture(assemblyFixture(t, RequirementStaticVisual))
			if err != nil {
				t.Fatal(err)
			}
			if err := Validate(manifest); err != nil {
				t.Fatal(err)
			}
			_, template, bundle, privateKey, options := authenticityFixture(t)
			template.Federation.SourceManifestSHA256 = manifest.Integrity.ManifestSHA256
			attestation, err := SignAttestation(manifest, template, privateKey)
			if err != nil {
				t.Fatal(err)
			}
			if err := VerifyAttestation(manifest, attestation, bundle, options); err != nil {
				t.Fatal(err)
			}
			tc.mutate(&manifest)
			manifest = reseal(t, manifest)
			if err := VerifyManifestSHA256(manifest); err != nil {
				t.Fatal(err)
			}
			if err := VerifyAttestation(manifest, attestation, bundle, options); !errors.Is(err, ErrAttestationInvalid) {
				t.Fatalf("changed signed binding accepted: %v", err)
			}
		})
	}
}
