package evidencepwa

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
)

func TestProjectWorkItemsPreservesCanonicalIdentityAndExplicitStates(t *testing.T) {
	bindings := testWorkItemBindings()
	original := cloneWorkItemBindings(t, bindings)
	policies := []WorkItemProjectionPolicy{
		{WorkItemRef: bindings[0].WorkItemRef, DescriptionState: WorkItemDescriptionVisible, RevisionState: WorkItemRevisionCurrent,
			CompletionContractRef: "completion-contract:task", CompletionCaseRef: "completion-case:task", SettlementPosture: "unsettled"},
		{WorkItemRef: bindings[1].WorkItemRef, DescriptionState: WorkItemDescriptionRedacted, RevisionState: WorkItemRevisionStale,
			CompletionDecisionRef: "completion-decision:epic", ProviderCloseReceiptRef: "provider-receipt:epic", ReopenRef: "reopen:epic", SettlementPosture: "reopened"},
	}

	items, err := ProjectWorkItems(bindings, policies)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].WorkItemRef != bindings[0].WorkItemRef || items[1].WorkItemRef != bindings[1].WorkItemRef {
		t.Fatalf("canonical ordering changed: %+v", items)
	}
	if items[0].Description != bindings[0].Description || items[0].DescriptionState != WorkItemDescriptionVisible {
		t.Fatalf("visible description projection mismatch: %+v", items[0])
	}
	if items[1].Description != "" || items[1].DescriptionState != WorkItemDescriptionRedacted ||
		items[1].DescriptionRef != bindings[1].DescriptionRef || items[1].DescriptionSHA256 != bindings[1].DescriptionSHA256 {
		t.Fatalf("redacted description leaked or lost identity: %+v", items[1])
	}
	if items[1].RevisionState != WorkItemRevisionStale || items[1].Authority.CompletionDecisionRef == "" ||
		items[1].Authority.ProviderCloseReceiptRef == "" || items[1].Authority.ReopenRef == "" {
		t.Fatalf("authority states collapsed: %+v", items[1])
	}
	if !reflect.DeepEqual(bindings, original) {
		t.Fatal("projection mutated canonical bindings")
	}
	items[0].ParentRefs[0] = "work-item:mutated"
	items[0].Authority.AcceptanceAtomRefs[0] = "atom:mutated"
	if !reflect.DeepEqual(bindings, original) {
		t.Fatal("projected slices alias canonical bindings")
	}
}

func TestProjectionAcceptsRichWorkItemsWithoutBreakingLegacyReader(t *testing.T) {
	projection := validProjection()
	items, err := ProjectWorkItems(testWorkItemBindings(), []WorkItemProjectionPolicy{
		{WorkItemRef: "work-item:task", DescriptionState: WorkItemDescriptionVisible, RevisionState: WorkItemRevisionCurrent},
		{WorkItemRef: "work-item:epic", DescriptionState: WorkItemDescriptionRedacted, RevisionState: WorkItemRevisionStale},
	})
	if err != nil {
		t.Fatal(err)
	}
	projection.Artifact.Scope.WorkItemRef = items[0].WorkItemRef
	projection.WorkItems = items
	if err := ValidateProjection(projection); err != nil {
		t.Fatal(err)
	}

	body, err := json.Marshal(projection)
	if err != nil {
		t.Fatal(err)
	}
	var legacy struct {
		Artifact struct {
			Scope struct {
				WorkItemRef string `json:"work_item_ref"`
			} `json:"scope"`
		} `json:"artifact"`
	}
	if err := json.Unmarshal(body, &legacy); err != nil {
		t.Fatal(err)
	}
	if legacy.Artifact.Scope.WorkItemRef != items[0].WorkItemRef {
		t.Fatalf("legacy compatibility ref = %q", legacy.Artifact.Scope.WorkItemRef)
	}

	legacyProjection := validProjection()
	if err := ValidateProjection(legacyProjection); err != nil {
		t.Fatalf("legacy projection rejected: %v", err)
	}
}

func TestVisibleDescriptionAcceptsCanonicalInlineDigestWithoutRef(t *testing.T) {
	bindings := testWorkItemBindings()
	digest := sha256.Sum256([]byte(bindings[0].Description))
	bindings[0].DescriptionSHA256 = hex.EncodeToString(digest[:])
	items, err := ProjectWorkItems(bindings, []WorkItemProjectionPolicy{
		{WorkItemRef: bindings[0].WorkItemRef, DescriptionState: WorkItemDescriptionVisible, RevisionState: WorkItemRevisionCurrent},
		{WorkItemRef: bindings[1].WorkItemRef, DescriptionState: WorkItemDescriptionRedacted, RevisionState: WorkItemRevisionStale},
	})
	if err != nil {
		t.Fatal(err)
	}
	if items[0].DescriptionRef != "" || items[0].DescriptionSHA256 != bindings[0].DescriptionSHA256 {
		t.Fatalf("inline description identity changed: %+v", items[0])
	}
}

func TestWorkItemProjectionFailsClosed(t *testing.T) {
	bindings := testWorkItemBindings()
	policies := []WorkItemProjectionPolicy{
		{WorkItemRef: bindings[0].WorkItemRef, DescriptionState: WorkItemDescriptionVisible, RevisionState: WorkItemRevisionCurrent},
		{WorkItemRef: bindings[1].WorkItemRef, DescriptionState: WorkItemDescriptionRedacted, RevisionState: WorkItemRevisionStale},
	}

	if _, err := ProjectWorkItems(bindings, policies[:1]); !errors.Is(err, ErrProjectionBindingMismatch) {
		t.Fatalf("missing policy error = %v", err)
	}
	duplicate := cloneWorkItemBindings(t, bindings)
	duplicate[1].WorkItemRef = duplicate[0].WorkItemRef
	if _, err := ProjectWorkItems(duplicate, policies); !errors.Is(err, ErrProjectionBindingMismatch) {
		t.Fatalf("duplicate work item error = %v", err)
	}
	unsafe := cloneWorkItemBindings(t, bindings)
	unsafe[0].Title = "/home/private/task"
	if _, err := ProjectWorkItems(unsafe, policies); !errors.Is(err, ErrProjectionBindingMismatch) {
		t.Fatalf("private title error = %v", err)
	}
	forgedDescription := cloneWorkItemBindings(t, bindings)
	forgedDescription[0].DescriptionRef = "description:task"
	forgedDescription[0].DescriptionSHA256 = strings.Repeat("f", 64)
	if _, err := ProjectWorkItems(forgedDescription, policies); !errors.Is(err, ErrProjectionBindingMismatch) {
		t.Fatalf("forged description digest error = %v", err)
	}
	missingRedactionIdentity := cloneWorkItemBindings(t, bindings)
	missingRedactionIdentity[1].DescriptionRef = ""
	missingRedactionIdentity[1].DescriptionSHA256 = ""
	if _, err := ProjectWorkItems(missingRedactionIdentity, policies); !errors.Is(err, ErrProjectionBindingMismatch) {
		t.Fatalf("redaction without identity error = %v", err)
	}

	items, err := ProjectWorkItems(bindings, policies)
	if err != nil {
		t.Fatal(err)
	}
	projection := validProjection()
	projection.WorkItems = items
	projection.Artifact.Scope.WorkItemRef = "work-item:not-present"
	if err := ValidateProjection(projection); !errors.Is(err, ErrProjectionBindingMismatch) {
		t.Fatalf("legacy anchor mismatch error = %v", err)
	}
}

func testWorkItemBindings() []evidenceartifact.WorkItemBinding {
	return []evidenceartifact.WorkItemBinding{
		{
			ProviderSurface: "github", WorkItemRef: "work-item:task", ItemID: "task", ItemType: "task",
			Title: "Implement projection", Description: "Project real provider work items.", Revision: "revision:7",
			Digest: strings.Repeat("a", 64), StatusAtCapture: "in_progress", ParentRefs: []string{"work-item:epic"},
			DependencyRefs: []string{"work-item:dependency"}, BlockerRefs: []string{"work-item:blocker"},
			AcceptanceAtomRefs: []string{"atom:projection"}, EvidenceRequirementRefs: []string{"evidence:consumer-test"},
			ReviewRequirementRefs: []string{"review:independent"}, ClosurePosture: "evidence_pending",
		},
		{
			ProviderSurface: "github", WorkItemRef: "work-item:epic", ItemID: "epic", ItemType: "epic",
			Title: "Evidence PWA", DescriptionRef: "description:epic", DescriptionSHA256: strings.Repeat("b", 64),
			Revision: "revision:3", Digest: strings.Repeat("c", 64), StatusAtCapture: "open",
			AcceptanceAtomRefs: []string{"atom:epic"}, ClosurePosture: "reopened",
		},
	}
}

func cloneWorkItemBindings(t *testing.T, input []evidenceartifact.WorkItemBinding) []evidenceartifact.WorkItemBinding {
	t.Helper()
	body, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	var output []evidenceartifact.WorkItemBinding
	if err := json.Unmarshal(body, &output); err != nil {
		t.Fatal(err)
	}
	return output
}
