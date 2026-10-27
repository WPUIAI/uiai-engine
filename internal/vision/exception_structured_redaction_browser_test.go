package vision

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestStructuredExceptionCredentialRedactionBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><title>Exception fixture</title><body>ordinary fixture</body></html>`))
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
	const canary = "UIAI_SYNTHETIC_EXCEPTION_0123456789"
	for _, payload := range []string{
		`JSON.stringify({password:"` + canary + `",echo:"` + canary + `"})`,
		`new Error(JSON.stringify({password:["` + canary + `"],echo:"` + canary + `"}))`,
		`JSON.stringify({echo:"` + canary + `",padding:"x".repeat(1200),password:"` + canary + `"})`,
	} {
		sess.ClearDiagnostics()
		_, _, err = sess.EvalAsync(`setTimeout(()=>{throw `+payload+`},0); await new Promise(resolve=>setTimeout(resolve,150)); return {scheduled:true}`, 5000)
		if err != nil {
			t.Fatal(err)
		}
		diagnostics := sess.Diagnostics(20, "all", false)
		if len(diagnostics.Exceptions) == 0 {
			t.Fatal("actual Chromium exception was not observed")
		}
		encoded, err := json.Marshal(diagnostics)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(encoded), canary) {
			t.Fatal("synthetic credential leaked through real browser diagnostics")
		}
		sess.diagnostics.mu.Lock()
		stored, storeErr := json.Marshal(sess.diagnostics.exceptions)
		sess.diagnostics.mu.Unlock()
		if storeErr != nil {
			t.Fatal(storeErr)
		}
		if strings.Contains(string(stored), canary) {
			t.Fatal("synthetic credential persisted in recorder")
		}
		if !strings.Contains(string(stored), "REDACTED_SECRET") {
			t.Fatal("redaction marker absent")
		}
	}
}
