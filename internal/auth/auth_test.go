package auth

import (
	"net/http"
	"testing"
)

func TestBrowserToolPathClassification(t *testing.T) {
	for _, path := range []string{"/api/session", "/api/session/abc/read", "/api/screenshot", "/api/screenshot/share"} {
		if !isBrowserToolPath(path) {
			t.Fatalf("expected browser tool path: %s", path)
		}
	}
	if isBrowserToolPath("/api/tools/graph") {
		t.Fatal("/api/tools should remain discovery, not browser tool path")
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
