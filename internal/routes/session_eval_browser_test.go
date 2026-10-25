package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/observability"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func TestEvaluationHTTPFailureAndLiteralPrefixParity(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for browser-backed HTTP evaluation")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<!doctype html><title>Evaluation fixture</title><p>owned fixture</p>`)
	}))
	defer site.Close()
	pool, err := vision.NewPoolWithConfig(vision.PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := vision.NewSessionManager(pool)
	defer manager.CloseAll()
	session, _, err := manager.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatal(err)
	}
	router := chi.NewRouter()
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	router.Route("/api/session", func(r chi.Router) { MountSessionRoutes(r, cfg, manager) })
	for _, route := range []string{"eval", "eval_async"} {
		t.Run(route, func(t *testing.T) {
			failure := httptest.NewRecorder()
			request := httptest.NewRequest("POST", "/api/session/"+session.ID+"/"+route, strings.NewReader(`{"js":"throw new Error('synthetic-private-eval-token')"}`))
			router.ServeHTTP(failure, request)
			var envelope observability.ErrorEnvelope
			if err := json.Unmarshal(failure.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if failure.Code != 409 || envelope.ErrorClass != "pending_side_effect_unknown" || envelope.Retryable {
				t.Fatalf("evaluation exception lost failure semantics: status=%d class=%s", failure.Code, envelope.ErrorClass)
			}
			if envelope.Details["side_effects_may_have_occurred"] != true || envelope.Details["cause_class"] != "javascript_exception" {
				t.Fatal("evaluation exception lost typed effect/cause clues")
			}
			if strings.Contains(failure.Body.String(), "synthetic-private-eval-token") {
				t.Fatal("HTTP error echoed submitted JavaScript exception content")
			}
			literal := httptest.NewRecorder()
			router.ServeHTTP(literal, httptest.NewRequest("POST", "/api/session/"+session.ID+"/"+route, strings.NewReader(`{"js":"return 'error: ordinary literal'"}`)))
			var result map[string]any
			if err := json.Unmarshal(literal.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if literal.Code != 200 && literal.Code != 202 {
				t.Fatalf("literal string falsely classified as an evaluation error: %d", literal.Code)
			}
			if result["result"] != "error: ordinary literal" {
				t.Fatal("HTTP result string was reinterpreted or lost")
			}
			if literal.Code == 202 && result["delivery_state"] == "ready" {
				t.Fatal("fixture claimed sealed ready evidence")
			}
			caller, cancel := context.WithCancel(context.Background())
			cancel()
			cancelled := httptest.NewRecorder()
			request = httptest.NewRequest("POST", "/api/session/"+session.ID+"/"+route, strings.NewReader(`{"js":"return true"}`)).WithContext(caller)
			router.ServeHTTP(cancelled, request)
			if err := json.Unmarshal(cancelled.Body.Bytes(), &envelope); err != nil {
				t.Fatal(err)
			}
			if cancelled.Code != 408 || envelope.ErrorClass != "operation_cancelled" || envelope.Retryable || envelope.Details["side_effects_may_have_occurred"] != false {
				t.Fatal("HTTP caller cancellation did not reach evaluation boundary")
			}
		})
	}
}
