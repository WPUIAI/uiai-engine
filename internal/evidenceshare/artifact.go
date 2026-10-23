package evidenceshare

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
)

// ArtifactInput is the immutable artifact-store projection used to assemble a
// portable EPWA package. Asset values are keyed by manifest asset_id.
type ArtifactInput struct {
	Manifest evidenceartifact.Manifest
	Assets   map[string][]byte
}

// AssembleArtifact creates an immutable, origin-neutral EPWA directory and a
// deterministic adjacent ZIP without changing the accepted source artifact.
func AssembleArtifact(root string, input ArtifactInput) (Result, error) {
	if strings.TrimSpace(root) == "" {
		return Result{}, ErrInvalidInput
	}
	manifest := evidenceartifact.Normalize(input.Manifest)
	if err := evidenceartifact.Validate(manifest); err != nil {
		return Result{}, err
	}
	if err := evidenceartifact.VerifyManifestSHA256(manifest); err != nil {
		return Result{}, err
	}
	if !validDigest(manifest.Integrity.BundleSHA256) || len(input.Assets) != len(manifest.Assets) {
		return Result{}, ErrInvalidInput
	}
	writes := make(map[string][]byte, len(manifest.Assets)+len(packagedAssetNames)+2)
	reserved := map[string]struct{}{"artifact.json": {}, "projection.json": {}}
	for _, name := range packagedAssetNames {
		reserved[name] = struct{}{}
	}
	for _, asset := range manifest.Assets {
		data, ok := input.Assets[asset.AssetID]
		_, collision := reserved[asset.Path]
		_, duplicate := writes[asset.Path]
		if !ok || collision || duplicate || int64(len(data)) != asset.ByteSize || digestBytes(data) != asset.SHA256 || !safePackagePath(asset.Path) {
			return Result{}, evidenceartifact.ErrAssetMismatch
		}
		writes[asset.Path] = append([]byte(nil), data...)
	}
	manifestBody, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return Result{}, err
	}
	manifestBody = append(manifestBody, '\n')
	writes["artifact.json"] = manifestBody
	projection, err := projectArtifact(manifest)
	if err != nil {
		return Result{}, err
	}
	projectionBody, err := json.MarshalIndent(projection, "", "  ")
	if err != nil {
		return Result{}, err
	}
	projectionBody = append(projectionBody, '\n')
	projectionSHA := digestBytes(projectionBody)
	writes["projection.json"] = projectionBody
	if err := addShellAssets(writes); err != nil {
		return Result{}, err
	}

	id, err := ArtifactPackageID(manifest)
	if err != nil {
		return Result{}, err
	}
	finalDir := filepath.Join(root, id)
	result := func(packageSHA string) Result {
		return Result{
			PackageID: id, ArtifactRef: manifest.ArtifactID, ArtifactSHA256: manifest.Integrity.BundleSHA256,
			ManifestSHA256: manifest.Integrity.ManifestSHA256, ProjectionRef: projection.ProjectionID,
			ProjectionSHA256: projectionSHA, OutputSHA256: manifest.Integrity.BundleSHA256, PackageSHA256: packageSHA,
			RelativePath: "./" + id + "/", PortableRelativePath: "./" + id + "/portable.zip", Directory: finalDir,
		}
	}
	if _, err := os.Stat(finalDir); err == nil {
		if err := verifyPackageFiles(finalDir, writes); err != nil {
			return Result{}, err
		}
		packageSHA, err := EnsurePortableArchive(root, id)
		if err != nil {
			return Result{}, err
		}
		return result(packageSHA), nil
	} else if !os.IsNotExist(err) {
		return Result{}, err
	}
	if err := os.MkdirAll(root, 0o750); err != nil {
		return Result{}, err
	}
	staging, err := os.MkdirTemp(root, ".artifact-share-")
	if err != nil {
		return Result{}, err
	}
	defer os.RemoveAll(staging)
	if err := writePackageFiles(staging, writes); err != nil {
		return Result{}, err
	}
	if err := os.Rename(staging, finalDir); err != nil {
		if _, statErr := os.Stat(finalDir); statErr == nil {
			return AssembleArtifact(root, input)
		}
		return Result{}, err
	}
	packageSHA, err := EnsurePortableArchive(root, id)
	if err != nil {
		return Result{}, err
	}
	return result(packageSHA), nil
}

// ArtifactPackageID returns the stable package identity without touching storage.
func ArtifactPackageID(input evidenceartifact.Manifest) (string, error) {
	manifest := evidenceartifact.Normalize(input)
	if err := evidenceartifact.Validate(manifest); err != nil {
		return "", err
	}
	if err := evidenceartifact.VerifyManifestSHA256(manifest); err != nil {
		return "", err
	}
	assetsSHA := embeddedAssetsSHA256()
	if !validDigest(assetsSHA) {
		return "", ErrInvalidInput
	}
	identity := sha256.Sum256([]byte(manifest.Integrity.ManifestSHA256 + "\n" + assetsSHA))
	return hex.EncodeToString(identity[:]), nil
}

func projectArtifact(manifest evidenceartifact.Manifest) (evidencepwa.Projection, error) {
	capturedAt, err := time.Parse(time.RFC3339Nano, manifest.CapturedAt)
	if err != nil {
		return evidencepwa.Projection{}, err
	}
	policies := make([]evidencepwa.WorkItemProjectionPolicy, 0, len(manifest.Scope.WorkItems))
	for _, item := range manifest.Scope.WorkItems {
		state := evidencepwa.WorkItemDescriptionUnavailable
		if item.Description != "" {
			state = evidencepwa.WorkItemDescriptionVisible
		} else if item.DescriptionRef != "" && item.DescriptionSHA256 != "" {
			state = evidencepwa.WorkItemDescriptionRedacted
		}
		policies = append(policies, evidencepwa.WorkItemProjectionPolicy{WorkItemRef: item.WorkItemRef, DescriptionState: state, RevisionState: evidencepwa.WorkItemRevisionUnknown})
	}
	workItems, err := evidencepwa.ProjectWorkItems(manifest.Scope.WorkItems, policies)
	if err != nil {
		return evidencepwa.Projection{}, err
	}
	assets := make([]evidencepwa.Asset, 0, len(manifest.Assets))
	claimAssets := make(map[string][]string, len(manifest.Claims))
	for _, asset := range manifest.Assets {
		assets = append(assets, evidencepwa.Asset{AssetID: asset.AssetID, Ref: "./" + asset.Path, SHA256: asset.SHA256, MIME: asset.MediaType, Modality: asset.Kind, Bytes: uint64(asset.ByteSize)})
		for _, claimRef := range asset.ClaimRefs {
			claimAssets[claimRef] = append(claimAssets[claimRef], asset.AssetID)
		}
	}
	claims := make([]evidencepwa.Claim, 0, len(manifest.Claims))
	for _, claim := range manifest.Claims {
		claims = append(claims, evidencepwa.Claim{ClaimID: claim.ClaimID, Statement: claim.Summary, Posture: string(claim.Status), AssetIDs: claimAssets[claim.ClaimID]})
	}
	timeline := make([]evidencepwa.TimelineEntry, 0, len(manifest.Provenance.Custody))
	for index, event := range manifest.Provenance.Custody {
		occurredAt, parseErr := time.Parse(time.RFC3339Nano, event.OccurredAt)
		if parseErr != nil {
			return evidencepwa.Projection{}, parseErr
		}
		timeline = append(timeline, evidencepwa.TimelineEntry{EntryID: fmt.Sprintf("timeline:custody:%d", index+1), OccurredAt: occurredAt, EventType: event.Action})
	}
	access := projectAccess(manifest.Policy.AccessClass)
	redaction, securityRefs, err := projectRedaction(manifest)
	if err != nil {
		return evidencepwa.Projection{}, err
	}
	availability := evidencepwa.AvailabilityReady
	if manifest.Policy.RedactionState == evidenceartifact.RedactionBlocked {
		availability = evidencepwa.AvailabilityBlocked
	}
	total := uint64(len(claims))
	if total == 0 {
		total = 1
	}
	projection := evidencepwa.Projection{
		Schema: evidencepwa.ProjectionSchema, ProjectionID: "uiai-evidence-projection:sha256:" + manifest.Integrity.ManifestSHA256,
		Artifact: evidencepwa.ArtifactBinding{
			ArtifactRef: manifest.ArtifactID, Revision: manifest.Revision, ManifestSHA256: manifest.Integrity.ManifestSHA256,
			BundleSHA256: manifest.Integrity.BundleSHA256,
			Scope: evidencepwa.ScopeBinding{
				ProjectRef: manifest.Scope.Project.ProjectRef, WorkstreamRef: manifest.Scope.Workstream.WorkstreamRef,
				WorksetRef: manifest.Scope.Workset.WorksetRef, CallGraphRef: manifest.Scope.CallGraph.FrameRef,
				WorkpointRef: manifest.Scope.Workpoint.WorkpointRef, WorkItemRef: manifest.Scope.WorkItems[0].WorkItemRef,
			},
		},
		WorkItems: workItems, Title: manifest.Title, Summary: artifactSummary(manifest),
		Sections: []evidencepwa.Section{{ID: evidencepwa.SectionOverview, Heading: "Overview"}, {ID: evidencepwa.SectionEvidence, Heading: "Evidence"}, {ID: evidencepwa.SectionTimeline, Heading: "Timeline"}, {ID: evidencepwa.SectionInspect, Heading: "Inspect"}, {ID: evidencepwa.SectionDeveloper, Heading: "Developer"}},
		Claims:   claims, Assets: assets, Timeline: timeline, InspectionRefs: append([]string(nil), manifest.Security.InspectionReceiptRefs...),
		SecurityRefs: securityRefs, OmissionRefs: append([]string(nil), manifest.Provenance.OmissionRefs...), RelatedArtifactRefs: append([]string(nil), manifest.Links.RelatedRefs...), ReceiptRefs: append([]string(nil), manifest.ReceiptRefs...),
		Availability: availability, Access: access, Redaction: redaction, FederationPosture: "source_local",
		FreshnessObservedAt: capturedAt, Page: evidencepwa.PageInfo{PageSize: uint32(min(total, evidencepwa.MaxPageSize)), TotalCount: total},
		Links: []evidencepwa.RelativeLink{{Rel: "manifest", Href: "./artifact.json"}, {Rel: "projection", Href: "./projection.json"}}, Interaction: evidencepwa.InteractionReadOnly,
	}
	if err := evidencepwa.ValidateProjection(projection); err != nil {
		return evidencepwa.Projection{}, err
	}
	return projection, nil
}

func projectAccess(value evidenceartifact.AccessClass) evidencepwa.AccessPosture {
	switch value {
	case evidenceartifact.AccessLocal:
		return evidencepwa.AccessLocalhost
	case evidenceartifact.AccessLAN:
		return evidencepwa.AccessLAN
	case evidenceartifact.AccessTailnet:
		return evidencepwa.AccessTailnet
	case evidenceartifact.AccessUnlisted:
		return evidencepwa.AccessUnlisted
	case evidenceartifact.AccessPublicSafe:
		return evidencepwa.AccessPublicSafe
	default:
		return evidencepwa.AccessPrivate
	}
}

func projectRedaction(manifest evidenceartifact.Manifest) (evidencepwa.Redaction, []string, error) {
	securityRefs := uniqueStrings(append(append([]string(nil), manifest.Security.SanitizationRefs...), manifest.Security.RedactionRefs...))
	result := evidencepwa.Redaction{State: evidencepwa.RedactionNotRequired}
	switch manifest.Policy.RedactionState {
	case evidenceartifact.RedactionNone:
		return result, securityRefs, nil
	case evidenceartifact.RedactionRedacted:
		result.State = evidencepwa.RedactionPartiallyApplied
	case evidenceartifact.RedactionPublicSafe:
		result.State = evidencepwa.RedactionApplied
	case evidenceartifact.RedactionBlocked:
		result.State = evidencepwa.RedactionBlocked
	default:
		return evidencepwa.Redaction{}, nil, ErrInvalidInput
	}
	if result.State == evidencepwa.RedactionApplied || result.State == evidencepwa.RedactionPartiallyApplied {
		if len(manifest.Security.RedactionRefs) == 0 {
			return evidencepwa.Redaction{}, nil, ErrInvalidInput
		}
		result.EvidenceRef = manifest.Security.RedactionRefs[0]
	}
	return result, securityRefs, nil
}

func artifactSummary(manifest evidenceartifact.Manifest) string {
	if strings.TrimSpace(manifest.Summary) != "" {
		return manifest.Summary
	}
	return "Bound immutable evidence artifact. Delivery does not establish review, completion, provider closure, or settlement."
}

func addShellAssets(writes map[string][]byte) error {
	assetsSHA := embeddedAssetsSHA256()
	if len(assetsSHA) < 12 {
		return ErrInvalidInput
	}
	assetVersion := assetsSHA[:12]
	for _, name := range packagedAssetNames {
		data, err := assets.ReadFile("assets/" + name)
		if err != nil {
			return err
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
	return nil
}

func writePackageFiles(root string, writes map[string][]byte) error {
	names := make([]string, 0, len(writes))
	for name := range writes {
		if !safePackagePath(name) {
			return ErrInvalidInput
		}
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		target := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
			return err
		}
		if err := os.WriteFile(target, writes[name], 0o640); err != nil {
			return err
		}
	}
	return nil
}

func verifyPackageFiles(root string, expected map[string][]byte) error {
	for name, want := range expected {
		got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name))) // #nosec G304 -- manifest path is validated.
		if err != nil || !equalBytes(got, want) {
			return ErrConflict
		}
	}
	return nil
}

func safePackagePath(value string) bool {
	return value != "" && !strings.ContainsAny(value, "\\\r\n\x00:*?\"<>|") && !strings.HasPrefix(value, "/") && path.Clean(value) == value && value != "." && value != ".." && !strings.HasPrefix(value, "../")
}

func validDigest(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size && value == strings.ToLower(value)
}

func digestBytes(value []byte) string {
	digest := sha256.Sum256(value)
	return hex.EncodeToString(digest[:])
}

func equalBytes(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
