package focusaevidence

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
	"time"
)

var fixtureNow = time.Date(2026, 10, 26, 16, 10, 0, 0, time.UTC)

func goldenHandoff(t testing.TB) Handoff {
	t.Helper()
	body, err := os.ReadFile("testdata/handoff.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	v, err := DecodeHandoff(body)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func goldenReconciliation(t testing.TB) Reconciliation {
	t.Helper()
	body, err := os.ReadFile("testdata/reconciliation.golden.json")
	if err != nil {
		t.Fatal(err)
	}
	v, err := DecodeReconciliation(body)
	if err != nil {
		t.Fatal(err)
	}
	return v
}
func clone[T any](v T) T {
	body, _ := json.Marshal(v)
	var out T
	if err := json.Unmarshal(body, &out); err != nil {
		panic(err)
	}
	return out
}
func fixtureApproval(t testing.TB, h Handoff) OperationApproval {
	t.Helper()
	d, err := DigestOperationBinding(h.Operation)
	if err != nil {
		t.Fatal(err)
	}
	return OperationApproval{BindingSHA256: d, EvidenceRefs: append([]string(nil), h.Operation.ApprovalEvidenceRefs...), ObservedAt: h.Operation.ObservedAt, ExpiresAt: h.Operation.ExpiresAt}
}
func rebind(t testing.TB, r Reconciliation, h Handoff) Reconciliation {
	t.Helper()
	var err error
	r.HandoffSHA256, err = DigestHandoff(h)
	if err != nil {
		t.Fatal(err)
	}
	r.OperationBindingSHA256, err = DigestOperationBinding(h.Operation)
	if err != nil {
		t.Fatal(err)
	}
	r.HandoffRef = h.HandoffID
	r.Scope = h.Artifact.Scope
	r.IdempotencyKey = h.IdempotencyKey
	return r
}
func invalidation(t testing.TB, h Handoff) Invalidation {
	t.Helper()
	d, err := DigestHandoff(h)
	if err != nil {
		t.Fatal(err)
	}
	return Invalidation{Schema: InvalidationSchema, InvalidationID: "fixture:invalidation:1", HandoffRef: h.HandoffID, HandoffSHA256: d, Scope: h.Artifact.Scope, ArtifactRef: h.Artifact.ArtifactRef, ManifestSHA256: h.Artifact.ManifestSHA256, Reason: ReasonRetraction, EvidenceRefs: []string{"fixture:retraction-evidence:1"}, ObservedAt: fixtureNow, RequestedAction: "reopen_candidate", Operation: h.Operation}
}
func expect(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestGoldenRoundTripAndThirtyRunDeterminism(t *testing.T) {
	h := goldenHandoff(t)
	r := goldenReconciliation(t)
	beforeH, _ := json.Marshal(h)
	beforeR, _ := json.Marshal(r)
	expectedH, _ := DigestHandoff(h)
	expectedR, _ := DigestReconciliation(r)
	for i := 0; i < 30; i++ {
		if err := ValidateHandoffAgainst(h, h.Artifact, fixtureNow); err != nil {
			t.Fatal(err)
		}
		if err := ValidateReconciliationAgainst(r, h, fixtureNow); err != nil {
			t.Fatal(err)
		}
		if err := ValidateOperationBindingAgainst(h.Operation, fixtureApproval(t, h), h.Artifact.Scope, fixtureNow); err != nil {
			t.Fatal(err)
		}
		if d, _ := DigestHandoff(h); d != expectedH {
			t.Fatal("handoff digest drift")
		}
		if d, _ := DigestReconciliation(r); d != expectedR {
			t.Fatal("result digest drift")
		}
		bh, _ := json.Marshal(h)
		br, _ := json.Marshal(r)
		if !bytes.Equal(bh, beforeH) || !bytes.Equal(br, beforeR) {
			t.Fatal("validator mutated input")
		}
		if _, err := DecodeHandoff(bh); err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeReconciliation(br); err != nil {
			t.Fatal(err)
		}
	}
	for _, v := range []any{h.Operation, invalidation(t, h)} {
		body, _ := json.Marshal(v)
		switch v.(type) {
		case OperationBinding:
			if _, err := DecodeOperationBinding(body); err != nil {
				t.Fatal(err)
			}
		case Invalidation:
			if _, err := DecodeInvalidation(body); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Logf("handoff_sha256=%s reconciliation_sha256=%s", expectedH, expectedR)
}

func TestExactLineageArtifactAndMetadataBindings(t *testing.T) {
	base := goldenHandoff(t)
	for i := 0; i < 6; i++ {
		h := clone(base)
		fields := []*string{&h.Artifact.Scope.ProjectRef, &h.Artifact.Scope.WorkstreamRef, &h.Artifact.Scope.WorksetRef, &h.Artifact.Scope.CallGraphRef, &h.Artifact.Scope.WorkpointRef, &h.Artifact.Scope.WorkItemRef}
		*fields[i] += "-foreign"
		h.Operation.AuthorityScope = h.Artifact.Scope
		expect(t, ValidateHandoffAgainst(h, base.Artifact, fixtureNow), ErrLineageBindingMismatch)
	}
	for _, change := range []func(*Handoff){func(h *Handoff) { h.Artifact.ArtifactRef += "-foreign" }, func(h *Handoff) { h.Artifact.Revision++ }, func(h *Handoff) { h.Artifact.ManifestSHA256 = strings.Repeat("b", 64) }, func(h *Handoff) { h.Artifact.BundleSHA256 = strings.Repeat("c", 64) }, func(h *Handoff) { h.Artifact.AttestationRefs = []string{"fixture:other-attestation"} }, func(h *Handoff) { h.Artifact.TrustRefs = []string{"fixture:other-trust"} }, func(h *Handoff) { h.Artifact.SecurityRefs = []string{"fixture:other-inspection"} }} {
		h := clone(base)
		change(&h)
		expect(t, ValidateHandoffAgainst(h, base.Artifact, fixtureNow), ErrArtifactBindingMismatch)
	}
	for _, change := range []func(*Handoff){func(h *Handoff) { h.Artifact.AttestationRefs = nil }, func(h *Handoff) { h.Artifact.TrustRefs = nil }, func(h *Handoff) { h.Artifact.SecurityRefs = nil }, func(h *Handoff) { h.JudgeResultRefs = nil }, func(h *Handoff) { h.AcceptanceAtomRefs = nil }, func(h *Handoff) { h.IdempotencyKey = "" }} {
		h := clone(base)
		change(&h)
		expect(t, ValidateHandoff(h), ErrHandoffContractInvalid)
	}
	h := clone(base)
	h.Operation.AuthorityScope.WorkItemRef += "-foreign"
	expect(t, ValidateHandoff(h), ErrAuthorityScopeMismatch)
}

func TestOperationApprovalIsExactAndUnknownOrStaleIsBlocked(t *testing.T) {
	h := goldenHandoff(t)
	a := fixtureApproval(t, h)
	scope := h.Artifact.Scope
	for _, change := range []func(*OperationBinding){func(v *OperationBinding) { v.OperationID += "-unknown" }, func(v *OperationBinding) { v.OperationVersion = "v2" }, func(v *OperationBinding) { v.RequestSchemaRef += "-foreign" }, func(v *OperationBinding) { v.ResponseSchemaRef += "-foreign" }, func(v *OperationBinding) { v.CapabilitySnapshotRef += "-foreign" }, func(v *OperationBinding) { v.CapabilitySnapshotSHA256 = strings.Repeat("c", 64) }, func(v *OperationBinding) { v.ApprovalEvidenceRefs = nil }, func(v *OperationBinding) { v.ConfirmationRequired = false }, func(v *OperationBinding) { v.IdempotencyRequired = false }, func(v *OperationBinding) { v.ReceiptRequired = false }, func(v *OperationBinding) { v.Consumer = ConsumerDesktop }} {
		v := clone(h.Operation)
		change(&v)
		out := AssessOperation(v, a, scope, fixtureNow)
		if out.State != "blocked" || out.ErrorCode == "" {
			t.Fatal("unknown binding escaped blocked projection")
		}
	}
	for i := 0; i < 6; i++ {
		v := clone(h.Operation)
		fields := []*string{&v.AuthorityScope.ProjectRef, &v.AuthorityScope.WorkstreamRef, &v.AuthorityScope.WorksetRef, &v.AuthorityScope.CallGraphRef, &v.AuthorityScope.WorkpointRef, &v.AuthorityScope.WorkItemRef}
		*fields[i] += "-widened"
		expect(t, ValidateOperationBindingAgainst(v, a, scope, fixtureNow), ErrAuthorityScopeMismatch)
	}
	for _, now := range []time.Time{{}, h.Operation.ObservedAt.Add(-time.Nanosecond), h.Operation.ExpiresAt, h.Operation.ExpiresAt.Add(time.Nanosecond)} {
		expect(t, ValidateOperationBindingAgainst(h.Operation, a, scope, now), ErrCapabilityBindingStale)
	}
	for _, change := range []func(*OperationApproval){func(a *OperationApproval) { a.BindingSHA256 = "" }, func(a *OperationApproval) { a.EvidenceRefs = nil }, func(a *OperationApproval) { a.EvidenceRefs = []string{"fixture:unapproved"} }} {
		v := clone(a)
		change(&v)
		expect(t, ValidateOperationBindingAgainst(h.Operation, v, scope, fixtureNow), ErrOperationUnapproved)
	}
	a.ExpiresAt = fixtureNow
	expect(t, ValidateOperationBindingAgainst(h.Operation, a, scope, fixtureNow), ErrCapabilityBindingStale)
	if out := AssessOperation(h.Operation, OperationApproval{}, scope, fixtureNow); out.State != "blocked" {
		t.Fatal("missing approval did not serialize blocked")
	}
	h.ConfirmationEvidenceRef = ""
	expect(t, ValidateHandoff(h), ErrOperationUnapproved)
}

func TestExpiryIsAdmissionNotLateReceiptObservation(t *testing.T) {
	h := goldenHandoff(t)
	r := goldenReconciliation(t)
	expect(t, ValidateHandoffAgainst(h, h.Artifact, h.ExpiresAt), ErrEvidenceStale)
	expect(t, ValidateHandoffAgainst(h, h.Artifact, h.ObservedAt.Add(-time.Nanosecond)), ErrEvidenceStale)
	r.ObservedAt = h.ExpiresAt.Add(time.Hour)
	if err := ValidateReconciliationAgainst(r, h, r.ObservedAt); err != nil {
		t.Fatal("late immutable receipt rejected", err)
	}
	r.ObservedAt = fixtureNow.Add(time.Second)
	expect(t, ValidateReconciliationAgainst(r, h, fixtureNow), ErrHandoffContractInvalid)
	h = goldenHandoff(t)
	h.ExpiresAt = h.Operation.ExpiresAt.Add(time.Second)
	expect(t, ValidateHandoff(h), ErrCapabilityBindingStale)
}

func TestReconciliationCannotBorrowOrWidenBindings(t *testing.T) {
	h := goldenHandoff(t)
	base := goldenReconciliation(t)
	for _, change := range []func(*Reconciliation){func(r *Reconciliation) { r.HandoffRef += "-other" }, func(r *Reconciliation) { r.HandoffSHA256 = strings.Repeat("a", 64) }, func(r *Reconciliation) { r.OperationBindingSHA256 = strings.Repeat("b", 64) }, func(r *Reconciliation) { r.IdempotencyKey += "-other" }, func(r *Reconciliation) { r.RequestSHA256 = strings.Repeat("c", 64) }} {
		r := clone(base)
		change(&r)
		expect(t, ValidateReconciliationAgainst(r, h, fixtureNow), ErrArtifactBindingMismatch)
	}
	r := clone(base)
	r.Scope.WorkItemRef += "-other"
	expect(t, ValidateReconciliationAgainst(r, h, fixtureNow), ErrLineageBindingMismatch)
	for _, partial := range []bool{false, true} {
		r = clone(base)
		if partial {
			r.PartialEffects = r.Effects
			r.Effects = nil
		}
		group := r.Effects
		if partial {
			group = r.PartialEffects
		}
		group[0].TargetRef = "fixture:foreign-item"
		expect(t, ValidateReconciliationAgainst(r, h, fixtureNow), ErrAuthorityScopeMismatch)
	}
	r = clone(base)
	r.ReceiptRefs = nil
	expect(t, ValidateReconciliation(r), ErrReceiptRequired)
	r = clone(base)
	r.AuthorityEvidenceRefs = nil
	expect(t, ValidateReconciliation(r), ErrFocusaReconciliationRequired)
	r = clone(base)
	r.Effects[0].EvidenceRef = "fixture:dangling"
	expect(t, ValidateReconciliation(r), ErrHandoffContractInvalid)
	r = clone(base)
	r.PartialEffects = clone(r.Effects)
	expect(t, ValidateReconciliation(r), ErrHandoffContractInvalid)
}

func TestAllConsumerRolesBindExactTargets(t *testing.T) {
	base := goldenHandoff(t)
	s := base.Artifact.Scope
	for _, tc := range []struct {
		role   ConsumerRole
		target string
	}{{ConsumerWorkpoint, s.WorkpointRef}, {ConsumerAcceptance, base.AcceptanceAtomRefs[0]}, {ConsumerContext, s.ProjectRef}, {ConsumerWorkset, s.CallGraphRef}, {ConsumerSilentSession, base.Artifact.ArtifactRef}, {ConsumerDesktop, base.Artifact.ArtifactRef}, {ConsumerCockpit, base.Artifact.ArtifactRef}, {ConsumerProvider, s.WorkItemRef}, {ConsumerFinalOutput, base.Artifact.ArtifactRef}} {
		t.Run(string(tc.role), func(t *testing.T) {
			h := clone(base)
			h.Operation.Consumer = tc.role
			r := rebind(t, goldenReconciliation(t), h)
			r.Effects[0].Kind = tc.role
			r.Effects[0].TargetRef = tc.target
			if err := ValidateReconciliationAgainst(r, h, fixtureNow); err != nil {
				t.Fatal(err)
			}
			r.Effects[0].TargetRef = "fixture:foreign-target"
			expect(t, ValidateReconciliationAgainst(r, h, fixtureNow), ErrAuthorityScopeMismatch)
		})
	}
}

func TestTimeoutLostResponseAndPartialEffectsReconcileBeforeRetry(t *testing.T) {
	for _, transport := range []string{"timeout", "lost_response"} {
		r := goldenReconciliation(t)
		r.State = StateOutcomeUnknown
		r.TransportOutcome = transport
		r.ResponseSHA256 = ""
		r.NextAction = "await_authority_reconciliation"
		r.PartialEffects = r.Effects
		r.Effects = nil
		r.ReceiptRefs = nil
		r.AuthorityEvidenceRefs = nil
		if err := ValidateReconciliation(r); err != nil {
			t.Fatal(err)
		}
		expect(t, ValidateRetryCandidateAgainst(r, goldenHandoff(t), fixtureNow), ErrFocusaOutcomeUnknown)
		r.NextAction = "retry_authorized"
		expect(t, ValidateReconciliation(r), ErrFocusaOutcomeUnknown)
	}
	r := goldenReconciliation(t)
	r.PartialEffects = r.Effects
	r.Effects = nil
	r.AuthorityEvidenceRefs = nil
	expect(t, ValidateRetryCandidateAgainst(r, goldenHandoff(t), fixtureNow), ErrFocusaReconciliationRequired)
	r = goldenReconciliation(t)
	if err := ValidateRetryCandidateAgainst(r, goldenHandoff(t), fixtureNow); err != nil {
		t.Fatal(err)
	}
	for _, state := range []State{StateProposed, StateLinked, StateVerificationPending, StateVerifiedCandidate, StateRejected, StateStale, StateInvalidated, StateReopenRequested, StateReconciled, StateBlocked, StateOutcomeUnknown} {
		r = goldenReconciliation(t)
		r.State = state
		r.Effects = nil
		r.NextAction = "no_retry"
		if state == StateOutcomeUnknown {
			r.NextAction = "await_authority_reconciliation"
		}
		if err := ValidateReconciliation(r); err != nil {
			t.Fatalf("valid state %s: %v", state, err)
		}
		want := ErrFocusaReconciliationRequired
		if state == StateStale {
			want = ErrEvidenceStale
		}
		if state == StateInvalidated {
			want = ErrEvidenceInvalidated
		}
		if state == StateOutcomeUnknown {
			want = ErrFocusaOutcomeUnknown
		}
		expect(t, ValidateRetryCandidateAgainst(r, goldenHandoff(t), fixtureNow), want)
	}
}

func TestInvalidationCandidatesNeverReopenOrCompleteWork(t *testing.T) {
	h := goldenHandoff(t)
	for _, reason := range []InvalidationReason{ReasonRetraction, ReasonCorruption, ReasonReinspection, ReasonRevokedTrust, ReasonCompromisedTrust, ReasonExpiry, ReasonLineage, ReasonAcceptance, ReasonContradiction} {
		v := invalidation(t, h)
		v.Reason = reason
		v.ObservedAt = h.ExpiresAt.Add(time.Second)
		if err := ValidateInvalidationAgainst(v, h, v.ObservedAt); err != nil {
			t.Fatal(reason, err)
		}
		body, _ := json.Marshal(v)
		if _, err := DecodeInvalidation(body); err != nil {
			t.Fatal(err)
		}
	}
	v := invalidation(t, h)
	v.RequestedAction = "reopen"
	expect(t, ValidateInvalidation(v), ErrCompletionAuthorityViolation)
	v = invalidation(t, h)
	v.Reason = ReasonExpiry
	expect(t, ValidateInvalidationAgainst(v, h, fixtureNow), ErrHandoffContractInvalid)
	v = invalidation(t, h)
	v.Scope.WorkpointRef += "-foreign"
	v.Operation.AuthorityScope = v.Scope
	expect(t, ValidateInvalidationAgainst(v, h, fixtureNow), ErrLineageBindingMismatch)
	for _, change := range []func(*Invalidation){func(v *Invalidation) { v.HandoffRef += "-foreign" }, func(v *Invalidation) { v.HandoffSHA256 = strings.Repeat("a", 64) }, func(v *Invalidation) { v.ArtifactRef += "-foreign" }, func(v *Invalidation) { v.ManifestSHA256 = strings.Repeat("a", 64) }} {
		v = invalidation(t, h)
		change(&v)
		expect(t, ValidateInvalidationAgainst(v, h, fixtureNow), ErrArtifactBindingMismatch)
	}
}

func TestStrictAuthorityFieldsDuplicateKeysUnknownAndNestedBounds(t *testing.T) {
	h := goldenHandoff(t)
	good, _ := json.Marshal(h)
	for _, field := range []string{"completed", "settled", "provider_closed", "task_status", "completion_decision", "settlement", "reopened", "review_approval", "providerClosed"} {
		body := append([]byte(`{"`+field+`":true,`), good[1:]...)
		_, err := DecodeHandoff(body)
		expect(t, err, ErrCompletionAuthorityViolation)
	}
	for _, body := range [][]byte{nil, []byte("null"), []byte(`{"schema":"a","schema":"b"}`), []byte(`{"Schema":"a"}`), []byte(`{"ſchema":"a"}`), append(good, []byte(`{}`)...), bytes.Replace(good, []byte(`"schema":`), []byte(`"unknown":`), 1), []byte(strings.Repeat("[", 34) + strings.Repeat("]", 34)), bytes.Repeat([]byte(" "), MaxContractBytes+1)} {
		if _, err := DecodeHandoff(body); err == nil {
			t.Fatal("malformed/oversized/ambiguous input accepted")
		}
	}
	for _, decode := range []func([]byte) error{func(b []byte) error { _, e := DecodeOperationBinding(b); return e }, func(b []byte) error { _, e := DecodeReconciliation(b); return e }, func(b []byte) error { _, e := DecodeInvalidation(b); return e }} {
		expect(t, decode([]byte(`{"completed":true}`)), ErrCompletionAuthorityViolation)
	}
	for _, unsafe := range []string{"/root/private", "https://user:fixture@private.invalid/", "file:../private", "fixture:password=synthetic-value", strings.Repeat("x", MaxRefBytes+1), "fixture:\ncontrol"} {
		h = goldenHandoff(t)
		h.HandoffID = unsafe
		err := ValidateHandoff(h)
		expect(t, err, ErrHandoffContractInvalid)
		if strings.Contains(err.Error(), unsafe) {
			t.Fatal("error echoed untrusted payload")
		}
	}
	h = goldenHandoff(t)
	h.AcceptanceAtomRefs = append(h.AcceptanceAtomRefs, h.AcceptanceAtomRefs[0])
	expect(t, ValidateHandoff(h), ErrHandoffContractInvalid)
	h = goldenHandoff(t)
	h.AcceptanceAtomRefs = make([]string, MaxRefs+1)
	expect(t, ValidateHandoff(h), ErrHandoffContractInvalid)
	r := goldenReconciliation(t)
	r.State = "completed"
	expect(t, ValidateReconciliation(r), ErrHandoffContractInvalid)
	r = goldenReconciliation(t)
	r.Effects = make([]Effect, MaxEffects+1)
	expect(t, ValidateReconciliation(r), ErrHandoffContractInvalid)
	r = goldenReconciliation(t)
	r.ObservedVersion = 1 << 53
	expect(t, ValidateReconciliation(r), ErrHandoffContractInvalid)
}

func TestProductionImportsCannotExecuteAuthorityOrIO(t *testing.T) {
	// A deterministic source-boundary check, not a claimed independent human review.
	allowed := map[string]bool{"bytes": true, "crypto/sha256": true, "encoding/hex": true, "encoding/json": true, "errors": true, "io": true, "strings": true, "time": true, "github.com/WPUIAI/uiai-engine/internal/secrettext": true, "github.com/WPUIAI/uiai-engine/internal/evidencejudge": true}
	for _, path := range []string{"types.go", "validate.go"} {
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatal(err)
		}
		for _, imp := range f.Imports {
			if !allowed[strings.Trim(imp.Path.Value, `"`)] {
				t.Fatal("unexpected production capability import")
			}
		}
	}
}

func FuzzStrictContractDecodersNeverPanic(f *testing.F) {
	f.Add([]byte(`{"completed":true}`))
	f.Add([]byte(`{"schema":"x","schema":"y"}`))
	f.Add([]byte(`[]`))
	f.Fuzz(func(t *testing.T, body []byte) {
		for _, err := range []error{func() error { _, e := DecodeHandoff(body); return e }(), func() error { _, e := DecodeReconciliation(body); return e }(), func() error { _, e := DecodeInvalidation(body); return e }(), func() error { _, e := DecodeOperationBinding(body); return e }()} {
			if err != nil && len(fmt.Sprint(err)) > 128 {
				t.Fatal("unbounded error")
			}
		}
	})
}
