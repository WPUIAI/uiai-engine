package secrettext

import (
	"encoding/json"
	"sort"
	"strings"
)

// Value redacts copied structured projections, including aliases of values from
// credential-labelled controls; it does not mutate browser state or cache data.
func Value(input any) any {
	var fragments []string
	classified := map[string]bool{}
	// Credential-labelled containers classify every scalar leaf, including aliases
	// elsewhere in this projection; the transient set never becomes a secret cache.
	var classify func(any)
	classify = func(value any) {
		switch value := value.(type) {
		case []any:
			for _, child := range value {
				classify(child)
			}
		case map[string]any:
			for _, child := range value {
				classify(child)
			}
		default:
			if text := secretScalar(value); text != "" {
				classified[text] = true
				fragments = append(fragments, "API key "+text)
			}
		}
	}
	var collect func(any)
	collect = func(value any) {
		switch value := value.(type) {
		case string:
			fragments = append(fragments, value)
		case []any:
			for _, child := range value {
				collect(child)
			}
		case map[string]any:
			sensitive := false
			for _, key := range []string{"name", "id", "label", "aria_label", "type", "selector"} {
				if label, ok := value[key].(string); ok && SensitiveField(label) {
					sensitive = true
				}
			}
			tag, _ := value["tag"].(string)
			_, hasValue := value["value"]
			control := hasValue || tag == "input" || tag == "textarea" || tag == "select"
			keys := make([]string, 0, len(value))
			for key := range value {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				child := value[key]
				collect(child)
				if SensitiveField(key) || (control && sensitive && (key == "value" || key == "text")) {
					classify(child)
				}
			}
		}
	}
	collect(input)
	clean := Fragments(fragments)
	translations := map[string]string{}
	forced := make([]string, 0, len(classified))
	for value := range classified {
		forced = append(forced, value)
	}
	sort.Slice(forced, func(i, j int) bool {
		if len(forced[i]) == len(forced[j]) {
			return forced[i] < forced[j]
		}
		return len(forced[i]) > len(forced[j])
	})
	for i, fragment := range fragments {
		for _, value := range forced {
			clean[i] = strings.ReplaceAll(clean[i], value, Marker)
		}
		translations[fragment] = clean[i]
	}
	var clone func(any) any
	clone = func(value any) any {
		switch value := value.(type) {
		case string:
			if out, ok := translations[value]; ok {
				return out
			}
			return Text(value)
		case []any:
			out := make([]any, len(value))
			for i, child := range value {
				out[i] = clone(child)
			}
			return out
		case map[string]any:
			out := make(map[string]any, len(value))
			for key, child := range value {
				out[key] = clone(child)
				if SensitiveField(key) && child != nil && secretScalar(child) != "" {
					out[key] = Marker
				}
			}
			return out
		default:
			if text := secretScalar(value); text != "" && classified[text] {
				return Marker
			}
			return value
		}
	}
	return clone(input)
}

// Match scalar aliases without coercing ordinary structured output to strings.
func secretScalar(value any) string {
	switch value := value.(type) {
	case string:
		return value
	case json.Number, float64, float32, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, bool:
		if encoded, err := json.Marshal(value); err == nil {
			return string(encoded)
		}
	}
	return ""
}
