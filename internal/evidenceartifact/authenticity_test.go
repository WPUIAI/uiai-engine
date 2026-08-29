package evidenceartifact

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestAttestationSignVerifyDeterministic(t *testing.T) {
	manifest, template, bundle, privateKey, options := authenticityFixture(t)
	first, err := SignAttestation(manifest, template, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	second, err := SignAttestation(manifest, template, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) {
		t.Fatal("deterministic signature differs")
	}
	if err := VerifyAttestation(manifest, first, bundle, options); err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(first)
	decoded, err := DecodeAttestation(encoded)
	if err != nil || decoded.AttestationID != first.AttestationID {
		t.Fatalf("decode=%v err=%v", decoded, err)
	}
}

func TestAttestationTamperingFails(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Manifest, *Attestation, *TrustBundle, *VerifyAttestationOptions)
		want   error
	}{
		{"manifest", func(m *Manifest, _ *Attestation, _ *TrustBundle, _ *VerifyAttestationOptions) { m.Title = "tampered" }, ErrAttestationInvalid},
		{"actor", func(_ *Manifest, a *Attestation, _ *TrustBundle, _ *VerifyAttestationOptions) {
			a.ActorRef = "agent:other"
		}, ErrAttestationInvalid},
		{"key", func(_ *Manifest, a *Attestation, _ *TrustBundle, _ *VerifyAttestationOptions) { a.KeyID = "key:other" }, ErrAttestationInvalid},
		{"policy", func(_ *Manifest, a *Attestation, _ *TrustBundle, _ *VerifyAttestationOptions) {
			a.PolicyRefs = []string{"policy:other"}
		}, ErrAttestationInvalid},
		{"custody", func(m *Manifest, _ *Attestation, _ *TrustBundle, _ *VerifyAttestationOptions) {
			m.Provenance.Custody[0].Action = "changed"
			*m = reseal(t, *m)
		}, ErrAttestationInvalid},
		{"signature", func(_ *Manifest, a *Attestation, _ *TrustBundle, _ *VerifyAttestationOptions) {
			a.Signature = base64.RawURLEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
		}, ErrSignatureInvalid},
		{"bundle_digest", func(_ *Manifest, _ *Attestation, _ *TrustBundle, o *VerifyAttestationOptions) {
			o.TrustedBundleSHA256 = strings.Repeat("0", 64)
		}, ErrTrustBundleInvalid},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, template, b, privateKey, o := authenticityFixture(t)
			a, err := SignAttestation(m, template, privateKey)
			if err != nil {
				t.Fatal(err)
			}
			tt.mutate(&m, &a, &b, &o)
			if err := VerifyAttestation(m, a, b, o); !errors.Is(err, tt.want) {
				t.Fatalf("err=%v want=%v", err, tt.want)
			}
		})
	}
}

func TestAttestationKeyLifecycle(t *testing.T) {
	manifest, template, bundle, privateKey, options := authenticityFixture(t)
	signed, _ := SignAttestation(manifest, template, privateKey)
	retired := bundle
	retired.Keys = append([]TrustKey(nil), bundle.Keys...)
	retired.Keys[0].Status = "retired"
	options.TrustedBundleSHA256 = mustBundleHash(t, retired)
	if err := VerifyAttestation(manifest, signed, retired, options); err != nil {
		t.Fatalf("retired historical key: %v", err)
	}
	revoked := retired
	revoked.Keys = append([]TrustKey(nil), retired.Keys...)
	revoked.Keys[0].Status = "revoked"
	revoked.Keys[0].RevokedAt = "2026-08-29T13:00:00Z"
	options.TrustedBundleSHA256 = mustBundleHash(t, revoked)
	if err := VerifyAttestation(manifest, signed, revoked, options); err != nil {
		t.Fatalf("pre-revocation signature: %v", err)
	}
	revoked.Keys[0].RetroactiveRevoke = true
	options.TrustedBundleSHA256 = mustBundleHash(t, revoked)
	if err := VerifyAttestation(manifest, signed, revoked, options); !errors.Is(err, ErrKeyRevoked) {
		t.Fatalf("retroactive err=%v", err)
	}
	compromised := retired
	compromised.Keys = append([]TrustKey(nil), retired.Keys...)
	compromised.Keys[0].Status = "compromised"
	compromised.Keys[0].CompromisedAt = "2026-08-29T11:59:59Z"
	options.TrustedBundleSHA256 = mustBundleHash(t, compromised)
	if err := VerifyAttestation(manifest, signed, compromised, options); !errors.Is(err, ErrKeyRevoked) {
		t.Fatalf("compromised err=%v", err)
	}
}

func TestAttestationDelegationAndTime(t *testing.T) {
	manifest, template, bundle, privateKey, options := authenticityFixture(t)
	signed, _ := SignAttestation(manifest, template, privateKey)
	expired := bundle
	expired.Delegations = append([]Delegation(nil), bundle.Delegations...)
	expired.Delegations[0].ValidUntil = "2026-08-29T12:00:00Z"
	options.TrustedBundleSHA256 = mustBundleHash(t, expired)
	if err := VerifyAttestation(manifest, signed, expired, options); !errors.Is(err, ErrDelegationInvalid) {
		t.Fatalf("delegation err=%v", err)
	}
	badTime := template
	badTime.TimeEvidence.Confidence = "unknown"
	if _, err := SignAttestation(manifest, badTime, privateKey); !errors.Is(err, ErrTimeUntrusted) {
		t.Fatalf("unknown time err=%v", err)
	}
	outside := template
	outside.IssuedAt = "2026-08-29T12:10:00Z"
	if signedOutside, err := SignAttestation(manifest, outside, privateKey); err != nil {
		t.Fatal(err)
	} else {
		options.TrustedBundleSHA256 = mustBundleHash(t, bundle)
		if err := VerifyAttestation(manifest, signedOutside, bundle, options); !errors.Is(err, ErrTimeUntrusted) {
			t.Fatalf("uncertainty err=%v", err)
		}
	}
}

func TestAttestationFederationStates(t *testing.T) {
	manifest, template, bundle, privateKey, options := authenticityFixture(t)
	for _, mode := range []string{"origin", "mirror", "import", "offline_copy"} {
		t.Run(mode, func(t *testing.T) {
			candidate := template
			candidate.Federation.Mode = mode
			if mode != "origin" {
				candidate.Federation.ImportedAt = "2026-08-29T12:00:02Z"
				candidate.Federation.ImportReceiptRef = "receipt:import"
				candidate.Federation.PreviousAttestationRef = "attestation:source"
				candidate.Federation.SourceAvailability = "lost"
			}
			signed, err := SignAttestation(manifest, candidate, privateKey)
			if err != nil {
				t.Fatal(err)
			}
			if err := VerifyAttestation(manifest, signed, bundle, options); err != nil {
				t.Fatal(err)
			}
		})
	}
	conflict := template
	conflict.Federation.ConflictRef = "conflict:split-brain"
	signedConflict, _ := SignAttestation(manifest, conflict, privateKey)
	if err := VerifyAttestation(manifest, signedConflict, bundle, options); !errors.Is(err, ErrFederationConflict) {
		t.Fatalf("conflict err=%v", err)
	}
	retracted := template
	retracted.Federation.RetractionRef = "retraction:1"
	signedRetraction, _ := SignAttestation(manifest, retracted, privateKey)
	if err := VerifyAttestation(manifest, signedRetraction, bundle, options); !errors.Is(err, ErrEvidenceRetracted) {
		t.Fatalf("retraction err=%v", err)
	}
	corrected := template
	corrected.Federation.CorrectionRef = "correction:1"
	signedCorrection, _ := SignAttestation(manifest, corrected, privateKey)
	if err := VerifyAttestation(manifest, signedCorrection, bundle, options); err != nil {
		t.Fatalf("correction err=%v", err)
	}
}

func TestAttestationMalformedInputs(t *testing.T) {
	manifest, template, bundle, privateKey, options := authenticityFixture(t)
	signed, _ := SignAttestation(manifest, template, privateKey)
	bundle.Keys[0].PublicKey = "not-base64"
	options.TrustedBundleSHA256 = strings.Repeat("0", 64)
	if err := VerifyAttestation(manifest, signed, bundle, options); !errors.Is(err, ErrTrustBundleInvalid) {
		t.Fatalf("key err=%v", err)
	}
	if _, err := DecodeAttestation([]byte(`{"schema":"x","unexpected":true}`)); !errors.Is(err, ErrAttestationInvalid) {
		t.Fatalf("decode err=%v", err)
	}
}

func authenticityFixture(t *testing.T) (Manifest, Attestation, TrustBundle, ed25519.PrivateKey, VerifyAttestationOptions) {
	t.Helper()
	manifest := reseal(t, testManifest())
	privateKey := ed25519.NewKeyFromSeed(bytesOf(7, ed25519.SeedSize))
	publicKey := privateKey.Public().(ed25519.PublicKey)
	template := Attestation{
		IssuerRef: "instance:uiai", ActorRef: "agent:executor", Operation: "operation:attest", KeyID: "key:2026-01", Algorithm: AlgorithmEd25519V1,
		IssuedAt: "2026-08-29T12:00:01Z", TimeEvidence: TimeEvidence{Confidence: "anchored", ObservedAt: "2026-08-29T12:00:01Z", UncertaintyMS: 1000, SourceRefs: []string{"clock:system"}, AnchorRefs: []string{"anchor:receipt"}},
		DelegationRefs: []string{"delegation:executor"}, PolicyRefs: []string{"policy:evidence"},
		Federation: FederationState{Mode: "origin", OriginRef: "instance:uiai", SourceManifestSHA256: manifest.Integrity.ManifestSHA256, SourceAvailability: "available"},
	}
	bundle := TrustBundle{
		Schema: TrustBundleSchemaV1, InstanceRef: "instance:uiai", AuthorityRef: "authority:uiai", Revision: 1,
		Keys:        []TrustKey{{KeyID: template.KeyID, Algorithm: AlgorithmEd25519V1, PublicKey: base64.RawURLEncoding.EncodeToString(publicKey), ValidFrom: "2026-01-01T00:00:00Z", ValidUntil: "2027-01-01T00:00:00Z", Status: "active"}},
		Delegations: []Delegation{{DelegationRef: template.DelegationRefs[0], ActorRef: template.ActorRef, InstanceRef: template.IssuerRef, KeyID: template.KeyID, Operations: []string{template.Operation}, PolicyRefs: []string{"policy:evidence"}, ValidFrom: "2026-08-01T00:00:00Z", ValidUntil: "2026-09-01T00:00:00Z"}},
	}
	options := VerifyAttestationOptions{TrustedBundleSHA256: mustBundleHash(t, bundle), AsOf: time.Date(2026, 8, 29, 13, 0, 0, 0, time.UTC)}
	return manifest, template, bundle, privateKey, options
}

func mustBundleHash(t *testing.T, bundle TrustBundle) string {
	t.Helper()
	digest, err := TrustBundleSHA256(bundle)
	if err != nil {
		t.Fatal(err)
	}
	return digest
}

func bytesOf(value byte, count int) []byte {
	out := make([]byte, count)
	for i := range out {
		out[i] = value
	}
	return out
}
