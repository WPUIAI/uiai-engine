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
	markdown := read.Text
	writeJSON(w, http.StatusOK, map[string]any{
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
		"metadata":     markdownMetadata(read, body, capturedAt),
		"diagnostics":  diagnostics,
		"focusa":       oneShotMarkdownFocusa(read, body.URL),
		"cleanup":      cleanup,
		"duration_ms":  time.Since(started).Milliseconds(),
		"source_stats": map[string]any{"format": "markdown", "mode": read.Mode, "max_chars": body.MaxChars},
	})
}

func markdownMetadata(read *vision.PageReadResult, body markdownRequest, capturedAt string) map[string]any {
	out := map[string]any{
		"source_type":    "webpage",
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
	h := sha256.Sum256([]byte(read.URL + "\n" + read.Text))
	prefix := hex.EncodeToString(h[:])[:16]
	title := focusapacket.Truncate(firstMarkdownNonEmpty(read.Title, read.URL, requestedURL), 160)
	return map[string]any{
		"target_ref":     "source-markdown:" + sanitizeMarkdownURLForFocusa(firstMarkdownNonEmpty(read.URL, requestedURL)),
		"evidence_ref":   "uiai-source-markdown:sha256:" + prefix,
		"preferred_tool": "focusa_evidence_capture",
		"summary":        fmt.Sprintf("Converted %s to %d chars of Markdown", title, read.Chars),
		"next_tools":     []string{"focusa_evidence_capture"},
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
