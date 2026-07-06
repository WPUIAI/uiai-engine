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
	Server     ServerConfig    `yaml:"server"`
	WordPress  WordPressConfig `yaml:"wordpress"`
	AI         AIConfig        `yaml:"ai"`
	Vision     VisionConfig    `yaml:"vision"`
	Credits    CreditsConfig   `yaml:"credits"`
	RateLimits RateLimitConfig `yaml:"rate_limits"`
	Storage    StorageConfig   `yaml:"storage"`
	Logging    LoggingConfig   `yaml:"logging"`
	CORS       CORSConfig      `yaml:"cors"`
	Media      MediaConfig     `yaml:"media"`
	Captcha    CaptchaYAML     `yaml:"captcha"`
	TwoFactor  TwoFactorConfig `yaml:"two_factor"`
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
