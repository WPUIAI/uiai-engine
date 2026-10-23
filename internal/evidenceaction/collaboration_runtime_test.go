package evidenceaction

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestCollaborationRuntimeAppendsItemScopedEntry(t *testing.T) {
	_, _, _, _, _, _, thread, now := validContracts(t)
	runtime := NewCollaborationRuntime()
	opened, err := runtime.OpenThread(thread)
	if err != nil {
		t.Fatal(err)
	}
	entry := runtimeReviewEntry(opened, "entry:3", now.Add(2*time.Minute))
	updated, err := runtime.Append(opened.ThreadID, opened.Revision, entry, ReviewAuthority{})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != 3 || updated.Entries[2].Revision != 3 || thread.Revision != 2 {
		t.Fatalf("append result = %#v, original revision = %d", updated, thread.Revision)
	}
	updated.Entries[0].Message = "mutated by caller"
	stored, err := runtime.Get(thread.ThreadID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Entries[0].Message == "mutated by caller" {
		t.Fatal("caller mutated stored review history")
	}
}

func TestCollaborationRuntimeRejectsStaleAndConcurrentRevisions(t *testing.T) {
	_, _, _, _, _, _, thread, now := validContracts(t)
	runtime := NewCollaborationRuntime()
	if _, err := runtime.OpenThread(thread); err != nil {
		t.Fatal(err)
	}
	if _, err := runtime.Append(thread.ThreadID, 1, runtimeReviewEntry(thread, "entry:stale", now), ReviewAuthority{}); !errors.Is(err, ErrStateVersionMismatch) {
		t.Fatalf("stale append error = %v", err)
	}
	const workers = 16
	results := make(chan error, workers)
	var wait sync.WaitGroup
	for index := 0; index < workers; index++ {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			entry := runtimeReviewEntry(thread, fmt.Sprintf("entry:concurrent-%d", index), now.Add(time.Duration(index+2)*time.Minute))
			_, err := runtime.Append(thread.ThreadID, thread.Revision, entry, ReviewAuthority{})
			results <- err
		}(index)
	}
	wait.Wait()
	close(results)
	succeeded, stale := 0, 0
	for err := range results {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, ErrStateVersionMismatch):
			stale++
		default:
			t.Fatalf("unexpected append error: %v", err)
		}
	}
	if succeeded != 1 || stale != workers-1 {
		t.Fatalf("append outcomes: succeeded=%d stale=%d", succeeded, stale)
	}
}

func TestCollaborationRuntimeGatesDecisionAuthorityAndImportedEntries(t *testing.T) {
	_, _, _, _, _, _, thread, now := validContracts(t)
	runtime := NewCollaborationRuntime()
	if _, err := runtime.OpenThread(thread); err != nil {
		t.Fatal(err)
	}
	decision := runtimeReviewEntry(thread, "entry:decision", now.Add(2*time.Minute))
	decision.Kind = EntryDecision
	decision.Decision = DecisionChangesRequested
	decision.SupersedesRef = "entry:2"
	if _, err := runtime.Append(thread.ThreadID, thread.Revision, decision, ReviewAuthority{}); !errors.Is(err, ErrReviewAuthorityDenied) {
		t.Fatalf("unauthorized decision error = %v", err)
	}
	authority := ReviewAuthority{DecisionDelegations: map[string]string{decision.ActorRef: decision.DelegationRef}}
	if _, err := runtime.Append(thread.ThreadID, thread.Revision, decision, authority); err != nil {
		t.Fatal(err)
	}

	_, _, _, _, _, _, importedThread, importedNow := validContracts(t)
	importedThread.ThreadID = "thread:imported-test"
	importedRuntime := NewCollaborationRuntime()
	if _, err := importedRuntime.OpenThread(importedThread); err != nil {
		t.Fatal(err)
	}
	imported := runtimeReviewEntry(importedThread, "entry:imported", importedNow.Add(2*time.Minute))
	imported.Imported = true
	imported.SourceTrust = SourceImportedUntrusted
	if _, err := importedRuntime.Append(importedThread.ThreadID, importedThread.Revision, imported, ReviewAuthority{}); !errors.Is(err, ErrImportedActionUntrusted) {
		t.Fatalf("imported append error = %v", err)
	}
}

func runtimeReviewEntry(thread ReviewThread, id string, occurredAt time.Time) ReviewEntry {
	return ReviewEntry{
		EntryID: id, Kind: EntryComment, Decision: DecisionNone, Message: "Bound review comment.",
		ItemRef: thread.Scope.WorkItemRef, AtomRefs: []string{thread.AtomRefs[0]}, ArtifactRef: thread.ArtifactRef,
		CitationRefs: []string{"citation:runtime"}, ActorRef: "reviewer:runtime", DelegationRef: "delegation:runtime",
		OccurredAt: occurredAt, ProvenanceRef: "provenance:runtime", SourceTrust: SourceVerified,
	}
}
