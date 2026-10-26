package evidencejudge

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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

func TestT06ACrossAtomEvidenceCannotVerify(t *testing.T) {
	for name, mutate := range map[string]func(*JudgeResult){
		"decision":  func(r *JudgeResult) { r.AtomDecisions[0].CitationIDs = []string{"citation:video"} },
		"rationale": func(r *JudgeResult) { r.Rationales[0].CitationIDs = []string{"citation:video"} },
	} {
		t.Run(name, func(t *testing.T) {
			v, q, r := validContracts(t)
			mutate(&r)
			if !errors.Is(ValidateJudgeResultAgainst(r, q, v), ErrCitationInvalid) {
				t.Fatal("unrelated atom evidence accepted")
			}
		})
	}
}

func TestT06AExpiryAndAllowedVerdictsBind(t *testing.T) {
	v, q, r := validContracts(t)
	r.EvaluatedAt = q.ExpiresAt
	if !errors.Is(ValidateJudgeResultAgainst(r, q, v), ErrJudgeExpired) {
		t.Fatal("result evaluated at expired boundary accepted")
	}
	v, q, r = validContracts(t)
	v.AllowedVerdicts = []Verdict{VerdictBlocked}
	q.ViewSHA256, _ = DigestJudgeView(v)
	r.ViewSHA256 = q.ViewSHA256
	r.RequestSHA256, _ = DigestJudgeRequest(q)
	if !errors.Is(ValidateJudgeResultAgainst(r, q, v), ErrJudgeResultInvalid) {
		t.Fatal("policy-forbidden supported verdict accepted")
	}
}

func TestT06ALocatorCannotInventSourceModality(t *testing.T) {
	for name, locator := range map[string]CitationLocator{
		"image_with_media_time": {Type: LocatorMediaTime, Start: 1, End: 2},
		"image_with_receipt":    {Type: LocatorReceiptRef, Value: "receipt:1"},
		"image_with_custody":    {Type: LocatorCustodyOrdinal, Start: 1},
	} {
		t.Run(name, func(t *testing.T) {
			v, _, _ := validContracts(t)
			v.Citations[0].Locator = locator
			if !errors.Is(ValidateJudgeView(v), ErrCitationInvalid) {
				t.Fatal("locator invented unsupported modality")
			}
		})
	}
}

func TestT06ADeclaredBoundsAndAmbiguousRequirements(t *testing.T) {
	v, _, _ := validContracts(t)
	v.AcceptanceAtoms[0].RequiredModalities = append(v.AcceptanceAtoms[0].RequiredModalities, ModalityStaticImage)
	if ValidateJudgeView(v) == nil {
		t.Fatal("duplicate required modality accepted")
	}
	v, q, _ := validContracts(t)
	q.RequiredModalities[0].AtomRef = "atom:missing"
	if ValidateJudgeRequest(q) == nil {
		t.Fatal("request contains dangling requirement atom")
	}
	// Distinct requirement IDs may require different evidence for the same atom/modality.
	v, _, _ = validContracts(t)
	v.Artifact.TrustRefs = nil
	for i := 0; i <= MaxRefs; i++ {
		v.Artifact.TrustRefs = append(v.Artifact.TrustRefs, fmt.Sprintf("trust:%d", i))
	}
	if ValidateJudgeView(v) == nil {
		t.Fatal("unbounded artifact refs accepted")
	}
	_, q, _ = validContracts(t)
	q.RequiredModalities[0].CitationIDs = nil
	for i := 0; i <= MaxCitations; i++ {
		q.RequiredModalities[0].CitationIDs = append(q.RequiredModalities[0].CitationIDs, fmt.Sprintf("citation:%d", i))
	}
	if ValidateJudgeRequest(q) == nil {
		t.Fatal("unbounded request citations accepted")
	}
}

func TestT06ACitationPolarityMustMatchVerdict(t *testing.T) {
	v, q, r := validContracts(t)
	v.Citations[0].SupportsAtoms = nil
	v.Citations[0].RebutsAtoms = []string{"atom:layout"}
	v.Modalities[0].Status = ModalityBlocked
	v.Modalities[0].CitationIDs = nil
	q.RequiredModalities = append([]ModalityRequirement(nil), v.Modalities...)
	q.ViewSHA256, _ = DigestJudgeView(v)
	r.ViewSHA256 = q.ViewSHA256
	r.RequestSHA256, _ = DigestJudgeRequest(q)
	r.AtomDecisions[0].Verdict = VerdictRebutted
	r.Outcome = OutcomeRejected
	if err := ValidateJudgeResultAgainst(r, q, v); err != nil {
		t.Fatalf("legitimate rebuttal rejected: %v", err)
	}
	r.AtomDecisions[0].CitationIDs = []string{"citation:video"}
	if !errors.Is(ValidateJudgeResultAgainst(r, q, v), ErrCitationInvalid) {
		t.Fatal("unrelated rebuttal citation accepted")
	}
}

func TestT06AGoldenRoundTrips(t *testing.T) {
	v, _, _ := validContracts(t)
	for _, name := range []string{"judge-request.golden.json", "judge-result.golden.json"} {
		raw, err := os.ReadFile(filepath.Join("testdata", name))
		if err != nil {
			t.Fatal(err)
		}
		var value any
		if name == "judge-request.golden.json" {
			var q JudgeRequest
			if err = json.Unmarshal(raw, &q); err != nil {
				t.Fatal(err)
			}
			if err = ValidateJudgeRequestAgainst(q, v); err != nil {
				t.Fatal(err)
			}
			value = q
		} else {
			var r JudgeResult
			if err = json.Unmarshal(raw, &r); err != nil {
				t.Fatal(err)
			}
			_, q, _ := validContracts(t)
			if err = ValidateJudgeResultAgainst(r, q, v); err != nil {
				t.Fatal(err)
			}
			value = r
		}
		next, err := json.MarshalIndent(value, "", "  ")
		if err != nil || !bytes.Equal(append(next, '\n'), raw) {
			t.Fatal("golden roundtrip drift")
		}
	}
}

func TestT06AOpaqueRefsRejectPrivateCredentialURLs(t *testing.T) {
	for _, bad := range []string{"/tmp/private", "file:///private/source", `C:\\private\\source`, "C:/private/source", "https://user:fixture@example.invalid/path?token=fixture"} {
		v, q, r := validContracts(t)
		v.Artifact.Scope.WorkItemRef = bad
		if ValidateJudgeView(v) == nil {
			t.Fatal("unsafe lineage ref accepted")
		}
		q.AssignmentRef = bad
		if ValidateJudgeRequest(q) == nil {
			t.Fatal("unsafe request ref accepted")
		}
		r.Rationales[0].CitationIDs = []string{bad}
		if ValidateJudgeResult(r) == nil {
			t.Fatal("unsafe rationale citation ref accepted")
		}
	}
	// JSON pointers are selectors, not filesystem refs; their root-relative syntax remains valid.
	v, _, _ := validContracts(t)
	v.Sources[0].Modality = ModalityStructuredData
	v.Citations[0].Modality = ModalityStructuredData
	v.Citations[0].Locator = CitationLocator{Type: LocatorJSONPointer, Value: "/items/a~1b/~0"}
	v.Modalities[0].Modality = ModalityStructuredData
	v.AcceptanceAtoms[0].RequiredModalities = []Modality{ModalityStructuredData}
	if err := ValidateJudgeView(v); err != nil {
		t.Fatal(err)
	}
	for _, ptr := range []string{"no-leading-slash", "/bad~2", "/bad~"} {
		v.Citations[0].Locator.Value = ptr
		if !errors.Is(ValidateJudgeView(v), ErrCitationInvalid) {
			t.Fatal("invalid JSON pointer accepted")
		}
	}
}

func TestT06ALocatorAndNestedCitationBounds(t *testing.T) {
	for _, locator := range []CitationLocator{{Type: LocatorAssetID, Value: strings.Repeat("x", MaxLocatorRunes+1)}, {Type: LocatorMediaTime, Start: 1, End: MaxLocatorOffset + 1}} {
		if validLocator(locator) {
			t.Fatal("unbounded locator accepted")
		}
	}
	_, _, r := validContracts(t)
	r.AtomDecisions[0].CitationIDs = nil
	for i := 0; i <= MaxCitations; i++ {
		r.AtomDecisions[0].CitationIDs = append(r.AtomDecisions[0].CitationIDs, fmt.Sprintf("citation:%d", i))
	}
	if ValidateJudgeResult(r) == nil {
		t.Fatal("unbounded decision citations accepted")
	}
	oversized := r.AtomDecisions[0].CitationIDs
	_, _, r = validContracts(t)
	r.Rationales[0].CitationIDs = oversized
	if ValidateJudgeResult(r) == nil {
		t.Fatal("unbounded rationale citations accepted")
	}
	// A valid immutable artifact anchor remains admissible, without altering its digest spelling.
	if !validJudgeRef("github:WPUIAI/uiai-engine#115") {
		t.Fatal("existing issue handle rejected")
	}
}

func TestT06AAllLineageFieldsInvalidateOldRequest(t *testing.T) {
	for _, mutate := range []func(*ScopeBinding){func(s *ScopeBinding) { s.ProjectRef = "project:other" }, func(s *ScopeBinding) { s.WorkstreamRef = "workstream:other" }, func(s *ScopeBinding) { s.WorksetRef = "workset:other" }, func(s *ScopeBinding) { s.CallGraphRef = "callgraph:other" }, func(s *ScopeBinding) { s.WorkpointRef = "workpoint:other" }, func(s *ScopeBinding) { s.WorkItemRef = "item:other" }} {
		v, q, r := validContracts(t)
		mutate(&v.Artifact.Scope)
		if !errors.Is(ValidateJudgeResultAgainst(r, q, v), ErrJudgeRequestInvalid) {
			t.Fatal("changed lineage retained previous result")
		}
	}
}

func TestT06APolicyVerdictRejectionAndPartialCitations(t *testing.T) {
	v, q, r := validContracts(t)
	v.AllowedVerdicts = []Verdict{VerdictBlocked}
	q.ViewSHA256, _ = DigestJudgeView(v)
	r.ViewSHA256 = q.ViewSHA256
	r.RequestSHA256, _ = DigestJudgeRequest(q)
	if !errors.Is(ValidateJudgeResultAgainst(r, q, v), ErrJudgeResultInvalid) {
		t.Fatal("policy-forbidden verdict accepted")
	}
	// Judges may reject evidence even when its producer tagged it as supporting: do not inherit producer verdicts.
	v, q, r = validContracts(t)
	r.Outcome = OutcomeRejected
	r.AtomDecisions[0].Verdict = VerdictRebutted
	if err := ValidateJudgeResultAgainst(r, q, v); err != nil {
		t.Fatalf("independent disagreement suppressed: %v", err)
	}
}
