package browserprofile

import (
	"context"
	cryptorand "crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
	"github.com/go-rod/rod/lib/proto"
)

const trackerResolverRules = "MAP www.google-analytics.com 0.0.0.0, " +
	"MAP google-analytics.com 0.0.0.0, " +
	"MAP www.googletagmanager.com 0.0.0.0, " +
	"MAP googletagmanager.com 0.0.0.0, " +
	"MAP connect.facebook.net 0.0.0.0, " +
	"MAP static.hotjar.com 0.0.0.0, " +
	"MAP us.posthog.com 0.0.0.0, " +
	"MAP app.posthog.com 0.0.0.0, " +
	"MAP clarity.ms 0.0.0.0"

// Runtime is a launched browser process bound to one resolved profile.
type Runtime struct {
	Profile  ResolvedProfile
	Browser  *rod.Browser
	Launcher *launcher.Launcher
	PID      int
}

// Launch starts a Chromium-compatible runtime for the resolved profile.
// Camoufox can be connected through driver_endpoint; native Camoufox process
// management is implemented by its adapter without changing this contract.
func Launch(ctx context.Context, profile ResolvedProfile) (*Runtime, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if profile.Engine == EngineCamoufox && profile.Launch.DriverEndpoint != "" {
		browser := rod.New().ControlURL(profile.Launch.DriverEndpoint)
		if err := browser.Connect(); err != nil {
			return nil, fmt.Errorf("connect camoufox driver: %w", err)
		}
		return &Runtime{Profile: profile, Browser: browser}, nil
	}
	if profile.Engine == EngineCamoufox {
		return nil, fmt.Errorf("camoufox profile %q requires driver_endpoint for the current Go adapter", profile.ID)
	}

	l, err := BuildChromiumLauncher(profile)
	if err != nil {
		return nil, err
	}
	u, err := l.Launch()
	if err != nil {
		return nil, fmt.Errorf("launch browser profile %q: %w", profile.ID, err)
	}
	browser := rod.New().ControlURL(u)
	if err := browser.Connect(); err != nil {
		l.Cleanup()
		return nil, fmt.Errorf("connect browser profile %q: %w", profile.ID, err)
	}
	if boolValue(profile.Launch.IgnoreCertErrors, false) {
		browser.IgnoreCertErrors(true)
	}
	return &Runtime{Profile: profile, Browser: browser, Launcher: l, PID: l.PID()}, nil
}

// BuildChromiumLauncher converts a resolved profile into Rod launcher settings.
func BuildChromiumLauncher(profile ResolvedProfile) (*launcher.Launcher, error) {
	if profile.Engine == EngineCamoufox {
		return nil, fmt.Errorf("camoufox is not a Chromium launcher")
	}

	l := launcher.New().Headless(boolValue(profile.Headless, true))
	if boolValue(profile.Launch.DisableGPU, true) {
		l = l.Set("disable-gpu")
	}
	if boolValue(profile.Launch.NoSandbox, true) {
		l = l.Set("no-sandbox")
	}
	if boolValue(profile.Launch.DisableExtensions, true) {
		l = l.Set("disable-extensions")
	}
	if boolValue(profile.Launch.DisableWebSecurity, false) {
		l = l.Set("disable-web-security")
	}
	if profile.Identity.DisableAutomationControlled {
		l = l.Set("disable-blink-features", "AutomationControlled")
	}
	if profile.Identity.Locale != "" {
		l = l.Set("lang", profile.Identity.Locale)
	}
	if ua := selectedUserAgent(profile); ua != "" {
		l = l.Set("user-agent", ua)
	}
	if profile.Network.ProxyServer != "" {
		l = l.Set("proxy-server", profile.Network.ProxyServer)
	}
	if profile.Launch.UserDataDir != "" {
		dir, err := expandPath(profile.Launch.UserDataDir)
		if err != nil {
			return nil, err
		}
		l = l.Set("user-data-dir", dir)
	}
	if boolValue(profile.Launch.BlockTrackers, false) {
		l = l.Set("host-resolver-rules", trackerResolverRules)
	}

	l = l.Set("no-first-run").
		Set("disable-component-update").
		Set("disable-domain-reliability").
		Set("disable-crash-reporter")

	for _, raw := range profile.Launch.ExtraArgs {
		key, value := splitArg(raw)
		if key == "" {
			continue
		}
		if value == "" {
			l = l.Set(key)
		} else {
			l = l.Set(key, value)
		}
	}

	bin := profile.Launch.ExecutablePath
	if bin == "" && profile.Engine == EngineSystemChromium {
		bin = FindSystemChromium()
	}
	if bin != "" {
		l = l.Bin(bin)
	}
	return l, nil
}

// OpenPage creates a page, installs the identity bundle before navigation,
// and navigates to targetURL.
func (r *Runtime) OpenPage(ctx context.Context, targetURL string) (*rod.Page, error) {
	if r == nil || r.Browser == nil {
		return nil, fmt.Errorf("browser runtime is not connected")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	page, err := r.Browser.Page(proto.TargetCreateTarget{URL: "about:blank"})
	if err != nil {
		return nil, fmt.Errorf("create page: %w", err)
	}
	viewport := r.Profile.Identity.Viewport
	if err := page.SetViewport(&proto.EmulationSetDeviceMetricsOverride{
		Width: viewport.Width,
		Height: viewport.Height,
		DeviceScaleFactor: viewport.DeviceScaleFactor,
	}); err != nil {
		page.Close()
		return nil, fmt.Errorf("set viewport: %w", err)
	}
	if err := ApplyPageIdentity(page, r.Profile); err != nil {
		page.Close()
		return nil, err
	}
	if err := page.Timeout(30 * time.Second).Navigate(targetURL); err != nil {
		page.Close()
		return nil, fmt.Errorf("navigate: %w", err)
	}
	return page, nil
}

// ApplyPageIdentity installs document-start patches and UA overrides.
func ApplyPageIdentity(page *rod.Page, profile ResolvedProfile) (err error) {
	if page == nil {
		return fmt.Errorf("nil page")
	}
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("apply browser profile %q: %v", profile.ID, recovered)
		}
	}()

	ua := selectedUserAgent(profile)
	if ua != "" {
		page.MustSetUserAgent(&proto.NetworkSetUserAgentOverride{
			UserAgent: ua,
			AcceptLanguage: strings.Join(profile.Identity.Languages, ","),
			Platform: profile.Identity.Platform,
		})
	}

	script, buildErr := identityScript(profile)
	if buildErr != nil {
		return buildErr
	}
	if script != "" {
		page.MustEvalOnNewDocument(script)
	}
	return nil
}

func identityScript(profile ResolvedProfile) (string, error) {
	id := profile.Identity
	patch := map[string]any{
		"webdriver": id.PatchWebDriver,
		"chrome": id.PatchChromeObject,
		"plugins": id.PatchPlugins,
		"languages_patch": id.PatchLanguages,
		"languages": id.Languages,
		"hardware_concurrency": id.HardwareConcurrency,
		"device_memory": id.DeviceMemoryGB,
		"webgl_vendor": id.WebGLVendor,
		"webgl_renderer": id.WebGLRenderer,
		"bundle_seed": profile.Digest,
	}
	b, err := json.Marshal(patch)
	if err != nil {
		return "", fmt.Errorf("marshal identity patch: %w", err)
	}
	if !id.PatchWebDriver && !id.PatchChromeObject && !id.PatchPlugins && !id.PatchLanguages && id.HardwareConcurrency == 0 && id.DeviceMemoryGB == 0 && id.WebGLVendor == "" && id.WebGLRenderer == "" {
		return "", nil
	}
	return fmt.Sprintf(`(() => {
  const cfg = %s;
  const define = (obj, key, value) => {
    try { Object.defineProperty(obj, key, { get: () => value, configurable: true }); } catch (_) {}
  };
  if (cfg.webdriver) {
    define(Navigator.prototype, 'webdriver', undefined);
    try { delete navigator.__proto__.webdriver; } catch (_) {}
  }
  if (cfg.languages_patch && Array.isArray(cfg.languages) && cfg.languages.length) {
    define(Navigator.prototype, 'languages', cfg.languages);
    define(Navigator.prototype, 'language', cfg.languages[0]);
  }
  if (cfg.plugins) {
    const plugins = [{name:'Chrome PDF Plugin'}, {name:'Chrome PDF Viewer'}, {name:'Native Client'}];
    define(Navigator.prototype, 'plugins', plugins);
  }
  if (cfg.chrome && !window.chrome) {
    Object.defineProperty(window, 'chrome', { value: { runtime: {}, loadTimes(){}, csi(){} }, configurable: true });
  }
  if (cfg.hardware_concurrency) define(Navigator.prototype, 'hardwareConcurrency', cfg.hardware_concurrency);
  if (cfg.device_memory) define(Navigator.prototype, 'deviceMemory', cfg.device_memory);
  if (cfg.webgl_vendor || cfg.webgl_renderer) {
    const original = WebGLRenderingContext.prototype.getParameter;
    WebGLRenderingContext.prototype.getParameter = function(parameter) {
      if (parameter === 37445 && cfg.webgl_vendor) return cfg.webgl_vendor;
      if (parameter === 37446 && cfg.webgl_renderer) return cfg.webgl_renderer;
      return original.call(this, parameter);
    };
    if (typeof WebGL2RenderingContext !== 'undefined') {
      const original2 = WebGL2RenderingContext.prototype.getParameter;
      WebGL2RenderingContext.prototype.getParameter = function(parameter) {
        if (parameter === 37445 && cfg.webgl_vendor) return cfg.webgl_vendor;
        if (parameter === 37446 && cfg.webgl_renderer) return cfg.webgl_renderer;
        return original2.call(this, parameter);
      };
    }
  }
})();`, string(b)), nil
}

func (r *Runtime) Close() {
	if r == nil {
		return
	}
	if r.Browser != nil {
		_ = r.Browser.Close()
	}
	if r.Launcher != nil {
		r.Launcher.Cleanup()
	}
}

// FindSystemChromium returns the first installed Chromium-compatible binary.
func FindSystemChromium() string {
	for _, candidate := range []string{
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"/usr/lib64/chromium-browser/chromium-browser",
		"/usr/lib/chromium-browser/chromium-browser",
		"/usr/bin/google-chrome-stable",
		"/usr/bin/google-chrome",
		"/usr/bin/chromium",
	} {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func selectedUserAgent(profile ResolvedProfile) string {
	id := profile.Identity
	if id.RandomUserAgent && len(id.UserAgents) > 0 {
		return id.UserAgents[secureIndex(len(id.UserAgents))]
	}
	return id.UserAgent
}

func secureIndex(length int) int {
	if length <= 1 {
		return 0
	}
	var b [8]byte
	if _, err := cryptorand.Read(b[:]); err != nil {
		return int(time.Now().UnixNano() % int64(length))
	}
	return int(binary.LittleEndian.Uint64(b[:]) % uint64(length))
}

func splitArg(raw string) (string, string) {
	raw = strings.TrimSpace(strings.TrimPrefix(raw, "--"))
	if raw == "" {
		return "", ""
	}
	parts := strings.SplitN(raw, "=", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func boolValue(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}
	return *value
}

func expandPath(path string) (string, error) {
	path = os.ExpandEnv(path)
	if path == "" || path[0] != '~' {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	if path == "~" {
		return home, nil
	}
	return filepath.Join(home, strings.TrimPrefix(path, "~/")), nil
}
