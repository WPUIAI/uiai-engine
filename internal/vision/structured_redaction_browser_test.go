package vision

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestStructuredCredentialRedactionBeforeEvalTruncationBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><head><title>Structured fixture</title></head><body>ordinary fixture</body></html>`))
	}))
	defer site.Close()
	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := NewSessionManager(pool)
	sess, _, err := manager.Open(site.URL, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(sess.ID)
	const canary = "UIAI_SYNTHETIC_CONTAINER_VALUE_0123456789"
	for index, script := range []string{
		`return {password:["` + canary + `"],echo:"` + canary + `"}`,
		`return {api_key:{primary:"` + canary + `"},echo:"` + canary + `"}`,
		`return {password:934567891,echo:934567891}`,
		`return {echo:"` + canary + `",padding:"x".repeat(1500),password:["` + canary + `"]}`,
		`return JSON.stringify({echo:"` + canary + `",padding:"x".repeat(1500),password:"` + canary + `"})`,
	} {
		for _, async := range []bool{false, true} {
			var value string
			var snapshot *SnapResult
			if async {
				value, snapshot, err = sess.EvalAsync(script, 5000)
			} else {
				value, snapshot, err = sess.Eval(script)
			}
			if err != nil {
				t.Fatal(err)
			}
			encoded, err := json.Marshal(map[string]any{"value": value, "snapshot": snapshot})
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(encoded), canary) || strings.Contains(string(encoded), "934567891") {
				t.Fatalf("structured synthetic credential leaked from browser consumer (case=%d async=%v value=%q)", index, async, strings.ReplaceAll(strings.ReplaceAll(value, canary, "SYNTHETIC_WITHHELD"), "934567891", "NUMERIC_WITHHELD"))
			}
			if !strings.Contains(value, "REDACTED_SECRET") {
				t.Fatal("structured evaluation lost its redaction marker")
			}
		}
	}
	value, _, err := sess.Eval(`return {answer:42,items:["ordinary",true,null]}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(value, `"answer":42`) || !strings.Contains(value, `"ordinary"`) {
		t.Fatal("ordinary structured evaluation no longer works")
	}
}
