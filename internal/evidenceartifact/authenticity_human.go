package evidenceartifact

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
)

// ErrHumanIdentityProofInvalid is returned when a human identity proof does
// not match the trust bundle it claims to describe.
var ErrHumanIdentityProofInvalid = errors.New("human identity proof does not match trust bundle")

// HumanIdentityProof (CG-06 clause: human-readable identity proofs) is the
// deterministic, human-verifiable rendering of one trust key inside a trust
// bundle. A human comparing two devices or printed sheets can compare the
// grouped fingerprint groups; a verifier recomputes the fingerprint from
// the bundle key bytes, so the proof is tamper-evident by construction.
type HumanIdentityProof struct {
	Schema       string `json:"schema"`
	InstanceRef  string `json:"instance_ref"`
	AuthorityRef string `json:"authority_ref"`
	KeyID        string `json:"key_id"`
	Algorithm    string `json:"algorithm"`
	Fingerprint  string `json:"fingerprint"` // sha256 of the decoded public key, 64 lowercase hex
	Grouped      string `json:"grouped"`     // fingerprint in 4-char groups separated by single spaces
	ProofSHA256  string `json:"proof_sha256"`
}

const HumanIdentityProofSchema = "uiai.evidence.human_identity_proof.v1"

// HumanIdentityProofFor derives the proof for one key of the bundle. The
// fingerprint is computed from the bundle's own public key bytes, so any
// bundle change that swaps the key invalidates the previously printed proof.
func HumanIdentityProofFor(bundle TrustBundle, keyID string) (HumanIdentityProof, error) {
	var matched *TrustKey
	for i := range bundle.Keys {
		if bundle.Keys[i].KeyID == keyID {
			matched = &bundle.Keys[i]
			break
		}
	}
	if matched == nil {
		return HumanIdentityProof{}, ErrKeyUntrusted
	}
	decoded, err := base64.RawURLEncoding.DecodeString(matched.PublicKey)
	if err != nil || len(decoded) == 0 {
		return HumanIdentityProof{}, ErrTrustBundleInvalid
	}
	proof := HumanIdentityProof{
		Schema:       HumanIdentityProofSchema,
		InstanceRef:  bundle.InstanceRef,
		AuthorityRef: bundle.AuthorityRef,
		KeyID:        matched.KeyID,
		Algorithm:    matched.Algorithm,
		Fingerprint:  FingerprintBytes(decoded),
	}
	proof.Grouped = groupHex(proof.Fingerprint)
	encoded, err := json.Marshal(proof)
	if err != nil {
		return HumanIdentityProof{}, err
	}
	sum := sha256.Sum256(encoded)
	proof.ProofSHA256 = hex.EncodeToString(sum[:])
	return proof, nil
}

// VerifyHumanIdentityProof recomputes every field of the proof from the
// bundle and rejects any divergence — a re-printed or edited proof cannot
// pass unless it matches the bundle exactly.
func VerifyHumanIdentityProof(bundle TrustBundle, proof HumanIdentityProof) error {
	if proof.Schema != HumanIdentityProofSchema {
		return ErrHumanIdentityProofInvalid
	}
	expected, err := HumanIdentityProofFor(bundle, proof.KeyID)
	if err != nil {
		return err
	}
	if proof != expected {
		return ErrHumanIdentityProofInvalid
	}
	return nil
}

func groupHex(fingerprint string) string {
	if len(fingerprint) == 0 {
		return ""
	}
	groups := make([]string, 0, len(fingerprint)/4+1)
	for i := 0; i < len(fingerprint); i += 4 {
		end := i + 4
		if end > len(fingerprint) {
			end = len(fingerprint)
		}
		groups = append(groups, fingerprint[i:end])
	}
	return strings.Join(groups, " ")
}
