package evidenceshare

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const SettingsSchema = "uiai.evidence_share_settings.v1"

var (
	ErrSettingsConflict = errors.New("evidence share settings revision conflict")
	ErrSettingsInvalid  = errors.New("invalid evidence share settings")
)

type SettingsScope struct{ ProjectRef, WorkstreamRef string }
type SettingsRecord struct {
	Scope     SettingsScope  `json:"scope"`
	Revision  uint64         `json:"revision"`
	UpdatedAt time.Time      `json:"updated_at"`
	Values    map[string]any `json:"values"`
}
type SettingsDocument struct {
	Schema  string           `json:"schema"`
	Records []SettingsRecord `json:"records"`
}
type SettingsResponse struct {
	Schema   string         `json:"schema"`
	Scope    SettingsScope  `json:"scope"`
	Revision uint64         `json:"revision"`
	Sources  []string       `json:"sources"`
	Values   map[string]any `json:"values"`
	Warnings []string       `json:"warnings,omitempty"`
}
type SettingsStore struct {
	mu   sync.Mutex
	path string
	doc  SettingsDocument
}

func DefaultSettings() map[string]any {
	return map[string]any{
		"enablement":   map[string]any{"enabled": true, "auto_screenshot": true, "auto_video": true, "read_only": true},
		"lifecycle":    map[string]any{"retention_days": 0, "max_packets": 1000, "max_bytes": 0, "pinning": true, "archive_before_expire": true, "legal_hold": false},
		"storage":      map[string]any{"location_template": "{project}/{workstream}", "portable_exports": true, "quota_headroom_percent": 15},
		"image":        map[string]any{"format": "webp", "quality": 82, "max_width": 2400, "max_height": 1600, "thumbnail_width": 720, "strip_metadata": true},
		"video":        map[string]any{"format": "webm", "quality": 75, "max_duration_seconds": 300, "poster_frame": true, "captions": true},
		"presentation": map[string]any{"theme": "system", "accent": "sage", "density": "comfortable", "frame": "editorial", "motion": "reduced"},
		"access":       map[string]any{"posture": "private", "strip_query": true, "redaction_required": true, "download": true},
		"verification": map[string]any{"auto_verify": true, "verify_on_open": true, "stale_after_hours": 24},
		"performance":  map[string]any{"lazy_load": true, "thumbnail_budget": 24, "list_page_size": 25, "offline_copy": true},
		"integrations": map[string]any{"pi": true, "chrome": true, "cockpit": true, "desktop_canvas": true},
	}
}

func NewSettingsStore(dataDir string) (*SettingsStore, error) {
	if dataDir == "" {
		return nil, fmt.Errorf("%w: empty data directory", ErrSettingsInvalid)
	}
	if err := os.MkdirAll(dataDir, 0750); err != nil {
		return nil, err
	}
	s := &SettingsStore{path: filepath.Join(dataDir, "evidence-share-settings.json"), doc: SettingsDocument{Schema: SettingsSchema}}
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &s.doc); err != nil || s.doc.Schema != SettingsSchema {
		return nil, fmt.Errorf("%w: malformed settings document", ErrSettingsInvalid)
	}
	return s, nil
}

func (s *SettingsStore) Effective(scope SettingsScope) SettingsResponse {
	s.mu.Lock()
	defer s.mu.Unlock()
	values := cloneMap(DefaultSettings())
	sources := []string{"defaults"}
	for _, r := range s.doc.Records {
		if r.Scope.ProjectRef == "" && r.Scope.WorkstreamRef == "" {
			values = mergeMap(values, r.Values)
			sources = append(sources, "global")
		}
	}
	for _, r := range s.doc.Records {
		if r.Scope.ProjectRef == scope.ProjectRef && scope.ProjectRef != "" && r.Scope.WorkstreamRef == "" {
			values = mergeMap(values, r.Values)
			sources = append(sources, "project")
		}
	}
	var revision uint64
	for _, r := range s.doc.Records {
		if r.Scope == scope {
			values = mergeMap(values, r.Values)
			revision = r.Revision
			sources = append(sources, "workstream")
		}
	}
	return SettingsResponse{Schema: SettingsSchema, Scope: scope, Revision: revision, Sources: sources, Values: values}
}

func (s *SettingsStore) Preview(scope SettingsScope, patch map[string]any) (SettingsResponse, error) {
	if err := validatePatch(patch); err != nil {
		return SettingsResponse{}, err
	}
	r := s.Effective(scope)
	r.Values = mergeMap(r.Values, patch)
	r.Warnings = warnings(r.Values)
	return r, nil
}
func (s *SettingsStore) Update(scope SettingsScope, expected uint64, patch map[string]any) (SettingsResponse, error) {
	if err := validatePatch(patch); err != nil {
		return SettingsResponse{}, err
	}
	s.mu.Lock()
	idx := -1
	for i := range s.doc.Records {
		if s.doc.Records[i].Scope == scope {
			idx = i
			break
		}
	}
	current := uint64(0)
	if idx >= 0 {
		current = s.doc.Records[idx].Revision
	}
	if current != expected {
		s.mu.Unlock()
		return SettingsResponse{}, ErrSettingsConflict
	}
	if idx < 0 {
		s.doc.Records = append(s.doc.Records, SettingsRecord{Scope: scope, Values: map[string]any{}})
		idx = len(s.doc.Records) - 1
	}
	s.doc.Records[idx].Values = mergeMap(s.doc.Records[idx].Values, patch)
	s.doc.Records[idx].Revision++
	s.doc.Records[idx].UpdatedAt = time.Now().UTC()
	if err := s.persistLocked(); err != nil {
		s.mu.Unlock()
		return SettingsResponse{}, err
	}
	s.mu.Unlock()
	return s.Effective(scope), nil
}
func (s *SettingsStore) Reset(scope SettingsScope, expected uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, r := range s.doc.Records {
		if r.Scope == scope {
			if r.Revision != expected {
				return ErrSettingsConflict
			}
			s.doc.Records = append(s.doc.Records[:i], s.doc.Records[i+1:]...)
			return s.persistLocked()
		}
	}
	if expected != 0 {
		return ErrSettingsConflict
	}
	return nil
}
func (s *SettingsStore) persistLocked() error {
	data, err := json.MarshalIndent(s.doc, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}
func SettingsDigest(values map[string]any) string {
	b, _ := json.Marshal(values)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func validatePatch(p map[string]any) error {
	allowed := map[string]bool{"enablement": true, "lifecycle": true, "storage": true, "image": true, "video": true, "presentation": true, "access": true, "verification": true, "performance": true, "integrations": true}
	for k, v := range p {
		if !allowed[k] {
			return fmt.Errorf("%w: unknown field %q", ErrSettingsInvalid, k)
		}
		m, ok := v.(map[string]any)
		if !ok || len(m) == 0 {
			return fmt.Errorf("%w: %s must be a non-empty object", ErrSettingsInvalid, k)
		}
	}
	return nil
}
func warnings(v map[string]any) []string {
	var out []string
	if m, ok := v["lifecycle"].(map[string]any); ok {
		if n, ok := m["retention_days"].(float64); ok && n > 0 && n < 7 {
			out = append(out, "retention under seven days requires explicit lifecycle review")
		}
	}
	return out
}
func cloneMap(in map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range in {
		if m, ok := v.(map[string]any); ok {
			out[k] = cloneMap(m)
		} else {
			out[k] = v
		}
	}
	return out
}
func mergeMap(base, patch map[string]any) map[string]any {
	out := cloneMap(base)
	for k, v := range patch {
		if m, ok := v.(map[string]any); ok {
			bm, _ := out[k].(map[string]any)
			out[k] = mergeMap(bm, m)
		} else {
			out[k] = v
		}
	}
	return out
}
