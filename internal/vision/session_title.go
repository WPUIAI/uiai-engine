package vision

import (
	"encoding/json"
	"github.com/WPUIAI/uiai-engine/internal/secrettext"
	"github.com/go-rod/rod"
	"reflect"
)

// safePageProjection classifies labels/controls and aliases before exposing
// action metadata or evaluation data; raw context never leaves this boundary.
func safePageProjection(page *rod.Page, values ...string) (string, []string) {
	raw := safeEvalStr(page, `() => JSON.stringify({title:document.title || '',body:document.body?.innerText || '',controls:Array.from(document.querySelectorAll('input,textarea,select')).map(e=>({tag:e.tagName.toLowerCase(),type:e.type||'',name:e.name||'',id:e.id||'',aria_label:e.getAttribute('aria-label')||'',value:e.value||''}))})`)
	return redactPageProjection(raw, values...)
}

// Keep page privacy classification pure so the same production boundary can be
// exercised without Chrome, including unavailable-context failures.
func redactPageProjection(raw string, values ...string) (string, []string) {
	var fields map[string]any
	outputs := make([]any, len(values))
	for i, value := range values {
		outputs[i] = secrettext.DecodeProjection(value)
	}
	if json.Unmarshal([]byte(raw), &fields) != nil || fields == nil {
		out := make([]string, len(values))
		for i := range out {
			out[i] = secrettext.Marker
		}
		return secrettext.Marker, out
	}
	fields["outputs"] = outputs
	clean := redactPageFields(fields)
	title, _ := clean["title"].(string)
	out := make([]string, len(values))
	for i, value := range clean["outputs"].([]any) {
		if text, ok := value.(string); ok {
			out[i] = text
		} else if encoded, err := json.Marshal(value); err == nil {
			out[i] = string(encoded)
		} else {
			out[i] = secrettext.Marker
		}
		out[i] = sanitizeBrowserContent(out[i])
	}
	return sanitizeDiagnosticText(title), out
}

// Structured title roles remain transient classification context; never expose
// this derived projection as a response field or retain a secret-value cache.
func redactPageFields(fields map[string]any) map[string]any {
	var titleProjection []any
	if title, ok := fields["title"].(string); ok {
		prefix, value, suffix := secrettext.DecodeFramedProjection(title)
		titleProjection = []any{prefix, value, suffix}
		fields["title_projection"] = titleProjection
	}
	clean := secrettext.Value(fields).(map[string]any)
	if titleProjection != nil && !reflect.DeepEqual(titleProjection, clean["title_projection"]) {
		// A title containing classified data is private as a whole: replacing
		// decoded scalar aliases alone cannot remove every JSON escape spelling.
		// Reuse Value for the raw container and its identical metadata aliases.
		fields["sensitive_title"] = map[string]any{"secret": fields["title"]}
		clean = secrettext.Value(fields).(map[string]any)
	}
	return clean
}

func safePageTitle(page *rod.Page) string { title, _ := safePageProjection(page); return title }
