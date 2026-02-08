// Package jsonutil provides JSON repair utilities for LLM responses.
//
// Every AI model occasionally returns broken JSON — trailing commas,
// markdown code fences, control characters, unterminated strings.
// This package provides a multi-pass repair pipeline that matches
// the PHP plugin's salvage_json() and extract_json() behavior.
//
// Port of:
//   - class-design-critic.php::salvage_json()
//   - class-ui-reference-analyzer.php::extract_json()
//   - class-intake-ai-inference.php::extract_json()
package jsonutil

import (
	"encoding/json"
	"regexp"
	"strings"
	"unicode"
)

// Repair attempts to parse raw LLM output into valid JSON.
// It applies multiple repair passes in order:
//  1. Strip markdown code fences (```json ... ```)
//  2. Find JSON object/array boundaries
//  3. Remove control characters
//  4. Fix trailing commas before } or ]
//  5. Fix unescaped newlines in strings
//  6. Try to parse
//
// Returns the parsed result (map or slice), or nil + error if unrecoverable.
func Repair(raw string) (any, error) {
	// Pass 0: quick try — maybe it's already valid
	raw = strings.TrimSpace(raw)
	var quick any
	if err := json.Unmarshal([]byte(raw), &quick); err == nil {
		return quick, nil
	}

	// Pass 1: Strip markdown code fences
	cleaned := stripCodeFences(raw)

	// Pass 2: Find JSON boundaries (outermost { } or [ ])
	cleaned = extractJSONBoundaries(cleaned)

	// Pass 3: Remove control characters (except \n \r \t)
	cleaned = removeControlChars(cleaned)

	// Pass 4: Fix trailing commas
	cleaned = fixTrailingCommas(cleaned)

	// Try parse
	var result any
	if err := json.Unmarshal([]byte(cleaned), &result); err == nil {
		return result, nil
	}

	// Pass 5: Fix unescaped newlines inside string values
	cleaned = fixUnescapedNewlines(cleaned)

	// Final try
	if err := json.Unmarshal([]byte(cleaned), &result); err != nil {
		return nil, err
	}
	return result, nil
}

// RepairObject is like Repair but expects a JSON object (map).
// Returns nil if the result is not an object.
func RepairObject(raw string) (map[string]any, error) {
	result, err := Repair(raw)
	if err != nil {
		return nil, err
	}
	if m, ok := result.(map[string]any); ok {
		return m, nil
	}
	return nil, &json.UnmarshalTypeError{Value: "non-object", Type: nil}
}

// RepairArray is like Repair but expects a JSON array.
func RepairArray(raw string) ([]any, error) {
	result, err := Repair(raw)
	if err != nil {
		return nil, err
	}
	if a, ok := result.([]any); ok {
		return a, nil
	}
	return nil, &json.UnmarshalTypeError{Value: "non-array", Type: nil}
}

// --- internal repair passes ---

var codeFenceRe = regexp.MustCompile("(?s)```(?:json)?\\s*(.*?)\\s*```")

func stripCodeFences(s string) string {
	if m := codeFenceRe.FindStringSubmatch(s); len(m) > 1 {
		return m[1]
	}
	return s
}

func extractJSONBoundaries(s string) string {
	// Find outermost { } or [ ]
	objStart := strings.IndexByte(s, '{')
	arrStart := strings.IndexByte(s, '[')

	start := -1
	endChar := byte('}')

	if objStart >= 0 && (arrStart < 0 || objStart <= arrStart) {
		start = objStart
		endChar = '}'
	} else if arrStart >= 0 {
		start = arrStart
		endChar = ']'
	}

	if start < 0 {
		return s
	}

	end := strings.LastIndexByte(s, endChar)
	if end <= start {
		return s
	}

	return s[start : end+1]
}

func removeControlChars(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsControl(r) && r != '\n' && r != '\r' && r != '\t' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}

var trailingCommaRe = regexp.MustCompile(`,\s*([\]\}])`)

func fixTrailingCommas(s string) string {
	return trailingCommaRe.ReplaceAllString(s, "$1")
}

// fixUnescapedNewlines attempts to escape literal newlines that appear
// inside JSON string values. This is a best-effort heuristic.
func fixUnescapedNewlines(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	inString := false
	escaped := false

	for i := 0; i < len(s); i++ {
		c := s[i]
		if escaped {
			b.WriteByte(c)
			escaped = false
			continue
		}
		if c == '\\' && inString {
			b.WriteByte(c)
			escaped = true
			continue
		}
		if c == '"' {
			inString = !inString
			b.WriteByte(c)
			continue
		}
		if c == '\n' && inString {
			b.WriteString("\\n")
			continue
		}
		if c == '\r' && inString {
			// skip \r inside strings
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}
