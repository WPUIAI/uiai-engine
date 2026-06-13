package routes

import (
	"encoding/json"
	"net/http"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/twofactor"
	"github.com/go-chi/chi/v5"
)

// MountTwoFactor registers portable 2FA helpers for agents. Secrets stay in
// config/env/rbw/Aegis vaults; responses expose only short-lived OTP codes and
// metadata needed for agents to complete an operator-approved login.
func MountTwoFactor(r chi.Router, cfg *config.Config) {
	svc := twofactor.New(cfg)
	r.Post("/code", func(w http.ResponseWriter, req *http.Request) {
		var body twofactor.Request
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		resp, err := svc.Code(req.Context(), body)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, resp)
	})
}
