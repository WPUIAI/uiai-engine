package routes

import (
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/evidenceshare"
)

// #236: a blocked EPWA delivery must name exactly which scope bindings are
// missing, so callers supplying an incomplete envelope see the precise gaps
// instead of one generic scope-required recovery ref.
func TestBlockedEPWADeliveryNamesMissingScopeBindings(t *testing.T) {
	t.Setenv("UIAI_EVIDENCE_SHARE_DIR", "")
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}

	partial := completeEvidenceScope()
	partial.WorkstreamRef = ""
	partial.WorkItems = nil

	delivery, err := publishScreenshotEPWA(nil, cfg, evidenceshare.Input{
		Screenshot: append([]byte("\x89PNG\r\n\x1a\n"), []byte("scope-diagnostic-pixels")...),
		Format:     "png", Width: 64, Height: 48, SourceURL: "https://example.com/scope",
		CapturedAt: time.Date(2026, 9, 14, 0, 6, 0, 0, time.UTC), Scope: partial,
	})
	if err != nil {
		t.Fatal(err)
	}
	if delivery.State != "blocked" {
		t.Fatalf("state=%q, want blocked", delivery.State)
	}
	if delivery.RecoveryRef != "reconcile:epwa-scope-required" {
		t.Fatalf("recovery_ref=%q", delivery.RecoveryRef)
	}
	if !strings.Contains(strings.Join(delivery.Scope.Missing, ","), "workstream_ref") {
		t.Fatalf("missing list must name workstream_ref: %v", delivery.Scope.Missing)
	}
	if !strings.Contains(strings.Join(delivery.Scope.Missing, ","), "work_items") {
		t.Fatalf("missing list must name work_items: %v", delivery.Scope.Missing)
	}
	for _, field := range []string{"project_ref", "workset_ref", "callgraph_ref", "workpoint_ref", "work_item_ref", "continuity_ref"} {
		if strings.Contains(strings.Join(delivery.Scope.Missing, ","), field) {
			t.Fatalf("supplied field %q must not be reported missing: %v", field, delivery.Scope.Missing)
		}
	}

	// The granted state stays unchanged: a complete scope publishes HTTPS URLs
	// and carries no missing list.
	t.Setenv("UIAI_EPWA_PUBLIC_BASE_URL", "https://evidence.example.test")
	granted, err := publishScreenshotEPWA(nil, cfg, evidenceshare.Input{
		Screenshot: append([]byte("\x89PNG\r\n\x1a\n"), []byte("scope-diagnostic-granted")...),
		Format:     "png", Width: 64, Height: 48, SourceURL: "https://example.com/scope",
		CapturedAt: time.Date(2026, 9, 14, 0, 8, 0, 0, time.UTC), Scope: completeEvidenceScope(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if granted.State != "ready" {
		t.Fatalf("granted state=%q, want ready", granted.State)
	}
	if !strings.HasPrefix(granted.EPWA.RecordURL, "https://evidence.example.test/") {
		t.Fatalf("granted record_url=%q", granted.EPWA.RecordURL)
	}
	if len(granted.Scope.Missing) != 0 {
		t.Fatalf("granted delivery must carry no missing list: %v", granted.Scope.Missing)
	}
}
