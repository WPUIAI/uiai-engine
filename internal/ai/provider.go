package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

// DefaultAITimeout is applied when the caller's context has no deadline.
const DefaultAITimeout = 60 * time.Second

type Response struct {
	Content      string  `json:"content"`
	Model        string  `json:"model"`
	InputTokens  int     `json:"inputTokens"`
	OutputTokens int     `json:"outputTokens"`
	CostUSD      float64 `json:"costUSD"`
}

type Request struct {
	Provider    string  `json:"provider"`
	Model       string  `json:"model"`
	Prompt      string  `json:"prompt"`
	MaxTokens   int     `json:"maxTokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	ImageBase64 string  `json:"imageBase64,omitempty"`
	ImageType   string  `json:"imageType,omitempty"` // image/jpeg, image/png
}

var modelCosts = map[string][2]float64{
	// Anthropic
	"claude-sonnet-4-20250514":   {0.000003, 0.000015},
	"claude-opus-4-20250514":     {0.000015, 0.000075},
	"claude-3-5-sonnet-20241022": {0.000003, 0.000015},
	// OpenAI
	"gpt-4o":               {0.0000025, 0.00001},
	"gpt-4o-mini":          {0.00000015, 0.0000006},
	"gpt-4-turbo":          {0.00001, 0.00003},
	"gpt-4-vision-preview": {0.00001, 0.00003}, // GPT-4 Vision
	"o1":                   {0.000015, 0.00006},
	"o1-mini":              {0.000003, 0.000012},
	// OpenRouter passthrough — costs come from OpenRouter pricing
	"anthropic/claude-sonnet-4": {0.000003, 0.000015},
	"anthropic/claude-opus-4":   {0.000015, 0.000075},
	"google/gemini-2.0-flash":   {0.0000001, 0.0000004},
	"meta-llama/llama-3.1-405b": {0.000003, 0.000003},
	// Fireworks
	"accounts/fireworks/models/llama-v3p1-405b-instruct": {0.000003, 0.000003},
	"accounts/fireworks/models/llama-v3p1-70b-instruct":  {0.0000009, 0.0000009},
	"accounts/fireworks/models/llama-v3p1-8b-instruct":   {0.0000002, 0.0000002},
	"accounts/fireworks/models/qwen2p5-72b-instruct":     {0.0000009, 0.0000009},
	// Kimi (Moonshot)
	"moonshot-v1-128k": {0.00006, 0.00006},
	"moonshot-v1-32k":  {0.000024, 0.000024},
	"moonshot-v1-8k":   {0.000012, 0.000012},
	// Qwen (Alibaba DashScope)
	"qwen-max":     {0.000016, 0.000016},
	"qwen-plus":    {0.000004, 0.000012},
	"qwen-turbo":   {0.000002, 0.000006},
	"qwen-vl-max":  {0.00001, 0.00001},
	"qwen-vl-plus": {0.000008, 0.000008},
}

type aiKeys struct {
	Anthropic  string
	OpenAI     string
	OpenRouter string
	Fireworks  string
	Kimi       string
	MiniMax    string
	Qwen       string
	fetchedAt  time.Time
}

// ProviderModel describes an available model from WP settings.
type ProviderModel struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Default  bool   `json:"default,omitempty"`
}

type Provider struct {
	cfg      *config.Config
	keys     aiKeys
	keysMu   sync.RWMutex
	keysTTL  time.Duration
	client   *http.Client
	models   []ProviderModel // populated from WP settings
	modelsMu sync.RWMutex
}

// AvailableModels returns the model list built from WP settings.
// Returns empty slice if settings haven't been fetched yet.
func (p *Provider) AvailableModels() []ProviderModel {
	p.modelsMu.RLock()
	defer p.modelsMu.RUnlock()
	return p.models
}

func NewProvider(cfg *config.Config) *Provider {
	prov := &Provider{
		cfg:     cfg,
		keysTTL: time.Duration(cfg.WordPress.CacheTTL) * time.Second,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
	// Eagerly fetch settings from WP on startup so default provider/model
	// are available before the first AI call.
	if err := prov.ensureKeys(); err != nil {
		log.Printf("[ai] WARNING: failed to fetch WP settings on startup: %v", err)
		log.Printf("[ai] AI calls will fail until WP settings are reachable")
	}
	return prov
}

func (p *Provider) Complete(ctx context.Context, req Request) (*Response, error) {
	// Apply default deadline if caller didn't set one
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultAITimeout)
		defer cancel()
	}

	if err := p.ensureKeys(); err != nil {
		return nil, fmt.Errorf("fetch AI keys: %w", err)
	}

	if req.MaxTokens == 0 {
		req.MaxTokens = 4096
	}
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}
	if req.Model == "" {
		req.Model = p.cfg.AI.DefaultModel
	}
	if req.Provider == "" {
		req.Provider = p.cfg.AI.DefaultProvider
	}
	if req.Provider == "" {
		return nil, fmt.Errorf("no AI provider configured — set default_provider in WP Settings or UIAI_DEFAULT_PROVIDER env")
	}
	if req.Model == "" {
		return nil, fmt.Errorf("no AI model configured — set default_model in WP Settings or UIAI_DEFAULT_MODEL env")
	}

	// Try the primary provider first
	resp, err := p.completeOnce(ctx, req)
	if err == nil {
		return resp, nil
	}

	// Only failover on retryable errors (5xx, 429, timeout, connection)
	if !isRetryableError(err) {
		return nil, err
	}

	// Build failover chain from configured providers (skip the one that just failed)
	fallbacks := p.buildFallbackChain(req.Provider)
	if len(fallbacks) == 0 {
		return nil, err // no fallbacks configured
	}

	for i, fb := range fallbacks {
		log.Printf("[ai] %s failed (%v), failing over to %s (attempt %d/%d)",
			req.Provider, summarizeError(err), fb.provider, i+1, len(fallbacks))

		fbReq := req
		fbReq.Provider = fb.provider
		fbReq.Model = fb.model

		// Short backoff between retries
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("failover aborted (context cancelled): %w", ctx.Err())
		case <-time.After(time.Duration(500*(i+1)) * time.Millisecond):
		}

		resp, err = p.completeOnce(ctx, fbReq)
		if err == nil {
			log.Printf("[ai] failover to %s succeeded (model: %s)", fb.provider, fb.model)
			return resp, nil
		}
		if !isRetryableError(err) {
			return nil, err
		}
	}

	return nil, fmt.Errorf("all providers failed, last error: %w", err)
}

// completeOnce makes a single attempt to the specified provider.
func (p *Provider) completeOnce(ctx context.Context, req Request) (*Response, error) {
	switch req.Provider {
	case "anthropic", "kimi":
		return p.anthropicComplete(ctx, req)
	case "openai", "openrouter", "fireworks", "minimax", "qwen":
		return p.openaiComplete(ctx, req)
	default:
		return nil, fmt.Errorf("unknown provider %q", req.Provider)
	}
}

// fallbackProvider pairs a provider name with its configured model.
type fallbackProvider struct {
	provider string
	model    string
}

// buildFallbackChain returns providers to try after the primary fails.
// Only includes providers that have API keys configured.
func (p *Provider) buildFallbackChain(skipProvider string) []fallbackProvider {
	p.keysMu.RLock()
	defer p.keysMu.RUnlock()

	type candidate struct {
		provider string
		key      string
		model    string
	}
	// Priority: subscribed/direct providers only. OpenRouter intentionally excluded from automatic fallback.
	all := []candidate{
		{"minimax", p.keys.MiniMax, "MiniMax-M2.5"},
		{"kimi", p.keys.Kimi, "kimi-k2.5"},
		{"anthropic", p.keys.Anthropic, "claude-sonnet-4-20250514"},
		{"openai", p.keys.OpenAI, "gpt-4o"},
		{"fireworks", p.keys.Fireworks, "accounts/fireworks/models/llama-v3p1-70b-instruct"},
	}

	var chain []fallbackProvider
	for _, c := range all {
		if c.provider == skipProvider || c.key == "" {
			continue
		}
		chain = append(chain, fallbackProvider{provider: c.provider, model: c.model})
		if len(chain) >= 2 { // max 2 fallbacks
			break
		}
	}
	return chain
}

// isRetryableError returns true for errors that warrant trying another provider.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// 5xx server errors
	for _, code := range []string{"500:", "502:", "503:", "504:", "520:", "521:", "522:", "529:"} {
		if strings.Contains(msg, code) {
			return true
		}
	}
	// Rate limits
	if strings.Contains(msg, "429:") || strings.Contains(msg, "rate limit") || strings.Contains(msg, "Rate limit") {
		return true
	}
	// Timeouts and connection errors
	if strings.Contains(msg, "context deadline exceeded") || strings.Contains(msg, "context canceled") {
		return true
	}
	if strings.Contains(msg, "connection refused") || strings.Contains(msg, "no such host") {
		return true
	}
	if strings.Contains(msg, "EOF") || strings.Contains(msg, "broken pipe") {
		return true
	}
	return false
}

// summarizeError returns a short version of the error for logging.
func summarizeError(err error) string {
	s := err.Error()
	if len(s) > 80 {
		return s[:80] + "..."
	}
	return s
}

func (p *Provider) anthropicComplete(ctx context.Context, req Request) (*Response, error) {
	p.keysMu.RLock()
	var key, apiURL, providerName string
	switch req.Provider {
	case "minimax":
		key = p.keys.MiniMax
		apiURL = p.cfg.AI.Providers["minimax"].APIURL
		providerName = "minimax"
		if apiURL == "" {
			apiURL = "https://api.minimax.io/anthropic/v1/messages"
		}
	case "kimi":
		key = p.keys.Kimi
		apiURL = p.cfg.AI.Providers["kimi"].APIURL
		providerName = "kimi"
		if apiURL == "" {
			apiURL = "https://api.kimi.com/coding/v1/messages"
		}
	default:
		key = p.keys.Anthropic
		apiURL = p.cfg.AI.Providers["anthropic"].APIURL
		providerName = "anthropic"
		if apiURL == "" {
			apiURL = "https://api.anthropic.com/v1/messages"
		}
	}
	p.keysMu.RUnlock()
	if key == "" {
		return nil, fmt.Errorf("%s API key not configured", providerName)
	}

	var content any
	if req.ImageBase64 != "" {
		mediaType := req.ImageType
		if mediaType == "" {
			mediaType = "image/jpeg"
		}
		content = []map[string]any{
			{"type": "image", "source": map[string]any{"type": "base64", "media_type": mediaType, "data": req.ImageBase64}},
			{"type": "text", "text": req.Prompt},
		}
	} else {
		content = req.Prompt
	}

	body := map[string]any{
		"model":       req.Model,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
		"messages":    []map[string]any{{"role": "user", "content": content}},
	}
	b, _ := json.Marshal(body)

	httpReq, _ := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewReader(b))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", key)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s %d: %s", providerName, resp.StatusCode, string(respBody[:min(len(respBody), 200)]))
	}

	var data struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	json.Unmarshal(respBody, &data)

	text := ""
	if len(data.Content) > 0 {
		text = data.Content[0].Text
	}
	cost := calcCost(req.Model, data.Usage.InputTokens, data.Usage.OutputTokens)

	return &Response{
		Content:      text,
		Model:        req.Model,
		InputTokens:  data.Usage.InputTokens,
		OutputTokens: data.Usage.OutputTokens,
		CostUSD:      cost,
	}, nil
}

func (p *Provider) openaiComplete(ctx context.Context, req Request) (*Response, error) {
	p.keysMu.RLock()
	var key, baseURL string
	switch req.Provider {
	case "openrouter":
		key = p.keys.OpenRouter
		baseURL = p.cfg.AI.Providers["openrouter"].APIURL
	case "fireworks":
		key = p.keys.Fireworks
		baseURL = p.cfg.AI.Providers["fireworks"].APIURL
	case "kimi":
		key = p.keys.Kimi
		baseURL = p.cfg.AI.Providers["kimi"].APIURL
		if baseURL == "" {
			baseURL = "https://api.moonshot.cn/v1"
		}
	case "minimax":
		key = p.keys.MiniMax
		baseURL = p.cfg.AI.Providers["minimax"].APIURL
		if baseURL == "" {
			baseURL = "https://api.minimax.io/v1"
		}
	case "qwen":
		key = p.keys.Qwen
		baseURL = p.cfg.AI.Providers["qwen"].APIURL
		if baseURL == "" {
			baseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		}
	default:
		key = p.keys.OpenAI
		baseURL = p.cfg.AI.Providers["openai"].APIURL
	}
	p.keysMu.RUnlock()

	if key == "" {
		return nil, fmt.Errorf("%s API key not configured", req.Provider)
	}
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}

	if req.Provider == "minimax" && req.ImageBase64 != "" {
		if resp, err := p.minimaxCodingPlanVisionComplete(ctx, key, baseURL, req); err == nil {
			return resp, nil
		} else {
			log.Printf("[ai] minimax coding-plan vision path failed, falling back to text API: %v", summarizeError(err))
		}
	}

	var msgContent any
	if req.ImageBase64 != "" {
		mediaType := req.ImageType
		if mediaType == "" {
			mediaType = "image/jpeg"
		}
		msgContent = []map[string]any{
			{"type": "text", "text": req.Prompt},
			{"type": "image_url", "image_url": map[string]string{"url": fmt.Sprintf("data:%s;base64,%s", mediaType, req.ImageBase64)}},
		}
	} else {
		msgContent = req.Prompt
	}

	messages := []map[string]any{{"role": "user", "content": msgContent}}
	if req.Provider == "openrouter" {
		messages = append([]map[string]any{{"role": "system", "content": "You must respond with ONLY valid JSON. No markdown, no explanations."}}, messages...)
	}

	body := map[string]any{
		"model":       req.Model,
		"max_tokens":  req.MaxTokens,
		"temperature": req.Temperature,
		"messages":    messages,
	}
	b, _ := json.Marshal(body)

	endpoint := baseURL + "/chat/completions"
	if req.Provider == "minimax" {
		endpoint = baseURL + "/text/chatcompletion_v2"
	}

	httpReq, _ := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(b))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+key)
	if req.Provider == "openrouter" {
		httpReq.Header.Set("HTTP-Referer", "https://wpuiai.com")
		httpReq.Header.Set("X-Title", "WPUIAI AI Cloud")
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%s %d: %s", req.Provider, resp.StatusCode, string(respBody[:min(len(respBody), 200)]))
	}

	var data struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
		BaseResp struct {
			StatusCode int    `json:"status_code"`
			StatusMsg  string `json:"status_msg"`
		} `json:"base_resp"`
	}
	json.Unmarshal(respBody, &data)
	if data.BaseResp.StatusCode != 0 {
		msg := strings.TrimSpace(data.BaseResp.StatusMsg)
		if msg == "" {
			msg = fmt.Sprintf("MiniMax API error %d", data.BaseResp.StatusCode)
		}
		return nil, fmt.Errorf("%s %d: %s", req.Provider, data.BaseResp.StatusCode, msg)
	}

	text := ""
	if len(data.Choices) > 0 {
		text = data.Choices[0].Message.Content
	}
	cost := calcCost(req.Model, data.Usage.PromptTokens, data.Usage.CompletionTokens)

	return &Response{
		Content:      text,
		Model:        req.Model,
		InputTokens:  data.Usage.PromptTokens,
		OutputTokens: data.Usage.CompletionTokens,
		CostUSD:      cost,
	}, nil
}

func (p *Provider) minimaxCodingPlanVisionComplete(ctx context.Context, key, baseURL string, req Request) (*Response, error) {
	apiHost := strings.TrimSuffix(baseURL, "/v1")
	if apiHost == "" || apiHost == baseURL {
		apiHost = "https://api.minimax.io"
	}
	mediaType := req.ImageType
	if mediaType == "" {
		mediaType = "image/jpeg"
	}
	payload := map[string]any{
		"prompt":    req.Prompt,
		"image_url": fmt.Sprintf("data:%s;base64,%s", mediaType, req.ImageBase64),
	}
	b, _ := json.Marshal(payload)

	httpReq, _ := http.NewRequestWithContext(ctx, "POST", apiHost+"/v1/coding_plan/vlm", bytes.NewReader(b))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+key)
	httpReq.Header.Set("MM-API-Source", "Minimax-MCP")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("minimax coding_plan_vlm %d: %s", resp.StatusCode, string(respBody[:min(len(respBody), 200)]))
	}

	var data struct {
		Content  string `json:"content"`
		BaseResp struct {
			StatusCode int    `json:"status_code"`
			StatusMsg  string `json:"status_msg"`
		} `json:"base_resp"`
	}
	json.Unmarshal(respBody, &data)
	if data.BaseResp.StatusCode != 0 {
		msg := strings.TrimSpace(data.BaseResp.StatusMsg)
		if msg == "" {
			msg = fmt.Sprintf("MiniMax API error %d", data.BaseResp.StatusCode)
		}
		return nil, fmt.Errorf("minimax %d: %s", data.BaseResp.StatusCode, msg)
	}
	if strings.TrimSpace(data.Content) == "" {
		return nil, fmt.Errorf("minimax coding_plan_vlm returned empty content")
	}

	return &Response{
		Content: strings.TrimSpace(data.Content),
		Model:   req.Model,
	}, nil
}

func (p *Provider) ensureKeys() error {
	p.keysMu.RLock()
	if time.Since(p.keys.fetchedAt) < p.keysTTL && p.keys.fetchedAt.Unix() > 0 {
		p.keysMu.RUnlock()
		return nil
	}
	p.keysMu.RUnlock()

	url := p.cfg.RESTURL("ai-settings")
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Webhook-Secret", p.cfg.WordPress.WebhookSecret)

	resp, err := p.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	var settings struct {
		DefaultProvider string                      `json:"default_provider"`
		DefaultModel    string                      `json:"default_model"`
		Anthropic       struct{ Key, Model string } `json:"anthropic"`
		OpenAI          struct{ Key, Model string } `json:"openai"`
		OpenRouter      struct{ Key, Model string } `json:"openrouter"`
		Fireworks       struct{ Key, Model string } `json:"fireworks"`
		Kimi            struct{ Key, Model string } `json:"kimi"`
		MiniMax         struct{ Key, Model string } `json:"minimax"`
		Qwen            struct{ Key, Model string } `json:"qwen"`
	}
	if resp.StatusCode != 200 {
		log.Printf("[ai] WARNING: WP /ai-settings returned HTTP %d — falling back to env vars", resp.StatusCode)
		p.loadKeysFromEnv()
		return nil
	}

	if err := json.Unmarshal(body, &settings); err != nil {
		log.Printf("[ai] WARNING: WP /ai-settings returned invalid JSON: %v — falling back to env vars", err)
		p.loadKeysFromEnv()
		return nil
	}

	// Update runtime config with WP-managed defaults, then allow explicit env overrides.
	if settings.DefaultProvider != "" {
		p.cfg.AI.DefaultProvider = settings.DefaultProvider
		log.Printf("[ai] default provider from WP: %s", settings.DefaultProvider)
	}
	if settings.DefaultModel != "" {
		p.cfg.AI.DefaultModel = settings.DefaultModel
		log.Printf("[ai] default model from WP: %s", settings.DefaultModel)
	}
	if v := envOr("UIAI_DEFAULT_PROVIDER", ""); v != "" {
		p.cfg.AI.DefaultProvider = v
		log.Printf("[ai] default provider from env override: %s", v)
	}
	if v := envOr("UIAI_DEFAULT_MODEL", ""); v != "" {
		p.cfg.AI.DefaultModel = v
		log.Printf("[ai] default model from env override: %s", v)
	}

	merge := func(primary, fallback string) string {
		if primary != "" {
			return primary
		}
		return fallback
	}

	p.keysMu.Lock()
	p.keys = aiKeys{
		Anthropic:  merge(settings.Anthropic.Key, envOr("ANTHROPIC_API_KEY", "")),
		OpenAI:     merge(settings.OpenAI.Key, envOr("OPENAI_API_KEY", "")),
		OpenRouter: merge(settings.OpenRouter.Key, envOr("OPENROUTER_API_KEY", "")),
		Fireworks:  merge(settings.Fireworks.Key, envOr("FIREWORKS_API_KEY", "")),
		Kimi:       merge(settings.Kimi.Key, envOr("KIMI_API_KEY", "")),
		MiniMax:    merge(settings.MiniMax.Key, envOr("MINIMAX_API_KEY", "")),
		Qwen:       merge(settings.Qwen.Key, envOr("QWEN_API_KEY", "")),
		fetchedAt:  time.Now(),
	}
	p.keysMu.Unlock()

	// Build available models list from WP settings.
	// Only include providers that have an API key configured.
	var models []ProviderModel
	seen := map[string]bool{}

	// Default model first.
	dm := p.cfg.AI.DefaultModel
	dp := p.cfg.AI.DefaultProvider
	if dm != "" {
		models = append(models, ProviderModel{ID: dm, Name: dm, Provider: dp, Default: true})
		seen[dm] = true
	}

	// Each configured provider: add its model if key is set.
	type pm struct {
		provider, key, model string
	}
	configured := []pm{
		{"anthropic", settings.Anthropic.Key, settings.Anthropic.Model},
		{"openai", settings.OpenAI.Key, settings.OpenAI.Model},
		{"openrouter", settings.OpenRouter.Key, settings.OpenRouter.Model},
		{"fireworks", settings.Fireworks.Key, settings.Fireworks.Model},
		{"kimi", settings.Kimi.Key, settings.Kimi.Model},
		{"minimax", settings.MiniMax.Key, settings.MiniMax.Model},
		{"qwen", settings.Qwen.Key, settings.Qwen.Model},
	}
	for _, c := range configured {
		if c.key != "" && c.model != "" && !seen[c.model] {
			models = append(models, ProviderModel{ID: c.model, Name: c.model, Provider: c.provider})
			seen[c.model] = true
		}
	}

	p.modelsMu.Lock()
	p.models = models
	p.modelsMu.Unlock()
	log.Printf("[ai] loaded %d available models from WP settings", len(models))

	return nil
}

func (p *Provider) loadKeysFromEnv() {
	keys := aiKeys{
		Anthropic:  envOr("ANTHROPIC_API_KEY", ""),
		OpenAI:     envOr("OPENAI_API_KEY", ""),
		OpenRouter: envOr("OPENROUTER_API_KEY", ""),
		Fireworks:  envOr("FIREWORKS_API_KEY", ""),
		Kimi:       envOr("KIMI_API_KEY", ""),
		MiniMax:    envOr("MINIMAX_API_KEY", ""),
		Qwen:       envOr("QWEN_API_KEY", ""),
		fetchedAt:  time.Now(),
	}
	if v := envOr("UIAI_DEFAULT_PROVIDER", ""); v != "" {
		p.cfg.AI.DefaultProvider = v
	}
	if v := envOr("UIAI_DEFAULT_MODEL", ""); v != "" {
		p.cfg.AI.DefaultModel = v
	}

	p.keysMu.Lock()
	p.keys = keys
	p.keysMu.Unlock()

	models := []ProviderModel{}
	if p.cfg.AI.DefaultProvider != "" && p.cfg.AI.DefaultModel != "" {
		models = append(models, ProviderModel{ID: p.cfg.AI.DefaultModel, Name: p.cfg.AI.DefaultModel, Provider: p.cfg.AI.DefaultProvider, Default: true})
	}
	p.modelsMu.Lock()
	p.models = models
	p.modelsMu.Unlock()
	log.Printf("[ai] loaded %d available models from env fallback", len(models))
}

func calcCost(model string, inTok, outTok int) float64 {
	costs, ok := modelCosts[model]
	if !ok {
		costs = [2]float64{0.000003, 0.000015} // unknown model — estimate at $3/$15 per M tokens
	}
	return float64(inTok)*costs[0] + float64(outTok)*costs[1]
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
