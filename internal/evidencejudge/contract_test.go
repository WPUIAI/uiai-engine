package evidencejudge

import (
	"bytes"
	"encoding/json"
	"errors"
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
	digestC = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	digestD = "dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"
)

func TestContractsValidateAndMatchGolden(t *testing.T) {
	view, req, result := validContracts(t)
	if err := ValidateJudgeView(view); err != nil {
		t.Fatalf("view: %v", err)
	}
	if err := ValidateJudgeRequestAgainst(req, view); err != nil {
		t.Fatalf("request: %v", err)
	}
	if err := ValidateJudgeResultAgainst(result, req, view); err != nil {
		t.Fatalf("result: %v", err)
	}
	assertGolden(t, "judge-request.golden.json", req)
	assertGolden(t, "judge-result.golden.json", result)
}

func TestContractSchemaAndClosedVocabularyFailClosed(t *testing.T) {
	view, req, result := validContracts(t)
	view.Schema = "uiai.evidence_judge_view.v2"
	if !errors.Is(ValidateJudgeView(view), ErrJudgeViewInvalid) {
		t.Fatal("unknown view schema accepted")
	}
	view, _, _ = validContracts(t)
	view.AllowedVerdicts[0] = Verdict("optimistic")
	if !errors.Is(ValidateJudgeView(view), ErrJudgeViewInvalid) {
		t.Fatal("unknown verdict accepted")
	}
	req.Schema = "unknown"
	if !errors.Is(ValidateJudgeRequest(req), ErrJudgeRequestInvalid) {
		t.Fatal("unknown request schema accepted")
	}
	result.Outcome = JudgeOutcome("complete")
	if !errors.Is(ValidateJudgeResult(result), ErrJudgeResultInvalid) {
		t.Fatal("completion-like outcome accepted")
	}
	view, _, _ = validContracts(t)
	view.Citations[0].Locator.Type = LocatorType("invented")
	if !errors.Is(ValidateJudgeView(view), ErrCitationInvalid) {
		t.Fatal("unknown locator accepted")
	}
	_, _, result = validContracts(t)
	result.ErrorCodes = []JudgeErrorCode{"invented"}
	if !errors.Is(ValidateJudgeResult(result), ErrJudgeResultInvalid) {
		t.Fatal("unknown result error code accepted")
	}
}

func TestCitationReferencesAndModalitiesFailClosed(t *testing.T) {
	view, _, _ := validContracts(t)
	view.Citations[1].CitationID = view.Citations[0].CitationID
	if !errors.Is(ValidateJudgeView(view), ErrCitationInvalid) {
		t.Fatal("duplicate citation accepted")
	}
	view, _, _ = validContracts(t)
	view.Citations[1].SupportsAtoms = []string{"atom:missing"}
	if !errors.Is(ValidateJudgeView(view), ErrCitationInvalid) {
		t.Fatal("dangling atom accepted")
	}
	view, _, _ = validContracts(t)
	view.Modalities[1].CitationIDs = []string{"citation:image"}
	if !errors.Is(ValidateJudgeView(view), ErrModalityUnsatisfied) {
		t.Fatal("static image satisfied temporal claim")
	}
}

func TestDigestAndInformationSetBindingsFailClosed(t *testing.T) {
	view, req, result := validContracts(t)
	req.ViewSHA256 = digestD
	if !errors.Is(ValidateJudgeRequestAgainst(req, view), ErrJudgeRequestInvalid) {
		t.Fatal("view digest mismatch accepted")
	}
	_, req, result = validContracts(t)
	result.InformationSetSHA256 = digestD
	if !errors.Is(ValidateJudgeResultAgainst(result, req, view), ErrInformationSetMismatch) {
		t.Fatal("information set mismatch accepted")
	}
	_, req, result = validContracts(t)
	result.RequestSHA256 = digestD
	if !errors.Is(ValidateJudgeResultAgainst(result, req, view), ErrJudgeResultInvalid) {
		t.Fatal("request digest mismatch accepted")
	}
}

func TestIndependentAssignmentRequired(t *testing.T) {
	view, req, _ := validContracts(t)
	req.VerifierIdentityRef = req.ExecutorIdentityRef
	if !errors.Is(ValidateJudgeRequestAgainst(req, view), ErrJudgeAssignmentInvalid) {
		t.Fatal("self-verification accepted")
	}
	view, req, _ = validContracts(t)
	req.AcceptanceAtomRefs = append(req.AcceptanceAtomRefs, "atom:outside-scope")
	if !errors.Is(ValidateJudgeRequestAgainst(req, view), ErrJudgeRequestInvalid) {
		t.Fatal("scope-expanded atom accepted")
	}
}

func TestVerifiedRequiresSupportedAtomsAndModalities(t *testing.T) {
	view, req, result := validContracts(t)
	result.AtomDecisions[1].Verdict = VerdictBlocked
	if !errors.Is(ValidateJudgeResult(result), ErrJudgeResultInvalid) {
		t.Fatal("verified result with blocked atom accepted")
	}
	view, req, result = validContracts(t)
	view.Modalities[1].Status = ModalityBlocked
	view.Modalities[1].CitationIDs = nil
	if err := ValidateJudgeView(view); err != nil {
		t.Fatalf("blocked view should remain valid: %v", err)
	}
	viewDigest, _ := DigestJudgeView(view)
	req.ViewSHA256 = viewDigest
	req.RequiredModalities = append([]ModalityRequirement(nil), view.Modalities...)
	reqDigest, _ := DigestJudgeRequest(req)
	result.ViewSHA256, result.RequestSHA256 = viewDigest, reqDigest
	if !errors.Is(ValidateJudgeResultAgainst(result, req, view), ErrModalityUnsatisfied) {
		t.Fatal("verified result ignored blocked modality")
	}
}

func TestRationaleIsBoundedAndContainsNoHiddenReasoningField(t *testing.T) {
	_, _, result := validContracts(t)
	result.Rationales = result.Rationales[:1]
	if !errors.Is(ValidateJudgeResult(result), ErrJudgeResultInvalid) {
		t.Fatal("missing atom rationale accepted")
	}
	_, _, result = validContracts(t)
	result.Rationales[0].Summary = strings.Repeat("x", MaxRationaleRunes+1)
	if !errors.Is(ValidateJudgeResult(result), ErrJudgeResultInvalid) {
		t.Fatal("oversized rationale accepted")
	}
	_, _, result = validContracts(t)
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"chain_of_thought", "hidden_reasoning", "private_reasoning"} {
		if bytes.Contains(body, []byte(forbidden)) {
			t.Fatalf("result exposes %s", forbidden)
		}
	}
}

func TestRequiredModalityCoverageCannotDisappear(t *testing.T) {
	view, _, _ := validContracts(t)
	view.Modalities = view.Modalities[:1]
	if !errors.Is(ValidateJudgeView(view), ErrModalityUnsatisfied) {
		t.Fatal("missing required modality accepted")
	}
}

func TestExpiredContractsFailAtExplicitClock(t *testing.T) {
	view, req, _ := validContracts(t)
	if !errors.Is(ValidateJudgeViewAt(view, view.ExpiresAt), ErrJudgeExpired) {
		t.Fatal("expired view accepted")
	}
	if !errors.Is(ValidateJudgeRequestAt(req, req.ExpiresAt), ErrJudgeExpired) {
		t.Fatal("expired request accepted")
	}
}

func TestValidationIsDeterministicAndNonMutating(t *testing.T) {
	view, req, result := validContracts(t)
	viewBefore, reqBefore, resultBefore := deepCopy(view), deepCopy(req), deepCopy(result)
	var first [3]string
	for i := 0; i < 30; i++ {
		if err := ValidateJudgeView(view); err != nil {
			t.Fatal(err)
		}
		if err := ValidateJudgeRequestAgainst(req, view); err != nil {
			t.Fatal(err)
		}
		if err := ValidateJudgeResultAgainst(result, req, view); err != nil {
			t.Fatal(err)
		}
		d1, _ := DigestJudgeView(view)
		d2, _ := DigestJudgeRequest(req)
		d3, _ := DigestJudgeResult(result)
		current := [3]string{d1, d2, d3}
		if i == 0 {
			first = current
		} else if current != first {
			t.Fatal("digest drift")
		}
	}
	if !reflect.DeepEqual(view, viewBefore) || !reflect.DeepEqual(req, reqBefore) || !reflect.DeepEqual(result, resultBefore) {
		t.Fatal("validator mutated caller input")
	}
}

func TestContractSizeBound(t *testing.T) {
	view, _, _ := validContracts(t)
	view.AcceptanceAtoms[0].Question = strings.Repeat("q", MaxContractBytes)
	if !errors.Is(ValidateJudgeView(view), ErrJudgeViewInvalid) && !errors.Is(ValidateJudgeView(view), ErrJudgeBudgetExceeded) {
		t.Fatal("oversized contract accepted")
	}
}

func validContracts(t *testing.T) (JudgeView, JudgeRequest, JudgeResult) {
	t.Helper()
	created := time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC)
	view := JudgeView{
		Schema: JudgeViewSchema, ViewID: "judge-view:1",
		Artifact: ArtifactBinding{ArtifactRef: "artifact:1", Revision: 5, BundleSHA256: digestA, ManifestSHA256: digestB,
			Scope:           ScopeBinding{ProjectRef: "project:1", WorkstreamRef: "workstream:1", WorksetRef: "workset:1", CallGraphRef: "callgraph:1", WorkpointRef: "workpoint:1", WorkItemRef: "github:WPUIAI/uiai-engine#115"},
			AttestationRefs: []string{"attestation:1"}, TrustRefs: []string{"trust:1"}, SecurityRefs: []string{"inspection:1"}},
		CompletionContractRef: "completion-contract:1", CompletionContractRevision: "7",
		AcceptanceAtoms: []AcceptanceAtom{
			{AtomRef: "atom:layout", Revision: 2, Question: "Does the exact image prove the required layout?", RequiredModalities: []Modality{ModalityStaticImage}},
			{AtomRef: "atom:interaction", Revision: 3, Question: "Does synchronized video prove the interaction?", RequiredModalities: []Modality{ModalitySynchronizedVideo}},
		},
		AllowedVerdicts:   []Verdict{VerdictSupported, VerdictRebutted, VerdictInsufficientEvidence, VerdictBlocked, VerdictDisputed},
		InformationSetRef: "information-set:1", InformationSetSHA256: digestC,
		Sources: []EvidenceSource{{SourceRef: "asset:image", SHA256: digestA, Modality: ModalityStaticImage}, {SourceRef: "asset:video", SHA256: digestB, Modality: ModalitySynchronizedVideo}},
		Citations: []Citation{
			{Schema: JudgeCitationSchema, CitationID: "citation:image", SourceRef: "asset:image", SourceSHA256: digestA, Modality: ModalityStaticImage, Locator: CitationLocator{Type: LocatorImageRegion, X: 0.1, Y: 0.1, Width: 0.5, Height: 0.5}, SupportsAtoms: []string{"atom:layout"}},
			{Schema: JudgeCitationSchema, CitationID: "citation:video", SourceRef: "asset:video", SourceSHA256: digestB, Modality: ModalitySynchronizedVideo, Locator: CitationLocator{Type: LocatorMediaTime, Start: 1000, End: 3000}, SupportsAtoms: []string{"atom:interaction"}},
		},
		Modalities: []ModalityRequirement{
			{Schema: JudgeModalitySchema, RequirementID: "modality:layout", AtomRef: "atom:layout", Modality: ModalityStaticImage, Required: true, Status: ModalitySatisfied, CitationIDs: []string{"citation:image"}},
			{Schema: JudgeModalitySchema, RequirementID: "modality:interaction", AtomRef: "atom:interaction", Modality: ModalitySynchronizedVideo, Required: true, Status: ModalitySatisfied, CitationIDs: []string{"citation:video"}},
		},
		Omissions: []Omission{{Ref: "diagnostics:full", ReasonCode: "bounded_view"}},
		Policy:    JudgePolicy{PolicyRef: "judge-policy:strict", PolicyRevision: "4", RubricRef: "rubric:1", IndependenceRequired: true, BlindingProfileRef: "blind:producer-verdict", ContradictionPolicyRef: "contradiction:fail-closed", ForbiddenAssumptions: []string{"uncited_success", "summary_proves_visual"}, RequiredCitations: true},
		CreatedAt: created, ExpiresAt: created.Add(time.Hour),
	}
	viewDigest, err := DigestJudgeView(view)
	if err != nil {
		t.Fatal(err)
	}
	req := JudgeRequest{Schema: JudgeRequestSchema, RequestID: "judge-request:1", IdempotencyRef: "idempotency:1", ViewRef: view.ViewID, ViewSHA256: viewDigest,
		InformationSetRef: view.InformationSetRef, InformationSetSHA256: view.InformationSetSHA256, AssignmentRef: "assignment:1", ExecutorIdentityRef: "agent:executor", VerifierIdentityRef: "agent:judge",
		PolicyRefs: []string{view.Policy.PolicyRef}, AcceptanceAtomRefs: []string{"atom:layout", "atom:interaction"}, RequiredModalities: append([]ModalityRequirement(nil), view.Modalities...),
		Budget: JudgeBudget{MaxTokens: 8000, MaxMediaBytes: 20 << 20, MaxSpendMicros: 500000, MaxDurationMS: 300000}, ExpiresAt: created.Add(30 * time.Minute), ResultDetail: "standard"}
	reqDigest, err := DigestJudgeRequest(req)
	if err != nil {
		t.Fatal(err)
	}
	result := JudgeResult{Schema: JudgeResultSchema, ResultID: "judge-result:1", RequestID: req.RequestID, RequestSHA256: reqDigest, ViewRef: req.ViewRef, ViewSHA256: req.ViewSHA256,
		InformationSetSHA256: req.InformationSetSHA256, JudgeIdentityRef: req.VerifierIdentityRef, ModelProvider: "provider:example", ModelVersion: "model:multimodal-v1", CapabilityDigest: digestD,
		PolicyRevision: view.Policy.PolicyRevision, EvaluatedAt: created.Add(10 * time.Minute), Outcome: OutcomeVerified,
		AtomDecisions: []AtomDecision{{AtomRef: "atom:layout", Verdict: VerdictSupported, CitationIDs: []string{"citation:image"}, ReasonCode: "cited_visual_match"}, {AtomRef: "atom:interaction", Verdict: VerdictSupported, CitationIDs: []string{"citation:video"}, ReasonCode: "cited_temporal_match"}},
		CitationIDs:   []string{"citation:image", "citation:video"},
		Rationales:    []Rationale{{AtomRef: "atom:layout", Summary: "The cited source image shows the required layout.", CitationIDs: []string{"citation:image"}}, {AtomRef: "atom:interaction", Summary: "The cited synchronized segment records the interaction.", CitationIDs: []string{"citation:video"}}},
		ConfidencePPM: 990000}
	return view, req, result
}

func assertGolden(t *testing.T, name string, value any) {
	t.Helper()
	body, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	body = append(body, '\n')
	path := filepath.Join("testdata", name)
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
		t.Fatalf("%s drifted; run UPDATE_GOLDEN=1 go test ./internal/evidencejudge", name)
	}
}

func deepCopy[T any](in T) T {
	body, _ := json.Marshal(in)
	var out T
	_ = json.Unmarshal(body, &out)
	return out
}
