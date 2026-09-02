package evidenceshare

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
)

func TestAssemblePortableSharePackage(t *testing.T) {
	root := t.TempDir()
	input := validInput()
	result, err := Assemble(root, input)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(result.ArtifactRef, "uiai-evidence-share:sha256:") || !strings.HasPrefix(result.RelativePath, "./") {
		t.Fatal("invalid portable identity")
	}
	for _, name := range []string{"artifact.json", "projection.json", "inspection.json", "screenshot.png", "index.html", "styles.css", "app.js"} {
		if info, err := os.Stat(filepath.Join(result.Directory, name)); err != nil || info.Size() == 0 {
			t.Fatalf("missing %s: %v", name, err)
		}
	}
	indexBody, _ := os.ReadFile(filepath.Join(result.Directory, "index.html"))
	if strings.Contains(string(indexBody), "__UIAI_ASSET_VERSION__") || !strings.Contains(string(indexBody), "?v="+embeddedAssetsSHA256()[:12]) {
		t.Fatal("generated page does not bind its exact asset revision")
	}
	body, _ := os.ReadFile(filepath.Join(result.Directory, "artifact.json"))
	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.SourceURL != "https://example.com/path" {
		t.Fatalf("source URL leaked credentials/query: %q", manifest.SourceURL)
	}
	if manifest.Interaction != "read_only" || manifest.Availability != "ready" || manifest.ProjectionRef != "./projection.json" {
		t.Fatal("truth posture drift")
	}
	projectionBody, _ := os.ReadFile(filepath.Join(result.Directory, "projection.json"))
	var projection evidencepwa.Projection
	if err := json.Unmarshal(projectionBody, &projection); err != nil || evidencepwa.ValidateProjection(projection) != nil {
		t.Fatal("canonical evidence PWA projection missing or invalid")
	}
	second, err := Assemble(root, input)
	if err != nil || second != result {
		t.Fatalf("assembly not idempotent: %#v %v", second, err)
	}
}

func TestAssembleProjectsAllCanonicalWorkItems(t *testing.T) {
	input := validInput()
	input.Scope.WorkItemRef = "work-item:task"
	input.Scope.WorkItems = []evidencepwa.WorkItemProjection{
		{
			ProviderSurface: "github", WorkItemRef: "work-item:task", ItemID: "task", ItemType: "task",
			Title: "Implement projection", Description: "Project real work items.", DescriptionState: evidencepwa.WorkItemDescriptionVisible,
			Revision: "revision:7", Digest: strings.Repeat("a", 64), RevisionState: evidencepwa.WorkItemRevisionCurrent,
			StatusAtCapture: "in_progress", DependencyRefs: []string{"work-item:dependency"}, ClosurePosture: "evidence_pending",
			Authority: evidencepwa.WorkItemAuthorityState{AcceptanceAtomRefs: []string{"atom:projection"}},
		},
		{
			ProviderSurface: "github", WorkItemRef: "work-item:epic", ItemID: "epic", ItemType: "epic",
			Title: "Evidence PWA", DescriptionRef: "description:epic", DescriptionSHA256: strings.Repeat("b", 64),
			DescriptionState: evidencepwa.WorkItemDescriptionRedacted, Revision: "revision:3", Digest: strings.Repeat("c", 64),
			RevisionState: evidencepwa.WorkItemRevisionStale, StatusAtCapture: "open", ClosurePosture: "reopened",
			Authority: evidencepwa.WorkItemAuthorityState{CompletionCaseRef: "completion-case:epic", ReopenRef: "reopen:epic"},
		},
	}
	result, err := Assemble(t.TempDir(), input)
	if err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(result.Directory, "projection.json"))
	if err != nil {
		t.Fatal(err)
	}
	var projection evidencepwa.Projection
	if err := json.Unmarshal(body, &projection); err != nil {
		t.Fatal(err)
	}
	items := projection.Artifact.Scope.WorkItems
	if len(items) != 2 || items[0].WorkItemRef != "work-item:task" || items[1].WorkItemRef != "work-item:epic" {
		t.Fatalf("canonical work items missing or reordered: %+v", items)
	}
	if items[1].Description != "" || items[1].DescriptionState != evidencepwa.WorkItemDescriptionRedacted ||
		items[1].Authority.CompletionCaseRef == "" || items[1].Authority.ReopenRef == "" {
		t.Fatalf("redaction or closure states collapsed: %+v", items[1])
	}
}

func TestAssembleIncompleteScopeIsExplicitlyBlocked(t *testing.T) {
	input := validInput()
	input.Scope = Scope{WorkpointRef: "workpoint:homepage"}
	result, err := Assemble(t.TempDir(), input)
	if err != nil {
		t.Fatal(err)
	}
	body, _ := os.ReadFile(filepath.Join(result.Directory, "artifact.json"))
	var manifest Manifest
	if err := json.Unmarshal(body, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Availability != "blocked" || manifest.ProjectionRef != "" {
		t.Fatal("incomplete lineage did not fail closed")
	}
	if _, err := os.Stat(filepath.Join(result.Directory, "projection.json")); !os.IsNotExist(err) {
		t.Fatal("projection emitted from incomplete lineage")
	}
}

func TestAssembleFailsClosed(t *testing.T) {
	input := validInput()
	input.Screenshot = nil
	if !errors.Is(assemble(t, input), ErrInvalidInput) {
		t.Fatal("empty screenshot accepted")
	}
	input = validInput()
	input.Format = "svg"
	if !errors.Is(assemble(t, input), ErrInvalidInput) {
		t.Fatal("unsafe format accepted")
	}
	input = validInput()
	result, err := Assemble(t.TempDir(), input)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(result.Directory, "artifact.json"), []byte("{}"), 0o640); err != nil {
		t.Fatal(err)
	}
	_, err = Assemble(filepath.Dir(result.Directory), input)
	if !errors.Is(err, ErrConflict) {
		t.Fatal("mismatched existing package overwritten")
	}
}

func TestEmbeddedPageIsSafePortableAndResponsive(t *testing.T) {
	if digest := embeddedAssetsSHA256(); len(digest) != 64 {
		t.Fatalf("embedded asset digest invalid: %q", digest)
	}
	for _, name := range []string{"index.html", "styles.css", "work-items.js", "app.js"} {
		body, err := assets.ReadFile("assets/" + name)
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		for _, forbidden := range []string{"https://", "http://", "innerHTML", "eval(", "/home/", "/root/"} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s contains %q", name, forbidden)
			}
		}
	}
	workItems, err := assets.ReadFile("assets/work-items.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, required := range []string{"scopeWorkItems(scope)", "item.parent_refs", "authority.acceptance_atom_refs", "authority.completion_case_ref", "authority.settlement_posture"} {
		if !strings.Contains(string(workItems), required) {
			t.Fatalf("work-items.js does not render canonical work-item field %q", required)
		}
	}
	html, _ := assets.ReadFile("assets/index.html")
	htmlText := string(html)
	if strings.Index(htmlText, "work-items.js") >= strings.Index(htmlText, "app.js") {
		t.Fatal("work-item renderer must load before the main application")
	}
	if strings.Count(htmlText, "<article") != 1 || strings.Count(htmlText, "</article>") != 1 {
		t.Fatal("semantic shell must contain exactly one evidence article")
	}
	cursor := -1
	for _, id := range []string{"overview", "evidence", "timeline", "inspect", "developer"} {
		next := strings.Index(htmlText, `id="`+id+`"`)
		if next <= cursor || strings.Count(htmlText, `id="`+id+`"`) != 1 {
			t.Fatalf("semantic section %s missing, duplicate, or out of order", id)
		}
		cursor = next
	}
	for _, required := range []string{`data-epwa-status="loading"`, `data-interaction="read-only"`, `id="registry"`, `id="registry-project"`, `id="registry-rows"`, `id="record-detail"`, `id="registry-back"`, `id="primary-evidence-frame"`, "UIAI <b>×</b> Focusa", "Independent states—not one “valid” badge", "not automatically legally admissible"} {
		if !strings.Contains(htmlText, required) {
			t.Fatalf("semantic shell missing %s", required)
		}
	}
	for _, layer := range []string{"integrity", "provenance", "observation", "sufficiency", "verification", "completion", "settlement", "legal"} {
		if strings.Count(htmlText, `data-validity-layer="`+layer+`"`) != 1 {
			t.Fatalf("validity layer %s missing or duplicated", layer)
		}
	}
	if strings.Contains(htmlText, "Evidence ready") {
		t.Fatal("renderer must not collapse validity layers into a success badge")
	}
	app, _ := assets.ReadFile("assets/app.js")
	for _, required := range []string{"/api/evidence/registry/public", "/projects", "/artifacts", "/work-items", "/edges", "/sync-status", "/events", "EventSource", "sessionStorage", "uiai.public_evidence_artifact_detail.v1", "artifactViewURL", "renderPublicRecord"} {
		if !strings.Contains(string(app), required) {
			t.Fatalf("registry consumer missing %s", required)
		}
	}
	for _, forbidden := range []string{"/api/evidence/registry/closure", "/api/evidence/registry/sync?"} {
		if strings.Contains(string(app), forbidden) {
			t.Fatalf("public consumer references private authority endpoint %s", forbidden)
		}
	}
	css, _ := assets.ReadFile("assets/styles.css")
	for _, required := range []string{"clamp(", "grid-template-columns:repeat(4", "prefers-color-scheme:dark", "prefers-reduced-motion:reduce", "overflow-x:hidden", `.registry-table td:nth-child(2){grid-column:1/-1`, ".record-navigation"} {
		if !strings.Contains(string(css), required) {
			t.Fatalf("responsive CSS missing %s", required)
		}
	}
	for _, forbidden := range []string{"min-width:720px", `.registry[data-registry-state="unavailable"] .registry-table-wrap{display:none}`, ".record-heading{.record-heading"} {
		if strings.Contains(string(css), forbidden) {
			t.Fatalf("responsive CSS retains forbidden artifact-detail/registry rule %s", forbidden)
		}
	}
}

func TestVisualProofFixture(t *testing.T) {
	inputPath, root := os.Getenv("UIAI_SHARE_VISUAL_INPUT"), os.Getenv("UIAI_SHARE_VISUAL_ROOT")
	if inputPath == "" || root == "" {
		t.Skip("visual proof fixture not requested")
	}
	data, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Assemble(root, Input{Screenshot: data, Format: "png", Width: 1440, Height: 1000, SourceURL: "https://focusa.dev/", CapturedAt: time.Date(2026, 8, 30, 2, 0, 0, 0, time.UTC), DurationMS: 87, Scope: completeScope()})
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("VISUAL_SHARE_DIR=%s\n", result.Directory)
}

func validInput() Input {
	return Input{Screenshot: append([]byte("\x89PNG\r\n\x1a\n"), []byte("bounded-pixel-data")...), Format: "png", Width: 1440, Height: 900, SourceURL: "https://user:secret@example.com/path?token=secret#private", CapturedAt: time.Date(2026, 8, 30, 1, 2, 3, 0, time.UTC), DurationMS: 87, Scope: completeScope()}
}
func completeScope() Scope {
	return Scope{ProjectRef: "project:focusa", WorkstreamRef: "workstream:epwa", WorksetRef: "workset:t08", CallGraphRef: "callgraph:133", WorkpointRef: "workpoint:t08b", WorkItemRef: "work-item:screenshot-share", ContinuityRef: "epwa-t08b"}
}
func assemble(t *testing.T, input Input) error {
	t.Helper()
	_, err := Assemble(t.TempDir(), input)
	return err
}
