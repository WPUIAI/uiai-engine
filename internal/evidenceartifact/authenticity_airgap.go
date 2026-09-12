package evidenceartifact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

// Air-gap verification contract (CG-06 clause: air-gapped / link-rot
// resistance): an attestation is verifiable offline only when every byte
// needed to verify it is embedded in the provided materials. Any remote
// surface (network URL) in the canonical JSON of the manifest, attestation,
// or trust bundle means the "offline" verify could silently depend on a
// fetchable resource that can rot, move, or be attacker-controlled.
//
// VerifyAttestation is already a pure function over provided bytes; this
// check makes the air-gap guarantee explicit and rejects contaminated
// materials before verification.

var ErrAirGapContaminated = errors.New("air-gapped verification materials contain remote surfaces")

var remoteSurfaceNeedles = []string{"http://", "https://", "ftp://", "wss://", "ws://"}

// AttestationRemoteSurfaces scans the canonical JSON of the manifest,
// attestation, and trust bundle for network URL surfaces and returns the
// material names where they occur. An empty result means all materials are
// self-contained and safe for air-gapped verification.
func AttestationRemoteSurfaces(manifest Manifest, attestation Attestation, bundle TrustBundle) ([]string, error) {
	canonicalManifest, err := CanonicalBytes(manifest)
	if err != nil {
		return nil, err
	}
	canonicalAttestation, err := json.Marshal(attestation)
	if err != nil {
		return nil, err
	}
	canonicalBundle, err := json.Marshal(bundle)
	if err != nil {
		return nil, err
	}
	var surfaces []string
	for _, part := range []struct {
		name  string
		bytes []byte
	}{
		{"manifest", canonicalManifest},
		{"attestation", canonicalAttestation},
		{"trust_bundle", canonicalBundle},
	} {
		for _, needle := range remoteSurfaceNeedles {
			if strings.Contains(string(part.bytes), needle) {
				surfaces = append(surfaces, part.name)
			}
		}
	}
	return surfaces, nil
}

// ValidateAirGapMaterials rejects materials containing any remote surface,
// then runs the full offline attestation verification. All inputs must be
// locally provided bytes; the function performs no network access.
func ValidateAirGapMaterials(manifest Manifest, attestation Attestation, bundle TrustBundle, options VerifyAttestationOptions) error {
	surfaces, err := AttestationRemoteSurfaces(manifest, attestation, bundle)
	if err != nil {
		return err
	}
	if len(surfaces) != 0 {
		return ErrAirGapContaminated
	}
	return VerifyAttestation(manifest, attestation, bundle, options)
}

// FingerprintBytes is the canonical fingerprint input for a trust key: the
// raw decoded public key bytes.
func FingerprintBytes(decodedPublicKey []byte) string {
	sum := sha256.Sum256(decodedPublicKey)
	return hex.EncodeToString(sum[:])
}
