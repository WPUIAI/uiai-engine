package routes

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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
			Inline      *bool               `json:"inline"` // C-010-09: default false → artifact_ref only
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
		// C-010-09 default: pixels stay out of envelopes; inline only on demand.
		inline := body.Inline != nil && *body.Inline
		resp := map[string]any{
			"width":           result.Width,
			"height":          result.Height,
			"format":          result.Format,
			"size":            len(result.Data),
			"duration":        result.Duration.Milliseconds(),
			"focusa_evidence": focusaEvidence,
			"focusa":          focusaEvidence,
		}
		// C-010-09: default to artifact-ref responses; pixels inline only on demand.
		fullHash := sha256.Sum256(result.Data)
		resp["artifact_sha256"] = hex.EncodeToString(fullHash[:])
		if path, err := persistScreenshotArtifact(result.Data, result.Format); err == nil && path != "" {
			resp["artifact_path"] = path
		} else if err != nil {
			resp["artifact_store_error"] = err.Error()
		}
		resp["artifact_retrieval"] = "GET /api/screenshot/artifact/{sha256}"
		if inline {
			resp["screenshot"] = base64.StdEncoding.EncodeToString(result.Data)
		}
		if result.DOMReport != "" {
			resp["dom_report"] = result.DOMReport
		}
		writeJSON(w, 200, resp)
	})

	mountScreenshotArtifact(r) // C-010-09 retrieval surface

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

// screenshotArtifactDir resolves the durable store for C-010-09 artifacts.
func screenshotStoreDir() string {
	dir := os.Getenv("UIAI_SCREENSHOT_DIR")
	if dir == "" {
		dir = "/home/wpuiai/uiai-engine/data/screenshots"
	}
	return dir
}

// persistScreenshotArtifact stores bytes under sha256 name; idempotent.
func persistScreenshotArtifact(data []byte, format string) (string, error) {
	dir := screenshotStoreDir()
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	name := hex.EncodeToString(sum[:])
	if format != "" && format != "png" {
		name += "." + format
	} else {
		name += ".png"
	}
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}
	if err := os.WriteFile(path, data, 0o640); err != nil {
		return "", err
	}
	return path, nil
}

// mountScreenshotArtifact serves stored artifacts by hash (bounded, read-only).
func mountScreenshotArtifact(r chi.Router) {
	r.Get("/artifact/{sha}", func(w http.ResponseWriter, req *http.Request) {
		sum := chi.URLParam(req, "sha")
		if len(sum) < 16 || strings.ContainsAny(sum, "/\\.") {
			http.Error(w, "invalid sha", http.StatusBadRequest)
			return
		}
		dir := screenshotStoreDir()
		entries, err := os.ReadDir(dir)
		if err != nil {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		for _, e := range entries {
			if strings.HasPrefix(e.Name(), sum) {
				data, err := os.ReadFile(filepath.Join(dir, e.Name()))
				if err != nil {
					http.Error(w, "unreadable", http.StatusInternalServerError)
					return
				}
				w.Header().Set("Content-Type", "image/"+strings.TrimPrefix(filepath.Ext(e.Name()), "."))
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				_, _ = w.Write(data)
				return
			}
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
}
