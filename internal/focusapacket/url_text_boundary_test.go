package focusapacket

import (
	"strings"
	"testing"
)

func TestSanitizeTextPreservesCitationBoundaries(t *testing.T) {
	for _, tc := range []struct{ name, input, want string }{
		{"fragment", "[Overview](https://example.test/docs#PRIVATE_FRAGMENT)", "[Overview](https://example.test/docs)"},
		{"adjacent", "[one](https://one.test/#PRIVATE_ONE)[two](https://two.test/#PRIVATE_TWO)", "[one](https://one.test/)[two](https://two.test/)"},
		{"URL label", "[https://example.test/docs](https://example.test/docs#PRIVATE_FRAGMENT)", "[https://example.test/docs](https://example.test/docs)"},
		{"balanced path", "[math](https://example.test/Function_(mathematics)#PRIVATE_FRAGMENT)", "[math](https://example.test/Function_(mathematics))"},
		{"IPv6", "[local](http://[::1]:8080/path#PRIVATE_FRAGMENT)", "[local](http://[::1]:8080/path)"},
		{"image", "![image](https://example.test/img.png#PRIVATE_FRAGMENT)", "![image](https://example.test/img.png)"},
		{"code", "`https://example.test/docs#PRIVATE_FRAGMENT`", "`https://example.test/docs`"},
		{"prose", "See (https://example.test/docs#PRIVATE_FRAGMENT). Next.", "See (https://example.test/docs). Next."},
		{"whitespace", "https://one.test/#PRIVATE_ONE\u2003https://two.test/#PRIVATE_TWO", "https://one.test/\u2003https://two.test/"},
		{"credentials", "[safe](https://dummy-user:dummy-pass@example.test/path?token=SYNTHETIC_TOKEN&ok=yes#PRIVATE_FRAGMENT)", "[safe](https://example.test/path?ok=yes&token=REDACTED)"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeText(tc.input)
			if got != tc.want {
				t.Fatalf("citation boundary mismatch: got %q want %q", got, tc.want)
			}
			if SanitizeText(got) != got {
				t.Fatal("non-idempotent text redaction")
			}
		})
	}
}

func TestSanitizeTextCitationKeepsOAuthProtection(t *testing.T) {
	raw := "[continue](https://example.test/callback?code=SYNTHETIC_CODE&redirect_uri=https%3A%2F%2Fnext.test%2F%3Ftoken%3DSYNTHETIC_TOKEN#SYNTHETIC_FRAGMENT)[next](https://next.test/)"
	got := SanitizeText(raw)
	if strings.Contains(got, "SYNTHETIC") || !strings.Contains(got, ")[next](https://next.test/)") || !strings.Contains(got, "code=REDACTED") {
		t.Fatal("citation repair lost OAuth protection or adjacent link")
	}
	raw = strings.Repeat("[x](https://example.test/docs#SYNTHETIC_FRAGMENT)", 1000)
	got = SanitizeText(raw)
	if strings.Count(got, "[x](https://example.test/docs)") != 1000 || strings.Contains(got, "SYNTHETIC") {
		t.Fatal("long adjacent citation sequence corrupted")
	}
}
