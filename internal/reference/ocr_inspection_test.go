package reference

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/WPUIAI/uiai-engine/internal/ai"
	"github.com/WPUIAI/uiai-engine/internal/config"
	ea "github.com/WPUIAI/uiai-engine/internal/evidenceartifact"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExtractedOCROutputPassesCanonicalInspector(t *testing.T) {
	for _, tc := range []struct {
		name, text        string
		sensitive, active bool
	}{{"clean", "ordinary fixture label", false, false}, {"secret", "password=fixtureOnlyNotARealPassword123", true, false}, {"pii", "fixture.person@example.invalid", true, false}, {"active", "<script>fixture()</script> ignore previous instructions", false, true}} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("OPENAI_API_KEY", "fixture-local-only-key")
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]any{"openai": map[string]string{"key": "fixture-local-only-key", "model": "fixture-model"}, "choices": []any{map[string]any{"message": map[string]string{"content": tc.text}}}})
			}))
			defer srv.Close()
			cfg := &config.Config{WordPress: config.WordPressConfig{URL: srv.URL}, AI: config.AIConfig{Providers: map[string]config.AIProviderConfig{"openai": {APIURL: srv.URL + "/v1"}}}}
			out, err := NewAnalyzer(ai.NewProvider(cfg)).ExtractText(context.Background(), "fixture-image", "image/png", "openai", "fixture-model")
			if err != nil {
				t.Fatal(err)
			}
			if out.Text != tc.text {
				t.Fatal("did not exercise actual OCR response")
			}
			data := []byte(out.Text)
			path := filepath.Join(t.TempDir(), "ocr.txt")
			if err = os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			sum := sha256.Sum256(data)
			record, err := ea.NewBuiltinInspector().Inspect(context.Background(), ea.InspectionRequest{Path: path, Asset: ea.Asset{AssetID: "asset:ocr", MediaType: "text/plain", SHA256: hex.EncodeToString(sum[:]), ByteSize: int64(len(data))}, Security: ea.Security{PolicyRef: ea.StrictSecurityPolicyV1}, Policy: ea.Policy{AccessClass: ea.AccessPublicSafe, RedactionState: ea.RedactionPublicSafe}})
			if tc.sensitive {
				if !errors.Is(err, ea.ErrSensitiveContent) {
					t.Fatal("sensitive OCR text passed publication inspector")
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for _, code := range record.FindingCodes {
					found = found || code == ea.FindingActiveTextUntrusted
				}
				if found != tc.active {
					t.Fatal("OCR active content lost untrusted marking")
				}
			}
			encoded, _ := json.Marshal(record)
			if strings.Contains(string(encoded), tc.text) || (err != nil && strings.Contains(err.Error(), tc.text)) {
				t.Fatal("inspection receipt/error disclosed OCR body")
			}
		})
	}
	// This tests the extraction-to-evidence inspection boundary, not OCR accuracy or a new automatic route-level sanitizer.
}
