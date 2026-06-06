package routes

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/focusapacket"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

var markdownURLRe = regexp.MustCompile(`https?://[^\s)\]"'<>]+`)

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
	Format        string              `json:"format,omitempty"`
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
		Format:        q.Get("format"),
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
	diagnostics := sanitizedMarkdownDiagnosticsSnapshot(sess.Diagnostics(25, "", false))
	if err != nil {
		_ = sm.Close(sess.ID)
		writeJSON(w, http.StatusBadGateway, markdownFailure("source_read_failed", err, body.URL, started))
		return
	}
	if err := sm.Close(sess.ID); err == nil {
		cleanup["closed"] = true
	}

	capturedAt := time.Now().UTC().Format(time.RFC3339)
	metadata := sanitizeMarkdownPayload(markdownMetadata(read, body, capturedAt)).(map[string]any)
	sourceMatchURL := firstMarkdownNonEmpty(body.URL, read.URL)
	var records []map[string]any
	if gh, ok := matchGitHubPublicSource(sourceMatchURL); ok {
		metadata = applyGitHubPublicMetadata(metadata, gh)
		read.Text = renderGitHubPublicMarkdown(read.Text, read, gh)
		read.Chars = len(read.Text)
		records = githubPublicRecords(read, gh)
		metadata["record_count"] = len(records)
	} else if rd, ok := matchRedditPublicSource(sourceMatchURL); ok {
		metadata = applyRedditPublicMetadata(metadata, rd)
		read.Text = renderRedditPublicMarkdown(read.Text, read, rd)
		read.Chars = len(read.Text)
		records = redditPublicRecords(read, rd)
		metadata["record_count"] = len(records)
	} else if hn, ok := matchHackerNewsPublicSource(sourceMatchURL); ok {
		metadata = applyHackerNewsPublicMetadata(metadata, hn)
		read.Text = renderHackerNewsPublicMarkdown(read.Text, read, hn)
		read.Chars = len(read.Text)
		records = hackerNewsPublicRecords(read, hn)
		metadata["record_count"] = len(records)
	} else if yt, ok := matchYouTubePublicSource(sourceMatchURL); ok {
		metadata = applyYouTubePublicMetadata(metadata, yt, read.Text)
		read.Text = renderYouTubePublicMarkdown(read.Text, read, yt)
		read.Chars = len(read.Text)
		records = youtubePublicRecords(read, yt)
		metadata["record_count"] = len(records)
	} else if xp, ok := matchXPublicSource(sourceMatchURL); ok {
		metadata = applyXPublicMetadata(metadata, xp, read.Text)
		read.Text = renderXPublicMarkdown(read.Text, read, xp)
		read.Chars = len(read.Text)
		records = xPublicRecords(read, xp)
		metadata["record_count"] = len(records)
	}
	markdown := sanitizeMarkdownText(read.Text)
	read.Text = markdown
	read.Links = sanitizeMarkdownLinks(read.Links)
	read.Metadata = sanitizeMarkdownPayload(read.Metadata).(map[string]any)
	evidenceRef := markdownEvidenceRef(read.URL, markdown)
	safeURL := sanitizeMarkdownURLForFocusa(firstMarkdownNonEmpty(read.URL, body.URL))
	format := normalizeSourceMarkdownFormat(body.Format)
	jsonl, chunks := sourceMarkdownJSONL(records, evidenceRef)
	if format == "jsonl" {
		metadata["output_format"] = "jsonl"
	}
	if len(chunks) > 0 {
		metadata["chunk_count"] = len(chunks)
	}
	wpuiAI := wpuiAISourceMarkdownProductization(read, metadata, records, evidenceRef, safeURL)
	response := map[string]any{
		"schema":       "uiai.source_markdown.v1",
		"url":          safeURL,
		"title":        read.Title,
		"description":  read.Description,
		"format":       format,
		"markdown":     markdown,
		"text":         markdown,
		"jsonl":        jsonl,
		"chars":        read.Chars,
		"truncated":    read.Truncated,
		"headings":     read.Headings,
		"links":        sanitizeMarkdownLinks(read.Links),
		"metadata":     metadata,
		"diagnostics":  diagnostics,
		"focusa":       oneShotMarkdownFocusa(read, body.URL),
		"wpuiai":       wpuiAI,
		"cleanup":      cleanup,
		"duration_ms":  time.Since(started).Milliseconds(),
		"source_stats": map[string]any{"format": format, "mode": read.Mode, "max_chars": body.MaxChars, "adapter": metadata["adapter"], "record_count": len(records), "chunk_count": len(chunks)},
	}
	if len(records) > 0 {
		response["records"] = records
	}
	if len(chunks) > 0 {
		response["chunks"] = chunks
	}
	writeJSON(w, http.StatusOK, response)
}

func normalizeSourceMarkdownFormat(format string) string {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "jsonl":
		return "jsonl"
	case "json", "":
		return "json"
	case "markdown", "md":
		return "markdown"
	default:
		return "json"
	}
}

func sourceMarkdownJSONL(records []map[string]any, evidenceRef string) (string, []map[string]any) {
	if len(records) == 0 {
		return "", nil
	}
	lines := make([]string, 0, len(records))
	chunks := make([]map[string]any, 0, len(records))
	for i, record := range records {
		copyRecord := make(map[string]any, len(record)+4)
		for k, v := range record {
			copyRecord[k] = v
		}
		copyRecord["chunk_index"] = i + 1
		copyRecord["chunk_count"] = len(records)
		copyRecord["parent_evidence_ref"] = evidenceRef
		if _, ok := copyRecord["schema"]; !ok {
			copyRecord["schema"] = "uiai.source_markdown_record.v1"
		}
		if _, ok := copyRecord["evidence_ref"]; !ok {
			copyRecord["evidence_ref"] = fmt.Sprintf("%s#record=%d", evidenceRef, i+1)
		}
		encoded, err := json.Marshal(copyRecord)
		if err != nil {
			continue
		}
		lines = append(lines, string(encoded))
		chunks = append(chunks, map[string]any{
			"schema":       "uiai.source_markdown_chunk.v1",
			"index":        i + 1,
			"count":        len(records),
			"evidence_ref": copyRecord["evidence_ref"],
			"record_type":  copyRecord["record_type"],
			"source_type":  copyRecord["source_type"],
		})
	}
	return strings.Join(lines, "\n"), chunks
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
	title := focusapacket.Truncate(firstMarkdownNonEmpty(sanitizeMarkdownText(read.Title), sanitizeMarkdownURLForFocusa(read.URL), sanitizeMarkdownURLForFocusa(requestedURL)), 160)
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

func wpuiAISourceMarkdownProductization(read *vision.PageReadResult, metadata map[string]any, records []map[string]any, evidenceRef string, safeSourceURL string) map[string]any {
	sourceType := stringFromMap(metadata, "source_type", "webpage")
	capturedAt := stringFromMap(metadata, "captured_at", time.Now().UTC().Format(time.RFC3339))
	excerpt := focusapacket.Truncate(strings.TrimSpace(read.Text), 1200)
	card := map[string]any{
		"schema":           "wpui.source_markdown_research_card.v1",
		"source_url":       safeSourceURL,
		"source_type":      sourceType,
		"title":            firstMarkdownNonEmpty(sanitizeMarkdownText(read.Title), safeSourceURL),
		"markdown_excerpt": excerpt,
		"evidence_ref":     evidenceRef,
		"captured_at":      capturedAt,
		"suggested_uses":   suggestedWPUIAIUses(sourceType),
		"metadata":         metadata,
	}
	if len(records) > 0 {
		card["record_count"] = len(records)
		card["record_refs"] = sourceMarkdownRecordRefs(records)
	}
	return map[string]any{
		"research_card": card,
		"report": map[string]any{
			"schema":       "wpui.source_markdown_report.v1",
			"summary":      fmt.Sprintf("Captured %s as Markdown for WPUIAI research/card workflows.", firstMarkdownNonEmpty(read.Title, sourceType)),
			"evidence_ref": evidenceRef,
			"source_url":   safeSourceURL,
			"source_type":  sourceType,
			"record_count": len(records),
		},
	}
}

func suggestedWPUIAIUses(sourceType string) []string {
	switch sourceType {
	case "github_issue", "github_pull_request", "github_discussion":
		return []string{"developer proof", "feature requirement", "release note source", "support reference"}
	case "reddit_thread", "x_thread", "x_article":
		return []string{"market research", "voice-of-customer", "FAQ seed", "competitor proof"}
	default:
		return []string{"content source", "competitor proof", "reference", "FAQ seed", "SEO seed"}
	}
}

func sourceMarkdownRecordRefs(records []map[string]any) []string {
	refs := make([]string, 0, len(records))
	for _, record := range records {
		if ref := stringFromMap(record, "evidence_ref", ""); ref != "" {
			refs = append(refs, ref)
		}
	}
	return refs
}

func stringFromMap(values map[string]any, key, fallback string) string {
	if value, ok := values[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
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

type hackerNewsPublicSource struct {
	ItemID string
	URL    string
}

func matchHackerNewsPublicSource(raw string) (hackerNewsPublicSource, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return hackerNewsPublicSource{}, false
	}
	host := strings.ToLower(u.Hostname())
	if host != "news.ycombinator.com" && host != "www.news.ycombinator.com" {
		return hackerNewsPublicSource{}, false
	}
	if strings.Trim(u.Path, "/") != "item" {
		return hackerNewsPublicSource{}, false
	}
	itemID := strings.TrimSpace(u.Query().Get("id"))
	if itemID == "" {
		return hackerNewsPublicSource{}, false
	}
	return hackerNewsPublicSource{ItemID: itemID, URL: sanitizeMarkdownURLForFocusa(raw)}, true
}

func applyHackerNewsPublicMetadata(metadata map[string]any, hn hackerNewsPublicSource) map[string]any {
	out := make(map[string]any, len(metadata)+6)
	for k, v := range metadata {
		out[k] = v
	}
	out["adapter"] = "hackernews_public"
	out["source_type"] = "hackernews_thread"
	out["item_id"] = hn.ItemID
	out["canonical_url"] = hn.URL
	out["best_effort"] = true
	return out
}

func renderHackerNewsPublicMarkdown(markdown string, read *vision.PageReadResult, hn hackerNewsPublicSource) string {
	title := hackerNewsPublicTitle(read, hn)
	frontmatter := []string{
		"---",
		"source: hackernews_thread",
		"adapter: hackernews_public",
		"item_id: " + hn.ItemID,
		"url: " + hn.URL,
		"best_effort: true",
		"evidence_ref: " + markdownEvidenceRef(read.URL, markdown),
		"---",
		"",
		"# " + title,
		"",
	}
	trimmed := strings.TrimSpace(markdown)
	if trimmed == "" {
		trimmed = "No readable Hacker News content extracted. Inspect diagnostics or retry with a browser session."
	}
	if !strings.Contains(strings.ToLower(trimmed), "comment") {
		trimmed += "\n\n## Comments\n\nComment extraction is browser-rendered and bounded; inspect records/diagnostics for capture details."
	}
	return strings.Join(frontmatter, "\n") + trimmed
}

func hackerNewsPublicTitle(read *vision.PageReadResult, hn hackerNewsPublicSource) string {
	title := strings.TrimSpace(read.Title)
	if title != "" {
		return title
	}
	return "Hacker News item " + hn.ItemID
}

func hackerNewsPublicRecords(read *vision.PageReadResult, hn hackerNewsPublicSource) []map[string]any {
	return []map[string]any{{
		"schema":       "uiai.source_markdown_record.v1",
		"source_type":  "hackernews_thread",
		"record_type":  "thread",
		"index":        1,
		"item_id":      hn.ItemID,
		"title":        hackerNewsPublicTitle(read, hn),
		"url":          hn.URL,
		"evidence_ref": markdownEvidenceRef(read.URL, read.Text) + "#record=1",
	}}
}

type youtubePublicSource struct {
	VideoID string
	URL     string
}

func matchYouTubePublicSource(raw string) (youtubePublicSource, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return youtubePublicSource{}, false
	}
	host := strings.ToLower(u.Hostname())
	var videoID string
	if host == "youtu.be" || host == "www.youtu.be" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if len(parts) >= 1 {
			videoID = parts[0]
		}
	} else if host == "youtube.com" || host == "www.youtube.com" || host == "m.youtube.com" {
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if strings.Trim(u.Path, "/") == "watch" {
			videoID = u.Query().Get("v")
		} else if len(parts) >= 2 && (parts[0] == "shorts" || parts[0] == "embed" || parts[0] == "live") {
			videoID = parts[1]
		}
	}
	videoID = sanitizeYouTubeVideoID(videoID)
	if videoID == "" {
		return youtubePublicSource{}, false
	}
	return youtubePublicSource{VideoID: videoID, URL: sanitizeMarkdownURLForFocusa(raw)}, true
}

func sanitizeYouTubeVideoID(videoID string) string {
	videoID = strings.TrimSpace(videoID)
	if videoID == "" {
		return ""
	}
	var out strings.Builder
	for _, r := range videoID {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			out.WriteRune(r)
		}
	}
	return out.String()
}

func applyYouTubePublicMetadata(metadata map[string]any, yt youtubePublicSource, markdown string) map[string]any {
	out := make(map[string]any, len(metadata)+10)
	for k, v := range metadata {
		out[k] = v
	}
	out["adapter"] = "youtube_public"
	out["source_type"] = "youtube_video"
	out["video_id"] = yt.VideoID
	out["canonical_url"] = yt.URL
	out["best_effort"] = true
	out["transcript_available"] = youtubePublicTranscriptAvailable(markdown)
	if youtubePublicLooksBlocked(markdown) {
		out["blocked"] = true
		out["failure_class"] = "auth_or_transcript_unavailable"
		out["recommended_next_action"] = "Capture available metadata as evidence, or retry only with a public transcript/authorized browser session when policy allows."
	}
	return out
}

func renderYouTubePublicMarkdown(markdown string, read *vision.PageReadResult, yt youtubePublicSource) string {
	title := youtubePublicTitle(read, yt)
	transcriptAvailable := youtubePublicTranscriptAvailable(markdown)
	blocked := youtubePublicLooksBlocked(markdown)
	frontmatter := []string{
		"---",
		"source: youtube_video",
		"adapter: youtube_public",
		"video_id: " + yt.VideoID,
		"url: " + yt.URL,
		"best_effort: true",
		"transcript_available: " + strconv.FormatBool(transcriptAvailable),
		"evidence_ref: " + markdownEvidenceRef(read.URL, markdown),
	}
	if blocked {
		frontmatter = append(frontmatter, "blocked: true", "failure_class: auth_or_transcript_unavailable")
	}
	frontmatter = append(frontmatter, "---", "", "# "+title, "")
	trimmed := strings.TrimSpace(markdown)
	if trimmed == "" || blocked {
		trimmed = "Public YouTube transcript or metadata was unavailable from the browser-rendered page.\n\n## Next options\n\n- Capture this metadata-only result as blocked evidence.\n- Retry only when a public transcript is visible or an authorized browser session is already available.\n- Do not scrape credentials or bypass access controls."
	} else if transcriptAvailable && !strings.Contains(strings.ToLower(trimmed), "## transcript") {
		trimmed += "\n\n## Transcript\n\nTranscript-like text was detected in the rendered page. Extraction is best-effort and bounded; verify against the public page before treating it as complete."
	} else if !transcriptAvailable {
		trimmed += "\n\n## Transcript\n\nNo public transcript text was detected in the bounded browser read."
	}
	return strings.Join(frontmatter, "\n") + trimmed
}

func youtubePublicTitle(read *vision.PageReadResult, yt youtubePublicSource) string {
	title := strings.TrimSpace(read.Title)
	if title != "" {
		return title
	}
	return "YouTube video " + yt.VideoID
}

func youtubePublicTranscriptAvailable(markdown string) bool {
	lower := strings.ToLower(markdown)
	if strings.Contains(lower, "no public transcript") || strings.Contains(lower, "no transcript") || strings.Contains(lower, "transcript unavailable") {
		return false
	}
	return strings.Contains(lower, "transcript") || strings.Contains(lower, "show transcript")
}

func youtubePublicLooksBlocked(markdown string) bool {
	lower := strings.ToLower(markdown)
	blockedMarkers := []string{"sign in to confirm", "this video is unavailable", "video unavailable", "private video", "members only", "age-restricted", "transcript unavailable", "no transcript"}
	for _, marker := range blockedMarkers {
		if strings.Contains(lower, marker) {
			return true
		}
	}
	return false
}

func youtubePublicRecords(read *vision.PageReadResult, yt youtubePublicSource) []map[string]any {
	record := map[string]any{
		"schema":               "uiai.source_markdown_record.v1",
		"source_type":          "youtube_video",
		"record_type":          "video_metadata",
		"index":                1,
		"video_id":             yt.VideoID,
		"title":                youtubePublicTitle(read, yt),
		"url":                  yt.URL,
		"transcript_available": youtubePublicTranscriptAvailable(read.Text),
		"evidence_ref":         markdownEvidenceRef(read.URL, read.Text) + "#record=1",
	}
	if youtubePublicLooksBlocked(read.Text) {
		record["blocked"] = true
		record["failure_class"] = "auth_or_transcript_unavailable"
	}
	return []map[string]any{record}
}

type xPublicSource struct {
	Author     string
	Kind       string
	StatusID   string
	ArticleID  string
	URL        string
	SourceType string
}

func matchXPublicSource(raw string) (xPublicSource, bool) {
	u, err := url.Parse(raw)
	if err != nil {
		return xPublicSource{}, false
	}
	host := strings.ToLower(u.Hostname())
	if host != "x.com" && host != "www.x.com" && host != "twitter.com" && host != "www.twitter.com" {
		return xPublicSource{}, false
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 1 || parts[0] == "" {
		return xPublicSource{}, false
	}
	xp := xPublicSource{Author: parts[0], URL: sanitizeMarkdownURLForFocusa(raw), SourceType: "x_thread"}
	if len(parts) >= 3 && (parts[1] == "status" || parts[1] == "statuses") && parts[2] != "" {
		xp.Kind = "status"
		xp.StatusID = parts[2]
		return xp, true
	}
	if len(parts) >= 3 && parts[1] == "articles" && parts[2] != "" {
		xp.Kind = "article"
		xp.ArticleID = parts[2]
		xp.SourceType = "x_article"
		return xp, true
	}
	return xPublicSource{}, false
}

func applyXPublicMetadata(metadata map[string]any, xp xPublicSource, markdown string) map[string]any {
	out := make(map[string]any, len(metadata)+10)
	for k, v := range metadata {
		out[k] = v
	}
	out["adapter"] = "x_public"
	out["source_type"] = xp.SourceType
	out["author"] = "@" + xp.Author
	out["canonical_url"] = xp.URL
	out["best_effort"] = true
	if xp.StatusID != "" {
		out["status_id"] = xp.StatusID
	}
	if xp.ArticleID != "" {
		out["article_id"] = xp.ArticleID
	}
	if xPublicLooksBlocked(markdown) {
		out["blocked"] = true
		out["failure_class"] = "auth_required"
		out["recommended_next_action"] = "Retry with a browser session carrying valid auth, try a public mirror, or capture the source as blocked evidence."
	}
	return out
}

func renderXPublicMarkdown(markdown string, read *vision.PageReadResult, xp xPublicSource) string {
	title := xPublicTitle(read, xp)
	blocked := xPublicLooksBlocked(markdown)
	frontmatter := []string{
		"---",
		"source: " + xp.SourceType,
		"adapter: x_public",
		"author: @" + xp.Author,
		"url: " + xp.URL,
		"best_effort: true",
		"evidence_ref: " + markdownEvidenceRef(read.URL, markdown),
	}
	if xp.StatusID != "" {
		frontmatter = append(frontmatter, "status_id: "+xp.StatusID)
	}
	if xp.ArticleID != "" {
		frontmatter = append(frontmatter, "article_id: "+xp.ArticleID)
	}
	if blocked {
		frontmatter = append(frontmatter, "blocked: true", "failure_class: auth_required")
	}
	frontmatter = append(frontmatter, "---", "", "# "+title, "")
	trimmed := strings.TrimSpace(markdown)
	if trimmed == "" || blocked {
		trimmed = "Source blocked or authentication required. X/Twitter often limits anonymous browser access.\n\n## Next options\n\n- Retry with an authenticated browser session when policy allows.\n- Use a public mirror/archive if available.\n- Capture this result as blocked source evidence with diagnostics."
	}
	return strings.Join(frontmatter, "\n") + trimmed
}

func xPublicTitle(read *vision.PageReadResult, xp xPublicSource) string {
	title := strings.TrimSpace(read.Title)
	if title != "" {
		return title
	}
	if xp.Kind == "article" {
		return "X Article by @" + xp.Author
	}
	return "X Thread by @" + xp.Author
}

func xPublicRecords(read *vision.PageReadResult, xp xPublicSource) []map[string]any {
	record := map[string]any{
		"schema":       "uiai.source_markdown_record.v1",
		"source_type":  xp.SourceType,
		"record_type":  xp.Kind,
		"index":        1,
		"author":       "@" + xp.Author,
		"title":        xPublicTitle(read, xp),
		"url":          xp.URL,
		"evidence_ref": markdownEvidenceRef(read.URL, read.Text) + "#record=1",
	}
	if xp.StatusID != "" {
		record["status_id"] = xp.StatusID
	}
	if xp.ArticleID != "" {
		record["article_id"] = xp.ArticleID
	}
	if xPublicLooksBlocked(read.Text) {
		record["blocked"] = true
		record["failure_class"] = "auth_required"
	}
	return []map[string]any{record}
}

func xPublicLooksBlocked(markdown string) bool {
	lower := strings.ToLower(markdown)
	blockedNeedles := []string{"log in", "sign in", "create account", "something went wrong", "rate limit", "account suspended", "this post is unavailable", "javascript is not available"}
	for _, needle := range blockedNeedles {
		if strings.Contains(lower, needle) {
			return true
		}
	}
	return strings.TrimSpace(markdown) == ""
}

func sanitizedMarkdownDiagnosticsSnapshot(value any) any {
	data, err := json.Marshal(value)
	if err != nil {
		return sanitizeMarkdownDiagnostics(value)
	}
	var decoded any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return sanitizeMarkdownDiagnostics(value)
	}
	return sanitizeMarkdownDiagnostics(decoded)
}

func sanitizeMarkdownDiagnostics(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = sanitizeMarkdownDiagnostics(item)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = sanitizeMarkdownDiagnostics(item)
		}
		return out
	case string:
		if strings.Contains(v, "://") {
			return sanitizeMarkdownURLForFocusa(v)
		}
		return redactMarkdownSecretFragments(v)
	default:
		return value
	}
}

func redactMarkdownSecretFragments(value string) string {
	return sanitizeMarkdownText(value)
}

func sanitizeMarkdownText(value string) string {
	if value == "" {
		return value
	}
	out := markdownURLRe.ReplaceAllStringFunc(value, sanitizeMarkdownURLForFocusa)
	out = regexp.MustCompile(`(?i)((?:api[_-]?key|access[_-]?token|refresh[_-]?token|token|secret|password|passwd|cookie|set-cookie)\s*[:=]\s*)[^\s,;&)\]]+`).ReplaceAllString(out, `${1}REDACTED`)
	return out
}

func sanitizeMarkdownLinks(links []map[string]any) []map[string]any {
	if len(links) == 0 {
		return links
	}
	out := make([]map[string]any, 0, len(links))
	for _, link := range links {
		copyLink := make(map[string]any, len(link))
		for key, value := range link {
			copyLink[key] = sanitizeMarkdownPayload(value)
		}
		out = append(out, copyLink)
	}
	return out
}

func sanitizeMarkdownPayload(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, item := range v {
			out[key] = sanitizeMarkdownPayload(item)
		}
		return out
	case []map[string]any:
		return sanitizeMarkdownLinks(v)
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = sanitizeMarkdownPayload(item)
		}
		return out
	case string:
		return sanitizeMarkdownText(v)
	default:
		return value
	}
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
