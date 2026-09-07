package routes

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/auth"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/credits"
	"github.com/WPUIAI/uiai-engine/internal/epwadelivery"
	"github.com/WPUIAI/uiai-engine/internal/evidenceshare"
	"github.com/WPUIAI/uiai-engine/internal/media"
	"github.com/WPUIAI/uiai-engine/internal/ratelimit"
	"github.com/WPUIAI/uiai-engine/internal/storage"
	"github.com/go-chi/chi/v5"
)

type mediaDeps struct {
	cfg     *config.Config
	credits *credits.Service
	limiter *ratelimit.Limiter
	usage   *storage.UsageStore
	jobs    *media.JobStore
}

// MountMediaReal registers media production routes.
func MountMediaReal(r chi.Router, cfg *config.Config, creds *credits.Service, lim *ratelimit.Limiter, usage *storage.UsageStore, jobs *media.JobStore) {
	d := &mediaDeps{cfg: cfg, credits: creds, limiter: lim, usage: usage, jobs: jobs}

	r.Post("/produce", d.handleProduce)
	r.Get("/status/{jobID}", d.handleStatus)
	r.Get("/jobs", d.handleList)

	// Device frame rendering (GitHub-vendored frame assets)
	r.Route("/frame", func(r chi.Router) {
		mountFrameRoutes(r, cfg)
	})
}

func (d *mediaDeps) handleProduce(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Type   media.JobType `json:"type"`
		Device string        `json:"device"`
		URLs   []string      `json:"urls"`
		URL    string        `json:"url"`
		Width  int           `json:"width"`
		Height int           `json:"height"`
		Frames int           `json:"frames"`
		Delay  int           `json:"delay"`
		Mode   string        `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}

	// Normalize
	if req.URL != "" && len(req.URLs) == 0 {
		req.URLs = []string{req.URL}
	}
	if len(req.URLs) == 0 {
		writeJSON(w, 400, map[string]string{"error": "url or urls required"})
		return
	}
	if req.Type == "" {
		writeJSON(w, 400, map[string]string{"error": "type required (device_mockup, animated_gif)"})
		return
	}

	// Validate URLs (SSRF protection)
	for _, u := range req.URLs {
		if err := validateMediaURL(u); err != nil {
			writeJSON(w, 400, map[string]string{"error": fmt.Sprintf("invalid URL %q: %v", u, err)})
			return
		}
	}

	// Validate device against allowlist
	allowedDevices := map[string]bool{"macbook-pro": true, "iphone-15": true, "browser-window": true}
	if req.Device != "" && !allowedDevices[req.Device] {
		writeJSON(w, 400, map[string]string{"error": fmt.Sprintf("invalid device %q", req.Device)})
		return
	}

	// Defaults
	if req.Device == "" {
		req.Device = "macbook-pro"
	}
	if req.Width <= 0 {
		req.Width = 1280
	}
	if req.Height <= 0 {
		req.Height = 800
	}
	if req.Frames <= 0 {
		req.Frames = 6
	}
	if req.Delay <= 0 {
		req.Delay = 200
	}
	if req.Mode == "" {
		if len(req.URLs) > 1 {
			req.Mode = "pages"
		} else {
			req.Mode = "scroll"
		}
	}

	// Generate job ID
	idBytes := make([]byte, 8)
	rand.Read(idBytes)
	jobID := hex.EncodeToString(idBytes)

	// Extract license from auth context for credit deduction
	var licenseID int
	if id := auth.FromContext(r.Context()); id != nil {
		licenseID = id.LicenseID
	}

	deliveryBaseURL := ""
	if base, err := canonicalEPWABase(r); err == nil {
		deliveryBaseURL = base.String()
	}
	job := &media.Job{
		ID:              jobID,
		Type:            req.Type,
		Status:          media.StatusPending,
		Device:          req.Device,
		URLs:            req.URLs,
		Width:           req.Width,
		Height:          req.Height,
		Frames:          req.Frames,
		Delay:           req.Delay,
		Mode:            req.Mode,
		LicenseID:       licenseID,
		CreatedAt:       time.Now().UTC(),
		DeliveryScope:   evidenceScopeFromRequest(r),
		DeliveryBaseURL: deliveryBaseURL,
	}

	d.jobs.Create(job)

	// Execute synchronously for now (local server has all tools)
	// TODO: dispatch to GitHub Actions for scale
	go d.executeJob(job)

	writeJSON(w, 202, map[string]any{
		"job_id":  jobID,
		"status":  "pending",
		"type":    req.Type,
		"message": "Media production started",
	})
}

func mediaJobResponse(job *media.Job) map[string]any {
	resp := map[string]any{
		"schema": "uiai.media_job_status.v2", "job_id": job.ID, "artifact_ref": "media:job:" + job.ID,
		"type": job.Type, "status": job.Status, "created_at": job.CreatedAt,
		"delivery_state": "pending_reconcile", "raw_output_posture": "withheld_by_mandatory_epwa_delivery",
	}
	if job.EPWADelivery != nil {
		resp["schema"] = "uiai.artifact_result.v2"
		resp["delivery_state"] = job.EPWADelivery.State
		resp["epwa_delivery"] = job.EPWADelivery
		if job.EPWADelivery.State == epwadelivery.StateReady {
			resp["artifact_url"] = job.EPWADelivery.EPWA.RecordURL
			resp["portable_url"] = job.EPWADelivery.EPWA.PortableURL
		}
	}
	if job.Error != "" {
		resp["error"] = job.Error
	}
	if job.StartedAt != nil {
		resp["started_at"] = job.StartedAt
	}
	if job.CompletedAt != nil {
		resp["completed_at"] = job.CompletedAt
		if job.StartedAt != nil {
			resp["duration_ms"] = job.CompletedAt.Sub(*job.StartedAt).Milliseconds()
		} else {
			resp["duration_ms"] = job.CompletedAt.Sub(job.CreatedAt).Milliseconds()
		}
	}
	if job.Credits > 0 {
		resp["credits_charged"] = job.Credits
	}
	return resp
}

func (d *mediaDeps) handleStatus(w http.ResponseWriter, r *http.Request) {
	job := d.jobs.Get(chi.URLParam(r, "jobID"))
	if job == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "job not found"})
		return
	}
	writeJSON(w, http.StatusOK, mediaJobResponse(job))
}

func (d *mediaDeps) handleList(w http.ResponseWriter, r *http.Request) {
	jobs := d.jobs.List(50)
	projections := make([]map[string]any, 0, len(jobs))
	for _, job := range jobs {
		projections = append(projections, mediaJobResponse(job))
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": projections, "count": len(projections)})
}

// executeJob runs media production locally with a timeout.
func (d *mediaDeps) executeJob(job *media.Job) {
	now := time.Now().UTC()
	d.jobs.Update(job.ID, func(j *media.Job) {
		j.Status = media.StatusProcessing
		j.StartedAt = &now
	})

	timeout := time.Duration(d.cfg.Media.JobTimeout) * time.Second
	if timeout <= 0 {
		timeout = 300 * time.Second
	}

	type result struct {
		path string
		err  error
	}
	ch := make(chan result, 1)

	go func() {
		var err error
		var outputPath string
		switch job.Type {
		case media.TypeDeviceMockup:
			outputPath, err = d.produceMockup(job)
		case media.TypeAnimatedGIF:
			outputPath, err = d.produceGIF(job)
		default:
			err = fmt.Errorf("unsupported media type: %s", job.Type)
		}
		ch <- result{path: outputPath, err: err}
	}()

	var err error
	var outputPath string
	select {
	case res := <-ch:
		outputPath = res.path
		err = res.err
	case <-time.After(timeout):
		err = fmt.Errorf("job timed out after %v", timeout)
	}

	completed := time.Now().UTC()
	if err != nil {
		log.Printf("[media] Job %s failed: %v", job.ID, err)
		d.jobs.Update(job.ID, func(j *media.Job) {
			j.Status = media.StatusFailed
			j.Error = err.Error()
			j.CompletedAt = &completed
		})
		return
	}

	log.Printf("[media] Job %s produced local compatibility output: %s", job.ID, outputPath)
	delivery, err := d.publishMediaOutput(job, outputPath, completed)
	if err != nil {
		log.Printf("[media] Job %s EPWA publication failed: %v", job.ID, err)
		d.jobs.Update(job.ID, func(j *media.Job) {
			j.Status = media.StatusFailed
			j.ResultPath = outputPath
			j.Error = "EPWA publication failed; raw output withheld"
			j.CompletedAt = &completed
		})
		return
	}
	if delivery.State != epwadelivery.StateReady {
		d.jobs.Update(job.ID, func(j *media.Job) {
			j.Status = media.StatusDeliveryBlocked
			j.ResultPath = outputPath
			j.EPWADelivery = &delivery
			j.Error = "EPWA delivery is not ready; raw output withheld"
			j.CompletedAt = &completed
		})
		return
	}

	creditCost := 0.0
	if d.credits != nil {
		creditCost = d.credits.Cost(string(job.Type))
	}
	if d.credits != nil && job.LicenseID > 0 && creditCost > 0 {
		go d.credits.Deduct(job.LicenseID, string(job.Type), fmt.Sprintf("media_job:%s", job.ID))
	}
	d.jobs.Update(job.ID, func(j *media.Job) {
		j.Status = media.StatusComplete
		j.ResultPath = outputPath
		j.ResultURL = ""
		j.EPWADelivery = &delivery
		j.Credits = creditCost
		j.CompletedAt = &completed
	})

	// Record usage
	if d.usage != nil {
		d.usage.Record(storage.UsageRecord{
			Type:      string(job.Type),
			Status:    "success",
			CostUSD:   creditCost * 0.005, // ~$0.005 per credit
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		})
	}
}

func (d *mediaDeps) publishMediaOutput(job *media.Job, outputPath string, capturedAt time.Time) (epwadelivery.Delivery, error) {
	payload, err := os.ReadFile(outputPath) // #nosec G304 -- outputPath is generated inside the configured media output directory.
	if err != nil {
		return epwadelivery.Delivery{}, err
	}
	extension := strings.TrimPrefix(strings.ToLower(filepath.Ext(outputPath)), ".")
	if extension == "" {
		extension = "bin"
	}
	base, _ := url.Parse(job.DeliveryBaseURL)
	return publishGenericEPWAAtBase(d.cfg, evidenceshare.GenericInput{
		ArtifactRef: "media:job:" + job.ID,
		Revision:    1,
		Title:       "Media production " + job.ID,
		Kind:        "media",
		MediaType:   http.DetectContentType(payload),
		Extension:   extension,
		Payload:     payload,
		CapturedAt:  capturedAt,
		Scope:       job.DeliveryScope,
	}, base)
}

func (d *mediaDeps) produceMockup(job *media.Job) (string, error) {
	if len(job.URLs) == 0 {
		return "", fmt.Errorf("no URLs provided")
	}

	scriptDir := d.cfg.Media.ScriptDir
	if scriptDir == "" {
		scriptDir = "/home/wpuiai/public_html/wp-content/plugins/wpuiai/assets/templates/devices"
	}
	script := filepath.Join(scriptDir, "generate-mockup.mjs")

	outputDir := filepath.Join(d.cfg.Storage.DataDir, "media")
	outputPath := filepath.Join(outputDir, fmt.Sprintf("mockup-%s-%s.png", job.Device, job.ID))

	args := []string{
		script,
		fmt.Sprintf("--url=%s", job.URLs[0]),
		fmt.Sprintf("--device=%s", job.Device),
		fmt.Sprintf("--output=%s", outputPath),
	}
	if job.Width > 0 {
		args = append(args, fmt.Sprintf("--width=%d", job.Width))
	}
	if job.Height > 0 {
		args = append(args, fmt.Sprintf("--height=%d", job.Height))
	}

	nodePath := findNode()
	cmd := exec.Command(nodePath, args...) // #nosec G204 -- nodePath is from fixed allowlist and args are constructed flags.
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("mockup script failed: %v\nOutput: %s", err, truncate(string(out), 500))
	}

	return outputPath, nil
}

func (d *mediaDeps) produceGIF(job *media.Job) (string, error) {
	if len(job.URLs) == 0 {
		return "", fmt.Errorf("no URLs provided")
	}

	scriptDir := d.cfg.Media.ScriptDir
	if scriptDir == "" {
		scriptDir = "/home/wpuiai/public_html/wp-content/plugins/wpuiai/assets/templates/devices"
	}
	script := filepath.Join(scriptDir, "generate-gif.mjs")

	outputDir := filepath.Join(d.cfg.Storage.DataDir, "media")
	outputPath := filepath.Join(outputDir, fmt.Sprintf("tour-%s.gif", job.ID))

	var args []string
	if len(job.URLs) > 1 {
		urlsJSON, _ := json.Marshal(job.URLs)
		args = []string{
			script,
			fmt.Sprintf("--urls=%s", string(urlsJSON)),
			"--mode=pages",
			fmt.Sprintf("--output=%s", outputPath),
			fmt.Sprintf("--delay=%d", job.Delay),
			fmt.Sprintf("--width=%d", job.Width),
			fmt.Sprintf("--height=%d", job.Height),
		}
	} else {
		args = []string{
			script,
			fmt.Sprintf("--url=%s", job.URLs[0]),
			"--mode=scroll",
			fmt.Sprintf("--frames=%d", job.Frames),
			fmt.Sprintf("--output=%s", outputPath),
			fmt.Sprintf("--delay=%d", job.Delay),
			fmt.Sprintf("--width=%d", job.Width),
			fmt.Sprintf("--height=%d", job.Height),
		}
	}

	nodePath := findNode()
	cmd := exec.Command(nodePath, args...) // #nosec G204 -- nodePath is from fixed allowlist and args are constructed flags.
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("GIF script failed: %v\nOutput: %s", err, truncate(string(out), 500))
	}

	return outputPath, nil
}

// findNode returns the full path to Node.js.
func findNode() string {
	// Try common locations
	for _, p := range []string{
		"/opt/cpanel/ea-nodejs20/bin/node",
		"/usr/local/bin/node",
		"/usr/bin/node",
	} {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return "node" // fallback to PATH
}

// validateMediaURL checks that a URL is safe to screenshot (no SSRF).
func validateMediaURL(rawURL string) error {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("malformed URL")
	}
	// Must be https (or http for localhost dev)
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("scheme must be http or https")
	}
	// Resolve hostname
	host := u.Hostname()
	if host == "" {
		return fmt.Errorf("missing hostname")
	}
	// Block well-known internal hostnames
	blocked := []string{"localhost", "127.0.0.1", "0.0.0.0", "[::]", "[::1]", "metadata.google.internal"}
	for _, b := range blocked {
		if host == b {
			return fmt.Errorf("internal hostname blocked")
		}
	}
	// Resolve and check for private IPs
	ips, err := net.LookupHost(host) // #nosec G704 -- DNS lookup is used only to reject private/internal addresses before outbound media access.
	if err == nil {
		for _, ipStr := range ips {
			ip := net.ParseIP(ipStr)
			if ip == nil {
				continue
			}
			if ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
				return fmt.Errorf("resolves to private/internal IP %s", ipStr)
			}
		}
	}
	return nil
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
