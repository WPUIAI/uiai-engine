package vision

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/go-rod/rod"
)

// PaceConfig is opt-in session timing, not an authentication/control grant.
// No profile (or instant) retains the existing action path and diagnostics shape.
type PaceConfig struct {
	Mode       string  `json:"mode"`
	MedianMS   int     `json:"median_ms,omitempty"`
	MinClickMS int     `json:"min_click_ms,omitempty"`
	CPSMin     float64 `json:"cps_min,omitempty"`
	CPSMax     float64 `json:"cps_max,omitempty"`
	Seed       *int64  `json:"seed,omitempty"`
}

type PaceStats struct {
	Mode          string  `json:"mode"`
	Seeded        bool    `json:"seeded"`
	ActionWaits   int64   `json:"action_waits"`
	KeyWaits      int64   `json:"key_waits"`
	CanceledWaits int64   `json:"canceled_waits"`
	ScheduledMS   float64 `json:"scheduled_ms"`
	ElapsedMS     float64 `json:"elapsed_ms"`
}

type sessionPacer struct {
	config PaceConfig
	random *rand.Rand
	stats  PaceStats
}

func newSessionPacer(config PaceConfig) (*sessionPacer, error) {
	if config.Mode == "" || config.Mode == "instant" {
		if config.MedianMS != 0 || config.MinClickMS != 0 || config.CPSMin != 0 || config.CPSMax != 0 || config.Seed != nil {
			return nil, fmt.Errorf("instant pace does not accept timing options")
		}
		return nil, nil
	}
	if config.Mode != "human" && config.Mode != "custom" {
		return nil, fmt.Errorf("unsupported pace mode")
	}
	if config.MedianMS == 0 {
		config.MedianMS = 200
	}
	if config.MinClickMS == 0 {
		config.MinClickMS = 50
	}
	if config.CPSMin == 0 {
		config.CPSMin = 8
	}
	if config.CPSMax == 0 {
		config.CPSMax = 16
	}
	if config.MedianMS < 1 || config.MedianMS > 2000 || config.MinClickMS < 0 || config.MinClickMS > 2000 {
		return nil, fmt.Errorf("pace timing must be within 1..2000ms (click minimum may be zero)")
	}
	if math.IsNaN(config.CPSMin) || math.IsNaN(config.CPSMax) || math.IsInf(config.CPSMin, 0) || math.IsInf(config.CPSMax, 0) || config.CPSMin < 1 || config.CPSMax < config.CPSMin || config.CPSMax > 1000 {
		return nil, fmt.Errorf("pace cps range must be finite, ordered and within [1,1000]")
	}
	seed := time.Now().UnixNano()
	seeded := config.Seed != nil
	if seeded {
		seed = *config.Seed
	}
	config.Seed = nil // Do not retain caller-owned pointers or publish a seed.
	return &sessionPacer{config: config, random: rand.New(rand.NewSource(seed)), stats: PaceStats{Mode: config.Mode, Seeded: seeded}}, nil
}

// SetPace changes only this session's optional timing profile. Validation failure
// preserves the previous profile; this cannot resolve a pending auth challenge.
func (s *Session) SetPace(config PaceConfig) error {
	pacer, err := newSessionPacer(config)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.authChallenge != nil {
		return fmt.Errorf("auth challenge pending")
	}
	s.pacer = pacer
	return nil
}

func (s *Session) Pacing() *PaceStats {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pacingStatsLocked()
}

func (s *Session) pacingStatsLocked() *PaceStats {
	if s.pacer == nil {
		return nil
	}
	copy := s.pacer.stats
	return &copy
}

func (p *sessionPacer) actionDelay(kind string) time.Duration {
	minimum := 0.0
	if kind == "click" {
		minimum = float64(p.config.MinClickMS)
	}
	ms := float64(p.config.MedianMS) * math.Exp(.35*p.random.NormFloat64())
	return time.Duration(math.Min(2000, math.Max(minimum, ms)) * float64(time.Millisecond))
}

func (p *sessionPacer) keyDelay() time.Duration {
	cps := p.config.CPSMin + p.random.Float64()*(p.config.CPSMax-p.config.CPSMin)
	return time.Duration(float64(time.Second) / cps)
}

// Caller holds the session lock; waits share the existing action context and do
// not create asynchronous input, background retries or post-cancel dispatch.
func (s *Session) waitPaceLocked(ctx context.Context, kind string) error {
	if s.pacer == nil {
		return nil
	}
	if s.authChallenge != nil {
		return fmt.Errorf("auth challenge pending")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return s.waitPaceDelayLocked(ctx, s.pacer.actionDelay(kind), false)
}

func (s *Session) waitPaceDelayLocked(ctx context.Context, delay time.Duration, key bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	p := s.pacer
	if key {
		p.stats.KeyWaits++
	} else {
		p.stats.ActionWaits++
	}
	p.stats.ScheduledMS += float64(delay) / float64(time.Millisecond)
	start := time.Now()
	err := waitSessionDelay(ctx, delay)
	if err != nil {
		p.stats.CanceledWaits++
	}
	p.stats.ElapsedMS += float64(time.Since(start)) / float64(time.Millisecond)
	return err
}

func (s *Session) pacedInputLocked(ctx context.Context, element *rod.Element, text string) error {
	if text == "" {
		return element.Input("")
	}
	first := true
	for _, character := range text {
		if !first {
			if err := s.waitPaceDelayLocked(ctx, s.pacer.keyDelay(), true); err != nil {
				return err
			}
		}
		first = false
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := element.Input(string(character)); err != nil {
			return err
		}
	}
	return nil
}
