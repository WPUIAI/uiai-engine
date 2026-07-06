package routes

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/focusapacket"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

type shareEntry struct {
	ID             string         `json:"id"`
	URL            string         `json:"url"`
	Title          string         `json:"title"`
	Data           map[string]any `json:"data"`
	CreatedAt      time.Time      `json:"createdAt"`
	ExpiresAt      time.Time      `json:"expiresAt"`
	Views          int            `json:"views"`
	FocusaEvidence map[string]any `json:"focusa_evidence,omitempty"`
}

var shareStore sync.Map
var shareDataDir string

// loadShareStore reads persisted shares from disk on startup.
func loadShareStore(dataDir string) {
	shareDataDir = filepath.Join(dataDir, "shares")
	os.MkdirAll(shareDataDir, 0750)

	entries, _ := os.ReadDir(shareDataDir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(shareDataDir, e.Name())) // #nosec G304 -- e.Name comes from os.ReadDir of shareDataDir.
		if err != nil {
			continue
		}
		var entry shareEntry
		if json.Unmarshal(data, &entry) == nil {
			if time.Now().Before(entry.ExpiresAt) {
				shareStore.Store(entry.ID, &entry)
			} else {
				os.Remove(filepath.Join(shareDataDir, e.Name()))
			}
		}
	}
}

func persistShare(entry *shareEntry) {
	if shareDataDir == "" {
		return
	}
	data, _ := json.MarshalIndent(entry, "", "  ")
	os.WriteFile(filepath.Join(shareDataDir, entry.ID+".json"), data, 0600)
}

func shareEvidence(id, targetURL, title string, scope *vision.FocusaScope) map[string]any {
	result := "UIAI share artifact created"
	if title != "" {
		result += ": " + title
	}
	evidence := map[string]any{
		"target_ref":          "browser:" + focusapacket.SanitizeURL(targetURL),
		"result":              result,
		"summary":             result,
		"evidence_ref":        "uiai-share:" + id,
		"artifact_ref":        "/api/share/" + id,
		"preferred_tool":      "focusa_evidence_capture",
		"next_tools":          []string{"focusa_evidence_capture", "focusa_active_object_resolve", "focusa_predict_record"},
		"focusa_scope_status": routeFocusaScopeStatus(scope),
	}
	if scope != nil {
		evidence["focusa_scope"] = scope
	}
	return evidence
}

func MountShareReal(r chi.Router, cfg *config.Config, pool vision.PoolSource) {
	loadShareStore(cfg.Storage.DataDir)

	// List shares
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		var shares []any
		shareStore.Range(func(_, v any) bool {
			e := v.(*shareEntry)
			shares = append(shares, map[string]any{
				"id": e.ID, "url": e.URL, "title": e.Title,
				"created_at": e.CreatedAt, "expires_at": e.ExpiresAt,
				"views": e.Views,
			})
			return true
		})
		if shares == nil {
			shares = []any{}
		}
		writeJSON(w, 200, map[string]any{"shares": shares, "count": len(shares)})
	})

	// Create share
	r.Post("/create", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			URL         string              `json:"url"`
			Title       string              `json:"title"`
			ExpiresIn   int                 `json:"expiresIn"` // minutes
			Data        map[string]any      `json:"data"`
			FocusaScope *vision.FocusaScope `json:"focusa_scope"`
		}
		json.NewDecoder(req.Body).Decode(&body)
		if body.URL == "" {
			writeJSON(w, 400, map[string]string{"error": "url required"})
			return
		}

		idBytes := make([]byte, 8)
		rand.Read(idBytes)
		id := hex.EncodeToString(idBytes)

		expiresIn := body.ExpiresIn
		if expiresIn <= 0 {
			expiresIn = 1440 // 24 hours default
		}

		entry := &shareEntry{
			ID:             id,
			URL:            body.URL,
			Title:          body.Title,
			Data:           body.Data,
			CreatedAt:      time.Now(),
			ExpiresAt:      time.Now().Add(time.Duration(expiresIn) * time.Minute),
			FocusaEvidence: shareEvidence(id, body.URL, body.Title, body.FocusaScope),
		}
		shareStore.Store(id, entry)
		persistShare(entry)

		writeJSON(w, 200, map[string]any{
			"id":              id,
			"shareUrl":        "/api/share/" + id,
			"expiresAt":       entry.ExpiresAt,
			"focusa_evidence": entry.FocusaEvidence,
		})
	})

	// Create multi-page share
	r.Post("/multi", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			URLs        []string            `json:"urls"`
			Title       string              `json:"title"`
			ExpiresIn   int                 `json:"expiresIn"`
			Data        map[string]any      `json:"data"`
			FocusaScope *vision.FocusaScope `json:"focusa_scope"`
		}
		json.NewDecoder(req.Body).Decode(&body)
		if len(body.URLs) == 0 {
			writeJSON(w, 400, map[string]string{"error": "urls required"})
			return
		}

		idBytes := make([]byte, 8)
		rand.Read(idBytes)
		id := "multi-" + hex.EncodeToString(idBytes)

		expiresIn := body.ExpiresIn
		if expiresIn <= 0 {
			expiresIn = 1440
		}

		entry := &shareEntry{
			ID:    id,
			URL:   body.URLs[0],
			Title: body.Title,
			Data: map[string]any{
				"urls":      body.URLs,
				"custom":    body.Data,
				"pageCount": len(body.URLs),
			},
			CreatedAt:      time.Now(),
			ExpiresAt:      time.Now().Add(time.Duration(expiresIn) * time.Minute),
			FocusaEvidence: shareEvidence(id, body.URLs[0], body.Title, body.FocusaScope),
		}
		shareStore.Store(id, entry)
		persistShare(entry)

		writeJSON(w, 200, map[string]any{
			"id":              id,
			"shareUrl":        "/api/share/" + id,
			"pages":           len(body.URLs),
			"expiresAt":       entry.ExpiresAt,
			"focusa_evidence": entry.FocusaEvidence,
		})
	})

	// Get share
	r.Get("/{id}", func(w http.ResponseWriter, req *http.Request) {
		id := chi.URLParam(req, "id")
		v, ok := shareStore.Load(id)
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "share not found"})
			return
		}
		entry := v.(*shareEntry)
		if time.Now().After(entry.ExpiresAt) {
			shareStore.Delete(id)
			writeJSON(w, 410, map[string]string{"error": "share expired"})
			return
		}
		entry.Views++
		writeJSON(w, 200, entry)
	})

	// Delete share
	r.Delete("/{id}", func(w http.ResponseWriter, req *http.Request) {
		shareStore.Delete(chi.URLParam(req, "id"))
		writeJSON(w, 200, map[string]string{"message": "share deleted"})
	})

	// Screenshot for share
	r.Post("/{id}/screenshot", func(w http.ResponseWriter, req *http.Request) {
		if pool == nil {
			writeJSON(w, 503, map[string]string{"error": "vision pool not available"})
			return
		}
		id := chi.URLParam(req, "id")
		v, ok := shareStore.Load(id)
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "share not found"})
			return
		}
		entry := v.(*shareEntry)

		result, err := pool.Screenshot(vision.ScreenshotOpts{
			URL:    entry.URL,
			Width:  1280,
			Height: 800,
			Format: "jpeg",
		})
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}

		artifactRef := screenshotEvidenceRef(result.Data)
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("X-Focusa-Evidence-Ref", artifactRef)
		w.Header().Set("X-Focusa-Target-Ref", entry.URL)
		w.Write(result.Data)
	})
}
