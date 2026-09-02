package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func readOptionsFromQuery(req *http.Request) (vision.ReadOptions, error) {
	query := req.URL.Query()
	options := vision.ReadOptions{
		Selector: query.Get("selector"),
		Format:   query.Get("format"),
		Mode:     query.Get("mode"),
	}
	if raw := query.Get("max_chars"); raw != "" {
		maxChars, err := strconv.Atoi(raw)
		if err != nil || maxChars < 0 {
			return vision.ReadOptions{}, fmt.Errorf("invalid max_chars")
		}
		options.MaxChars = maxChars
	}
	if raw := query.Get("include_links"); raw != "" {
		includeLinks, err := strconv.ParseBool(raw)
		if err != nil {
			return vision.ReadOptions{}, fmt.Errorf("invalid include_links")
		}
		options.IncludeLinks = includeLinks
	}
	if raw := query.Get("include_images"); raw != "" {
		includeImages, err := strconv.ParseBool(raw)
		if err != nil {
			return vision.ReadOptions{}, fmt.Errorf("invalid include_images")
		}
		options.IncludeImages = includeImages
	}
	return options, nil
}

func sessionReadHandler(sm *vision.SessionManager) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		sess, ok := sm.Get(chi.URLParam(req, "sessionID"))
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "session not found"})
			return
		}

		var options vision.ReadOptions
		if req.Method == http.MethodGet {
			parsed, err := readOptionsFromQuery(req)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			options = parsed
		} else {
			_ = json.NewDecoder(req.Body).Decode(&options)
		}

		if options.Selector != "" {
			if resolved, resolveErr := sess.ResolveSelector(options.Selector); resolveErr == nil {
				options.Selector = resolved
			} else {
				writeSessionError(w, http.StatusNotFound, "selector_not_found", resolveErr, sess, map[string]any{"action": "read", "selector": options.Selector})
				return
			}
		}
		result, err := sess.ReadPage(options)
		if err != nil {
			writeSessionError(w, http.StatusInternalServerError, classifySessionError(err), err, sess, map[string]any{"action": "read", "selector": options.Selector})
			return
		}
		writeJSON(w, http.StatusOK, result)
	}
}
