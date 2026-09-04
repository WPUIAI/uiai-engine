package evidenceartifact

import (
	"math"
	"strings"
	"time"
)

func validateCaptureManifest(manifest Manifest) error {
	capture := manifest.Capture
	if capture == nil {
		return nil
	}
	for _, ref := range []string{capture.RunRef, capture.RuntimeRef, capture.EnvironmentRef, capture.TargetRef, capture.AccountRef} {
		if !validRef(ref, true) {
			return ErrRuntimeAmbiguous
		}
	}
	if err := validateCaptureAnnotations(manifest, capture.Annotations, capture.Observations); err != nil {
		return err
	}
	return validateAssemblyCoverage(manifest, capture.Observations, capture.Viewports, capture.Requirements, capture.Omissions, capture.WindowComplete)
}

func validateAssemblyIdentity(request CaptureAssembly) error {
	if request.Base.Capture != nil {
		return ErrAssemblyInvalid
	}
	for _, ref := range []string{request.RunRef, request.RuntimeRef, request.EnvironmentRef, request.TargetRef, request.AccountRef} {
		if !validRef(ref, true) {
			return ErrRuntimeAmbiguous
		}
	}
	if err := Validate(request.Base); err != nil {
		return ErrAssemblyInvalid
	}
	if len(request.Observations) == 0 || len(request.Requirements) == 0 {
		return ErrAssemblyInvalid
	}
	return nil
}

func validateAssemblyCoverage(base Manifest, observations []CaptureObservation, viewports []CaptureViewport, requirements []ClaimRequirement, omissions []CaptureOmission, windowComplete bool) error {
	viewportByRef := make(map[string]CaptureViewport, len(viewports))
	for _, viewport := range viewports {
		if !validRef(viewport.ViewportRef, true) || viewport.Width <= 0 || viewport.Width > 16384 || viewport.Height <= 0 || viewport.Height > 16384 || math.IsNaN(viewport.DPR) || math.IsInf(viewport.DPR, 0) || viewport.DPR < 0.5 || viewport.DPR > 8 {
			return ErrAssemblyInvalid
		}
		if _, duplicate := viewportByRef[viewport.ViewportRef]; duplicate {
			return ErrAssemblyInvalid
		}
		viewportByRef[viewport.ViewportRef] = viewport
	}
	omissionByRef := make(map[string]CaptureOmission, len(omissions))
	for _, omission := range omissions {
		if !validRef(omission.ObservationRef, true) || !validText(omission.Reason, 500, true) || !validRef(omission.PolicyRef, true) {
			return ErrAssemblyInvalid
		}
		if _, duplicate := omissionByRef[omission.ObservationRef]; duplicate {
			return ErrOmissionConflict
		}
		omissionByRef[omission.ObservationRef] = omission
	}
	assetByID := make(map[string]Asset, len(base.Assets))
	for _, asset := range base.Assets {
		assetByID[asset.AssetID] = asset
	}
	observationByAsset := make(map[string]CaptureObservation, len(observations))
	observationRefs := make(map[string]struct{}, len(observations))
	var previous time.Time
	for index, observation := range observations {
		if observation.Ordinal != uint64(index) || !validRef(observation.ObservationRef, true) {
			return ErrObservationGap
		}
		if _, duplicate := observationRefs[observation.ObservationRef]; duplicate {
			return ErrObservationGap
		}
		observationRefs[observation.ObservationRef] = struct{}{}
		occurred, ok := canonicalTime(observation.OccurredAt, true)
		if !ok || (!previous.IsZero() && occurred.Before(previous)) {
			return ErrObservationGap
		}
		previous = occurred
		if !validRef(observation.ActionRef, true) || !validRef(observation.ReceiptRef, true) || !validRef(observation.ViewportRef, false) {
			return ErrAssemblyInvalid
		}
		_, omitted := omissionByRef[observation.ObservationRef]
		if (observation.AssetID == "") == !omitted {
			return ErrOmissionConflict
		}
		if observation.ViewportRef != "" {
			if _, ok := viewportByRef[observation.ViewportRef]; !ok {
				return ErrCoverageIncomplete
			}
		}
		if observation.AssetID != "" {
			asset, ok := assetByID[observation.AssetID]
			if !ok || asset.SourceRef != observation.ObservationRef {
				return ErrCoverageIncomplete
			}
			if _, duplicate := observationByAsset[observation.AssetID]; duplicate {
				return ErrAssemblyInvalid
			}
			observationByAsset[observation.AssetID] = observation
		}
	}
	for observationRef := range omissionByRef {
		if _, ok := observationRefs[observationRef]; !ok {
			return ErrOmissionConflict
		}
	}
	if len(observationByAsset) != len(base.Assets) {
		return ErrCoverageIncomplete
	}
	claimByID := make(map[string]Claim, len(base.Claims))
	for _, claim := range base.Claims {
		claimByID[claim.ClaimID] = claim
	}
	requiredClaims := make(map[string]struct{}, len(requirements))
	for _, requirement := range requirements {
		if _, duplicate := requiredClaims[requirement.ClaimID]; duplicate {
			return ErrAssemblyInvalid
		}
		requiredClaims[requirement.ClaimID] = struct{}{}
		claim, ok := claimByID[requirement.ClaimID]
		if !ok {
			return ErrCoverageIncomplete
		}
		if requirement.Kind == RequirementProofOfAbsence && len(omissions) != 0 {
			return ErrCoverageIncomplete
		}
		if err := validateClaimRequirement(requirement, claim, base.Assets, observationByAsset, viewportByRef, windowComplete); err != nil {
			return err
		}
	}
	if len(requiredClaims) != len(claimByID) {
		return ErrCoverageIncomplete
	}
	return nil
}

func validateClaimRequirement(requirement ClaimRequirement, claim Claim, assets []Asset, observations map[string]CaptureObservation, viewports map[string]CaptureViewport, windowComplete bool) error {
	if !validRef(requirement.ClaimID, true) {
		return ErrAssemblyInvalid
	}
	claimAssets := make([]Asset, 0)
	for _, asset := range assets {
		assetClaims := stringSet(asset.ClaimRefs)
		claimEvidence := stringSet(claim.EvidenceRefs)
		_, assetClaimsClaim := assetClaims[claim.ClaimID]
		_, claimNamesAsset := claimEvidence[asset.AssetID]
		if assetClaimsClaim != claimNamesAsset {
			return ErrCoverageIncomplete
		}
		if assetClaimsClaim {
			claimAssets = append(claimAssets, asset)
		}
	}
	if len(claimAssets) == 0 {
		return ErrCoverageIncomplete
	}
	if requirement.ActualRequired {
		for _, asset := range claimAssets {
			if asset.VerificationClass != VerificationActual {
				return ErrEvidenceClassMismatch
			}
		}
	}
	switch requirement.Kind {
	case RequirementStaticVisual:
		if len(requirement.ViewportRefs) == 0 {
			return ErrCoverageIncomplete
		}
		covered := make(map[string]struct{})
		for _, asset := range claimAssets {
			observation := observations[asset.AssetID]
			if strings.HasPrefix(asset.MediaType, "image/") && observation.ViewportRef != "" {
				covered[observation.ViewportRef] = struct{}{}
			}
		}
		for _, viewportRef := range requirement.ViewportRefs {
			if _, exists := viewports[viewportRef]; !exists {
				return ErrCoverageIncomplete
			}
			if _, exists := covered[viewportRef]; !exists {
				return ErrCoverageIncomplete
			}
		}
	case RequirementTemporalInteraction:
		for _, asset := range claimAssets {
			observation := observations[asset.AssetID]
			if strings.HasPrefix(asset.MediaType, "video/") && observation.ActionRef != "" && observation.ReceiptRef != "" {
				return nil
			}
		}
		return ErrCoverageIncomplete
	case RequirementDiagnostic:
		for _, asset := range claimAssets {
			if asset.MediaType == "application/json" || asset.MediaType == "text/plain" {
				return nil
			}
		}
		return ErrCoverageIncomplete
	case RequirementStructured:
		for _, asset := range claimAssets {
			if asset.MediaType == "application/json" || asset.MediaType == "text/csv" {
				return nil
			}
		}
		return ErrCoverageIncomplete
	case RequirementProofOfAbsence:
		if !windowComplete {
			return ErrCoverageIncomplete
		}
		for _, asset := range claimAssets {
			if asset.MediaType == "application/json" && asset.VerificationClass == VerificationActual {
				return nil
			}
		}
		return ErrCoverageIncomplete
	default:
		return ErrAssemblyInvalid
	}
	return nil
}

func stringSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, value := range values {
		out[value] = struct{}{}
	}
	return out
}
