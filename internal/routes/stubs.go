// Package routes provides HTTP route handlers for every API module.
// Health mount and writeJSON helper. Stub mounts kept for fallback only.
package routes

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/ai"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/go-chi/chi/v5"
)

// --- Health (implemented) ---

func MountHealth(r chi.Router, cfg *config.Config, aiProv *ai.Provider) {
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{
			"status":    "healthy",
			"service":   "uiai-engine",
			"version":   "2.0.0",
			"timestamp": time.Now().UTC().Format(time.RFC3339Nano),
		})
	})
	r.Get("/providers", func(w http.ResponseWriter, req *http.Request) {
		providerSet := map[string]bool{}
		for _, m := range aiProv.AvailableModels() {
			providerSet[m.Provider] = true
		}
		result := map[string]any{}
		for _, name := range []string{"anthropic", "openai", "openrouter", "fireworks", "kimi", "qwen"} {
			result[name] = map[string]any{"available": providerSet[name]}
		}
		writeJSON(w, 200, result)
	})
}

type shareViewerPage struct {
	Title string
	URL   string
	Views int
}

type shareViewerErrorPage struct {
	Title   string
	Message string
}

var shareViewerTemplate = template.Must(template.New("share-viewer").Parse(`<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Title}}</title></head>
<body><main><h1>{{.Title}}</h1><p>Shared URL: <a href="{{.URL}}">{{.URL}}</a> · Views: {{.Views}}</p>
<iframe title="Shared design" src="{{.URL}}" loading="lazy" referrerpolicy="no-referrer"></iframe></main></body></html>`))

var shareViewerErrorTemplate = template.Must(template.New("share-viewer-error").Parse(`<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"><title>{{.Title}}</title></head>
<body><main><h1>{{.Title}}</h1><p>{{.Message}}</p></main></body></html>`))

func setShareViewerSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; frame-src http: https:; base-uri 'none'; form-action 'none'; object-src 'none'; script-src 'none'; style-src 'none'; frame-ancestors 'none'")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow, noarchive")
}

func normalizeShareViewerURL(raw string) (string, bool) {
	if raw == "" || raw != strings.TrimSpace(raw) || strings.ContainsAny(raw, "\x00\r\n\t") {
		return "", false
	}
	parsed, err := url.ParseRequestURI(raw)
	if err != nil || !parsed.IsAbs() || parsed.Hostname() == "" || parsed.User != nil || parsed.Opaque != "" {
		return "", false
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", false
	}
	return parsed.String(), true
}

func writeShareViewerError(w http.ResponseWriter, status int, title, message string) {
	var body bytes.Buffer
	if err := shareViewerErrorTemplate.Execute(&body, shareViewerErrorPage{Title: title, Message: message}); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(status)
	_, _ = w.Write(body.Bytes())
}

// HandleShareViewer serves a public share page with live screenshot.
func HandleShareViewer(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setShareViewerSecurityHeaders(w)
		token := chi.URLParam(r, "token")
		v, ok := shareStore.Load(token)
		if !ok {
			writeShareViewerError(w, http.StatusNotFound, "Share Not Found", "This share link has expired or does not exist.")
			return
		}
		entry := v.(*shareEntry)
		if time.Now().After(entry.ExpiresAt) {
			shareStore.Delete(token)
			writeShareViewerError(w, http.StatusGone, "Share Expired", "This share link has expired.")
			return
		}
		shareURL, ok := normalizeShareViewerURL(entry.URL)
		if !ok {
			writeShareViewerError(w, http.StatusUnprocessableEntity, "Share Unavailable", "This share contains an invalid destination URL.")
			return
		}

		title := entry.Title
		if title == "" {
			title = "WPUIAI Shared Design"
		}
		entry.Views++

		var body bytes.Buffer
		if err := shareViewerTemplate.Execute(&body, shareViewerPage{Title: title, URL: shareURL, Views: entry.Views}); err != nil {
			writeShareViewerError(w, http.StatusInternalServerError, "Share Unavailable", "The share viewer could not be rendered.")
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body.Bytes())
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(InjectCost(data, w))
}
