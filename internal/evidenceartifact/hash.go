package evidenceartifact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// CanonicalJSON returns the deterministic normalized manifest JSON used for
// hashing. The self-referential manifest_sha256 field is always cleared.
func CanonicalJSON(in Manifest) ([]byte, error) {
	m := Normalize(in)
	m.Integrity.ManifestSHA256 = ""
	if err := Validate(m); err != nil {
		return nil, err
	}
	return json.Marshal(m)
}

func ComputeManifestSHA256(in Manifest) (string, error) {
	canonical, err := CanonicalJSON(in)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(canonical)
	return hex.EncodeToString(digest[:]), nil
}

func Seal(in Manifest) (Manifest, error) {
	out := Normalize(in)
	digest, err := ComputeManifestSHA256(out)
	if err != nil {
		return Manifest{}, err
	}
	out.Integrity.ManifestSHA256 = digest
	return out, nil
}

func VerifyManifestSHA256(in Manifest) error {
	m := Normalize(in)
	if !validSHA256(m.Integrity.ManifestSHA256) {
		return ErrInvalidIntegrity
	}
	expected := m.Integrity.ManifestSHA256
	actual, err := ComputeManifestSHA256(m)
	if err != nil {
		return err
	}
	if expected != actual {
		return ErrInvalidIntegrity
	}
	return nil
}
