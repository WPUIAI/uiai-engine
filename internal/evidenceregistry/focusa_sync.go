package evidenceregistry

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type FocusaSyncConfig struct {
	BaseURL             string
	TokenFile           string
	BRPath              string
	ProjectIDs          []string
	AllowedRootPrefixes []string
	MaxProjects         int
	MaxItems            int
	HTTPClient          *http.Client
}

type FocusaSyncResult struct {
	Projects        uint64                `json:"projects"`
	ChangedProjects uint64                `json:"changed_projects"`
	Items           uint64                `json:"items"`
	Results         []ProviderGraphResult `json:"results"`
	Warnings        []string              `json:"warnings,omitempty"`
}

type focusaProjectDashboard struct {
	Schema   string          `json:"schema"`
	Projects []focusaProject `json:"projects"`
}

type focusaProject struct {
	ProjectID     string `json:"project_id"`
	CanonicalName string `json:"canonical_name"`
	ProjectRoot   string `json:"project_root"`
	Fingerprint   string `json:"fingerprint"`
	WorkspaceKind string `json:"workspace_kind"`
	ScopeSafety   string `json:"scope_safety"`
	LastVerified  string `json:"last_verified_at"`
}

type brList struct {
	Issues  []brIssue `json:"issues"`
	Total   int       `json:"total"`
	HasMore bool      `json:"has_more"`
}

type brIssue struct {
	ID             string   `json:"id"`
	Title          string   `json:"title"`
	Description    string   `json:"description"`
	Status         string   `json:"status"`
	Priority       int      `json:"priority"`
	IssueType      string   `json:"issue_type"`
	UpdatedAt      string   `json:"updated_at"`
	CreatedAt      string   `json:"created_at"`
	ExternalRef    string   `json:"external_ref"`
	Labels         []string `json:"labels"`
	SourceRepoPath string   `json:"source_repo_path"`
}

type brGraph struct {
	Components []struct {
		Edges [][]string `json:"edges"`
	} `json:"components"`
}

func (m *Manager) SyncFocusa(ctx context.Context, cfg FocusaSyncConfig) (FocusaSyncResult, error) {
	if m == nil {
		return FocusaSyncResult{}, ErrConfig
	}
	if cfg.MaxProjects <= 0 {
		cfg.MaxProjects = 100
	}
	if cfg.MaxItems <= 0 {
		cfg.MaxItems = 100000
	}
	if cfg.MaxProjects > 1000 || cfg.MaxItems > 100000 || strings.TrimSpace(cfg.BRPath) == "" || !filepath.IsAbs(cfg.BRPath) {
		return FocusaSyncResult{}, ErrConfig
	}
	if _, err := exec.LookPath(cfg.BRPath); err != nil {
		return FocusaSyncResult{}, ErrConfig
	}
	base, err := url.Parse(strings.TrimRight(cfg.BaseURL, "/"))
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" {
		return FocusaSyncResult{}, ErrConfig
	}
	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base.String()+"/v1/project/list", nil)
	if err != nil {
		return FocusaSyncResult{}, ErrConfig
	}
	response, err := client.Do(req)
	if err != nil {
		return FocusaSyncResult{}, fmt.Errorf("%w: Focusa project list: %v", ErrIndexUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return FocusaSyncResult{}, fmt.Errorf("%w: Focusa project list HTTP %d", ErrIndexUnavailable, response.StatusCode)
	}
	var dashboard focusaProjectDashboard
	if err := json.NewDecoder(response.Body).Decode(&dashboard); err != nil || dashboard.Schema != "focusa.project_dashboard.v1" {
		return FocusaSyncResult{}, fmt.Errorf("%w: invalid Focusa project dashboard", ErrIndexCorrupt)
	}
	allowedRoots, err := canonicalAllowedRoots(cfg.AllowedRootPrefixes)
	if err != nil {
		return FocusaSyncResult{}, err
	}
	selected := make(map[string]struct{}, len(cfg.ProjectIDs))
	for _, id := range cfg.ProjectIDs {
		selected[strings.TrimSpace(id)] = struct{}{}
	}
	counts := make(map[string]int)
	for _, project := range dashboard.Projects {
		counts[project.ProjectID]++
	}
	result := FocusaSyncResult{Results: make([]ProviderGraphResult, 0)}
	for _, project := range dashboard.Projects {
		if len(result.Results) >= cfg.MaxProjects {
			result.Warnings = append(result.Warnings, "project_limit_reached")
			break
		}
		if len(selected) > 0 {
			if _, ok := selected[project.ProjectID]; !ok {
				continue
			}
		}
		if project.ScopeSafety != "safe" || project.ProjectRoot == "" || project.ProjectID == "" || project.CanonicalName == "" || project.Fingerprint == "" {
			result.Warnings = append(result.Warnings, "unsafe_or_incomplete_project_skipped:"+project.ProjectID)
			continue
		}
		providerRoot, err := allowedProviderRoot(project.ProjectRoot, allowedRoots)
		if err != nil {
			result.Warnings = append(result.Warnings, "project_root_not_allowed:"+project.ProjectID)
			continue
		}
		projectRef := "project:" + project.ProjectID
		if counts[project.ProjectID] > 1 {
			projectRef += ":" + shortFingerprint(project.Fingerprint)
		}
		items, warning, err := readBRProject(ctx, cfg.BRPath, providerRoot, projectRef, cfg.MaxItems)
		if err != nil {
			result.Warnings = append(result.Warnings, "provider_unavailable:"+projectRef)
			continue
		}
		if warning != "" {
			result.Warnings = append(result.Warnings, warning+":"+projectRef)
		}
		store, err := m.EnsureProject(ctx, projectRef)
		if err != nil {
			return result, err
		}
		graphResult, err := store.ReplaceProviderGraph(ctx, ProviderGraphInput{Project: ProjectProjection{
			ProjectRef: projectRef, ProjectID: project.ProjectID, DisplayName: project.CanonicalName,
			Fingerprint: project.Fingerprint, WorkspaceKind: project.WorkspaceKind, ScopeSafety: project.ScopeSafety,
			SourceSchema: dashboard.Schema, SourceRevision: project.LastVerified, ObservedAt: time.Now().UTC(),
		}, Items: items})
		if err != nil {
			return result, err
		}
		result.Projects++
		if graphResult.Changed {
			result.ChangedProjects++
		}
		result.Items += graphResult.Items
		result.Results = append(result.Results, graphResult)
	}
	return result, nil
}

func canonicalAllowedRoots(values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, ErrConfig
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = filepath.Clean(strings.TrimSpace(value))
		if !filepath.IsAbs(value) {
			return nil, ErrConfig
		}
		resolved, err := filepath.EvalSymlinks(value)
		if err != nil {
			return nil, ErrConfig
		}
		out = append(out, resolved)
	}
	return uniqueStrings(out), nil
}

func allowedProviderRoot(value string, allowed []string) (string, error) {
	if !filepath.IsAbs(value) {
		return "", ErrConfig
	}
	resolved, err := filepath.EvalSymlinks(filepath.Clean(value))
	if err != nil {
		return "", ErrConfig
	}
	for _, root := range allowed {
		if resolved == root || strings.HasPrefix(resolved, root+string(filepath.Separator)) {
			return resolved, nil
		}
	}
	return "", ErrConfig
}

func readBRProject(ctx context.Context, executable, root, projectRef string, limit int) ([]ProviderWorkItem, string, error) {
	maxListBytes := int64(limit) * 16 * 1024
	if maxListBytes < 8<<20 {
		maxListBytes = 8 << 20
	}
	if maxListBytes > 128<<20 {
		maxListBytes = 128 << 20
	}
	body, err := boundedCommandOutput(ctx, executable, root, maxListBytes, "list", "--all", "--limit", strconv.Itoa(limit), "--json", "--no-auto-flush", "--no-auto-import", "--allow-stale")
	if err != nil {
		return nil, "", err
	}
	var listed brList
	if err := json.Unmarshal(body, &listed); err != nil {
		return nil, "", err
	}
	allowedIssues := make(map[string]brIssue, len(listed.Issues))
	skippedScope := 0
	for _, issue := range listed.Issues {
		if issue.SourceRepoPath != "" {
			issueRoot, resolveErr := filepath.EvalSymlinks(filepath.Clean(issue.SourceRepoPath))
			if resolveErr != nil || issueRoot != root {
				skippedScope++
				continue
			}
		}
		allowedIssues[issue.ID] = issue
	}
	dependencies := make(map[string][]string)
	graphBody, graphErr := boundedCommandOutput(ctx, executable, root, 64<<20, "graph", "--all", "--json", "--no-auto-flush", "--no-auto-import", "--allow-stale")
	if graphErr == nil {
		var graph brGraph
		if json.Unmarshal(graphBody, &graph) == nil {
			for _, component := range graph.Components {
				for _, edge := range component.Edges {
					if len(edge) == 2 {
						_, sourceOK := allowedIssues[edge[0]]
						_, targetOK := allowedIssues[edge[1]]
						if sourceOK && targetOK {
							dependencies[edge[0]] = append(dependencies[edge[0]], "work-item:br:"+edge[1])
						}
					}
				}
			}
		}
	}
	items := make([]ProviderWorkItem, 0, len(allowedIssues))
	observed := time.Now().UTC()
	for _, issue := range allowedIssues {
		revision := issue.UpdatedAt
		if revision == "" {
			revision = issue.CreatedAt
		}
		ref := "work-item:br:" + issue.ID
		item := ProviderWorkItem{ProjectRef: projectRef, WorkItemRef: ref, ProviderSurface: "br", ItemID: issue.ID,
			ItemType: issue.IssueType, Title: issue.Title, Description: issue.Description, Status: issue.Status,
			Priority: issue.Priority, Revision: revision, DependencyRefs: dependencies[issue.ID], ExternalRef: issue.ExternalRef,
			SourceAuthority: "provider:br", BindingState: "focusa_binding_pending", ObservedAt: observed}
		item.Digest = providerWorkItemDigest(item)
		items = append(items, item)
	}
	warning := ""
	if listed.HasMore || listed.Total > len(listed.Issues) {
		warning = "provider_item_limit_reached"
	} else if graphErr != nil {
		warning = "dependency_graph_unavailable"
	} else if skippedScope > 0 {
		warning = "cross_project_items_filtered"
	}
	return items, warning, nil
}

type boundedBuffer struct {
	bytes.Buffer
	max int64
}

func (b *boundedBuffer) Write(value []byte) (int, error) {
	remaining := b.max - int64(b.Len())
	if remaining <= 0 {
		return 0, fmt.Errorf("provider output exceeds %d bytes", b.max)
	}
	if int64(len(value)) > remaining {
		written, _ := b.Buffer.Write(value[:remaining])
		return written, fmt.Errorf("provider output exceeds %d bytes", b.max)
	}
	return b.Buffer.Write(value)
}

func boundedCommandOutput(ctx context.Context, executable, root string, maxBytes int64, args ...string) ([]byte, error) {
	output := &boundedBuffer{max: maxBytes}
	command := exec.CommandContext(ctx, executable, args...)
	command.Dir = root
	command.Stdout = output
	if err := command.Run(); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func providerWorkItemDigest(item ProviderWorkItem) string {
	item.Digest = ""
	item.ObservedAt = time.Time{}
	body, _ := json.Marshal(item)
	digest := sha256.Sum256(body)
	return hex.EncodeToString(digest[:])
}

func shortFingerprint(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])[:12]
}
