package captcha

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"strings"

	"github.com/WPUIAI/uiai-engine/internal/ai"
)

// ─── Prompt templates ──────────────────────────────────────────────────────

var promptTemplates = map[string]string{
	"blind_assistant": `Act as a blind person assistant. Read the text from the image and give me only the text answer.
The text may be distorted with noise, grid lines, or warping — read through the distortion and return only the characters you can identify.
If there is no text, give me an empty string.
Return ONLY the characters, nothing else.`,

	"verification_code": `This image contains a verification code. Read ONLY the characters shown.
Return only uppercase letters and digits, no spaces or punctuation.
If unsure about a character, make your best guess.
Return ONLY the code, nothing else.`,

	"lowercase_captcha": `This image is a CAPTCHA containing distorted text.
The text is exactly 5 lowercase English letters. No digits, no uppercase.
Ignore ALL background noise, grid lines, crosshatch patterns, and visual distortion completely.
Focus only on the actual letter shapes.
Return ONLY the 5 lowercase letters, nothing else.`,
}

// ─── Text captcha solver ───────────────────────────────────────────────────

// SolveTextCaptcha attempts to read text from a captcha image.
// It tries the VLM first, then falls back through the configured chain.
func SolveTextCaptcha(ctx context.Context, aiProv *ai.Provider, imgBase64, imgType string, cfg SolveConfig, captchaCfg CaptchaConfig) (*ImageSolveResponse, error) {
	start := ctx.Value(ctxKeyStart{})
	_ = start

	provider := captchaCfg.DefaultProvider
	model := captchaCfg.DefaultModel

	prompt := buildTextPrompt(cfg)

	fallbackChain := captchaCfg.Text.FallbackChain
	if len(fallbackChain) == 0 {
		fallbackChain = []string{"vlm", "tesseract"}
	}

	var answers []string
	var lastMethod string
	var lastErr error

	for _, backend := range fallbackChain {
		switch backend {
		case "vlm":
			text, method, err := solveWithVLM(ctx, aiProv, imgBase64, imgType, prompt, provider, model)
			if err != nil {
				log.Printf("[captcha] VLM failed: %v", err)
				lastErr = err
				continue
			}
			if text != "" {
				answers = append(answers, text)
				lastMethod = method
				return &ImageSolveResponse{
					Text:         text,
					Confidence:   "high",
					Method:       method,
					Alternatives: answers,
				}, nil
			}

		case "tesseract":
			text, err := solveWithTesseract(ctx, imgBase64, imgType, cfg.Preprocessing)
			if err != nil {
				log.Printf("[captcha] Tesseract failed: %v", err)
				lastErr = err
				continue
			}
			if text != "" {
				answers = append(answers, text)
				lastMethod = "tesseract"
				return &ImageSolveResponse{
					Text:         text,
					Confidence:   "medium",
					Method:       "tesseract",
					Alternatives: answers,
				}, nil
			}

		case "ddddocr":
			text, err := solveWithDdddocr(ctx, imgBase64)
			if err != nil {
				log.Printf("[captcha] ddddocr failed: %v", err)
				lastErr = err
				continue
			}
			if text != "" {
				answers = append(answers, text)
				lastMethod = "ddddocr"
				return &ImageSolveResponse{
					Text:         text,
					Confidence:   "low",
					Method:       "ddddocr",
					Alternatives: answers,
				}, nil
			}
		}
	}

	errMsg := "all backends failed"
	if lastErr != nil {
		errMsg = lastErr.Error()
	}
	return &ImageSolveResponse{
		Text:         "",
		Confidence:   "none",
		Method:       lastMethod,
		Alternatives: answers,
	}, fmt.Errorf("%s", errMsg)
}

// ─── VLM backend ───────────────────────────────────────────────────────────

func solveWithVLM(ctx context.Context, aiProv *ai.Provider, imgBase64, imgType, prompt, provider, model string) (string, string, error) {
	resp, err := aiProv.Complete(ctx, ai.Request{
		Provider:    provider,
		Model:       model,
		Prompt:      prompt,
		MaxTokens:   256,
		Temperature: 1.0,
		ImageBase64: imgBase64,
		ImageType:   imgType,
	})
	if err != nil {
		return "", "", err
	}
	text := cleanVLMResponse(resp.Content)
	method := fmt.Sprintf("vlm:%s/%s", provider, model)
	return text, method, nil
}

// cleanVLMResponse strips common VLM noise from captcha answers.
func cleanVLMResponse(raw string) string {
	s := strings.TrimSpace(raw)

	// Remove markdown code blocks
	s = strings.TrimPrefix(s, "```json\n")
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```\n")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "\n```")
	s = strings.TrimSuffix(s, "```")
	s = strings.Trim(s, "`")
	s = strings.TrimSpace(s)

	// Handle multi-line: take last non-empty line (chatty models put explanation first)
	if strings.Contains(s, "\n") {
		lines := strings.Split(s, "\n")
		for i := len(lines) - 1; i >= 0; i-- {
			line := strings.TrimSpace(lines[i])
			if line != "" && !strings.HasPrefix(line, "{") && !strings.HasPrefix(line, "//") {
				s = line
				break
			}
		}
		s = strings.TrimSpace(s)
	}

	// Try to extract from JSON wrapper like {"answer": "m0ghn"}
	if strings.HasPrefix(s, "{") || strings.Contains(raw, `"answer"`) || strings.Contains(raw, `"text"`) {
		src := s
		if !strings.HasPrefix(src, "{") {
			// Find JSON in original raw
			if idx := strings.Index(raw, "{"); idx >= 0 {
				src = raw[idx:]
			}
		}
		for _, key := range []string{"answer", "text", "code", "captcha", "result", "response", "characters", "letters", "value", "word"} {
			pattern := `"` + key + `"`
			if idx := strings.Index(strings.ToLower(src), strings.ToLower(pattern)); idx >= 0 {
				rest := src[idx+len(pattern):]
				// Skip : and whitespace
				rest = strings.TrimLeft(rest, ": \t\n")
				rest = strings.TrimLeft(rest, `"`)
				if endIdx := strings.IndexByte(rest, '"'); endIdx >= 0 {
					extracted := rest[:endIdx]
					if len(extracted) >= 2 {
						s = extracted
						break
					}
				}
			}
		}
	}

	// Remove common prefixes from chatty models
	for _, prefix := range []string{
		"The text is: ", "The text reads: ", "The captcha text is: ",
		"Answer: ", "Text: ", "The answer is: ", "Code: ",
		"the text is ", "the answer is ", "The characters are: ",
	} {
		lower := strings.ToLower(s)
		if strings.HasPrefix(lower, strings.ToLower(prefix)) {
			s = s[len(prefix):]
		}
	}

	// Remove quotes
	s = strings.Trim(s, `"'`)
	// Remove trailing periods/punctuation
	s = strings.TrimRight(s, ".!?,;: ")
	s = strings.TrimSpace(s)
	return s
}

// ─── Tesseract backend ─────────────────────────────────────────────────────

func solveWithTesseract(ctx context.Context, imgBase64, imgType string, ppCfg *PreprocessConfig) (string, error) {
	if _, err := exec.LookPath("tesseract"); err != nil {
		return "", fmt.Errorf("tesseract not installed")
	}

	// Preprocess first
	processed := imgBase64
	if ppCfg != nil {
		var err error
		processed, err = Preprocess(imgBase64, imgType, ppCfg)
		if err != nil {
			log.Printf("[captcha] preprocess for tesseract failed: %v", err)
			processed = imgBase64
		}
	}

	// Decode to temp file
	raw, err := decodeBase64(processed)
	if err != nil {
		return "", err
	}

	tmpFile, err := writeTempFile(raw, "captcha-tess-*.png")
	if err != nil {
		return "", err
	}
	defer removeTempFile(tmpFile)

	// Try multiple PSM modes
	var bestText string
	var bestScore int
	for _, psm := range []string{"7", "8", "13"} {
		cmd := exec.CommandContext(ctx, "tesseract", tmpFile, "stdout", // #nosec G204 -- tesseract args are fixed options and generated temp-file paths.
			"--psm", psm, "-l", "eng",
			"-c", "tessedit_char_whitelist=abcdefghijklmnopqrstuvwxyz0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ")
		out, err := cmd.Output()
		if err != nil {
			continue
		}
		text := strings.TrimSpace(string(out))
		text = strings.ReplaceAll(text, " ", "")
		text = strings.ReplaceAll(text, "\n", "")
		score := scoreCandidate(text)
		if score > bestScore {
			bestScore = score
			bestText = text
		}
	}
	return bestText, nil
}

// ─── ddddocr backend ──────────────────────────────────────────────────────

func solveWithDdddocr(ctx context.Context, imgBase64 string) (string, error) {
	// ddddocr is Python — shell out to a small script
	script := `
import sys, base64, ddddocr
ocr = ddddocr.DdddOcr(show_ad=False)
img = base64.b64decode(sys.stdin.read())
print(ocr.classification(img))
`
	cmd := exec.CommandContext(ctx, "python3", "-c", script)
	cmd.Stdin = strings.NewReader(imgBase64)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("ddddocr: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// ─── Helpers ───────────────────────────────────────────────────────────────

func buildTextPrompt(cfg SolveConfig) string {
	tmpl := cfg.PromptTemplate
	if tmpl == "" {
		tmpl = "blind_assistant"
	}
	prompt, ok := promptTemplates[tmpl]
	if !ok {
		prompt = promptTemplates["blind_assistant"]
	}
	if cfg.Hint != "" {
		prompt += "\nHint: " + cfg.Hint
	}
	return prompt
}

func scoreCandidate(s string) int {
	if len(s) == 0 {
		return 0
	}
	score := 0
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			score += 2
		}
	}
	// Penalize too short or too long
	if len(s) >= 4 && len(s) <= 7 {
		score += 5
	}
	return score
}

// ctxKeyStart is a context key for tracking timing.
type ctxKeyStart struct{}
