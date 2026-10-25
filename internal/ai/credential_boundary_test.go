package ai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

type credentialBoundaryTransport func(*http.Request) (*http.Response, error)

func (f credentialBoundaryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type credentialBrokenBody struct{}

func (credentialBrokenBody) Read(p []byte) (int, error) {
	return 0, errors.New("synthetic-secret-body-error")
}
func (credentialBrokenBody) Close() error { return nil }
func boundaryProvider(endpoint string) *Provider {
	return &Provider{cfg: &config.Config{WordPress: config.WordPressConfig{URL: endpoint}}, client: &http.Client{Timeout: time.Second}}
}

func TestCredentialSettingsHonorsCompleteDeadline(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	p := boundaryProvider(server.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := p.Complete(ctx, Request{})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("credential request lost caller deadline: %v", err)
	}
	if time.Since(started) > time.Second {
		t.Fatal("credential lookup outlived caller deadline")
	}
}

func TestCredentialSettingsMalformedEndpointDoesNotPanicOrLeak(t *testing.T) {
	p := boundaryProvider("http://synthetic-secret.invalid/%zz")
	err := p.ensureKeys()
	if err == nil || strings.Contains(err.Error(), "synthetic-secret") {
		t.Fatal("invalid endpoint was accepted or exposed")
	}
}

func TestCredentialSettingsRejectsOversizeAndPartialBodies(t *testing.T) {
	for _, test := range []struct {
		name string
		body func() io.ReadCloser
	}{
		{"oversized", func() io.ReadCloser {
			return io.NopCloser(strings.NewReader(strings.Repeat("x", maxCredentialSettingsBytes+1)))
		}},
		{"partial", func() io.ReadCloser { return credentialBrokenBody{} }},
	} {
		t.Run(test.name, func(t *testing.T) {
			p := boundaryProvider("https://example.invalid")
			p.client.Transport = credentialBoundaryTransport(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Body: test.body(), Header: make(http.Header)}, nil
			})
			err := p.ensureKeys()
			if err == nil {
				t.Fatal("incomplete credential response accepted")
			}
			if strings.Contains(err.Error(), "synthetic-secret") {
				t.Fatal("credential read error escaped")
			}
			if !p.keys.fetchedAt.IsZero() {
				t.Fatal("rejected response populated credential cache")
			}
		})
	}
}

func TestCredentialSettingsTransportErrorsCannotEchoSecrets(t *testing.T) {
	p := boundaryProvider("https://synthetic-secret.invalid")
	p.client.Transport = credentialBoundaryTransport(func(*http.Request) (*http.Response, error) { return nil, errors.New("synthetic-secret-header") })
	err := p.ensureKeys()
	if err == nil || strings.Contains(err.Error(), "synthetic-secret") {
		t.Fatal("transport exposed endpoint or credential material")
	}
}

func TestCredentialSettingsCanceledContextNeverUsesTransport(t *testing.T) {
	p := boundaryProvider("https://example.invalid")
	called := false
	p.client.Transport = credentialBoundaryTransport(func(*http.Request) (*http.Response, error) {
		called = true
		return nil, errors.New("unexpected transport")
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is(p.ensureKeysContext(ctx), context.Canceled) || called {
		t.Fatal("canceled lookup reached transport")
	}
}
