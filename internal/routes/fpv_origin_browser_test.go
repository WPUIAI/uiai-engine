package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func TestFPVExistingShareRejectsCrossOriginNavigationBrowser(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("actual Chromium origin boundary")
	}
	t.Chdir(t.TempDir())
	fixture := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<!doctype html><title>Owned fixture</title><body>Ordinary public page</body>")
	})
	original := httptest.NewServer(fixture)
	defer original.Close()
	destination := httptest.NewServer(fixture)
	defer destination.Close()
	pool, err := vision.NewPoolWithConfig(vision.PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := vision.NewSessionManager(pool)
	defer manager.CloseAll()
	router := chi.NewRouter()
	router.Route("/m", func(r chi.Router) { MountFPVPublicRoutes(r, manager) })
	t.Run("unbound legacy share", func(t *testing.T) {
		session, _, err := manager.Open(original.URL, 320, 240)
		if err != nil {
			t.Fatal(err)
		}
		defer manager.Close(session.ID)
		token := "synthetic-unbound-legacy-origin"
		fpvShares.Store(token, &fpvShare{Token: token, SessionID: session.ID, ExpiresAt: time.Now().Add(time.Minute)})
		defer fpvShares.Delete(token)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, httptest.NewRequest("GET", "/m/"+token+"/status", nil))
		if response.Code != http.StatusForbidden {
			t.Fatalf("unbound legacy share status=%d", response.Code)
		}
		if _, available := fpvEntry(token); available {
			t.Fatal("unbound legacy share was not permanently revoked")
		}
	})
	for _, path := range []string{"", "/status", "/events", "/screenshot.jpg", "/stream.cdp.mjpg", "/stream.mjpg"} {
		t.Run(path, func(t *testing.T) {
			session, _, err := manager.Open(original.URL, 320, 240)
			if err != nil {
				t.Fatal(err)
			}
			defer manager.Close(session.ID)
			token := fmt.Sprintf("synthetic-origin-transition-%d", time.Now().UnixNano())
			// Decode the persisted-share boundary, including the intended origin binding.
			raw, err := json.Marshal(map[string]any{"token": token, "session_id": session.ID, "origin": original.URL, "expires_at": time.Now().Add(time.Minute)})
			if err != nil {
				t.Fatal(err)
			}
			var entry fpvShare
			if err := json.Unmarshal(raw, &entry); err != nil {
				t.Fatal(err)
			}
			fpvShares.Store(token, &entry)
			defer fpvShares.Delete(token)
			before := httptest.NewRecorder()
			router.ServeHTTP(before, httptest.NewRequest("GET", "/m/"+token+"/status", nil))
			if before.Code != http.StatusOK {
				t.Fatalf("original origin status=%d", before.Code)
			}
			if _, err := session.Navigate(original.URL + "/same-origin"); err != nil {
				t.Fatal(err)
			}
			same := httptest.NewRecorder()
			router.ServeHTTP(same, httptest.NewRequest("GET", "/m/"+token+"/status", nil))
			if same.Code != http.StatusOK {
				t.Fatalf("same-origin navigation status=%d", same.Code)
			}
			if _, err := session.Navigate(destination.URL); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest("GET", "/m/"+token+path, nil).WithContext(ctx))
			if response.Code != http.StatusForbidden && response.Code != http.StatusGone {
				t.Fatalf("public share survived cross-origin navigation: status=%d", response.Code)
			}
			if _, err := session.Navigate(original.URL); err != nil {
				t.Fatal(err)
			}
			if _, available := fpvEntry(token); available {
				t.Fatal("returning to original origin resurrected revoked share")
			}
		})
	}
}
