package evidenceartifact

import (
	"crypto/ed25519"
	"encoding/base64"
	"time"
)

func validateAttestation(in Attestation) error {
	if in.Schema != AttestationSchemaV1 || !validRef(in.AttestationID, true) || !validRef(in.ArtifactID, true) || in.Revision == 0 || !validSHA256(in.ManifestSHA256) || !validRef(in.IssuerRef, true) || !validRef(in.ActorRef, true) || !validRef(in.Operation, true) || !validRef(in.KeyID, true) || in.Algorithm != AlgorithmEd25519V1 || !validSHA256(in.CustodyHeadHash) {
		return ErrAttestationInvalid
	}
	if _, ok := canonicalTime(in.IssuedAt, true); !ok {
		return ErrAttestationInvalid
	}
	if err := validateRefs(in.DelegationRefs, 0, MaxRefsPerList); err != nil {
		return ErrDelegationInvalid
	}
	if err := validateRefs(in.PolicyRefs, 1, MaxRefsPerList); err != nil {
		return ErrAttestationInvalid
	}
	if err := validateTimeEvidence(in.TimeEvidence); err != nil {
		return err
	}
	if err := validateFederation(in.Federation); err != nil {
		return err
	}
	if _, err := base64.RawURLEncoding.DecodeString(in.Signature); err != nil {
		return ErrSignatureInvalid
	}
	return nil
}

func validateTimeEvidence(in TimeEvidence) error {
	observed, ok := canonicalTime(in.ObservedAt, true)
	if !ok || observed.IsZero() || in.UncertaintyMS > uint64((24*time.Hour)/time.Millisecond) {
		return ErrTimeUntrusted
	}
	if err := validateRefs(in.SourceRefs, 1, MaxRefsPerList); err != nil {
		return ErrTimeUntrusted
	}
	if err := validateRefs(in.AnchorRefs, 0, MaxRefsPerList); err != nil {
		return ErrTimeUntrusted
	}
	switch in.Confidence {
	case "anchored", "source_authenticated":
		if len(in.AnchorRefs) == 0 {
			return ErrTimeUntrusted
		}
	case "local_clock":
	case "unknown":
		return ErrTimeUntrusted
	default:
		return ErrTimeUntrusted
	}
	return nil
}

func validateFederation(in FederationState) error {
	if !validRef(in.OriginRef, true) || !validSHA256(in.SourceManifestSHA256) {
		return ErrAttestationInvalid
	}
	if !validRef(in.PreviousAttestationRef, false) || !validRef(in.CorrectionRef, false) || !validRef(in.RetractionRef, false) || !validRef(in.ConflictRef, false) {
		return ErrAttestationInvalid
	}
	switch in.SourceAvailability {
	case "available", "lost", "unknown":
	default:
		return ErrAttestationInvalid
	}
	switch in.Mode {
	case "origin":
		if in.ImportedAt != "" || in.ImportReceiptRef != "" || in.PreviousAttestationRef != "" {
			return ErrAttestationInvalid
		}
	case "mirror", "import", "offline_copy":
		if _, ok := canonicalTime(in.ImportedAt, true); !ok || !validRef(in.ImportReceiptRef, true) || !validRef(in.PreviousAttestationRef, true) {
			return ErrAttestationInvalid
		}
	default:
		return ErrAttestationInvalid
	}
	return nil
}

func validateTrustBundle(bundle TrustBundle) error {
	if bundle.Schema != TrustBundleSchemaV1 || !validRef(bundle.InstanceRef, true) || !validRef(bundle.AuthorityRef, true) || bundle.Revision == 0 || len(bundle.Keys) == 0 || len(bundle.Keys) > 128 || len(bundle.Delegations) > 512 {
		return ErrTrustBundleInvalid
	}
	seenKeys := make(map[string]struct{}, len(bundle.Keys))
	for _, key := range bundle.Keys {
		if !validRef(key.KeyID, true) || key.Algorithm != AlgorithmEd25519V1 {
			return ErrTrustBundleInvalid
		}
		if _, duplicate := seenKeys[key.KeyID]; duplicate {
			return ErrTrustBundleInvalid
		}
		seenKeys[key.KeyID] = struct{}{}
		decoded, err := base64.RawURLEncoding.DecodeString(key.PublicKey)
		if err != nil || len(decoded) != ed25519.PublicKeySize {
			return ErrTrustBundleInvalid
		}
		from, fromOK := canonicalTime(key.ValidFrom, true)
		until, untilOK := canonicalTime(key.ValidUntil, true)
		if !fromOK || !untilOK || !from.Before(until) {
			return ErrTrustBundleInvalid
		}
		switch key.Status {
		case "active", "retired":
			if key.RevokedAt != "" || key.CompromisedAt != "" || key.RetroactiveRevoke {
				return ErrTrustBundleInvalid
			}
		case "revoked":
			if _, ok := canonicalTime(key.RevokedAt, true); !ok || key.CompromisedAt != "" {
				return ErrTrustBundleInvalid
			}
		case "compromised":
			if _, ok := canonicalTime(key.CompromisedAt, true); !ok || key.RevokedAt != "" {
				return ErrTrustBundleInvalid
			}
		default:
			return ErrTrustBundleInvalid
		}
	}
	seenDelegations := make(map[string]struct{}, len(bundle.Delegations))
	for _, delegation := range bundle.Delegations {
		if !validRef(delegation.DelegationRef, true) || !validRef(delegation.ActorRef, true) || delegation.InstanceRef != bundle.InstanceRef || !validRef(delegation.KeyID, true) || len(delegation.Operations) == 0 || len(delegation.PolicyRefs) == 0 {
			return ErrTrustBundleInvalid
		}
		if _, duplicate := seenDelegations[delegation.DelegationRef]; duplicate {
			return ErrTrustBundleInvalid
		}
		seenDelegations[delegation.DelegationRef] = struct{}{}
		if _, exists := seenKeys[delegation.KeyID]; !exists {
			return ErrTrustBundleInvalid
		}
		from, fromOK := canonicalTime(delegation.ValidFrom, true)
		until, untilOK := canonicalTime(delegation.ValidUntil, true)
		if !fromOK || !untilOK || !from.Before(until) {
			return ErrTrustBundleInvalid
		}
		if err := validateRefs(delegation.Operations, 1, MaxRefsPerList); err != nil {
			return ErrTrustBundleInvalid
		}
		if err := validateRefs(delegation.PolicyRefs, 1, MaxRefsPerList); err != nil {
			return ErrTrustBundleInvalid
		}
	}
	return nil
}

func trustedKey(attestation Attestation, bundle TrustBundle) (TrustKey, ed25519.PublicKey, error) {
	for _, key := range bundle.Keys {
		if key.KeyID != attestation.KeyID || key.Algorithm != attestation.Algorithm {
			continue
		}
		decoded, err := base64.RawURLEncoding.DecodeString(key.PublicKey)
		if err != nil || len(decoded) != ed25519.PublicKeySize {
			return TrustKey{}, nil, ErrKeyUntrusted
		}
		return key, ed25519.PublicKey(decoded), nil
	}
	return TrustKey{}, nil, ErrKeyUntrusted
}

func verifyAttestationTime(attestation Attestation, key TrustKey, asOf time.Time) error {
	issued, _ := canonicalTime(attestation.IssuedAt, true)
	observed, _ := canonicalTime(attestation.TimeEvidence.ObservedAt, true)
	uncertainty := time.Duration(attestation.TimeEvidence.UncertaintyMS) * time.Millisecond
	if issued.Before(observed.Add(-uncertainty)) || issued.After(observed.Add(uncertainty)) || asOf.IsZero() || asOf.Before(issued) {
		return ErrTimeUntrusted
	}
	from, _ := canonicalTime(key.ValidFrom, true)
	until, _ := canonicalTime(key.ValidUntil, true)
	if issued.Before(from) || !issued.Before(until) {
		return ErrKeyUntrusted
	}
	if key.Status == "revoked" {
		revoked, _ := canonicalTime(key.RevokedAt, true)
		if key.RetroactiveRevoke || !issued.Before(revoked) {
			return ErrKeyRevoked
		}
	}
	if key.Status == "compromised" {
		compromised, _ := canonicalTime(key.CompromisedAt, true)
		if key.RetroactiveRevoke || !issued.Before(compromised) {
			return ErrKeyRevoked
		}
	}
	return nil
}

func verifyDelegation(attestation Attestation, bundle TrustBundle) error {
	if attestation.ActorRef == attestation.IssuerRef && len(attestation.DelegationRefs) == 0 {
		return nil
	}
	if len(attestation.DelegationRefs) == 0 {
		return ErrDelegationInvalid
	}
	issued, _ := canonicalTime(attestation.IssuedAt, true)
	for _, ref := range attestation.DelegationRefs {
		matched := false
		for _, delegation := range bundle.Delegations {
			if delegation.DelegationRef != ref || delegation.ActorRef != attestation.ActorRef || delegation.KeyID != attestation.KeyID || delegation.InstanceRef != attestation.IssuerRef || !hasValue(delegation.Operations, attestation.Operation) {
				continue
			}
			from, _ := canonicalTime(delegation.ValidFrom, true)
			until, _ := canonicalTime(delegation.ValidUntil, true)
			if issued.Before(from) || !issued.Before(until) {
				continue
			}
			matched = true
			for _, policy := range attestation.PolicyRefs {
				if !hasValue(delegation.PolicyRefs, policy) {
					matched = false
				}
			}
		}
		if !matched {
			return ErrDelegationInvalid
		}
	}
	return nil
}
