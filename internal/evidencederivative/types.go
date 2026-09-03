package evidencederivative

import (
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
)

const (
	RequestSchema      = "uiai.evidence_derivative_request.v1"
	ManifestSchema     = "uiai.evidence_derivative_manifest.v1"
	DeliverySchema     = "uiai.evidence_derivative_delivery.v1"
	ViewerMatrixSchema = "uiai.evidence_derivative_viewer_matrix.v1"
	LicenseSchema      = "uiai.evidence_derivative_license.v1"
	MaxContractBytes   = 1 << 20
	MaxRefs            = 512
	MaxEntries         = 2048
)

type ScopeBinding = evidencepwa.ScopeBinding

type DerivativeType string

const (
	DerivativePrint           DerivativeType = "print"
	DerivativePDF             DerivativeType = "pdf"
	DerivativeEmailText       DerivativeType = "email_text"
	DerivativeEmailHTML       DerivativeType = "email_html"
	DerivativeMarkdown        DerivativeType = "markdown"
	DerivativeRichText        DerivativeType = "rich_text"
	DerivativeHTML            DerivativeType = "html"
	DerivativeJSON            DerivativeType = "json"
	DerivativeCSV             DerivativeType = "csv"
	DerivativeArchive         DerivativeType = "archive"
	DerivativePPTX            DerivativeType = "pptx"
	DerivativeHTMLSlides      DerivativeType = "html_slides"
	DerivativePresentationPDF DerivativeType = "presentation_pdf"
)

type AccessibilityTarget string

const (
	AccessibilityPlainText     AccessibilityTarget = "plain_text"
	AccessibilityWCAG22AA      AccessibilityTarget = "wcag_2_2_aa"
	AccessibilityPDFUA1        AccessibilityTarget = "pdf_ua_1"
	AccessibilityPDFUA2        AccessibilityTarget = "pdf_ua_2"
	AccessibilityNotApplicable AccessibilityTarget = "not_applicable"
)

const (
	PDFAProfile2B = "PDF/A-2b"
	PDFUAProfile1 = "PDF/UA-1"
	PDFUAProfile2 = "PDF/UA-2"
)

type ConformancePosture string

const (
	ConformanceNotClaimed ConformancePosture = "not_claimed"
	ConformanceTargeted   ConformancePosture = "targeted"
	ConformanceVerified   ConformancePosture = "verified"
	ConformanceFailed     ConformancePosture = "failed"
)

type ArchivePosture string

const (
	ArchiveNotApplicable ArchivePosture = "not_applicable"
	ArchiveSafe          ArchivePosture = "safe"
	ArchiveBlocked       ArchivePosture = "blocked"
)

type Direction string

const (
	DirectionLTR Direction = "ltr"
	DirectionRTL Direction = "rtl"
)

type RenderingProfile struct {
	ProfileRef      string   `json:"profile_ref"`
	ProfileSHA256   string   `json:"profile_sha256"`
	FontRefs        []string `json:"font_refs"`
	ColorProfileRef string   `json:"color_profile_ref"`
	DependencyRefs  []string `json:"dependency_refs,omitempty"`
}

type DerivativeRequest struct {
	Schema               string              `json:"schema"`
	RequestID            string              `json:"request_id"`
	Scope                ScopeBinding        `json:"scope"`
	ArtifactRef          string              `json:"artifact_ref"`
	ArtifactSHA256       string              `json:"artifact_sha256"`
	ArtifactRevision     uint64              `json:"artifact_revision"`
	ProjectionRef        string              `json:"projection_ref"`
	ProjectionSHA256     string              `json:"projection_sha256"`
	DerivativeType       DerivativeType      `json:"derivative_type"`
	ClaimRefs            []string            `json:"claim_refs"`
	AssetRefs            []string            `json:"asset_refs"`
	CitationRefs         []string            `json:"citation_refs"`
	OmissionRefs         []string            `json:"omission_refs,omitempty"`
	RequiredEvidenceRefs []string            `json:"required_evidence_refs"`
	Rendering            RenderingProfile    `json:"rendering"`
	Locale               string              `json:"locale"`
	Direction            Direction           `json:"direction"`
	AccessibilityTarget  AccessibilityTarget `json:"accessibility_target"`
	LicensePolicyRef     string              `json:"license_policy_ref"`
	LicensePolicySHA256  string              `json:"license_policy_sha256"`
	IdempotencyKey       string              `json:"idempotency_key"`
}

type RendererIdentity struct {
	RendererRef  string `json:"renderer_ref"`
	Version      string `json:"version"`
	BinarySHA256 string `json:"binary_sha256"`
}

type ViewerStatus string

const (
	ViewerSupported   ViewerStatus = "supported"
	ViewerDegraded    ViewerStatus = "degraded"
	ViewerUnsupported ViewerStatus = "unsupported"
	ViewerUntested    ViewerStatus = "untested"
)

type ViewerEntry struct {
	Client       string       `json:"client"`
	Version      string       `json:"version"`
	Status       ViewerStatus `json:"status"`
	EvidenceRefs []string     `json:"evidence_refs,omitempty"`
}
type ViewerMatrix struct {
	Schema    string        `json:"schema"`
	MatrixRef string        `json:"matrix_ref"`
	Entries   []ViewerEntry `json:"entries"`
}

type LicenseAttestation struct {
	Schema              string `json:"schema"`
	AssetRef            string `json:"asset_ref"`
	LicenseRef          string `json:"license_ref"`
	LicenseSHA256       string `json:"license_sha256"`
	AttributionRequired bool   `json:"attribution_required"`
	AttributionRef      string `json:"attribution_ref,omitempty"`
	DerivativePermitted bool   `json:"derivative_permitted"`
	EvidenceRef         string `json:"evidence_ref"`
}

type ArchiveEntry struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	MIME   string `json:"mime"`
	Bytes  uint64 `json:"bytes"`
	Link   bool   `json:"link"`
}

type DerivativeManifest struct {
	Schema                    string               `json:"schema"`
	DerivativeID              string               `json:"derivative_id"`
	RequestRef                string               `json:"request_ref"`
	RequestSHA256             string               `json:"request_sha256"`
	ArtifactRef               string               `json:"artifact_ref"`
	ArtifactSHA256            string               `json:"artifact_sha256"`
	ProjectionRef             string               `json:"projection_ref"`
	ProjectionSHA256          string               `json:"projection_sha256"`
	OutputRef                 string               `json:"output_ref"`
	OutputSHA256              string               `json:"output_sha256"`
	OutputBytes               uint64               `json:"output_bytes"`
	OutputMIME                string               `json:"output_mime"`
	Renderer                  RendererIdentity     `json:"renderer"`
	Rendering                 RenderingProfile     `json:"rendering"`
	TranscriptRefs            []string             `json:"transcript_refs,omitempty"`
	KeyframeRefs              []string             `json:"keyframe_refs,omitempty"`
	AccessibilityTarget       AccessibilityTarget  `json:"accessibility_target"`
	AccessibilityPosture      ConformancePosture   `json:"accessibility_posture"`
	PDFAProfile               string               `json:"pdfa_profile,omitempty"`
	PDFUAProfile              string               `json:"pdfua_profile,omitempty"`
	AccessibilityEvidenceRefs []string             `json:"accessibility_evidence_refs,omitempty"`
	ArchivePosture            ArchivePosture       `json:"archive_posture"`
	ArchiveEntries            []ArchiveEntry       `json:"archive_entries,omitempty"`
	ViewerMatrix              ViewerMatrix         `json:"viewer_matrix"`
	Licenses                  []LicenseAttestation `json:"licenses"`
	OmissionRefs              []string             `json:"omission_refs,omitempty"`
	WarningRefs               []string             `json:"warning_refs,omitempty"`
	ReceiptRef                string               `json:"receipt_ref"`
	CreatedAt                 time.Time            `json:"created_at"`
}

type DeliveryState string

const (
	DeliveryNotAttempted   DeliveryState = "not_attempted"
	DeliveryQueued         DeliveryState = "queued"
	DeliveryAccepted       DeliveryState = "accepted"
	DeliveryDelivered      DeliveryState = "delivered"
	DeliveryRejected       DeliveryState = "rejected"
	DeliveryBounced        DeliveryState = "bounced"
	DeliveryBlocked        DeliveryState = "blocked"
	DeliveryOutcomeUnknown DeliveryState = "outcome_unknown"
)

type DeliveryReceipt struct {
	Schema             string        `json:"schema"`
	DeliveryID         string        `json:"delivery_id"`
	DerivativeRef      string        `json:"derivative_ref"`
	DerivativeSHA256   string        `json:"derivative_sha256"`
	DestinationRef     string        `json:"destination_ref"`
	IdempotencyKey     string        `json:"idempotency_key"`
	PolicyRef          string        `json:"policy_ref,omitempty"`
	PolicySHA256       string        `json:"policy_sha256,omitempty"`
	State              DeliveryState `json:"state"`
	ProviderReceiptRef string        `json:"provider_receipt_ref,omitempty"`
	AcceptedAt         *time.Time    `json:"accepted_at,omitempty"`
	DeliveredAt        *time.Time    `json:"delivered_at,omitempty"`
	EvidenceRefs       []string      `json:"evidence_refs,omitempty"`
	ReconciliationRefs []string      `json:"reconciliation_refs,omitempty"`
	RetryPermitted     bool          `json:"retry_permitted"`
	ObservedAt         time.Time     `json:"observed_at"`
}
