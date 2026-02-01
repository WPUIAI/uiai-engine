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
	Server     ServerConfig     `yaml:"server"`
	WordPress  WordPressConfig  `yaml:"wordpress"`
	AI         AIConfig         `yaml:"ai"`
	Vision     VisionConfig     `yaml:"vision"`
	Credits    CreditsConfig    `yaml:"credits"`
	RateLimits RateLimitConfig  `yaml:"rate_limits"`
	Storage    StorageConfig    `yaml:"storage"`
	Logging    LoggingConfig    `yaml:"logging"`
	CORS       CORSConfig       `yaml:"cors"`
}

type ServerConfig struct {
	Port         int           `yaml:"port"`
	Host         string        `yaml:"host"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
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

type VisionConfig struct {
	PoolSize          int           `yaml:"pool_size"`
	MaxPool           int           `yaml:"max_pool"`
	IdleTimeout       time.Duration `yaml:"idle_timeout"`
	ScreenshotQuality int           `yaml:"screenshot_quality"`
	ShareDir          string        `yaml:"share_dir"`
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
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	// Expand ${ENV_VAR} references
	expanded := os.Expand(string(data), func(key string) string {
		return os.Getenv(key)
	})

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
	if c.AI.DefaultModel == "" {
		c.AI.DefaultModel = "claude-sonnet-4-20250514"
	}
	if c.AI.DefaultProvider == "" {
		c.AI.DefaultProvider = "anthropic"
	}
	if c.Vision.PoolSize == 0 {
		c.Vision.PoolSize = 3
	}
	if c.Vision.MaxPool == 0 {
		c.Vision.MaxPool = 8
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
