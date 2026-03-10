package captcha

import "time"

// ─── Request / Response types ───────────────────────────────────────────────

// SolveRequest is the JSON body for POST /api/session/{id}/captcha/solve
type SolveRequest struct {
	Type     string       `json:"type"`               // "text", "recaptcha_v2", "auto"
	Profile  string       `json:"profile,omitempty"`   // per-site profile name from config
	Config   SolveConfig  `json:"config,omitempty"`    // inline config (overrides profile)
	Provider string       `json:"provider,omitempty"`  // AI provider override
	Model    string       `json:"model,omitempty"`     // AI model override
	Voting   *VotingConfig `json:"voting,omitempty"`   // multi-model voting
}

// SolveConfig provides captcha-specific parameters.
type SolveConfig struct {
	ImageSelector  string          `json:"image_selector,omitempty"`
	AnswerSelector string          `json:"answer_selector,omitempty"`
	SubmitSelector string          `json:"submit_selector,omitempty"`
	AutoSubmit     bool            `json:"auto_submit,omitempty"`
	MaxAttempts    int             `json:"max_attempts,omitempty"`
	Hint           string          `json:"hint,omitempty"`
	PromptTemplate string          `json:"prompt_template,omitempty"` // "blind_assistant" | "verification_code"
	Preprocessing  *PreprocessConfig `json:"preprocessing,omitempty"`
	Strategy       string          `json:"strategy,omitempty"` // reCAPTCHA: "full_grid" | "per_tile"
}

// PreprocessConfig for image cleaning before OCR/VLM.
type PreprocessConfig struct {
	Upscale           int `json:"upscale,omitempty" yaml:"upscale,omitempty"`                         // 2-4x
	MedianKernel      int `json:"median_kernel,omitempty" yaml:"median_kernel,omitempty"`             // odd, e.g. 5 — median filter to blur thin grid lines
	Threshold         int `json:"threshold,omitempty" yaml:"threshold,omitempty"`                     // 0-255
	MorphologyKernel  int `json:"morphology_kernel,omitempty" yaml:"morphology_kernel,omitempty"`     // odd, e.g. 3 or 5
	ComponentMinArea  int `json:"component_min_area,omitempty" yaml:"component_min_area,omitempty"`   // min pixel area
	ComponentMaxAspect int `json:"component_max_aspect,omitempty" yaml:"component_max_aspect,omitempty"` // max w/h ratio
}

// VotingConfig for multi-model consensus.
type VotingConfig struct {
	Enabled bool     `json:"enabled"`
	Models  []string `json:"models,omitempty"`
}

// SolveResponse is returned from all solve endpoints.
type SolveResponse struct {
	Solved     bool              `json:"solved"`
	Type       string            `json:"type"`
	Answer     string            `json:"answer,omitempty"`
	Token      string            `json:"token,omitempty"` // reCAPTCHA token
	Attempts   int               `json:"attempts"`
	Rounds     int               `json:"rounds,omitempty"` // reCAPTCHA rounds
	Method     string            `json:"method"`
	DurationMs int64             `json:"duration_ms"`
	Error      string            `json:"error,omitempty"`
	Debug      *SolveDebug       `json:"debug,omitempty"`
}

// SolveDebug contains diagnostic info on failure.
type SolveDebug struct {
	AllAnswers    []string `json:"all_answers,omitempty"`
	Preprocessing string   `json:"preprocessing,omitempty"`
	LastAnswer    string   `json:"last_answer,omitempty"`
}

// ─── Image-only solve (stateless) ──────────────────────────────────────────

// ImageSolveRequest is the JSON body for POST /api/captcha/solve-image
type ImageSolveRequest struct {
	ImageBase64   string          `json:"image_base64"`
	ImageType     string          `json:"image_type,omitempty"` // default "image/png"
	Site          string          `json:"site,omitempty"`       // profile name (e.g. "prlog") — resolves preprocessing/hint/template
	Hint          string          `json:"hint,omitempty"`
	Preprocessing *PreprocessConfig `json:"preprocessing,omitempty"`
	Provider      string          `json:"provider,omitempty"`
	Model         string          `json:"model,omitempty"`
	Voting        bool            `json:"voting,omitempty"`     // multi-model voting
	MultiPass     bool            `json:"multipass,omitempty"`  // multi-pass preprocessing + voting
}

// ImageSolveResponse is returned from solve-image.
type ImageSolveResponse struct {
	Text         string   `json:"text"`
	Confidence   string   `json:"confidence"` // "high", "medium", "low"
	Method       string   `json:"method"`
	Alternatives []string `json:"alternatives,omitempty"`
	DurationMs   int64    `json:"duration_ms"`
}

// ─── Status endpoint ───────────────────────────────────────────────────────

// StatusResponse is returned from GET /api/captcha/status
type StatusResponse struct {
	AvailableTypes []string            `json:"available_types"`
	Backends       map[string]Backend  `json:"backends"`
	Stats          *SolverStats        `json:"stats,omitempty"`
}

type Backend struct {
	Available bool     `json:"available"`
	Provider  string   `json:"provider,omitempty"`
	Models    []string `json:"models,omitempty"`
	Version   string   `json:"version,omitempty"`
	Endpoint  string   `json:"endpoint,omitempty"`
}

// ─── Config types (from config.yaml) ───────────────────────────────────────

// CaptchaConfig is the top-level captcha config in config.yaml.
type CaptchaConfig struct {
	Enabled         bool                     `yaml:"enabled" json:"enabled"`
	DefaultProvider string                   `yaml:"default_provider" json:"default_provider"`
	DefaultModel    string                   `yaml:"default_model" json:"default_model"`
	Text            TextConfig               `yaml:"text" json:"text"`
	Recaptcha       RecaptchaConfig           `yaml:"recaptcha" json:"recaptcha"`
	Voting          VotingConfig             `yaml:"voting" json:"voting"`
	Stealth         StealthConfig            `yaml:"stealth" json:"stealth"`
	Proxy           ProxyConfig              `yaml:"proxy" json:"proxy"`
	Stats           StatsConfig              `yaml:"stats" json:"stats"`
	Profiles        map[string]ProfileConfig `yaml:"profiles" json:"profiles"`
}

type TextConfig struct {
	MaxAttempts    int           `yaml:"max_attempts" json:"max_attempts"`
	RetryDelay     string        `yaml:"retry_delay" json:"retry_delay"`
	PromptTemplate string        `yaml:"prompt_template" json:"prompt_template"`
	FallbackChain  []string      `yaml:"fallback_chain" json:"fallback_chain"`
	Voters         []VoterModel  `yaml:"voters" json:"voters"`
}

type RecaptchaConfig struct {
	Strategy             string `yaml:"strategy" json:"strategy"` // "full_grid" | "per_tile"
	MaxRounds            int    `yaml:"max_rounds" json:"max_rounds"`
	MaxAttempts          int    `yaml:"max_attempts" json:"max_attempts"`
	ActionDelayMinMs     int    `yaml:"action_delay_min_ms" json:"action_delay_min_ms"`
	ActionDelayMaxMs     int    `yaml:"action_delay_max_ms" json:"action_delay_max_ms"`
	VerifyDelayMinMs     int    `yaml:"verify_delay_min_ms" json:"verify_delay_min_ms"`
	VerifyDelayMaxMs     int    `yaml:"verify_delay_max_ms" json:"verify_delay_max_ms"`
	DynamicMaxIterations int    `yaml:"dynamic_max_iterations" json:"dynamic_max_iterations"`
	AudioFallback        bool   `yaml:"audio_fallback" json:"audio_fallback"`
	WhisperURL           string `yaml:"whisper_url" json:"whisper_url"`
}

type StealthConfig struct {
	PatchWebdriver  bool     `yaml:"patch_webdriver" json:"patch_webdriver"`
	RandomUserAgent bool     `yaml:"random_user_agent" json:"random_user_agent"`
	UserAgents      []string `yaml:"user_agents" json:"user_agents"`
}

type StatsConfig struct {
	Enabled bool   `yaml:"enabled" json:"enabled"`
	LogFile string `yaml:"log_file" json:"log_file"`
}

type ProfileConfig struct {
	Type           string          `yaml:"type" json:"type"`
	ImageSelector  string          `yaml:"image_selector" json:"image_selector"`
	AnswerSelector string          `yaml:"answer_selector" json:"answer_selector"`
	SubmitSelector string          `yaml:"submit_selector" json:"submit_selector"`
	SiteKey        string          `yaml:"site_key" json:"site_key"`
	Strategy       string          `yaml:"strategy" json:"strategy"`
	VotingEnabled  bool            `yaml:"voting" json:"voting"`
	Hint           string          `yaml:"hint" json:"hint"`
	PromptTemplate string          `yaml:"prompt_template" json:"prompt_template"`
	Preprocessing  *PreprocessConfig `yaml:"preprocessing" json:"preprocessing"`
}

// ─── Stats tracking ────────────────────────────────────────────────────────

type SolverStats struct {
	TotalAttempts int                      `json:"total_attempts"`
	TotalSolved   int                      `json:"total_solved"`
	SuccessRate   float64                  `json:"success_rate"`
	ByType        map[string]*TypeStats    `json:"by_type"`
}

type TypeStats struct {
	Attempts int     `json:"attempts"`
	Solved   int     `json:"solved"`
	Rate     float64 `json:"rate"`
}

// StatsEntry is a single log line in the stats JSONL file.
type StatsEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	Type       string    `json:"type"`
	Solved     bool      `json:"solved"`
	Attempts   int       `json:"attempts"`
	DurationMs int64     `json:"duration_ms"`
	Method     string    `json:"method"`
	Profile    string    `json:"profile,omitempty"`
	Error      string    `json:"error,omitempty"`
}

// ─── Defaults ──────────────────────────────────────────────────────────────

func DefaultCaptchaConfig() CaptchaConfig {
	return CaptchaConfig{
		Enabled:         true,
		DefaultProvider: "openrouter",
		DefaultModel:    "google/gemini-2.0-flash-001",
		Text: TextConfig{
			MaxAttempts:    5,
			RetryDelay:     "1s",
			PromptTemplate: "blind_assistant",
			FallbackChain:  []string{"vlm", "tesseract"},
		},
		Recaptcha: RecaptchaConfig{
			Strategy:             "full_grid",
			MaxRounds:            4,
			MaxAttempts:          4,
			ActionDelayMinMs:     200,
			ActionDelayMaxMs:     800,
			VerifyDelayMinMs:     500,
			VerifyDelayMaxMs:     1000,
			DynamicMaxIterations: 4,
			AudioFallback:        true,
			WhisperURL:           "http://localhost:8115",
		},
		Voting: VotingConfig{
			Enabled: false,
			Models: []string{
				"google/gemini-2.0-flash-001",
				"anthropic/claude-3.5-sonnet",
			},
		},
		Stealth: StealthConfig{
			PatchWebdriver:  true,
			RandomUserAgent: true,
			UserAgents: []string{
				"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
				"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
				"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
			},
		},
		Stats: StatsConfig{
			Enabled: true,
			LogFile: "/var/log/uiai/captcha-stats.jsonl",
		},
		Profiles: map[string]ProfileConfig{
			"prlog": {
				Type:           "text",
				ImageSelector:  "img[alt*='human'], img[alt*='Human']",
				AnswerSelector: "input[name='captcha_hash']",
				SubmitSelector: "button[value='Create']",
				Hint:           "Read the distorted text in the image. The text contains exactly 5 lowercase letters (a-z only, no digits). The image has crosshatch grid lines overlaying the text — ignore the grid lines and focus only on the thick dark letter shapes. Output ONLY the 5 letters, nothing else.",
				PromptTemplate: "lowercase_captcha",
				VotingEnabled:  true,
				// No preprocessing — VLMs read raw captchas better than preprocessed
			},
			"openpr": {
				Type:           "text",
				ImageSelector:  "img.captcha-image",
				AnswerSelector: "input[name='captcha']",
				Hint:           "Verification code: uppercase letters and digits only.",
				PromptTemplate: "verification_code",
			},
			"prcom": {
				Type:     "recaptcha_v2",
				SiteKey:  "6LcOef8SAAAAANHRO0asSp6bjrMFbIT105J3b2ow",
				Strategy: "full_grid",
			},
			"247pressrelease": {
				Type:     "recaptcha_v2",
				Strategy: "full_grid",
			},
		},
	}
}
