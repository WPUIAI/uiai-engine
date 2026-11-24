package captcha

import (
	"path/filepath"
	"strings"
	"testing"
)

// #298: the default stats log file must not assume a host-specific /var/log
// path; an empty default means the server merge owns portability (storage
// data_dir) and the tracker stays cleanly disabled here.
func TestDefaultStatsLogFileIsNotHostSpecific(t *testing.T) {
	cfg := DefaultCaptchaConfig()
	if strings.Contains(cfg.Stats.LogFile, "/var/log/") || filepath.IsAbs(cfg.Stats.LogFile) {
		t.Fatalf("default stats log file %q must not be a host-specific absolute path", cfg.Stats.LogFile)
	}
}
