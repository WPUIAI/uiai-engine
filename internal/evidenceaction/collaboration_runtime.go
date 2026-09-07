package evidenceaction

import (
	"errors"
	"sync"
)

var (
	ErrReviewThreadNotFound  = errors.New("evidence review thread not found")
	ErrReviewAuthorityDenied = errors.New("evidence review authority denied")
)

type ReviewAuthority struct {
	DecisionDelegations map[string]string
}

type CollaborationRuntime struct {
	mu      sync.Mutex
	threads map[string]ReviewThread
}

func NewCollaborationRuntime() *CollaborationRuntime {
	return &CollaborationRuntime{threads: map[string]ReviewThread{}}
}

func (runtime *CollaborationRuntime) OpenThread(thread ReviewThread) (ReviewThread, error) {
	if err := ValidateReviewThread(thread); err != nil {
		return ReviewThread{}, err
	}
	if err := ValidateReviewAuthority(thread); err != nil {
		return ReviewThread{}, err
	}
	thread = cloneReviewThread(thread)
	digest, err := DigestReviewThread(thread)
	if err != nil {
		return ReviewThread{}, err
	}
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	if existing, found := runtime.threads[thread.ThreadID]; found {
		existingDigest, digestErr := DigestReviewThread(existing)
		if digestErr != nil || existingDigest != digest {
			return ReviewThread{}, ErrReplayDetected
		}
		return cloneReviewThread(existing), nil
	}
	runtime.threads[thread.ThreadID] = thread
	return cloneReviewThread(thread), nil
}

func (runtime *CollaborationRuntime) Append(threadID string, expectedRevision uint64, entry ReviewEntry, authority ReviewAuthority) (ReviewThread, error) {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	thread, found := runtime.threads[threadID]
	if !found {
		return ReviewThread{}, ErrReviewThreadNotFound
	}
	if thread.Revision != expectedRevision {
		return ReviewThread{}, ErrStateVersionMismatch
	}
	if entry.ItemRef != thread.Scope.WorkItemRef || entry.ArtifactRef != thread.ArtifactRef {
		return ReviewThread{}, ErrActionScopeMismatch
	}
	if entry.Imported {
		if entry.SourceTrust != SourceImportedUntrusted || entry.Kind == EntryDecision || entry.Kind == EntrySupersession {
			return ReviewThread{}, ErrImportedActionUntrusted
		}
	} else if entry.SourceTrust != SourceVerified {
		return ReviewThread{}, ErrImportedActionUntrusted
	}
	if entry.Kind == EntryDecision {
		delegation, allowed := authority.DecisionDelegations[entry.ActorRef]
		if !allowed || delegation == "" || delegation != entry.DelegationRef {
			return ReviewThread{}, ErrReviewAuthorityDenied
		}
	}
	entry.Revision = thread.Revision + 1
	entry.AtomRefs = append([]string(nil), entry.AtomRefs...)
	entry.CitationRefs = append([]string(nil), entry.CitationRefs...)
	candidate := cloneReviewThread(thread)
	candidate.Revision = entry.Revision
	candidate.Entries = append(candidate.Entries, entry)
	if entry.Imported {
		candidate.SourceTrust = SourceImportedUntrusted
		candidate.HumanReviewMandated = true
		candidate.AutonomousEligibility = AutonomousIneligible
	}
	if err := ValidateReviewThread(candidate); err != nil {
		return ReviewThread{}, err
	}
	if err := ValidateReviewAuthority(candidate); err != nil {
		return ReviewThread{}, err
	}
	runtime.threads[threadID] = candidate
	return cloneReviewThread(candidate), nil
}

func (runtime *CollaborationRuntime) Get(threadID string) (ReviewThread, error) {
	runtime.mu.Lock()
	defer runtime.mu.Unlock()
	thread, found := runtime.threads[threadID]
	if !found {
		return ReviewThread{}, ErrReviewThreadNotFound
	}
	return cloneReviewThread(thread), nil
}

func cloneReviewThread(thread ReviewThread) ReviewThread {
	thread.AtomRefs = append([]string(nil), thread.AtomRefs...)
	thread.Entries = append([]ReviewEntry(nil), thread.Entries...)
	for index := range thread.Entries {
		thread.Entries[index].AtomRefs = append([]string(nil), thread.Entries[index].AtomRefs...)
		thread.Entries[index].CitationRefs = append([]string(nil), thread.Entries[index].CitationRefs...)
	}
	return thread
}
