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
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/config"
	"github.com/philoveracity/uiai-engine/internal/credits"
	"github.com/philoveracity/uiai-engine/internal/media"
	"github.com/philoveracity/uiai-engine/internal/ratelimit"
	"github.com/philoveracity/uiai-engine/internal/storage"
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

	job := &media.Job{
		ID:        jobID,
		Type:      req.Type,
		Status:    media.StatusPending,
		Device:    req.Device,
		URLs:      req.URLs,
		Width:     req.Width,
		Height:    req.Height,
		Frames:    req.Frames,
		Delay:     req.Delay,
		Mode:      req.Mode,
		CreatedAt: time.Now().UTC(),
	}

	d.jobs.Create(job)

	// Execute synchronously for now (local server has all tools)
	// TODO: dispatch to GitHub Actions for scale
	go d.executeJob(job)

	writeJSON(w, 202, map[string]any{
		"job_id": jobID,
		"status": "pending",
		"type":   req.Type,
		"message": "Media production started",
	})
}

func (d *mediaDeps) handleStatus(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobID")
	job := d.jobs.Get(jobID)
	if job == nil {
		writeJSON(w, 404, map[string]string{"error": "job not found"})
		return
	}

	resp := map[string]any{
		"job_id":    job.ID,
		"type":      job.Type,
		"status":    job.Status,
		"created_at": job.CreatedAt,
	}
	if job.ResultURL != "" {
		resp["result_url"] = job.ResultURL
	}
	// Expose result_path — status endpoint is open (read-only, no auth required)
	// but the path is only useful on the same server. External callers would use result_url.
	if job.ResultPath != "" {
		resp["result_path"] = job.ResultPath
	}
	if job.Error != "" {
		resp["error"] = job.Error
	}
	if job.CompletedAt != nil {
		resp["completed_at"] = job.CompletedAt
		resp["duration_ms"] = job.CompletedAt.Sub(job.CreatedAt).Milliseconds()
	}

	writeJSON(w, 200, resp)
}

func (d *mediaDeps) handleList(w http.ResponseWriter, r *http.Request) {
	jobs := d.jobs.List(50)
	writeJSON(w, 200, map[string]any{
		"jobs":  jobs,
		"count": len(jobs),
	})
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

	log.Printf("[media] Job %s complete: %s", job.ID, outputPath)
	d.jobs.Update(job.ID, func(j *media.Job) {
		j.Status = media.StatusComplete
		j.ResultPath = outputPath
		j.CompletedAt = &completed
	})

	// Record usage
	if d.usage != nil {
		d.usage.Record(storage.UsageRecord{
			Type:      string(job.Type),
			Status:    "success",
			CostUSD:   0.01, // Minimal cost for local execution
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		})
	}
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
	cmd := exec.Command(nodePath, args...)
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
	cmd := exec.Command(nodePath, args...)
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
	ips, err := net.LookupHost(host)
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


