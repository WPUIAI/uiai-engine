package focusapacket

import (
	"net/url"
	"strings"
	"testing"
)

func TestSanitizeURLCallbackAndNestedRedirects(t *testing.T) {
	const sentinel = "SYNTHETIC_OAUTH_CANARY"
	callback := "https://user:" + sentinel + "@example.test/callback?CoDe=" + sentinel + "&%73tate=" + sentinel + "&code=second" + sentinel + "&SAMLResponse=" + sentinel + "&assertion=" + sentinel + "&ok=useful#access_token=" + sentinel
	for _, raw := range []string{callback, "https://outer.test/?redirect=" + url.QueryEscape(callback), "https://outer.test/?redirect=" + url.QueryEscape(url.QueryEscape(callback)), "https://bad.test/?code=" + sentinel + "%invalid"} {
		got := SanitizeURL(raw)
		if strings.Contains(got, sentinel) {
			t.Fatal("callback canary leaked")
		}
	}
	got := SanitizeURL(callback)
	if !strings.Contains(got, "example.test/callback") || !strings.Contains(got, "ok=useful") {
		t.Fatalf("useful routing context was lost: %s", got)
	}
	if strings.Contains(got, "user:") || strings.Contains(got, "#") {
		t.Fatal("userinfo or fragment retained")
	}
}

func TestSanitizeURLBoundedNestedRedirects(t *testing.T) {
	raw := "https://example.test/callback?code=SYNTHETIC_DEEP_CANARY"
	for depth := 0; depth < 12; depth++ {
		raw = "https://redirect.test/?return_to=" + url.QueryEscape(raw)
	}
	if strings.Contains(SanitizeURL(raw), "SYNTHETIC_DEEP_CANARY") {
		t.Fatal("deep redirect leaked credential")
	}
}
