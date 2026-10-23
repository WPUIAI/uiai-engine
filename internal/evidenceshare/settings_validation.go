package evidenceshare

import (
	"fmt"
	"math"
	"reflect"
	"strings"
)

// DefaultSettings owns the field inventory and scalar types. Validation extends
// that inventory with value constraints rather than maintaining a second schema.
func validatePatch(p map[string]any) error {
	if p == nil {
		return fmt.Errorf("%w: settings object required", ErrSettingsInvalid)
	}
	defaults := DefaultSettings()
	for domain, raw := range p {
		fields, known := defaults[domain].(map[string]any)
		if !known {
			return fmt.Errorf("%w: unknown domain %q", ErrSettingsInvalid, domain)
		}
		values, ok := raw.(map[string]any)
		if !ok || len(values) == 0 {
			return fmt.Errorf("%w: %s must be a non-empty object", ErrSettingsInvalid, domain)
		}
		for field, value := range values {
			path := domain + "." + field
			example, exists := fields[field]
			if !exists {
				return fmt.Errorf("%w: unknown field %s", ErrSettingsInvalid, path)
			}
			valid := false
			switch example.(type) {
			case bool:
				_, valid = value.(bool)
			case string:
				text, ok := value.(string)
				valid = ok && text != "" && len(text) <= 4096 && !strings.ContainsAny(text, "\x00\r\n")
			case int:
				number, ok := settingNumber(value)
				valid = ok && number >= 0 && number <= 9007199254740991 && math.Trunc(number) == number
				if domain == "image" || domain == "video" {
					if field == "quality" {
						valid = valid && number >= 1 && number <= 100
					}
				}
				if field == "quota_headroom_percent" {
					valid = valid && number <= 100
				}
				if field == "list_page_size" || field == "max_width" || field == "max_height" || field == "thumbnail_width" || field == "max_duration_seconds" {
					valid = valid && number >= 1
				}
			}
			if !valid {
				return fmt.Errorf("%w: invalid value for %s", ErrSettingsInvalid, path)
			}
			if path == "enablement.read_only" && value != true {
				return fmt.Errorf("%w: evidence packages remain read-only", ErrSettingsInvalid)
			}
			if choices, restricted := settingsChoices[path]; restricted && !choices[value.(string)] {
				return fmt.Errorf("%w: unsupported value for %s", ErrSettingsInvalid, path)
			}
			if path == "storage.location_template" && !safeLocationTemplate(value.(string)) {
				return fmt.Errorf("%w: storage template must remain relative and use known placeholders", ErrSettingsInvalid)
			}
		}
	}
	return nil
}

var settingsChoices = map[string]map[string]bool{
	"image.format":       {"png": true, "jpg": true, "jpeg": true, "webp": true},
	"video.format":       {"webm": true, "mp4": true},
	"presentation.theme": {"system": true, "light": true, "dark": true},
}

func settingNumber(value any) (float64, bool) {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(v.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(v.Uint()), true
	case reflect.Float32, reflect.Float64:
		n := v.Float()
		return n, !math.IsNaN(n) && !math.IsInf(n, 0)
	default:
		return 0, false
	}
}

func safeLocationTemplate(value string) bool {
	if strings.HasPrefix(value, "/") || strings.ContainsAny(value, "\\:%") {
		return false
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." {
			return false
		}
	}
	value = strings.ReplaceAll(strings.ReplaceAll(value, "{project}", "project"), "{workstream}", "workstream")
	return !strings.ContainsAny(value, "{}")
}
