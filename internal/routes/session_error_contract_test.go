package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func sessionContractTestRouter() http.Handler {
	router := chi.NewRouter()
	router.Route("/api/session", func(r chi.Router) {
		MountSessionRoutes(r, nil, &vision.SessionManager{})
	})
	return router
}

func TestDirectSessionErrorsUseStableRecoveryContract(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/session/missing/screenshot", nil)
	sessionContractTestRouter().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Code      string   `json:"code"`
		Retryable bool     `json:"retryable"`
		Recover   []string `json:"recover"`
		StateLost []string `json:"state_lost"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != "session_not_found" || response.Retryable || len(response.Recover) == 0 || len(response.StateLost) == 0 {
		t.Fatalf("incomplete session error contract: %+v", response)
	}
}

func TestAlreadyClosedResponseUsesStableRecoveryContract(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/api/session/missing", nil)
	sessionContractTestRouter().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Status    string   `json:"status"`
		Code      string   `json:"code"`
		Retryable bool     `json:"retryable"`
		Recover   []string `json:"recover"`
		StateLost []string `json:"state_lost"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Status != "already_closed" || response.Code != "already_closed" || response.Retryable || response.Recover == nil || response.StateLost == nil {
		t.Fatalf("incomplete already-closed contract: %+v", response)
	}
}
