package routes

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/config"
)

var (
	siteStore    sync.Map // siteId → *registeredSite
	workflowRuns sync.Map // runId → *workflowRun
)

type registeredSite struct {
	ID            string    `json:"id"`
	SiteURL       string    `json:"site_url"`
	AdminUsername string    `json:"admin_username,omitempty"`
	PluginSlug    string    `json:"plugin_slug"`
	RESTBase      string    `json:"rest_base"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	LastPing      time.Time `json:"last_ping,omitempty"`
}

type workflowRun struct {
	ID        string         `json:"id"`
	SiteID    string         `json:"site_id"`
	Status    string         `json:"status"` // pending, running, completed, failed
	Step      int            `json:"step"`
	TotalSteps int           `json:"total_steps"`
	Log       []string       `json:"log"`
	Result    map[string]any `json:"result,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
}

func MountWorkflowReal(r chi.Router, cfg *config.Config) {
	// --- Site registration ---
	r.Post("/sites", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			SiteURL       string `json:"site_url"`
			AdminUsername string `json:"admin_username"`
			AdminPassword string `json:"admin_password"`
			PluginSlug    string `json:"plugin_slug"`
			PluginZipURL  string `json:"plugin_zip_url"`
		}
		json.NewDecoder(req.Body).Decode(&body)
		if body.SiteURL == "" {
			writeJSON(w, 400, map[string]string{"error": "site_url required"})
			return
		}
		if body.PluginSlug == "" {
			body.PluginSlug = "wpuiai"
		}

		id := "site_" + time.Now().Format("20060102150405")
		site := &registeredSite{
			ID:            id,
			SiteURL:       body.SiteURL,
			AdminUsername: body.AdminUsername,
			PluginSlug:    body.PluginSlug,
			RESTBase:      body.SiteURL + "/wp-json/wpuiai/v1",
			Status:        "registered",
			CreatedAt:     time.Now(),
		}

		// Ping the site to verify it's reachable
		go verifySite(site)

		siteStore.Store(id, site)
		writeJSON(w, 201, map[string]any{"success": true, "site": site})
	})

	r.Get("/sites", func(w http.ResponseWriter, req *http.Request) {
		var sites []any
		siteStore.Range(func(_, v any) bool {
			sites = append(sites, v)
			return true
		})
		if sites == nil {
			sites = []any{}
		}
		writeJSON(w, 200, map[string]any{"sites": sites, "count": len(sites)})
	})

	r.Get("/sites/{id}", func(w http.ResponseWriter, req *http.Request) {
		if v, ok := siteStore.Load(chi.URLParam(req, "id")); ok {
			writeJSON(w, 200, v)
		} else {
			writeJSON(w, 404, map[string]string{"error": "site not found"})
		}
	})

	r.Delete("/sites/{id}", func(w http.ResponseWriter, req *http.Request) {
		siteStore.Delete(chi.URLParam(req, "id"))
		writeJSON(w, 200, map[string]string{"message": "site removed"})
	})

	r.Get("/sites/{id}/status", func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")
		v, ok := siteStore.Load(id)
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "site not found"})
			return
		}
		site := v.(*registeredSite)
		writeJSON(w, 200, map[string]any{
			"site_id": id, "status": site.Status,
			"last_ping": site.LastPing, "site_url": site.SiteURL,
		})
	})

	r.Post("/sites/{id}/ping", func(w http.ResponseWriter, req *http.Request) {
		v, ok := siteStore.Load(chi.URLParam(req, "id"))
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "site not found"})
			return
		}
		site := v.(*registeredSite)
		err := verifySite(site)
		if err != nil {
			writeJSON(w, 200, map[string]any{"reachable": false, "error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]any{"reachable": true, "status": site.Status})
	})

	// --- Workflow operations (remote orchestration) ---

	r.Post("/sites/{id}/workflow/create-run", func(w http.ResponseWriter, req *http.Request) {
		siteId := chi.URLParam(req, "id")
		var body map[string]any
		json.NewDecoder(req.Body).Decode(&body)

		runId := "run-" + time.Now().Format("20060102-150405")
		run := &workflowRun{
			ID:         runId,
			SiteID:     siteId,
			Status:     "pending",
			Step:       0,
			TotalSteps: 29,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}
		workflowRuns.Store(runId, run)
		writeJSON(w, 201, map[string]any{"success": true, "run_id": runId, "status": "pending"})
	})

	r.Post("/sites/{id}/workflow/execute", func(w http.ResponseWriter, req *http.Request) {
		siteId := chi.URLParam(req, "id")
		var body struct {
			RunID  string `json:"run_id"`
			Step   int    `json:"step"`
			Action string `json:"action"`
			Params any    `json:"params"`
		}
		json.NewDecoder(req.Body).Decode(&body)

		// Forward the action to the remote WP site via REST
		v, ok := siteStore.Load(siteId)
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "site not found"})
			return
		}
		site := v.(*registeredSite)

		result, err := callRemoteWP(site, "workflow/execute", map[string]any{
			"run_id": body.RunID, "step": body.Step,
			"action": body.Action, "params": body.Params,
		})
		if err != nil {
			writeJSON(w, 502, map[string]string{"error": "remote execution failed: " + err.Error()})
			return
		}

		// Update run state
		if v, ok := workflowRuns.Load(body.RunID); ok {
			run := v.(*workflowRun)
			run.Step = body.Step
			run.Status = "running"
			run.UpdatedAt = time.Now()
			run.Log = append(run.Log, fmt.Sprintf("Step %d: %s", body.Step, body.Action))
		}

		writeJSON(w, 200, map[string]any{"success": true, "result": result})
	})

	for _, action := range []string{"start", "run", "complete", "skip", "set-active-run"} {
		act := action
		r.Post("/sites/{id}/workflow/"+act, func(w http.ResponseWriter, req *http.Request) {
			siteId := chi.URLParam(req, "id")
			var body map[string]any
			json.NewDecoder(req.Body).Decode(&body)
			writeJSON(w, 200, map[string]any{
				"success": true, "site_id": siteId, "action": act,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
		})
	}

	// --- Workflow run status ---
	r.Get("/runs/{runId}", func(w http.ResponseWriter, req *http.Request) {
		if v, ok := workflowRuns.Load(chi.URLParam(req, "runId")); ok {
			writeJSON(w, 200, v)
		} else {
			writeJSON(w, 404, map[string]string{"error": "run not found"})
		}
	})

	r.Get("/runs", func(w http.ResponseWriter, req *http.Request) {
		var runs []any
		workflowRuns.Range(func(_, v any) bool {
			runs = append(runs, v)
			return true
		})
		if runs == nil {
			runs = []any{}
		}
		writeJSON(w, 200, map[string]any{"runs": runs, "count": len(runs)})
	})

	// --- Templates ---
	r.Get("/templates", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{"templates": []any{
			map[string]any{"id": "default", "name": "Default 29-Step MIMIC", "steps": 29},
			map[string]any{"id": "lite", "name": "Quick 10-Step", "steps": 10},
			map[string]any{"id": "full", "name": "Full 30-Step with Verify", "steps": 30},
		}})
	})

	r.Get("/templates/{id}", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{"id": chi.URLParam(req, "id"), "name": "Default 29-Step MIMIC", "steps": 29})
	})

	// --- Direct execution (no site registration needed) ---
	r.Post("/execute", func(w http.ResponseWriter, req *http.Request) {
		var body map[string]any
		json.NewDecoder(req.Body).Decode(&body)
		writeJSON(w, 200, map[string]any{"status": "accepted", "timestamp": time.Now().UTC().Format(time.RFC3339)})
	})

	r.Post("/run", func(w http.ResponseWriter, req *http.Request) {
		var body map[string]any
		json.NewDecoder(req.Body).Decode(&body)
		runId := "run-" + time.Now().Format("20060102-150405")
		writeJSON(w, 200, map[string]any{"run_id": runId, "status": "started"})
	})

	r.Get("/status/{runId}", func(w http.ResponseWriter, req *http.Request) {
		runId := chi.URLParam(req, "runId")
		if v, ok := workflowRuns.Load(runId); ok {
			writeJSON(w, 200, v)
		} else {
			writeJSON(w, 200, map[string]any{"run_id": runId, "status": "unknown"})
		}
	})
}

// verifySite pings a registered WP site to check reachability.
func verifySite(site *registeredSite) error {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(site.SiteURL + "/wp-json/")
	if err != nil {
		site.Status = "unreachable"
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 200 {
		site.Status = "ready"
		site.LastPing = time.Now()
	} else {
		site.Status = "error"
		return fmt.Errorf("site returned HTTP %d", resp.StatusCode)
	}
	return nil
}

// callRemoteWP sends a POST request to a registered WP site's REST API.
func callRemoteWP(site *registeredSite, path string, body any) (map[string]any, error) {
	payloadBytes, _ := json.Marshal(body)
	url := site.RESTBase + "/" + path

	client := &http.Client{Timeout: 60 * time.Second}
	req, _ := http.NewRequest("POST", url, bytes.NewReader(payloadBytes))
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("remote WP %d: %s", resp.StatusCode, string(respBody[:min(len(respBody), 200)]))
	}

	var result map[string]any
	json.Unmarshal(respBody, &result)
	return result, nil
}
