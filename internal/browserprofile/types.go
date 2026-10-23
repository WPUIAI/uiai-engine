package browserprofile

import "time"

// Mode controls the browser's operating posture.
type Mode string

const (
	ModeDetect   Mode = "detect"
	ModeNoDetect Mode = "no_detect"
	ModeOperator Mode = "operator"
	ModeResearch Mode = "research"
	ModeAuto     Mode = "auto"
)

// Engine selects the browser runtime adapter.
type Engine string

const (
	EngineChromium       Engine = "chromium"
	EngineSystemChromium Engine = "system_chromium"
	EngineCamoufox       Engine = "camoufox"
)

// ChallengePolicy controls how detected browser challenges are handled.
type ChallengePolicy string

const (
	ChallengeDisabled      ChallengePolicy = "disabled"
	ChallengeDetect        ChallengePolicy = "detect"
	ChallengeAssist        ChallengePolicy = "assist"
	ChallengeSolve         ChallengePolicy = "solve"
	ChallengeSolveAndRetry ChallengePolicy = "solve_and_retry"
)

// Config is the browser profile section loaded from config.yaml.
type Config struct {
	DefaultProfile string             `yaml:"default_profile" json:"default_profile"`
	Profiles       map[string]Profile `yaml:"profiles" json:"profiles"`
	DomainRules    []DomainRule       `yaml:"domain_rules,omitempty" json:"domain_rules,omitempty"`
}

// Profile is a declarative browser launch and identity contract.
type Profile struct {
	Extends       string              `yaml:"extends,omitempty" json:"extends,omitempty"`
	Label         string              `yaml:"label,omitempty" json:"label,omitempty"`
	Mode          Mode                `yaml:"mode" json:"mode"`
	Engine        Engine              `yaml:"engine" json:"engine"`
	Headless      *bool               `yaml:"headless,omitempty" json:"headless,omitempty"`
	Launch        LaunchConfig        `yaml:"launch,omitempty" json:"launch,omitempty"`
	Identity      IdentityConfig      `yaml:"identity,omitempty" json:"identity,omitempty"`
	Network       NetworkConfig       `yaml:"network,omitempty" json:"network,omitempty"`
	Storage       StorageConfig       `yaml:"storage,omitempty" json:"storage,omitempty"`
	Challenge     ChallengeConfig     `yaml:"challenge,omitempty" json:"challenge,omitempty"`
	Behavior      BehaviorConfig      `yaml:"behavior,omitempty" json:"behavior,omitempty"`
	Observability ObservabilityConfig `yaml:"observability,omitempty" json:"observability,omitempty"`
}

type LaunchConfig struct {
	ExecutablePath   string   `yaml:"executable_path,omitempty" json:"executable_path,omitempty"`
	DriverEndpoint   string   `yaml:"driver_endpoint,omitempty" json:"driver_endpoint,omitempty"`
	UserDataDir      string   `yaml:"user_data_dir,omitempty" json:"user_data_dir,omitempty"`
	Persistent       bool     `yaml:"persistent_context,omitempty" json:"persistent_context,omitempty"`
	NoSandbox        *bool    `yaml:"no_sandbox,omitempty" json:"no_sandbox,omitempty"`
	DisableGPU       *bool    `yaml:"disable_gpu,omitempty" json:"disable_gpu,omitempty"`
	DisableExtensions *bool   `yaml:"disable_extensions,omitempty" json:"disable_extensions,omitempty"`
	DisableWebSecurity *bool  `yaml:"disable_web_security,omitempty" json:"disable_web_security,omitempty"`
	IgnoreCertErrors *bool    `yaml:"ignore_cert_errors,omitempty" json:"ignore_cert_errors,omitempty"`
	BlockTrackers    *bool    `yaml:"block_trackers,omitempty" json:"block_trackers,omitempty"`
	ExtraArgs        []string `yaml:"extra_args,omitempty" json:"extra_args,omitempty"`
}

type IdentityConfig struct {
	BundleID            string            `yaml:"bundle_id,omitempty" json:"bundle_id,omitempty"`
	UserAgent           string            `yaml:"user_agent,omitempty" json:"user_agent,omitempty"`
	RandomUserAgent     bool              `yaml:"random_user_agent,omitempty" json:"random_user_agent,omitempty"`
	UserAgents          []string          `yaml:"user_agents,omitempty" json:"user_agents,omitempty"`
	Platform            string            `yaml:"platform,omitempty" json:"platform,omitempty"`
	Locale              string            `yaml:"locale,omitempty" json:"locale,omitempty"`
	Languages           []string          `yaml:"languages,omitempty" json:"languages,omitempty"`
	Timezone            string            `yaml:"timezone,omitempty" json:"timezone,omitempty"`
	Geolocation         *Geolocation      `yaml:"geolocation,omitempty" json:"geolocation,omitempty"`
	Viewport            Viewport          `yaml:"viewport,omitempty" json:"viewport,omitempty"`
	PatchWebDriver      bool              `yaml:"patch_webdriver,omitempty" json:"patch_webdriver,omitempty"`
	PatchChromeObject   bool              `yaml:"patch_chrome_object,omitempty" json:"patch_chrome_object,omitempty"`
	PatchPlugins        bool              `yaml:"patch_plugins,omitempty" json:"patch_plugins,omitempty"`
	PatchLanguages      bool              `yaml:"patch_languages,omitempty" json:"patch_languages,omitempty"`
	DisableAutomationControlled bool      `yaml:"disable_automation_controlled,omitempty" json:"disable_automation_controlled,omitempty"`
	ClientHints         map[string]string `yaml:"client_hints,omitempty" json:"client_hints,omitempty"`
	HardwareConcurrency int               `yaml:"hardware_concurrency,omitempty" json:"hardware_concurrency,omitempty"`
	DeviceMemoryGB      int               `yaml:"device_memory_gb,omitempty" json:"device_memory_gb,omitempty"`
	WebGLVendor         string            `yaml:"webgl_vendor,omitempty" json:"webgl_vendor,omitempty"`
	WebGLRenderer       string            `yaml:"webgl_renderer,omitempty" json:"webgl_renderer,omitempty"`
	FontsProfile        string            `yaml:"fonts_profile,omitempty" json:"fonts_profile,omitempty"`
	CanvasProfile       string            `yaml:"canvas_profile,omitempty" json:"canvas_profile,omitempty"`
	AudioProfile        string            `yaml:"audio_profile,omitempty" json:"audio_profile,omitempty"`
	MediaDevicesProfile string            `yaml:"media_devices_profile,omitempty" json:"media_devices_profile,omitempty"`
}

type Geolocation struct {
	Latitude  float64 `yaml:"latitude" json:"latitude"`
	Longitude float64 `yaml:"longitude" json:"longitude"`
	Accuracy  float64 `yaml:"accuracy,omitempty" json:"accuracy,omitempty"`
}

type Viewport struct {
	Width             int     `yaml:"width,omitempty" json:"width,omitempty"`
	Height            int     `yaml:"height,omitempty" json:"height,omitempty"`
	DeviceScaleFactor float64 `yaml:"device_scale_factor,omitempty" json:"device_scale_factor,omitempty"`
}

type NetworkConfig struct {
	Route          string `yaml:"route,omitempty" json:"route,omitempty"`
	RouteRef       string `yaml:"route_ref,omitempty" json:"route_ref,omitempty"`
	ProxyServer    string `yaml:"proxy_server,omitempty" json:"proxy_server,omitempty"`
	DNSMode        string `yaml:"dns_mode,omitempty" json:"dns_mode,omitempty"`
	WebRTCMode     string `yaml:"webrtc_mode,omitempty" json:"webrtc_mode,omitempty"`
	GeoConsistency bool   `yaml:"geo_consistency,omitempty" json:"geo_consistency,omitempty"`
	Sticky         bool   `yaml:"sticky,omitempty" json:"sticky,omitempty"`
}

type StorageConfig struct {
	IsolationKey         string `yaml:"isolation_key,omitempty" json:"isolation_key,omitempty"`
	CookieMode           string `yaml:"cookie_mode,omitempty" json:"cookie_mode,omitempty"`
	CacheMode            string `yaml:"cache_mode,omitempty" json:"cache_mode,omitempty"`
	LocalStorageMode     string `yaml:"local_storage_mode,omitempty" json:"local_storage_mode,omitempty"`
	CredentialProfileRef string `yaml:"credential_profile_ref,omitempty" json:"credential_profile_ref,omitempty"`
	ExclusiveLock        bool   `yaml:"exclusive_lock,omitempty" json:"exclusive_lock,omitempty"`
}

type ChallengeConfig struct {
	Policy             ChallengePolicy `yaml:"policy,omitempty" json:"policy,omitempty"`
	MaxAttempts        int             `yaml:"max_attempts,omitempty" json:"max_attempts,omitempty"`
	RouteRotation      bool            `yaml:"route_rotation,omitempty" json:"route_rotation,omitempty"`
	OperatorEscalation bool            `yaml:"operator_escalation,omitempty" json:"operator_escalation,omitempty"`
	TextSolver         string          `yaml:"text_solver,omitempty" json:"text_solver,omitempty"`
	ImageGridSolver    string          `yaml:"image_grid_solver,omitempty" json:"image_grid_solver,omitempty"`
	AudioSolver        string          `yaml:"audio_solver,omitempty" json:"audio_solver,omitempty"`
}

type BehaviorConfig struct {
	TimingProfile     string `yaml:"timing_profile,omitempty" json:"timing_profile,omitempty"`
	PointerProfile    string `yaml:"pointer_profile,omitempty" json:"pointer_profile,omitempty"`
	ScrollingProfile  string `yaml:"scrolling_profile,omitempty" json:"scrolling_profile,omitempty"`
	NavigationProfile string `yaml:"navigation_profile,omitempty" json:"navigation_profile,omitempty"`
}

type ObservabilityConfig struct {
	DiagnosticsLevel   string `yaml:"diagnostics_level,omitempty" json:"diagnostics_level,omitempty"`
	FingerprintCapture bool   `yaml:"fingerprint_capture,omitempty" json:"fingerprint_capture,omitempty"`
	ChallengeCapture   bool   `yaml:"challenge_capture,omitempty" json:"challenge_capture,omitempty"`
	NetworkCapture     bool   `yaml:"network_capture,omitempty" json:"network_capture,omitempty"`
	EvidenceGrade      bool   `yaml:"evidence_grade,omitempty" json:"evidence_grade,omitempty"`
}

type DomainRule struct {
	Pattern        string `yaml:"pattern" json:"pattern"`
	Profile        string `yaml:"profile" json:"profile"`
	ChallengePolicy ChallengePolicy `yaml:"challenge_policy,omitempty" json:"challenge_policy,omitempty"`
	Priority       int    `yaml:"priority,omitempty" json:"priority,omitempty"`
}

// ResolvedProfile is complete, validated, immutable launch input.
type ResolvedProfile struct {
	ID        string    `json:"profile_id"`
	Digest    string    `json:"profile_digest"`
	ResolvedAt time.Time `json:"resolved_at"`
	Profile
}

// Selection records an explicit or automatic profile decision.
type Selection struct {
	RequestedProfile string   `json:"requested_profile,omitempty"`
	RequestedMode    Mode     `json:"requested_mode,omitempty"`
	EffectiveProfile string   `json:"effective_profile"`
	EffectiveMode    Mode     `json:"effective_mode"`
	Digest           string   `json:"profile_digest"`
	PolicyRefs       []string `json:"policy_refs,omitempty"`
	Reasons          []string `json:"reasons,omitempty"`
}
