package epwadelivery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProducerInventoryHasRuntimeBindings(t *testing.T) {
	bindings := map[ProducerID][]struct{ file, token string }{
		ProducerScreenshot:       {{"../routes/epwa_delivery.go", "ProducerScreenshot"}},
		ProducerSessionVisual:    {{"../routes/epwa_delivery.go", "ProducerSessionVisual"}},
		ProducerInteractive:      {{"../routes/vision_interactive.go", "ProducerInteractive"}},
		ProducerMedia:            {{"../routes/media_frame.go", "ProducerMedia"}, {"../routes/media.go", `Kind:        "media"`}},
		ProducerRuntimeArtifact:  {{"../intelligence/routes.go", `"runtime_binary"`}},
		ProducerShareScreenshot:  {{"../routes/share.go", "ProducerShareScreenshot"}},
		ProducerFPV:              {{"../routes/fpv.go", "ProducerFPV"}},
		ProducerArtifactCommit:   {{"../routes/epwa_delivery.go", "ProducerArtifactCommit"}},
		ProducerDerivative:       {{"../evidencederivative/delivery_runtime.go", "ProducerDerivative"}},
		ProducerDiagnostics:      {{"../routes/session.go", `"diagnostics_bundle"`}},
		ProducerSourceSnapshot:   {{"../routes/session.go", `"source_snapshot"`}, {"../routes/markdown.go", `"source_snapshot"`}},
		ProducerDOMSnapshot:      {{"../routes/session.go", `"dom_snapshot"`}},
		ProducerCritique:         {{"../routes/critique.go", `"critique_report"`}},
		ProducerVisualComparison: {{"../routes/screenshot_compare.go", `"visual_comparison_report"`}},
		ProducerResearch:         {{"../routes/search.go", `"research_packet"`}, {"../routes/agent_packet.go", `"research_packet"`}},
		ProducerGeneratedReport:  {{"../routes/pipeline.go", `"generated_report"`}},
	}
	for _, policy := range ProducerInventory() {
		checks := bindings[policy.ID]
		if len(checks) == 0 {
			t.Fatalf("producer %q has no runtime binding", policy.ID)
		}
		for _, check := range checks {
			body, err := os.ReadFile(filepath.Clean(check.file))
			if err != nil {
				t.Fatalf("read %s for producer %q: %v", check.file, policy.ID, err)
			}
			if !strings.Contains(string(body), check.token) {
				t.Fatalf("producer %q missing binding %q in %s", policy.ID, check.token, check.file)
			}
		}
	}
}

func TestConsumerAdaptersRequireReadyEPWA(t *testing.T) {
	checks := []struct {
		consumer string
		file     string
		tokens   []string
	}{
		{consumer: "Cockpit and Desktop", file: "../../apps/cockpit/src/lib/engine-client.ts", tokens: []string{"validateArtifactDelivery", "EngineDeliveryError", "https://"}},
		{consumer: "Cockpit live workspace", file: "../../apps/cockpit/src/lib/ui/LiveWorkspace.svelte", tokens: []string{"artifact_url", "portable_url", "durable EPWA"}},
		{consumer: "Chrome, Veragensia, and Pi MCP bridge", file: "../../mcp/browser-session-mcp.mjs", tokens: []string{"findRawArtifactField(data)", "evidenceScopeHeaders(args.focusa_scope)"}},
		{consumer: "CLI", file: "../../scripts/uiai", tokens: []string{"uiai.epwa_delivery.v1", "artifact delivery is not a ready, identity-bound HTTPS EPWA result"}},
		{consumer: "CLI selected-result workflow", file: "../../scripts/uiai-open-result.sh", tokens: []string{"require_delivery", "UIAI_EVIDENCE_SCOPE_JSON"}},
		{consumer: "FPV visual smoke", file: "../../scripts/smoke-fpv-visual-breakpoints.sh", tokens: []string{"require_epwa", "portable_url"}},
		{consumer: "Focusa packet smoke", file: "../../scripts/smoke-focusa-packet.sh", tokens: []string{"require_delivery", "uiai.epwa_delivery.v1"}},
		{consumer: "browser diagnostics stress", file: "../../scripts/stress-browser-diagnostics.sh", tokens: []string{"require_delivery", "UIAI_EVIDENCE_SCOPE_JSON", "UIAI_EPWA_PUBLIC_BASE_URL"}},
		{consumer: "browser soak", file: "../../scripts/soak-browser-flakiness.sh", tokens: []string{"require_delivery", "UIAI_EPWA_PUBLIC_BASE_URL"}},
		{consumer: "deployment browser smoke", file: "../../scripts/deploy-engine-ovh.sh", tokens: []string{"epwa_delivery_ready", "REMOTE_EVIDENCE_SCOPE_JSON"}},
	}
	for _, check := range checks {
		body, err := os.ReadFile(filepath.Clean(check.file))
		if err != nil {
			t.Fatalf("read %s adapter %s: %v", check.consumer, check.file, err)
		}
		for _, token := range check.tokens {
			if !strings.Contains(string(body), token) {
				t.Errorf("%s adapter %s lacks mandatory EPWA binding %q", check.consumer, check.file, token)
			}
		}
	}
}
