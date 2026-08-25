// C-010-03 — Budget governors: caps, pause/resume, typed denials.
package routes

import (
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/auth"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

var (
	ErrBudgetPaused   = errors.New("budget paused")
	ErrBudgetExceeded = errors.New("budget exceeded")
	ErrBudgetUnknown  = errors.New("unknown budget")
)

type BudgetLimits struct {
	TotalMS  int64 `json:"total_ms,omitempty"`
	MaxPages int   `json:"max_pages,omitempty"`
	MaxBytes int64 `json:"max_bytes,omitempty"`
}

type Budget struct {
	ID          string       `json:"id"`
	Limits      BudgetLimits `json:"limits"`
	Used        BudgetUsed   `json:"used"`
	Paused      bool         `json:"paused"`
	ResumeToken string       `json:"-"`
	CreatedAt   time.Time    `json:"created_at"`
	Owner       string       `json:"owner,omitempty"`
}

type BudgetUsed struct {
	MS    int64 `json:"ms"`
	Pages int   `json:"pages"`
	Bytes int64 `json:"bytes"`
	Steps int   `json:"steps"`
}

type budgetStore struct {
	mu      sync.RWMutex
	budgets map[string]*Budget
	tokens  map[string]string // resume token -> budget id
}

var budgets = &budgetStore{budgets: map[string]*Budget{}, tokens: map[string]string{}}

func CreateBudget(l BudgetLimits, owner string) *Budget {
	id := "bud_" + uuid.NewString()[:13]
	token := "res_" + uuid.NewString()
	b := &Budget{ID: id, Limits: l, ResumeToken: token, CreatedAt: time.Now().UTC(), Owner: owner}
	budgets.mu.Lock()
	budgets.budgets[id] = b
	budgets.tokens[token] = id
	budgets.mu.Unlock()
	return b
}

func GetBudget(id string) (*Budget, bool) {
	budgets.mu.RLock()
	defer budgets.mu.RUnlock()
	b, ok := budgets.budgets[id]
	return b, ok
}

// Charge records usage and returns the limiting violation, if any.
func Charge(id string, ms int64, pages int, bytes int64) error {
	budgets.mu.Lock()
	defer budgets.mu.Unlock()
	b, ok := budgets.budgets[id]
	if !ok {
		return ErrBudgetUnknown
	}
	if b.Paused {
		return ErrBudgetPaused
	}
	if b.Limits.TotalMS > 0 && b.Used.MS+ms > b.Limits.TotalMS {
		return ErrBudgetExceeded
	}
	if b.Limits.MaxPages > 0 && b.Used.Pages+pages > b.Limits.MaxPages {
		return ErrBudgetExceeded
	}
	if b.Limits.MaxBytes > 0 && b.Used.Bytes+bytes > b.Limits.MaxBytes {
		return ErrBudgetExceeded
	}
	b.Used.MS += ms
	b.Used.Pages += pages
	b.Used.Bytes += bytes
	b.Used.Steps++
	return nil
}

func SetPaused(id, token string, paused bool) (*Budget, bool) {
	budgets.mu.Lock()
	defer budgets.mu.Unlock()
	b, ok := budgets.budgets[id]
	if !ok || b.ResumeToken != token {
		return nil, false
	}
	b.Paused = paused
	return b, true
}

// ── HTTP surface ────────────────────────────────────────────────────────────

func MountBudgetRoutes(r chi.Router) {
	r.Post("/", func(w http.ResponseWriter, r *http.Request) {
		var body BudgetLimits
		_ = jsonDecode(r, &body)
		owner := ""
		if id := auth.FromContext(r.Context()); id != nil {
			owner = id.Tier
		}
		b := CreateBudget(body, owner)
		writeJSON(w, 201, map[string]any{"budget_id": b.ID, "resume_token": b.ResumeToken, "limits": b.Limits})
	})

	r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) {
		if b, ok := GetBudget(chi.URLParam(r, "id")); ok {
			writeJSON(w, 200, b)
			return
		}
		writeJSON(w, 404, map[string]string{"error": "unknown budget"})
	})

	r.Post("/{id}/pause", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Token string `json:"token"`
		}
		_ = jsonDecode(r, &body)
		if b, ok := SetPaused(chi.URLParam(r, "id"), body.Token, true); ok {
			writeJSON(w, 200, map[string]any{"paused": b.Paused, "used": b.Used})
			return
		}
		writeJSON(w, 403, map[string]string{"error": "invalid budget or token"})
	})

	r.Post("/{id}/resume", func(w http.ResponseWriter, r *http.Request) {
		var body struct {
			Token string `json:"token"`
		}
		_ = jsonDecode(r, &body)
		if b, ok := SetPaused(chi.URLParam(r, "id"), body.Token, false); ok {
			writeJSON(w, 200, map[string]any{"paused": b.Paused, "used": b.Used})
			return
		}
		writeJSON(w, 403, map[string]string{"error": "invalid budget or token"})
	})
}

// Guard enforces a budget before executing one step; writes the canonical
// envelope on denial and returns false.
func GuardBudget(w http.ResponseWriter, r *http.Request, id string, estMS int64, pages int, estBytes int64) (*Budget, bool) {
	b, ok := GetBudget(id)
	if !ok {
		writeJSON(w, 400, map[string]string{"error": "unknown_budget", "budget_id": id})
		return nil, false
	}
	if err := Charge(id, estMS, pages, estBytes); err != nil {
		code := http.StatusPaymentRequired
		reason := "budget_exceeded"
		if errors.Is(err, ErrBudgetPaused) {
			reason = "budget_paused"
		}
		if errors.Is(err, ErrBudgetUnknown) {
			code = http.StatusBadRequest
			reason = "unknown_budget"
		}
		writeJSON(w, code, map[string]any{
			"error": reason, "budget_id": id, "retry": reason == "budget_paused",
			"recovery": map[string]string{"paused": "POST /api/budget/{id}/resume with token"},
		})
		return nil, false
	}
	return b, true
}

func jsonDecode(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
