package routes

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/config"
)

type memoryEntry struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type userMemory struct {
	Conversation []memoryEntry  `json:"conversation"`
	Preferences  map[string]any `json:"preferences"`
	Context      map[string]any `json:"context"`
}

// persistentMemStore uses file-backed storage for memory persistence.
type persistentMemStore struct {
	dataDir string
	cache   sync.Map // userId → *userMemory
}

var memBackend *persistentMemStore

func initMemStore(dataDir string) {
	memBackend = &persistentMemStore{
		dataDir: filepath.Join(dataDir, "memory"),
	}
	os.MkdirAll(memBackend.dataDir, 0755)
}

func (s *persistentMemStore) get(userId string) *userMemory {
	// Check cache
	if v, ok := s.cache.Load(userId); ok {
		return v.(*userMemory)
	}

	// Try loading from disk
	path := filepath.Join(s.dataDir, userId+".json")
	data, err := os.ReadFile(path)
	if err == nil {
		m := &userMemory{Preferences: map[string]any{}, Context: map[string]any{}}
		if json.Unmarshal(data, m) == nil {
			s.cache.Store(userId, m)
			return m
		}
	}

	// Create new
	m := &userMemory{
		Preferences: map[string]any{},
		Context:     map[string]any{},
	}
	s.cache.Store(userId, m)
	return m
}

func (s *persistentMemStore) save(userId string) {
	v, ok := s.cache.Load(userId)
	if !ok {
		return
	}
	m := v.(*userMemory)
	data, _ := json.MarshalIndent(m, "", "  ")
	os.WriteFile(filepath.Join(s.dataDir, userId+".json"), data, 0644)
}

func (s *persistentMemStore) delete(userId string) {
	s.cache.Delete(userId)
	os.Remove(filepath.Join(s.dataDir, userId+".json"))
}

func MountMemoryReal(r chi.Router, cfg *config.Config) {
	initMemStore(cfg.Storage.DataDir)

	r.Get("/{userId}", func(w http.ResponseWriter, req *http.Request) {
		m := memBackend.get(chi.URLParam(req, "userId"))
		writeJSON(w, 200, m)
	})

	r.Get("/{userId}/stats", func(w http.ResponseWriter, req *http.Request) {
		m := memBackend.get(chi.URLParam(req, "userId"))
		writeJSON(w, 200, map[string]any{
			"conversation_count": len(m.Conversation),
			"has_preferences":    len(m.Preferences) > 0,
			"has_context":        len(m.Context) > 0,
		})
	})

	r.Get("/{userId}/conversation", func(w http.ResponseWriter, req *http.Request) {
		m := memBackend.get(chi.URLParam(req, "userId"))
		writeJSON(w, 200, map[string]any{"messages": m.Conversation})
	})

	r.Post("/{userId}/conversation", func(w http.ResponseWriter, req *http.Request) {
		userId := chi.URLParam(req, "userId")
		m := memBackend.get(userId)
		var entry memoryEntry
		json.NewDecoder(req.Body).Decode(&entry)
		m.Conversation = append(m.Conversation, entry)
		if len(m.Conversation) > 200 {
			m.Conversation = m.Conversation[len(m.Conversation)-200:]
		}
		memBackend.save(userId)
		writeJSON(w, 200, map[string]any{"added": true, "count": len(m.Conversation)})
	})

	r.Delete("/{userId}/conversation", func(w http.ResponseWriter, req *http.Request) {
		userId := chi.URLParam(req, "userId")
		m := memBackend.get(userId)
		m.Conversation = nil
		memBackend.save(userId)
		writeJSON(w, 200, map[string]string{"message": "conversation cleared"})
	})

	r.Get("/{userId}/preferences", func(w http.ResponseWriter, req *http.Request) {
		m := memBackend.get(chi.URLParam(req, "userId"))
		writeJSON(w, 200, m.Preferences)
	})

	r.Put("/{userId}/preferences", func(w http.ResponseWriter, req *http.Request) {
		userId := chi.URLParam(req, "userId")
		m := memBackend.get(userId)
		json.NewDecoder(req.Body).Decode(&m.Preferences)
		memBackend.save(userId)
		writeJSON(w, 200, map[string]string{"message": "preferences updated"})
	})

	r.Get("/{userId}/context", func(w http.ResponseWriter, req *http.Request) {
		m := memBackend.get(chi.URLParam(req, "userId"))
		writeJSON(w, 200, m.Context)
	})

	r.Put("/{userId}/context", func(w http.ResponseWriter, req *http.Request) {
		userId := chi.URLParam(req, "userId")
		m := memBackend.get(userId)
		json.NewDecoder(req.Body).Decode(&m.Context)
		memBackend.save(userId)
		writeJSON(w, 200, map[string]string{"message": "context updated"})
	})

	r.Delete("/{userId}", func(w http.ResponseWriter, req *http.Request) {
		memBackend.delete(chi.URLParam(req, "userId"))
		writeJSON(w, 200, map[string]string{"message": "memory cleared"})
	})

	r.Delete("/{userId}/all", func(w http.ResponseWriter, req *http.Request) {
		memBackend.delete(chi.URLParam(req, "userId"))
		writeJSON(w, 200, map[string]string{"message": "all memory cleared"})
	})

	// List all stored memories (admin)
	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		entries, _ := os.ReadDir(memBackend.dataDir)
		var users []string
		for _, e := range entries {
			if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
				users = append(users, e.Name()[:len(e.Name())-5])
			}
		}
		writeJSON(w, 200, map[string]any{"users": users, "count": len(users)})
	})
}
