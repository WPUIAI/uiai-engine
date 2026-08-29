package evidenceartifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

const StrictSecurityPolicyV1 = "uiai.evidence_security.strict.v1"

var (
	ErrInspectionRequired    = errors.New("evidence asset inspection required")
	ErrInspectionFailed      = errors.New("evidence asset inspection failed")
	ErrMediaTypeMismatch     = errors.New("evidence asset media type mismatch")
	ErrUnsafeContent         = errors.New("evidence asset unsafe content")
	ErrSensitiveContent      = errors.New("evidence asset sensitive content")
	ErrSanitizerRequired     = errors.New("evidence asset sanitizer required")
	ErrInspectionUnavailable = errors.New("evidence asset inspection unavailable")
)

type InspectionStatus string

const (
	InspectionPassed             InspectionStatus = "passed"
	InspectionPassedWithFindings InspectionStatus = "passed_with_findings"
)

const (
	FindingActiveTextUntrusted = "active_text_untrusted"
	FindingPIIDetected         = "pii_detected"
)

type InspectionRecord struct {
	AssetID           string           `json:"asset_id"`
	PolicyRef         string           `json:"policy_ref"`
	Status            InspectionStatus `json:"status"`
	ObservedMediaType string           `json:"observed_media_type"`
	FindingCodes      []string         `json:"finding_codes"`
	InspectionSHA256  string           `json:"inspection_sha256"`
}

type InspectionRequest struct {
	Path     string
	Asset    Asset
	Security Security
	Policy   Policy
}

type AssetInspector interface {
	Inspect(context.Context, InspectionRequest) (InspectionRecord, error)
}

type BuiltinInspector struct {
	MaxTextBytes    int64
	MaxInspectBytes int64
}

func NewBuiltinInspector() *BuiltinInspector {
	return &BuiltinInspector{MaxTextBytes: 4 * 1024 * 1024, MaxInspectBytes: 32 * 1024 * 1024}
}

func validateSecurity(s Security) error {
	if !validRef(s.PolicyRef, true) {
		return invalid(ErrInvalidPolicy, "security policy ref")
	}
	for _, refs := range [][]string{s.InspectionReceiptRefs, s.SanitizationRefs, s.RedactionRefs} {
		if err := validateRefs(refs, 0, MaxRefsPerList); err != nil {
			return invalid(ErrInvalidPolicy, "security refs")
		}
	}
	return nil
}

func textSHA256(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}

func inspectionDigest(record InspectionRecord, asset Asset) string {
	parts := []string{record.AssetID, record.PolicyRef, string(record.Status), record.ObservedMediaType, asset.SHA256, uintString(uint64(asset.ByteSize))}
	parts = append(parts, record.FindingCodes...)
	return deterministicID(parts...)
}

func validInspectionRecord(record InspectionRecord) bool {
	if !validRef(record.AssetID, true) || record.PolicyRef != StrictSecurityPolicyV1 || !validSHA256(record.InspectionSHA256) || record.ObservedMediaType == "" {
		return false
	}
	if record.Status != InspectionPassed && record.Status != InspectionPassedWithFindings {
		return false
	}
	for index, code := range record.FindingCodes {
		if code != FindingActiveTextUntrusted && code != FindingPIIDetected {
			return false
		}
		if index > 0 && code <= record.FindingCodes[index-1] {
			return false
		}
	}
	return (record.Status == InspectionPassed) == (len(record.FindingCodes) == 0)
}

func equalInspection(left, right InspectionRecord) bool {
	if left.AssetID != right.AssetID || left.PolicyRef != right.PolicyRef || left.Status != right.Status || left.ObservedMediaType != right.ObservedMediaType || left.InspectionSHA256 != right.InspectionSHA256 || len(left.FindingCodes) != len(right.FindingCodes) {
		return false
	}
	for index := range left.FindingCodes {
		if left.FindingCodes[index] != right.FindingCodes[index] {
			return false
		}
	}
	return true
}
