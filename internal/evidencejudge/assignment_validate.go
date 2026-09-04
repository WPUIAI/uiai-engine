package evidencejudge

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

func ValidateJudgeCapabilityAt(capability JudgeCapability, now time.Time) error {
	if err := validateJudgeCapabilityShape(capability, true); err != nil {
		return err
	}
	if err := VerifyJudgeCapabilityDigest(capability); err != nil {
		return err
	}
	if capability.Status != CapabilityEligible {
		return ErrJudgeCandidateUnavailable
	}
	if now.Before(capability.ValidFrom) || !now.Before(capability.ValidUntil) {
		return ErrJudgeExpired
	}
	if capability.Calibration.Status != CalibrationPassed || !now.Before(capability.Calibration.ValidUntil) {
		return ErrJudgeCalibrationInvalid
	}
	return nil
}

func ValidateJudgeAssignmentRequest(request JudgeAssignmentRequest) error {
	if request.Schema != JudgeAssignmentRequestSchema || !validAssignmentRef(request.RequestID) ||
		!validAssignmentRef(request.IdempotencyRef) || !validAssignmentArtifact(request.Artifact) ||
		!validAssignmentRef(request.ViewRef) || !validAssignmentSHA256(request.ViewSHA256) ||
		!validAssignmentRef(request.InformationSetRef) || !validAssignmentSHA256(request.InformationSetSHA256) ||
		!validAssignmentRef(request.ExecutorIdentityRef) || !validAssignmentRef(request.ExecutorPrincipalRef) ||
		!validAssignmentRef(request.ExecutorInstanceRef) || !validAssignmentRef(request.VerifierPolicyRef) ||
		!validAssignmentRef(request.VerifierPolicyRevision) || !validAssignmentRef(request.ResultDetail) ||
		!validAssignmentRef(request.AssignmentAuthorityRef) || request.RequestedAt.IsZero() ||
		!request.ExpiresAt.After(request.RequestedAt) || request.LeaseDurationMS == 0 ||
		request.LeaseDurationMS > MaxAssignmentLeaseMillis || request.MaxCandidates == 0 ||
		request.MaxCandidates > MaxJudgeCandidates || len(request.RequiredModalities) == 0 ||
		len(request.RequiredModalities) > MaxModalities || assignmentHasDuplicateModalities(request.RequiredModalities) ||
		assignmentHasBadRefs(request.ExecutorCorrelationRefs, MaxAssignmentRefs, true) ||
		assignmentHasBadRefs(request.DisallowedCorrelationRefs, MaxAssignmentRefs, true) ||
		!validAssignmentBudget(request.Budget) {
		return ErrJudgeAssignmentInvalid
	}
	leaseExpires := request.RequestedAt.Add(time.Duration(request.LeaseDurationMS) * time.Millisecond)
	if leaseExpires.After(request.ExpiresAt) {
		return ErrJudgeExpired
	}
	if !request.AutonomousEligible && !request.HumanAuthorityRequired {
		return ErrJudgeAssignmentInvalid
	}
	body, err := json.Marshal(request)
	if err != nil || len(body) > MaxAssignmentBytes {
		return ErrJudgeBudgetExceeded
	}
	return nil
}

func ValidateJudgeAssignmentAt(assignment JudgeAssignment, request JudgeAssignmentRequest, capability JudgeCapability, now time.Time) error {
	if err := ValidateJudgeAssignmentRequest(request); err != nil {
		return err
	}
	if err := ValidateJudgeCapabilityAt(capability, now); err != nil {
		return err
	}
	if err := validateJudgeAssignmentShape(assignment, true); err != nil {
		return err
	}
	if assignment.Status != AssignmentAppointed {
		return ErrJudgeAssignmentMismatch
	}
	if err := VerifyJudgeAssignmentDigest(assignment); err != nil {
		return err
	}
	requestDigest, err := ComputeJudgeAssignmentRequestDigest(request)
	if err != nil {
		return err
	}
	if assignment.RequestID != request.RequestID || assignment.RequestSHA256 != requestDigest ||
		!reflect.DeepEqual(assignment.Artifact, request.Artifact) || assignment.ViewRef != request.ViewRef ||
		assignment.ViewSHA256 != request.ViewSHA256 || assignment.InformationSetRef != request.InformationSetRef ||
		assignment.InformationSetSHA256 != request.InformationSetSHA256 || assignment.JudgeIdentityRef != capability.JudgeIdentityRef ||
		assignment.JudgePrincipalRef != capability.PrincipalRef || assignment.JudgeInstanceRef != capability.InstanceRef ||
		assignment.CapabilityDigest != capability.CapabilityDigest || assignment.VerifierPolicyRef != request.VerifierPolicyRef ||
		assignment.VerifierPolicyRevision != request.VerifierPolicyRevision ||
		assignment.AssignmentAuthorityRef != request.AssignmentAuthorityRef || !reflect.DeepEqual(assignment.Budget, request.Budget) {
		return ErrJudgeAssignmentMismatch
	}
	if now.Before(assignment.Lease.IssuedAt) || !now.Before(assignment.Lease.ExpiresAt) ||
		assignment.Lease.ExpiresAt.After(request.ExpiresAt) || assignment.Lease.ExpiresAt.After(capability.ValidUntil) ||
		assignment.Lease.ExpiresAt.After(capability.Calibration.ValidUntil) {
		return ErrJudgeExpired
	}
	return nil
}

func validateJudgeCapabilityShape(capability JudgeCapability, requireDigest bool) error {
	if capability.Schema != JudgeCapabilitySchema ||
		(requireDigest && !validAssignmentSHA256(capability.CapabilityDigest)) ||
		(!requireDigest && capability.CapabilityDigest != "" && !validAssignmentSHA256(capability.CapabilityDigest)) ||
		!validAssignmentRef(capability.JudgeIdentityRef) || !validAssignmentRef(capability.PrincipalRef) ||
		!validAssignmentRef(capability.InstanceRef) || !validAssignmentRef(capability.HarnessRef) ||
		!validAssignmentRef(capability.ProviderRef) || !validAssignmentRef(capability.ModelRef) ||
		len(capability.SupportedModalities) == 0 || len(capability.SupportedModalities) > MaxModalities ||
		assignmentHasDuplicateModalities(capability.SupportedModalities) ||
		assignmentHasBadRefs(capability.SupportedResultDetail, MaxAssignmentRefs, false) ||
		assignmentHasBadRefs(capability.IndependenceRefs, MaxAssignmentRefs, false) ||
		assignmentHasBadRefs(capability.CorrelationRefs, MaxAssignmentRefs, true) ||
		assignmentHasBadRefs(capability.PolicyRefs, MaxAssignmentRefs, false) ||
		(capability.Status != CapabilityEligible && capability.Status != CapabilityBlocked && capability.Status != CapabilityRevoked) ||
		!validAssignmentBudget(capability.MaxBudget) || capability.ValidFrom.IsZero() ||
		!capability.ValidUntil.After(capability.ValidFrom) || !validAssignmentCalibration(capability.Calibration) ||
		!capability.Calibration.ValidUntil.After(capability.ValidFrom) {
		return ErrJudgeCapabilityInvalid
	}
	body, err := json.Marshal(capability)
	if err != nil || len(body) > MaxAssignmentBytes {
		return ErrJudgeBudgetExceeded
	}
	return nil
}

func validateJudgeAssignmentShape(assignment JudgeAssignment, requireDigest bool) error {
	if assignment.Schema != JudgeAssignmentSchema || !validAssignmentRef(assignment.AssignmentID) ||
		(requireDigest && !validAssignmentSHA256(assignment.AssignmentSHA256)) ||
		(!requireDigest && assignment.AssignmentSHA256 != "" && !validAssignmentSHA256(assignment.AssignmentSHA256)) ||
		!validAssignmentRef(assignment.RequestID) || !validAssignmentSHA256(assignment.RequestSHA256) ||
		!validAssignmentArtifact(assignment.Artifact) || !validAssignmentRef(assignment.ViewRef) ||
		!validAssignmentSHA256(assignment.ViewSHA256) || !validAssignmentRef(assignment.InformationSetRef) ||
		!validAssignmentSHA256(assignment.InformationSetSHA256) || !validAssignmentRef(assignment.VerifierPolicyRef) ||
		!validAssignmentRef(assignment.VerifierPolicyRevision) || !validAssignmentRef(assignment.AssignmentAuthorityRef) ||
		!validAssignmentRef(assignment.SelectionReason) || (assignment.Status != AssignmentAppointed &&
		assignment.Status != AssignmentBlocked && assignment.Status != AssignmentAutonomousIneligible) {
		return ErrJudgeAssignmentInvalid
	}
	if assignment.Status == AssignmentAppointed {
		if !validAssignmentRef(assignment.JudgeIdentityRef) || !validAssignmentRef(assignment.JudgePrincipalRef) ||
			!validAssignmentRef(assignment.JudgeInstanceRef) || !validAssignmentSHA256(assignment.CapabilityDigest) ||
			!validAssignmentRef(assignment.Lease.LeaseID) || assignment.Lease.Generation == 0 || assignment.Lease.IssuedAt.IsZero() ||
			!assignment.Lease.ExpiresAt.After(assignment.Lease.IssuedAt) || !validAssignmentBudget(assignment.Budget) ||
			assignmentHasBadRefs(assignment.IndependenceEvidenceRefs, MaxAssignmentRefs, false) ||
			assignmentHasBadRefs(assignment.CalibrationRefs, MaxAssignmentRefs, false) {
			return ErrJudgeAssignmentInvalid
		}
	} else if assignment.JudgeIdentityRef != "" || assignment.CapabilityDigest != "" || assignment.Lease.Generation != 0 {
		return ErrJudgeAssignmentInvalid
	}
	body, err := json.Marshal(assignment)
	if err != nil || len(body) > MaxAssignmentBytes {
		return ErrJudgeBudgetExceeded
	}
	return nil
}

func validAssignmentCalibration(calibration JudgeCalibration) bool {
	return validAssignmentRef(calibration.CorpusRef) && validAssignmentRef(calibration.ResultRef) &&
		validAssignmentRef(calibration.Revision) && (calibration.Status == CalibrationPassed ||
		calibration.Status == CalibrationFailed || calibration.Status == CalibrationExpired) && !calibration.ValidUntil.IsZero()
}

func validAssignmentArtifact(artifact ArtifactBinding) bool {
	return validAssignmentRef(artifact.ArtifactRef) && artifact.Revision > 0 &&
		validAssignmentSHA256(artifact.BundleSHA256) && validAssignmentSHA256(artifact.ManifestSHA256) &&
		validAssignmentRef(artifact.Scope.ProjectRef) && validAssignmentRef(artifact.Scope.WorkstreamRef) &&
		validAssignmentRef(artifact.Scope.WorksetRef) && validAssignmentRef(artifact.Scope.CallGraphRef) &&
		validAssignmentRef(artifact.Scope.WorkpointRef) && validAssignmentRef(artifact.Scope.WorkItemRef) &&
		!assignmentHasBadRefs(artifact.AttestationRefs, MaxAssignmentRefs, true) &&
		!assignmentHasBadRefs(artifact.TrustRefs, MaxAssignmentRefs, true) &&
		!assignmentHasBadRefs(artifact.SecurityRefs, MaxAssignmentRefs, true)
}

func validAssignmentBudget(budget JudgeBudget) bool {
	return budget.MaxTokens > 0 && budget.MaxDurationMS > 0
}

func assignmentHasBadRefs(values []string, max int, allowEmpty bool) bool {
	if (!allowEmpty && len(values) == 0) || len(values) > max {
		return true
	}
	seen := map[string]struct{}{}
	for _, value := range values {
		if !validAssignmentRef(value) {
			return true
		}
		if _, exists := seen[value]; exists {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}

func assignmentHasDuplicateModalities(values []Modality) bool {
	seen := map[Modality]struct{}{}
	for _, value := range values {
		if !validModality(value) {
			return true
		}
		if _, exists := seen[value]; exists {
			return true
		}
		seen[value] = struct{}{}
	}
	return false
}

func normalizeJudgeCapability(capability JudgeCapability) JudgeCapability {
	body, _ := json.Marshal(capability)
	var out JudgeCapability
	_ = json.Unmarshal(body, &out)
	sort.Slice(out.SupportedModalities, func(i, j int) bool { return out.SupportedModalities[i] < out.SupportedModalities[j] })
	out.SupportedResultDetail = sortedAssignmentStrings(out.SupportedResultDetail)
	out.IndependenceRefs = sortedAssignmentStrings(out.IndependenceRefs)
	out.CorrelationRefs = sortedAssignmentStrings(out.CorrelationRefs)
	out.PolicyRefs = sortedAssignmentStrings(out.PolicyRefs)
	return out
}

func normalizeJudgeAssignmentRequest(request JudgeAssignmentRequest) JudgeAssignmentRequest {
	body, _ := json.Marshal(request)
	var out JudgeAssignmentRequest
	_ = json.Unmarshal(body, &out)
	sort.Slice(out.RequiredModalities, func(i, j int) bool { return out.RequiredModalities[i] < out.RequiredModalities[j] })
	out.ExecutorCorrelationRefs = sortedAssignmentStrings(out.ExecutorCorrelationRefs)
	out.DisallowedCorrelationRefs = sortedAssignmentStrings(out.DisallowedCorrelationRefs)
	normalizeAssignmentArtifact(&out.Artifact)
	return out
}

func normalizeJudgeAssignment(assignment JudgeAssignment) JudgeAssignment {
	body, _ := json.Marshal(assignment)
	var out JudgeAssignment
	_ = json.Unmarshal(body, &out)
	normalizeAssignmentArtifact(&out.Artifact)
	out.IndependenceEvidenceRefs = sortedAssignmentStrings(out.IndependenceEvidenceRefs)
	out.CalibrationRefs = sortedAssignmentStrings(out.CalibrationRefs)
	return out
}

func normalizeAssignmentArtifact(artifact *ArtifactBinding) {
	artifact.AttestationRefs = sortedAssignmentStrings(artifact.AttestationRefs)
	artifact.TrustRefs = sortedAssignmentStrings(artifact.TrustRefs)
	artifact.SecurityRefs = sortedAssignmentStrings(artifact.SecurityRefs)
}

func validAssignmentRef(value string) bool {
	if value == "" || value != strings.TrimSpace(value) || !utf8.ValidString(value) ||
		utf8.RuneCountInString(value) > 512 || strings.HasPrefix(value, "/") ||
		strings.HasPrefix(strings.ToLower(value), "file:") || strings.Contains(value, "\\") ||
		strings.Contains(value, "://") || strings.ContainsAny(value, "?#") {
		return false
	}
	for _, r := range value {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return false
		}
	}
	return true
}

func validAssignmentSHA256(value string) bool {
	if len(value) != 64 || value != strings.ToLower(value) {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func assignmentDigestString(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
