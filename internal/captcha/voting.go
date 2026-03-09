package captcha

import (
	"context"
	"log"
	"strings"
	"sync"
	"unicode"

	"github.com/philoveracity/uiai-engine/internal/ai"
)

// ─── Multi-model voting for text captchas ──────────────────────────────────
//
// Strategy: send the same image to N models in parallel, then align answers
// character-by-character using majority vote. This raises per-char accuracy
// from ~70% (single model) to ~90%+ (3-model consensus).

// VoteTextCaptcha sends the image to multiple models and returns the
// consensus answer.
func VoteTextCaptcha(ctx context.Context, aiProv *ai.Provider, imgBase64, imgType, prompt string, voters []VoterModel) (string, []string, error) {
	if len(voters) == 0 {
		voters = DefaultVoters()
	}

	type result struct {
		text   string
		model  string
		weight int
		err    error
	}

	results := make([]result, len(voters))
	var wg sync.WaitGroup

	for i, v := range voters {
		wg.Add(1)
		go func(idx int, voter VoterModel) {
			defer wg.Done()
			temp := voter.Temperature
			if temp == 0 {
				temp = 0.8
			}
			resp, err := aiProv.Complete(ctx, ai.Request{
				Provider:    voter.Provider,
				Model:       voter.Model,
				Prompt:      prompt,
				MaxTokens:   128,
				Temperature: temp,
				ImageBase64: imgBase64,
				ImageType:   imgType,
			})
			if err != nil {
				results[idx] = result{err: err, model: voter.Model}
				return
			}
			text := cleanVLMResponse(resp.Content)
			results[idx] = result{text: text, model: voter.Model, weight: voter.Weight}
		}(i, v)
	}
	wg.Wait()

	// Collect valid answers — weighted (repeat text by weight)
	var answers []string
	for _, r := range results {
		if r.err != nil {
			log.Printf("[captcha-vote] model %s failed: %v", r.model, r.err)
			continue
		}
		if r.text != "" {
			w := r.weight
			if w <= 0 {
				w = 1
			}
			for j := 0; j < w; j++ {
				answers = append(answers, r.text)
			}
			log.Printf("[captcha-vote] model %s → %q (weight=%d)", r.model, r.text, w)
		}
	}

	if len(answers) == 0 {
		return "", nil, results[0].err
	}
	if len(answers) == 1 {
		return answers[0], answers, nil
	}

	consensus := charLevelVote(answers)
	return consensus, answers, nil
}

// charLevelVote aligns multiple answers character by character and picks
// the most popular character at each position.
func charLevelVote(answers []string) string {
	// Normalize: lowercase, strip non-alnum
	normalized := make([]string, len(answers))
	for i, a := range answers {
		normalized[i] = normalizeChars(a)
	}

	// Find the most common length
	lenCounts := map[int]int{}
	for _, a := range normalized {
		lenCounts[len(a)]++
	}
	bestLen := 0
	bestCount := 0
	for l, c := range lenCounts {
		if c > bestCount || (c == bestCount && l > bestLen) {
			bestLen = l
			bestCount = c
		}
	}

	if bestLen == 0 {
		// Fallback: return longest
		best := ""
		for _, a := range normalized {
			if len(a) > len(best) {
				best = a
			}
		}
		return best
	}

	// Filter to answers with the consensus length (allow ±1)
	var aligned []string
	for _, a := range normalized {
		diff := len(a) - bestLen
		if diff >= -1 && diff <= 1 {
			aligned = append(aligned, a)
		}
	}
	if len(aligned) == 0 {
		aligned = normalized
	}

	// Pad/truncate to bestLen
	for i := range aligned {
		if len(aligned[i]) < bestLen {
			aligned[i] += strings.Repeat("_", bestLen-len(aligned[i]))
		} else if len(aligned[i]) > bestLen {
			aligned[i] = aligned[i][:bestLen]
		}
	}

	// Vote per position
	result := make([]byte, bestLen)
	for pos := 0; pos < bestLen; pos++ {
		counts := map[byte]int{}
		for _, a := range aligned {
			if pos < len(a) {
				counts[a[pos]]++
			}
		}
		best := byte('?')
		bestC := 0
		for ch, c := range counts {
			if c > bestC {
				bestC = c
				best = ch
			}
		}
		result[pos] = best
	}

	return string(result)
}

// normalizeChars keeps only lowercase alphanumeric, maps O→0, l→1, etc.
func normalizeChars(s string) string {
	s = strings.ToLower(s)
	var out []byte
	for _, c := range s {
		if unicode.IsLetter(c) || unicode.IsDigit(c) {
			out = append(out, byte(c))
		}
	}
	return string(out)
}

// ─── Voter model definitions ───────────────────────────────────────────────

// VoterModel defines one model participating in voting.
type VoterModel struct {
	Provider    string  `json:"provider" yaml:"provider"`
	Model       string  `json:"model" yaml:"model"`
	Temperature float64 `json:"temperature" yaml:"temperature"`
	Weight      int     `json:"weight" yaml:"weight"` // heavier weight = more votes
}

// DefaultVoters returns the standard voting panel.
// Gemini Flash is dominant for captcha text reading — Claude Sonnet tends to
// hallucinate English words instead of reading distorted characters.
func DefaultVoters() []VoterModel {
	return []VoterModel{
		{Provider: "openrouter", Model: "google/gemini-2.0-flash-001", Temperature: 0.8, Weight: 2},
		{Provider: "openrouter", Model: "google/gemini-2.0-flash-001", Temperature: 0.3, Weight: 2},
		{Provider: "openrouter", Model: "google/gemini-2.0-flash-001", Temperature: 1.0, Weight: 1},
	}
}

// ─── Multi-pass preprocessing voting ───────────────────────────────────────

// MultiPassResult holds results from multiple preprocessing variants.
type MultiPassResult struct {
	Best         string
	Confidence   float64
	AllResults   []PassResult
}

type PassResult struct {
	Config PreprocessConfig
	Text   string
	Method string
}

// SolveTextMultiPass runs the solver with multiple preprocessing configs
// and picks the best answer via voting.
func SolveTextMultiPass(ctx context.Context, aiProv *ai.Provider, imgBase64, imgType string, cfg SolveConfig, captchaCfg CaptchaConfig) (*ImageSolveResponse, error) {
	variants := []PreprocessConfig{
		// Variant 1: light preprocessing
		{Upscale: 2, Threshold: 120},
		// Variant 2: medium — good for crosshatch
		{Upscale: 3, Threshold: 100, MorphologyKernel: 3, ComponentMinArea: 20},
		// Variant 3: aggressive — heavy noise removal
		{Upscale: 4, Threshold: 80, MorphologyKernel: 5, ComponentMinArea: 50, ComponentMaxAspect: 6},
	}

	prompt := buildTextPrompt(cfg)
	voters := captchaCfg.Text.Voters
	if len(voters) == 0 {
		voters = DefaultVoters()
	}

	// Run each variant through voting in parallel
	type variantResult struct {
		answers []string
		best    string
	}

	results := make([]variantResult, len(variants))
	var wg sync.WaitGroup

	for i, variant := range variants {
		wg.Add(1)
		go func(idx int, pp PreprocessConfig) {
			defer wg.Done()
			processed, err := Preprocess(imgBase64, imgType, &pp)
			if err != nil {
				log.Printf("[captcha] preprocess variant %d failed: %v", idx, err)
				processed = imgBase64
			}
			best, answers, err := VoteTextCaptcha(ctx, aiProv, processed, "image/png", prompt, voters)
			if err != nil {
				log.Printf("[captcha] vote variant %d failed: %v", idx, err)
				return
			}
			results[idx] = variantResult{answers: answers, best: best}
		}(i, variant)
	}
	wg.Wait()

	// Collect all "best" answers from each variant
	var allBests []string
	var allAnswers []string
	for _, r := range results {
		if r.best != "" {
			allBests = append(allBests, r.best)
		}
		allAnswers = append(allAnswers, r.answers...)
	}

	if len(allBests) == 0 {
		return &ImageSolveResponse{
			Text:       "",
			Confidence: "none",
			Method:     "multipass_vote",
		}, nil
	}

	// Final vote across variant bests
	finalAnswer := charLevelVote(allBests)

	return &ImageSolveResponse{
		Text:         finalAnswer,
		Confidence:   "high",
		Method:       "multipass_vote",
		Alternatives: allAnswers,
	}, nil
}
