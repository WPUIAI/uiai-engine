package vision

import (
	"encoding/json"
	"github.com/WPUIAI/uiai-engine/internal/secrettext"
	"github.com/go-rod/rod"
)

// safePageProjection classifies labels/controls and aliases before exposing
// action metadata or evaluation data; raw context never leaves this boundary.
func safePageProjection(page *rod.Page, values ...string) (string, []string) {
	raw := safeEvalStr(page, `() => JSON.stringify({title:document.title || '',body:document.body?.innerText || '',controls:Array.from(document.querySelectorAll('input,textarea,select')).map(e=>({tag:e.tagName.toLowerCase(),type:e.type||'',name:e.name||'',id:e.id||'',aria_label:e.getAttribute('aria-label')||'',value:e.value||''}))})`)
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
	clean := secrettext.Value(fields).(map[string]any)
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
func safePageTitle(page *rod.Page) string { title, _ := safePageProjection(page); return title }
