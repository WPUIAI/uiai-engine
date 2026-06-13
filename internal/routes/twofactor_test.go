package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/go-chi/chi/v5"
)

func TestTwoFactorRouteDisabled(t *testing.T) {
	r := chi.NewRouter()
	MountTwoFactor(r, &config.Config{})
	req := httptest.NewRequest(http.MethodPost, "/code", bytes.NewBufferString(`{"profile":"demo"}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTwoFactorRouteTOTP(t *testing.T) {
	r := chi.NewRouter()
	MountTwoFactor(r, &config.Config{TwoFactor: config.TwoFactorConfig{Enabled: true, Profiles: map[string]config.TwoFactorProfile{
		"demo": {Provider: "totp", Secret: "JBSWY3DPEHPK3PXP"},
	}}})
	req := httptest.NewRequest(http.MethodPost, "/code", bytes.NewBufferString(`{"profile":"demo"}`))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", w.Code, w.Body.String())
	}
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp["code"] == "" || resp["provider"] != "totp" {
		t.Fatalf("unexpected response: %#v", resp)
	}
}
