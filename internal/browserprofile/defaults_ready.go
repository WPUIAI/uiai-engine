package browserprofile

// ReadyDefaultConfig returns defaults that pass registry validation and can be
// loaded immediately by the Go core. Operator paths remain configurable.
func ReadyDefaultConfig() Config {
	cfg := DefaultConfig()

	noDetect := cfg.Profiles["no_detect"]
	// When no explicit UA catalog is configured, retain the runtime browser UA.
	// Profile consistency is more important than randomizing one field alone.
	noDetect.Identity.RandomUserAgent = false
	cfg.Profiles["no_detect"] = noDetect

	operator := cfg.Profiles["operator"]
	operator.Launch.UserDataDir = "~/.config/uiai/operator-profile"
	cfg.Profiles["operator"] = operator

	return cfg
}
