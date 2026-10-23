package browserprofile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Registry resolves named profiles and domain rules.
type Registry struct {
	config Config
}

func NewRegistry(cfg Config) (*Registry, error) {
	cfg.ApplyDefaults()
	r := &Registry{config: cfg}
	for name := range cfg.Profiles {
		if _, err := r.Resolve(name); err != nil {
			return nil, fmt.Errorf("browser profile %q: %w", name, err)
		}
	}
	if _, ok := cfg.Profiles[cfg.DefaultProfile]; !ok {
		return nil, fmt.Errorf("default browser profile %q not found", cfg.DefaultProfile)
	}
	return r, nil
}

func (c *Config) ApplyDefaults() {
	if c.Profiles == nil {
		c.Profiles = DefaultConfig().Profiles
	}
	if c.DefaultProfile == "" {
		c.DefaultProfile = "detect"
	}
}

func DefaultConfig() Config {
	headless := true
	noSandbox := true
	disableGPU := true
	disableExtensions := true
	disableWebSecurity := false
	ignoreCertErrors := false
	blockTrackers := false

	return Config{
		DefaultProfile: "detect",
		Profiles: map[string]Profile{
			"detect": {
				Label:    "Detect / Diagnostic",
				Mode:     ModeDetect,
				Engine:   EngineSystemChromium,
				Headless: &headless,
				Launch: LaunchConfig{
					NoSandbox:          &noSandbox,
					DisableGPU:         &disableGPU,
					DisableExtensions:  &disableExtensions,
					DisableWebSecurity: &disableWebSecurity,
					IgnoreCertErrors:   &ignoreCertErrors,
					BlockTrackers:      &blockTrackers,
				},
				Identity: IdentityConfig{
					Locale:    "en-US",
					Languages: []string{"en-US", "en"},
					Viewport:  Viewport{Width: 1280, Height: 800, DeviceScaleFactor: 1},
				},
				Network:   NetworkConfig{Route: "direct"},
				Storage:   StorageConfig{CookieMode: "ephemeral", CacheMode: "ephemeral", LocalStorageMode: "ephemeral"},
				Challenge: ChallengeConfig{Policy: ChallengeDetect, MaxAttempts: 1, OperatorEscalation: true},
				Behavior:  BehaviorConfig{TimingProfile: "deterministic", PointerProfile: "direct", ScrollingProfile: "direct", NavigationProfile: "deterministic"},
				Observability: ObservabilityConfig{DiagnosticsLevel: "developer", FingerprintCapture: true, ChallengeCapture: true, NetworkCapture: true},
			},
			"no_detect": {
				Extends:  "detect",
				Label:    "No Detect",
				Mode:     ModeNoDetect,
				Identity: IdentityConfig{PatchWebDriver: true, PatchChromeObject: true, PatchPlugins: true, PatchLanguages: true, DisableAutomationControlled: true, RandomUserAgent: true},
				Network:  NetworkConfig{Route: "local_ip_pool", RouteRef: "captcha-default", DNSMode: "proxy", WebRTCMode: "proxy_only", GeoConsistency: true, Sticky: true},
				Storage:  StorageConfig{CookieMode: "profile", CacheMode: "profile", LocalStorageMode: "profile"},
				Challenge: ChallengeConfig{Policy: ChallengeSolveAndRetry, MaxAttempts: 3, RouteRotation: true, OperatorEscalation: true, TextSolver: "auto", ImageGridSolver: "auto", AudioSolver: "whisper"},
				Behavior: BehaviorConfig{TimingProfile: "humanized", PointerProfile: "humanized", ScrollingProfile: "humanized", NavigationProfile: "humanized"},
			},
			"operator": {
				Extends:  "detect",
				Label:    "Operator",
				Mode:     ModeOperator,
				Headless: boolPtr(false),
				Launch:   LaunchConfig{Persistent: true},
				Network:  NetworkConfig{Route: "operator_route", GeoConsistency: true, Sticky: true},
				Storage:  StorageConfig{CookieMode: "persistent", CacheMode: "persistent", LocalStorageMode: "persistent", ExclusiveLock: true},
				Challenge: ChallengeConfig{Policy: ChallengeAssist, MaxAttempts: 1, OperatorEscalation: true},
				Observability: ObservabilityConfig{DiagnosticsLevel: "standard", ChallengeCapture: true},
			},
			"research": {
				Extends:  "no_detect",
				Label:    "Research / Eval",
				Mode:     ModeResearch,
				Storage:  StorageConfig{CookieMode: "ephemeral", CacheMode: "ephemeral", LocalStorageMode: "ephemeral"},
				Observability: ObservabilityConfig{DiagnosticsLevel: "full", FingerprintCapture: true, ChallengeCapture: true, NetworkCapture: true, EvidenceGrade: true},
			},
		},
	}
}

func boolPtr(v bool) *bool { return &v }

func (r *Registry) Config() Config { return r.config }

func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.config.Profiles))
	for name := range r.config.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (r *Registry) Resolve(name string) (ResolvedProfile, error) {
	if name == "" {
		name = r.config.DefaultProfile
	}
	profile, err := r.resolve(name, map[string]bool{})
	if err != nil {
		return ResolvedProfile{}, err
	}
	applyProfileDefaults(&profile)
	if err := validateResolved(name, profile); err != nil {
		return ResolvedProfile{}, err
	}
	digest, err := profileDigest(name, profile)
	if err != nil {
		return ResolvedProfile{}, err
	}
	return ResolvedProfile{ID: name, Digest: digest, ResolvedAt: time.Now().UTC(), Profile: profile}, nil
}

func (r *Registry) resolve(name string, seen map[string]bool) (Profile, error) {
	if seen[name] {
		return Profile{}, fmt.Errorf("profile inheritance cycle at %q", name)
	}
	profile, ok := r.config.Profiles[name]
	if !ok {
		return Profile{}, fmt.Errorf("unknown profile %q", name)
	}
	if profile.Extends == "" {
		return profile, nil
	}
	seen[name] = true
	base, err := r.resolve(profile.Extends, seen)
	delete(seen, name)
	if err != nil {
		return Profile{}, err
	}
	return mergeProfile(base, profile), nil
}

// Select applies explicit profile, mode, and domain rules. ModeAuto delegates to rules/default.
func (r *Registry) Select(rawURL, requestedProfile string, requestedMode Mode) (ResolvedProfile, Selection, error) {
	selected := requestedProfile
	reasons := []string{}
	if selected != "" {
		reasons = append(reasons, "explicit_profile")
	}
	if selected == "" && requestedMode != "" && requestedMode != ModeAuto {
		for _, name := range r.Names() {
			p, _ := r.Resolve(name)
			if p.Mode == requestedMode {
				selected = name
				reasons = append(reasons, "explicit_mode")
				break
			}
		}
	}
	if selected == "" {
		if host := ruleHost(rawURL); host != "" {
			bestPriority := -1 << 30
			for _, rule := range r.config.DomainRules {
				if domainMatches(host, rule.Pattern) && rule.Priority >= bestPriority {
					selected = rule.Profile
					bestPriority = rule.Priority
				}
			}
			if selected != "" {
				reasons = append(reasons, "domain_rule")
			}
		}
	}
	if selected == "" {
		selected = r.config.DefaultProfile
		reasons = append(reasons, "default_profile")
	}
	resolved, err := r.Resolve(selected)
	if err != nil {
		return ResolvedProfile{}, Selection{}, err
	}
	if resolved.Mode == ModeAuto {
		return ResolvedProfile{}, Selection{}, fmt.Errorf("auto profile %q must resolve to a concrete profile", selected)
	}
	return resolved, Selection{
		RequestedProfile: requestedProfile,
		RequestedMode: requestedMode,
		EffectiveProfile: resolved.ID,
		EffectiveMode: resolved.Mode,
		Digest: resolved.Digest,
		Reasons: reasons,
	}, nil
}

func applyProfileDefaults(p *Profile) {
	if p.Mode == "" { p.Mode = ModeDetect }
	if p.Engine == "" { p.Engine = EngineSystemChromium }
	if p.Headless == nil { p.Headless = boolPtr(p.Mode != ModeOperator) }
	if p.Identity.Locale == "" { p.Identity.Locale = "en-US" }
	if len(p.Identity.Languages) == 0 { p.Identity.Languages = []string{"en-US", "en"} }
	if p.Identity.Viewport.Width == 0 { p.Identity.Viewport.Width = 1280 }
	if p.Identity.Viewport.Height == 0 { p.Identity.Viewport.Height = 800 }
	if p.Identity.Viewport.DeviceScaleFactor == 0 { p.Identity.Viewport.DeviceScaleFactor = 1 }
	if p.Network.Route == "" { p.Network.Route = "direct" }
	if p.Challenge.Policy == "" { p.Challenge.Policy = ChallengeDetect }
	if p.Challenge.MaxAttempts == 0 { p.Challenge.MaxAttempts = 1 }
	if p.Observability.DiagnosticsLevel == "" { p.Observability.DiagnosticsLevel = "standard" }
}

func validateResolved(name string, p Profile) error {
	switch p.Mode {
	case ModeDetect, ModeNoDetect, ModeOperator, ModeResearch:
	default:
		return fmt.Errorf("profile %q has non-launchable mode %q", name, p.Mode)
	}
	switch p.Engine {
	case EngineChromium, EngineSystemChromium, EngineCamoufox:
	default:
		return fmt.Errorf("profile %q has unknown engine %q", name, p.Engine)
	}
	if p.Engine == EngineCamoufox && p.Launch.ExecutablePath == "" && p.Launch.DriverEndpoint == "" {
		return fmt.Errorf("profile %q uses camoufox but has no executable_path or driver_endpoint", name)
	}
	if p.Mode == ModeOperator {
		if !p.Launch.Persistent || p.Launch.UserDataDir == "" {
			return fmt.Errorf("operator profile %q requires persistent_context and user_data_dir", name)
		}
		if !p.Storage.ExclusiveLock {
			return fmt.Errorf("operator profile %q requires storage.exclusive_lock", name)
		}
	}
	if p.Network.Route == "named_proxy" && p.Network.RouteRef == "" && p.Network.ProxyServer == "" {
		return fmt.Errorf("profile %q named_proxy route requires route_ref or proxy_server", name)
	}
	if p.Identity.RandomUserAgent && len(p.Identity.UserAgents) == 0 && p.Identity.UserAgent == "" {
		return fmt.Errorf("profile %q enables random_user_agent without user_agents or user_agent", name)
	}
	return nil
}

func profileDigest(name string, p Profile) (string, error) {
	payload := struct {
		Name string `json:"name"`
		Profile Profile `json:"profile"`
	}{Name: name, Profile: p}
	b, err := json.Marshal(payload)
	if err != nil { return "", fmt.Errorf("marshal profile digest: %w", err) }
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func ruleHost(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil { return "" }
	return strings.ToLower(u.Hostname())
}

func domainMatches(host, pattern string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	pattern = strings.ToLower(strings.TrimSpace(pattern))
	if pattern == "" { return false }
	if strings.HasPrefix(pattern, "*.") {
		suffix := strings.TrimPrefix(pattern, "*.")
		return host == suffix || strings.HasSuffix(host, "."+suffix)
	}
	return host == pattern
}

func mergeProfile(base, override Profile) Profile {
	out := base
	out.Extends = ""
	if override.Label != "" { out.Label = override.Label }
	if override.Mode != "" { out.Mode = override.Mode }
	if override.Engine != "" { out.Engine = override.Engine }
	if override.Headless != nil { out.Headless = override.Headless }
	out.Launch = mergeLaunch(out.Launch, override.Launch)
	out.Identity = mergeIdentity(out.Identity, override.Identity)
	out.Network = mergeNetwork(out.Network, override.Network)
	out.Storage = mergeStorage(out.Storage, override.Storage)
	out.Challenge = mergeChallenge(out.Challenge, override.Challenge)
	out.Behavior = mergeBehavior(out.Behavior, override.Behavior)
	out.Observability = mergeObservability(out.Observability, override.Observability)
	return out
}

func mergeLaunch(a, b LaunchConfig) LaunchConfig {
	if b.ExecutablePath != "" { a.ExecutablePath = b.ExecutablePath }
	if b.DriverEndpoint != "" { a.DriverEndpoint = b.DriverEndpoint }
	if b.UserDataDir != "" { a.UserDataDir = b.UserDataDir }
	if b.Persistent { a.Persistent = true }
	if b.NoSandbox != nil { a.NoSandbox = b.NoSandbox }
	if b.DisableGPU != nil { a.DisableGPU = b.DisableGPU }
	if b.DisableExtensions != nil { a.DisableExtensions = b.DisableExtensions }
	if b.DisableWebSecurity != nil { a.DisableWebSecurity = b.DisableWebSecurity }
	if b.IgnoreCertErrors != nil { a.IgnoreCertErrors = b.IgnoreCertErrors }
	if b.BlockTrackers != nil { a.BlockTrackers = b.BlockTrackers }
	if len(b.ExtraArgs) > 0 { a.ExtraArgs = append([]string{}, b.ExtraArgs...) }
	return a
}

func mergeIdentity(a, b IdentityConfig) IdentityConfig {
	if b.BundleID != "" { a.BundleID = b.BundleID }
	if b.UserAgent != "" { a.UserAgent = b.UserAgent }
	if b.RandomUserAgent { a.RandomUserAgent = true }
	if len(b.UserAgents) > 0 { a.UserAgents = append([]string{}, b.UserAgents...) }
	if b.Platform != "" { a.Platform = b.Platform }
	if b.Locale != "" { a.Locale = b.Locale }
	if len(b.Languages) > 0 { a.Languages = append([]string{}, b.Languages...) }
	if b.Timezone != "" { a.Timezone = b.Timezone }
	if b.Geolocation != nil { a.Geolocation = b.Geolocation }
	if b.Viewport.Width != 0 { a.Viewport.Width = b.Viewport.Width }
	if b.Viewport.Height != 0 { a.Viewport.Height = b.Viewport.Height }
	if b.Viewport.DeviceScaleFactor != 0 { a.Viewport.DeviceScaleFactor = b.Viewport.DeviceScaleFactor }
	if b.PatchWebDriver { a.PatchWebDriver = true }
	if b.PatchChromeObject { a.PatchChromeObject = true }
	if b.PatchPlugins { a.PatchPlugins = true }
	if b.PatchLanguages { a.PatchLanguages = true }
	if b.DisableAutomationControlled { a.DisableAutomationControlled = true }
	if len(b.ClientHints) > 0 { a.ClientHints = b.ClientHints }
	if b.HardwareConcurrency != 0 { a.HardwareConcurrency = b.HardwareConcurrency }
	if b.DeviceMemoryGB != 0 { a.DeviceMemoryGB = b.DeviceMemoryGB }
	if b.WebGLVendor != "" { a.WebGLVendor = b.WebGLVendor }
	if b.WebGLRenderer != "" { a.WebGLRenderer = b.WebGLRenderer }
	if b.FontsProfile != "" { a.FontsProfile = b.FontsProfile }
	if b.CanvasProfile != "" { a.CanvasProfile = b.CanvasProfile }
	if b.AudioProfile != "" { a.AudioProfile = b.AudioProfile }
	if b.MediaDevicesProfile != "" { a.MediaDevicesProfile = b.MediaDevicesProfile }
	return a
}

func mergeNetwork(a, b NetworkConfig) NetworkConfig {
	if b.Route != "" { a.Route = b.Route }
	if b.RouteRef != "" { a.RouteRef = b.RouteRef }
	if b.ProxyServer != "" { a.ProxyServer = b.ProxyServer }
	if b.DNSMode != "" { a.DNSMode = b.DNSMode }
	if b.WebRTCMode != "" { a.WebRTCMode = b.WebRTCMode }
	if b.GeoConsistency { a.GeoConsistency = true }
	if b.Sticky { a.Sticky = true }
	return a
}

func mergeStorage(a, b StorageConfig) StorageConfig {
	if b.IsolationKey != "" { a.IsolationKey = b.IsolationKey }
	if b.CookieMode != "" { a.CookieMode = b.CookieMode }
	if b.CacheMode != "" { a.CacheMode = b.CacheMode }
	if b.LocalStorageMode != "" { a.LocalStorageMode = b.LocalStorageMode }
	if b.CredentialProfileRef != "" { a.CredentialProfileRef = b.CredentialProfileRef }
	if b.ExclusiveLock { a.ExclusiveLock = true }
	return a
}

func mergeChallenge(a, b ChallengeConfig) ChallengeConfig {
	if b.Policy != "" { a.Policy = b.Policy }
	if b.MaxAttempts != 0 { a.MaxAttempts = b.MaxAttempts }
	if b.RouteRotation { a.RouteRotation = true }
	if b.OperatorEscalation { a.OperatorEscalation = true }
	if b.TextSolver != "" { a.TextSolver = b.TextSolver }
	if b.ImageGridSolver != "" { a.ImageGridSolver = b.ImageGridSolver }
	if b.AudioSolver != "" { a.AudioSolver = b.AudioSolver }
	return a
}

func mergeBehavior(a, b BehaviorConfig) BehaviorConfig {
	if b.TimingProfile != "" { a.TimingProfile = b.TimingProfile }
	if b.PointerProfile != "" { a.PointerProfile = b.PointerProfile }
	if b.ScrollingProfile != "" { a.ScrollingProfile = b.ScrollingProfile }
	if b.NavigationProfile != "" { a.NavigationProfile = b.NavigationProfile }
	return a
}

func mergeObservability(a, b ObservabilityConfig) ObservabilityConfig {
	if b.DiagnosticsLevel != "" { a.DiagnosticsLevel = b.DiagnosticsLevel }
	if b.FingerprintCapture { a.FingerprintCapture = true }
	if b.ChallengeCapture { a.ChallengeCapture = true }
	if b.NetworkCapture { a.NetworkCapture = true }
	if b.EvidenceGrade { a.EvidenceGrade = true }
	return a
}
