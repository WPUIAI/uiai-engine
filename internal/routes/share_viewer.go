package routes

import (
	"html/template"
	"log"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/focusapacket"
	"github.com/go-chi/chi/v5"
)

const legacyShareCSP = "default-src 'none'; base-uri 'none'; object-src 'none'; frame-ancestors 'none'; form-action 'none'; script-src 'none'; style-src 'unsafe-inline'; frame-src http: https:"

// Used by both legacy share HTML and immutable Evidence PWA asset responses.
func setShareSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
}

var legacyShareTemplate = template.Must(template.New("share-viewer").Parse(`<!DOCTYPE html><html><head><title>{{.Title}}</title>
<meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<style>body{font-family:system-ui;max-width:1200px;margin:2em auto;padding:0 1em;color:#1e293b;background:#f8fafc}h1{color:#059669}iframe{width:100%;height:80vh;border:1px solid #e2e8f0;border-radius:8px}.meta{color:#64748b;font-size:13px;margin-bottom:1em}</style></head><body>
<h1>{{.Title}}</h1>{{if .Message}}<p>{{.Message}}</p>{{end}}
{{if .URL}}<div class="meta">Shared URL: <a href="{{.URL}}" rel="noreferrer noopener">{{.URL}}</a> · Viewed at: {{.ViewedAt}}</div>
<iframe src="{{.URL}}" loading="lazy" sandbox referrerpolicy="no-referrer" title="Read-only shared page"></iframe>{{end}}
</body></html>`))

type legacySharePage struct{ Title, Message, URL, ViewedAt string }

func validatedShareURL(raw string) (string, bool) {
	if strings.IndexFunc(raw, func(r rune) bool { return unicode.IsControl(r) || unicode.IsSpace(r) }) >= 0 {
		return "", false
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Opaque != "" || parsed.User != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", false
	}
	host := parsed.Hostname()
	if host == "" {
		return "", false
	}
	if strings.HasSuffix(parsed.Host, ":") {
		return "", false
	}
	if decoded, err := url.PathUnescape(raw); err != nil || strings.IndexFunc(decoded, unicode.IsControl) >= 0 {
		return "", false
	}
	if net.ParseIP(host) == nil {
		if len(host) > 253 {
			return "", false
		}
		for _, label := range strings.Split(strings.TrimSuffix(host, "."), ".") {
			if len(label) == 0 || len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
				return "", false
			}
			for _, r := range label {
				if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-') {
					return "", false
				}
			}
		}
	}
	if port := parsed.Port(); port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return "", false
		}
	}
	return focusapacket.SanitizeURL(parsed.String()), true
}

// HandleShareViewer renders untrusted stored metadata only through html/template.
func HandleShareViewer(_ *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setShareSecurityHeaders(w)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Content-Security-Policy", legacyShareCSP)
		w.Header().Set("X-Robots-Tag", "noindex, noarchive")
		w.Header().Set("Cache-Control", "no-store")
		status := http.StatusOK
		page := legacySharePage{}
		token := chi.URLParam(r, "token")
		value, ok := shareStore.Load(token)
		if !ok {
			status = http.StatusNotFound
			page = legacySharePage{Title: "Share Not Found", Message: "This share link has expired or does not exist."}
		} else {
			entry := value.(*shareEntry)
			if time.Now().After(entry.ExpiresAt) {
				shareStore.Delete(token)
				status = http.StatusGone
				page = legacySharePage{Title: "Share Expired", Message: "This share link has expired."}
			} else if target, valid := validatedShareURL(entry.URL); !valid {
				status = http.StatusBadRequest
				page = legacySharePage{Title: "Share Unavailable", Message: "The stored share URL is not permitted."}
			} else {
				entry.Views++
				title := entry.Title
				if title == "" {
					title = "WPUIAI Shared Design"
				}
				page = legacySharePage{Title: title, URL: target, ViewedAt: time.Now().UTC().Format(time.RFC3339)}
			}
		}
		w.WriteHeader(status)
		if err := legacyShareTemplate.Execute(w, page); err != nil {
			log.Printf("share viewer response write failed (%T)", err)
		}
	}
}
