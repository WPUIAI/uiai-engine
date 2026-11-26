package twofactor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

// #86 acceptance: inject a dummy code through the command provider and
// assert the value appears only in the authorized Response - never in any
// error string. The two-factor service is the custody point that serves
// codes to authorized callers by design; what it must guarantee is that
// failures never embed the code.
func TestCommandProviderCodeNeverEntersErrorPaths(t *testing.T) {
	dir := t.TempDir()
	vault := filepath.Join(dir, "vault.json")
	if err := os.WriteFile(vault, []byte(`{"entries":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	script := filepath.Join(dir, "emit.sh")
	dummy := "413726"
	if err := os.WriteFile(script, []byte("#!/bin/sh\ncat <<JSON\n{\"code\": \""+dummy+"\", \"expires_in\": 25, \"period\": 30}\nJSON\n"), 0755); err != nil {
		t.Fatal(err)
	}
	cfg := &config.Config{}
	cfg.TwoFactor.Enabled = true
	cfg.TwoFactor.Profiles = map[string]config.TwoFactorProfile{
		"probe": {Provider: "command", Command: script, VaultFile: vault},
	}
	svc := New(cfg)
	resp, err := svc.Code(context.Background(), Request{Profile: "probe"})
	if err != nil {
		t.Fatalf("code resolve failed: %v", err)
	}
	if resp.Code != dummy {
		t.Fatalf("dummy code not served to the authorized caller: %q", resp.Code)
	}

	// Failure paths never embed the code: a missing profile, a bad command,
	// and a codeless command all produce typed errors without the value.
	for _, profileName := range []string{"missing", "bad-command", "empty-code"} {
		switch profileName {
		case "bad-command":
			cfg.TwoFactor.Profiles["bad-command"] = config.TwoFactorProfile{Provider: "command", Command: filepath.Join(dir, "no-such-cmd"), VaultFile: vault}
		case "empty-code":
			empty := filepath.Join(dir, "empty.sh")
			_ = os.WriteFile(empty, []byte("#!/bin/sh\necho '{\"code\": \"\"}'\n"), 0755)
			cfg.TwoFactor.Profiles["empty-code"] = config.TwoFactorProfile{Provider: "command", Command: empty, VaultFile: vault}
		}
		_, err := svc.Code(context.Background(), Request{Profile: profileName})
		if err == nil {
			t.Fatalf("profile %q unexpectedly resolved", profileName)
		}
		if strings.Contains(err.Error(), dummy) {
			t.Fatalf("error path for %q leaked the code value: %v", profileName, err)
		}
	}
}
