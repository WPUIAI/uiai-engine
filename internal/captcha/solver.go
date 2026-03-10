package captcha

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/philoveracity/uiai-engine/internal/ai"
	"github.com/philoveracity/uiai-engine/internal/vision"
)

// Solver is the main captcha solving engine.
type Solver struct {
	AI     *ai.Provider
	Config CaptchaConfig
	Stats  *StatsTracker
	pool   *IPPool // IP rotation pool for proxied solves
}

// NewSolver creates a Solver with the given config.
func NewSolver(aiProv *ai.Provider, cfg CaptchaConfig) *Solver {
	s := &Solver{
		AI:     aiProv,
		Config: cfg,
		Stats:  NewStatsTracker(cfg.Stats),
	}
	// Initialize IP pool if proxy is enabled
	if cfg.Proxy.Enabled && (len(cfg.Proxy.LocalIPs) > 0 || len(cfg.Proxy.Proxies) > 0) {
		s.pool = NewIPPool(cfg.Proxy)
	}
	return s
}

// Pool returns the IP pool for API access (status, add/remove).
func (s *Solver) Pool() *IPPool { return s.pool }

// SolveInSession solves a captcha within an active browser session.
func (s *Solver) SolveInSession(ctx context.Context, sess *vision.Session, req SolveRequest) *SolveResponse {
	start := time.Now()

	// Resolve config from profile or inline
	cfg := s.resolveConfig(req)
	captchaType := s.resolveType(req, sess)

	log.Printf("[captcha] solving type=%s profile=%s provider=%s", captchaType, req.Profile, s.Config.DefaultProvider)

	var resp *SolveResponse
	switch captchaType {
	case "text":
		resp = s.solveTextInSession(ctx, sess, cfg)
	case "recaptcha_v2":
		resp = s.SolveRecaptchaV2(ctx, sess, cfg)
	default:
		resp = &SolveResponse{
			Solved: false,
			Type:   captchaType,
			Error:  fmt.Sprintf("unsupported captcha type: %s", captchaType),
			Method: "none",
		}
	}

	resp.DurationMs = time.Since(start).Milliseconds()
	resp.Type = captchaType

	// Record stats
	s.Stats.Record(StatsEntry{
		Type:       captchaType,
		Solved:     resp.Solved,
		Attempts:   resp.Attempts,
		DurationMs: resp.DurationMs,
		Method:     resp.Method,
		Profile:    req.Profile,
		Error:      resp.Error,
	})

	return resp
}

// SolveImage solves a captcha from a raw image (no browser session).
func (s *Solver) SolveImage(ctx context.Context, req ImageSolveRequest) (*ImageSolveResponse, error) {
	start := time.Now()

	imgType := req.ImageType
	if imgType == "" {
		imgType = "image/png"
	}

	cfg := SolveConfig{
		Hint:          req.Hint,
		Preprocessing: req.Preprocessing,
	}

	var result *ImageSolveResponse
	var err error

	switch {
	case req.MultiPass:
		// Multi-pass: 3 preprocessing variants × multi-model voting
		result, err = SolveTextMultiPass(ctx, s.AI, req.ImageBase64, imgType, cfg, s.Config)

	case req.Voting:
		// Single preprocessing + multi-model voting
		processed := req.ImageBase64
		if req.Preprocessing != nil {
			p, pErr := Preprocess(req.ImageBase64, imgType, req.Preprocessing)
			if pErr != nil {
				log.Printf("[captcha] preprocess failed, using raw: %v", pErr)
			} else {
				processed = p
			}
		}
		prompt := buildTextPrompt(cfg)
		voters := s.Config.Text.Voters
		if len(voters) == 0 {
			voters = DefaultVoters()
		}
		best, alts, vErr := VoteTextCaptcha(ctx, s.AI, processed, imgType, prompt, voters)
		err = vErr
		result = &ImageSolveResponse{
			Text:         best,
			Confidence:   "high",
			Method:       "vote",
			Alternatives: alts,
		}

	default:
		// Single model, single pass
		processed := req.ImageBase64
		if req.Preprocessing != nil {
			p, pErr := Preprocess(req.ImageBase64, imgType, req.Preprocessing)
			if pErr != nil {
				log.Printf("[captcha] preprocess failed, using raw: %v", pErr)
			} else {
				processed = p
			}
		}
		result, err = SolveTextCaptcha(ctx, s.AI, processed, imgType, cfg, s.Config)
	}

	if result != nil {
		result.DurationMs = time.Since(start).Milliseconds()
	}

	// Record stats
	method := ""
	if result != nil {
		method = result.Method
	}
	solved := err == nil && result != nil && result.Text != ""
	s.Stats.Record(StatsEntry{
		Type:       "text",
		Solved:     solved,
		Attempts:   1,
		DurationMs: time.Since(start).Milliseconds(),
		Method:     method,
	})

	return result, err
}

// GetStatus returns solver capabilities.
func (s *Solver) GetStatus() *StatusResponse {
	resp := &StatusResponse{
		AvailableTypes: []string{"text", "recaptcha_v2"},
		Backends:       make(map[string]Backend),
		Stats:          s.Stats.Get(),
	}

	// VLM
	resp.Backends["vlm"] = Backend{
		Available: true,
		Provider:  s.Config.DefaultProvider,
		Models:    []string{s.Config.DefaultModel},
	}

	// Tesseract
	if _, err := exec.LookPath("tesseract"); err == nil {
		resp.Backends["tesseract"] = Backend{Available: true, Version: "installed"}
	} else {
		resp.Backends["tesseract"] = Backend{Available: false}
	}

	// ddddocr
	cmd := exec.Command("python3", "-c", "import ddddocr; print('ok')")
	if out, err := cmd.Output(); err == nil && strings.TrimSpace(string(out)) == "ok" {
		resp.Backends["ddddocr"] = Backend{Available: true}
	} else {
		resp.Backends["ddddocr"] = Backend{Available: false}
	}

	// Whisper
	if s.Config.Recaptcha.WhisperURL != "" {
		resp.Backends["whisper"] = Backend{
			Available: true,
			Endpoint:  s.Config.Recaptcha.WhisperURL,
		}
	}

	// IP Pool
	if s.pool != nil {
		ps := s.pool.Status()
		healthy := 0
		for _, ip := range ps.IPs {
			if ip.Status == "healthy" {
				healthy++
			}
		}
		resp.Backends["ip_pool"] = Backend{
			Available: true,
			Version:   fmt.Sprintf("%d total, %d healthy, strategy=%s", ps.TotalIPs, healthy, ps.Strategy),
		}
	} else {
		resp.Backends["ip_pool"] = Backend{
			Available: false,
			Version:   "disabled",
		}
	}

	return resp
}

// ─── Text captcha in session ───────────────────────────────────────────────

func (s *Solver) solveTextInSession(ctx context.Context, sess *vision.Session, cfg SolveConfig) *SolveResponse {
	maxAttempts := cfg.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = s.Config.Text.MaxAttempts
	}
	if maxAttempts <= 0 {
		maxAttempts = 5
	}

	retryDelay := parseRetryDelay(s.Config.Text.RetryDelay)

	var allAnswers []string

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.Printf("[captcha] text attempt %d/%d", attempt, maxAttempts)

		// 1. Extract captcha image from DOM
		imgBase64, err := extractCaptchaImage(sess, cfg.ImageSelector)
		if err != nil {
			return &SolveResponse{
				Solved:   false,
				Attempts: attempt,
				Error:    fmt.Sprintf("extract image: %v", err),
				Method:   "none",
			}
		}

		// 2. Preprocess
		processed := imgBase64
		if cfg.Preprocessing != nil {
			p, err := Preprocess(imgBase64, "image/png", cfg.Preprocessing)
			if err != nil {
				log.Printf("[captcha] preprocess failed, using raw: %v", err)
			} else {
				processed = p
			}
		}

		// 3. Solve via voting or fallback chain
		// Use multi-pass voting in session mode for higher accuracy
		result, solveErr := SolveTextMultiPass(ctx, s.AI, imgBase64, "image/png", cfg, s.Config)
		if solveErr != nil || result == nil || result.Text == "" {
			// Fallback to single-model
			result, solveErr = SolveTextCaptcha(ctx, s.AI, processed, "image/png", cfg, s.Config)
		}
		_ = solveErr
		if err != nil || result == nil || result.Text == "" {
			log.Printf("[captcha] attempt %d: no answer from solver", attempt)
			if attempt < maxAttempts {
				jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
				time.Sleep(retryDelay + jitter)
			}
			continue
		}

		answer := result.Text
		allAnswers = append(allAnswers, answer)
		log.Printf("[captcha] attempt %d: answer=%q method=%s", attempt, answer, result.Method)

		// 4. Fill the answer field
		if cfg.AnswerSelector != "" {
			_, err := sess.Fill(cfg.AnswerSelector, answer)
			if err != nil {
				log.Printf("[captcha] fill answer field failed: %v", err)
				// Try via eval as fallback
				js := fmt.Sprintf(`var el = document.querySelector(%q); if(el){el.value=%q;el.dispatchEvent(new Event('input',{bubbles:true}));} return el?"ok":"not_found"`,
					cfg.AnswerSelector, answer)
				sess.Eval(js)
			}
		}

		// 5. Auto-submit if requested
		if cfg.AutoSubmit && cfg.SubmitSelector != "" {
			time.Sleep(300 * time.Millisecond)
			sess.Eval(fmt.Sprintf(`var btn = document.querySelector(%q); if(btn) btn.click(); return btn?"clicked":"not_found"`, cfg.SubmitSelector))
			time.Sleep(2 * time.Second)

			// Check for error indicators
			errorJS := `return document.body.innerText.includes("Invalid") || document.body.innerText.includes("issue found") ? "FAIL" : "OK"`
			checkResult, _, _ := sess.Eval(errorJS)
			if strings.Contains(checkResult, "FAIL") {
				log.Printf("[captcha] attempt %d: rejected by site", attempt)
				if attempt < maxAttempts {
					jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
					time.Sleep(retryDelay + jitter)
				}
				continue
			}
		}

		// Success
		return &SolveResponse{
			Solved:   true,
			Answer:   answer,
			Attempts: attempt,
			Method:   result.Method,
			Debug: &SolveDebug{
				AllAnswers: allAnswers,
			},
		}
	}

	return &SolveResponse{
		Solved:   false,
		Attempts: maxAttempts,
		Error:    "max attempts exceeded",
		Method:   "vlm",
		Debug: &SolveDebug{
			AllAnswers: allAnswers,
			LastAnswer: lastOrEmpty(allAnswers),
		},
	}
}

// ─── DOM helpers ───────────────────────────────────────────────────────────

// extractCaptchaImage extracts a captcha image from the page DOM.
// Supports: <img src="data:...">, <img src="http...">, or canvas.
func extractCaptchaImage(sess *vision.Session, selector string) (string, error) {
	if selector == "" {
		selector = "img[src^='data:image']"
	}

	// Try to get image as base64 via canvas
	js := fmt.Sprintf(`
		var el = document.querySelector(%q);
		if (!el) return "ERR:not_found";
		if (el.tagName === "CANVAS") {
			return el.toDataURL("image/png").split(",")[1];
		}
		if (el.tagName === "IMG") {
			var c = document.createElement("canvas");
			c.width = el.naturalWidth || el.width;
			c.height = el.naturalHeight || el.height;
			c.getContext("2d").drawImage(el, 0, 0);
			return c.toDataURL("image/png").split(",")[1];
		}
		return "ERR:unsupported_element";
	`, selector)

	result, _, err := sess.Eval(js)
	if err != nil {
		return "", fmt.Errorf("eval: %w", err)
	}
	if strings.HasPrefix(result, "ERR:") {
		return "", fmt.Errorf("DOM: %s", result[4:])
	}
	if result == "" || result == "null" || result == "<nil>" {
		return "", fmt.Errorf("empty image data")
	}
	return result, nil
}

// ─── Config resolution ─────────────────────────────────────────────────────

func (s *Solver) resolveConfig(req SolveRequest) SolveConfig {
	cfg := req.Config

	// Apply profile defaults if profile is set
	if req.Profile != "" {
		if profile, ok := s.Config.Profiles[req.Profile]; ok {
			if cfg.ImageSelector == "" {
				cfg.ImageSelector = profile.ImageSelector
			}
			if cfg.AnswerSelector == "" {
				cfg.AnswerSelector = profile.AnswerSelector
			}
			if cfg.SubmitSelector == "" {
				cfg.SubmitSelector = profile.SubmitSelector
			}
			if cfg.Hint == "" {
				cfg.Hint = profile.Hint
			}
			if cfg.PromptTemplate == "" {
				cfg.PromptTemplate = profile.PromptTemplate
			}
			if cfg.Preprocessing == nil {
				cfg.Preprocessing = profile.Preprocessing
			}
			if cfg.Strategy == "" {
				cfg.Strategy = profile.Strategy
			}
		}
	}
	return cfg
}

func (s *Solver) resolveType(req SolveRequest, sess *vision.Session) string {
	if req.Type != "" && req.Type != "auto" {
		return req.Type
	}
	// Check profile
	if req.Profile != "" {
		if profile, ok := s.Config.Profiles[req.Profile]; ok && profile.Type != "" {
			return profile.Type
		}
	}
	// Auto-detect via DOM
	if sess != nil {
		result, _, _ := sess.Eval(`
			if (document.querySelector("iframe[title='reCAPTCHA']")) return "recaptcha_v2";
			if (document.querySelector("[data-sitekey]")) return "recaptcha_v2";
			if (document.querySelector(".cf-turnstile")) return "turnstile";
			return "text";
		`)
		if result != "" && result != "<nil>" {
			return result
		}
	}
	return "text"
}

// ─── Utilities ─────────────────────────────────────────────────────────────

func parseRetryDelay(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 1 * time.Second
	}
	return d
}

func lastOrEmpty(ss []string) string {
	if len(ss) == 0 {
		return ""
	}
	return ss[len(ss)-1]
}

func decodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

func writeTempFile(data []byte, pattern string) (string, error) {
	f, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	defer f.Close()
	_, err = f.Write(data)
	return f.Name(), err
}

func removeTempFile(path string) {
	os.Remove(path)
}
