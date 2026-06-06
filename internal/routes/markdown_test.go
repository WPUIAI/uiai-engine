package routes

import (
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/vision"
)

func TestMatchGitHubPublicSource(t *testing.T) {
	tests := []struct {
		url        string
		sourceType string
		fieldName  string
		number     string
	}{
		{"https://github.com/WPUIAI/uiai-engine/issues/123?token=secret#frag", "github_issue", "issue", "123"},
		{"https://github.com/WPUIAI/uiai-engine/pull/45", "github_pull_request", "pull_request", "45"},
		{"https://github.com/WPUIAI/uiai-engine/discussions/7", "github_discussion", "discussion", "7"},
	}
	for _, tc := range tests {
		gh, ok := matchGitHubPublicSource(tc.url)
		if !ok {
			t.Fatalf("expected match for %s", tc.url)
		}
		if gh.Owner != "WPUIAI" || gh.Repo != "uiai-engine" || gh.SourceType != tc.sourceType || gh.FieldName != tc.fieldName || gh.Number != tc.number {
			t.Fatalf("bad github source: %+v", gh)
		}
		if strings.Contains(gh.URL, "secret") || strings.Contains(gh.URL, "#frag") {
			t.Fatalf("url not sanitized: %s", gh.URL)
		}
	}
}

func TestMatchGitHubPublicSourceRejectsNonTargets(t *testing.T) {
	for _, raw := range []string{
		"https://example.com/WPUIAI/uiai-engine/issues/1",
		"https://github.com/WPUIAI/uiai-engine/blob/main/README.md",
		"https://github.com/WPUIAI/uiai-engine",
	} {
		if gh, ok := matchGitHubPublicSource(raw); ok {
			t.Fatalf("unexpected match for %s: %+v", raw, gh)
		}
	}
}

func TestRenderGitHubPublicMarkdown(t *testing.T) {
	gh, ok := matchGitHubPublicSource("https://github.com/WPUIAI/uiai-engine/issues/123?token=secret#frag")
	if !ok {
		t.Fatal("expected github match")
	}
	read := &vision.PageReadResult{URL: "https://github.com/WPUIAI/uiai-engine/issues/123", Title: "Example issue · WPUIAI/uiai-engine"}
	out := renderGitHubPublicMarkdown("## Description\n\nIssue body", read, gh)
	for _, want := range []string{
		"source: github_issue",
		"adapter: github_public",
		"repo: WPUIAI/uiai-engine",
		"issue: 123",
		"evidence_ref: uiai-source-markdown:sha256:",
		"# Example issue · WPUIAI/uiai-engine",
		"## Description",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in markdown:\n%s", want, out)
		}
	}
	if strings.Contains(out, "secret") || strings.Contains(out, "#frag") {
		t.Fatalf("secret leaked in markdown:\n%s", out)
	}
}

func TestApplyGitHubPublicMetadata(t *testing.T) {
	gh, ok := matchGitHubPublicSource("https://github.com/WPUIAI/uiai-engine/pull/45")
	if !ok {
		t.Fatal("expected github match")
	}
	meta := applyGitHubPublicMetadata(map[string]any{"schema": "uiai.source_markdown.v1"}, gh)
	if meta["adapter"] != "github_public" || meta["source_type"] != "github_pull_request" || meta["repo"] != "WPUIAI/uiai-engine" || meta["pull_request"] != "45" {
		t.Fatalf("bad metadata: %+v", meta)
	}
}

func TestGitHubPublicRecords(t *testing.T) {
	gh, ok := matchGitHubPublicSource("https://github.com/WPUIAI/uiai-engine/discussions/7")
	if !ok {
		t.Fatal("expected github match")
	}
	read := &vision.PageReadResult{URL: "https://github.com/WPUIAI/uiai-engine/discussions/7", Title: "Example discussion", Text: "# Example discussion"}
	records := githubPublicRecords(read, gh)
	if len(records) != 1 {
		t.Fatalf("record count = %d", len(records))
	}
	rec := records[0]
	if rec["schema"] != "uiai.source_markdown_record.v1" || rec["source_type"] != "github_discussion" || rec["record_type"] != "discussions" || rec["discussion"] != "7" {
		t.Fatalf("bad record: %+v", rec)
	}
	if !strings.Contains(rec["evidence_ref"].(string), "#record=1") {
		t.Fatalf("missing record suffix: %+v", rec)
	}
}

func TestSourceMarkdownJSONLAndChunks(t *testing.T) {
	records := []map[string]any{
		{"schema": "uiai.source_markdown_record.v1", "source_type": "github_issue", "record_type": "issue", "evidence_ref": "uiai-source-markdown:sha256:abc#record=1"},
		{"schema": "uiai.source_markdown_record.v1", "source_type": "github_issue", "record_type": "comment", "evidence_ref": "uiai-source-markdown:sha256:abc#record=2"},
	}
	jsonl, chunks := sourceMarkdownJSONL(records, "uiai-source-markdown:sha256:abc")
	if strings.Count(jsonl, "\n") != 1 || !strings.Contains(jsonl, "\"chunk_index\":1") || !strings.Contains(jsonl, "\"parent_evidence_ref\":\"uiai-source-markdown:sha256:abc\"") {
		t.Fatalf("bad jsonl: %s", jsonl)
	}
	if len(chunks) != 2 || chunks[0]["schema"] != "uiai.source_markdown_chunk.v1" || chunks[1]["index"] != 2 {
		t.Fatalf("bad chunks: %+v", chunks)
	}
}

func TestNormalizeSourceMarkdownFormat(t *testing.T) {
	if normalizeSourceMarkdownFormat("jsonl") != "jsonl" || normalizeSourceMarkdownFormat("md") != "markdown" || normalizeSourceMarkdownFormat("bogus") != "json" {
		t.Fatalf("unexpected format normalization")
	}
}

func TestMatchRedditPublicSource(t *testing.T) {
	tests := []struct {
		url       string
		subreddit string
		postID    string
		slug      string
		commentID string
	}{
		{"https://www.reddit.com/r/example/comments/abc123/example_title/?token=secret#frag", "example", "abc123", "example_title", ""},
		{"https://old.reddit.com/r/example/comments/abc123/example_title/def456/", "example", "abc123", "example_title", "def456"},
		{"https://new.reddit.com/r/UIAI/comments/xyz789/agent_markdown/", "UIAI", "xyz789", "agent_markdown", ""},
	}
	for _, tc := range tests {
		rd, ok := matchRedditPublicSource(tc.url)
		if !ok {
			t.Fatalf("expected reddit match for %s", tc.url)
		}
		if rd.Subreddit != tc.subreddit || rd.PostID != tc.postID || rd.Slug != tc.slug || rd.CommentID != tc.commentID {
			t.Fatalf("bad reddit source: %+v", rd)
		}
		if strings.Contains(rd.URL, "secret") || strings.Contains(rd.URL, "#frag") {
			t.Fatalf("url not sanitized: %s", rd.URL)
		}
	}
}

func TestMatchRedditPublicSourceRejectsNonTargets(t *testing.T) {
	for _, raw := range []string{
		"https://example.com/r/example/comments/abc123/title",
		"https://reddit.com/r/example/",
		"https://reddit.com/user/example/comments/abc123/title",
	} {
		if rd, ok := matchRedditPublicSource(raw); ok {
			t.Fatalf("unexpected reddit match for %s: %+v", raw, rd)
		}
	}
}

func TestRenderRedditPublicMarkdown(t *testing.T) {
	rd, ok := matchRedditPublicSource("https://old.reddit.com/r/example/comments/abc123/example_title/def456/?token=secret#frag")
	if !ok {
		t.Fatal("expected reddit match")
	}
	read := &vision.PageReadResult{URL: "https://old.reddit.com/r/example/comments/abc123/example_title/def456/", Title: "Example title : r/example"}
	out := renderRedditPublicMarkdown("Post body", read, rd)
	for _, want := range []string{
		"source: reddit_thread",
		"adapter: reddit_public",
		"subreddit: r/example",
		"post_id: abc123",
		"evidence_ref: uiai-source-markdown:sha256:",
		"# Example title : r/example",
		"## Top comments",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in markdown:\n%s", want, out)
		}
	}
	if strings.Contains(out, "secret") || strings.Contains(out, "#frag") {
		t.Fatalf("secret leaked in markdown:\n%s", out)
	}
}

func TestApplyRedditPublicMetadataAndRecords(t *testing.T) {
	rd, ok := matchRedditPublicSource("https://reddit.com/r/example/comments/abc123/title/def456/")
	if !ok {
		t.Fatal("expected reddit match")
	}
	meta := applyRedditPublicMetadata(map[string]any{"schema": "uiai.source_markdown.v1"}, rd)
	if meta["adapter"] != "reddit_public" || meta["source_type"] != "reddit_thread" || meta["subreddit"] != "r/example" || meta["post_id"] != "abc123" || meta["comment_id"] != "def456" {
		t.Fatalf("bad metadata: %+v", meta)
	}
	read := &vision.PageReadResult{URL: "https://reddit.com/r/example/comments/abc123/title/def456/", Title: "Example", Text: "# Example"}
	records := redditPublicRecords(read, rd)
	if len(records) != 1 {
		t.Fatalf("record count = %d", len(records))
	}
	rec := records[0]
	if rec["schema"] != "uiai.source_markdown_record.v1" || rec["source_type"] != "reddit_thread" || rec["record_type"] != "comment_thread" || rec["comment_id"] != "def456" {
		t.Fatalf("bad reddit record: %+v", rec)
	}
}

func TestMatchHackerNewsPublicSource(t *testing.T) {
	hn, ok := matchHackerNewsPublicSource("https://news.ycombinator.com/item?id=12345&token=secret#frag")
	if !ok {
		t.Fatal("expected HN match")
	}
	if hn.ItemID != "12345" || strings.Contains(hn.URL, "secret") || strings.Contains(hn.URL, "#frag") {
		t.Fatalf("bad HN source: %+v", hn)
	}
	for _, raw := range []string{"https://news.ycombinator.com/news", "https://example.com/item?id=123"} {
		if got, ok := matchHackerNewsPublicSource(raw); ok {
			t.Fatalf("unexpected HN match for %s: %+v", raw, got)
		}
	}
}

func TestRenderHackerNewsPublicMarkdownAndRecords(t *testing.T) {
	hn, ok := matchHackerNewsPublicSource("https://news.ycombinator.com/item?id=12345")
	if !ok {
		t.Fatal("expected HN match")
	}
	read := &vision.PageReadResult{URL: "https://news.ycombinator.com/item?id=12345", Title: "Example story | Hacker News", Text: "Story body"}
	meta := applyHackerNewsPublicMetadata(map[string]any{"schema": "uiai.source_markdown.v1"}, hn)
	if meta["adapter"] != "hackernews_public" || meta["source_type"] != "hackernews_thread" || meta["item_id"] != "12345" {
		t.Fatalf("bad HN metadata: %+v", meta)
	}
	out := renderHackerNewsPublicMarkdown("Story body", read, hn)
	for _, want := range []string{"source: hackernews_thread", "adapter: hackernews_public", "item_id: 12345", "## Comments"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in markdown:\n%s", want, out)
		}
	}
	records := hackerNewsPublicRecords(read, hn)
	if len(records) != 1 || records[0]["source_type"] != "hackernews_thread" || records[0]["record_type"] != "thread" || records[0]["item_id"] != "12345" {
		t.Fatalf("bad HN records: %+v", records)
	}
}

func TestMatchYouTubePublicSource(t *testing.T) {
	tests := []struct {
		url string
		id  string
	}{
		{"https://www.youtube.com/watch?v=abc_123-XYZ&token=secret#frag", "abc_123-XYZ"},
		{"https://youtu.be/abc123", "abc123"},
		{"https://m.youtube.com/shorts/short_123", "short_123"},
		{"https://www.youtube.com/embed/embed-123", "embed-123"},
	}
	for _, tc := range tests {
		yt, ok := matchYouTubePublicSource(tc.url)
		if !ok || yt.VideoID != tc.id {
			t.Fatalf("bad YouTube match for %s: %+v ok=%v", tc.url, yt, ok)
		}
		if strings.Contains(yt.URL, "secret") || strings.Contains(yt.URL, "#frag") {
			t.Fatalf("url not sanitized: %s", yt.URL)
		}
	}
	for _, raw := range []string{"https://youtube.com/feed/subscriptions", "https://example.com/watch?v=abc"} {
		if got, ok := matchYouTubePublicSource(raw); ok {
			t.Fatalf("unexpected YouTube match for %s: %+v", raw, got)
		}
	}
}

func TestRenderYouTubePublicMarkdownAndRecords(t *testing.T) {
	yt, ok := matchYouTubePublicSource("https://www.youtube.com/watch?v=abc123")
	if !ok {
		t.Fatal("expected YouTube match")
	}
	read := &vision.PageReadResult{URL: "https://www.youtube.com/watch?v=abc123", Title: "Example video - YouTube", Text: "Description\nShow transcript\nhello world"}
	meta := applyYouTubePublicMetadata(map[string]any{"schema": "uiai.source_markdown.v1"}, yt, read.Text)
	if meta["adapter"] != "youtube_public" || meta["source_type"] != "youtube_video" || meta["video_id"] != "abc123" || meta["transcript_available"] != true {
		t.Fatalf("bad YouTube metadata: %+v", meta)
	}
	out := renderYouTubePublicMarkdown(read.Text, read, yt)
	for _, want := range []string{"source: youtube_video", "adapter: youtube_public", "video_id: abc123", "transcript_available: true", "## Transcript"} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in markdown:\n%s", want, out)
		}
	}
	records := youtubePublicRecords(read, yt)
	if len(records) != 1 || records[0]["source_type"] != "youtube_video" || records[0]["record_type"] != "video_metadata" || records[0]["video_id"] != "abc123" || records[0]["transcript_available"] != true {
		t.Fatalf("bad YouTube records: %+v", records)
	}
}

func TestYouTubePublicBlockedMetadata(t *testing.T) {
	yt, ok := matchYouTubePublicSource("https://youtu.be/abc123")
	if !ok {
		t.Fatal("expected YouTube match")
	}
	read := &vision.PageReadResult{URL: "https://youtu.be/abc123", Title: "Blocked", Text: "This video is unavailable"}
	meta := applyYouTubePublicMetadata(map[string]any{"schema": "uiai.source_markdown.v1"}, yt, read.Text)
	if meta["blocked"] != true || meta["failure_class"] != "auth_or_transcript_unavailable" {
		t.Fatalf("blocked metadata missing: %+v", meta)
	}
	out := renderYouTubePublicMarkdown(read.Text, read, yt)
	if !strings.Contains(out, "Do not scrape credentials") || !strings.Contains(out, "blocked: true") {
		t.Fatalf("blocked guidance missing:\n%s", out)
	}
}

func TestWPUIAISourceMarkdownProductization(t *testing.T) {
	read := &vision.PageReadResult{URL: "https://example.com/source", Title: "Example Source", Text: strings.Repeat("research ", 300)}
	metadata := map[string]any{"source_type": "reddit_thread", "captured_at": "2026-06-05T00:00:00Z"}
	records := []map[string]any{{"evidence_ref": "uiai-source-markdown:sha256:abc#record=1"}}
	out := wpuiAISourceMarkdownProductization(read, metadata, records, "uiai-source-markdown:sha256:abc", "https://example.test/thread")
	card := out["research_card"].(map[string]any)
	if card["schema"] != "wpui.source_markdown_research_card.v1" || card["source_type"] != "reddit_thread" || card["evidence_ref"] != "uiai-source-markdown:sha256:abc" || card["record_count"] != 1 {
		t.Fatalf("bad research card: %+v", card)
	}
	if len(card["markdown_excerpt"].(string)) > 1205 {
		t.Fatalf("excerpt not bounded: %d", len(card["markdown_excerpt"].(string)))
	}
	uses := card["suggested_uses"].([]string)
	if len(uses) == 0 || uses[0] != "market research" {
		t.Fatalf("bad uses: %+v", uses)
	}
	report := out["report"].(map[string]any)
	if report["schema"] != "wpui.source_markdown_report.v1" || report["record_count"] != 1 {
		t.Fatalf("bad report: %+v", report)
	}
}

func TestMatchXPublicSource(t *testing.T) {
	tests := []struct {
		url       string
		author    string
		kind      string
		statusID  string
		articleID string
		source    string
	}{
		{"https://x.com/example/status/12345?token=secret#frag", "example", "status", "12345", "", "x_thread"},
		{"https://twitter.com/example/statuses/67890", "example", "status", "67890", "", "x_thread"},
		{"https://x.com/example/articles/999", "example", "article", "", "999", "x_article"},
	}
	for _, tc := range tests {
		xp, ok := matchXPublicSource(tc.url)
		if !ok {
			t.Fatalf("expected x match for %s", tc.url)
		}
		if xp.Author != tc.author || xp.Kind != tc.kind || xp.StatusID != tc.statusID || xp.ArticleID != tc.articleID || xp.SourceType != tc.source {
			t.Fatalf("bad x source: %+v", xp)
		}
		if strings.Contains(xp.URL, "secret") || strings.Contains(xp.URL, "#frag") {
			t.Fatalf("url not sanitized: %s", xp.URL)
		}
	}
}

func TestMatchXPublicSourceRejectsNonTargets(t *testing.T) {
	for _, raw := range []string{
		"https://example.com/user/status/123",
		"https://x.com/user",
		"https://x.com/user/likes",
	} {
		if xp, ok := matchXPublicSource(raw); ok {
			t.Fatalf("unexpected x match for %s: %+v", raw, xp)
		}
	}
}

func TestRenderXPublicMarkdownBlocked(t *testing.T) {
	xp, ok := matchXPublicSource("https://x.com/example/status/12345?token=secret#frag")
	if !ok {
		t.Fatal("expected x match")
	}
	read := &vision.PageReadResult{URL: "https://x.com/example/status/12345", Title: "Example on X"}
	out := renderXPublicMarkdown("Log in to view this post", read, xp)
	for _, want := range []string{
		"source: x_thread",
		"adapter: x_public",
		"author: @example",
		"status_id: 12345",
		"best_effort: true",
		"blocked: true",
		"failure_class: auth_required",
		"Source blocked or authentication required",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("missing %q in markdown:\n%s", want, out)
		}
	}
	if strings.Contains(out, "secret") || strings.Contains(out, "#frag") {
		t.Fatalf("secret leaked in markdown:\n%s", out)
	}
}

func TestApplyXPublicMetadataAndRecords(t *testing.T) {
	xp, ok := matchXPublicSource("https://x.com/example/articles/999")
	if !ok {
		t.Fatal("expected x match")
	}
	meta := applyXPublicMetadata(map[string]any{"schema": "uiai.source_markdown.v1"}, xp, "Something went wrong")
	if meta["adapter"] != "x_public" || meta["source_type"] != "x_article" || meta["author"] != "@example" || meta["article_id"] != "999" || meta["blocked"] != true || meta["failure_class"] != "auth_required" {
		t.Fatalf("bad metadata: %+v", meta)
	}
	read := &vision.PageReadResult{URL: "https://x.com/example/articles/999", Title: "X Article", Text: "Something went wrong"}
	records := xPublicRecords(read, xp)
	if len(records) != 1 {
		t.Fatalf("record count = %d", len(records))
	}
	rec := records[0]
	if rec["schema"] != "uiai.source_markdown_record.v1" || rec["source_type"] != "x_article" || rec["record_type"] != "article" || rec["article_id"] != "999" || rec["blocked"] != true {
		t.Fatalf("bad x record: %+v", rec)
	}
}

func TestSanitizeMarkdownPayloadRedactsLinksMetadataAndText(t *testing.T) {
	payload := map[string]any{
		"canonical_url": "https://example.test/page?token=SECRET123&ok=1#frag",
		"links": []map[string]any{{
			"href": "https://example.test/next?api_key=BAD&token=SECRET123#frag",
			"text": "secret link",
		}},
		"markdown": "[secret link](https://example.test/next?api_key=BAD&token=SECRET123#frag) cookie=YUM",
	}
	got := sanitizeMarkdownPayload(payload).(map[string]any)
	text := got["markdown"].(string)
	links := got["links"].([]map[string]any)
	combined := got["canonical_url"].(string) + "\n" + links[0]["href"].(string) + "\n" + text
	for _, leaked := range []string{"SECRET123", "api_key=BAD", "#frag", "cookie=YUM"} {
		if strings.Contains(combined, leaked) {
			t.Fatalf("leaked %q in sanitized payload: %+v", leaked, got)
		}
	}
	if !strings.Contains(combined, "token=REDACTED") || !strings.Contains(combined, "api_key=REDACTED") || !strings.Contains(combined, "cookie=REDACTED") {
		t.Fatalf("missing redaction markers: %+v", got)
	}
}
