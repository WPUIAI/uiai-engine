package evidenceartifact

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// CG-06 — T05 crypto/time/custody independent join producer.
//
// AuthenticityDigestScenario builds the deterministic clause battery and
// RunAuthenticityDigest executes it run-by-run, hashing the canonical clause
// results each run. An independent reviewer replays with
// VerifyAuthenticityDigestReport and compares digests byte-for-byte.
//
// Clauses (matching the amended CG-06 row):
//
//	key_rotation      — retired signing key cannot sign forward; old
//	                    attestations within their window still verify
//	key_revocation    — revoked key rejects post-revocation signatures;
//	                    retroactive revocation invalidates even earlier ones
//	import_distrust   — mirror/import/offline_copy sources require import
//	                    receipt + previous attestation; origin mode must
//	                    carry no import provenance
//	custody_chain     — custody chronology is enforced; reordering or
//	                    duplication of custody events invalidates the seal
//	time_confidence   — unknown confidence, anchorless anchored time, and
//	                    issued/outside-uncertainty attestations are rejected
//	proof_of_absence  — a complete window with an actual JSON asset and no
//	                    omissions assembles; omissions break the absence claim
//	air_gap           — offline materials verify; any remote URL surface in
//	                    manifest/attestation/bundle is rejected first
//	human_identity    — identity proof derives from bundle key bytes and
//	                    rejects any edited reprint

const AuthenticityDigestSchema = "uiai.authenticity_digest_report.v1"

var (
	ErrAuthenticityDigestInvalid = errors.New("authenticity digest report invalid")
	ErrAuthenticityDigestDrift   = errors.New("authenticity digest run divergence")
)

type AuthenticityDigestReport struct {
	Schema  string   `json:"schema"`
	CodeRef string   `json:"code_ref"`
	Runs    int      `json:"runs"`
	Clauses []string `json:"clauses"`

	// DigestSHA256 is the hash of the canonical clause-result JSON shared by
	// every run.
	DigestSHA256 string `json:"digest_sha256"`
	AllIdentical bool   `json:"all_identical"`

	// PerRun hashes (all identical by construction; kept so an independent
	// reviewer can verify run-by-run).
	PerRun []string `json:"per_run"`
}

type authenticityClauseResult struct {
	Clause string `json:"clause"`
	Pass   bool   `json:"pass"`
	Note   string `json:"note"`
}

// AuthenticityDigestClauses is the frozen clause list for the join.
var AuthenticityDigestClauses = []string{
	"key_rotation",
	"key_revocation",
	"import_distrust",
	"custody_chain",
	"time_confidence",
	"proof_of_absence",
	"air_gap",
	"human_identity",
}

type authenticityDigestMaterials struct {
	manifest   Manifest
	template   Attestation
	bundle     TrustBundle
	privateKey ed25519.PrivateKey
	options    VerifyAttestationOptions
}

// AuthenticityDigestFixture builds the deterministic base attestation
// materials (same canonical base as the unit-test fixture, usable from
// non-test code).
func AuthenticityDigestFixture() (Manifest, Attestation, TrustBundle, ed25519.PrivateKey, VerifyAttestationOptions, error) {
	manifest, err := Seal(ReferenceEvidenceManifest())
	if err != nil {
		return Manifest{}, Attestation{}, TrustBundle{}, nil, VerifyAttestationOptions{}, err
	}
	privateKey := ed25519.NewKeyFromSeed(bytesFill(7, ed25519.SeedSize))
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
	options := VerifyAttestationOptions{
		TrustedBundleSHA256: mustDigest(func() ([]byte, error) { return json.Marshal(bundle) }),
		AsOf:                time.Date(2026, 8, 29, 13, 0, 0, 0, time.UTC),
	}
	digest, err := TrustBundleSHA256(bundle)
	if err != nil {
		return Manifest{}, Attestation{}, TrustBundle{}, nil, VerifyAttestationOptions{}, err
	}
	options.TrustedBundleSHA256 = digest
	return manifest, template, bundle, privateKey, options, nil
}

func mustDigest(fn func() ([]byte, error)) string {
	body, err := fn()
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func bytesFill(value byte, count int) []byte {
	out := make([]byte, count)
	for i := range out {
		out[i] = value
	}
	return out
}

// RunAuthenticityDigest executes the clause battery runs times and returns
// the digest report. Any clause failure aborts the run loudly.
func RunAuthenticityDigest(runs int, codeRef string) (AuthenticityDigestReport, error) {
	if runs < 1 || runs > 1000 {
		return AuthenticityDigestReport{}, fmt.Errorf("runs must be 1..1000: %w", ErrAuthenticityDigestInvalid)
	}
	perRun := make([]string, 0, runs)
	for i := 0; i < runs; i++ {
		results, err := runAuthenticityClauses()
		if err != nil {
			return AuthenticityDigestReport{}, fmt.Errorf("clause run %d: %w", i+1, err)
		}
		encoded, err := json.Marshal(results)
		if err != nil {
			return AuthenticityDigestReport{}, fmt.Errorf("canonical run %d: %w", i+1, err)
		}
		sum := sha256.Sum256(encoded)
		perRun = append(perRun, hex.EncodeToString(sum[:]))
	}
	for _, h := range perRun {
		if h != perRun[0] {
			return AuthenticityDigestReport{}, ErrAuthenticityDigestDrift
		}
	}
	return AuthenticityDigestReport{
		Schema:       AuthenticityDigestSchema,
		CodeRef:      codeRef,
		Runs:         runs,
		Clauses:      append([]string(nil), AuthenticityDigestClauses...),
		DigestSHA256: perRun[0],
		AllIdentical: true,
		PerRun:       perRun,
	}, nil
}

// runAuthenticityClauses executes one full pass of the clause battery and
// returns deterministic pass results. Any unexpected outcome is an error.
func runAuthenticityClauses() ([]authenticityClauseResult, error) {
	results := make([]authenticityClauseResult, 0, len(AuthenticityDigestClauses))

	// --- key_rotation ---------------------------------------------------
	manifest, template, bundle, privateKey, options, err := AuthenticityDigestFixture()
	if err != nil {
		return nil, err
	}
	attestationOne, err := SignAttestation(manifest, template, privateKey)
	if err != nil {
		return nil, fmt.Errorf("rotation sign: %w", err)
	}
	publicOne := privateKey.Public().(ed25519.PublicKey)
	privateKeyTwo := ed25519.NewKeyFromSeed(bytesFill(8, ed25519.SeedSize))
	publicTwo := privateKeyTwo.Public().(ed25519.PublicKey)
	rotated := bundle
	rotated.Revision = 2
	rotated.Keys = []TrustKey{
		{KeyID: template.KeyID, Algorithm: AlgorithmEd25519V1, PublicKey: base64.RawURLEncoding.EncodeToString(publicOne), ValidFrom: "2026-01-01T00:00:00Z", ValidUntil: "2026-12-01T00:00:00Z", Status: "retired"},
		{KeyID: "key:2026-02", Algorithm: AlgorithmEd25519V1, PublicKey: base64.RawURLEncoding.EncodeToString(publicTwo), ValidFrom: "2026-08-01T00:00:00Z", ValidUntil: "2027-01-01T00:00:00Z", Status: "active"},
	}
	templateTwo := template
	templateTwo.KeyID = "key:2026-02"
	rotated.Delegations = append(
		append([]Delegation(nil), bundle.Delegations...),
		Delegation{DelegationRef: "delegation:executor-2", ActorRef: templateTwo.ActorRef, InstanceRef: templateTwo.IssuerRef, KeyID: templateTwo.KeyID, Operations: []string{templateTwo.Operation}, PolicyRefs: []string{"policy:evidence"}, ValidFrom: "2026-08-01T00:00:00Z", ValidUntil: "2026-09-01T00:00:00Z"},
	)
	templateTwo.DelegationRefs = []string{"delegation:executor-2"}
	attestationTwo, err := SignAttestation(manifest, templateTwo, privateKeyTwo)
	if err != nil {
		return nil, fmt.Errorf("rotation new-key sign: %w", err)
	}
	rotatedOptions := options
	rotatedDigest, err := TrustBundleSHA256(rotated)
	if err != nil {
		return nil, fmt.Errorf("rotation bundle: %w", err)
	}
	rotatedOptions.TrustedBundleSHA256 = rotatedDigest
	if err := VerifyAttestation(manifest, attestationTwo, rotated, rotatedOptions); err != nil {
		return nil, fmt.Errorf("rotation new-key verify: %w", err)
	}
	if err := VerifyAttestation(manifest, attestationOne, rotated, rotatedOptions); err != nil {
		return nil, fmt.Errorf("rotation old attestation within window: %w", err)
	}
	if err := VerifyAttestation(manifest, attestationTwo, bundle, options); !errors.Is(err, ErrKeyUntrusted) {
		return nil, fmt.Errorf("rotation cross-era verify: %w", err)
	}
	results = append(results, authenticityClauseResult{Clause: "key_rotation", Pass: true})

	// --- key_revocation -------------------------------------------------
	revoked := bundle
	revoked.Revision = 3
	revoked.Keys = []TrustKey{{
		KeyID: template.KeyID, Algorithm: AlgorithmEd25519V1, PublicKey: base64.RawURLEncoding.EncodeToString(publicOne),
		ValidFrom: "2026-01-01T00:00:00Z", ValidUntil: "2027-01-01T00:00:00Z", Status: "revoked", RevokedAt: "2026-09-01T00:00:00Z",
	}}
	revokedDigest, err := TrustBundleSHA256(revoked)
	if err != nil {
		return nil, fmt.Errorf("revocation bundle: %w", err)
	}
	revokedOptions := options
	revokedOptions.TrustedBundleSHA256 = revokedDigest
	if err := VerifyAttestation(manifest, attestationOne, revoked, revokedOptions); err != nil {
		return nil, fmt.Errorf("pre-revocation attestation: %w", err)
	}
	late := template
	late.IssuedAt = "2026-09-02T12:00:01Z"
	late.TimeEvidence.ObservedAt = "2026-09-02T12:00:01Z"
	lateSigned, err := SignAttestation(manifest, late, privateKey)
	if err != nil {
		return nil, fmt.Errorf("revocation late sign: %w", err)
	}
	lateOptions := revokedOptions
	lateOptions.AsOf = time.Date(2026, 9, 2, 13, 0, 0, 0, time.UTC)
	if err := VerifyAttestation(manifest, lateSigned, revoked, lateOptions); !errors.Is(err, ErrKeyRevoked) {
		return nil, fmt.Errorf("post-revocation verify: %w", err)
	}
	retro := revoked
	retro.Keys = append([]TrustKey(nil), revoked.Keys...)
	retro.Keys[0].RetroactiveRevoke = true
	retroDigest, err := TrustBundleSHA256(retro)
	if err != nil {
		return nil, fmt.Errorf("retroactive bundle: %w", err)
	}
	retroOptions := revokedOptions
	retroOptions.TrustedBundleSHA256 = retroDigest
	if err := VerifyAttestation(manifest, attestationOne, retro, retroOptions); !errors.Is(err, ErrKeyRevoked) {
		return nil, fmt.Errorf("retroactive revocation verify: %w", err)
	}
	results = append(results, authenticityClauseResult{Clause: "key_revocation", Pass: true})

	// --- import_distrust ------------------------------------------------
	imported := template
	imported.Federation = FederationState{
		Mode: "import", OriginRef: "instance:peer", SourceManifestSHA256: manifest.Integrity.ManifestSHA256,
		ImportedAt: "2026-08-30T12:00:00Z", ImportReceiptRef: "receipt:import-001", PreviousAttestationRef: attestationOne.AttestationID,
		SourceAvailability: "lost",
	}
	importedSigned, err := SignAttestation(manifest, imported, privateKey)
	if err != nil {
		return nil, fmt.Errorf("import sign: %w", err)
	}
	if err := VerifyAttestation(manifest, importedSigned, bundle, options); err != nil {
		return nil, fmt.Errorf("import verify: %w", err)
	}
	noReceipt := imported
	noReceipt.Federation.ImportReceiptRef = ""
	if _, err := SignAttestation(manifest, noReceipt, privateKey); !errors.Is(err, ErrAttestationInvalid) {
		return nil, fmt.Errorf("import no-receipt sign: %w", err)
	}
	pollutedOrigin := template
	pollutedOrigin.Federation.ImportedAt = "2026-08-30T12:00:00Z"
	pollutedOriginSigned, err := SignAttestation(manifest, pollutedOrigin, privateKey)
	if err == nil {
		if err := VerifyAttestation(manifest, pollutedOriginSigned, bundle, options); !errors.Is(err, ErrAttestationInvalid) {
			return nil, fmt.Errorf("origin with import provenance: %w", err)
		}
	} else if !errors.Is(err, ErrAttestationInvalid) {
		return nil, fmt.Errorf("origin with import provenance sign: %w", err)
	}
	results = append(results, authenticityClauseResult{Clause: "import_distrust", Pass: true})

	// --- custody_chain ---------------------------------------------------
	if err := Validate(manifest); err != nil {
		return nil, fmt.Errorf("custody sealed manifest: %w", err)
	}
	reordered := manifest
	reordered.Provenance.Custody = append(append([]CustodyEvent(nil), reordered.Provenance.Custody...), CustodyEvent{EventID: "custody:2", Action: "transferred", ActorRef: "agent:executor", InstanceRef: "instance:uiai", InputRefs: []string{"asset:proof"}, OutputRefs: []string{"asset:proof"}, OccurredAt: "2026-08-28T12:00:00Z"})
	if err := Validate(reordered); !errors.Is(err, ErrInvalidIntegrity) {
		return nil, fmt.Errorf("custody chronology validate: %w", err)
	}
	duplicated := manifest
	duplicated.Provenance.Custody = append(append([]CustodyEvent(nil), duplicated.Provenance.Custody...), CustodyEvent{EventID: "custody:1", Action: "transferred", ActorRef: "agent:executor", InstanceRef: "instance:uiai", InputRefs: []string{"asset:proof"}, OutputRefs: []string{"asset:proof"}, OccurredAt: "2026-08-29T12:00:05Z"})
	if err := Validate(duplicated); !errors.Is(err, ErrInvalidIntegrity) {
		return nil, fmt.Errorf("custody duplicate validate: %w", err)
	}
	results = append(results, authenticityClauseResult{Clause: "custody_chain", Pass: true})

	// --- time_confidence -------------------------------------------------
	unknownTime := template
	unknownTime.TimeEvidence.Confidence = "unknown"
	if _, err := SignAttestation(manifest, unknownTime, privateKey); !errors.Is(err, ErrTimeUntrusted) {
		return nil, fmt.Errorf("time unknown sign: %w", err)
	}
	anchorless := template
	anchorless.TimeEvidence.AnchorRefs = nil
	if _, err := SignAttestation(manifest, anchorless, privateKey); !errors.Is(err, ErrTimeUntrusted) {
		return nil, fmt.Errorf("time anchorless sign: %w", err)
	}
	drifted := template
	drifted.IssuedAt = "2026-08-29T18:00:01Z"
	drifted.TimeEvidence.ObservedAt = "2026-08-29T12:00:01Z"
	driftedSigned, err := SignAttestation(manifest, drifted, privateKey)
	if err != nil {
		return nil, fmt.Errorf("time drift sign: %w", err)
	}
	if err := VerifyAttestation(manifest, driftedSigned, bundle, options); !errors.Is(err, ErrTimeUntrusted) {
		return nil, fmt.Errorf("time drift verify: %w", err)
	}
	results = append(results, authenticityClauseResult{Clause: "time_confidence", Pass: true})

	// --- proof_of_absence ------------------------------------------------
	scenario, err := CaptureDigestScenario()
	if err != nil {
		return nil, fmt.Errorf("absence scenario: %w", err)
	}
	absent := scenario
	omittedRefs := make(map[string]struct{}, len(absent.Omissions))
	for _, omission := range absent.Omissions {
		omittedRefs[omission.ObservationRef] = struct{}{}
	}
	kept := make([]CaptureObservation, 0, len(absent.Observations))
	for _, observation := range absent.Observations {
		if _, wasOmitted := omittedRefs[observation.ObservationRef]; !wasOmitted {
			kept = append(kept, observation)
		}
	}
	for i := range kept {
		kept[i].Ordinal = uint64(i)
	}
	absent.Observations = kept
	absent.Omissions = nil
	for i := range absent.Requirements {
		if absent.Requirements[i].ClaimID == "claim:manifest-valid" {
			absent.Requirements[i].Kind = RequirementProofOfAbsence
		}
	}
	if _, err := AssembleCapture(absent); err != nil {
		return nil, fmt.Errorf("absence assemble: %w", err)
	}
	withOmission := absent
	var blockedRef string
	var blockedIndex int
	for i, observation := range withOmission.Observations {
		if observation.AssetID != "" {
			blockedRef, blockedIndex = observation.ObservationRef, i
			break
		}
	}
	if blockedRef == "" {
		return nil, errors.New("absence negative scenario: no observation to block")
	}
	withOmission.Observations[blockedIndex].AssetID = ""
	withOmission.Omissions = []CaptureOmission{{
		ObservationRef: blockedRef, Reason: "unavailable", PolicyRef: "omission:evidence-1",
	}}
	if _, err := AssembleCapture(withOmission); !errors.Is(err, ErrCoverageIncomplete) {
		return nil, fmt.Errorf("absence with omission: %w", err)
	}
	results = append(results, authenticityClauseResult{Clause: "proof_of_absence", Pass: true})

	// --- air_gap ---------------------------------------------------------
	if err := ValidateAirGapMaterials(manifest, attestationOne, bundle, options); err != nil {
		return nil, fmt.Errorf("air-gap clean verify: %w", err)
	}
	contaminated := manifest
	contaminated.Integrity.ManifestSHA256 = ""
	contaminated.Title = "Evidence artifact contract proof https://remote.example/manifest"
	sealedContaminated, err := Seal(contaminated)
	if err != nil {
		return nil, fmt.Errorf("air-gap contaminated seal: %w", err)
	}
	if err := ValidateAirGapMaterials(sealedContaminated, attestationOne, bundle, options); !errors.Is(err, ErrAirGapContaminated) {
		return nil, fmt.Errorf("air-gap contaminated verify: %w", err)
	}
	results = append(results, authenticityClauseResult{Clause: "air_gap", Pass: true})

	// --- human_identity --------------------------------------------------
	proof, err := HumanIdentityProofFor(bundle, template.KeyID)
	if err != nil {
		return nil, fmt.Errorf("human identity derive: %w", err)
	}
	if err := VerifyHumanIdentityProof(bundle, proof); err != nil {
		return nil, fmt.Errorf("human identity verify: %w", err)
	}
	tampered := proof
	tampered.Fingerprint = flipHex(tampered.Fingerprint)
	if err := VerifyHumanIdentityProof(bundle, tampered); !errors.Is(err, ErrHumanIdentityProofInvalid) {
		return nil, fmt.Errorf("human identity tampered: %w", err)
	}
	results = append(results, authenticityClauseResult{Clause: "human_identity", Pass: true})

	return results, nil
}

func flipHex(in string) string {
	if len(in) == 0 {
		return in
	}
	if in[0] == '0' {
		return "1" + in[1:]
	}
	return "0" + in[1:]
}

// Validate checks report internal consistency for independent verification.
func (r AuthenticityDigestReport) Validate() error {
	if r.Schema != AuthenticityDigestSchema || r.Runs < 1 || r.Runs > 1000 || len(r.PerRun) != r.Runs || !r.AllIdentical {
		return ErrAuthenticityDigestInvalid
	}
	for _, clause := range AuthenticityDigestClauses {
		found := false
		for _, listed := range r.Clauses {
			if listed == clause {
				found = true
				break
			}
		}
		if !found {
			return ErrAuthenticityDigestInvalid
		}
	}
	if len(r.Clauses) != len(AuthenticityDigestClauses) {
		return ErrAuthenticityDigestInvalid
	}
	for _, run := range r.PerRun {
		if !validSHA256(run) {
			return ErrAuthenticityDigestInvalid
		}
	}
	if r.DigestSHA256 != r.PerRun[0] {
		return ErrAuthenticityDigestInvalid
	}
	return nil
}

// VerifyAuthenticityDigestReport replays the clause battery and compares
// digests byte-for-byte against a prior report.
func VerifyAuthenticityDigestReport(prior AuthenticityDigestReport, runs int) error {
	if err := prior.Validate(); err != nil {
		return err
	}
	fresh, err := RunAuthenticityDigest(runs, prior.CodeRef)
	if err != nil {
		return err
	}
	if fresh.DigestSHA256 != prior.DigestSHA256 {
		return fmt.Errorf("digest mismatch: %w", ErrAuthenticityDigestDrift)
	}
	return nil
}
