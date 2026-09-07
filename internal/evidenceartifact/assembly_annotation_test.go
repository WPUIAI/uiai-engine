package evidenceartifact

import (
	"errors"
	"strings"
	"testing"
)

func TestAssembleCaptureBindsAnnotationGeometry(t *testing.T) {
	request := assemblyFixture(t, RequirementStaticVisual)
	first := validCaptureAnnotation(request)
	first.AnnotationRef = "annotation:z"
	second := validCaptureAnnotation(request)
	second.AnnotationRef = "annotation:a"
	second.Geometry = AnnotationGeometry{
		CoordinateSpace: "source_pixels", X: 20, Y: 30, Width: 100, Height: 120,
		SourceWidth: 390, SourceHeight: 844, CropX: 10, CropY: 20, CropWidth: 200, CropHeight: 300,
		RotationDegrees: 90,
	}
	request.Annotations = []CaptureAnnotation{first, second}
	manifest, err := AssembleCapture(request)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Capture.Annotations) != 2 || manifest.Capture.Annotations[0].AnnotationRef != "annotation:a" {
		t.Fatalf("annotations not normalized: %#v", manifest.Capture.Annotations)
	}
	if request.Annotations[0].AnnotationRef != "annotation:z" {
		t.Fatal("input annotations were mutated")
	}
	encoded, err := CanonicalJSON(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"annotation_ref":"annotation:a"`) || strings.Contains(string(encoded), `"AnnotationRef"`) {
		t.Fatalf("annotation contract is not canonical snake_case: %s", encoded)
	}
}

func TestAssembleCaptureRejectsInvalidAnnotationGeometry(t *testing.T) {
	tests := map[string]func(*CaptureAnnotation){
		"unknown observation": func(a *CaptureAnnotation) { a.ObservationRef = "observation:missing" },
		"source mismatch":     func(a *CaptureAnnotation) { a.SourceAssetSHA256 = strings.Repeat("e", 64) },
		"revision mismatch":   func(a *CaptureAnnotation) { a.SourceRevision++ },
		"missing author":      func(a *CaptureAnnotation) { a.AuthorRef = "" },
		"bad time":            func(a *CaptureAnnotation) { a.CreatedAt = "tomorrow" },
		"bad overlay hash":    func(a *CaptureAnnotation) { a.OverlaySHA256 = "short" },
		"out of bounds":       func(a *CaptureAnnotation) { a.Geometry.X = 380; a.Geometry.Width = 20 },
		"integer overflow":    func(a *CaptureAnnotation) { a.Geometry.X = int(^uint(0) >> 1); a.Geometry.Width = 2 },
		"invalid crop":        func(a *CaptureAnnotation) { a.Geometry.CropWidth = 10 },
		"invalid rotation":    func(a *CaptureAnnotation) { a.Geometry.RotationDegrees = 45 },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			request := assemblyFixture(t, RequirementStaticVisual)
			annotation := validCaptureAnnotation(request)
			mutate(&annotation)
			request.Annotations = []CaptureAnnotation{annotation}
			if _, err := AssembleCapture(request); !errors.Is(err, ErrAnnotationInvalid) {
				t.Fatalf("AssembleCapture() error = %v, want ErrAnnotationInvalid", err)
			}
		})
	}
}

func TestAssembleCaptureRejectsDuplicateAnnotationRefs(t *testing.T) {
	request := assemblyFixture(t, RequirementStaticVisual)
	annotation := validCaptureAnnotation(request)
	request.Annotations = []CaptureAnnotation{annotation, annotation}
	if _, err := AssembleCapture(request); !errors.Is(err, ErrAnnotationInvalid) {
		t.Fatalf("AssembleCapture() error = %v, want ErrAnnotationInvalid", err)
	}
}

func validCaptureAnnotation(request CaptureAssembly) CaptureAnnotation {
	asset := request.Base.Assets[0]
	return CaptureAnnotation{
		AnnotationRef: "annotation:proof", ObservationRef: request.Observations[0].ObservationRef,
		SourceAssetID: asset.AssetID, SourceAssetSHA256: asset.SHA256, SourceRevision: request.Base.Revision,
		AuthorRef: "agent:reviewer", CreatedAt: "2026-08-29T12:00:02Z", Label: "Observed control",
		Geometry: AnnotationGeometry{
			CoordinateSpace: "source_pixels", X: 10, Y: 20, Width: 120, Height: 80,
			SourceWidth: asset.Width, SourceHeight: asset.Height, RotationDegrees: 0,
		},
		OverlaySHA256: strings.Repeat("d", 64),
	}
}
