package ai

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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

func TestCredentialSettingsRedirectCannotForwardWebhookSecret(t *testing.T) {
	var received atomic.Int32
	destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		received.Add(1)
		if r.Header.Get("X-Webhook-Secret") != "" {
			t.Error("unexpected credential header reached redirect destination")
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}))
	defer destination.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, destination.URL, http.StatusFound) }))
	defer source.Close()
	p := boundaryProvider(source.URL)
	p.cfg.WordPress.WebhookSecret = "synthetic-webhook-fixture"
	_ = p.ensureKeys()
	if received.Load() != 0 {
		t.Fatal("credential refresh followed a redirect outside the approved origin")
	}
}

func TestCredentialSettingsSameOriginRedirectPreservesPolicy(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/fixture-settings" {
			http.Redirect(w, r, "/fixture-settings", http.StatusFound)
			return
		}
		calls++
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{}`)
	}))
	defer server.Close()
	p := boundaryProvider(server.URL)
	policyCalls := 0
	p.client.CheckRedirect = func(*http.Request, []*http.Request) error { policyCalls++; return nil }
	if err := p.ensureKeys(); err != nil {
		t.Fatal(err)
	}
	if calls != 1 || policyCalls != 1 {
		t.Fatal("same-origin redirect or existing client policy regressed")
	}
}

func TestCredentialSettingsRejectsSchemeAndUserinfoRedirects(t *testing.T) {
	for _, tc := range []struct{ name, location string }{{"downgrade", "http://approved.invalid/next"}, {"userinfo", "https://synthetic-user:synthetic-password@approved.invalid/next"}} {
		t.Run(tc.name, func(t *testing.T) {
			p := boundaryProvider("https://approved.invalid")
			calls := 0
			p.client.Transport = credentialBoundaryTransport(func(*http.Request) (*http.Response, error) {
				calls++
				return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": []string{tc.location}}, Body: io.NopCloser(strings.NewReader(""))}, nil
			})
			_ = p.ensureKeys()
			if calls != 1 {
				t.Fatal("credential redirect widened authority scope")
			}
			if p.client.CheckRedirect != nil {
				t.Fatal("credential refresh mutated shared inference redirect policy")
			}
		})
	}
}

func TestCredentialSettingsRetainsDefaultRedirectBound(t *testing.T) {
	p := boundaryProvider("https://approved.invalid")
	calls := 0
	p.client.Transport = credentialBoundaryTransport(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: http.StatusFound, Header: http.Header{"Location": []string{"https://approved.invalid/loop"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	if err := p.ensureKeys(); err == nil {
		t.Fatal("credential redirect loop accepted")
	}
	if calls != 10 {
		t.Fatalf("default redirect bound changed: %d requests", calls)
	}
}
