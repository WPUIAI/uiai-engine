package routes

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/focusapacket"
	"github.com/WPUIAI/uiai-engine/internal/storage"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func screenshotEvidenceRef(data []byte) string {
	sum := sha256.Sum256(data)
	return "uiai-screenshot:sha256:" + hex.EncodeToString(sum[:])[:16]
}

func routeFocusaScopeStatus(scope *vision.FocusaScope) string {
	if scope == nil {
		return string(focusapacket.ScopeMissing)
	}
	if scope.ProjectRoot != "" && scope.ContinuityID != "" {
		return string(focusapacket.ScopePresent)
	}
	return string(focusapacket.ScopePartial)
}

func screenshotFocusaMetadata(targetURL, artifactRef, format string, bytes int, scope *vision.FocusaScope) map[string]any {
	summary := "UIAI screenshot captured"
	if bytes > 0 {
		summary += " bytes=" + strconv.Itoa(bytes)
	}
	evidence := map[string]any{
		"target_ref":          "browser:" + focusapacket.SanitizeURL(targetURL),
		"result":              summary,
		"summary":             summary,
		"evidence_ref":        artifactRef,
		"artifact_ref":        artifactRef,
		"preferred_tool":      "focusa_evidence_capture",
		"next_tools":          []string{"focusa_evidence_capture", "focusa_active_object_resolve", "focusa_predict_record"},
		"focusa_scope_status": routeFocusaScopeStatus(scope),
		"mime_type":           "image/" + format,
		"bytes":               bytes,
	}
	if scope != nil {
		evidence["focusa_scope"] = scope
	}
	return evidence
}

func MountScreenshotReal(r chi.Router, _ *config.Config, pool vision.PoolSource, usage *storage.UsageStore) {
	r.Post("/", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			URL         string              `json:"url"`
			Width       int                 `json:"width"`
			Height      int                 `json:"height"`
			FullPage    bool                `json:"fullPage"`
			Format      string              `json:"format"`
			Quality     int                 `json:"quality"`
			WaitFor     string              `json:"waitFor"`
			Delay       int                 `json:"delay"`
			Cookies     string              `json:"cookies"` // "name=value; name2=value2"
			Timeout     int                 `json:"timeout"` // overall timeout in seconds (default: 30)
			NoCache     bool                `json:"nocache"` // skip cache, always take fresh screenshot
			FocusaScope *vision.FocusaScope `json:"focusa_scope"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
			return
		}
		if body.URL == "" {
			writeJSON(w, 400, map[string]string{"error": "url required"})
			return
		}

		if pool == nil {
			writeJSON(w, 503, map[string]string{"error": "vision pool not initialized"})
			return
		}

		result, err := pool.Screenshot(vision.ScreenshotOpts{
			URL:      body.URL,
			Width:    body.Width,
			Height:   body.Height,
			FullPage: body.FullPage,
			Format:   body.Format,
			Quality:  body.Quality,
			WaitFor:  body.WaitFor,
			Delay:    body.Delay,
			Cookies:  body.Cookies,
			Timeout:  body.Timeout,
			NoCache:  body.NoCache,
		})
		if err != nil {
			if errors.Is(err, vision.ErrQueueFull) {
				w.Header().Set("Retry-After", "10")
				writeJSON(w, 429, map[string]string{"error": "too many requests — screenshot queue full, retry after 10s"})
				return
			}
			if errors.Is(err, vision.ErrQueueTimeout) {
				writeJSON(w, 408, map[string]string{"error": "request timed out waiting in screenshot queue"})
				return
			}
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}

		// Record usage
		if usage != nil {
			usage.Record(storage.UsageRecord{
				Type:      "screenshot",
				Status:    "success",
				CostUSD:   0.005, // flat per-screenshot cost
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
			})
		}

		artifactRef := screenshotEvidenceRef(result.Data)
		focusaEvidence := screenshotFocusaMetadata(body.URL, artifactRef, result.Format, len(result.Data), body.FocusaScope)
		if body.FocusaScope != nil {
			focusaEvidence["focusa_scope"] = body.FocusaScope
		}
		resp := map[string]any{
			"screenshot":      base64.StdEncoding.EncodeToString(result.Data),
			"width":           result.Width,
			"height":          result.Height,
			"format":          result.Format,
			"size":            len(result.Data),
			"duration":        result.Duration.Milliseconds(),
			"focusa_evidence": focusaEvidence,
			"focusa":          focusaEvidence,
		}
		if result.DOMReport != "" {
			resp["dom_report"] = result.DOMReport
		}
		writeJSON(w, 200, resp)
	})

	r.Get("/health", func(w http.ResponseWriter, req *http.Request) {
		if pool == nil {
			writeJSON(w, 503, map[string]string{"status": "unavailable", "error": "vision pool not initialized"})
			return
		}
		writeJSON(w, 200, map[string]any{
			"status": "healthy",
			"pool":   pool.Stats(),
		})
	})
}
