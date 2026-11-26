package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
)

type stubInvalidator struct{ called int }

func (s *stubInvalidator) InvalidateCredentialCache() { s.called++ }

// #86: the broker revocation entry fences the cached credential snapshot
// on call; a missing provider is a typed refusal, never a silent success.
func TestCredentialRevocationRouteInvalidates(t *testing.T) {
	stub := &stubInvalidator{}
	router := chi.NewRouter()
	MountCredentialRevocationRoutes(router, stub)

	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, httptest.NewRequest("POST", "/credentials/invalidate", nil))
	if resp.Code != http.StatusOK || !stub.calledValid(1) {
		t.Fatalf("revocation did not fence the cache: status=%d calls=%d body=%.150s", resp.Code, stub.called, resp.Body.String())
	}

	missing := chi.NewRouter()
	MountCredentialRevocationRoutes(missing, nil)
	refused := httptest.NewRecorder()
	missing.ServeHTTP(refused, httptest.NewRequest("POST", "/credentials/invalidate", nil))
	if refused.Code != http.StatusServiceUnavailable {
		t.Fatalf("missing provider not typed-refused: status=%d", refused.Code)
	}
}

func (s *stubInvalidator) calledValid(n int) bool { return s.called == n }
