package evidenceartifact

import (
	"encoding/json"
	"errors"
	"testing"
)

func TestAirGapValidationAcceptsCleanMaterials(t *testing.T) {
	manifest, _, bundle, privateKey, options, err := AuthenticityDigestFixture()
	if err != nil {
		t.Fatal(err)
	}
	template := Attestation{
		IssuerRef: "instance:uiai", ActorRef: "agent:executor", Operation: "operation:attest", KeyID: "key:2026-01", Algorithm: AlgorithmEd25519V1,
		IssuedAt: "2026-08-29T12:00:01Z", TimeEvidence: TimeEvidence{Confidence: "anchored", ObservedAt: "2026-08-29T12:00:01Z", UncertaintyMS: 1000, SourceRefs: []string{"clock:system"}, AnchorRefs: []string{"anchor:receipt"}},
		DelegationRefs: []string{"delegation:executor"}, PolicyRefs: []string{"policy:evidence"},
		Federation: FederationState{Mode: "origin", OriginRef: "instance:uiai", SourceManifestSHA256: manifest.Integrity.ManifestSHA256, SourceAvailability: "available"},
	}
	attestation, err := SignAttestation(manifest, template, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateAirGapMaterials(manifest, attestation, bundle, options); err != nil {
		t.Fatalf("clean materials: %v", err)
	}
	if surfaces, err := AttestationRemoteSurfaces(manifest, attestation, bundle); err != nil || len(surfaces) != 0 {
		t.Fatalf("surfaces=%v err=%v", surfaces, err)
	}
}

func TestAirGapValidationAllowsCitationsButRequiresPinnedTrust(t *testing.T) {
	_, template, bundle, privateKey, options, err := AuthenticityDigestFixture()
	if err != nil {
		t.Fatal(err)
	}
	manifest, err := Seal(ReferenceEvidenceManifest())
	if err != nil {
		t.Fatal(err)
	}
	manifest.Title = "Proof https://remote.example/manifest"
	manifest, err = Seal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	template.Federation.SourceManifestSHA256 = manifest.Integrity.ManifestSHA256
	template.Federation.OriginRef = "instance:uiai"
	attestation, err := SignAttestation(manifest, template, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateAirGapMaterials(manifest, attestation, bundle, options); err != nil {
		t.Fatalf("offline citation verification: %v", err)
	}
	options.TrustedBundleSHA256 = ""
	if err := ValidateAirGapMaterials(manifest, attestation, bundle, options); !errors.Is(err, ErrTrustBundleInvalid) {
		t.Fatalf("missing trust pin accepted: %v", err)
	}
	surfaces, err := AttestationRemoteSurfaces(manifest, attestation, bundle)
	if err != nil || len(surfaces) != 1 || surfaces[0] != "manifest" {
		t.Fatalf("surfaces=%v err=%v", surfaces, err)
	}
}

func TestHumanIdentityProofRoundTripAndTamper(t *testing.T) {
	_, _, bundle, _, _, err := AuthenticityDigestFixture()
	if err != nil {
		t.Fatal(err)
	}
	proof, err := HumanIdentityProofFor(bundle, bundle.Keys[0].KeyID)
	if err != nil {
		t.Fatal(err)
	}
	if proof.Schema != HumanIdentityProofSchema || len(proof.Fingerprint) != 64 {
		t.Fatalf("proof shape: %+v", proof)
	}
	if len(proof.Grouped) == 0 || proof.Grouped == proof.Fingerprint {
		t.Fatalf("grouped rendering missing: %q", proof.Grouped)
	}
	if err := VerifyHumanIdentityProof(bundle, proof); err != nil {
		t.Fatalf("verify: %v", err)
	}
	tampered := proof
	tampered.InstanceRef = "instance:other"
	if err := VerifyHumanIdentityProof(bundle, tampered); !errors.Is(err, ErrHumanIdentityProofInvalid) {
		t.Fatalf("tampered err=%v", err)
	}
	missing, err := HumanIdentityProofFor(bundle, "key:missing")
	if err == nil || missing.KeyID != "" {
		t.Fatalf("missing key err=%v proof=%+v", err, missing)
	}
}

func TestHumanIdentityProofRejectsInvalidTrustBundle(t *testing.T) {
	_, _, original, _, _, err := AuthenticityDigestFixture()
	if err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*TrustBundle){
		func(b *TrustBundle) { b.Keys[0].PublicKey = "YQ" },
		func(b *TrustBundle) { b.Keys = append(b.Keys, b.Keys[0]) },
		func(b *TrustBundle) { b.AuthorityRef = "" },
		func(b *TrustBundle) { b.Keys[0].Algorithm = "unknown" },
	} {
		bundle := original
		bundle.Keys = append([]TrustKey(nil), original.Keys...)
		change(&bundle)
		if _, err := HumanIdentityProofFor(bundle, original.Keys[0].KeyID); !errors.Is(err, ErrTrustBundleInvalid) {
			t.Fatalf("invalid bundle accepted: %v", err)
		}
	}
}

func TestAuthenticityDigestDeterministicAndVerifiable(t *testing.T) {
	first, err := RunAuthenticityDigest(3, "test-cg06")
	if err != nil {
		t.Fatal(err)
	}
	second, err := RunAuthenticityDigest(3, "cg06-local-probe")
	if err != nil {
		t.Fatal(err)
	}
	if first.DigestSHA256 != second.DigestSHA256 {
		t.Fatalf("digest drift: %s vs %s", first.DigestSHA256, second.DigestSHA256)
	}
	if len(first.Clauses) != len(AuthenticityDigestClauses) || first.Runs != 3 || len(first.PerRun) != 3 {
		t.Fatalf("report shape: %+v", first)
	}
	body, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	var roundTrip AuthenticityDigestReport
	if err := json.Unmarshal(body, &roundTrip); err != nil {
		t.Fatal(err)
	}
	if err := VerifyAuthenticityDigestReport(roundTrip, 2); err != nil {
		t.Fatalf("evidence report round trip: %v", err)
	}
	if _, err := RunAuthenticityDigest(1, ""); err == nil {
		t.Fatal("missing code reference accepted")
	}
	if err := VerifyAuthenticityDigestReport(first, 2); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := VerifyAuthenticityDigestReport(first, 1001); err == nil {
		t.Fatal("runs out of range accepted")
	}
	bad := first
	bad.PerRun = append([]string(nil), first.PerRun...)
	bad.PerRun[1] = "0000000000000000000000000000000000000000000000000000000000000000"
	if err := VerifyAuthenticityDigestReport(bad, 2); err == nil {
		t.Fatal("divergent later run accepted despite all_identical claim")
	}
	bad = first
	bad.Results = append([]authenticityClauseResult(nil), first.Results...)
	bad.Results[0].Materials = []byte(`["substituted evidence"]`)
	if err := VerifyAuthenticityDigestReport(bad, 2); err == nil {
		t.Fatal("substituted evidence accepted with unchanged digest")
	}
	bad = first
	bad.Schema = "uiai.authenticity_digest_report.v1"
	if err := VerifyAuthenticityDigestReport(bad, 2); err == nil {
		t.Fatal("legacy pass-label report accepted as evidence-bound v2")
	}
	bad = first
	bad.DigestSHA256 = ""
	if err := VerifyAuthenticityDigestReport(bad, 2); err == nil {
		t.Fatal("empty digest accepted")
	}
}
