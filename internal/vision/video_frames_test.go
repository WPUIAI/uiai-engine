package vision

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// #294 watch slice: bounded frame sampling from a session <video> element.
// The fixture is a real seekable MP4 generated with ffmpeg (distinct color
// per second, so sampled frames must differ); the test skips when ffmpeg
// is unavailable.
func TestSampleVideoFrames(t *testing.T) {
	if os.Getenv("UIAI_INPUT_TESTS") == "" {
		t.Skip("set UIAI_INPUT_TESTS=1 for browser video regression")
	}
	ffmpeg, err := exec.LookPath("ffmpeg")
	if err != nil {
		t.Skip("ffmpeg not available for the video fixture")
	}
	dir := t.TempDir()
	video := filepath.Join(dir, "scenes.webm")
	// Three one-second solid scenes (red/lime/blue) concatenated to a 3s
	// clip: sampled frames must differ. The double-source chained form
	// silently truncates some encoders to the first scene, so each source
	// is declared exactly once.
	script := "color=red:duration=1:size=320x240:rate=10[red];" +
		"color=lime:duration=1:size=320x240:rate=10[green];" +
		"color=blue:duration=1:size=320x240:rate=10[blue];" +
		"[red][green][blue]concat=n=3:v=1:a=0[out]"
	// VP8/WebM: rod's bundled Chromium lacks proprietary H.264 codecs, so the
	// fixture must decode on every engine browser, not just system Chrome.
	cmd := exec.Command(ffmpeg, "-filter_complex", script, "-map", "[out]", "-pix_fmt", "yuv420p", "-c:v", "libvpx", "-b:v", "300k", "-y", video)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("ffmpeg fixture generation failed: %v: %s", err, string(out)[:200])
	}

	videoHandler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/" {
			fmt.Fprintf(w, `<!doctype html><html><body><video id="v" src="/scenes.webm" preload="auto" width="320" height="240"></video></body></html>`)
			return
		}
		w.Header().Set("Content-Type", "video/webm")
		http.ServeFile(w, req, video)
	})
	site := httptest.NewServer(videoHandler)
	defer site.Close()

	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	sm := NewSessionManager(pool)
	defer sm.CloseAll()

	session, _, err := sm.Open(site.URL, 400, 320)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	// Wait for the video to load enough to seek. The wait resolves with a
	// diagnostic string instead of throwing so the engine's eval error
	// classification cannot swallow the real media state.
	wstate, _, werr := session.EvalAsync(`return await new Promise((resolve) => {
			const v = document.getElementById('v');
			if (v.readyState >= 2) { resolve('ready'); return; }
			v.addEventListener('loadeddata', () => resolve('ready'), { once: true });
			v.addEventListener('error', () => resolve('err:' + (v.error ? v.error.code : 'none') + ':net=' + v.networkState), { once: true });
			try { v.load(); } catch (e) { resolve('loaderr'); }
			setTimeout(() => resolve('timeout:rs=' + v.readyState), 8000);
		});`, 9000)
	if werr != nil {
		t.Fatalf("video load eval: %v", werr)
	}
	if wstate[:5] != "ready" {
		t.Fatalf("video never reached a loaded state: %q", wstate)
	}

	frames, truncated, err := session.SampleVideoFrames("#v", 3, 0, 0)
	if err != nil {
		t.Fatalf("sample: %v", err)
	}
	if len(frames) != 3 {
		t.Fatalf("expected 3 frames, got %d", len(frames))
	}
	if truncated {
		t.Fatal("short fixture must not hit the byte bound")
	}
	seen := map[string]bool{}
	for i, f := range frames {
		if f.Bytes < 500 {
			t.Fatalf("frame %d suspiciously small: %d bytes", i, f.Bytes)
		}
		sum := sha256.Sum256([]byte(f.JPEGB64))
		key := hex.EncodeToString(sum[:8])
		if seen[key] && i > 0 {
			// The color scenes must make distinct frames.
			for j, f := range frames {
				t.Logf("frame %d: ts=%dms bytes=%d hash=%s", j, f.TimestampMs, f.Bytes, f.Hash)
			}
			t.Fatalf("frames %d duplicated", i)
		}
		seen[key] = true
	}

	// Non-video selectors and empty ranges fail closed.
	if _, _, err := session.SampleVideoFrames("body", 1, 0, 0); err == nil {
		t.Fatal("sampling a non-video element must be refused")
	}
	if _, _, err := session.SampleVideoFrames("#v", 1, 5000, 1000); err == nil {
		t.Fatal("inverted sample range must be refused")
	}
}
