package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

// All advertised OpenAPI session operations must be mounted with their method.
func TestAdvertisedSessionOperationRoutes(t *testing.T) {
	router := chi.NewRouter()
	manager := vision.NewSessionManagerWithPools(nil)
	router.Route("/api/session", func(r chi.Router) { MountSessionRoutes(r, &config.Config{}, manager) })
	mounted := map[string]bool{}
	if err := chi.Walk(router, func(method, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
		mounted[method+" "+strings.TrimRight(route, "/")] = true
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	data, err := openapiEmbedFS.ReadFile("openapi_embed.json")
	if err != nil {
		t.Fatal(err)
	}
	var document struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err = json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	checked := 0
	for path, methods := range document.Paths {
		if path != "/api/session" && !strings.HasPrefix(path, "/api/session/") {
			continue
		}
		for method := range methods {
			if method == "parameters" {
				continue
			}
			checked++
			if !mounted[strings.ToUpper(method)+" "+path] {
				t.Errorf("advertised session operation is not mounted: %s %s", method, path)
			}
		}
	}
	if checked < 6 {
		t.Fatalf("unexpectedly sparse session route coverage: %d", checked)
	}
}

func TestReadTransportParityBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for live-router browser regression")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<html><body><main><p>` + strings.Repeat("bounded read ", 80) + `</p></main><script>window.readProbe=73</script></body></html>`))
	}))
	defer site.Close()
	pool, err := vision.NewPoolWithConfig(vision.PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := vision.NewSessionManager(pool)
	sess, _, err := manager.Open(site.URL, 800, 600)
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close(sess.ID)
	router := chi.NewRouter()
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	router.Route("/api/session", func(r chi.Router) { MountSessionRoutes(r, cfg, manager) })
	var firstText string
	for _, method := range []string{http.MethodGet, http.MethodPost} {
		path := "/api/session/" + sess.ID + "/read"
		body := ""
		if method == http.MethodGet {
			path += "?max_chars=80&include_links=true"
		} else {
			body = `{"max_chars":80,"include_links":true}`
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest(method, path, strings.NewReader(body)))
		if response.Code != http.StatusAccepted {
			t.Fatalf("%s status=%d body=%s", method, response.Code, response.Body.String())
		}
		var result map[string]any
		if err = json.Unmarshal(response.Body.Bytes(), &result); err != nil {
			t.Fatal(err)
		}
		text, _ := result["text"].(string)
		if len(text) != 80 || result["truncated"] != true || result["epwa_delivery"] == nil {
			t.Fatalf("%s read was not bounded/evidence-bound: %#v", method, result)
		}
		if firstText != "" && text != firstText {
			t.Fatal("GET and POST return different read content")
		}
		firstText = text
	}
	probe, _, err := sess.Eval("return window.readProbe")
	if err != nil || probe != "73" {
		t.Fatalf("read transport altered page state: %q %v", probe, err)
	}
}
