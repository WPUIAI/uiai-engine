package routes

import (
	"testing"
)

func TestValidateMediaURL(t *testing.T) {
	tests := []struct {
		url     string
		wantErr bool
		desc    string
	}{
		// Valid
		{"https://example.com", false, "normal HTTPS"},
		{"https://wpuiai.com/features", false, "WPUIAI page"},
		{"http://example.com", false, "HTTP allowed"},

		// Blocked schemes
		{"ftp://example.com", true, "FTP blocked"},
		{"file:///etc/passwd", true, "file:// blocked"},
		{"javascript:alert(1)", true, "javascript: blocked"},

		// Blocked hosts
		{"http://localhost", true, "localhost blocked"},
		{"http://127.0.0.1", true, "loopback blocked"},
		{"http://0.0.0.0", true, "0.0.0.0 blocked"},
		{"http://[::1]", true, "IPv6 loopback blocked"},
		{"http://metadata.google.internal", true, "cloud metadata blocked"},

		// Malformed
		{"", true, "empty string"},
		{"not-a-url", true, "no scheme"},
	}

	for _, tt := range tests {
		err := validateMediaURL(tt.url)
		if tt.wantErr && err == nil {
			t.Errorf("%s: expected error for %q, got nil", tt.desc, tt.url)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("%s: unexpected error for %q: %v", tt.desc, tt.url, err)
		}
	}
}

func TestValidateMediaURL_PrivateIPs(t *testing.T) {
	// These might fail in CI if DNS can't resolve, so skip selectively
	privateHosts := []string{
		"http://10.0.0.1",
		"http://172.16.0.1",
		"http://192.168.1.1",
	}
	for _, u := range privateHosts {
		err := validateMediaURL(u)
		if err == nil {
			t.Errorf("expected error for private IP %q, got nil", u)
		}
	}
}
