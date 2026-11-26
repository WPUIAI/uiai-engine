package routes

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/epwadelivery"
	"github.com/WPUIAI/uiai-engine/internal/focusapacket"
	"github.com/WPUIAI/uiai-engine/internal/secrettext"
	"github.com/go-chi/chi/v5"
)

// MountProofPublishRoutes registers the #106 durable Browser Proof surface:
//
//	POST /api/proof/publish — compose existing proof responses (viewport
//	screenshots, assertions, diagnostics summary, interaction trace) into one
//	immutable, human-viewable report artifact that outlives the session and
//	the browser.
//
// The report publishes through the generic EPWA contract: read-only by
// default, durable on disk, redacted before assembly, and linked to caller
// screenshot artifact refs as children.
func MountProofPublishRoutes(r chi.Router, cfg *config.Config) {
	r.Post("/publish", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			ArtifactRef  string       `json:"artifact_ref"`
			Title        string       `json:"title"`
			TargetURL    string       `json:"target_url"`
			EngineLabel  string       `json:"engine_version"`
			Viewports    []proofView  `json:"viewports"`
			Assertions   []proofCheck `json:"assertions"`
			Diagnostics  *proofDiag   `json:"diagnostics"`
			Interactions []string     `json:"interaction_notes"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid json", "code": "proof_request_invalid"})
			return
		}
		title := secrettext.Text(strings.TrimSpace(body.Title))
		if title == "" || title == "[secret]" {
			writeJSON(w, 400, map[string]string{"error": "title required", "code": "proof_title_required"})
			return
		}
		if len(body.Viewports) == 0 || len(body.Viewports) > 12 {
			writeJSON(w, 400, map[string]string{"error": "viewports required (1-12)", "code": "proof_viewports_bounded"})
			return
		}
		if len(body.Assertions) > 64 || len(body.Interactions) > 16 {
			writeJSON(w, 400, map[string]string{"error": "bounded input exceeded (assertions <= 64, interaction notes <= 16)", "code": "proof_input_bounded"})
			return
		}

		children := make([]string, 0, len(body.Viewports))
		viewports := make([]proofView, 0, len(body.Viewports))
		for _, view := range body.Viewports {
			view.Label = secrettext.Text(strings.TrimSpace(view.Label))
			view.ScreenshotURL = strings.TrimSpace(view.ScreenshotURL)
			if view.ScreenshotRef != "" {
				children = append(children, strings.TrimSpace(view.ScreenshotRef))
			}
			if view.ScreenshotURL != "" && !isHTTPURL(view.ScreenshotURL) {
				writeJSON(w, 400, map[string]string{"error": "screenshot_url must be http(s)", "code": "proof_screenshot_url_invalid"})
				return
			}
			viewports = append(viewports, view)
		}
		assertions := make([]proofCheck, 0, len(body.Assertions))
		passed, failed := 0, 0
		for _, check := range body.Assertions {
			check.Name = truncateProofText(secrettext.Text(strings.TrimSpace(check.Name)))
			check.Expected = truncateProofText(secrettext.Text(strings.TrimSpace(check.Expected)))
			check.Actual = truncateProofText(secrettext.Text(strings.TrimSpace(check.Actual)))
			if check.Passed {
				passed++
			} else {
				failed++
			}
			assertions = append(assertions, check)
		}
		notes := make([]string, 0, len(body.Interactions))
		for _, note := range body.Interactions {
			notes = append(notes, truncateProofText(secrettext.Text(strings.TrimSpace(note))))
		}

		payload, err := renderProofReport(reportData{
			Title:        title,
			TargetURL:    focusapacket.SanitizeURL(strings.TrimSpace(body.TargetURL)),
			Engine:       truncateProofText(secrettext.Text(strings.TrimSpace(body.EngineLabel))),
			Viewports:    viewports,
			Assertions:   assertions,
			Diagnostics:  body.Diagnostics,
			Interactions: notes,
			Passed:       passed,
			Failed:       failed,
			GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		})
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "report assembly failed", "code": "proof_assembly_failed"})
			return
		}

		delivery, err := PublishGenericArtifactEPWA(req, cfg, strings.TrimSpace(body.ArtifactRef), title, "browser_proof", "text/html", "html", payload)
		if err != nil {
			writeEPWAPublishError(w, http.StatusServiceUnavailable, "epwa_publication_failed", err, "", "", "proof:browser-proof-epwa-publication")
			return
		}
		status := http.StatusAccepted
		response := map[string]any{
			"schema":           "uiai.browser_proof_result.v1",
			"artifact_ref":     delivery.Artifact.ArtifactRef,
			"delivery_state":   delivery.State,
			"epwa_delivery":    delivery,
			"redaction":        "secrettext_applied_to_all_caller_text",
			"generated_notice": "this report is generated, non-live evidence; screenshots are caller-published artifacts linked below",
			"viewport_count":   len(viewports),
			"assertions":       map[string]any{"passed": passed, "failed": failed},
		}
		if delivery.State == epwadelivery.StateReady {
			status = http.StatusOK
			response["artifact_url"] = delivery.EPWA.RecordURL
			response["portable_url"] = delivery.EPWA.PortableURL
			response["created_at"] = time.Now().UTC().Format(time.RFC3339)
		}
		writeJSON(w, status, response)
	})
}

type proofView struct {
	Label         string `json:"label"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	ScreenshotURL string `json:"screenshot_url"`
	ScreenshotRef string `json:"screenshot_ref"`
}

type proofCheck struct {
	Name     string `json:"name"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Passed   bool   `json:"passed"`
}

type proofDiag struct {
	ConsoleErrors  int `json:"console_errors"`
	Exceptions     int `json:"exceptions"`
	FailedRequests int `json:"failed_requests"`
	HTTP4xx        int `json:"http_4xx"`
	HTTP5xx        int `json:"http_5xx"`
}

type reportData struct {
	Title        string
	TargetURL    string
	Engine       string
	Viewports    []proofView
	Assertions   []proofCheck
	Diagnostics  *proofDiag
	Interactions []string
	Passed       int
	Failed       int
	GeneratedAt  string
}

const proofReportTemplate = `<!doctype html>
<html lang="en"><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}} — Browser Proof</title>
<style>
body{font-family:system-ui,sans-serif;margin:0;background:#0f1216;color:#e8eaf0;line-height:1.5}
main{max-width:960px;margin:0 auto;padding:24px 16px 64px}
h1{font-size:1.4rem;margin:0 0 4px}.meta{color:#98a2b3;font-size:.85rem;margin-bottom:20px}
.badge{display:inline-block;padding:2px 10px;border-radius:12px;font-size:.75rem;margin-right:6px}
.pass{background:#10331a;color:#5fe08a}.fail{background:#3a1414;color:#ff8a8a}.warn{background:#332b10;color:#ffd76a}
section{background:#171b22;border:1px solid #262c38;border-radius:10px;padding:16px;margin-bottom:16px}
h2{font-size:1rem;margin:0 0 12px;color:#c6cfdd}
.vp{display:flex;flex-wrap:wrap;gap:8px}.vp a{display:block;background:#1f2530;border:1px solid #313a4a;border-radius:8px;padding:10px 14px;color:#8ab8ff;text-decoration:none;font-size:.85rem}
table{width:100%;border-collapse:collapse;font-size:.85rem}
td,th{padding:6px 8px;border-bottom:1px solid #262c38;text-align:left}
.diag{display:flex;flex-wrap:wrap;gap:10px;font-size:.85rem}.diag span{background:#1f2530;padding:6px 10px;border-radius:6px}
li{margin-bottom:6px;font-size:.88rem}
.notice{border-left:3px solid #ffd76a;padding:8px 12px;background:#1c1a10;font-size:.8rem;color:#d9cba0}
</style></head><body><main>
<h1>{{.Title}}</h1>
<div class="meta">{{.TargetURL}} · generated {{.GeneratedAt}} · engine {{if .Engine}}{{.Engine}}{{else}}n/a{{end}}</div>
<div><span class="badge pass">{{.Passed}} passed</span><span class="badge fail">{{.Failed}} failed</span></div>
<section><h2>Viewport matrix</h2><div class="vp">
{{range .Viewports}}<a href="{{.ScreenshotURL}}" target="_blank" rel="noopener">{{if .Label}}{{.Label}}{{else}}{{.Width}}×{{.Height}}{{end}} — {{.Width}}×{{.Height}}</a>
{{end}}</div></section>
{{if .Assertions}}<section><h2>Assertions</h2><table><tr><th>Check</th><th>Expected</th><th>Measured</th><th>Result</th></tr>
{{range .Assertions}}<tr><td>{{.Name}}</td><td>{{.Expected}}</td><td>{{.Actual}}</td><td>{{if .Passed}}<span class="badge pass">pass</span>{{else}}<span class="badge fail">fail</span>{{end}}</td></tr>
{{end}}</table></section>{{end}}
{{if .Diagnostics}}<section><h2>Diagnostics summary</h2><div class="diag">
<span>console errors: {{.Diagnostics.ConsoleErrors}}</span><span>exceptions: {{.Diagnostics.Exceptions}}</span>
<span>failed requests: {{.Diagnostics.FailedRequests}}</span><span>HTTP 4xx: {{.Diagnostics.HTTP4xx}}</span><span>HTTP 5xx: {{.Diagnostics.HTTP5xx}}</span>
</div></section>{{end}}
{{if .Interactions}}<section><h2>Interaction trace</h2><ul>{{range .Interactions}}<li>{{.}}</li>{{end}}</ul></section>{{end}}
<p class="notice">Generated, non-live evidence. Screenshots link to caller-published artifacts; the page state is not live and capture-time failures are reported above, not observable here.</p>
</main></body></html>`

var proofReportTmpl = template.Must(template.New("proof").Parse(proofReportTemplate))

func renderProofReport(data reportData) ([]byte, error) {
	var out strings.Builder
	if err := proofReportTmpl.Execute(&out, data); err != nil {
		return nil, fmt.Errorf("template: %w", err)
	}
	return []byte(out.String()), nil
}

func isHTTPURL(raw string) bool {
	lower := strings.ToLower(raw)
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

func truncateProofText(value string) string {
	if len(value) > 400 {
		return value[:400] + "…"
	}
	return value
}
