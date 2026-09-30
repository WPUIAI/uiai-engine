// Package config loads and provides access to engine configuration.
package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server           ServerConfig           `yaml:"server"`
	WordPress        WordPressConfig        `yaml:"wordpress"`
	AI               AIConfig               `yaml:"ai"`
	Vision           VisionConfig           `yaml:"vision"`
	Credits          CreditsConfig          `yaml:"credits"`
	RateLimits       RateLimitConfig        `yaml:"rate_limits"`
	Storage          StorageConfig          `yaml:"storage"`
	Logging          LoggingConfig          `yaml:"logging"`
	CORS             CORSConfig             `yaml:"cors"`
	Media            MediaConfig            `yaml:"media"`
	Captcha          CaptchaYAML            `yaml:"captcha"`
	TwoFactor        TwoFactorConfig        `yaml:"two_factor"`
	EvidenceRegistry EvidenceRegistryConfig `yaml:"evidence_registry"`
	Search           SearchConfig           `yaml:"search"`
}

// SearchConfig makes search providers and their credentials operator-configurable
// rather than hardcoded to one provider's environment variable. Any provider can be
// added by configuration alone; no bespoke code path per vendor.
type SearchConfig struct {
	// DefaultProvider selects the provider used when a request omits one.
	// Empty means "use the built-in default chain".
	DefaultProvider string `yaml:"default_provider,omitempty"`
	// Providers is keyed by provider id (for example "brave"). Built-in
	// providers remain available when they are absent from this map.
	Providers map[string]SearchProviderConfig `yaml:"providers,omitempty"`
}

// SearchProviderConfig describes one search provider. Credentials are resolved in
// order: api_key_file, api_key, api_key_env. Every one of those fields goes through
// the config loader's reference expansion first, so a secret can be written as
// ${ENV_VAR}, ${rbw:item}, or ${rbw:item:field} and resolved from the host's own
// secret store instead of being pasted into this file or the unit environment.
// Literal token values are never expected in api_key, matching the repository's
// existing credential-file convention (see evidence_registry.focusa_token_file).
type SearchProviderConfig struct {
	APIURL     string `yaml:"api_url,omitempty"`
	APIKey     string `yaml:"api_key,omitempty"`
	APIKeyEnv  string `yaml:"api_key_env,omitempty"`
	APIKeyFile string `yaml:"api_key_file,omitempty"`
	// Disabled removes a built-in provider from the ready set without deleting
	// its configuration.
	Disabled bool `yaml:"disabled,omitempty"`
}

// ResolveSearchProviderKey returns the credential for a configured provider, or an
// empty string when the operator has not supplied one. Resolution order is
// api_key_file, then api_key, then api_key_env.
func (s SearchConfig) ResolveSearchProviderKey(providerID string) (string, error) {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return "", nil
	}
	provider, ok := s.Providers[providerID]
	if !ok {
		return "", nil
	}
	if file := strings.TrimSpace(provider.APIKeyFile); file != "" {
		data, err := os.ReadFile(file) // #nosec G304 -- path is explicit operator config, not request input.
		if err != nil {
			return "", fmt.Errorf("read search provider %s credential file: %w", providerID, err)
		}
		return strings.TrimSpace(string(data)), nil
	}
	if key := strings.TrimSpace(provider.APIKey); key != "" {
		return key, nil
	}
	if name := strings.TrimSpace(provider.APIKeyEnv); name != "" {
		return strings.TrimSpace(os.Getenv(name)), nil
	}
	return "", nil
}

// SearchProviderConfigured reports whether a provider has a usable credential.
func (s SearchConfig) SearchProviderConfigured(providerID string) bool {
	key, err := s.ResolveSearchProviderKey(providerID)
	return err == nil && key != ""
}

// SearchProviderEnabled reports whether a provider is usable and not disabled.
func (s SearchConfig) SearchProviderEnabled(providerID string) bool {
	providerID = strings.TrimSpace(providerID)
	if providerID == "" {
		return false
	}
	if provider, ok := s.Providers[providerID]; ok && provider.Disabled {
		return false
	}
	if _, declared := s.Providers[providerID]; !declared {
		// Undeclared providers keep their built-in credential lookup so existing
		// deployments that export a provider's conventional variable keep working.
		return BuiltinSearchProviderKey(providerID) != ""
	}
	return s.SearchProviderConfigured(providerID)
}

// BuiltinSearchProviderEnv maps a provider id to its conventional environment
// variable. It is a fallback only: an explicit search.providers entry always wins,
// so an operator can relocate or rename any credential without a rebuild.
var BuiltinSearchProviderEnv = map[string]string{
	"brave": "BRAVE_SEARCH_API_KEY",
}

// BuiltinSearchProviderKey returns the conventional environment credential for a
// built-in provider, or an empty string.
func BuiltinSearchProviderKey(providerID string) string {
	name, ok := BuiltinSearchProviderEnv[strings.TrimSpace(providerID)]
	if !ok {
		return ""
	}
	return strings.TrimSpace(os.Getenv(name))
}

// CaptchaYAML mirrors the YAML structure for captcha config loading.
// Converted to captcha.CaptchaConfig at runtime.
type CaptchaYAML struct {
	Enabled         bool             `yaml:"enabled"`
	DefaultProvider string           `yaml:"default_provider"`
	DefaultModel    string           `yaml:"default_model"`
	Text            map[string]any   `yaml:"text"`
	Recaptcha       map[string]any   `yaml:"recaptcha"`
	Proxy           CaptchaProxyYAML `yaml:"proxy"`
	Stealth         map[string]any   `yaml:"stealth"`
	Stats           map[string]any   `yaml:"stats"`
}

type CaptchaProxyYAML struct {
	Enabled            bool     `yaml:"enabled"`
	LocalIPs           []string `yaml:"local_ips"`
	Proxies            []string `yaml:"proxies"`
	Strategy           string   `yaml:"strategy"`
	MaxConcurrentPerIP int      `yaml:"max_concurrent_per_ip"`
	CooldownMinutes    int      `yaml:"cooldown_minutes"`
	HealthFile         string   `yaml:"health_file"`
	HealthProbeURL     string   `yaml:"health_probe_url"`
	HealthProbeSeconds int      `yaml:"health_probe_seconds"`
	MaxRetries         int      `yaml:"max_retries"`
}

type MediaConfig struct {
	ScriptDir   string `yaml:"script_dir"`
	GitHubOrg   string `yaml:"github_org"`
	GitHubRepo  string `yaml:"github_repo"`
	GitHubToken string `yaml:"github_token"`
	R2PublicURL string `yaml:"r2_public_url"`
	R2Bucket    string `yaml:"r2_bucket"`
	JobTimeout  int    `yaml:"job_timeout"`
}

type ServerConfig struct {
	Port           int           `yaml:"port"`
	Host           string        `yaml:"host"`
	ReadTimeout    time.Duration `yaml:"read_timeout"`
	WriteTimeout   time.Duration `yaml:"write_timeout"`
	DisableVision  bool          `yaml:"disable_vision"`
	VisionPoolSize int           `yaml:"vision_pool_size"`
}

type WordPressConfig struct {
	URL           string `yaml:"url"`
	RESTNamespace string `yaml:"rest_namespace"`
	WebhookSecret string `yaml:"webhook_secret"`
	CacheTTL      int    `yaml:"cache_ttl"`
}

type AIConfig struct {
	DefaultModel    string                      `yaml:"default_model"`
	DefaultProvider string                      `yaml:"default_provider"`
	Providers       map[string]AIProviderConfig `yaml:"providers"`
}

type AIProviderConfig struct {
	APIURL     string `yaml:"api_url"`
	APIVersion string `yaml:"api_version,omitempty"`
	SiteURL    string `yaml:"site_url,omitempty"`
	SiteName   string `yaml:"site_name,omitempty"`
}

type TwoFactorConfig struct {
	Enabled  bool                        `yaml:"enabled"`
	Profiles map[string]TwoFactorProfile `yaml:"profiles"`
}

type TwoFactorProfile struct {
	Provider     string `yaml:"provider"` // totp | aegis/aegis-rs | command
	Secret       string `yaml:"secret,omitempty"`
	OTPAuthURL   string `yaml:"otpauth_url,omitempty"`
	Issuer       string `yaml:"issuer,omitempty"`
	Name         string `yaml:"name,omitempty"`
	Algorithm    string `yaml:"algorithm,omitempty"`
	Digits       int    `yaml:"digits,omitempty"`
	Period       int    `yaml:"period,omitempty"`
	Command      string `yaml:"command,omitempty"`
	VaultFile    string `yaml:"vault_file,omitempty"`
	Password     string `yaml:"password,omitempty"`
	PasswordFile string `yaml:"password_file,omitempty"`
}

type VisionConfig struct {
	PoolSize          int           `yaml:"pool_size"`
	MaxPool           int           `yaml:"max_pool"`
	Browsers          int           `yaml:"browsers"`
	PoolPages         int           `yaml:"pool_pages"`
	IdleTimeout       time.Duration `yaml:"idle_timeout"`
	ScreenshotQuality int           `yaml:"screenshot_quality"`
	ShareDir          string        `yaml:"share_dir"`
	AllowPrivateURLs  bool          `yaml:"allow_private_urls"` // disable SSRF private-IP blocking (for local dev/staging)
}

type CreditsConfig struct {
	Costs map[string]float64 `yaml:"costs"`
}

type RateLimitConfig struct {
	Tiers map[string]TierLimit `yaml:"tiers"`
}

type TierLimit struct {
	PerHour int `yaml:"per_hour"`
	PerDay  int `yaml:"per_day"`
}

type EvidenceRegistryConfig struct {
	ArtifactStoreEnabled bool          `yaml:"artifact_store_enabled"`
	ArtifactStoreRoot    string        `yaml:"artifact_store_root"`
	MaxArtifactBytes     int64         `yaml:"max_artifact_bytes"`
	MaxArtifactCount     int           `yaml:"max_artifact_count"`
	MaxAssetBytes        int64         `yaml:"max_asset_bytes"`
	ProviderSyncEnabled  bool          `yaml:"provider_sync_enabled"`
	FocusaURL            string        `yaml:"focusa_url"`
	FocusaTokenFile      string        `yaml:"focusa_token_file"`
	BRPath               string        `yaml:"br_path"`
	ProjectIDs           []string      `yaml:"project_ids"`
	PublicProjectRefs    []string      `yaml:"public_project_refs"`
	AllowedRootPrefixes  []string      `yaml:"allowed_root_prefixes"`
	MaxProjects          int           `yaml:"max_projects"`
	MaxItems             int           `yaml:"max_items"`
	ReconcileInterval    time.Duration `yaml:"reconcile_interval"`
}

type StorageConfig struct {
	DataDir   string `yaml:"data_dir"`
	UsageFile string `yaml:"usage_file"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

type CORSConfig struct {
	Origins []string `yaml:"origins"`
	Methods []string `yaml:"methods"`
	Headers []string `yaml:"headers"`
}

// Load reads config from a YAML file and expands environment variables.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- config path is explicit operator/CLI input or fixed config candidate.
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	// Expand ${ENV_VAR} and explicit secret-provider references such as ${rbw:item}.
	expanded, err := expandConfigRefs(string(data))
	if err != nil {
		return nil, fmt.Errorf("expand config %s: %w", path, err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	cfg.applyDefaults()
	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Port == 0 {
		c.Server.Port = 7456
	}
	if c.Server.Host == "" {
		c.Server.Host = "127.0.0.1"
	}
	if c.Server.ReadTimeout == 0 {
		c.Server.ReadTimeout = 30 * time.Second
	}
	if c.Server.WriteTimeout == 0 {
		c.Server.WriteTimeout = 120 * time.Second
	}
	if c.WordPress.URL == "" {
		c.WordPress.URL = "https://wpuiai.com"
	}
	if c.WordPress.RESTNamespace == "" {
		c.WordPress.RESTNamespace = "wpuiai-ai-cloud/v1"
	}
	if c.WordPress.CacheTTL == 0 {
		c.WordPress.CacheTTL = 300
	}
	// No hardcoded fallbacks. Default provider/model comes exclusively
	// from WP admin settings via the /ai-settings REST endpoint.
	// If WP settings are empty, AI calls will fail with a clear error.
	if c.Vision.PoolSize == 0 {
		c.Vision.PoolSize = 3
	}
	if c.Vision.MaxPool == 0 {
		c.Vision.MaxPool = 8
	}
	if c.Vision.Browsers == 0 {
		c.Vision.Browsers = 1
	}
	if c.Vision.PoolPages == 0 {
		c.Vision.PoolPages = c.Server.VisionPoolSize
	}
	if c.Vision.PoolPages == 0 {
		c.Vision.PoolPages = c.Vision.PoolSize
	}
	if c.Vision.ScreenshotQuality == 0 {
		c.Vision.ScreenshotQuality = 65
	}
	if c.EvidenceRegistry.ReconcileInterval == 0 {
		c.EvidenceRegistry.ReconcileInterval = 5 * time.Second
	}
	if c.Storage.DataDir == "" {
		c.Storage.DataDir = "./data"
	}
	if c.Storage.UsageFile == "" {
		c.Storage.UsageFile = "usage.json"
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	for name, profile := range c.TwoFactor.Profiles {
		if profile.Provider == "" {
			profile.Provider = "totp"
		}
		if profile.Digits == 0 {
			profile.Digits = 6
		}
		if profile.Period == 0 {
			profile.Period = 30
		}
		if profile.Algorithm == "" {
			profile.Algorithm = "SHA1"
		}
		c.TwoFactor.Profiles[name] = profile
	}
}

// RESTURL builds a full WP REST API URL for a given path.
func (c *Config) RESTURL(path string) string {
	base := strings.TrimRight(c.WordPress.URL, "/")
	ns := strings.Trim(c.WordPress.RESTNamespace, "/")
	path = strings.TrimLeft(path, "/")
	return fmt.Sprintf("%s/wp-json/%s/%s", base, ns, path)
}

// Addr returns the listen address as host:port.
func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
