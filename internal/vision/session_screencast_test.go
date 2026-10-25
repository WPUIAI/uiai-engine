package vision

import (
	"bytes"
	"context"
	"fmt"
	"image/jpeg"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestScreencastSharedPumpSurvivesViewerDisconnect(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("actual Chromium screencast fixture")
	}
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<!doctype html><body style="margin:0;background:rgb(30,80,160)"><script>let n=0;setInterval(()=>{document.body.style.backgroundColor=++n%2?'rgb(30,80,160)':'rgb(50,140,90)'},80)</script>`)
	}))
	defer site.Close()
	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := NewSessionManager(pool)
	defer manager.CloseAll()
	session, _, err := manager.Open(site.URL, 320, 240)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	one, stopOne, err := session.CDPScreencast(ctx, 60, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer stopOne()
	read := func(frames <-chan []byte) {
		t.Helper()
		select {
		case data, ok := <-frames:
			if !ok {
				t.Fatal("viewer channel closed unexpectedly")
			}
			frame, err := jpeg.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatal("stream did not yield a JPEG")
			}
			r, g, b, _ := frame.At(100, 100).RGBA()
			if r+g+b == 0 {
				t.Fatal("stream yielded a black frame")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("screencast pump stalled")
		}
	}
	read(one)
	two, stopTwo, err := session.CDPScreencast(ctx, 60, 1)
	if err != nil {
		t.Fatal(err)
	}
	defer stopTwo()
	read(two)
	stopOne()
	stopOne()
	for i := 0; i < 4; i++ {
		read(two)
	}
	// Slow readers retain only the latest frame, not seconds of queued history.
	if len(two) > 1 {
		t.Fatal("slow viewer accumulated stale frame backlog")
	}
	if err := manager.Close(session.ID); err != nil {
		t.Fatal(err)
	}
	select {
	case _, ok := <-two:
		if ok {
			t.Fatal("closed session retained viewer frames")
		}
	case <-time.After(time.Second):
		t.Fatal("session close did not terminate screencast")
	}
	stopTwo()
	if _, _, err := session.CDPScreencast(ctx, 60, 1); err == nil {
		t.Fatal("closed session admitted new pump")
	}
}
