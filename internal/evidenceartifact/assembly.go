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
	ErrAnnotationInvalid     = errors.New("evidence annotation geometry invalid")
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
	Annotations    []CaptureAnnotation  `json:"annotations,omitempty"`
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

type CaptureAnnotation struct {
	AnnotationRef     string             `json:"annotation_ref"`
	ObservationRef    string             `json:"observation_ref"`
	SourceAssetID     string             `json:"source_asset_id"`
	SourceAssetSHA256 string             `json:"source_asset_sha256"`
	SourceRevision    uint64             `json:"source_revision"`
	AuthorRef         string             `json:"author_ref"`
	CreatedAt         string             `json:"created_at"`
	Label             string             `json:"label"`
	Geometry          AnnotationGeometry `json:"geometry"`
	OverlaySHA256     string             `json:"overlay_sha256"`
}

type AnnotationGeometry struct {
	CoordinateSpace string `json:"coordinate_space"`
	X               int    `json:"x"`
	Y               int    `json:"y"`
	Width           int    `json:"width"`
	Height          int    `json:"height"`
	SourceWidth     int    `json:"source_width"`
	SourceHeight    int    `json:"source_height"`
	CropX           int    `json:"crop_x"`
	CropY           int    `json:"crop_y"`
	CropWidth       int    `json:"crop_width"`
	CropHeight      int    `json:"crop_height"`
	RotationDegrees int    `json:"rotation_degrees"`
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
	out.Annotations = append([]CaptureAnnotation(nil), in.Annotations...)
	sort.Slice(out.Annotations, func(i, j int) bool { return out.Annotations[i].AnnotationRef < out.Annotations[j].AnnotationRef })
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
	annotations := append([]CaptureAnnotation(nil), request.Annotations...)
	sort.Slice(annotations, func(i, j int) bool { return annotations[i].AnnotationRef < annotations[j].AnnotationRef })

	base := Normalize(request.Base)
	if err := validateAssemblyCoverage(base, observations, viewports, requirements, omissions, request.WindowComplete); err != nil {
		return Manifest{}, err
	}
	base.Capture = &CaptureMetadata{
		RunRef: request.RunRef, RuntimeRef: request.RuntimeRef, EnvironmentRef: request.EnvironmentRef,
		TargetRef: request.TargetRef, AccountRef: request.AccountRef, WindowComplete: request.WindowComplete,
		Viewports: viewports, Observations: observations, Requirements: requirements, Omissions: omissions,
		Annotations: annotations,
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
