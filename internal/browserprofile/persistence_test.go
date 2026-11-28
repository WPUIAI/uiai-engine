package browserprofile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-rod/rod"
)

func TestExplicitProfileDirectoryRetainedAcrossRuntimeRestart(t *testing.T) {
	binary := FindSystemChromium()
	if binary == "" {
		t.Skip("system Chromium is not installed")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<!doctype html><title>persistent storage fixture</title>"))
	}))
	defer server.Close()

	dir := filepath.Join(t.TempDir(), "browser-profile")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	marker := filepath.Join(dir, "retention-marker")
	if err := os.WriteFile(marker, []byte("retain"), 0600); err != nil {
		t.Fatal(err)
	}
	profile := ResolvedProfile{ID: "persistent-fixture", Profile: Profile{
		Engine: EngineSystemChromium,
		Headless: boolPtr(true),
		Launch: LaunchConfig{ExecutablePath: binary, UserDataDir: dir, Persistent: true},
		Identity: IdentityConfig{Viewport: Viewport{Width: 800, Height: 600, DeviceScaleFactor: 1}},
		Storage: StorageConfig{CookieMode: "persistent", CacheMode: "persistent", LocalStorageMode: "persistent", ExclusiveLock: true},
	}}
	r, err := Launch(context.Background(), profile)
	if err != nil {
		t.Fatalf("launch first runtime: %v", err)
	}
	func() {
		defer r.Close()
		page, err := r.OpenPage(context.Background(), server.URL)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := page.Eval(`() => { localStorage.setItem("fixture", "retained"); document.cookie = "fixture=retained; Max-Age=86400; Path=/"; return true }`); err != nil {
			t.Fatal(err)
		}
	}()
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("explicit profile directory deleted at shutdown: %v", err)
	}
	r, err = Launch(context.Background(), profile)
	if err != nil {
		t.Fatalf("relaunch same profile: %v", err)
	}
	defer r.Close()
	page, err := r.OpenPage(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	state, err := page.Eval(`() => localStorage.getItem("fixture") + "|" + document.cookie`)
	if err != nil {
		t.Fatal(err)
	}
	if got := state.Value.Str(); !strings.HasPrefix(got, "retained|") || !strings.Contains(got, "fixture=retained") {
		t.Fatalf("profile did not retain cookie and localStorage: %q", got)
	}
}

func TestProfileDirectoryAliasesShareReservation(t *testing.T) {
	parent := t.TempDir()
	alias := filepath.Join(t.TempDir(), "alias")
	if err := os.Symlink(parent, alias); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	m := &Manager{locks: make(map[string]string)}
	profile := ResolvedProfile{ID: "first", Profile: Profile{Launch: LaunchConfig{UserDataDir: filepath.Join(parent, "not-created")}}}
	key, _, err := m.reserveProfileLock(profile)
	if err != nil || key == "" {
		t.Fatalf("reserve explicit directory: key=%q err=%v", key, err)
	}
	profile.Launch.UserDataDir = filepath.Join(alias, "not-created")
	if _, _, err := m.reserveProfileLock(profile); err == nil {
		t.Fatal("alias admitted a competing profile")
	}
}

type blockingProfileRuntime struct {
	entered chan struct{}
	release chan struct{}
}

func (r *blockingProfileRuntime) OpenPage(context.Context, string) (*rod.Page, error) {
	return nil, nil
}
func (r *blockingProfileRuntime) RuntimePID() int { return 0 }
func (r *blockingProfileRuntime) Close() {
	close(r.entered)
	<-r.release
}

func TestProfileLockRetainedUntilRuntimeClose(t *testing.T) {
	m := &Manager{locks: make(map[string]string), sessions: make(map[string]*ManagedSession)}
	profile := ResolvedProfile{ID: "persistent", Profile: Profile{Launch: LaunchConfig{UserDataDir: filepath.Join(t.TempDir(), "profile")}}}
	key, _, err := m.reserveProfileLock(profile)
	if err != nil {
		t.Fatal(err)
	}
	r := &blockingProfileRuntime{entered: make(chan struct{}), release: make(chan struct{})}
	m.sessions["one"] = &ManagedSession{ID: "one", lockKey: key, runtime: r}
	m.locks[key] = "one"
	done := make(chan error, 1)
	go func() { done <- m.Close("one") }()
	defer close(r.release)
	select {
	case <-r.entered:
	case <-time.After(5 * time.Second):
		t.Fatal("runtime close not reached")
	}
	if _, _, err := m.reserveProfileLock(profile); err == nil {
		t.Fatal("profile lock released before runtime shutdown")
	}
}

func TestExplicitProfileRejectsRawDirectoryOverride(t *testing.T) {
	p := ResolvedProfile{Profile: Profile{Engine: EngineSystemChromium, Launch: LaunchConfig{UserDataDir: t.TempDir(), ExtraArgs: []string{"--user-data-dir=/unmanaged"}}}}
	if _, err := BuildChromiumLauncher(p); err == nil {
		t.Fatal("raw extra args bypassed managed profile directory")
	}
}

func TestPersistentProfileCannotUseEphemeralContext(t *testing.T) {
	cfg := ReadyDefaultConfig()
	p := cfg.Profiles["operator"]
	p.Storage.CookieMode = "ephemeral"
	cfg.Profiles["operator"] = p
	if _, err := NewRegistry(cfg); err == nil {
		t.Fatal("mixed persistent and ephemeral storage accepted")
	}
}
