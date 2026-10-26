package evidencepwa

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"golang.org/x/net/html"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

const (
	digestA = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	digestB = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
)

func TestProjectionValidAndGoldenStable(t *testing.T) {
	projection := validProjection()
	if err := ValidateProjection(projection); err != nil {
		t.Fatalf("valid projection: %v", err)
	}
	assertGolden(t, projection)
	first, err := DigestProjection(projection)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 30; i++ {
		current, err := DigestProjection(projection)
		if err != nil {
			t.Fatal(err)
		}
		if current != first {
			t.Fatal("projection digest drift")
		}
	}
}

func TestEveryAvailabilityStateIsExplicit(t *testing.T) {
	states := []AvailabilityState{AvailabilityLoading, AvailabilityReady, AvailabilityUnavailable, AvailabilityBlocked, AvailabilityCorrupt, AvailabilityStale, AvailabilityRedacted, AvailabilityDegraded}
	for _, state := range states {
		projection := validProjection()
		projection.Availability = state
		if err := ValidateProjection(projection); err != nil {
			t.Fatalf("%s: %v", state, err)
		}
	}
	projection := validProjection()
	projection.Availability = "optimistic"
	if !errors.Is(ValidateProjection(projection), ErrAvailabilityInvalid) {
		t.Fatal("unknown availability accepted")
	}
}

func TestSectionIdentityAndOrderAreFrozen(t *testing.T) {
	projection := validProjection()
	projection.Sections[0], projection.Sections[1] = projection.Sections[1], projection.Sections[0]
	if !errors.Is(ValidateProjection(projection), ErrProjectionInvalid) {
		t.Fatal("reordered sections accepted")
	}
	projection = validProjection()
	projection.Sections = projection.Sections[:4]
	if !errors.Is(ValidateProjection(projection), ErrProjectionInvalid) {
		t.Fatal("missing section accepted")
	}
}

func TestDanglingAndDuplicateReferencesFailClosed(t *testing.T) {
	projection := validProjection()
	projection.Claims[0].CitationIDs = []string{"citation:missing"}
	if !errors.Is(ValidateProjection(projection), ErrProjectionReferenceDangling) {
		t.Fatal("dangling citation accepted")
	}
	projection = validProjection()
	projection.Assets = append(projection.Assets, projection.Assets[0])
	if !errors.Is(ValidateProjection(projection), ErrProjectionInvalid) {
		t.Fatal("duplicate asset accepted")
	}
	projection = validProjection()
	projection.Timeline[0].Refs = []string{"unknown:ref"}
	if !errors.Is(ValidateProjection(projection), ErrProjectionReferenceDangling) {
		t.Fatal("dangling timeline ref accepted")
	}
}

func TestUnsafeAndNonPortableReferencesFailClosed(t *testing.T) {
	unsafe := []string{"https://example.com/a.png", "/assets/a.png", "../a.png", "./x/../a.png", "javascript:alert(1)", "data:text/html,x", `C:\\Users\\operator\\a.png`, "file:///tmp/a.png"}
	for _, ref := range unsafe {
		projection := validProjection()
		projection.Assets[0].Ref = ref
		if !errors.Is(ValidateProjection(projection), ErrProjectionInvalid) && !errors.Is(ValidateProjection(projection), ErrRelativeRefInvalid) {
			t.Fatalf("unsafe ref accepted: %q", ref)
		}
	}
	projection := validProjection()
	projection.Summary = "private source /home/operator/report.json"
	if !errors.Is(ValidateProjection(projection), ErrRelativeRefInvalid) {
		t.Fatal("private path accepted")
	}
}

func TestAccessAndRedactionVocabulariesAreClosed(t *testing.T) {
	for _, access := range []AccessPosture{AccessLocalhost, AccessLAN, AccessTailnet, AccessPrivate, AccessUnlisted, AccessPublicSafe, AccessOfflineSnapshot} {
		projection := validProjection()
		projection.Access = access
		if access != AccessPublicSafe {
			projection.Redaction = Redaction{State: RedactionNotRequired}
		}
		if err := ValidateProjection(projection); err != nil {
			t.Fatalf("access %s: %v", access, err)
		}
	}
	for _, state := range []RedactionState{RedactionNotRequired, RedactionApplied, RedactionPartiallyApplied, RedactionBlocked, RedactionUnknown} {
		projection := validProjection()
		projection.Access = AccessPrivate
		projection.Redaction.State = state
		if state != RedactionApplied && state != RedactionPartiallyApplied {
			projection.Redaction.EvidenceRef = ""
		}
		if err := ValidateProjection(projection); err != nil {
			t.Fatalf("redaction %s: %v", state, err)
		}
	}
	projection := validProjection()
	projection.Access = "internet"
	if !errors.Is(ValidateProjection(projection), ErrAccessPostureInvalid) {
		t.Fatal("unknown access accepted")
	}
	projection = validProjection()
	projection.Access = AccessPrivate
	projection.Redaction.State = "best_effort"
	if !errors.Is(ValidateProjection(projection), ErrRedactionInvalid) {
		t.Fatal("unknown redaction accepted")
	}
}

func TestPublicSafeRequiresCompleteRedactionProof(t *testing.T) {
	mutations := []func(*Projection){
		func(p *Projection) { p.Redaction.State = RedactionUnknown },
		func(p *Projection) { p.Redaction.EvidenceRef = "" },
		func(p *Projection) { p.Redaction.PrivateRefCount = 1 },
		func(p *Projection) { p.Redaction.SecretFindingCount = 1 },
		func(p *Projection) { p.Redaction.PIIFindingCount = 1 },
		func(p *Projection) { p.Interaction = InteractionAuthenticatedHandoff; p.HandoffRef = "./handoff" },
	}
	for index, mutate := range mutations {
		projection := validProjection()
		mutate(&projection)
		if err := ValidateProjection(projection); !errors.Is(err, ErrRedactionInvalid) {
			t.Fatalf("mutation %d: %v", index, err)
		}
	}
	projection := validProjection()
	projection.Redaction.EvidenceRef = "inspection:missing"
	if !errors.Is(ValidateProjection(projection), ErrRedactionInvalid) {
		t.Fatal("unbound redaction evidence accepted")
	}
}

func TestOfflineAndPublicProjectionsCannotAdvertiseMutationAuthority(t *testing.T) {
	for _, access := range []AccessPosture{AccessPublicSafe, AccessOfflineSnapshot} {
		projection := validProjection()
		projection.Access = access
		projection.Interaction = InteractionAuthenticatedHandoff
		projection.HandoffRef = "./focusa/handoff"
		if err := ValidateProjection(projection); !errors.Is(err, ErrRedactionInvalid) && !errors.Is(err, ErrAccessPostureInvalid) {
			t.Fatalf("%s mutation accepted: %v", access, err)
		}
	}
}

func TestProjectionBoundsFailClosed(t *testing.T) {
	projection := validProjection()
	projection.Page.PageSize = MaxPageSize + 1
	if !errors.Is(ValidateProjection(projection), ErrProjectionInvalid) {
		t.Fatal("oversized page accepted")
	}
	projection = validProjection()
	projection.Summary = strings.Repeat("x", MaxProjectionBytes)
	if !errors.Is(ValidateProjection(projection), ErrProjectionTooLarge) {
		t.Fatal("oversized projection accepted")
	}
}

func TestValidationDoesNotMutateInput(t *testing.T) {
	projection := validProjection()
	before := deepCopy(projection)
	for i := 0; i < 30; i++ {
		if err := ValidateProjection(projection); err != nil {
			t.Fatal(err)
		}
	}
	if !reflect.DeepEqual(projection, before) {
		t.Fatal("validation mutated caller input")
	}
}

func TestSemanticShellContract(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("..", "..", "web", "evidence", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	decoder := xml.NewDecoder(bytes.NewReader(body))
	decoder.Strict = true
	ids := map[string]int{}
	sections := []string{}
	tags := map[string]int{}
	hrefs := []string{}
	var viewport, status, noscript, article bool
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("HTML parse: %v", err)
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		name := strings.ToLower(start.Name.Local)
		tags[name]++
		attrs := map[string]string{}
		for _, attr := range start.Attr {
			attrs[attr.Name.Local] = attr.Value
		}
		if id := attrs["id"]; id != "" {
			ids[id]++
		}
		if name == "section" {
			sections = append(sections, attrs["id"])
		}
		if name == "a" {
			hrefs = append(hrefs, attrs["href"])
		}
		if name == "meta" && attrs["name"] == "viewport" {
			viewport = true
		}
		if attrs["role"] == "status" && attrs["aria-live"] == "polite" {
			status = true
		}
		if name == "noscript" {
			noscript = true
		}
		if name == "article" {
			article = true
		}
	}
	wantSections := []string{"overview", "evidence", "timeline", "inspect", "developer"}
	if !reflect.DeepEqual(sections, wantSections) || tags["article"] != 1 || !article || !viewport || !status || !noscript {
		t.Fatal("semantic shell landmarks drifted")
	}
	if tags["script"] != 0 || tags["link"] != 0 || tags["iframe"] != 0 {
		t.Fatal("shell gained executable or remote dependency")
	}
	for _, id := range wantSections {
		if ids[id] != 1 {
			t.Fatalf("section id %s count=%d", id, ids[id])
		}
	}
	for _, href := range hrefs {
		if !relativeRef(href) {
			t.Fatalf("non-relative shell href %q", href)
		}
	}
	text := string(body)
	for _, required := range []string{"read-only", "does not by itself prove review, completion, provider closure, or settlement", "Unavailable, blocked, corrupt, stale, redacted, degraded, and ready"} {
		if !strings.Contains(text, required) {
			t.Fatalf("missing shell truth copy %q", required)
		}
	}
}

func validProjection() Projection {
	now := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	return Projection{
		Schema: ProjectionSchema, ProjectionID: "projection:1",
		Artifact: ArtifactBinding{ArtifactRef: "artifact:1", Revision: 5, ManifestSHA256: digestA, BundleSHA256: digestB,
			Scope: ScopeBinding{ProjectRef: "project:1", WorkstreamRef: "workstream:1", WorksetRef: "workset:1", CallGraphRef: "callgraph:1", WorkpointRef: "workpoint:1", WorkItemRef: "github:WPUIAI/uiai-engine#116"}},
		Title: "Responsive evidence proof", Summary: "A bounded read-only projection of immutable evidence.",
		Sections:       []Section{{ID: SectionOverview, Heading: "Overview"}, {ID: SectionEvidence, Heading: "Evidence"}, {ID: SectionTimeline, Heading: "Timeline"}, {ID: SectionInspect, Heading: "Inspect"}, {ID: SectionDeveloper, Heading: "Developer"}},
		Claims:         []Claim{{ClaimID: "claim:layout", Statement: "The required layout is visible.", Posture: "supported", CitationIDs: []string{"citation:layout"}, AssetIDs: []string{"asset:layout"}}},
		Assets:         []Asset{{AssetID: "asset:layout", Ref: "./assets/layout.png", SHA256: digestA, MIME: "image/png", Modality: "static_image", Bytes: 2048}},
		Citations:      []Citation{{CitationID: "citation:layout", SourceRef: "asset:layout", SHA256: digestA, Locator: "region:0.1,0.1,0.5,0.5"}},
		Timeline:       []TimelineEntry{{EntryID: "event:capture", OccurredAt: now, EventType: "evidence_captured", Refs: []string{"claim:layout", "citation:layout"}}},
		InspectionRefs: []string{"inspection:redaction"}, SecurityRefs: []string{"security:1"}, CustodyRefs: []string{"custody:1"}, AttestationRefs: []string{"attestation:1"}, TrustRefs: []string{"trust:1"}, OmissionRefs: []string{"omission:1"}, RelatedArtifactRefs: []string{"artifact:prior"}, ReceiptRefs: []string{"receipt:1"},
		Availability: AvailabilityReady, Access: AccessPublicSafe, Redaction: Redaction{State: RedactionApplied, EvidenceRef: "inspection:redaction"}, FederationPosture: "origin", FreshnessObservedAt: now,
		Warnings: []Warning{{Code: "bounded_projection", Message: "Large diagnostics remain ref-first.", EvidenceRefs: []string{"claim:layout"}}},
		Page:     PageInfo{PageSize: 1, TotalCount: 1}, Links: []RelativeLink{{Rel: "self", Href: "./artifact.json"}, {Rel: "overview", Href: "#overview"}}, Interaction: InteractionReadOnly,
	}
}

func assertGolden(t *testing.T, projection Projection) {
	t.Helper()
	body, err := json.MarshalIndent(projection, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := filepath.Join("testdata", "projection.golden.json")
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, body, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(body, want) {
		t.Fatal("projection golden drifted; run UPDATE_GOLDEN=1 go test ./internal/evidencepwa")
	}
}

func deepCopy[T any](in T) T {
	body, _ := json.Marshal(in)
	var out T
	_ = json.Unmarshal(body, &out)
	return out
}

func TestT08ADecodedUnsafeRelativeRefsFailClosed(t *testing.T) {
	for _, ref := range []string{"%2fetc/passwd", ".%2f%2e%2e/private", "asset%00.png", "asset%5cprivate.png", "asset%0a.png"} {
		p := validProjection()
		p.Assets[0].Ref = ref
		if ValidateProjection(p) == nil {
			t.Fatalf("unsafe encoded asset ref accepted: %q", ref)
		}
	}
	for _, ref := range []string{"./assets/safe%20name.png", "#overview", "./next?cursor=opaque"} {
		if !relativeRef(ref) {
			t.Fatalf("safe portable ref rejected: %q", ref)
		}
	}
}

func TestT08ADeclaredReceiptsAreValidTimelineEvidence(t *testing.T) {
	p := validProjection()
	p.Timeline[0].Refs = []string{p.ReceiptRefs[0], p.CustodyRefs[0], p.InspectionRefs[0]}
	p.Warnings[0].EvidenceRefs = []string{p.TrustRefs[0]}
	if err := ValidateProjection(p); err != nil {
		t.Fatalf("declared immutable evidence refs rejected: %v", err)
	}
	p.Timeline[0].Refs = []string{"receipt:not-declared"}
	if !errors.Is(ValidateProjection(p), ErrProjectionReferenceDangling) {
		t.Fatal("undeclared receipt accepted")
	}
}

func TestT08APaginationAndReadOnlyPostureCannotContradictData(t *testing.T) {
	p := validProjection()
	second := p.Claims[0]
	second.ClaimID = "claim:second"
	p.Claims = append(p.Claims, second)
	p.Page.TotalCount = 2
	if !errors.Is(ValidateProjection(p), ErrProjectionInvalid) {
		t.Fatal("page contains more claims than its declared size")
	}
	p = validProjection()
	p.Access = AccessPrivate
	p.HandoffRef = "./focusa/handoff"
	if !errors.Is(ValidateProjection(p), ErrAccessPostureInvalid) {
		t.Fatal("read-only projection advertises handoff")
	}
	p.Interaction = InteractionAuthenticatedHandoff
	if err := ValidateProjection(p); err != nil {
		t.Fatalf("explicit private authenticated handoff rejected: %v", err)
	}
}

func TestT08AImmutableProjectionGoldenRoundTrips(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "projection.golden.json"))
	if err != nil {
		t.Fatal(err)
	}
	var p Projection
	if err = json.Unmarshal(raw, &p); err != nil {
		t.Fatal(err)
	}
	if err = ValidateProjection(p); err != nil {
		t.Fatal(err)
	}
	next, err := json.MarshalIndent(p, "", "  ")
	if err != nil || !bytes.Equal(append(next, '\n'), raw) {
		t.Fatal("projection golden roundtrip drift")
	}
}

func TestT08APrivatePathTokensAndNestedBounds(t *testing.T) {
	for _, ref := range []string{"/tmp/private", "/etc/private", "/private/client", "C:/private/client", `Z:\\private\\client`} {
		p := validProjection()
		p.Artifact.Scope.WorkItemRef = ref
		if !errors.Is(ValidateProjection(p), ErrRelativeRefInvalid) {
			t.Fatal("private source path token accepted")
		}
	}
	p := validProjection()
	p.ReceiptRefs = nil
	for i := 0; i < MaxRefs; i++ {
		p.ReceiptRefs = append(p.ReceiptRefs, fmt.Sprintf("receipt:%d", i))
	}
	p.Timeline[0].Refs = append([]string{p.Artifact.ArtifactRef}, p.ReceiptRefs...)
	if ValidateProjection(p) == nil {
		t.Fatal("unbounded timeline refs accepted")
	}
	p = validProjection()
	p.Page.NextCursor, p.Page.Cursor = "same", "same"
	if ValidateProjection(p) == nil {
		t.Fatal("cursor loop accepted")
	}
}

func TestT08AHTML5SemanticStructureHasNoExecutableOrNetworkDependencies(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "web", "evidence", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := html.Parse(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	ids := map[string]bool{}
	var skip, lang bool
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			counts[n.Data]++
			attrs := map[string]string{}
			for _, a := range n.Attr {
				attrs[a.Key] = a.Val
				if strings.HasPrefix(a.Key, "on") {
					t.Fatal("inline event handler")
				}
				if a.Key == "src" || a.Key == "href" {
					if !relativeRef(a.Val) {
						t.Fatal("nonportable dependency")
					}
				}
			}
			if n.Data == "html" {
				lang = attrs["lang"] != ""
			}
			if attrs["id"] != "" {
				if ids[attrs["id"]] {
					t.Fatal("duplicate HTML identity")
				}
				ids[attrs["id"]] = true
			}
			if n.Data == "a" && attrs["href"] == "#evidence-artifact" {
				skip = true
			}
			if n.Data == "button" || n.Data == "input" || n.Data == "form" || n.Data == "script" || n.Data == "iframe" || n.Data == "link" {
				t.Fatal("static shell gained executable/mutation dependency")
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	for _, tag := range []string{"html", "head", "title", "body", "header", "nav", "main", "article", "footer", "noscript"} {
		if counts[tag] != 1 {
			t.Fatalf("landmark %s count %d", tag, counts[tag])
		}
	}
	if !lang || !skip {
		t.Fatal("missing language/skip link")
	}
}
