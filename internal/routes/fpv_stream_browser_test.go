package routes

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image/jpeg"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
)

func TestFPVStreamTwoConsumersAndPaintLatency(t *testing.T) {
	if os.Getenv("UIAI_AUTH_TESTS") == "" {
		t.Skip("actual Chromium/HTTP stream consumer")
	}
	t.Setenv("FOCUSA_DAEMON_URL", "")
	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `<!doctype html><body style="margin:0;height:100vh;background:rgb(30,80,160)">`)
	}))
	defer site.Close()
	pool, err := vision.NewPoolWithConfig(vision.PoolConfig{MaxPages: 2, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	manager := vision.NewSessionManager(pool)
	defer manager.CloseAll()
	session, _, err := manager.Open(site.URL, 320, 240)
	if err != nil {
		t.Fatal(err)
	}
	token := fmt.Sprintf("synthetic-stream-%d", time.Now().UnixNano())
	fpvLoadRegistry()
	fpvShares.Store(token, &fpvShare{Token: token, SessionID: session.ID, ExpiresAt: time.Now().Add(time.Minute)})
	defer fpvShares.Delete(token)
	router := chi.NewRouter()
	router.Route("/m", func(r chi.Router) { MountFPVPublicRoutes(r, manager) })
	server := httptest.NewServer(router)
	defer func() { manager.CloseAll(); server.CloseClientConnections(); server.Close() }()
	client := &http.Client{Timeout: 15 * time.Second}
	open := func() (*http.Response, *multipart.Reader) {
		t.Helper()
		response, err := client.Get(server.URL + "/m/" + token + "/stream.cdp.mjpg?fps=8")
		if err != nil {
			t.Fatal("stream open failed")
		}
		if response.StatusCode != 200 {
			response.Body.Close()
			t.Fatalf("stream returned%d", response.StatusCode)
		}
		_, params, err := mime.ParseMediaType(response.Header.Get("Content-Type"))
		if err != nil {
			t.Fatal(err)
		}
		return response, multipart.NewReader(response.Body, params["boundary"])
	}
	first, reader := open()
	defer first.Body.Close()
	second, other := open()
	defer second.Body.Close()
	read := func(reader *multipart.Reader) []byte {
		t.Helper()
		part, err := reader.NextPart()
		if err != nil {
			t.Fatal("stream consumer stalled or disconnected")
		}
		defer part.Close()
		data, err := io.ReadAll(part)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := jpeg.Decode(bytes.NewReader(data)); err != nil {
			t.Fatal("invalid JPEG in multipart stream")
		}
		return data
	}
	read(reader)
	read(other)
	second.Body.Close() // The surviving client must keep receiving the same pump.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	timings := []int64{}
	var sample []byte
	for i := 0; i < 16; i++ {
		red, green := 40+(i%8)*24, 40+(i/8)*64
		started := time.Now()
		_, _, err := session.EvalContext(ctx, fmt.Sprintf(`document.body.style.backgroundColor="rgb(%d,%d,180)";true`, red, green))
		if err != nil {
			t.Fatal(err)
		}
		for {
			data := read(reader)
			frame, err := jpeg.Decode(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			r, g, b, _ := frame.At(100, 100).RGBA()
			abs := func(v int) int {
				if v < 0 {
					return -v
				}
				return v
			}
			if abs(int(r>>8)-red) <= 4 && abs(int(g>>8)-green) <= 4 && abs(int(b>>8)-180) <= 4 {
				sample = data
				break
			}
			if time.Since(started) > time.Second {
				t.Fatal("surviving viewer did not receive changed painted pixels")
			}
		}
		timings = append(timings, time.Since(started).Milliseconds())
	}
	first.Body.Close()
	deadline := time.After(2 * time.Second)
	for fpvViewerCount(token) != 0 {
		select {
		case <-deadline:
			t.Fatal("disconnect leaked viewer slots")
		case <-time.After(5 * time.Millisecond):
		}
	}
	sort.Slice(timings, func(i, j int) bool { return timings[i] < timings[j] })
	p95 := timings[(len(timings)*95-1)/100]
	result := map[string]any{"samples": len(timings), "p95_ms": p95, "fps": 8, "metric": "owned-target style mutation to received-and-decoded matching JPEG; not public-network glass-to-glass", "surviving_viewer_after_disconnect": true, "final_viewer_count": 0}
	visitor, _, err := manager.Open(server.URL+"/m/"+token, 1280, 800)
	if err != nil {
		t.Fatal(err)
	}
	uiCtx, uiCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer uiCancel()
	lastState := "not observed"
	for {
		state, _, err := visitor.EvalContext(uiCtx, `return JSON.stringify({label:document.getElementById('fps').textContent,width:document.getElementById('shot').naturalWidth,mode:document.getElementById('mode').textContent,hasSrc:!!document.getElementById('shot').getAttribute('src'),streams:performance.getEntriesByType('resource').filter(x=>x.name.includes('stream.cdp')).length,assets:performance.getEntriesByType('resource').filter(x=>x.name.includes('/assets/')).length});`)
		if err != nil {
			t.Fatalf("viewer UI did not become readable: %s", lastState)
		}
		lastState = state
		var status struct {
			Label string `json:"label"`
			Width int    `json:"width"`
		}
		if json.Unmarshal([]byte(state), &status) != nil {
			t.Fatalf("viewer state format: %s", state)
		}
		if status.Width > 0 && status.Label == "Live" {
			break
		}
		select {
		case <-uiCtx.Done():
			t.Fatalf("viewer failed to display live frames with truthful cadence: %s", state)
		case <-time.After(50 * time.Millisecond):
		}
	}
	result["actual_browser_viewer_live"] = true
	paintCtx, paintCancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer paintCancel()
	_, _, err = visitor.EvalContext(paintCtx, `document.getElementById("qSmooth").click();window.__fpvPaints={};window.__fpvCanvas=document.createElement("canvas");window.__fpvCanvas.width=320;window.__fpvCanvas.height=240;(function poll(){const image=document.getElementById("shot");if(image.naturalWidth){const ctx=window.__fpvCanvas.getContext("2d");ctx.drawImage(image,0,0);const p=ctx.getImageData(100,100,1,1).data;const key=[p[0],p[1],p[2]].join(",");if(!window.__fpvPaints[key])window.__fpvPaints[key]=Date.now();}requestAnimationFrame(poll)})();return true;`)
	if err != nil {
		t.Fatal(err)
	}
	browserTimings := []int64{}
	for i := 0; i < 8; i++ {
		red := 30 + i*26
		started := time.Now().UnixMilli()
		_, _, err = session.EvalContext(paintCtx, fmt.Sprintf(`document.body.style.backgroundColor="rgb(%d,190,50)";return true;`, red))
		if err != nil {
			t.Fatal(err)
		}
		observed, _, err := visitor.EvalAsyncContext(paintCtx, fmt.Sprintf(`const end=Date.now()+700;while(Date.now()<end){const match=Object.entries(window.__fpvPaints).find(([key,time])=>{const p=key.split(",").map(Number);return Math.abs(p[0]-%d)<=4&&Math.abs(p[1]-190)<=4&&Math.abs(p[2]-50)<=4});if(match)return String(match[1]);await new Promise(r=>setTimeout(r,16));}return "0";`, red), 1000)
		if err != nil {
			t.Fatal(err)
		}
		var stamp int64
		if json.Unmarshal([]byte(observed), &stamp) != nil || stamp < started {
			t.Fatal("actual viewer did not observe changed pixels")
		}
		browserTimings = append(browserTimings, stamp-started+16)
	}
	sort.Slice(browserTimings, func(i, j int) bool { return browserTimings[i] < browserTimings[j] })
	result["actual_browser_paint_p95_ms"] = browserTimings[len(browserTimings)-1]
	result["actual_browser_paint_metric"] = "same-host target mutation to viewer canvas observing decoded pixels on animation frame, plus16ms paint allowance; not a public-network guarantee"

	if os.Getenv("UIAI_STREAM_PERF_TESTS") != "" {
		snapshot, err := visitor.Screenshot("jpeg", 75)
		if err != nil {
			t.Fatal(err)
		}
		pixels, err := base64.StdEncoding.DecodeString(snapshot.Screenshot)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile("/tmp/uiai-fpv-viewer-ui.jpg", pixels, 0600); err != nil {
			t.Fatal(err)
		}
		if browserTimings[len(browserTimings)-1] > 300 {
			t.Fatalf("actual browser smooth paint p95=%dms exceeds local300ms target", browserTimings[len(browserTimings)-1])
		}
		if p95 > 300 {
			t.Fatalf("local decoded-frame p95=%dms exceeds300ms target", p95)
		}

		if err := manager.Close(visitor.ID); err != nil {
			t.Fatal(err)
		}
		if err := manager.Close(session.ID); err != nil {
			t.Fatal(err)
		}
		until := time.Now().Add(2 * time.Second)
		for fpvViewerCount(token) != 0 {
			if time.Now().After(until) {
				t.Fatal("actual viewer close leaked stream slots")
			}
			time.Sleep(5 * time.Millisecond)
		}
		result["final_viewer_count"] = fpvViewerCount(token)
		data, _ := json.MarshalIndent(result, "", "  ")
		if err := os.WriteFile("/tmp/uiai-fpv-stream-latency.json", data, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile("/tmp/uiai-fpv-stream-fixture.jpg", sample, 0600); err != nil {
			t.Fatal(err)
		}
	}
}
