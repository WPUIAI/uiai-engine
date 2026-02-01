package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/philoveracity/uiai-engine/internal/config"
)

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
	"claude-sonnet-4-20250514":                             {0.000003, 0.000015},
	"claude-opus-4-20250514":                               {0.000015, 0.000075},
	"gpt-4o":                                               {0.0000025, 0.00001},
	"gpt-4o-mini":                                          {0.00000015, 0.0000006},
	"accounts/fireworks/models/llama-v3p1-405b-instruct":   {0.000003, 0.000003},
	"accounts/fireworks/models/llama-v3p1-70b-instruct":    {0.0000009, 0.0000009},
	"accounts/fireworks/models/llama-v3p1-8b-instruct":     {0.0000002, 0.0000002},
	"accounts/fireworks/models/qwen2p5-72b-instruct":       {0.0000009, 0.0000009},
}

type aiKeys struct {
	Anthropic  string
	OpenAI     string
	OpenRouter string
	Fireworks  string
	fetchedAt  time.Time
}

type Provider struct {
	cfg       *config.Config
	keys      aiKeys
	keysMu    sync.RWMutex
	keysTTL   time.Duration
	client    *http.Client
}

func NewProvider(cfg *config.Config) *Provider {
	return &Provider{
		cfg:     cfg,
		keysTTL: time.Duration(cfg.WordPress.CacheTTL) * time.Second,
		client:  &http.Client{Timeout: 120 * time.Second},
	}
}

func (p *Provider) Complete(req Request) (*Response, error) {
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

	switch req.Provider {
	case "anthropic":
		return p.anthropicComplete(req)
	case "openai", "openrouter", "fireworks":
		return p.openaiComplete(req)
	default:
		return nil, fmt.Errorf("unknown provider: %s", req.Provider)
	}
}

func (p *Provider) anthropicComplete(req Request) (*Response, error) {
	p.keysMu.RLock()
	key := p.keys.Anthropic
	p.keysMu.RUnlock()
	if key == "" {
		return nil, fmt.Errorf("anthropic API key not configured")
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

	apiURL := p.cfg.AI.Providers["anthropic"].APIURL
	if apiURL == "" {
		apiURL = "https://api.anthropic.com/v1/messages"
	}
	httpReq, _ := http.NewRequest("POST", apiURL, bytes.NewReader(b))
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
		return nil, fmt.Errorf("anthropic %d: %s", resp.StatusCode, string(respBody[:min(len(respBody), 200)]))
	}

	var data struct {
		Content []struct{ Text string `json:"text"` } `json:"content"`
		Usage   struct {
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

func (p *Provider) openaiComplete(req Request) (*Response, error) {
	p.keysMu.RLock()
	var key, baseURL string
	switch req.Provider {
	case "openrouter":
		key = p.keys.OpenRouter
		baseURL = p.cfg.AI.Providers["openrouter"].APIURL
	case "fireworks":
		key = p.keys.Fireworks
		baseURL = p.cfg.AI.Providers["fireworks"].APIURL
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

	httpReq, _ := http.NewRequest("POST", baseURL+"/chat/completions", bytes.NewReader(b))
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
			Message struct{ Content string `json:"content"` } `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
	}
	json.Unmarshal(respBody, &data)

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
		Anthropic  struct{ Key string `json:"key"` } `json:"anthropic"`
		OpenAI     struct{ Key string `json:"key"` } `json:"openai"`
		OpenRouter struct{ Key string `json:"key"` } `json:"openrouter"`
		Fireworks  struct{ Key string `json:"key"` } `json:"fireworks"`
	}
	if err := json.Unmarshal(body, &settings); err != nil {
		// Fallback to env vars
		p.keysMu.Lock()
		p.keys = aiKeys{
			Anthropic:  envOr("ANTHROPIC_API_KEY", ""),
			OpenRouter: envOr("OPENROUTER_API_KEY", ""),
			fetchedAt:  time.Now(),
		}
		p.keysMu.Unlock()
		return nil
	}

	p.keysMu.Lock()
	p.keys = aiKeys{
		Anthropic:  settings.Anthropic.Key,
		OpenAI:     settings.OpenAI.Key,
		OpenRouter: settings.OpenRouter.Key,
		Fireworks:  settings.Fireworks.Key,
		fetchedAt:  time.Now(),
	}
	p.keysMu.Unlock()
	return nil
}

func calcCost(model string, inTok, outTok int) float64 {
	costs, ok := modelCosts[model]
	if !ok {
		costs = [2]float64{0.000003, 0.000015} // default to sonnet pricing
	}
	return float64(inTok)*costs[0] + float64(outTok)*costs[1]
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok {
		return v
	}
	return fallback
}
