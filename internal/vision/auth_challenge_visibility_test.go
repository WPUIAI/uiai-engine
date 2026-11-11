package vision

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// Real-browser fixtures: selectors must inspect all rendered matches, never values.
func TestAuthChallengeDetectorVisibleMarkers(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("requires UIAI_AUTH_TESTS=1")
	}
	cases := []struct{ name, html, want string }{
		{"hidden-password", `<input type="password" hidden>`, ""},
		{"hidden-ancestor", `<section style="display:none"><input type="password"></section>`, ""},
		{"invisible-password", `<input type="password" style="visibility:hidden">`, ""},
		{"transparent-ancestor", `<section style="opacity:0"><input type="password"></section>`, ""},
		{"hidden-login-form", `<form action="/login" hidden><input></form>`, ""},
		{"visible-password", `<input type="password">`, "credential_form"},
		{"hidden-first-visible-second", `<input type="password" hidden><input type="password">`, "credential_form"},
		{"visible-login-form", `<form action="/login"><input type="text"></form>`, "credential_form"},
		{"hidden-captcha", `<div data-sitekey="fixture" hidden></div>`, ""},
		{"visible-captcha", `<div data-sitekey="fixture" style="width:120px;height:50px">Challenge</div>`, "captcha"},
		{"hidden-password-visible-captcha", `<input type="password" hidden><div data-sitekey="fixture" style="width:120px;height:50px">Challenge</div>`, "captcha"},
		{"oidc-redirect", `<p>Authorization required</p>`, "oidc_redirect"},
		{"visible-security-key", `<p>Use your security key</p>`, "webauthn"},
		{"visible-payment-challenge", `<p>3-D Secure verification</p>`, "payment_sca"},
	}
	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	sm := NewSessionManager(pool)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/html")
				_, _ = w.Write([]byte("<!doctype html><html><body>" + tc.html + "</body></html>"))
			}))
			defer server.Close()
			path := "/fixture"
			if tc.name == "oidc-redirect" {
				path = "/authorize"
			}
			sess, _, err := sm.Open(server.URL+path, 800, 600)
			if err != nil {
				t.Fatal(err)
			}
			defer sm.Close(sess.ID)
			detected, err := sess.DetectAuthChallenge()
			if err != nil {
				t.Fatal(err)
			}
			got := ""
			if detected != nil {
				got = detected.Class
			}
			if got != tc.want {
				t.Fatalf("class=%q want=%q", got, tc.want)
			}
		})
	}
}
