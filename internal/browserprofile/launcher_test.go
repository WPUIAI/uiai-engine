package browserprofile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

func TestEphemeralSessionsDoNotShareCookiesAcrossBrowserContexts(t *testing.T) {
	binary := FindSystemChromium()
	if binary == "" {
		t.Skip("system Chromium is not installed")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<!doctype html><title>context isolation</title>"))
	}))
	defer server.Close()

	launchURL, err := launcher.New().Bin(binary).Headless(true).Leakless(false).Set("no-sandbox").Launch()
	if err != nil {
		t.Skipf("Chromium unavailable: %v", err)
	}
	browser := rod.New().ControlURL(launchURL)
	if err := browser.Connect(); err != nil {
		t.Fatalf("connect Chromium: %v", err)
	}
	defer browser.Close()

	headless := true
	profile := ResolvedProfile{
		ID: "detect",
		Profile: Profile{
			Engine:   EngineSystemChromium,
			Headless: &headless,
			Identity: IdentityConfig{Viewport: Viewport{Width: 800, Height: 600, DeviceScaleFactor: 1}},
			Storage:  StorageConfig{CookieMode: "ephemeral", CacheMode: "ephemeral", LocalStorageMode: "ephemeral"},
		},
	}
	runtimeOne := &Runtime{Profile: profile, Browser: browser}
	runtimeTwo := &Runtime{Profile: profile, Browser: browser}
	pageOne, err := runtimeOne.OpenPage(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("open first session: %v", err)
	}
	defer pageOne.Close()
	pageTwo, err := runtimeTwo.OpenPage(context.Background(), server.URL)
	if err != nil {
		t.Fatalf("open second session: %v", err)
	}
	defer pageTwo.Close()

	if _, err := pageOne.Eval(`() => { document.cookie = "session=one; path=/"; return document.cookie }`); err != nil {
		t.Fatalf("set first-session cookie: %v", err)
	}
	cookies, err := pageTwo.Eval(`() => document.cookie`)
	if err != nil {
		t.Fatalf("read second-session cookies: %v", err)
	}
	if got := cookies.Value.Str(); got != "" {
		t.Fatalf("ephemeral browser contexts shared cookies: %q", got)
	}

	_ = runtimeOne.sessionBrowser.Close()
	runtimeOne.sessionBrowser = nil
	_ = runtimeTwo.sessionBrowser.Close()
	runtimeTwo.sessionBrowser = nil
}
