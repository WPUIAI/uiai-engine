package evidenceartifact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

// Offline attestation verification uses only the supplied manifest, signature,
// trust bundle and pinned verification options. Source URLs are citations, not
// dependencies: their presence or absence cannot prove offline completeness.
// Packaging assets/tools and current revocation are separate acceptance checks.

const OfflineVerificationNotice = "Verified only against supplied trust state; current revocation, supersession and live-source availability are not checked offline."

var ErrAirGapContaminated = errors.New("air-gapped verification materials contain remote surfaces")

var remoteSurfaceNeedles = []string{"http://", "https://", "ftp://", "wss://", "ws://"}

// AttestationRemoteSurfaces scans the canonical JSON of the manifest,
// attestation, and trust bundle for network URL surfaces and returns the
// material names where they occur. This is diagnostic metadata only; neither
// an empty nor a nonempty result establishes offline verification readiness.
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

// ValidateAirGapMaterials validates the manifest and verifies its attestation
// against locally supplied, explicitly pinned trust state without network access.
// It does not certify portable asset completeness or current revocation status.
func ValidateAirGapMaterials(manifest Manifest, attestation Attestation, bundle TrustBundle, options VerifyAttestationOptions) error {
	if err := Validate(manifest); err != nil {
		return err
	}
	return VerifyAttestation(manifest, attestation, bundle, options)
}

// FingerprintBytes is the canonical fingerprint input for a trust key: the
// raw decoded public key bytes.
func FingerprintBytes(decodedPublicKey []byte) string {
	sum := sha256.Sum256(decodedPublicKey)
	return hex.EncodeToString(sum[:])
}
