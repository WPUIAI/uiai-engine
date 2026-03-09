// Package reference implements the 4-pass UI Reference Analyzer.
//
// This is a direct port of PHP class-ui-reference-analyzer.php (1001 LOC).
// Each pass uses vision AI to extract increasingly detailed information:
//
//	Pass 1: analyze_reference   → page metadata + section inventory
//	Pass 2: extract_components  → component inventory per section
//	Pass 3: extract_tokens      → design tokens (colors, typography, spacing, shapes)
//	Pass 4: extract_spacing     → spacing system + alignment patterns
//
// The Go implementation stores prompts as constants and uses jsonutil.Repair
// for response parsing, matching the PHP behavior exactly.
package reference

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/philoveracity/uiai-engine/internal/ai"
	"github.com/philoveracity/uiai-engine/internal/jsonutil"
)

// Analyzer performs multi-pass vision analysis of reference screenshots.
type Analyzer struct {
	AI *ai.Provider
}

// NewAnalyzer creates a new reference analyzer.
func NewAnalyzer(aiProv *ai.Provider) *Analyzer {
	return &Analyzer{AI: aiProv}
}

// AnalysisResult is the output of Pass 1.
type AnalysisResult struct {
	Page     PageMeta  `json:"page"`
	Sections []Section `json:"sections"`
	Warnings []string  `json:"warnings,omitempty"`
}

type PageMeta struct {
	Type            string `json:"type"`
	AspectRatio     string `json:"aspect_ratio"`
	Background      string `json:"background"`
	PrimaryFunction string `json:"primary_function"`
}

type Section struct {
	Name        string   `json:"name"`
	HeightRatio string   `json:"height_ratio"`
	Background  string   `json:"background"`
	Components  []string `json:"components"`
}

// ComponentResult is the output of Pass 2.
type ComponentResult struct {
	Components []Component `json:"components"`
	Warnings   []string    `json:"warnings,omitempty"`
}

type Component struct {
	Section          string           `json:"section"`
	Type             string           `json:"type"`
	Position         string           `json:"position"`
	Geometry         string           `json:"geometry"`
	VisualAttributes VisualAttributes `json:"visual_attributes"`
	ContentHint      string           `json:"content_hint"`
}

type VisualAttributes struct {
	Fill   string `json:"fill"`
	Border string `json:"border"`
	Shadow string `json:"shadow"`
	Radius string `json:"radius"`
}

// TokenResult is the output of Pass 3.
type TokenResult struct {
	Colors     map[string]string `json:"colors"`
	Typography map[string]any    `json:"typography"`
	Spacing    map[string]string `json:"spacing"`
	Shapes     map[string]string `json:"shapes"`
	Warnings   []string          `json:"warnings,omitempty"`
}

// SpacingResult is the output of Pass 4.
type SpacingResult struct {
	BaseUnit          int               `json:"base_unit"`
	Scale             []int             `json:"scale"`
	Detected          map[string]any    `json:"detected"`
	VerticalRhythm    map[string]string `json:"vertical_rhythm"`
	HorizontalLayout  map[string]string `json:"horizontal_layout"`
	ContainerPadding  map[string]string `json:"container_padding"`
	AlignmentPatterns map[string]string `json:"alignment_patterns"`
	Warnings          []string          `json:"warnings,omitempty"`
}

// TextExtractionResult is the OCR-style output for visible image text.
type TextExtractionResult struct {
	Text     string   `json:"text"`
	Warnings []string `json:"warnings,omitempty"`
}

// ═══════════════════════════════════════════════════════════
// PASS 1: Reference Analysis
// ═══════════════════════════════════════════════════════════

// AnalyzeReference performs Pass 1: extract page metadata and section inventory.
func (a *Analyzer) AnalyzeReference(ctx context.Context, imageBase64, imageType, provider, model string) (*AnalysisResult, error) {
	resp, err := a.callVision(ctx, referenceAnalysisPrompt, imageBase64, imageType, provider, model)
	if err != nil {
		return nil, fmt.Errorf("pass 1 (analyze_reference): %w", err)
	}

	parsed, err := jsonutil.RepairObject(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("pass 1 parse: %w", err)
	}

	result := &AnalysisResult{}

	// Extract page metadata
	if page, ok := parsed["page"].(map[string]any); ok {
		result.Page = PageMeta{
			Type:            strVal(page, "type"),
			AspectRatio:     strVal(page, "aspect_ratio"),
			Background:      strVal(page, "background"),
			PrimaryFunction: strVal(page, "primary_function"),
		}
	}

	// Extract sections
	if sections, ok := parsed["sections"].([]any); ok {
		for _, s := range sections {
			sm, ok := s.(map[string]any)
			if !ok {
				continue
			}
			sec := Section{
				Name:        strVal(sm, "name"),
				HeightRatio: strVal(sm, "height_ratio"),
				Background:  strVal(sm, "background"),
			}
			if comps, ok := sm["components"].([]any); ok {
				for _, c := range comps {
					if cs, ok := c.(string); ok {
						sec.Components = append(sec.Components, cs)
					}
				}
			}
			result.Sections = append(result.Sections, sec)
		}
	}

	// Validate
	result.Warnings = validateAnalysis(result)

	return result, nil
}

// ═══════════════════════════════════════════════════════════
// PASS 2: Component Extraction
// ═══════════════════════════════════════════════════════════

// ExtractComponents performs Pass 2: extract component inventory.
func (a *Analyzer) ExtractComponents(ctx context.Context, imageBase64, imageType, provider, model string, sections []Section) (*ComponentResult, error) {
	prompt := buildComponentPrompt(sections)

	resp, err := a.callVision(ctx, prompt, imageBase64, imageType, provider, model)
	if err != nil {
		return nil, fmt.Errorf("pass 2 (extract_components): %w", err)
	}

	parsed, err := jsonutil.RepairObject(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("pass 2 parse: %w", err)
	}

	result := &ComponentResult{}

	if comps, ok := parsed["components"].([]any); ok {
		for _, c := range comps {
			cm, ok := c.(map[string]any)
			if !ok {
				continue
			}
			comp := Component{
				Section:     strVal(cm, "section"),
				Type:        strVal(cm, "type"),
				Position:    strVal(cm, "position"),
				Geometry:    strVal(cm, "geometry"),
				ContentHint: strVal(cm, "content_hint"),
			}
			if va, ok := cm["visual_attributes"].(map[string]any); ok {
				comp.VisualAttributes = VisualAttributes{
					Fill:   strVal(va, "fill"),
					Border: strVal(va, "border"),
					Shadow: strVal(va, "shadow"),
					Radius: strVal(va, "radius"),
				}
			}
			result.Components = append(result.Components, comp)
		}
	}

	result.Warnings = validateComponents(result)

	return result, nil
}

// ═══════════════════════════════════════════════════════════
// PASS 3: Token Extraction
// ═══════════════════════════════════════════════════════════

// ExtractTokens performs Pass 3: extract design system tokens.
func (a *Analyzer) ExtractTokens(ctx context.Context, imageBase64, imageType, provider, model string) (*TokenResult, error) {
	resp, err := a.callVision(ctx, tokenExtractionPrompt, imageBase64, imageType, provider, model)
	if err != nil {
		return nil, fmt.Errorf("pass 3 (extract_tokens): %w", err)
	}

	parsed, err := jsonutil.RepairObject(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("pass 3 parse: %w", err)
	}

	result := &TokenResult{
		Colors:     extractStringMap(parsed, "colors"),
		Typography: extractAnyMap(parsed, "typography"),
		Spacing:    extractStringMap(parsed, "spacing"),
		Shapes:     extractStringMap(parsed, "shapes"),
	}

	result.Warnings = validateTokens(result)

	return result, nil
}

// ═══════════════════════════════════════════════════════════
// PASS 4: Spacing Extraction
// ═══════════════════════════════════════════════════════════

// ExtractSpacing performs Pass 4: extract spacing system and alignment.
func (a *Analyzer) ExtractSpacing(ctx context.Context, imageBase64, imageType, provider, model string, components []Component) (*SpacingResult, error) {
	prompt := buildSpacingPrompt(components)

	resp, err := a.callVision(ctx, prompt, imageBase64, imageType, provider, model)
	if err != nil {
		return nil, fmt.Errorf("pass 4 (extract_spacing): %w", err)
	}

	parsed, err := jsonutil.RepairObject(resp.Content)
	if err != nil {
		return nil, fmt.Errorf("pass 4 parse: %w", err)
	}

	result := &SpacingResult{
		Detected:          extractAnyMap(parsed, "detected"),
		VerticalRhythm:    extractStringMap(parsed, "vertical_rhythm"),
		HorizontalLayout:  extractStringMap(parsed, "horizontal_layout"),
		ContainerPadding:  extractStringMap(parsed, "container_padding"),
		AlignmentPatterns: extractStringMap(parsed, "alignment_patterns"),
	}

	if bu, ok := parsed["base_unit"].(float64); ok {
		result.BaseUnit = int(bu)
	} else {
		result.BaseUnit = 8
	}

	if scale, ok := parsed["scale"].([]any); ok {
		for _, s := range scale {
			if n, ok := s.(float64); ok {
				result.Scale = append(result.Scale, int(n))
			}
		}
	}
	if len(result.Scale) == 0 {
		result.Scale = []int{8, 16, 24, 32, 48, 64, 96}
	}

	return result, nil
}

// ═══════════════════════════════════════════════════════════
// TEXT EXTRACTION
// ═══════════════════════════════════════════════════════════

// ExtractText performs OCR-style visible text extraction from an image.
func (a *Analyzer) ExtractText(ctx context.Context, imageBase64, imageType, provider, model string) (*TextExtractionResult, error) {
	prompt := strings.Join([]string{
		"Read the image and extract only the visible human-readable text.",
		"Return plain text only.",
		"Preserve line breaks when they help readability.",
		"Do not describe the image.",
		"Do not return JSON, markdown, labels, or explanations.",
		"If the image is a verification challenge or captcha, return only the exact uppercase alphanumeric token shown in the image, with no spaces or commentary.",
	}, " ")

	resolvedProvider := strings.TrimSpace(provider)
	resolvedModel := strings.TrimSpace(model)
	warnings := []string{}

	// MiniMax docs split image handling into two paths:
	// - Coding Plan keys expose image understanding via the coding-plan VLM/MCP path.
	// - Generic text API image examples use MiniMax-Text-01 on /v1/text/chatcompletion_v2.
	if resolvedProvider == "" || strings.EqualFold(resolvedProvider, "minimax") {
		if resolvedProvider == "" {
			resolvedProvider = "minimax"
			warnings = append(warnings, "OCR defaulted provider to MiniMax to avoid caller-side hardcoded OpenRouter routing.")
		}
		if resolvedModel == "" || strings.EqualFold(resolvedModel, "MiniMax-M2.5") {
			resolvedModel = "MiniMax-Text-01"
			warnings = append(warnings, "MiniMax OCR switched away from the generic M2.5 text route; Coding Plan keys may use coding-plan VLM, while generic text API image examples use MiniMax-Text-01.")
		}
	}

	resp, err := a.callVision(ctx, prompt, imageBase64, imageType, resolvedProvider, resolvedModel)
	if err != nil {
		if strings.EqualFold(resolvedProvider, "minimax") {
			localText, localWarnings, localErr := extractTextWithLocalTesseract(ctx, imageBase64, imageType)
			if localErr == nil && strings.TrimSpace(localText) != "" {
				warnings = append(warnings, "MiniMax OCR failed; local Tesseract fallback used.")
				warnings = append(warnings, localWarnings...)
				return &TextExtractionResult{Text: strings.TrimSpace(localText), Warnings: warnings}, nil
			}
		}
		return nil, fmt.Errorf("text extraction: %w", err)
	}

	text := strings.TrimSpace(resp.Content)
	if text == "" && strings.EqualFold(resolvedProvider, "minimax") {
		localText, localWarnings, localErr := extractTextWithLocalTesseract(ctx, imageBase64, imageType)
		if localErr == nil && strings.TrimSpace(localText) != "" {
			warnings = append(warnings, "MiniMax OCR returned empty content; local Tesseract fallback used.")
			warnings = append(warnings, localWarnings...)
			text = strings.TrimSpace(localText)
		}
	}

	result := &TextExtractionResult{
		Text:     text,
		Warnings: warnings,
	}
	if result.Text == "" {
		result.Warnings = append(result.Warnings, "No text extracted from image.")
	}
	return result, nil
}

var nonAlnumOCR = regexp.MustCompile(`[^A-Z0-9]+`)

func extractTextWithLocalTesseract(ctx context.Context, imageBase64, imageType string) (string, []string, error) {
	if strings.TrimSpace(imageBase64) == "" {
		return "", nil, fmt.Errorf("empty image")
	}
	if _, err := exec.LookPath("tesseract"); err != nil {
		return "", nil, err
	}
	if _, err := exec.LookPath("convert"); err != nil {
		return "", nil, err
	}

	imgBytes, err := base64.StdEncoding.DecodeString(imageBase64)
	if err != nil {
		return "", nil, err
	}

	tmpDir, err := os.MkdirTemp("", "ocr-tess-*")
	if err != nil {
		return "", nil, err
	}
	defer os.RemoveAll(tmpDir)

	ext := ".jpg"
	switch strings.ToLower(strings.TrimSpace(imageType)) {
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	case "image/jpeg", "image/jpg", "":
		ext = ".jpg"
	}
	inputPath := filepath.Join(tmpDir, "input"+ext)
	if err := os.WriteFile(inputPath, imgBytes, 0o600); err != nil {
		return "", nil, err
	}

	variants := []struct {
		name string
		args []string
	}{
		{"gray-threshold", []string{inputPath, "-colorspace", "Gray", "-resize", "350%", "-contrast-stretch", "0", "-threshold", "55%", filepath.Join(tmpDir, "v1.png")}},
		{"gray-sharpen", []string{inputPath, "-colorspace", "Gray", "-resize", "400%", "-contrast-stretch", "0", "-sharpen", "0x1", filepath.Join(tmpDir, "v2.png")}},
		{"gray-negate", []string{inputPath, "-colorspace", "Gray", "-resize", "350%", "-negate", "-threshold", "60%", filepath.Join(tmpDir, "v3.png")}},
	}

	candidate := ""
	warnings := []string{}
	for _, variant := range variants {
		cmd := exec.CommandContext(ctx, "convert", variant.args...)
		if out, err := cmd.CombinedOutput(); err != nil {
			warnings = append(warnings, fmt.Sprintf("convert %s failed: %s", variant.name, strings.TrimSpace(string(out))))
			continue
		}
		outPath := variant.args[len(variant.args)-1]
		for _, psm := range []string{"8", "7", "6"} {
			cmd = exec.CommandContext(ctx, "tesseract", outPath, "stdout", "--psm", psm, "-l", "eng", "-c", "tessedit_char_whitelist=ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
			out, err := cmd.CombinedOutput()
			if err != nil {
				warnings = append(warnings, fmt.Sprintf("tesseract %s/psm%s failed: %s", variant.name, psm, strings.TrimSpace(string(out))))
				continue
			}
			text := normalizeOCRToken(string(out))
			if scoreOCRCandidate(text) > scoreOCRCandidate(candidate) {
				candidate = text
			}
		}
	}

	if candidate == "" {
		return "", warnings, fmt.Errorf("no local OCR text extracted")
	}
	return candidate, warnings, nil
}

func normalizeOCRToken(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = nonAlnumOCR.ReplaceAllString(s, "")
	return strings.TrimSpace(s)
}

func scoreOCRCandidate(s string) int {
	if s == "" {
		return 0
	}
	score := len(s)
	if len(s) >= 4 && len(s) <= 8 {
		score += 20
	}
	if len(s) == 6 {
		score += 15
	}
	return score
}

// ═══════════════════════════════════════════════════════════
// FULL ANALYSIS (all 4 passes)
// ═══════════════════════════════════════════════════════════

// FullAnalysis holds the combined result of all 4 passes.
type FullAnalysis struct {
	Analysis   *AnalysisResult  `json:"analysis"`
	Components *ComponentResult `json:"components"`
	Tokens     *TokenResult     `json:"tokens"`
	Spacing    *SpacingResult   `json:"spacing"`
	Model      string           `json:"model"`
	Provider   string           `json:"provider"`
}

// AnalyzeFull runs all 4 passes sequentially.
func (a *Analyzer) AnalyzeFull(ctx context.Context, imageBase64, imageType, provider, model string) (*FullAnalysis, error) {
	// Pass 1
	analysis, err := a.AnalyzeReference(ctx, imageBase64, imageType, provider, model)
	if err != nil {
		return nil, err
	}

	// Pass 2
	components, err := a.ExtractComponents(ctx, imageBase64, imageType, provider, model, analysis.Sections)
	if err != nil {
		return nil, err
	}

	// Pass 3
	tokens, err := a.ExtractTokens(ctx, imageBase64, imageType, provider, model)
	if err != nil {
		return nil, err
	}

	// Pass 4
	spacing, err := a.ExtractSpacing(ctx, imageBase64, imageType, provider, model, components.Components)
	if err != nil {
		return nil, err
	}

	return &FullAnalysis{
		Analysis:   analysis,
		Components: components,
		Tokens:     tokens,
		Spacing:    spacing,
		Model:      model,
		Provider:   provider,
	}, nil
}

// ═══════════════════════════════════════════════════════════
// AI CALLER
// ═══════════════════════════════════════════════════════════

func (a *Analyzer) callVision(ctx context.Context, prompt, imageBase64, imageType, provider, model string) (*ai.Response, error) {
	return a.AI.Complete(ctx, ai.Request{
		Provider:    provider,
		Model:       model,
		Prompt:      prompt,
		MaxTokens:   4096,
		Temperature: 0.5,
		ImageBase64: imageBase64,
		ImageType:   imageType,
	})
}

// ═══════════════════════════════════════════════════════════
// VALIDATORS
// ═══════════════════════════════════════════════════════════

func validateAnalysis(r *AnalysisResult) []string {
	var w []string
	if len(r.Sections) < 3 {
		w = append(w, fmt.Sprintf("Only %d sections identified (minimum 3 expected).", len(r.Sections)))
	}
	if r.Page.Type == "" {
		w = append(w, "Page metadata missing: type")
	}
	if r.Page.Background == "" {
		w = append(w, "Page metadata missing: background")
	}
	for i, s := range r.Sections {
		if s.Name == "" {
			w = append(w, fmt.Sprintf("Section %d missing: name", i))
		}
		if s.HeightRatio == "" {
			w = append(w, fmt.Sprintf("Section %d missing: height_ratio", i))
		}
	}
	return w
}

func validateComponents(r *ComponentResult) []string {
	var w []string
	if len(r.Components) < 8 {
		w = append(w, fmt.Sprintf("Only %d components identified (minimum 8 expected).", len(r.Components)))
	}
	heroCount := 0
	for _, c := range r.Components {
		if strings.EqualFold(c.Section, "hero") {
			heroCount++
		}
	}
	if heroCount < 3 {
		w = append(w, "Hero section has fewer than 3 components.")
	}
	return w
}

func validateTokens(r *TokenResult) []string {
	var w []string
	requiredColors := []string{"background_primary", "text_primary", "accent_primary"}
	for _, c := range requiredColors {
		if r.Colors[c] == "" {
			w = append(w, fmt.Sprintf("Missing color token: %s", c))
		}
	}
	if r.Typography["heading_font"] == nil {
		w = append(w, "Missing typography token: heading_font")
	}
	return w
}

// ═══════════════════════════════════════════════════════════
// HELPERS
// ═══════════════════════════════════════════════════════════

func strVal(m map[string]any, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func extractStringMap(m map[string]any, key string) map[string]string {
	result := make(map[string]string)
	if sub, ok := m[key].(map[string]any); ok {
		for k, v := range sub {
			if s, ok := v.(string); ok {
				result[k] = s
			}
		}
	}
	return result
}

func extractAnyMap(m map[string]any, key string) map[string]any {
	if sub, ok := m[key].(map[string]any); ok {
		return sub
	}
	return map[string]any{}
}
