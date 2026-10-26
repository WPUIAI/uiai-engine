package secrettext

import (
	"strings"
	"testing"
)

func TestSyntheticSecretTextAndSummary(t *testing.T) {
	const canary = "UIAI_SYNTHETIC_VALUE_0123456789"
	for name, input := range map[string]string{
		"api_neighbor":             "API key\n" + canary,
		"markdown":                 "API key\n```\n" + canary + "\n```",
		"bearer_label":             "Bearer token\n" + canary,
		"bearer":                   "Authorization: Bearer " + canary,
		"assignment":               "password=" + canary,
		"private_key":              "-----BEGIN PRIVATE KEY-----\n" + canary + "\n-----END PRIVATE KEY-----",
		"finite_material_withheld": "Recovery codes:\nSYNTHETIC_NOT_A_CODE\n" + canary,
	} {
		t.Run(name, func(t *testing.T) {
			out, summary := Redact(input)
			if strings.Contains(out, canary) || !strings.Contains(out, Marker) {
				t.Fatal("synthetic fixture was not withheld")
			}
			if summary.Count < 1 || len(summary.Classes) == 0 {
				t.Fatal("missing value-free redaction summary")
			}
			again, report := Redact(out)
			if again != out || report.Count != 0 {
				t.Fatal("redaction was not idempotent")
			}
		})
	}
}
func TestFragmentsAndStructuredCopies(t *testing.T) {
	const canary = "UIAI_SYNTHETIC_ALIAS_0123456789"
	for _, fragments := range [][]string{{"API key", canary, canary}, {"-----BEGIN PRIVATE KEY-----", canary, "-----END PRIVATE KEY-----"}, {"Recovery codes:", "SYNTHETIC_NOT_A_CODE", canary}} {
		for _, out := range Fragments(fragments) {
			if strings.Contains(out, canary) {
				t.Fatal("fragment alias escaped redaction")
			}
		}
	}
	original := map[string]any{"title": canary, "controls": []any{map[string]any{"tag": "input", "name": "api_key", "text": canary}}, "key": "Enter"}
	cleaned := Value(original).(map[string]any)
	if cleaned["title"] != Marker || cleaned["key"] != "Enter" || original["title"] != canary {
		t.Fatal("structured privacy/copy/ordinary-key boundary broken")
	}
}

func TestOrdinaryDocumentationRemainsReadable(t *testing.T) {
	for _, input := range []string{"API key documentation explains scopes", "Authorization: Bearer <YOUR_TOKEN>", "API key YOUR_API_KEY", "Read the recovery-code documentation before configuring account access.", "Ordinary article text and code examples."} {
		out, report := Redact(input)
		if out != input || report.Count != 0 {
			t.Fatal("ordinary documentation was unnecessarily masked")
		}
	}
}

func TestTokenTypeDocumentationDoesNotSeedSecretAliases(t *testing.T) {
	input := "switch tt {\ncase html.ErrorToken:\nreturn z.Err()\ncase html.TextToken:\nif depth > 0 { emit(z.Text()) }\n}\nspecification"
	clean, report := Redact(input)
	if clean != input || report.Count != 0 {
		t.Fatal("public tokenizer type labels were treated as credentials")
	}
	fragments := strings.Split(input, "\n")
	actual := Fragments(fragments)
	for i := range fragments {
		if actual[i] != fragments[i] {
			t.Fatal("public code label seeded a false fragment secret alias")
		}
	}
	for _, label := range []string{"token", "Token", "access_token", "authToken", "sessionToken", "csrfToken", "idToken", "dbPassword"} {
		for _, value := range []string{"if", "return", "UIAI_SYNTHETIC_VALUE_0123456789"} {
			clean, summary := Redact(label + "=" + value)
			if strings.HasSuffix(clean, "="+value) || summary.Count == 0 {
				t.Fatalf("actual labelled credential no longer protected: %s", label)
			}
		}
	}
}
