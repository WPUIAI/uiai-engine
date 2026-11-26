package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

// #81 acceptance: inject a dummy secret that does not look secret-shaped,
// then assert ZERO verbatim occurrences in every textual output channel -
// eval results, reads, snapshot text, and diagnostics. Binding is the
// guarantee pattern redaction cannot give.
func TestFillSecretNeverSurfacesInTextualChannels(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for fill_secret browser tests")
	}
	const dummy = "apple123" // deliberately non-secret-looking
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`<html><body><input id="pw" type="password">
		<span id="mirror"></span>
		<script>document.getElementById('pw').addEventListener('input', () => {
		document.getElementById('mirror').textContent = 'echo:' + document.getElementById('pw').value;
		console.log('filled with ' + document.getElementById('pw').value);
		});</script></body></html>`))
	}))
	defer site.Close()
	pool, err := vision.NewPoolWithConfig(vision.PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := vision.NewSessionManager(pool)
	defer manager.CloseAll()
	router := chi.NewRouter()
	MountSessionRoutes(router, &config.Config{}, manager)

	session, _, err := manager.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	// Inject the dummy secret through the broker fill path.
	filled := httptest.NewRecorder()
	router.ServeHTTP(filled, httptest.NewRequest("POST", "/"+session.ID+"/fill_secret",
		strings.NewReader(`{"selector":"#pw","value":"`+dummy+`","label":"test-credential"}`)))
	body := filled.Body.String()
	if filled.Code != http.StatusOK || !strings.Contains(body, `"bound":true`) {
		t.Fatalf("fill_secret failed: status=%d body=%.200s", filled.Code, body)
	}
	if strings.Contains(body, dummy) {
		t.Fatalf("fill_secret response echoed the value: %.200s", body)
	}

	// Eval result channel: reading the field value back surfaces the marker.
	echo, _, err := session.Eval(`return document.getElementById('pw').value;`)
	if err != nil {
		t.Fatalf("eval: %v", err)
	}
	if strings.Contains(echo, dummy) {
		t.Fatalf("eval result leaked the bound value: %q", echo)
	}

	// DOM readback channel: the mirror span also never carries it.
	sessText, err := session.TextContent("#mirror")
	if err == nil && strings.Contains(sessText, dummy) {
		t.Fatalf("text read leaked the bound value: %q", sessText)
	}

	// Diagnostics channel: the console echo of the injected value is scrubbed.
	diag := session.Diagnostics(50, "all", false)
	for _, entry := range diag.Console {
		if strings.Contains(entry.Text, dummy) {
			t.Fatalf("diagnostics console leaked the bound value: %q", entry.Text)
		}
	}

	// Snapshot channel: bound values never appear in the accessibility tree.
	snap, err := session.Snapshot(vision.SnapshotOptions{Interactive: true, MaxDepth: 12})
	if err == nil {
		if snap != nil && strings.Contains(snap.Tree, dummy) {
			t.Fatalf("snapshot leaked the bound value")
		}
	}
}
