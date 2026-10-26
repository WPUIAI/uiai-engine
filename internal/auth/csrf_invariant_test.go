package auth

import (
	"github.com/WPUIAI/uiai-engine/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestMutationRoutesRejectCookieOnlyIdentity(t *testing.T) {
	const token = "fixture-local-header-only-token"
	t.Setenv("UIAI_LOCAL_API_TOKEN", token)
	t.Setenv("UIAI_LOCAL_API_TOKENS", "")
	a := &Authenticator{cfg: &config.Config{}}
	for _, path := range []string{"/api/session/fixture/click", "/api/session/fixture/type", "/api/screenshot", "/api/review/fixture"} {
		for _, method := range []string{"POST", "PUT", "PATCH", "DELETE"} {
			for _, bearer := range []bool{false, true} {
				called := false
				h := a.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					called = true
					if FromContext(r.Context()) == nil {
						t.Fatal("missing header-verified identity")
					}
					w.WriteHeader(204)
				}))
				r := httptest.NewRequest(method, path, nil)
				r.RemoteAddr = "198.51.100.23:12345"
				r.Header.Set("Origin", "https://untrusted.example.invalid")
				r.AddCookie(&http.Cookie{Name: "token", Value: token})
				r.AddCookie(&http.Cookie{Name: "Authorization", Value: "Bearer " + token})
				r.AddCookie(&http.Cookie{Name: "session", Value: token})
				if bearer {
					r.Header.Set("Authorization", "Bearer "+token)
				}
				out := httptest.NewRecorder()
				h.ServeHTTP(out, r)
				want := 401
				if bearer {
					want = 204
				}
				if out.Code != want || called != bearer {
					t.Fatalf("method=%s path=%s bearer=%v status=%d called=%v", method, path, bearer, out.Code, called)
				}
			}
		}
	}
	// The existing loopback tool exemption is separate from cookie authentication; no widening is claimed here.
}
