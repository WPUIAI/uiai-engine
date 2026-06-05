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
