package routes

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/philoveracity/uiai-engine/internal/auth"
	"github.com/philoveracity/uiai-engine/internal/config"
)

type extToken struct {
	Token     string    `json:"token"`
	UserID    string    `json:"userId"`
	SiteURL   string    `json:"siteUrl"`
	Platform  string    `json:"platform"`
	Scope     []string  `json:"scope"`
	CreatedAt time.Time `json:"createdAt"`
	ExpiresAt time.Time `json:"expiresAt"`
}

var (
	extTokens   sync.Map // token string → *extToken
)

func MountExtensionReal(r chi.Router, cfg *config.Config, authenticator *auth.Authenticator) {
	r.Post("/token", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			LicenseKey  string `json:"licenseKey"`
			APIKey      string `json:"apiKey"`
			UserID      string `json:"userId"`
			SiteURL     string `json:"siteUrl"`
			ExtensionID string `json:"extensionId"`
			Platform    string `json:"platform"`
		}
		if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
			writeJSON(w, 400, map[string]string{"error": "invalid JSON"})
			return
		}
		if body.ExtensionID == "" {
			writeJSON(w, 400, map[string]string{"error": "extensionId required"})
			return
		}
		if body.Platform == "" {
			body.Platform = "chrome"
		}

		// Validate credentials
		fakeReq, _ := http.NewRequest("GET", "/", nil)
		if body.LicenseKey != "" {
			fakeReq.Header.Set("X-License-Key", body.LicenseKey)
		} else if body.APIKey != "" {
			fakeReq.Header.Set("X-API-Key", body.APIKey)
		} else {
			writeJSON(w, 400, map[string]string{"error": "licenseKey or apiKey required"})
			return
		}

		id, err := authenticator.Authenticate(fakeReq)
		if err != nil {
			writeJSON(w, 401, map[string]string{"error": err.Error()})
			return
		}

		// Generate token
		tokenBytes := make([]byte, 32)
		rand.Read(tokenBytes)
		token := hex.EncodeToString(tokenBytes)

		et := &extToken{
			Token:     token,
			UserID:    body.UserID,
			SiteURL:   body.SiteURL,
			Platform:  body.Platform,
			Scope:     []string{"critique", "ui-reverse", "screenshot", "memory", "copilot"},
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(24 * time.Hour),
		}
		extTokens.Store(token, et)

		writeJSON(w, 200, map[string]any{
			"token":     token,
			"tokenType": "Bearer",
			"expiresIn": 86400,
			"user": map[string]any{
				"id":      body.UserID,
				"siteUrl": body.SiteURL,
				"tier":    id.Tier,
			},
			"scope": et.Scope,
		})
	})

	r.Get("/verify", func(w http.ResponseWriter, req *http.Request) {
		token := req.Header.Get("X-Extension-Token")
		if token == "" {
			token = req.URL.Query().Get("token")
		}
		if v, ok := extTokens.Load(token); ok {
			et := v.(*extToken)
			if time.Now().Before(et.ExpiresAt) {
				writeJSON(w, 200, map[string]any{"valid": true, "userId": et.UserID, "scope": et.Scope, "expiresAt": et.ExpiresAt})
				return
			}
			extTokens.Delete(token)
		}
		writeJSON(w, 401, map[string]any{"valid": false, "error": "invalid or expired token"})
	})

	r.Delete("/token", func(w http.ResponseWriter, req *http.Request) {
		token := req.Header.Get("X-Extension-Token")
		extTokens.Delete(token)
		writeJSON(w, 200, map[string]string{"message": "token revoked"})
	})

	r.Get("/rate-limits", func(w http.ResponseWriter, req *http.Request) {
		writeJSON(w, 200, map[string]any{
			"tiers": cfg.RateLimits.Tiers,
		})
	})
}
