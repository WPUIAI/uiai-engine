package evidenceartifact

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"
	"time"
)

var (
	ErrAttestationInvalid = errors.New("invalid evidence attestation")
	ErrSignatureInvalid   = errors.New("invalid evidence signature")
	ErrTrustBundleInvalid = errors.New("invalid evidence trust bundle")
	ErrKeyUntrusted       = errors.New("evidence signing key untrusted")
	ErrKeyRevoked         = errors.New("evidence signing key revoked")
	ErrDelegationInvalid  = errors.New("evidence delegation invalid")
	ErrTimeUntrusted      = errors.New("evidence time untrusted")
	ErrFederationConflict = errors.New("evidence federation conflict")
	ErrEvidenceRetracted  = errors.New("evidence attestation retracted")
)

const (
	AttestationSchemaV1 = "uiai.evidence_attestation.v1"
	TrustBundleSchemaV1 = "uiai.evidence_trust_bundle.v1"
	AlgorithmEd25519V1  = "ed25519-v1"
)

type Attestation struct {
	Schema          string          `json:"schema"`
	AttestationID   string          `json:"attestation_id"`
	ArtifactID      string          `json:"artifact_id"`
	Revision        uint64          `json:"revision"`
	ManifestSHA256  string          `json:"manifest_sha256"`
	IssuerRef       string          `json:"issuer_ref"`
	ActorRef        string          `json:"actor_ref"`
	Operation       string          `json:"operation"`
	KeyID           string          `json:"key_id"`
	Algorithm       string          `json:"algorithm"`
	IssuedAt        string          `json:"issued_at"`
	TimeEvidence    TimeEvidence    `json:"time_evidence"`
	CustodyHeadHash string          `json:"custody_head_hash"`
	DelegationRefs  []string        `json:"delegation_refs"`
	PolicyRefs      []string        `json:"policy_refs"`
	Federation      FederationState `json:"federation"`
	Signature       string          `json:"signature"`
}

type TimeEvidence struct {
	Confidence    string   `json:"confidence"`
	ObservedAt    string   `json:"observed_at"`
	UncertaintyMS uint64   `json:"uncertainty_ms"`
	SourceRefs    []string `json:"source_refs"`
	AnchorRefs    []string `json:"anchor_refs"`
}

type FederationState struct {
	Mode                   string `json:"mode"`
	OriginRef              string `json:"origin_ref"`
	SourceManifestSHA256   string `json:"source_manifest_sha256"`
	ImportedAt             string `json:"imported_at,omitempty"`
	ImportReceiptRef       string `json:"import_receipt_ref,omitempty"`
	PreviousAttestationRef string `json:"previous_attestation_ref,omitempty"`
	SourceAvailability     string `json:"source_availability"`
	CorrectionRef          string `json:"correction_ref,omitempty"`
	RetractionRef          string `json:"retraction_ref,omitempty"`
	ConflictRef            string `json:"conflict_ref,omitempty"`
}

type TrustBundle struct {
	Schema       string       `json:"schema"`
	InstanceRef  string       `json:"instance_ref"`
	AuthorityRef string       `json:"authority_ref"`
	Revision     uint64       `json:"revision"`
	Keys         []TrustKey   `json:"keys"`
	Delegations  []Delegation `json:"delegations"`
}

type TrustKey struct {
	KeyID             string `json:"key_id"`
	Algorithm         string `json:"algorithm"`
	PublicKey         string `json:"public_key"`
	ValidFrom         string `json:"valid_from"`
	ValidUntil        string `json:"valid_until"`
	Status            string `json:"status"`
	RevokedAt         string `json:"revoked_at,omitempty"`
	CompromisedAt     string `json:"compromised_at,omitempty"`
	RetroactiveRevoke bool   `json:"retroactive_revoke"`
}

type Delegation struct {
	DelegationRef string   `json:"delegation_ref"`
	ActorRef      string   `json:"actor_ref"`
	InstanceRef   string   `json:"instance_ref"`
	KeyID         string   `json:"key_id"`
	Operations    []string `json:"operations"`
	PolicyRefs    []string `json:"policy_refs"`
	ValidFrom     string   `json:"valid_from"`
	ValidUntil    string   `json:"valid_until"`
}

type VerifyAttestationOptions struct {
	TrustedBundleSHA256 string
	AsOf                time.Time
}

type signingProjection Attestation

func SignAttestation(manifest Manifest, template Attestation, privateKey ed25519.PrivateKey) (Attestation, error) {
	if err := VerifyManifestSHA256(manifest); err != nil || len(privateKey) != ed25519.PrivateKeySize {
		return Attestation{}, ErrAttestationInvalid
	}
	out := normalizeAttestation(template)
	out.Schema = AttestationSchemaV1
	out.ArtifactID, out.Revision, out.ManifestSHA256 = manifest.ArtifactID, manifest.Revision, manifest.Integrity.ManifestSHA256
	out.CustodyHeadHash = custodyHeadHash(manifest.Provenance.Custody)
	if out.Federation.SourceManifestSHA256 != out.ManifestSHA256 {
		return Attestation{}, ErrAttestationInvalid
	}
	for _, policy := range out.PolicyRefs {
		if !hasValue(manifest.Policy.PolicyRefs, policy) {
			return Attestation{}, ErrAttestationInvalid
		}
	}
	out.Signature, out.AttestationID = "", ""
	payload, err := attestationPayload(out)
	if err != nil {
		return Attestation{}, err
	}
	out.AttestationID = "attestation:" + textSHA256(string(payload))
	payload, err = attestationPayload(out)
	if err != nil {
		return Attestation{}, err
	}
	out.Signature = base64.RawURLEncoding.EncodeToString(ed25519.Sign(privateKey, payload))
	if err := validateAttestation(out); err != nil {
		return Attestation{}, err
	}
	return out, nil
}

func VerifyAttestation(manifest Manifest, attestation Attestation, bundle TrustBundle, options VerifyAttestationOptions) error {
	if err := VerifyManifestSHA256(manifest); err != nil {
		return ErrAttestationInvalid
	}
	attestation = normalizeAttestation(attestation)
	bundle = normalizeTrustBundle(bundle)
	if err := validateAttestation(attestation); err != nil {
		return err
	}
	idProjection := attestation
	idProjection.AttestationID, idProjection.Signature = "", ""
	idPayload, err := attestationPayload(idProjection)
	if err != nil || attestation.AttestationID != "attestation:"+textSHA256(string(idPayload)) {
		return ErrAttestationInvalid
	}
	if err := validateTrustBundle(bundle); err != nil {
		return err
	}
	digest, err := TrustBundleSHA256(bundle)
	if err != nil || options.TrustedBundleSHA256 == "" || digest != options.TrustedBundleSHA256 {
		return ErrTrustBundleInvalid
	}
	if attestation.ArtifactID != manifest.ArtifactID || attestation.Revision != manifest.Revision || attestation.ManifestSHA256 != manifest.Integrity.ManifestSHA256 || attestation.Federation.SourceManifestSHA256 != manifest.Integrity.ManifestSHA256 || attestation.CustodyHeadHash != custodyHeadHash(manifest.Provenance.Custody) {
		return ErrAttestationInvalid
	}
	for _, policy := range attestation.PolicyRefs {
		if !hasValue(manifest.Policy.PolicyRefs, policy) {
			return ErrAttestationInvalid
		}
	}
	if attestation.IssuerRef != bundle.InstanceRef {
		return ErrKeyUntrusted
	}
	if attestation.Federation.Mode == "origin" && attestation.Federation.OriginRef != attestation.IssuerRef {
		return ErrAttestationInvalid
	}
	key, publicKey, err := trustedKey(attestation, bundle)
	if err != nil {
		return err
	}
	signature, err := base64.RawURLEncoding.DecodeString(attestation.Signature)
	if err != nil || len(signature) != ed25519.SignatureSize {
		return ErrSignatureInvalid
	}
	unsigned := attestation
	unsigned.Signature = ""
	payload, err := attestationPayload(unsigned)
	if err != nil || !ed25519.Verify(publicKey, payload, signature) {
		return ErrSignatureInvalid
	}
	if err := verifyAttestationTime(attestation, key, options.AsOf); err != nil {
		return err
	}
	if err := verifyDelegation(attestation, bundle); err != nil {
		return err
	}
	if attestation.Federation.ConflictRef != "" {
		return ErrFederationConflict
	}
	if attestation.Federation.RetractionRef != "" {
		return ErrEvidenceRetracted
	}
	return nil
}

func TrustBundleSHA256(bundle TrustBundle) (string, error) {
	bundle = normalizeTrustBundle(bundle)
	if err := validateTrustBundle(bundle); err != nil {
		return "", err
	}
	encoded, err := json.Marshal(bundle)
	if err != nil {
		return "", ErrTrustBundleInvalid
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func DecodeAttestation(data []byte) (Attestation, error) {
	var out Attestation
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&out); err != nil {
		return Attestation{}, ErrAttestationInvalid
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return Attestation{}, ErrAttestationInvalid
	}
	return out, nil
}

func attestationPayload(in Attestation) ([]byte, error) {
	unsigned := in
	unsigned.Signature = ""
	encoded, err := json.Marshal(signingProjection(unsigned))
	if err != nil {
		return nil, ErrAttestationInvalid
	}
	return encoded, nil
}

func custodyHeadHash(events []CustodyEvent) string {
	encoded, _ := json.Marshal(events)
	return textSHA256(string(encoded))
}

func normalizeAttestation(in Attestation) Attestation {
	in.DelegationRefs = normalizeSet(in.DelegationRefs)
	in.PolicyRefs = normalizeSet(in.PolicyRefs)
	in.TimeEvidence.SourceRefs = normalizeSet(in.TimeEvidence.SourceRefs)
	in.TimeEvidence.AnchorRefs = normalizeSet(in.TimeEvidence.AnchorRefs)
	return in
}

func normalizeTrustBundle(in TrustBundle) TrustBundle {
	out := in
	out.Keys = append([]TrustKey(nil), in.Keys...)
	sort.Slice(out.Keys, func(i, j int) bool { return out.Keys[i].KeyID < out.Keys[j].KeyID })
	out.Delegations = append([]Delegation(nil), in.Delegations...)
	for i := range out.Delegations {
		out.Delegations[i].Operations = normalizeSet(out.Delegations[i].Operations)
		out.Delegations[i].PolicyRefs = normalizeSet(out.Delegations[i].PolicyRefs)
	}
	sort.Slice(out.Delegations, func(i, j int) bool { return out.Delegations[i].DelegationRef < out.Delegations[j].DelegationRef })
	return out
}

func hasValue(values []string, wanted string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == wanted {
			return true
		}
	}
	return false
}
