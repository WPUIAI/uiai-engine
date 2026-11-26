package routes

import (
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

// #294 hear slice: the UIAI_WHISPER_URL environment override lets each
// deployment reach its own Whisper service address while the config default
// stays the CAPTCHA audio path's documented loopback.
func TestResolveWhisperURLOverrideAndDefault(t *testing.T) {
	t.Setenv("UIAI_WHISPER_URL", " http://100.69.132.82:8115 ")
	cfg := &config.Config{}
	if got := resolveWhisperURL(cfg); got != "http://100.69.132.82:8115" {
		t.Fatalf("env override not applied: %q", got)
	}

	t.Setenv("UIAI_WHISPER_URL", "")
	cfg.Captcha.Recaptcha = map[string]any{"whisper_url": "http://localhost:8115"}
	if got := resolveWhisperURL(cfg); got != "http://localhost:8115" {
		t.Fatalf("config default not honored: %q", got)
	}

	cfg.Captcha.Recaptcha = map[string]any{}
	if got := resolveWhisperURL(cfg); got != "" {
		t.Fatalf("unconfigured provider must resolve empty, got %q", got)
	}
}
