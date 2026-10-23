package vision

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
)

func TestSessionAndSnapshotURLSerialization(t *testing.T) {
	const original = "https://example.test/callback?code=SYNTHETIC_CODE&state=SYNTHETIC_STATE#SYNTHETIC_FRAGMENT"
	sess := &Session{ID: "synthetic-session", URL: original}
	for _, value := range []any{sess, SnapResult{URL: original}} {
		data, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "SYNTHETIC_") {
			t.Fatal("session/snapshot JSON leaked callback values")
		}
	}
	if sess.URL != original {
		t.Fatal("serialization changed private navigation state")
	}
}

func TestOAuthTelemetryRedactionBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for browser regression")
	}
	var logs callbackLogBuffer
	previous := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(previous)
	received := make(chan string, 1)
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case received <- r.URL.Query().Get("code"):
		default:
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body>callback fixture</body></html>`))
	}))
	defer site.Close()
	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := NewSessionManager(pool)
	sess, snap, err := manager.Open(site.URL+"?code=SYNTHETIC_CALLBACK&state=SYNTHETIC_STATE#SYNTHETIC_FRAGMENT", 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(sess.ID)
	if <-received != "SYNTHETIC_CALLBACK" {
		t.Fatal("telemetry sanitizer changed browser navigation")
	}
	read, err := sess.ReadPage(ReadOptions{})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal([]any{sess, snap, read})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "SYNTHETIC_") || strings.Contains(logs.String(), "SYNTHETIC_") {
		t.Fatal("callback canary appeared in serialized telemetry or logs")
	}
}

type callbackLogBuffer struct {
	mu   sync.Mutex
	data bytes.Buffer
}

func (b *callbackLogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.data.Write(p)
}
func (b *callbackLogBuffer) String() string { b.mu.Lock(); defer b.mu.Unlock(); return b.data.String() }
