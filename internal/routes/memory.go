package routes

import (
	"encoding/json"
	"net/http"
	"sync"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/config"
)

type memoryEntry struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type userMemory struct {
	Conversation []memoryEntry          `json:"conversation"`
	Preferences  map[string]any         `json:"preferences"`
	Context      map[string]any         `json:"context"`
}

var memStore sync.Map // userId → *userMemory

func getMemory(userId string) *userMemory {
	v, _ := memStore.LoadOrStore(userId, &userMemory{
		Preferences: map[string]any{},
		Context:     map[string]any{},
	})
	return v.(*userMemory)
}

func MountMemoryReal(r chi.Router, _ *config.Config) {
	r.Get("/{userId}", func(w http.ResponseWriter, req *http.Request) {
		m := getMemory(chi.URLParam(req, "userId"))
		writeJSON(w, 200, m)
	})

	r.Get("/{userId}/stats", func(w http.ResponseWriter, req *http.Request) {
		m := getMemory(chi.URLParam(req, "userId"))
		writeJSON(w, 200, map[string]any{"conversation_count": len(m.Conversation), "has_preferences": len(m.Preferences) > 0, "has_context": len(m.Context) > 0})
	})

	r.Get("/{userId}/conversation", func(w http.ResponseWriter, req *http.Request) {
		m := getMemory(chi.URLParam(req, "userId"))
		writeJSON(w, 200, map[string]any{"messages": m.Conversation})
	})

	r.Post("/{userId}/conversation", func(w http.ResponseWriter, req *http.Request) {
		m := getMemory(chi.URLParam(req, "userId"))
		var entry memoryEntry
		json.NewDecoder(req.Body).Decode(&entry)
		m.Conversation = append(m.Conversation, entry)
		if len(m.Conversation) > 50 {
			m.Conversation = m.Conversation[len(m.Conversation)-50:]
		}
		writeJSON(w, 200, map[string]any{"added": true, "count": len(m.Conversation)})
	})

	r.Delete("/{userId}/conversation", func(w http.ResponseWriter, req *http.Request) {
		m := getMemory(chi.URLParam(req, "userId"))
		m.Conversation = nil
		writeJSON(w, 200, map[string]string{"message": "conversation cleared"})
	})

	r.Get("/{userId}/preferences", func(w http.ResponseWriter, req *http.Request) {
		m := getMemory(chi.URLParam(req, "userId"))
		writeJSON(w, 200, m.Preferences)
	})

	r.Put("/{userId}/preferences", func(w http.ResponseWriter, req *http.Request) {
		m := getMemory(chi.URLParam(req, "userId"))
		json.NewDecoder(req.Body).Decode(&m.Preferences)
		writeJSON(w, 200, map[string]string{"message": "preferences updated"})
	})

	r.Get("/{userId}/context", func(w http.ResponseWriter, req *http.Request) {
		m := getMemory(chi.URLParam(req, "userId"))
		writeJSON(w, 200, m.Context)
	})

	r.Put("/{userId}/context", func(w http.ResponseWriter, req *http.Request) {
		m := getMemory(chi.URLParam(req, "userId"))
		json.NewDecoder(req.Body).Decode(&m.Context)
		writeJSON(w, 200, map[string]string{"message": "context updated"})
	})

	r.Delete("/{userId}", func(w http.ResponseWriter, req *http.Request) {
		memStore.Delete(chi.URLParam(req, "userId"))
		writeJSON(w, 200, map[string]string{"message": "memory cleared"})
	})

	r.Delete("/{userId}/all", func(w http.ResponseWriter, req *http.Request) {
		memStore.Delete(chi.URLParam(req, "userId"))
		writeJSON(w, 200, map[string]string{"message": "all memory cleared"})
	})
}
