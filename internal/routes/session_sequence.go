package routes

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func mountSessionSequenceRoute(r chi.Router, cfg *config.Config, manager *vision.SessionManager) {
	r.Post("/sequence", func(w http.ResponseWriter, req *http.Request) {
		session, ok := manager.Get(chi.URLParam(req, "sessionID"))
		if !ok {
			writeJSON(w, 404, map[string]string{"error": "session not found"})
			return
		}
		var body *struct {
			Operations    []vision.SequenceOperation `json:"operations"`
			TotalBudgetMS int                        `json:"total_budget_ms"`
		}
		if err := decodeStrictJSON(req, &body); err != nil || body == nil {
			writeJSON(w, 400, map[string]string{"error": "invalid sequence request"})
			return
		}
		// Validate before duration conversion, including integer overflow inputs.
		if body.TotalBudgetMS < 1 || body.TotalBudgetMS > 10000 {
			writeJSON(w, 400, map[string]string{"error": "sequence budget must be 1..10000ms"})
			return
		}
		snapshot, result, err := session.Sequence(req.Context(), body.Operations, time.Duration(body.TotalBudgetMS)*time.Millisecond)
		if err != nil {
			status := http.StatusUnprocessableEntity
			cause := "action_failed"
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				status = http.StatusRequestTimeout
				cause = "deadline_or_cancellation"
			}
			if session.LeaseState() == vision.LeaseAwaitingAuth {
				status = http.StatusConflict
				cause = "auth_challenge_pending"
			}
			writeJSON(w, status, map[string]any{"error": "sequence_incomplete", "cause": cause, "sequence": result, "atomic": false, "rollback_performed": false})
			return
		}
		writeSessionSnapshot(w, req, cfg, session, snapshot, http.StatusOK, map[string]any{"sequence": result, "atomic": false, "rollback_performed": false})
	})
}
