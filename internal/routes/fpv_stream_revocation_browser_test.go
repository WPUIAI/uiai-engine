package routes

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func TestOpenFPVCDPStreamStopsAfterRevocationOrExpiryBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("set UIAI_AUTH_TESTS=1")
	}
	t.Chdir(t.TempDir())
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<!doctype html><title>Owned revocation fixture</title><body>public fixture</body>`)
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
	router.Route("/m", func(r chi.Router) { MountFPVPublicRoutes(r, manager) })
	server := httptest.NewServer(router)
	defer server.Close()
	for _, cause := range []string{"revoked", "expired", "closed", "authentication"} {
		t.Run(cause, func(t *testing.T) {
			session, _, err := manager.Open(site.URL, 640, 480)
			if err != nil {
				t.Fatal(err)
			}
			defer manager.Close(session.ID)
			token := fmt.Sprintf("synthetic-owned-stop-%s-%d", cause, time.Now().UnixNano())
			fpvShares.Store(token, &fpvShare{Token: token, SessionID: session.ID, ExpiresAt: time.Now().Add(time.Minute)})
			defer fpvShares.Delete(token)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			req, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/m/"+token+"/stream.cdp.mjpg?fps=30", nil)
			response, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal("owned stream did not open")
			}
			defer response.Body.Close()
			if response.StatusCode != 200 {
				t.Fatal("owned stream did not start successfully")
			}
			part, err := multipart.NewReader(response.Body, "frame").NextPart()
			if err != nil {
				t.Fatal("no first frame")
			}
			data, err := io.ReadAll(part)
			if err != nil || len(data) == 0 {
				t.Fatal("no actual first frame bytes")
			}
			// A second valid share must retain its independent subscription when only
			// the first share is revoked or expires.
			var otherBody io.ReadCloser
			var otherReader *multipart.Reader
			if cause != "closed" && cause != "authentication" {
				otherToken := token + "-other"
				fpvShares.Store(otherToken, &fpvShare{Token: otherToken, SessionID: session.ID, ExpiresAt: time.Now().Add(time.Minute)})
				defer fpvShares.Delete(otherToken)
				otherReq, _ := http.NewRequestWithContext(ctx, "GET", server.URL+"/m/"+otherToken+"/stream.cdp.mjpg?fps=30", nil)
				otherResponse, openErr := http.DefaultClient.Do(otherReq)
				if openErr != nil || otherResponse.StatusCode != 200 {
					t.Fatal("second valid viewer could not join")
				}
				otherBody = otherResponse.Body
				defer otherBody.Close()
				otherReader = multipart.NewReader(otherBody, "frame")
				otherPart, readErr := otherReader.NextPart()
				if readErr != nil {
					t.Fatal("second viewer did not receive first frame")
				}
				if bytes, readErr := io.ReadAll(otherPart); readErr != nil || len(bytes) == 0 {
					t.Fatal("second viewer had no initial bytes")
				}
			}
			switch cause {
			case "revoked":
				_, _ = fpvUpdateShare(token, func(entry *fpvShare) bool { entry.Revoked = true; return true })
			case "expired":
				_, _ = fpvUpdateShare(token, func(entry *fpvShare) bool { entry.ExpiresAt = time.Now().Add(-time.Second); return true })
			case "closed":
				manager.Close(session.ID)
			case "authentication":
				if _, _, err := session.Eval(`document.body.innerHTML='<input type="password">'; return true;`); err != nil {
					t.Fatal(err)
				}
				if challenge, err := session.DetectAuthChallenge(); err != nil || challenge == nil {
					t.Fatal("owned authentication transition was not recognized")
				}
			}
			start := time.Now()
			_, copyErr := io.Copy(io.Discard, response.Body)
			if copyErr != nil || time.Since(start) > time.Second {
				t.Fatal("open CDP stream retained output after its share/session became invalid")
			}
			if cause == "authentication" {
				stored, ok := fpvShares.Load(token)
				if !ok || !stored.(*fpvShare).Revoked {
					t.Fatal("authentication did not revoke the streaming capability")
				}
			}
			if otherReader != nil {
				nextPart, readErr := otherReader.NextPart()
				if readErr != nil {
					t.Fatal("revoking one share stopped another valid viewer")
				}
				if bytes, readErr := io.ReadAll(nextPart); readErr != nil || len(bytes) == 0 {
					t.Fatal("surviving viewer lost its frames")
				}
			}
		})
	}
}
