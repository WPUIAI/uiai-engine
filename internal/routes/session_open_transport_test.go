package routes

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func TestMCPOpenRedirectCompressionAndMalformedCleanup(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1 for real open transport")
	}
	node, err := exec.LookPath("node")
	if err != nil {
		t.Fatal(err)
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/github/login" {
			http.Redirect(w, r, "/callback?code=synthetic-oauth-code", 302)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!doctype html><title>Owned OAuth redirect fixture</title><p>Redirect complete</p><input type="password" aria-label="Synthetic credential field">`)
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
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	router.Route("/api/session", func(r chi.Router) { MountSessionRoutes(r, cfg, manager) })
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"plain", "gzip", "malformed"} {
		t.Run(mode, func(t *testing.T) {
			proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				result := httptest.NewRecorder()
				router.ServeHTTP(result, r)
				for key, values := range result.Header() {
					for _, value := range values {
						w.Header().Add(key, value)
					}
				}
				body := result.Body.Bytes()
				if r.Method == "POST" && r.URL.Path == "/api/session" && mode != "plain" {
					if mode == "gzip" {
						w.Header().Set("Content-Encoding", "gzip")
					}
					var compressed bytes.Buffer
					writer := gzip.NewWriter(&compressed)
					if _, err := writer.Write(body); err != nil {
						t.Error(err)
					}
					if err := writer.Close(); err != nil {
						t.Error(err)
					}
					body = compressed.Bytes()
				}
				w.WriteHeader(result.Code)
				if _, err := w.Write(body); err != nil {
					t.Error(err)
				}
			}))
			defer proxy.Close()
			defer manager.CloseAll()
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, node, filepath.Join(root, "mcp/browser-session-mcp.mjs"))
			cmd.Dir = root
			cmd.Env = append(os.Environ(), "UIAI_ENGINE_URL="+proxy.URL, "UIAI_MCP_TIMEOUT_MS=10000")
			input, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			output, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			defer func() {
				input.Close()
				if err := cmd.Wait(); err != nil && ctx.Err() == nil {
					t.Error(err)
				}
			}()
			encoder, decoder := json.NewEncoder(input), json.NewDecoder(output)
			call := func(id int, name string, args map[string]any) map[string]any {
				t.Helper()
				if err := encoder.Encode(map[string]any{"jsonrpc": "2.0", "id": id, "method": "tools/call", "params": map[string]any{"name": name, "arguments": args}}); err != nil {
					t.Fatal(err)
				}
				var reply struct {
					Result map[string]any `json:"result"`
				}
				if err := decoder.Decode(&reply); err != nil {
					t.Fatal(err)
				}
				return reply.Result
			}
			result := call(1, "browser_open", map[string]any{"url": site.URL + "/github/login", "width": 640, "height": 480})
			encoded, _ := json.Marshal(result)
			if strings.Contains(string(encoded), "synthetic-oauth-code") {
				t.Fatal("OAuth query value leaked through open response")
			}
			if mode == "malformed" {
				if result["isError"] != true || !strings.Contains(string(encoded), "session_cleanup=closed") {
					t.Fatal("malformed compressed response was not a cleaned-up structured failure")
				}
			} else {
				if result["isError"] == true {
					t.Fatalf("valid %s response failed: %s", mode, encoded)
				}
				content := result["content"].([]any)[0].(map[string]any)["text"].(string)
				var opened struct {
					Session struct {
						ID string `json:"id"`
					} `json:"session"`
				}
				if err := json.Unmarshal([]byte(content), &opened); err != nil {
					t.Fatal(err)
				}
				if opened.Session.ID == "" {
					t.Fatal("open succeeded without session identity")
				}
				if sessions := manager.List(); len(sessions) != 1 || sessions[0].LeaseState() != vision.LeaseAwaitingAuth {
					t.Fatal("fixture did not exercise awaiting-auth cleanup")
				}
				if closed := call(2, "browser_close", map[string]any{"session_id": opened.Session.ID}); closed["isError"] == true {
					t.Fatal("owned open cleanup failed")
				}
			}
			if len(manager.List()) != 0 {
				t.Fatal("open response failure leaked an engine session")
			}
			if fmt.Sprint(pool.Stats()["active_pages"]) != "0" {
				t.Fatal("open response failure leaked a pooled page")
			}
		})
	}
}
