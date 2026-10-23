package intelligence

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/ai"
	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/epwadelivery"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Layer is the Intelligence Layer service container.
type ArtifactPublisher func(*http.Request, string, string, string, string, string, []byte) (epwadelivery.Delivery, error)

type Layer struct {
	store             *Store
	usage             *UsageTracker
	github            *GitHubConfig
	cfg               *config.Config
	aiProv            *ai.Provider
	svcToken          string // AI_API_TOKEN for service auth
	artifactPublisher ArtifactPublisher
}

// NewLayer creates a fully-wired intelligence layer.
func NewLayer(cfg *config.Config, aiProv *ai.Provider) *Layer {
	dataDir := cfg.Storage.DataDir
	if dataDir == "" {
		dataDir = "./data"
	}
	indexRoot := filepath.Join(dataDir, "indexes")
	usagePath := filepath.Join(dataDir, "intelligence-usage.json")

	return &Layer{
		store: NewStore(indexRoot),
		usage: NewUsageTracker(usagePath),
		github: &GitHubConfig{
			Token:       os.Getenv("GITHUB_TOKEN"),
			Repo:        os.Getenv("GITHUB_REPO"),
			WorkflowID:  envOr("GITHUB_WORKFLOW_ID", "build-docfind-index.yml"),
			WorkflowRef: envOr("GITHUB_WORKFLOW_REF", "main"),
		},
		cfg:      cfg,
		aiProv:   aiProv,
		svcToken: os.Getenv("AI_API_TOKEN"),
	}
}

func (l *Layer) SetArtifactPublisher(publisher ArtifactPublisher) {
	l.artifactPublisher = publisher
}

// Mount registers all intelligence routes on the given chi.Router.
func (l *Layer) Mount(r chi.Router) {
	r.Get("/health", l.handleHealth)

	// Index management (service-token auth)
	r.Post("/index/trigger", l.handleIndexTrigger)
	r.Post("/index/upload", l.handleIndexUpload)
	r.Get("/index/{runId}", l.handleIndexStatus)
	r.Get("/documents/{runId}", l.handleDocuments)

	// Search (intelligence auth)
	r.Post("/search", l.handleSearch)

	// Embeddings (tier-gated)
	r.Post("/embed", l.handleEmbed)

	// Artifact delivery
	r.Get("/wasm/{runId}", l.handleServeWASM)
	r.Post("/wasm/{runId}", l.handleUploadWASM)
	r.Get("/js/{runId}", l.handleServeJS)
	r.Post("/js/{runId}", l.handleUploadJS)
}

// ── Health ──────────────────────────────────────────────

func (l *Layer) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, map[string]string{
		"status":    "healthy",
		"module":    "intelligence",
		"timestamp": NowISO(),
	})
}

// ── Index Trigger ───────────────────────────────────────

func (l *Layer) handleIndexTrigger(w http.ResponseWriter, r *http.Request) {
	if !l.requireServiceToken(w, r) {
		return
	}

	var req TriggerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	if req.RunID == "" {
		writeJSON(w, 400, map[string]string{"error": "runId required"})
		return
	}
	if len(req.Documents) == 0 {
		writeJSON(w, 400, map[string]string{"error": "documents required (min 1)"})
		return
	}

	// Validate every document against the 22-field schema
	for i, doc := range req.Documents {
		if doc.RunID != req.RunID {
			writeJSON(w, 400, map[string]any{
				"error": fmt.Sprintf("document[%d].runId mismatch: got %q, expected %q", i, doc.RunID, req.RunID),
			})
			return
		}
		if err := doc.Validate(); err != nil {
			writeJSON(w, 400, map[string]any{
				"error":   fmt.Sprintf("document[%d] validation failed", i),
				"field":   err.(*ValidationError).Field,
				"message": err.Error(),
			})
			return
		}
	}

	// Persist documents to disk
	if err := l.store.WriteDocuments(req.RunID, req.Documents); err != nil {
		writeJSON(w, 500, map[string]string{"error": "failed to persist documents: " + err.Error()})
		return
	}

	// Update metadata
	buildID := uuid.New().String()
	now := NowISO()
	source := req.Source
	if source == "" {
		source = "trigger"
	}

	meta, err := l.store.UpdateMetadata(req.RunID, func(m *IndexMetadata) {
		m.BuildID = buildID
		m.Status = "queued"
		m.DocCount = len(req.Documents)
		m.ChunkCount = len(req.Documents)
		if m.CreatedAt == "" {
			m.CreatedAt = now
		}
		m.UpdatedAt = now
		m.Source = source
	})
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": "metadata update failed"})
		return
	}

	// Dispatch GitHub Actions build
	var dispatch DispatchResult
	if l.github.IsConfigured() {
		dispatch = TriggerDocfindBuild(l.github, req.RunID, buildID)
		if !dispatch.OK {
			log.Printf("[intelligence] GitHub dispatch failed for %s: %s", req.RunID, dispatch.Error)
			l.store.UpdateMetadata(req.RunID, func(m *IndexMetadata) {
				m.Status = "failed"
			})
		}
	} else {
		dispatch = DispatchResult{OK: false, Error: "GitHub not configured (documents stored, no WASM build)"}
	}

	writeJSON(w, 200, map[string]any{
		"success":  true,
		"runId":    req.RunID,
		"buildId":  buildID,
		"status":   meta.Status,
		"docCount": meta.DocCount,
		"dispatch": dispatch,
	})
}

// ── Index Upload (multipart) ────────────────────────────

func (l *Layer) handleIndexUpload(w http.ResponseWriter, r *http.Request) {
	if !l.requireServiceToken(w, r) {
		return
	}

	contentType := r.Header.Get("Content-Type")
	mediaType, params, _ := mime.ParseMediaType(contentType)

	var runID string
	var docsData []byte
	var wasmData []byte
	var jsData []byte

	if mediaType == "multipart/form-data" {
		// Multipart form: wasm, js, documents, runId fields
		reader := multipart.NewReader(r.Body, params["boundary"])
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				writeJSON(w, 400, map[string]string{"error": "multipart parse error: " + err.Error()})
				return
			}
			data, _ := io.ReadAll(io.LimitReader(part, 50*1024*1024)) // 50MB per part

			switch part.FormName() {
			case "runId", "run_id":
				runID = strings.TrimSpace(string(data))
			case "documents":
				docsData = data
			case "wasm":
				wasmData = data
			case "js":
				jsData = data
			}
			part.Close()
		}
	} else {
		// JSON fallback
		var body struct {
			RunID     string     `json:"runId"`
			Documents []Document `json:"documents"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		runID = body.RunID
		if len(body.Documents) > 0 {
			docsData, _ = json.Marshal(body.Documents)
		}
	}

	if runID == "" {
		writeJSON(w, 400, map[string]string{"error": "runId is required"})
		return
	}

	var docs []Document
	if len(docsData) > 0 {
		if err := json.Unmarshal(docsData, &docs); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "documents are invalid"})
			return
		}
	}
	type artifactInput struct {
		ref, title, kind, mediaType, extension string
		payload                                []byte
	}
	inputs := []artifactInput{}
	if len(docsData) > 0 {
		inputs = append(inputs, artifactInput{"intelligence:documents:" + runID, "Intelligence document snapshot " + runID, "source_snapshot", "application/json", "json", docsData})
	}
	if len(wasmData) > 0 {
		inputs = append(inputs, artifactInput{"intelligence:wasm:" + runID, "Intelligence WASM artifact " + runID, "runtime_binary", "application/wasm", "wasm", wasmData})
	}
	if len(jsData) > 0 {
		inputs = append(inputs, artifactInput{"intelligence:javascript:" + runID, "Intelligence JavaScript artifact " + runID, "runtime_binary", "application/javascript", "js", jsData})
	}
	if len(inputs) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "documents, wasm, or js artifact required"})
		return
	}
	deliveries := make([]epwadelivery.Delivery, 0, len(inputs))
	allReady := true
	for _, input := range inputs {
		delivery, err := l.publishArtifact(r, input.ref, input.title, input.kind, input.mediaType, input.extension, input.payload)
		if err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "EPWA publication failed"})
			return
		}
		deliveries = append(deliveries, delivery)
		allReady = allReady && delivery.State == epwadelivery.StateReady
	}
	if !allReady {
		writeJSON(w, http.StatusAccepted, map[string]any{
			"schema": "uiai.intelligence_index_upload_result.v2", "runId": runID,
			"delivery_state": "blocked", "epwa_deliveries": deliveries, "storage_posture": "withheld_until_all_epwa_deliveries_ready",
		})
		return
	}
	if len(docs) > 0 {
		if err := l.store.WriteDocuments(runID, docs); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "document compatibility write failed after EPWA publication"})
			return
		}
	}
	if len(wasmData) > 0 {
		if err := l.store.WriteArtifact(runID, "docfind_bg.wasm", wasmData); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "WASM compatibility write failed after EPWA publication"})
			return
		}
	}
	if len(jsData) > 0 {
		if err := l.store.WriteArtifact(runID, "docfind.js", jsData); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "JavaScript compatibility write failed after EPWA publication"})
			return
		}
	}
	meta, err := l.store.UpdateMetadata(runID, func(m *IndexMetadata) {
		m.Status = "ready"
		m.Artifacts = ArtifactStatus{WASM: len(wasmData) > 0, JS: len(jsData) > 0}
		m.UpdatedAt = NowISO()
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "metadata write failed after EPWA publication"})
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"schema": "uiai.intelligence_index_upload_result.v2", "success": true, "runId": runID,
		"status": meta.Status, "artifacts": meta.Artifacts, "delivery_state": "ready", "epwa_deliveries": deliveries,
	})
}

// ── Index Status ────────────────────────────────────────

func (l *Layer) handleIndexStatus(w http.ResponseWriter, r *http.Request) {
	runID := chi.URLParam(r, "runId")
	meta, err := l.store.ReadMetadata(runID)
	if err != nil || meta == nil {
		writeJSON(w, 404, map[string]string{"error": "Index not found"})
		return
	}

	// Refresh artifact status from disk
	meta.Artifacts = ArtifactStatus{
		WASM: l.store.HasArtifact(runID, "docfind_bg.wasm"),
		JS:   l.store.HasArtifact(runID, "docfind.js"),
	}

	writeJSON(w, 200, meta)
}

// ── Documents ───────────────────────────────────────────

func (l *Layer) handleDocuments(w http.ResponseWriter, r *http.Request) {
	if !l.requireServiceToken(w, r) {
		return
	}

	runID := chi.URLParam(r, "runId")
	docs, err := l.store.ReadDocuments(runID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if docs == nil {
		writeJSON(w, 404, map[string]string{"error": "Documents not found"})
		return
	}

	writeJSON(w, 200, map[string]any{
		"runId":     runID,
		"documents": docs,
		"count":     len(docs),
	})
}

// ── Search ──────────────────────────────────────────────

func (l *Layer) handleSearch(w http.ResponseWriter, r *http.Request) {
	// Auth: service token OR license/api key
	tier := "free"
	authKey := ""

	if l.hasServiceToken(r) {
		tier = "enterprise"
		authKey = "service"
	} else {
		// Accept X-License-Key or X-API-Key for tier resolution
		// For now, we read the tier from the global auth context if available
		licKey := r.Header.Get("X-License-Key")
		apiKey := r.Header.Get("X-API-Key")
		if licKey == "" && apiKey == "" {
			writeJSON(w, 401, map[string]string{"error": "Authentication required"})
			return
		}
		authKey = licKey
		if authKey == "" {
			authKey = apiKey
		}
		// TODO: validate key and resolve tier from WP REST
		// For now default to developer tier for any authenticated request
		tier = "developer"
	}

	var req SearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}

	limits := GetTierLimits(tier)

	if len(req.Query) > limits.MaxQueryLength {
		writeJSON(w, 400, map[string]string{"error": "Query too long"})
		return
	}

	docs, err := l.store.ReadDocuments(req.RunID)
	if err != nil {
		writeJSON(w, 500, map[string]string{"error": err.Error()})
		return
	}
	if docs == nil {
		writeJSON(w, 404, map[string]string{"error": "No documents indexed for run"})
		return
	}

	limit := req.Limit
	if limit <= 0 || limit > limits.SearchLimit {
		limit = limits.SearchLimit
	}

	results := Search(docs, req.Query, limit)
	if results == nil {
		results = []SearchResult{}
	}

	writeJSON(w, 200, map[string]any{
		"query":   req.Query,
		"runId":   req.RunID,
		"count":   len(results),
		"limit":   limit,
		"tier":    tier,
		"results": results,
	})
	_ = authKey
}

// ── Embeddings ──────────────────────────────────────────

func (l *Layer) handleEmbed(w http.ResponseWriter, r *http.Request) {
	tier := "free"
	authKey := ""

	if l.hasServiceToken(r) {
		tier = "enterprise"
		authKey = "service"
	} else {
		licKey := r.Header.Get("X-License-Key")
		apiKey := r.Header.Get("X-API-Key")
		if licKey == "" && apiKey == "" {
			writeJSON(w, 401, map[string]string{"error": "Authentication required"})
			return
		}
		authKey = licKey
		if authKey == "" {
			authKey = apiKey
		}
		tier = "developer"
	}

	limits := GetTierLimits(tier)
	if !limits.AllowEmbeddings {
		writeJSON(w, 403, map[string]string{"error": "Embeddings not available for current tier"})
		return
	}

	var req EmbedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
		return
	}
	if len(req.Texts) == 0 {
		writeJSON(w, 400, map[string]string{"error": "texts required (min 1)"})
		return
	}

	model := req.Model
	if model == "" {
		model = "text-embedding-3-small"
	}

	// Check daily usage
	today := time.Now().UTC().Format("2006-01-02")
	usage := l.usage.Get(authKey, today)
	nextCount := usage.Count + len(req.Texts)

	if limits.EmbedDailyLimit > 0 && nextCount > limits.EmbedDailyLimit {
		writeJSON(w, 429, map[string]any{
			"error": "Embedding limit exceeded",
			"limit": limits.EmbedDailyLimit,
			"used":  usage.Count,
		})
		return
	}

	// Call OpenRouter embeddings (matches Bun implementation)
	embeddings, err := l.computeEmbeddings(req.Texts, model)
	if err != nil {
		log.Printf("[intelligence] embed error: %v", err)
		// Graceful degradation: return zero vectors
		zeros := make([]map[string]any, len(req.Texts))
		for i := range zeros {
			zeros[i] = map[string]any{"embedding": make([]float64, 256), "index": i}
		}
		writeJSON(w, 200, map[string]any{
			"success":    false,
			"model":      model,
			"embeddings": zeros,
			"status":     "degraded",
			"error":      err.Error(),
		})
		return
	}

	// Track usage
	updated := l.usage.Increment(authKey, today, len(req.Texts))

	writeJSON(w, 200, map[string]any{
		"success":    true,
		"model":      model,
		"embeddings": embeddings,
		"usage": map[string]any{
			"date":  updated.Date,
			"count": updated.Count,
			"limit": limits.EmbedDailyLimit,
		},
	})
}

// ── WASM/JS Artifact Delivery ───────────────────────────

func (l *Layer) publishArtifact(req *http.Request, artifactRef, title, kind, mediaType, extension string, payload []byte) (epwadelivery.Delivery, error) {
	if l.artifactPublisher == nil {
		return epwadelivery.Delivery{}, errors.New("EPWA artifact publisher unavailable")
	}
	return l.artifactPublisher(req, artifactRef, title, kind, mediaType, extension, payload)
}

func writeArtifactDelivery(w http.ResponseWriter, delivery epwadelivery.Delivery, successStatus int) {
	status := http.StatusAccepted
	response := map[string]any{
		"schema": "uiai.binary_artifact_result.v2", "artifact_ref": delivery.Artifact.ArtifactRef,
		"delivery_state": delivery.State, "epwa_delivery": delivery, "raw_output_posture": "withheld_by_mandatory_epwa_delivery",
	}
	if delivery.State == epwadelivery.StateReady {
		status = successStatus
		response["artifact_url"] = delivery.EPWA.RecordURL
		response["portable_url"] = delivery.EPWA.PortableURL
	}
	writeJSON(w, status, response)
}

func (l *Layer) handleServeWASM(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusGone, map[string]any{
		"schema": "uiai.epwa_delivery_error.v1", "code": "legacy_raw_artifact_removed",
		"artifact_kind": "wasm", "runId": chi.URLParam(r, "runId"),
		"message": "raw WASM is available only through its EPWA viewer and portable package",
	})
}

func (l *Layer) handleUploadWASM(w http.ResponseWriter, r *http.Request) {
	if !l.requireServiceToken(w, r) {
		return
	}
	runID := chi.URLParam(r, "runId")
	const maxWASMBytes = 50 * 1024 * 1024
	data, err := io.ReadAll(io.LimitReader(r.Body, maxWASMBytes+1))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read body failed"})
		return
	}
	if len(data) > maxWASMBytes {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "WASM artifact exceeds 50 MiB"})
		return
	}
	delivery, err := l.publishArtifact(r, "intelligence:wasm:"+runID, "Intelligence WASM artifact "+runID, "runtime_binary", "application/wasm", "wasm", data)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "EPWA publication failed"})
		return
	}
	if delivery.State != epwadelivery.StateReady {
		writeArtifactDelivery(w, delivery, http.StatusCreated)
		return
	}
	if err := l.store.WriteArtifact(runID, "docfind_bg.wasm", data); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "compatibility artifact write failed after EPWA publication"})
		return
	}
	_, _ = l.store.UpdateMetadata(runID, func(m *IndexMetadata) {
		m.Artifacts.WASM = true
		m.UpdatedAt = NowISO()
	})
	writeArtifactDelivery(w, delivery, http.StatusCreated)
}

func (l *Layer) handleServeJS(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusGone, map[string]any{
		"schema": "uiai.epwa_delivery_error.v1", "code": "legacy_raw_artifact_removed",
		"artifact_kind": "javascript", "runId": chi.URLParam(r, "runId"),
		"message": "raw JavaScript is available only through its EPWA viewer and portable package",
	})
}

func (l *Layer) handleUploadJS(w http.ResponseWriter, r *http.Request) {
	if !l.requireServiceToken(w, r) {
		return
	}
	runID := chi.URLParam(r, "runId")
	const maxJavaScriptBytes = 20 * 1024 * 1024
	data, err := io.ReadAll(io.LimitReader(r.Body, maxJavaScriptBytes+1))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read body failed"})
		return
	}
	if len(data) > maxJavaScriptBytes {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{"error": "JavaScript artifact exceeds 20 MiB"})
		return
	}
	delivery, err := l.publishArtifact(r, "intelligence:javascript:"+runID, "Intelligence JavaScript artifact "+runID, "runtime_binary", "application/javascript", "js", data)
	if err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "EPWA publication failed"})
		return
	}
	if delivery.State != epwadelivery.StateReady {
		writeArtifactDelivery(w, delivery, http.StatusCreated)
		return
	}
	if err := l.store.WriteArtifact(runID, "docfind.js", data); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "compatibility artifact write failed after EPWA publication"})
		return
	}
	_, _ = l.store.UpdateMetadata(runID, func(m *IndexMetadata) {
		m.Artifacts.JS = true
		m.UpdatedAt = NowISO()
	})
	writeArtifactDelivery(w, delivery, http.StatusCreated)
}

// ── Embeddings call ─────────────────────────────────────

func (l *Layer) computeEmbeddings(input []string, model string) ([]map[string]any, error) {
	// Use OpenRouter (matches Bun implementation per INTELLIGENCE_LAYER.md)
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	if apiKey == "" {
		// Fallback to OpenAI
		apiKey = os.Getenv("OPENAI_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OPENROUTER_API_KEY or OPENAI_API_KEY not configured")
		}
	}

	baseURL := "https://openrouter.ai/api/v1"
	if os.Getenv("OPENROUTER_API_KEY") == "" {
		baseURL = "https://api.openai.com/v1"
	}

	body, _ := json.Marshal(map[string]any{"input": input, "model": model})
	req, _ := http.NewRequest("POST", baseURL+"/embeddings", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("embeddings API %d: %s", resp.StatusCode, truncate(string(respBody), 200))
	}

	var data struct {
		Model string `json:"model"`
		Data  []struct {
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &data); err != nil {
		return nil, fmt.Errorf("parse embeddings: %w", err)
	}

	result := make([]map[string]any, len(data.Data))
	for i, d := range data.Data {
		result[i] = map[string]any{"embedding": d.Embedding, "index": d.Index}
	}
	return result, nil
}

// ── Auth helpers ────────────────────────────────────────

func (l *Layer) hasServiceToken(r *http.Request) bool {
	if l.svcToken == "" {
		return false
	}
	headerToken := r.Header.Get("X-API-Token")
	bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	return headerToken == l.svcToken || bearer == l.svcToken
}

func (l *Layer) requireServiceToken(w http.ResponseWriter, r *http.Request) bool {
	if l.svcToken == "" {
		writeJSON(w, 503, map[string]string{"error": "AI_API_TOKEN not configured"})
		return false
	}
	if !l.hasServiceToken(r) {
		writeJSON(w, 401, map[string]string{"error": "Invalid service token"})
		return false
	}
	return true
}

// ── Utilities ───────────────────────────────────────────

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
