package evidenceshare

import (
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
)

var (
	ErrInvalidInput = errors.New("screenshot evidence share input invalid")
	ErrConflict     = errors.New("screenshot evidence share content conflict")
)

//go:embed assets/index.html assets/styles.css assets/work-items.js assets/locale.js assets/pwa.js assets/app.js assets/manifest.webmanifest assets/icon.svg assets/sw.js
var assets embed.FS

var packagedAssetNames = []string{"index.html", "styles.css", "work-items.js", "locale.js", "pwa.js", "app.js", "manifest.webmanifest", "icon.svg", "sw.js"}

func Assemble(root string, input Input) (Result, error) {
	if strings.TrimSpace(root) == "" || len(input.Screenshot) == 0 || input.Width <= 0 || input.Height <= 0 || input.CapturedAt.IsZero() {
		return Result{}, ErrInvalidInput
	}
	format, mime, ok := normalizedFormat(input.Format)
	if !ok {
		return Result{}, ErrInvalidInput
	}
	screenshotDigest := sha256.Sum256(input.Screenshot)
	screenshotSHA := hex.EncodeToString(screenshotDigest[:])
	scopeBody, err := json.Marshal(input.Scope)
	if err != nil {
		return Result{}, err
	}
	idInput := fmt.Sprintf("%s\n%d\n%d\n%s\n%s\n%s\n%s", screenshotSHA, input.Width, input.Height, sanitizeURL(input.SourceURL), input.CapturedAt.UTC().Format(time.RFC3339Nano), embeddedAssetsSHA256(), scopeBody)
	idDigest := sha256.Sum256([]byte(idInput))
	id := hex.EncodeToString(idDigest[:])
	artifactRef := "uiai-evidence-share:sha256:" + id
	_, inspectionBody, inspectionErr := inspectScreenshot(input.Screenshot, mime, screenshotSHA)
	availability := "blocked"
	projectionRef := ""
	inspectionRef := ""
	if inspectionErr == nil {
		inspectionRef = "./inspection.json"
	}
	if input.Scope.Complete() && inspectionErr == nil {
		availability = "ready"
		projectionRef = "./projection.json"
	}
	manifest := Manifest{Schema: Schema, ArtifactRef: artifactRef, ArtifactSHA256: id, ScreenshotRef: "./screenshot." + format, ScreenshotSHA256: screenshotSHA, Format: format, MIME: mime, Bytes: len(input.Screenshot), Width: input.Width, Height: input.Height, SourceURL: sanitizeURL(input.SourceURL), CapturedAt: input.CapturedAt.UTC(), DurationMS: input.DurationMS, Availability: availability, Access: "public_safe_read_only", Interaction: "read_only", Scope: input.Scope, TruthNotice: "This screenshot proves only the captured visual state. It does not by itself prove review, completion, provider closure, or settlement.", ProjectionRef: projectionRef, InspectionRef: inspectionRef}
	body, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return Result{}, err
	}
	body = append(body, '\n')
	manifestDigest := sha256.Sum256(body)
	manifestBindingSHA := hex.EncodeToString(manifestDigest[:])
	manifestSHA := id
	var projectionBody []byte
	if input.Scope.Complete() {
		projection, err := buildProjection(manifest, manifestBindingSHA)
		if err != nil {
			return Result{}, err
		}
		projectionBody, err = json.MarshalIndent(projection, "", "  ")
		if err != nil {
			return Result{}, err
		}
		projectionBody = append(projectionBody, '\n')
	}
	finalDir := filepath.Join(root, id)
	if existing, err := os.ReadFile(filepath.Join(finalDir, "artifact.json")); err == nil {
		if string(existing) != string(body) {
			return Result{}, ErrConflict
		}
		return Result{ArtifactRef: artifactRef, ArtifactSHA256: manifestSHA, RelativePath: "./" + id + "/", Directory: finalDir}, nil
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return Result{}, err
	}
	staging, err := os.MkdirTemp(root, ".share-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(staging)
	writes := map[string][]byte{"artifact.json": body}
	if inspectionErr == nil {
		writes["screenshot."+format] = input.Screenshot
		writes["inspection.json"] = inspectionBody
	}
	if len(projectionBody) != 0 {
		writes["projection.json"] = projectionBody
	}
	assetVersion := embeddedAssetsSHA256()[:12]
	for _, name := range packagedAssetNames {
		data, err := assets.ReadFile("assets/" + name)
		if err != nil {
			return Result{}, err
		}
		switch name {
		case "index.html":
			page := strings.ReplaceAll(string(data), "__UIAI_ASSET_VERSION__", assetVersion)
			page = strings.Replace(page, `data-default-view="registry"`, `data-default-view="record"`, 1)
			data = []byte(page)
		case "sw.js":
			data = []byte(strings.ReplaceAll(string(data), "__UIAI_ASSET_VERSION__", assetVersion))
		}
		writes[name] = data
	}
	for name, data := range writes {
		if err := os.WriteFile(filepath.Join(staging, name), data, 0o640); err != nil {
			return Result{}, err
		}
	}
	if err := os.Rename(staging, finalDir); err != nil {
		if _, statErr := os.Stat(finalDir); statErr == nil {
			return Assemble(root, input)
		}
		return Result{}, err
	}
	return Result{ArtifactRef: artifactRef, ArtifactSHA256: manifestSHA, RelativePath: "./" + id + "/", Directory: finalDir}, nil
}

func buildProjection(manifest Manifest, manifestSHA string) (evidencepwa.Projection, error) {
	assetID := "asset:screenshot"
	citationID := "citation:screenshot"
	claimID := "claim:captured-visual-state"
	projection := evidencepwa.Projection{
		Schema:       evidencepwa.ProjectionSchema,
		ProjectionID: "uiai-evidence-projection:sha256:" + manifest.ArtifactSHA256,
		Artifact: evidencepwa.ArtifactBinding{
			ArtifactRef:    manifest.ArtifactRef,
			Revision:       1,
			ManifestSHA256: manifestSHA,
			BundleSHA256:   manifest.ArtifactSHA256,
			Scope: evidencepwa.ScopeBinding{
				ProjectRef: manifest.Scope.ProjectRef, WorkstreamRef: manifest.Scope.WorkstreamRef,
				WorksetRef: manifest.Scope.WorksetRef, CallGraphRef: manifest.Scope.CallGraphRef,
				WorkpointRef: manifest.Scope.WorkpointRef, WorkItemRef: manifest.Scope.WorkItemRef,
			},
		},
		WorkItems: manifest.Scope.WorkItems,
		Title:     "Evidence Artifact",
		Summary:   manifest.TruthNotice,
		Sections: []evidencepwa.Section{
			{ID: evidencepwa.SectionOverview, Heading: "Overview"},
			{ID: evidencepwa.SectionEvidence, Heading: "Evidence"},
			{ID: evidencepwa.SectionTimeline, Heading: "Timeline"},
			{ID: evidencepwa.SectionInspect, Heading: "Inspect"},
			{ID: evidencepwa.SectionDeveloper, Heading: "Developer"},
		},
		Claims:              []evidencepwa.Claim{{ClaimID: claimID, Statement: "The bound screenshot records the captured browser visual state at the stated time.", Posture: "captured_visual_only", CitationIDs: []string{citationID}, AssetIDs: []string{assetID}}},
		Assets:              []evidencepwa.Asset{{AssetID: assetID, Ref: manifest.ScreenshotRef, SHA256: manifest.ScreenshotSHA256, MIME: manifest.MIME, Modality: "screenshot", Bytes: uint64(manifest.Bytes)}},
		Citations:           []evidencepwa.Citation{{CitationID: citationID, SourceRef: assetID, SHA256: manifest.ScreenshotSHA256, Locator: "full_frame"}},
		Timeline:            []evidencepwa.TimelineEntry{{EntryID: "timeline:capture", OccurredAt: manifest.CapturedAt, EventType: "screenshot_captured", Refs: []string{assetID, claimID}}},
		SecurityRefs:        []string{manifest.InspectionRef},
		Availability:        evidencepwa.AvailabilityReady,
		Access:              evidencepwa.AccessPublicSafe,
		Redaction:           evidencepwa.Redaction{State: evidencepwa.RedactionApplied, EvidenceRef: manifest.InspectionRef},
		FederationPosture:   "source_local",
		FreshnessObservedAt: manifest.CapturedAt,
		Page:                evidencepwa.PageInfo{PageSize: 1, TotalCount: 1},
		Links:               []evidencepwa.RelativeLink{{Rel: "manifest", Href: "./artifact.json"}, {Rel: "screenshot", Href: manifest.ScreenshotRef}, {Rel: "inspection", Href: manifest.InspectionRef}},
		Interaction:         evidencepwa.InteractionReadOnly,
	}
	if err := evidencepwa.ValidateProjection(projection); err != nil {
		return evidencepwa.Projection{}, err
	}
	return projection, nil
}

func inspectScreenshot(data []byte, mediaType, screenshotSHA string) (evidenceartifact.InspectionRecord, []byte, error) {
	file, err := os.CreateTemp("", ".uiai-share-inspection-")
	if err != nil {
		return evidenceartifact.InspectionRecord{}, nil, err
	}
	path := file.Name()
	defer os.Remove(path)
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return evidenceartifact.InspectionRecord{}, nil, err
	}
	if err := file.Close(); err != nil {
		return evidenceartifact.InspectionRecord{}, nil, err
	}
	record, err := evidenceartifact.NewBuiltinInspector().Inspect(context.Background(), evidenceartifact.InspectionRequest{
		Path:     path,
		Asset:    evidenceartifact.Asset{AssetID: "asset:screenshot", MediaType: mediaType, SHA256: screenshotSHA, ByteSize: int64(len(data))},
		Security: evidenceartifact.Security{PolicyRef: evidenceartifact.StrictSecurityPolicyV1},
		Policy:   evidenceartifact.Policy{AccessClass: evidenceartifact.AccessPublicSafe, RedactionState: evidenceartifact.RedactionPublicSafe, Audience: "public"},
	})
	if err != nil {
		return evidenceartifact.InspectionRecord{}, nil, err
	}
	body, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return evidenceartifact.InspectionRecord{}, nil, err
	}
	return record, append(body, '\n'), nil
}

func embeddedAssetsSHA256() string {
	hash := sha256.New()
	for _, name := range packagedAssetNames {
		data, err := assets.ReadFile("assets/" + name)
		if err != nil {
			return ""
		}
		_, _ = hash.Write([]byte(name))
		_, _ = hash.Write(data)
	}
	return hex.EncodeToString(hash.Sum(nil))
}

func normalizedFormat(value string) (string, string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "png":
		return "png", "image/png", true
	case "jpg", "jpeg":
		return "jpg", "image/jpeg", true
	case "webp":
		return "webp", "image/webp", true
	default:
		return "", "", false
	}
}
func sanitizeURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return ""
	}
	return parsed.String()
}
