package twofactor

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

func TestGenerateRFC6238SHA1(t *testing.T) {
	code, err := Generate("GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", "SHA1", 8, 30, time.Unix(59, 0))
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if code.Code != "94287082" {
		t.Fatalf("unexpected code: %s", code.Code)
	}
}

func TestServiceTOTPProfile(t *testing.T) {
	cfg := &config.Config{TwoFactor: config.TwoFactorConfig{Enabled: true, Profiles: map[string]config.TwoFactorProfile{
		"demo": {Provider: "totp", Secret: "GEZDGNBVGY3TQOJQGEZDGNBVGY3TQOJQ", Digits: 8, Period: 30},
	}}}
	svc := New(cfg)
	svc.Now = func() time.Time { return time.Unix(59, 0) }
	resp, err := svc.Code(context.Background(), Request{Profile: "demo"})
	if err != nil {
		t.Fatalf("Code returned error: %v", err)
	}
	if resp.Code != "94287082" || resp.Provider != "totp" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestServiceCommandProvider(t *testing.T) {
	cmd := fakeAegisRS(t)
	cfg := &config.Config{TwoFactor: config.TwoFactorConfig{Enabled: true, Profiles: map[string]config.TwoFactorProfile{
		"github": {Provider: "aegis", Command: cmd, VaultFile: "/tmp/vault.json", Password: "secret", Issuer: "GitHub", Name: "agent@example.test"},
	}}}
	svc := New(cfg)
	resp, err := svc.Code(context.Background(), Request{Profile: "github"})
	if err != nil {
		t.Fatalf("Code returned error: %v", err)
	}
	if resp.Code != "123456" || resp.ExpiresIn != 17 || resp.Issuer != "GitHub" {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestServiceDisabled(t *testing.T) {
	_, err := New(&config.Config{}).Code(context.Background(), Request{Profile: "demo"})
	if err == nil {
		t.Fatal("expected disabled error")
	}
}

func fakeAegisRS(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	name := "aegis"
	if runtime.GOOS == "windows" {
		name = "aegis.bat"
	}
	path := filepath.Join(dir, name)
	var script string
	if runtime.GOOS == "windows" {
		script = "@echo off\necho {\"issuer\":\"GitHub\",\"name\":\"agent@example.test\",\"otp\":\"123456\",\"remaining_time\":17}\n"
	} else {
		script = `#!/usr/bin/env sh
printf '{"issuer":"GitHub","name":"agent@example.test","otp":"123456","remaining_time":17}\n'
`
	}
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatalf("write fake aegis: %v", err)
	}
	return path
}
