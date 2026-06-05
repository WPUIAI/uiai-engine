package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/focusapacket"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

// MountMarkdownRoutes registers the one-shot Source-to-Markdown API.
func MountMarkdownRoutes(r chi.Router, _ *config.Config, sm *vision.SessionManager) {
	if sm == nil {
		return
	}

	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		body := markdownRequestFromQuery(req)
		handleMarkdownRequest(w, body, sm)
	})

	r.Post("/", func(w http.ResponseWriter, req *http.Request) {
		var body markdownRequest
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
			return
		}
		handleMarkdownRequest(w, body, sm)
	})
}

type markdownRequest struct {
	URL           string              `json:"url"`
	Selector      string              `json:"selector,omitempty"`
	MaxChars      int                 `json:"max_chars,omitempty"`
	Mode          string              `json:"mode,omitempty"`
	IncludeLinks  *bool               `json:"include_links,omitempty"`
	IncludeImages bool                `json:"include_images,omitempty"`
	Width         int                 `json:"width,omitempty"`
	Height        int                 `json:"height,omitempty"`
	FocusaScope   *vision.FocusaScope `json:"focusa_scope,omitempty"`
	WorkpointID   string              `json:"workpoint_id,omitempty"`
	ContinuityID  string              `json:"continuity_id,omitempty"`
	ProjectRoot   string              `json:"project_root,omitempty"`
	EvidenceRef   string              `json:"evidence_ref,omitempty"`
}

func markdownRequestFromQuery(req *http.Request) markdownRequest {
	q := req.URL.Query()
	maxChars, _ := strconv.Atoi(q.Get("max_chars"))
	width, _ := strconv.Atoi(q.Get("width"))
	height, _ := strconv.Atoi(q.Get("height"))
	var includeLinks *bool
	if q.Has("include_links") {
		v := truthy(q.Get("include_links"))
		includeLinks = &v
	}
	return markdownRequest{
		URL:           q.Get("url"),
		Selector:      q.Get("selector"),
		MaxChars:      maxChars,
		Mode:          q.Get("mode"),
		IncludeLinks:  includeLinks,
		IncludeImages: truthy(q.Get("include_images")),
		Width:         width,
		Height:        height,
		WorkpointID:   q.Get("workpoint_id"),
		ContinuityID:  q.Get("continuity_id"),
		ProjectRoot:   q.Get("project_root"),
		EvidenceRef:   q.Get("evidence_ref"),
	}
}

func handleMarkdownRequest(w http.ResponseWriter, body markdownRequest, sm *vision.SessionManager) {
	if strings.TrimSpace(body.URL) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "url required"})
		return
	}
	if body.MaxChars <= 0 {
		body.MaxChars = 30000
	}
	started := time.Now()
	sess, _, err := sm.Open(body.URL, body.Width, body.Height)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, markdownFailure("source_open_failed", err, body.URL, started))
		return
	}
	sess.SetFocusaScope(resolveFocusaScope(body.FocusaScope, body.WorkpointID, body.ContinuityID, body.ProjectRoot, body.EvidenceRef))
	cleanup := map[string]any{"closed": false, "session_id": sess.ID}

	includeLinks := true
	if body.IncludeLinks != nil {
		includeLinks = *body.IncludeLinks
	}
	read, err := sess.ReadPage(vision.ReadOptions{
		Selector:      body.Selector,
		MaxChars:      body.MaxChars,
		IncludeLinks:  includeLinks,
		Format:        "markdown",
		Mode:          body.Mode,
		IncludeImages: body.IncludeImages,
	})
	diagnostics := sess.Diagnostics(25, "", false)
	if err != nil {
		_ = sm.Close(sess.ID)
		writeJSON(w, http.StatusBadGateway, markdownFailure("source_read_failed", err, body.URL, started))
		return
	}
	if err := sm.Close(sess.ID); err == nil {
		cleanup["closed"] = true
	}

	capturedAt := time.Now().UTC().Format(time.RFC3339)
	metadata := markdownMetadata(read, body, capturedAt)
	var records []map[string]any
	if gh, ok := matchGitHubPublicSource(firstMarkdownNonEmpty(read.URL, body.URL)); ok {
		metadata = applyGitHubPublicMetadata(metadata, gh)
		read.Text = renderGitHubPublicMarkdown(read.Text, read, gh)
		read.Chars = len(read.Text)
		records = githubPublicRecords(read, gh)
		metadata["record_count"] = len(records)
	} else if rd, ok := matchRedditPublicSource(firstMarkdownNonEmpty(read.URL, body.URL)); ok {
		metadata = applyRedditPublicMetadata(metadata, rd)
		read.Text = renderRedditPublicMarkdown(read.Text, read, rd)
		read.Chars = len(read.Text)
		records = redditPublicRecords(read, rd)
		metadata["record_count"] = len(records)
	}
	markdown := read.Text
	response := map[string]any{
		"schema":       "uiai.source_markdown.v1",
		"url":          read.URL,
		"title":        read.Title,
		"description":  read.Description,
		"markdown":     markdown,
		"text":         markdown,
		"chars":        read.Chars,
		"truncated":    read.Truncated,
		"headings":     read.Headings,
		"links":        read.Links,
		"metadata":     metadata,
		"diagnostics":  diagnostics,
		"focusa":       oneShotMarkdownFocusa(read, body.URL),
		"cleanup":      cleanup,
		"duration_ms":  time.Since(started).Milliseconds(),
		"source_stats": map[string]any{"format": "markdown", "mode": read.Mode, "max_chars": body.MaxChars, "adapter": metadata["adapter"]},
	}
	if len(records) > 0 {
		response["records"] = records
	}
	writeJSON(w, http.StatusOK, response)
}

func markdownMetadata(read *vision.PageReadResult, body markdownRequest, capturedAt string) map[string]any {
	out := map[string]any{
		"source_type":    "webpage",
		"adapter":        "webpage_browser",
		"schema":         "uiai.source_markdown.v1",
		"captured_at":    capturedAt,
		"selector":       read.Selector,
		"mode":           read.Mode,
		"include_images": body.IncludeImages,
	}
	for k, v := range read.Metadata {
		out[k] = v
	}
	return out
}

func oneShotMarkdownFocusa(read *vision.PageReadResult, requestedURL string) map[string]any {
	evidenceRef := markdownEvidenceRef(read.URL, read.Text)
	title := focusapacket.Truncate(firstMarkdownNonEmpty(read.Title, read.URL, requestedURL), 160)
	return map[string]any{
		"target_ref":     "source-markdown:" + sanitizeMarkdownURLForFocusa(firstMarkdownNonEmpty(read.URL, requestedURL)),
		"evidence_ref":   evidenceRef,
		"preferred_tool": "focusa_evidence_capture",
		"summary":        fmt.Sprintf("Converted %s to %d chars of Markdown", title, read.Chars),
		"next_tools":     []string{"focusa_evidence_capture"},
	}
}

func markdownEvidenceRef(pageURL, markdown string) string {
	h := sha256.Sum256([]byte(pageURL + "\n" + markdown))
	return "uiai-source-markdown:sha256:" + hex.EncodeToString(h[:])[:16]
}

type githubPublicSource struct {
	Owner      string
	Repo       string
	Kind       string
	Number     string
	SourceType string
	FieldName  string
	URL        string
}

func matchGitHubPublicSource(raw string) (githubPublicSource, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return githubPublicSource{}, false
	}
	host := strings.ToLower(u.Hostname())
	if host != "github.com" && host != "www.github.com" {
		return githubPublicSource{}, false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 || parts[0] == "" || parts[1] == "" || parts[3] == "" {
		return githubPublicSource{}, false
	}
	gh := githubPublicSource{Owner: parts[0], Repo: parts[1], Kind: parts[2], Number: parts[3], URL: sanitizeMarkdownURLForFocusa(raw)}
	switch parts[2] {
	case "issues":
		gh.SourceType = "github_issue"
		gh.FieldName = "issue"
	case "pull":
		gh.SourceType = "github_pull_request"
		gh.FieldName = "pull_request"
	case "discussions":
		gh.SourceType = "github_discussion"
		gh.FieldName = "discussion"
	default:
		return githubPublicSource{}, false
	}
	return gh, true
}

func applyGitHubPublicMetadata(metadata map[string]any, gh githubPublicSource) map[string]any {
	out := make(map[string]any, len(metadata)+8)
	for k, v := range metadata {
		out[k] = v
	}
	out["adapter"] = "github_public"
	out["source_type"] = gh.SourceType
	out["repo"] = gh.Owner + "/" + gh.Repo
	out["owner"] = gh.Owner
	out["repository"] = gh.Repo
	out[gh.FieldName] = gh.Number
	out["canonical_url"] = gh.URL
	return out
}

func renderGitHubPublicMarkdown(markdown string, read *vision.PageReadResult, gh githubPublicSource) string {
	title := githubPublicTitle(read, gh)
	frontmatter := []string{
		"---",
		"source: " + gh.SourceType,
		"adapter: github_public",
		"repo: " + gh.Owner + "/" + gh.Repo,
		gh.FieldName + ": " + gh.Number,
		"url: " + gh.URL,
		"evidence_ref: " + markdownEvidenceRef(read.URL, markdown),
		"---",
		"",
		"# " + title,
		"",
	}
	trimmed := strings.TrimSpace(markdown)
	if trimmed == "" {
		trimmed = "No readable GitHub content extracted. Inspect diagnostics or retry with a narrower selector."
	}
	return strings.Join(frontmatter, "\n") + trimmed
}

func githubPublicTitle(read *vision.PageReadResult, gh githubPublicSource) string {
	title := strings.TrimSpace(read.Title)
	if title == "" {
		return gh.Owner + "/" + gh.Repo + " " + gh.Kind + " " + gh.Number
	}
	return title
}

func githubPublicRecords(read *vision.PageReadResult, gh githubPublicSource) []map[string]any {
	return []map[string]any{{
		"schema":       "uiai.source_markdown_record.v1",
		"source_type":  gh.SourceType,
		"record_type":  gh.Kind,
		"index":        1,
		"repo":         gh.Owner + "/" + gh.Repo,
		gh.FieldName:   gh.Number,
		"title":        githubPublicTitle(read, gh),
		"url":          gh.URL,
		"evidence_ref": markdownEvidenceRef(read.URL, read.Text) + "#record=1",
	}}
}

type redditPublicSource struct {
	Subreddit string
	PostID    string
	Slug      string
	CommentID string
	URL       string
}

func matchRedditPublicSource(raw string) (redditPublicSource, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return redditPublicSource{}, false
	}
	host := strings.ToLower(u.Hostname())
	if host != "reddit.com" && host != "www.reddit.com" && host != "old.reddit.com" && host != "new.reddit.com" {
		return redditPublicSource{}, false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 || strings.ToLower(parts[0]) != "r" || parts[1] == "" || strings.ToLower(parts[2]) != "comments" || parts[3] == "" {
		return redditPublicSource{}, false
	}
	rd := redditPublicSource{Subreddit: parts[1], PostID: parts[3], URL: sanitizeMarkdownURLForFocusa(raw)}
	if len(parts) >= 5 {
		rd.Slug = parts[4]
	}
	if len(parts) >= 6 {
		rd.CommentID = parts[5]
	}
	return rd, true
}

func applyRedditPublicMetadata(metadata map[string]any, rd redditPublicSource) map[string]any {
	out := make(map[string]any, len(metadata)+8)
	for k, v := range metadata {
		out[k] = v
	}
	out["adapter"] = "reddit_public"
	out["source_type"] = "reddit_thread"
	out["subreddit"] = "r/" + rd.Subreddit
	out["post_id"] = rd.PostID
	out["canonical_url"] = rd.URL
	if rd.CommentID != "" {
		out["comment_id"] = rd.CommentID
	}
	return out
}

func renderRedditPublicMarkdown(markdown string, read *vision.PageReadResult, rd redditPublicSource) string {
	title := redditPublicTitle(read, rd)
	frontmatter := []string{
		"---",
		"source: reddit_thread",
		"adapter: reddit_public",
		"subreddit: r/" + rd.Subreddit,
		"post_id: " + rd.PostID,
		"url: " + rd.URL,
		"evidence_ref: " + markdownEvidenceRef(read.URL, markdown),
		"---",
		"",
		"# " + title,
		"",
	}
	trimmed := strings.TrimSpace(markdown)
	if trimmed == "" {
		trimmed = "No readable Reddit content extracted. Inspect diagnostics or retry with old.reddit.com."
	}
	if !strings.Contains(strings.ToLower(trimmed), "comment") {
		trimmed += "\n\n## Top comments\n\nComment extraction is browser-rendered and bounded; inspect records/diagnostics for capture details."
	}
	return strings.Join(frontmatter, "\n") + trimmed
}

func redditPublicTitle(read *vision.PageReadResult, rd redditPublicSource) string {
	title := strings.TrimSpace(read.Title)
	if title != "" {
		return title
	}
	if rd.Slug != "" {
		return strings.ReplaceAll(rd.Slug, "_", " ")
	}
	return "Reddit thread r/" + rd.Subreddit + " " + rd.PostID
}

func redditPublicRecords(read *vision.PageReadResult, rd redditPublicSource) []map[string]any {
	record := map[string]any{
		"schema":       "uiai.source_markdown_record.v1",
		"source_type":  "reddit_thread",
		"record_type":  "post",
		"index":        1,
		"subreddit":    "r/" + rd.Subreddit,
		"post_id":      rd.PostID,
		"title":        redditPublicTitle(read, rd),
		"url":          rd.URL,
		"evidence_ref": markdownEvidenceRef(read.URL, read.Text) + "#record=1",
	}
	if rd.CommentID != "" {
		record["comment_id"] = rd.CommentID
		record["record_type"] = "comment_thread"
	}
	return []map[string]any{record}
}

func markdownFailure(class string, err error, url string, started time.Time) map[string]any {
	return map[string]any{
		"schema":      "uiai.source_markdown.v1",
		"error":       class,
		"message":     err.Error(),
		"url":         sanitizeMarkdownURLForFocusa(url),
		"duration_ms": time.Since(started).Milliseconds(),
		"diagnostics": map[string]any{"failure_class": class},
	}
}

func truthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func firstMarkdownNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func sanitizeMarkdownURLForFocusa(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return focusapacket.Truncate(raw, 220)
	}
	u.Fragment = ""
	q := u.Query()
	keys := make([]string, 0, len(q))
	for key := range q {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	redacted := url.Values{}
	for _, key := range keys {
		value := q.Get(key)
		lower := strings.ToLower(key)
		if strings.Contains(lower, "key") || strings.Contains(lower, "token") || strings.Contains(lower, "secret") || strings.Contains(lower, "password") || strings.Contains(lower, "auth") || strings.Contains(lower, "sig") || strings.Contains(lower, "session") {
			value = "REDACTED"
		}
		redacted.Set(key, value)
	}
	u.RawQuery = redacted.Encode()
	return focusapacket.Truncate(u.String(), 220)
}
