package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
)

func TestRedactedStringFormatting(t *testing.T) {
	secret := redactedString("dummy-credential-never-emit")
	for _, format := range []string{"%s", "%v", "%+v", "%#v", "%q", "%x"} {
		if out := fmt.Sprintf(format, secret); strings.Contains(out, secret.value()) {
			t.Fatalf("%s exposed credential", format)
		}
	}
	data, err := json.Marshal(struct{ Key redactedString }{secret})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), secret.value()) || !strings.Contains(string(data), "[secret]") {
		t.Fatal("JSON did not redact credential")
	}
	if secret.value() != "dummy-credential-never-emit" {
		t.Fatal("explicit transport boundary changed credential")
	}
}

func TestProviderHTTPErrorDoesNotEchoCredential(t *testing.T) {
	const dummy = "dummy-provider-key"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("rejected key=" + dummy))
	}))
	defer server.Close()
	p := &Provider{cfg: &config.Config{AI: config.AIConfig{Providers: map[string]config.AIProviderConfig{"openai": {APIURL: server.URL}, "anthropic": {APIURL: server.URL}}}}, client: server.Client(), keys: aiKeys{OpenAI: redactedString(dummy), Anthropic: redactedString(dummy)}}
	for _, provider := range []string{"openai", "anthropic"} {
		_, err := p.completeOnce(context.Background(), Request{Provider: provider, Model: "dummy", Prompt: "dummy"})
		if err == nil {
			t.Fatalf("%s accepted unauthorized response", provider)
		}
		if strings.Contains(err.Error(), dummy) {
			t.Fatalf("%s error exposed credential", provider)
		}
	}
}
