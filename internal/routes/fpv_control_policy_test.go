package routes

import (
	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFPVPublicShareCannotSelfAuthorizeControls(t *testing.T) {
	pool, err := vision.NewPoolWithConfig(vision.PoolConfig{MaxPages: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := vision.NewSessionManager(pool)
	router := chi.NewRouter()
	MountFPVRoutes(router, &config.Config{}, manager)
	for _, body := range []string{`{"session_id":"absent-fixture","controls":true}`, `{"session_id":"absent-fixture","controls":true,"confirm":true,"operator_confirmed":true}`} {
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/share", strings.NewReader(body)))
		if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "fpv_control_approval_required") {
			t.Fatal("caller-controlled booleans authorized a public control share")
		}
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/share", strings.NewReader(`{"session_id":`)))
	if response.Code != http.StatusBadRequest {
		t.Fatal("malformed share request was not rejected")
	}
}
