package epwadelivery

// ProducerID identifies an artifact-producing UIAI boundary. Every listed
// producer is fail-closed: raw-only success is forbidden and both the HTTPS
// viewer and portable package are required for StateReady.
type ProducerID string

const (
	ProducerScreenshot       ProducerID = "screenshot.capture"
	ProducerSessionVisual    ProducerID = "session.visual"
	ProducerInteractive      ProducerID = "vision.interactive"
	ProducerMedia            ProducerID = "media.render"
	ProducerRuntimeArtifact  ProducerID = "artifact.runtime"
	ProducerShareScreenshot  ProducerID = "share.screenshot"
	ProducerFPV              ProducerID = "fpv.create"
	ProducerArtifactCommit   ProducerID = "artifact.commit"
	ProducerDerivative       ProducerID = "derivative.connect"
	ProducerDiagnostics      ProducerID = "diagnostics.bundle"
	ProducerSourceSnapshot   ProducerID = "source.snapshot"
	ProducerDOMSnapshot      ProducerID = "dom.snapshot"
	ProducerCritique         ProducerID = "report.critique"
	ProducerVisualComparison ProducerID = "report.visual_compare"
	ProducerResearch         ProducerID = "research.packet"
	ProducerGeneratedReport  ProducerID = "report.generated"
)

type ProducerPolicy struct {
	ID                    ProducerID `json:"id"`
	ArtifactClass         string     `json:"artifact_class"`
	DeliverySchema        string     `json:"delivery_schema"`
	RawOnlySuccess        bool       `json:"raw_only_success"`
	HTTPSViewerRequired   bool       `json:"https_viewer_required"`
	PortableZIPRequired   bool       `json:"portable_zip_required"`
	CompleteScopeRequired bool       `json:"complete_scope_required"`
}

var producerInventory = []ProducerPolicy{
	{ProducerScreenshot, "screenshot", Schema, false, true, true, true},
	{ProducerSessionVisual, "session_visual", Schema, false, true, true, true},
	{ProducerInteractive, "interactive_visual", Schema, false, true, true, true},
	{ProducerMedia, "media", Schema, false, true, true, true},
	{ProducerRuntimeArtifact, "runtime_binary", Schema, false, true, true, true},
	{ProducerShareScreenshot, "share_screenshot", Schema, false, true, true, true},
	{ProducerFPV, "fpv_mirror_with_durable_evidence", Schema, false, true, true, true},
	{ProducerArtifactCommit, "immutable_artifact", Schema, false, true, true, true},
	{ProducerDerivative, "derivative", Schema, false, true, true, true},
	{ProducerDiagnostics, "diagnostics_bundle", Schema, false, true, true, true},
	{ProducerSourceSnapshot, "source_snapshot", Schema, false, true, true, true},
	{ProducerDOMSnapshot, "dom_snapshot", Schema, false, true, true, true},
	{ProducerCritique, "critique_report", Schema, false, true, true, true},
	{ProducerVisualComparison, "visual_comparison_report", Schema, false, true, true, true},
	{ProducerResearch, "research_packet", Schema, false, true, true, true},
	{ProducerGeneratedReport, "generated_report", Schema, false, true, true, true},
}

func ProducerInventory() []ProducerPolicy {
	return append([]ProducerPolicy(nil), producerInventory...)
}

func ValidProducer(id ProducerID) bool {
	for _, policy := range producerInventory {
		if policy.ID == id {
			return true
		}
	}
	return false
}

func ProducerForArtifactKind(kind string) ProducerID {
	switch kind {
	case "diagnostics", "diagnostics_bundle":
		return ProducerDiagnostics
	case "source_snapshot":
		return ProducerSourceSnapshot
	case "dom_snapshot":
		return ProducerDOMSnapshot
	case "critique_report":
		return ProducerCritique
	case "visual_comparison_report":
		return ProducerVisualComparison
	case "research_packet":
		return ProducerResearch
	case "runtime_binary":
		return ProducerRuntimeArtifact
	case "media":
		return ProducerMedia
	default:
		return ProducerGeneratedReport
	}
}
