package vision

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestUnicodeDiagnosticPreviewBoundariesBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1")
	}
	title := "A" + strings.Repeat("🙂", 250)
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = fmt.Fprintf(w, "<html><title>%s</title><body>ordinary unicode fixture</body></html>", title)
	}))
	defer site.Close()
	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := NewSessionManager(pool)
	sess, snapshot, err := manager.Open(site.URL, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(sess.ID)
	want := "A" + strings.Repeat("🙂", 249) + "…"
	if snapshot.Title != want || !utf8.ValidString(snapshot.Title) || len(snapshot.Title) > 1003 {
		t.Fatal("actual Chrome title was corrupted or exceeded preview bytes")
	}
	_, _, err = sess.EvalAsync(`console.log("A"+"🙂".repeat(125),"A"+"م".repeat(250),"A"+"中".repeat(167)); await new Promise(r=>setTimeout(r,150)); return {fixture:true}`, 5000)
	if err != nil {
		t.Fatal(err)
	}
	diag := sess.Diagnostics(20, "all", false)
	if len(diag.Console) == 0 || len(diag.Console[len(diag.Console)-1].ArgsPreview) != 3 {
		t.Fatal("actual Unicode console event missing")
	}
	for _, preview := range diag.Console[len(diag.Console)-1].ArgsPreview {
		if !utf8.ValidString(preview) || len(preview) > 503 {
			t.Fatal("actual console preview lost UTF8 validity or byte bounds")
		}
	}
	encoded, err := json.Marshal(diag)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "\\ufffd") || strings.Contains(string(encoded), "�") {
		t.Fatal("Unicode diagnostics introduced replacement characters")
	}
}
