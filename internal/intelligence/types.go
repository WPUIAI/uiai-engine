package intelligence

import "time"

// Document matches the 22-field IndexDocument schema from INTELLIGENCE_LAYER.md.
type Document struct {
	// Core identification
	ID     string `json:"id"`
	RunID  string `json:"runId"`
	SiteID string `json:"siteId,omitempty"`

	// Content
	Title   string `json:"title"`
	Body    string `json:"body"`
	Summary string `json:"summary,omitempty"`

	// Source tracking
	SourceType string `json:"sourceType"` // upload | gdrive | scrape | intake | generated
	SourceURL  string `json:"sourceUrl,omitempty"`
	SourcePath string `json:"sourcePath,omitempty"`

	// Classification
	Category     string `json:"category"`
	DocumentType string `json:"documentType"`
	PageType     string `json:"pageType,omitempty"`

	// Metadata
	Keywords []string `json:"keywords,omitempty"`
	Entities []string `json:"entities,omitempty"`
	Tone     string   `json:"tone,omitempty"`
	Industry string   `json:"industry,omitempty"`

	// Hierarchy
	ParentID string `json:"parentId,omitempty"`
	Order    int    `json:"order,omitempty"`

	// Timestamps
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`

	// Search optimization
	Boost             float64 `json:"boost,omitempty"`
	ExcludeFromSearch bool    `json:"excludeFromSearch,omitempty"`
}

// ValidSourceTypes are the allowed values for Document.SourceType.
var ValidSourceTypes = map[string]bool{
	"upload":    true,
	"gdrive":    true,
	"scrape":    true,
	"intake":    true,
	"generated": true,
}

// Validate checks required fields per the INTELLIGENCE_LAYER.md schema.
func (d *Document) Validate() error {
	if d.ID == "" {
		return errField("id")
	}
	if d.RunID == "" {
		return errField("runId")
	}
	if d.Title == "" {
		return errField("title")
	}
	if d.Body == "" {
		return errField("body")
	}
	if d.SourceType == "" {
		return errField("sourceType")
	}
	if !ValidSourceTypes[d.SourceType] {
		return &ValidationError{Field: "sourceType", Message: "must be one of: upload, gdrive, scrape, intake, generated"}
	}
	if d.Category == "" {
		return errField("category")
	}
	if d.DocumentType == "" {
		return errField("documentType")
	}
	if d.CreatedAt == "" {
		return errField("createdAt")
	}
	if d.UpdatedAt == "" {
		return errField("updatedAt")
	}
	return nil
}

// ValidationError represents a document schema validation failure.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	return e.Field + ": " + e.Message
}

func errField(name string) *ValidationError {
	return &ValidationError{Field: name, Message: "required"}
}

// IndexMetadata tracks build status per the INTELLIGENCE_LAYER.md spec.
type IndexMetadata struct {
	RunID      string         `json:"runId"`
	BuildID    string         `json:"buildId"`
	Status     string         `json:"status"` // queued | building | ready | failed
	DocCount   int            `json:"docCount"`
	ChunkCount int            `json:"chunkCount"`
	CreatedAt  string         `json:"createdAt"`
	UpdatedAt  string         `json:"updatedAt"`
	Artifacts  ArtifactStatus `json:"artifacts"`
	Source     string         `json:"source"` // trigger | upload | manual
}

// ArtifactStatus tracks which WASM/JS artifacts exist.
type ArtifactStatus struct {
	WASM bool `json:"wasm"`
	JS   bool `json:"js"`
}

// SearchRequest is the input for POST /api/intelligence/search.
type SearchRequest struct {
	Query  string `json:"query"`
	RunID  string `json:"runId"`
	Limit  int    `json:"limit,omitempty"`
	Hybrid bool   `json:"hybrid,omitempty"`
}

// SearchResult is a single scored document match.
type SearchResult struct {
	ID         string           `json:"id"`
	RunID      string           `json:"runId"`
	Title      string           `json:"title"`
	Summary    string           `json:"summary,omitempty"`
	SourceType string           `json:"sourceType"`
	SourceURL  string           `json:"sourceUrl,omitempty"`
	Score      float64          `json:"score"`
	Matches    []string         `json:"matches"`
	Snippet    string           `json:"snippet"`
	Metadata   SearchResultMeta `json:"metadata"`
}

// SearchResultMeta holds classification fields in search results.
type SearchResultMeta struct {
	Category     string   `json:"category"`
	DocumentType string   `json:"documentType"`
	PageType     string   `json:"pageType,omitempty"`
	Keywords     []string `json:"keywords,omitempty"`
}

// EmbedRequest is the input for POST /api/intelligence/embed.
type EmbedRequest struct {
	Texts []string `json:"texts"`
	Model string   `json:"model,omitempty"`
}

// TriggerRequest is the input for POST /api/intelligence/index/trigger.
type TriggerRequest struct {
	RunID     string     `json:"runId"`
	Documents []Document `json:"documents"`
	Source    string     `json:"source,omitempty"`
}

// IntelligenceLimits defines per-tier limits per INTELLIGENCE_LAYER.md.
type IntelligenceLimits struct {
	MaxDocs         int  `json:"maxDocs"`
	MaxDocsPerRun   int  `json:"maxDocsPerRun"`
	MaxQueryLength  int  `json:"maxQueryLength"`
	SearchLimit     int  `json:"searchLimit"`
	AllowEmbeddings bool `json:"allowEmbeddings"`
	EmbedDailyLimit int  `json:"embedDailyLimit"`
}

// TierLimits maps tier name → limits per INTELLIGENCE_LAYER.md spec.
var TierLimits = map[string]IntelligenceLimits{
	"free": {
		MaxDocs: 10, MaxDocsPerRun: 10, MaxQueryLength: 500,
		SearchLimit: 10, AllowEmbeddings: false, EmbedDailyLimit: 0,
	},
	"developer": {
		MaxDocs: 100, MaxDocsPerRun: 100, MaxQueryLength: 1000,
		SearchLimit: 20, AllowEmbeddings: true, EmbedDailyLimit: 1000,
	},
	"pro": {
		MaxDocs: 1000, MaxDocsPerRun: 1000, MaxQueryLength: 2000,
		SearchLimit: 50, AllowEmbeddings: true, EmbedDailyLimit: 10000,
	},
	"agency": {
		MaxDocs: 5000, MaxDocsPerRun: 5000, MaxQueryLength: 4000,
		SearchLimit: 100, AllowEmbeddings: true, EmbedDailyLimit: 50000,
	},
	"enterprise": {
		MaxDocs: 100000, MaxDocsPerRun: 100000, MaxQueryLength: 8000,
		SearchLimit: 200, AllowEmbeddings: true, EmbedDailyLimit: 1000000,
	},
	"api_client": {
		MaxDocs: 1000, MaxDocsPerRun: 1000, MaxQueryLength: 2000,
		SearchLimit: 50, AllowEmbeddings: true, EmbedDailyLimit: 10000,
	},
}

// DefaultLimits is used for unknown tiers.
var DefaultLimits = IntelligenceLimits{
	MaxDocs: 10, MaxDocsPerRun: 10, MaxQueryLength: 500,
	SearchLimit: 10, AllowEmbeddings: false, EmbedDailyLimit: 0,
}

// GetTierLimits returns limits for a given tier name.
func GetTierLimits(tier string) IntelligenceLimits {
	if l, ok := TierLimits[tier]; ok {
		return l
	}
	return DefaultLimits
}

// NowISO returns current time in ISO 8601 format.
func NowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}
