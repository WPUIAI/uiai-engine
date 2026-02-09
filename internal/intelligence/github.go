package intelligence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// GitHubConfig holds GitHub Actions dispatch configuration.
type GitHubConfig struct {
	Token       string // GITHUB_TOKEN
	Repo        string // e.g. "WPUIAI/ai-api"
	WorkflowID  string // e.g. "build-docfind-index.yml"
	WorkflowRef string // e.g. "main"
}

// IsConfigured returns true if GitHub dispatch is configured.
func (g *GitHubConfig) IsConfigured() bool {
	return g.Token != "" && g.Repo != ""
}

// DispatchResult is the outcome of a GitHub Actions dispatch.
type DispatchResult struct {
	OK     bool   `json:"ok"`
	Status int    `json:"status,omitempty"`
	Error  string `json:"error,omitempty"`
}

// TriggerDocfindBuild dispatches a GitHub Actions workflow to build a DocFind index.
func TriggerDocfindBuild(cfg *GitHubConfig, runID, buildID string) DispatchResult {
	if !cfg.IsConfigured() {
		return DispatchResult{OK: false, Error: "GITHUB_TOKEN or GITHUB_REPO not configured"}
	}

	workflowID := cfg.WorkflowID
	if workflowID == "" {
		workflowID = "build-docfind-index.yml"
	}
	ref := cfg.WorkflowRef
	if ref == "" {
		ref = "main"
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/actions/workflows/%s/dispatches", cfg.Repo, workflowID)

	body, _ := json.Marshal(map[string]any{
		"ref": ref,
		"inputs": map[string]string{
			"run_id":   runID,
			"build_id": buildID,
		},
	})

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return DispatchResult{OK: false, Error: err.Error()}
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+cfg.Token)
	req.Header.Set("User-Agent", "uiai-engine-intelligence")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return DispatchResult{OK: false, Error: err.Error()}
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errBody, _ := io.ReadAll(resp.Body)
		errStr := string(errBody)
		if len(errStr) > 200 {
			errStr = errStr[:200]
		}
		return DispatchResult{OK: false, Status: resp.StatusCode, Error: errStr}
	}

	return DispatchResult{OK: true, Status: resp.StatusCode}
}
