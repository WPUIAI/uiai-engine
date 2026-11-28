package vision

import (
	"testing"

	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/launcher/flags"
)

func TestPoolLaunchSettingsDefaultHeadless(t *testing.T) {
	s, err := resolvePoolLaunchSettings(PoolConfig{})
	if err != nil {
		t.Fatal(err)
	}
	l := s.apply(launcher.New())
	if !l.Has(flags.Headless) {
		t.Fatal("zero config must remain headless")
	}
}

func TestPoolLaunchSettingsExplicitHeaded(t *testing.T) {
	headless := false
	s, err := resolvePoolLaunchSettings(PoolConfig{Headless: &headless, WindowSize: "1920, 1080", Locale: "en-US"})
	if err != nil {
		t.Fatal(err)
	}
	// The caller cannot change a running pool's launch preference by
	// modifying its original config pointer.
	headless = true
	l := s.apply(launcher.New())
	if l.Has(flags.Headless) || l.Get("window-size") != "1920,1080" || l.Get("lang") != "en-US" {
		t.Fatalf("unexpected resolved launch settings: %+v", s)
	}
}

func TestPoolLaunchSettingsRejectInvalidWindowSizes(t *testing.T) {
	for _, size := range []string{"1920x1080", "0,1080", "-1,1", "100000,1", "1,2,3", "width,100"} {
		t.Run(size, func(t *testing.T) {
			if _, err := resolvePoolLaunchSettings(PoolConfig{WindowSize: size}); err == nil {
				t.Fatalf("accepted invalid dimensions %q", size)
			}
		})
	}
}

func TestPoolLaunchSettingsRejectInvalidLocale(t *testing.T) {
	if _, err := resolvePoolLaunchSettings(PoolConfig{Locale: "en-US --invalid"}); err == nil {
		t.Fatal("accepted malformed locale")
	}
}

func TestPoolStatsExposeConfiguredLaunchSettings(t *testing.T) {
	headless := false
	p, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, Headless: &headless, WindowSize: "1920,1080", Locale: "en-US"})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	stats := p.Stats()
	if stats["headless"] != false || stats["window_size"] != "1920,1080" || stats["locale"] != "en-US" {
		t.Fatalf("launch settings missing from diagnostics: %+v", stats)
	}
}
