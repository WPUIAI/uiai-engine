package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

func TestLoopbackToolPathClassification(t *testing.T) {
	for _, path := range []string{"/api/session", "/api/session/abc/read", "/api/screenshot", "/api/screenshot/share", "/api/search", "/api/search/providers", "/api/markdown", "/api/markdown/", "/api/agent/research-packet", "/api/fpv/events"} {
		if !isLoopbackToolPath(path) {
			t.Fatalf("expected loopback tool path: %s", path)
		}
	}
	if isLoopbackToolPath("/api/tools/graph") {
		t.Fatal("/api/tools should remain discovery, not loopback tool path")
	}
}

func TestSearchToolPathLoopbackPublicRemoteAuth(t *testing.T) {
	authenticator := New(&config.Config{})
	hit := false
	handler := authenticator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusNoContent)
	}))

	loopbackReq := httptest.NewRequest("GET", "http://example.test/api/search/providers", nil)
	loopbackReq.RemoteAddr = "127.0.0.1:12345"
	loopbackRes := httptest.NewRecorder()
	handler.ServeHTTP(loopbackRes, loopbackReq)
	if loopbackRes.Code != http.StatusNoContent || !hit {
		t.Fatalf("expected loopback search request through, code=%d hit=%v", loopbackRes.Code, hit)
	}

	hit = false
	remoteReq := httptest.NewRequest("GET", "http://example.test/api/search/providers", nil)
	remoteReq.RemoteAddr = "203.0.113.10:12345"
	remoteRes := httptest.NewRecorder()
	handler.ServeHTTP(remoteRes, remoteReq)
	if remoteRes.Code != http.StatusUnauthorized || hit {
		t.Fatalf("expected remote search request unauthorized, code=%d hit=%v", remoteRes.Code, hit)
	}
}

func TestErrorsToolPathLoopbackPublicRemoteAuth(t *testing.T) {
	authenticator := New(&config.Config{})
	hit := false
	handler := authenticator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusNoContent)
	}))

	loopbackReq := httptest.NewRequest("GET", "http://example.test/api/errors", nil)
	loopbackReq.RemoteAddr = "127.0.0.1:12345"
	loopbackRes := httptest.NewRecorder()
	handler.ServeHTTP(loopbackRes, loopbackReq)
	if loopbackRes.Code != http.StatusNoContent || !hit {
		t.Fatalf("expected loopback errors request through, code=%d hit=%v", loopbackRes.Code, hit)
	}

	hit = false
	remoteReq := httptest.NewRequest("GET", "http://example.test/api/errors", nil)
	remoteReq.RemoteAddr = "203.0.113.10:12345"
	remoteRes := httptest.NewRecorder()
	handler.ServeHTTP(remoteRes, remoteReq)
	if remoteRes.Code != http.StatusUnauthorized || hit {
		t.Fatalf("expected remote errors request unauthorized, code=%d hit=%v", remoteRes.Code, hit)
	}
}

func TestMarkdownToolPathLoopbackPublicRemoteAuth(t *testing.T) {
	authenticator := New(&config.Config{})
	hit := false
	handler := authenticator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusNoContent)
	}))

	loopbackReq := httptest.NewRequest("POST", "http://example.test/api/markdown", nil)
	loopbackReq.RemoteAddr = "127.0.0.1:12345"
	loopbackRes := httptest.NewRecorder()
	handler.ServeHTTP(loopbackRes, loopbackReq)
	if loopbackRes.Code != http.StatusNoContent || !hit {
		t.Fatalf("expected loopback markdown request through, code=%d hit=%v", loopbackRes.Code, hit)
	}

	hit = false
	remoteReq := httptest.NewRequest("POST", "http://example.test/api/markdown", nil)
	remoteReq.RemoteAddr = "203.0.113.10:12345"
	remoteRes := httptest.NewRecorder()
	handler.ServeHTTP(remoteRes, remoteReq)
	if remoteRes.Code != http.StatusUnauthorized || hit {
		t.Fatalf("expected remote markdown request unauthorized, code=%d hit=%v", remoteRes.Code, hit)
	}
}

func TestAgentPacketToolPathLoopbackPublicRemoteAuth(t *testing.T) {
	authenticator := New(&config.Config{})
	hit := false
	handler := authenticator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusNoContent)
	}))

	loopbackReq := httptest.NewRequest("POST", "http://example.test/api/agent/research-packet", nil)
	loopbackReq.RemoteAddr = "127.0.0.1:12345"
	loopbackRes := httptest.NewRecorder()
	handler.ServeHTTP(loopbackRes, loopbackReq)
	if loopbackRes.Code != http.StatusNoContent || !hit {
		t.Fatalf("expected loopback agent packet request through, code=%d hit=%v", loopbackRes.Code, hit)
	}

	hit = false
	remoteReq := httptest.NewRequest("POST", "http://example.test/api/agent/research-packet", nil)
	remoteReq.RemoteAddr = "203.0.113.10:12345"
	remoteRes := httptest.NewRecorder()
	handler.ServeHTTP(remoteRes, remoteReq)
	if remoteRes.Code != http.StatusUnauthorized || hit {
		t.Fatalf("expected remote agent packet request unauthorized, code=%d hit=%v", remoteRes.Code, hit)
	}
}

func TestMediaFrameToolPathLoopbackPublicRemoteAuth(t *testing.T) {
	authenticator := New(&config.Config{})
	hit := false
	handler := authenticator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
		w.WriteHeader(http.StatusNoContent)
	}))

	loopbackReq := httptest.NewRequest("GET", "http://example.test/api/media/frame/catalog", nil)
	loopbackReq.RemoteAddr = "127.0.0.1:12345"
	loopbackRes := httptest.NewRecorder()
	handler.ServeHTTP(loopbackRes, loopbackReq)
	if loopbackRes.Code != http.StatusNoContent || !hit {
		t.Fatalf("expected loopback media frame request through, code=%d hit=%v", loopbackRes.Code, hit)
	}

	hit = false
	remoteReq := httptest.NewRequest("GET", "http://example.test/api/media/frame/catalog", nil)
	remoteReq.RemoteAddr = "203.0.113.10:12345"
	remoteRes := httptest.NewRecorder()
	handler.ServeHTTP(remoteRes, remoteReq)
	if remoteRes.Code != http.StatusUnauthorized || hit {
		t.Fatalf("expected remote media frame request unauthorized, code=%d hit=%v", remoteRes.Code, hit)
	}
}

func TestLoopbackToolPathsRemoteAuthPositive(t *testing.T) {
	t.Setenv("UIAI_LOCAL_API_TOKEN", "remote-smoke-token")
	for _, path := range []string{
		"/api/search/providers",
		"/api/errors",
		"/api/media/frame/catalog",
		"/api/session",
		"/api/screenshot",
		"/api/markdown",
		"/api/agent/research-packet",
	} {
		for _, tc := range []struct {
			name   string
			header string
			value  string
		}{
			{name: "api_key", header: "X-API-Key", value: "remote-smoke-token"},
			{name: "bearer", header: "Authorization", value: "Bearer remote-smoke-token"},
		} {
			t.Run(path+"/"+tc.name, func(t *testing.T) {
				authenticator := New(&config.Config{})
				hit := false
				handler := authenticator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					hit = true
					id := FromContext(r.Context())
					if id == nil || id.ClientID != "local-vps" || id.Tier != "internal" {
						t.Fatalf("unexpected auth identity: %#v", id)
					}
					w.WriteHeader(http.StatusNoContent)
				}))

				req := httptest.NewRequest(http.MethodGet, "http://example.test"+path, nil)
				req.RemoteAddr = "203.0.113.10:12345"
				req.Header.Set(tc.header, tc.value)
				res := httptest.NewRecorder()
				handler.ServeHTTP(res, req)
				if res.Code != http.StatusNoContent || !hit {
					t.Fatalf("expected remote authenticated request through, path=%s code=%d hit=%v", path, res.Code, hit)
				}
			})
		}
	}
}

func TestLoopbackRequestDetection(t *testing.T) {
	req, _ := http.NewRequest("GET", "http://example.test/api/session", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	if !isLoopbackRequest(req) {
		t.Fatal("expected 127.0.0.1 request to be loopback")
	}
	req.RemoteAddr = "203.0.113.10:12345"
	if isLoopbackRequest(req) {
		t.Fatal("expected public IP request to be non-loopback")
	}
}

func TestLocalAPITokenAuthenticatesAPIKeyAndBearer(t *testing.T) {
	t.Setenv("UIAI_LOCAL_API_TOKEN", "local-test-token")
	authenticator := New(&config.Config{})

	apiReq, _ := http.NewRequest("GET", "http://example.test/api/media/frame/catalog", nil)
	apiReq.Header.Set("X-API-Key", "local-test-token")
	apiID, err := authenticator.Authenticate(apiReq)
	if err != nil {
		t.Fatalf("expected X-API-Key local token to authenticate: %v", err)
	}
	if apiID.ClientID != "local-vps" || apiID.Tier != "internal" {
		t.Fatalf("unexpected local token identity: %#v", apiID)
	}

	bearerReq, _ := http.NewRequest("GET", "http://example.test/api/media/frame/catalog", nil)
	bearerReq.Header.Set("Authorization", "Bearer local-test-token")
	bearerID, err := authenticator.Authenticate(bearerReq)
	if err != nil {
		t.Fatalf("expected Bearer local token to authenticate: %v", err)
	}
	if bearerID.ClientID != "local-vps" || bearerID.Tier != "internal" {
		t.Fatalf("unexpected bearer token identity: %#v", bearerID)
	}
}

func TestLocalAPITokensSupportsCommaSeparatedFallbacks(t *testing.T) {
	t.Setenv("UIAI_LOCAL_API_TOKEN", "")
	t.Setenv("UIAI_LOCAL_API_TOKENS", "first-token, second-token")
	id, ok := validateLocalToken("second-token")
	if !ok {
		t.Fatal("expected comma-separated local token to authenticate")
	}
	if id.ClientID != "local-vps" || id.Tier != "internal" {
		t.Fatalf("unexpected local token identity: %#v", id)
	}
}
