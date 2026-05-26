package captcha

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/philoveracity/uiai-engine/internal/ai"
	"github.com/philoveracity/uiai-engine/internal/vision"
)

// ─── reCAPTCHA v2 solver ───────────────────────────────────────────────────
//
// Two strategies:
//   A. full_grid (recommended): screenshot entire 3×3 or 4×4 grid,
//      send one image to VLM → JSON list of matching tile numbers.
//      1 API call per round instead of 9-16.
//   B. per_tile: extract each tile via canvas toDataURL, classify individually,
//      multi-model vote on each. More API calls but potentially more precise.
//
// Flow:
//   1. Click reCAPTCHA checkbox
//   2. Wait for challenge to appear
//   3. Extract challenge info (target object, grid size)
//   4. Screenshot grid / extract tiles
//   5. VLM classify → select matching tiles
//   6. Click verify
//   7. Check if solved, repeat if new challenge appears

const (
	rcCheckboxSel   = `iframe[title="reCAPTCHA"]`
	rcChallengeSel  = `iframe[title*="challenge"]`
	rcGridImageSel  = `.rc-imageselect-challenge img, .rc-image-tile-wrapper img`
	rcDescSel       = `.rc-imageselect-desc, .rc-imageselect-desc-wrapper`
	rcVerifyBtnSel  = `#recaptcha-verify-button`
	rcAnchorChecked = `#recaptcha-anchor`
)

// SolveRecaptchaV2 solves a reCAPTCHA v2 challenge in a live session.
func (s *Solver) SolveRecaptchaV2(ctx context.Context, sess *vision.Session, cfg SolveConfig) *SolveResponse {
	// Create a longer timeout context for the full reCAPTCHA solve
	solveCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	ctx = solveCtx

	rcCfg := s.Config.Recaptcha
	maxAttempts := rcCfg.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 2
	}
	maxRounds := rcCfg.MaxRounds
	if maxRounds <= 0 {
		maxRounds = 3
	}

	strategy := cfg.Strategy
	if strategy == "" {
		strategy = rcCfg.Strategy
	}
	if strategy == "" {
		strategy = "full_grid"
	}

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		log.Printf("[recaptcha] attempt %d/%d strategy=%s", attempt, maxAttempts, strategy)

		result := s.attemptRecaptcha(ctx, sess, strategy, maxRounds, rcCfg)
		if result.Solved {
			return result
		}

		// Not solved — maybe try audio fallback
		if rcCfg.AudioFallback && attempt == maxAttempts-1 {
			log.Printf("[recaptcha] trying audio fallback")
			audioResult := s.tryAudioFallback(ctx, sess, rcCfg)
			if audioResult.Solved {
				return audioResult
			}
		}

		// Wait before retry
		randomDelay(500, 1500)
	}

	return &SolveResponse{
		Solved:   false,
		Type:     "recaptcha_v2",
		Attempts: maxAttempts,
		Error:    "max attempts exceeded",
		Method:   strategy,
	}
}

func (s *Solver) attemptRecaptcha(ctx context.Context, sess *vision.Session, strategy string, maxRounds int, rcCfg RecaptchaConfig) *SolveResponse {
	// 1. Click checkbox
	log.Printf("[recaptcha] clicking checkbox")
	clickJS := fmt.Sprintf(`
		var iframe = document.querySelector(%q);
		if (!iframe) return "ERR:no_checkbox_iframe";
		try {
			var cb = iframe.contentDocument.querySelector(".recaptcha-checkbox-border");
			if (!cb) return "ERR:no_checkbox";
			cb.click();
			return "clicked";
		} catch(e) { return "ERR:" + e.message; }
	`, rcCheckboxSel)
	result, _, _ := sess.Eval(clickJS)
	if strings.HasPrefix(result, "ERR:") {
		return &SolveResponse{Solved: false, Error: result, Method: "none"}
	}

	randomDelay(800, 1500)

	// 2. Check if already solved (no challenge appeared)
	if s.isRecaptchaSolved(sess) {
		return &SolveResponse{Solved: true, Type: "recaptcha_v2", Attempts: 1, Rounds: 0, Method: "checkbox_only"}
	}

	// 3. Solve challenge rounds
	totalRounds := 0
	for round := 1; round <= maxRounds; round++ {
		totalRounds = round
		log.Printf("[recaptcha] round %d/%d", round, maxRounds)

		// Extract challenge info
		info, err := s.extractChallengeInfo(sess)
		if err != nil {
			log.Printf("[recaptcha] extract challenge info failed: %v", err)
			return &SolveResponse{Solved: false, Error: err.Error(), Rounds: totalRounds, Method: strategy}
		}

		log.Printf("[recaptcha] target=%q gridSize=%d", info.Target, info.GridSize)

		// Select matching tiles
		var selectedTiles []int
		switch strategy {
		case "full_grid":
			selectedTiles, err = s.solveFullGrid(ctx, sess, info)
		case "per_tile":
			selectedTiles, err = s.solvePerTile(ctx, sess, info)
		default:
			selectedTiles, err = s.solveFullGrid(ctx, sess, info)
		}

		if err != nil {
			log.Printf("[recaptcha] solve round failed: %v", err)
			continue
		}

		if len(selectedTiles) == 0 {
			log.Printf("[recaptcha] no tiles selected, skipping")
			// Click verify anyway — sometimes no tiles is correct
		}

		// Click selected tiles
		for _, tileIdx := range selectedTiles {
			err := s.clickTile(sess, tileIdx, info.GridSize)
			if err != nil {
				log.Printf("[recaptcha] click tile %d failed: %v", tileIdx, err)
			}
			randomDelay(int(rcCfg.ActionDelayMinMs), int(rcCfg.ActionDelayMaxMs))
		}

		// Click verify
		randomDelay(int(rcCfg.VerifyDelayMinMs), int(rcCfg.VerifyDelayMaxMs))
		s.clickVerify(sess)

		// Wait for result
		time.Sleep(2 * time.Second)

		if s.isRecaptchaSolved(sess) {
			return &SolveResponse{
				Solved:   true,
				Type:     "recaptcha_v2",
				Rounds:   totalRounds,
				Attempts: 1,
				Method:   strategy,
			}
		}

		// Check for "please also check" / dynamic tiles
		hasDynamic := s.hasDynamicTiles(sess)
		if hasDynamic {
			log.Printf("[recaptcha] dynamic tiles detected, handling...")
			s.handleDynamicTiles(ctx, sess, info, rcCfg)
		}
	}

	return &SolveResponse{
		Solved:   false,
		Type:     "recaptcha_v2",
		Rounds:   totalRounds,
		Attempts: 1,
		Error:    "max rounds exceeded",
		Method:   strategy,
	}
}

// ─── Challenge info extraction ─────────────────────────────────────────────

type challengeInfo struct {
	Target   string // e.g. "bicycles", "traffic lights"
	GridSize int    // 9 (3x3) or 16 (4x4)
}

func (s *Solver) extractChallengeInfo(sess *vision.Session) (*challengeInfo, error) {
	js := fmt.Sprintf(`
		var cf = document.querySelector(%q);
		if (!cf) return JSON.stringify({"error": "no_challenge_iframe"});
		try {
			var doc = cf.contentDocument;
			var descEl = doc.querySelector(".rc-imageselect-desc strong, .rc-imageselect-desc-no-canonical strong");
			var target = descEl ? descEl.textContent.trim() : "";
			var tiles = doc.querySelectorAll(".rc-imageselect-tile");
			return JSON.stringify({"target": target, "grid_size": tiles.length});
		} catch(e) { return JSON.stringify({"error": e.message}); }
	`, rcChallengeSel)

	result, _, err := sess.Eval(js)
	if err != nil {
		return nil, err
	}

	var data struct {
		Error    string `json:"error"`
		Target   string `json:"target"`
		GridSize int    `json:"grid_size"`
	}
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		return nil, fmt.Errorf("parse challenge info: %w (raw: %s)", err, result)
	}
	if data.Error != "" {
		return nil, fmt.Errorf("challenge: %s", data.Error)
	}

	return &challengeInfo{
		Target:   data.Target,
		GridSize: data.GridSize,
	}, nil
}

// ─── Strategy A: Full-grid screenshot ──────────────────────────────────────

func (s *Solver) solveFullGrid(ctx context.Context, sess *vision.Session, info *challengeInfo) ([]int, error) {
	// reCAPTCHA uses ONE source image for all tiles (CSS clip shows different parts).
	// Extract the full source image — it's already a grid composite.
	js := fmt.Sprintf(`
		var cf = document.querySelector(%q);
		if (!cf) return "";
		try {
			var doc = cf.contentDocument;
			var img = doc.querySelector(".rc-imageselect-tile img");
			if (!img) return "ERR:no_tile_img";
			var c = document.createElement("canvas");
			c.width = img.naturalWidth;
			c.height = img.naturalHeight;
			c.getContext("2d").drawImage(img, 0, 0);
			return c.toDataURL("image/jpeg", 0.9).split(",")[1];
		} catch(e) { return "ERR:" + e.message; }
	`, rcChallengeSel)

	imgBase64, _, err := sess.Eval(js)
	if err != nil {
		return nil, fmt.Errorf("screenshot grid: %w", err)
	}
	if imgBase64 == "" || imgBase64 == "<nil>" {
		return nil, fmt.Errorf("empty grid screenshot")
	}
	if strings.HasPrefix(imgBase64, "ERR:") {
		return nil, fmt.Errorf("grid screenshot: %s", imgBase64[4:])
	}

	// Send to VLM with grid-aware prompt
	cols := 3
	if info.GridSize == 16 {
		cols = 4
	}
	rows := info.GridSize / cols

	prompt := fmt.Sprintf(`This image shows a %d×%d grid of tiles from a visual puzzle.
Each tile is numbered 1 to %d, left-to-right, top-to-bottom.
Identify ALL tiles that contain: %s

Return ONLY a JSON array of tile numbers. Example: [1, 4, 7]
If no tiles match, return: []
Be thorough — select ALL matching tiles, including partially visible objects.`,
		rows, cols, info.GridSize, info.Target)

	// Multi-model voting for higher accuracy
	voters := s.Config.Text.Voters
	if len(voters) == 0 {
		voters = DefaultVoters()
	}

	var results []gridVoteResult
	for _, v := range voters {
		resp, err := s.AI.Complete(ctx, ai.Request{
			Provider:    v.Provider,
			Model:       v.Model,
			Prompt:      prompt,
			MaxTokens:   256,
			Temperature: 0.1, // low temp for classification
			ImageBase64: imgBase64,
			ImageType:   "image/jpeg",
		})
		if err != nil {
			log.Printf("[recaptcha] VLM %s failed: %v", v.Model, err)
			continue
		}
		tiles := parseGridResponse(resp.Content, info.GridSize)
		results = append(results, gridVoteResult{tiles: tiles, model: v.Model})
		log.Printf("[recaptcha] VLM %s → tiles=%v", v.Model, tiles)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("all VLM models failed")
	}

	// Intersection voting: tile must be selected by majority
	return intersectionVote(results, info.GridSize), nil
}

// ─── Strategy B: Per-tile classification ───────────────────────────────────

func (s *Solver) solvePerTile(ctx context.Context, sess *vision.Session, info *challengeInfo) ([]int, error) {
	// Extract each tile as individual base64 image
	tiles, err := s.extractTiles(sess, info.GridSize)
	if err != nil {
		return nil, err
	}

	prompt := fmt.Sprintf(`Does this image contain a %s? Answer ONLY "yes" or "no".`, info.Target)

	var selected []int
	for i, tileImg := range tiles {
		if tileImg == "" {
			continue
		}
		resp, err := s.AI.Complete(ctx, ai.Request{
			Provider:    s.Config.DefaultProvider,
			Model:       s.Config.DefaultModel,
			Prompt:      prompt,
			MaxTokens:   8,
			Temperature: 0.1,
			ImageBase64: tileImg,
			ImageType:   "image/jpeg",
		})
		if err != nil {
			log.Printf("[recaptcha] tile %d classify failed: %v", i+1, err)
			continue
		}
		answer := strings.ToLower(strings.TrimSpace(resp.Content))
		if strings.Contains(answer, "yes") {
			selected = append(selected, i+1)
		}
		// Rate limit protection
		time.Sleep(200 * time.Millisecond)
	}

	return selected, nil
}

func (s *Solver) extractTiles(sess *vision.Session, gridSize int) ([]string, error) {
	js := fmt.Sprintf(`
		var cf = document.querySelector(%q);
		if (!cf) return "[]";
		try {
			var doc = cf.contentDocument;
			var imgs = doc.querySelectorAll(".rc-imageselect-tile img");
			var result = [];
			for (var i = 0; i < imgs.length; i++) {
				var c = document.createElement("canvas");
				c.width = imgs[i].naturalWidth || imgs[i].width;
				c.height = imgs[i].naturalHeight || imgs[i].height;
				c.getContext("2d").drawImage(imgs[i], 0, 0);
				result.push(c.toDataURL("image/jpeg", 0.85).split(",")[1]);
			}
			return JSON.stringify(result);
		} catch(e) { return "[]"; }
	`, rcChallengeSel)

	result, _, err := sess.Eval(js)
	if err != nil {
		return nil, err
	}
	var tiles []string
	json.Unmarshal([]byte(result), &tiles)
	return tiles, nil
}

// ─── Tile clicking ─────────────────────────────────────────────────────────

func (s *Solver) clickTile(sess *vision.Session, tileNum, gridSize int) error {
	// tileNum is 1-indexed
	idx := tileNum - 1
	js := fmt.Sprintf(`
		var cf = document.querySelector(%q);
		if (!cf) return "ERR:no_iframe";
		try {
			var doc = cf.contentDocument;
			var tiles = doc.querySelectorAll(".rc-imageselect-tile");
			if (%d >= tiles.length) return "ERR:tile_out_of_range";
			tiles[%d].click();
			return "clicked";
		} catch(e) { return "ERR:" + e.message; }
	`, rcChallengeSel, idx, idx)

	result, _, _ := sess.Eval(js)
	if strings.HasPrefix(result, "ERR:") {
		return errors.New(result)
	}
	return nil
}

func (s *Solver) clickVerify(sess *vision.Session) {
	js := fmt.Sprintf(`
		var cf = document.querySelector(%q);
		if (!cf) return "ERR:no_iframe";
		try {
			var btn = cf.contentDocument.querySelector(%q);
			if (!btn) return "ERR:no_verify_btn";
			btn.click();
			return "clicked";
		} catch(e) { return "ERR:" + e.message; }
	`, rcChallengeSel, rcVerifyBtnSel)
	result, _, _ := sess.Eval(js)
	if strings.HasPrefix(result, "ERR:") {
		log.Printf("[recaptcha] clickVerify: %s", result)
	}
}

// ─── Solved check ──────────────────────────────────────────────────────────

func (s *Solver) isRecaptchaSolved(sess *vision.Session) bool {
	js := fmt.Sprintf(`
		var iframe = document.querySelector(%q);
		if (!iframe) return "unknown";
		try {
			var anchor = iframe.contentDocument.querySelector(%q);
			if (!anchor) return "unknown";
			return anchor.getAttribute("aria-checked");
		} catch(e) { return "error:" + e.message; }
	`, rcCheckboxSel, rcAnchorChecked)

	result, _, _ := sess.Eval(js)
	return result == "true"
}

// ─── Dynamic tiles (new tiles appear after selection) ──────────────────────

func (s *Solver) hasDynamicTiles(sess *vision.Session) bool {
	js := fmt.Sprintf(`
		var cf = document.querySelector(%q);
		if (!cf) return "false";
		try {
			var doc = cf.contentDocument;
			var dynamic = doc.querySelector(".rc-imageselect-dynamic-selected");
			return dynamic ? "true" : "false";
		} catch(e) { return "false"; }
	`, rcChallengeSel)
	result, _, _ := sess.Eval(js)
	return result == "true"
}

func (s *Solver) handleDynamicTiles(ctx context.Context, sess *vision.Session, info *challengeInfo, rcCfg RecaptchaConfig) {
	maxIter := rcCfg.DynamicMaxIterations
	if maxIter <= 0 {
		maxIter = 4
	}
	for i := 0; i < maxIter; i++ {
		time.Sleep(1 * time.Second) // wait for new tile to load

		// Re-extract and classify new tiles
		tiles, err := s.solveFullGrid(ctx, sess, info)
		if err != nil || len(tiles) == 0 {
			break
		}
		for _, t := range tiles {
			s.clickTile(sess, t, info.GridSize)
			randomDelay(int(rcCfg.ActionDelayMinMs), int(rcCfg.ActionDelayMaxMs))
		}
	}
}

// ─── Audio fallback via Whisper STT ────────────────────────────────────────

func (s *Solver) tryAudioFallback(ctx context.Context, sess *vision.Session, rcCfg RecaptchaConfig) *SolveResponse {
	if rcCfg.WhisperURL == "" {
		return &SolveResponse{Solved: false, Error: "whisper_url not configured"}
	}

	// Click audio button
	js := fmt.Sprintf(`
		var cf = document.querySelector(%q);
		if (!cf) return "ERR:no_iframe";
		try {
			var btn = cf.contentDocument.querySelector("#recaptcha-audio-button");
			if (!btn) return "ERR:no_audio_btn";
			btn.click();
			return "clicked";
		} catch(e) { return "ERR:" + e.message; }
	`, rcChallengeSel)
	result, _, _ := sess.Eval(js)
	if strings.HasPrefix(result, "ERR:") {
		return &SolveResponse{Solved: false, Error: "audio switch failed: " + result, Method: "audio"}
	}

	time.Sleep(2 * time.Second)

	// Check if blocked ("Your computer or network may be sending automated queries")
	blockJS := fmt.Sprintf(`
		var cf = document.querySelector(%q);
		if (!cf) return "";
		try {
			return cf.contentDocument.body.innerText;
		} catch(e) { return ""; }
	`, rcChallengeSel)
	blockText, _, _ := sess.Eval(blockJS)
	if strings.Contains(blockText, "automated queries") {
		return &SolveResponse{Solved: false, Error: "audio blocked by Google", Method: "audio"}
	}

	// Extract audio URL
	audioJS := fmt.Sprintf(`
		var cf = document.querySelector(%q);
		if (!cf) return "";
		try {
			var audio = cf.contentDocument.querySelector("#audio-source");
			return audio ? audio.src : "";
		} catch(e) { return ""; }
	`, rcChallengeSel)
	audioURL, _, _ := sess.Eval(audioJS)
	if audioURL == "" || audioURL == "<nil>" {
		return &SolveResponse{Solved: false, Error: "no audio source found", Method: "audio"}
	}

	// Send to Whisper STT
	transcription, err := s.transcribeAudio(ctx, audioURL, rcCfg.WhisperURL)
	if err != nil {
		return &SolveResponse{Solved: false, Error: fmt.Sprintf("transcribe: %v", err), Method: "audio"}
	}

	log.Printf("[recaptcha] audio transcription: %q", transcription)

	// Fill audio response
	fillJS := fmt.Sprintf(`
		var cf = document.querySelector(%q);
		if (!cf) return "ERR:no_iframe";
		try {
			var input = cf.contentDocument.querySelector("#audio-response");
			if (!input) return "ERR:no_audio_input";
			input.value = %q;
			input.dispatchEvent(new Event('input', {bubbles: true}));
			return "filled";
		} catch(e) { return "ERR:" + e.message; }
	`, rcChallengeSel, transcription)
	sess.Eval(fillJS)

	randomDelay(500, 1000)

	// Click verify
	s.clickVerify(sess)
	time.Sleep(2 * time.Second)

	if s.isRecaptchaSolved(sess) {
		return &SolveResponse{Solved: true, Type: "recaptcha_v2", Method: "audio_whisper", Rounds: 1}
	}
	return &SolveResponse{Solved: false, Error: "audio transcription rejected", Method: "audio_whisper"}
}

func (s *Solver) transcribeAudio(ctx context.Context, audioURL, whisperURL string) (string, error) {
	// Download audio and send to local Whisper STT
	// Whisper STT at localhost:8115 accepts POST /transcribe with audio file
	js := fmt.Sprintf(`
		var resp = await fetch(%q);
		var blob = await resp.blob();
		var fd = new FormData();
		fd.append("audio_file", blob, "audio.mp3");
		var stt = await fetch(%q + "/transcribe", {method: "POST", body: fd});
		var json = await stt.json();
		return json.text || json.transcription || "";
	`, audioURL, whisperURL)
	// This won't work via sess.Eval since fetch to localhost from reCAPTCHA origin is blocked.
	// Need server-side approach.
	_ = js

	// Server-side: download audio, send to Whisper
	// For now, return error — we'll use a proper HTTP client
	return "", fmt.Errorf("server-side audio transcription not yet implemented")
}

// gridVoteResult holds one model's tile selections for grid voting.
type gridVoteResult struct {
	tiles []int
	model string
}

// ─── Response parsing ──────────────────────────────────────────────────────

// parseGridResponse extracts tile numbers from VLM JSON response.
func parseGridResponse(raw string, maxTile int) []int {
	s := strings.TrimSpace(raw)
	// Strip markdown
	s = strings.Trim(s, "`")
	s = strings.TrimPrefix(s, "```json")
	s = strings.TrimPrefix(s, "```")
	s = strings.TrimSuffix(s, "```")
	s = strings.TrimSpace(s)

	var tiles []int
	if err := json.Unmarshal([]byte(s), &tiles); err != nil {
		// Try to find array in response
		start := strings.Index(s, "[")
		end := strings.LastIndex(s, "]")
		if start >= 0 && end > start {
			json.Unmarshal([]byte(s[start:end+1]), &tiles)
		}
	}

	// Filter valid range
	var valid []int
	for _, t := range tiles {
		if t >= 1 && t <= maxTile {
			valid = append(valid, t)
		}
	}
	return valid
}

// intersectionVote returns tiles that appear in majority of results.
func intersectionVote(results []gridVoteResult, gridSize int) []int {
	counts := make(map[int]int)
	for _, r := range results {
		for _, t := range r.tiles {
			counts[t]++
		}
	}

	threshold := len(results) / 2 // majority
	if threshold < 1 {
		threshold = 1
	}

	var selected []int
	for tile, count := range counts {
		if count >= threshold {
			selected = append(selected, tile)
		}
	}
	return selected
}

// ─── Helpers ───────────────────────────────────────────────────────────────

func randomDelay(minMs, maxMs int) {
	if maxMs <= minMs {
		maxMs = minMs + 100
	}
	d := time.Duration(minMs+rand.Intn(maxMs-minMs)) * time.Millisecond
	time.Sleep(d)
}
