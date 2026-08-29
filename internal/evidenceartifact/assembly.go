package evidenceartifact

import (
	"errors"
	"sort"
)

var (
	ErrAssemblyInvalid       = errors.New("invalid evidence capture assembly")
	ErrObservationGap        = errors.New("evidence observation sequence gap")
	ErrCoverageIncomplete    = errors.New("evidence capture coverage incomplete")
	ErrEvidenceClassMismatch = errors.New("evidence class mismatch")
	ErrOmissionConflict      = errors.New("evidence omission conflict")
	ErrRuntimeAmbiguous      = errors.New("evidence runtime identity ambiguous")
)

type RequirementKind string

const (
	RequirementStaticVisual        RequirementKind = "static_visual"
	RequirementTemporalInteraction RequirementKind = "temporal_interaction"
	RequirementDiagnostic          RequirementKind = "diagnostic"
	RequirementStructured          RequirementKind = "structured"
	RequirementProofOfAbsence      RequirementKind = "proof_of_absence"
)

type CaptureAssembly struct {
	Base Manifest
	CaptureMetadata
}

type CaptureMetadata struct {
	RunRef         string               `json:"run_ref"`
	RuntimeRef     string               `json:"runtime_ref"`
	EnvironmentRef string               `json:"environment_ref"`
	TargetRef      string               `json:"target_ref"`
	AccountRef     string               `json:"account_ref"`
	WindowComplete bool                 `json:"window_complete"`
	Viewports      []CaptureViewport    `json:"viewports"`
	Observations   []CaptureObservation `json:"observations"`
	Requirements   []ClaimRequirement   `json:"requirements"`
	Omissions      []CaptureOmission    `json:"omissions"`
}

type CaptureViewport struct {
	ViewportRef string
	Width       int
	Height      int
	DPR         float64
}

type CaptureObservation struct {
	ObservationRef string
	Ordinal        uint64
	OccurredAt     string
	ActionRef      string
	ReceiptRef     string
	ViewportRef    string
	AssetID        string
}

type ClaimRequirement struct {
	ClaimID        string
	Kind           RequirementKind
	ActualRequired bool
	ViewportRefs   []string
}

type CaptureOmission struct {
	ObservationRef string
	Reason         string
	PolicyRef      string
}

func normalizeCapture(in CaptureMetadata) CaptureMetadata {
	out := in
	out.Viewports = append([]CaptureViewport(nil), in.Viewports...)
	sort.Slice(out.Viewports, func(i, j int) bool { return out.Viewports[i].ViewportRef < out.Viewports[j].ViewportRef })
	out.Observations = append([]CaptureObservation(nil), in.Observations...)
	sort.Slice(out.Observations, func(i, j int) bool { return out.Observations[i].Ordinal < out.Observations[j].Ordinal })
	out.Requirements = append([]ClaimRequirement(nil), in.Requirements...)
	for i := range out.Requirements {
		out.Requirements[i].ViewportRefs = normalizeSet(out.Requirements[i].ViewportRefs)
	}
	sort.Slice(out.Requirements, func(i, j int) bool { return out.Requirements[i].ClaimID < out.Requirements[j].ClaimID })
	out.Omissions = append([]CaptureOmission(nil), in.Omissions...)
	sort.Slice(out.Omissions, func(i, j int) bool { return out.Omissions[i].ObservationRef < out.Omissions[j].ObservationRef })
	return out
}

func AssembleCapture(request CaptureAssembly) (Manifest, error) {
	if err := validateAssemblyIdentity(request); err != nil {
		return Manifest{}, err
	}
	observations := append([]CaptureObservation(nil), request.Observations...)
	sort.Slice(observations, func(i, j int) bool { return observations[i].Ordinal < observations[j].Ordinal })
	viewports := append([]CaptureViewport(nil), request.Viewports...)
	sort.Slice(viewports, func(i, j int) bool { return viewports[i].ViewportRef < viewports[j].ViewportRef })
	requirements := append([]ClaimRequirement(nil), request.Requirements...)
	for i := range requirements {
		requirements[i].ViewportRefs = normalizeSet(requirements[i].ViewportRefs)
	}
	sort.Slice(requirements, func(i, j int) bool { return requirements[i].ClaimID < requirements[j].ClaimID })
	omissions := append([]CaptureOmission(nil), request.Omissions...)
	sort.Slice(omissions, func(i, j int) bool { return omissions[i].ObservationRef < omissions[j].ObservationRef })

	base := Normalize(request.Base)
	if err := validateAssemblyCoverage(base, observations, viewports, requirements, omissions, request.WindowComplete); err != nil {
		return Manifest{}, err
	}
	base.Capture = &CaptureMetadata{
		RunRef: request.RunRef, RuntimeRef: request.RuntimeRef, EnvironmentRef: request.EnvironmentRef,
		TargetRef: request.TargetRef, AccountRef: request.AccountRef, WindowComplete: request.WindowComplete,
		Viewports: viewports, Observations: observations, Requirements: requirements, Omissions: omissions,
	}
	base.Integrity.ManifestSHA256 = ""
	base.Provenance.SourceRefs = normalizeSet(append(base.Provenance.SourceRefs, request.RunRef, request.TargetRef, request.AccountRef))
	base.Provenance.EnvironmentRefs = normalizeSet(append(base.Provenance.EnvironmentRefs, request.RuntimeRef, request.EnvironmentRef))
	base.Provenance.OmissionRefs = []string{}
	base.Provenance.Custody = []CustodyEvent{}
	omissionByObservation := make(map[string]CaptureOmission, len(omissions))
	for _, omission := range omissions {
		omissionByObservation[omission.ObservationRef] = omission
		base.Provenance.OmissionRefs = append(base.Provenance.OmissionRefs, "omission:"+deterministicID(omission.ObservationRef, omission.Reason, omission.PolicyRef))
	}
	base.Provenance.OmissionRefs = normalizeSet(base.Provenance.OmissionRefs)
	for _, observation := range observations {
		outputs := []string{}
		if observation.AssetID != "" {
			outputs = append(outputs, observation.AssetID)
		} else {
			omission := omissionByObservation[observation.ObservationRef]
			outputs = append(outputs, "omission:"+deterministicID(omission.ObservationRef, omission.Reason, omission.PolicyRef))
		}
		base.Provenance.Custody = append(base.Provenance.Custody, CustodyEvent{
			EventID: observation.ObservationRef, Action: observation.ActionRef, ActorRef: base.Authority.ProducerRef,
			InstanceRef: request.RuntimeRef, InputRefs: normalizeSet([]string{request.TargetRef, observation.ReceiptRef}),
			OutputRefs: outputs, OccurredAt: observation.OccurredAt,
		})
	}
	base.Links.RelatedRefs = normalizeSet(append(base.Links.RelatedRefs, request.RunRef, request.RuntimeRef, request.EnvironmentRef, request.TargetRef, request.AccountRef))
	return Seal(base)
}
