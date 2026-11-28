package vision

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/go-rod/rod/lib/launcher"
)

// poolLaunchSettings is immutable after construction and does not affect
// per-session viewport, storage isolation, or remote owner-browser settings.
type poolLaunchSettings struct {
	headed     bool
	windowSize string
	locale     string
}

func resolvePoolLaunchSettings(cfg PoolConfig) (poolLaunchSettings, error) {
	s := poolLaunchSettings{locale: strings.TrimSpace(cfg.Locale)}
	if cfg.Headless != nil {
		s.headed = !*cfg.Headless
	}
	if raw := strings.TrimSpace(cfg.WindowSize); raw != "" {
		parts := strings.Split(raw, ",")
		if len(parts) != 2 {
			return s, fmt.Errorf("vision.window_size must be WIDTH,HEIGHT")
		}
		w, e1 := strconv.Atoi(strings.TrimSpace(parts[0]))
		h, e2 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if e1 != nil || e2 != nil || w < 1 || h < 1 || w > 16384 || h > 16384 {
			return s, fmt.Errorf("vision.window_size dimensions must be integers between 1 and 16384")
		}
		s.windowSize = fmt.Sprintf("%d,%d", w, h)
	}
	for _, c := range s.locale {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			return s, fmt.Errorf("vision.locale must be a locale tag")
		}
	}
	return s, nil
}

func (s poolLaunchSettings) apply(l *launcher.Launcher) *launcher.Launcher {
	l = l.Headless(!s.headed)
	if s.windowSize != "" {
		l = l.Set("window-size", s.windowSize)
	}
	if s.locale != "" {
		l = l.Set("lang", s.locale)
	}
	return l
}
