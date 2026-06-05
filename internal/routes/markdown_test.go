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
