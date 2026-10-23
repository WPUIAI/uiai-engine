package evidenceshare

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
)

// LifecyclePolicy is the settings-derived retention contract for the immutable
// artifact store (completion graph CG24). It never silently deletes: expiry,
// age, and quota pressure archive entries through tombstones, GC retires
// commits to the retired layout, and legal holds block everything.
type LifecyclePolicy struct {
	RetentionDays       int   `json:"retention_days"`
	MaxPackets          int   `json:"max_packets"`
	MaxBytes            int64 `json:"max_bytes"`
	Pinning             bool  `json:"pinning"`
	ArchiveBeforeExpire bool  `json:"archive_before_expire"`
	LegalHold           bool  `json:"legal_hold"`
}

// RetentionSweepReceipt is the typed audit trail for one governed sweep.
type RetentionSweepReceipt struct {
	Schema            string                     `json:"schema"`
	Scope             SettingsScope              `json:"scope"`
	Policy            LifecyclePolicy            `json:"policy"`
	AuthorityRef      string                     `json:"authority_ref"`
	Now               string                     `json:"now"`
	Reviewed          int                        `json:"reviewed"`
	HeldLegal         int                        `json:"held_legal"`
	HeldPinned        int                        `json:"held_pinned"`
	ExpiredTombstoned int                        `json:"expired_tombstoned"`
	AgedTombstoned    int                        `json:"aged_tombstoned"`
	QuotaTombstoned   int                        `json:"quota_tombstoned"`
	SkippedUnarchived int                        `json:"skipped_unarchived"`
	Remaining         int                        `json:"remaining"`
	GC                *evidenceartifact.GCResult `json:"gc,omitempty"`
	TombstoneErrors   []string                   `json:"tombstone_errors,omitempty"`
}

func settingBool(value any) (bool, bool) {
	v := reflect.ValueOf(value)
	if v.IsValid() && v.Kind() == reflect.Bool {
		return v.Bool(), true
	}
	return false, false
}

// MapLifecycleSettings projects the effective settings lifecycle domain onto
// the store retention contract. Defaults survive only through their types;
// missing or mistyped fields fail closed.
func MapLifecycleSettings(values map[string]any) (LifecyclePolicy, error) {
	raw, ok := values["lifecycle"].(map[string]any)
	if !ok {
		return LifecyclePolicy{}, fmt.Errorf("%w: lifecycle domain required", ErrSettingsInvalid)
	}
	invalid := func(field string) error {
		return fmt.Errorf("%w: lifecycle.%s required and typed", ErrSettingsInvalid, field)
	}
	var policy LifecyclePolicy
	number := func(field string, target *int) error {
		n, ok := settingNumber(raw[field])
		if !ok || n < 0 || n != float64(int(n)) {
			return invalid(field)
		}
		*target = int(n)
		return nil
	}
	if err := number("retention_days", &policy.RetentionDays); err != nil {
		return LifecyclePolicy{}, err
	}
	if err := number("max_packets", &policy.MaxPackets); err != nil {
		return LifecyclePolicy{}, err
	}
	if n, ok := settingNumber(raw["max_bytes"]); ok && n >= 0 && n == float64(int(n)) {
		policy.MaxBytes = int64(n)
	} else {
		return LifecyclePolicy{}, invalid("max_bytes")
	}
	for _, field := range []struct {
		name   string
		target *bool
	}{
		{"pinning", &policy.Pinning},
		{"archive_before_expire", &policy.ArchiveBeforeExpire},
		{"legal_hold", &policy.LegalHold},
	} {
		v, ok := settingBool(raw[field.name])
		if !ok {
			return LifecyclePolicy{}, invalid(field.name)
		}
		*field.target = v
	}
	return policy, nil
}

func parseStoreTime(value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed.UTC(), true
		}
	}
	return time.Time{}, false
}

func expiresBefore(expiresAt string, now time.Time) bool {
	if expiresAt == "" {
		return false
	}
	expires, ok := parseStoreTime(expiresAt)
	return ok && expires.Before(now)
}

type retentionCandidate struct {
	entry     evidenceartifact.Entry
	reason    string
	committed time.Time
}

func retentionProtection(entry evidenceartifact.Entry, policy LifecyclePolicy) (legalHold bool, pinned bool) {
	if entry.RetentionClass == evidenceartifact.RetentionLegalHold || policy.LegalHold {
		return true, false
	}
	if policy.Pinning && entry.RetentionClass == evidenceartifact.RetentionRelease {
		return false, true
	}
	return false, false
}

// Sweep applies the mapped lifecycle policy to the immutable artifact store.
// Tombstones archive; the store's GC retires tombstoned commits after its own
// grace window, so canonical evidence is never silently deleted.
func Sweep(store *evidenceartifact.Store, policy LifecyclePolicy, scope SettingsScope, authorityRef string, now time.Time) (RetentionSweepReceipt, error) {
	receipt := RetentionSweepReceipt{
		Schema: "uiai.evidence_retention_sweep.v1", Scope: scope, Policy: policy,
		AuthorityRef: authorityRef, Now: now.UTC().Format(time.RFC3339Nano),
	}
	if store == nil {
		return receipt, errors.New("evidence artifact store unavailable")
	}
	if authorityRef == "" {
		return receipt, fmt.Errorf("%w: authority_ref required", ErrSettingsInvalid)
	}
	entries := store.List()
	receipt.Reviewed = len(entries)

	ageCutoff := now.AddDate(0, 0, -policy.RetentionDays)
	var candidates []retentionCandidate
	var unheldCount, unheldBytes int64
	for _, entry := range entries {
		legalHold, pinned := retentionProtection(entry, policy)
		switch {
		case legalHold:
			receipt.HeldLegal++
			continue
		case pinned:
			receipt.HeldPinned++
			continue
		case !policy.ArchiveBeforeExpire:
			receipt.SkippedUnarchived++
			continue
		}
		unheldCount++
		for _, asset := range entry.Assets {
			unheldBytes += asset.ByteSize
		}
		committed, committedOK := parseStoreTime(entry.CommittedAt)
		switch {
		case expiresBefore(entry.ExpiresAt, now):
			candidates = append(candidates, retentionCandidate{entry: entry, reason: "retention:expired"})
		case policy.RetentionDays > 0 && committedOK && committed.Before(ageCutoff):
			candidates = append(candidates, retentionCandidate{entry: entry, reason: "retention:age", committed: committed})
		}
	}

	// Quota pressure archives the oldest unheld entries first; it never deletes.
	// Held entries still count toward usage but are never archived by quota.
	totalLive := unheldCount + int64(receipt.HeldLegal+receipt.HeldPinned)
	quotaOver := func() bool {
		if policy.MaxPackets > 0 && unheldCount > int64(policy.MaxPackets) {
			return true
		}
		if policy.MaxBytes > 0 && unheldBytes > policy.MaxBytes {
			return true
		}
		return false
	}
	if (policy.MaxPackets > 0 && totalLive > int64(policy.MaxPackets)) || (policy.MaxBytes > 0 && unheldBytes > policy.MaxBytes) {
		var unheld []retentionCandidate
		for _, entry := range entries {
			if legalHold, pinned := retentionProtection(entry, policy); legalHold || pinned {
				continue
			}
			committed, _ := parseStoreTime(entry.CommittedAt)
			unheld = append(unheld, retentionCandidate{entry: entry, committed: committed})
		}
		sort.SliceStable(unheld, func(i, j int) bool {
			return unheld[i].committed.Before(unheld[j].committed)
		})
		for _, candidate := range unheld {
			if !quotaOver() {
				break
			}
			candidate.reason = "retention:quota"
			candidates = append(candidates, candidate)
			unheldCount--
			for _, asset := range candidate.entry.Assets {
				unheldBytes -= asset.ByteSize
			}
		}
	}

	seen := make(map[string]bool, len(candidates))
	for _, candidate := range candidates {
		if seen[candidate.entry.ArtifactID] {
			continue
		}
		seen[candidate.entry.ArtifactID] = true
		if _, err := store.Tombstone(candidate.entry.ArtifactID, candidate.entry.Revision, candidate.reason, authorityRef); err != nil {
			if errors.Is(err, evidenceartifact.ErrRetentionBlocked) {
				receipt.HeldLegal++
				continue
			}
			receipt.TombstoneErrors = append(receipt.TombstoneErrors,
				fmt.Sprintf("%s rev%d %s: %v", candidate.entry.ArtifactID, candidate.entry.Revision, candidate.reason, err))
			continue
		}
		switch candidate.reason {
		case "retention:expired":
			receipt.ExpiredTombstoned++
		case "retention:age":
			receipt.AgedTombstoned++
		case "retention:quota":
			receipt.QuotaTombstoned++
		}
	}

	result, err := store.GC()
	switch {
	case err == nil:
		receipt.GC = &result
	case errors.Is(err, evidenceartifact.ErrStoreCorrupt) || errors.Is(err, evidenceartifact.ErrOutcomeUnknown):
		return receipt, fmt.Errorf("evidence retention gc: %w", err)
	default:
		receipt.TombstoneErrors = append(receipt.TombstoneErrors, fmt.Sprintf("gc: %v", err))
	}
	receipt.Remaining = len(store.List())
	return receipt, nil
}
