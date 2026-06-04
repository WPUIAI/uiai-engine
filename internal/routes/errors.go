package routes

import (
	"net/http"
	"strconv"

	"github.com/WPUIAI/uiai-engine/internal/observability"
	"github.com/go-chi/chi/v5"
)

// MountErrorsRoutes exposes bounded engine/browser error history for agents/operators.
func MountErrorsRoutes(r chi.Router) {
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		limit, _ := strconv.Atoi(req.URL.Query().Get("limit"))
		source := req.URL.Query().Get("source")
		class := req.URL.Query().Get("class")
		events := observability.Recent(limit, source, class)
		writeJSON(w, 200, map[string]any{
			"events":       events,
			"count":        len(events),
			"stored_count": observability.Count(),
			"filters": map[string]string{
				"source": source,
				"class":  class,
			},
			"redaction": "query strings, fragments, secrets, auth headers, cookies, and request bodies are not stored",
		})
	})
}
