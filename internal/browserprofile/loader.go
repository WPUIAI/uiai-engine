package browserprofile

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadFile reads only the browser section from the engine YAML config. This
// keeps browser profile loading independently testable while the main config
// package migrates to the canonical profile types.
func LoadFile(path string) (*Registry, error) {
	data, err := os.ReadFile(path) // #nosec G304 -- explicit operator config path.
	if err != nil {
		return nil, fmt.Errorf("read browser profile config %s: %w", path, err)
	}
	var root struct {
		Browser Config `yaml:"browser"`
	}
	if err := yaml.Unmarshal([]byte(os.ExpandEnv(string(data))), &root); err != nil {
		return nil, fmt.Errorf("parse browser profile config %s: %w", path, err)
	}
	cfg := mergeConfigDefaults(root.Browser)
	return NewRegistry(cfg)
}

func mergeConfigDefaults(user Config) Config {
	defaults := ReadyDefaultConfig()
	if user.DefaultProfile != "" {
		defaults.DefaultProfile = user.DefaultProfile
	}
	if len(user.Profiles) > 0 {
		for name, profile := range user.Profiles {
			defaults.Profiles[name] = profile
		}
	}
	if user.DomainRules != nil {
		defaults.DomainRules = user.DomainRules
	}
	return defaults
}
