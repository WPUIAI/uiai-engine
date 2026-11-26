package vision

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// #294 understand slice: one call returns frames plus the transcript of the
// same media. The fixture is a real VP8 video carrying synthesized speech
// over color scenes; the test skips when the fixture tooling or provider is
// unavailable.
func TestMediaUnderstandFramesPlusTranscript(t *testing.T) {
	if os.Getenv("UIAI_INPUT_TESTS") == "" {
		t.Skip("set UIAI_INPUT_TESTS=1 for browser understand regression")
	}
	espeak, err := exec.LookPath("espeak-ng")
	if err != nil {
		t.Skip("espeak-ng not available for the speech fixture")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available for the fixture")
	}
	whisperURL := os.Getenv("UIAI_TEST_WHISPER_URL")
	if whisperURL == "" {
		whisperURL = "http://localhost:8115"
	}
	if !whisperReachable(whisperURL) {
		t.Skipf("whisper provider not reachable at %s", whisperURL)
	}

	dir := t.TempDir()
	wav := filepath.Join(dir, "speech.wav")
	video := filepath.Join(dir, "scene.webm")
	if out, err := exec.Command(espeak, "-v", "en-us", "-s", "140", "-w", wav,
		"The invoice total is forty two dollars.").CombinedOutput(); err != nil {
		t.Skipf("speech synthesis failed: %v: %s", err, string(out)[:120])
	}
	// Three color scenes with the speech track muxed in: every frame differs
	// and the audio carries the fixture phrase.
	script := "color=red:duration=1:size=320x240:rate=10[red];" +
		"color=lime:duration=1:size=320x240:rate=10[green];" +
		"color=blue:duration=1:size=320x240:rate=10[blue];" +
		"[red][green][blue]concat=n=3:v=1:a=0[v]"
	if out, err := exec.Command("ffmpeg", "-filter_complex", script,
		"-i", wav, "-map", "[v]", "-map", "0:a", "-pix_fmt", "yuv420p", "-c:v", "libvpx", "-b:v", "300k",
		"-c:a", "libopus", "-b:a", "48k", "-shortest", "-y", video).CombinedOutput(); err != nil {
		t.Skipf("fixture mux failed: %v: %s", err, string(out)[:160])
	}

	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/" {
			fmt.Fprintf(w, `<!doctype html><html><body>
			<video id="v" src="/scene.webm" preload="auto" width="320" height="240"></video>
			<audio id="a" src="/scene.webm" controls></audio>
			</body></html>`)
			return
		}
		w.Header().Set("Content-Type", "video/webm")
		http.ServeFile(w, req, video)
	}))
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

	// Video element: frames plus transcript in one call.
	result, err := session.MediaUnderstand("#v", "", whisperURL, 3)
	if err != nil {
		t.Fatalf("understand: %v", err)
	}
	if len(result.Frames) != 3 {
		t.Fatalf("expected 3 frames, got %d (notes: %v)", len(result.Frames), result.Notes)
	}
	seen := map[string]bool{}
	for i, f := range result.Frames {
		if seen[f.Hash] && i > 0 {
			t.Fatalf("frame %d duplicated", i)
		}
		seen[f.Hash] = true
	}
	if result.Transcript == nil || !speechSurvived(result.Transcript.Text) {
		t.Fatalf("understand lost the transcript: %+v", result.Transcript)
	}

	// Audio-only element: transcript with the honest frames note.
	result, err = session.MediaUnderstand("#a", "", whisperURL, 3)
	if err != nil {
		t.Fatalf("understand audio-only: %v", err)
	}
	if result.Transcript == nil || !speechSurvived(result.Transcript.Text) {
		t.Fatalf("audio-only understand lost the transcript: %+v", result.Transcript)
	}
	if len(result.Frames) != 0 || !strings.Contains(strings.Join(result.Notes, ";"), "audio-only") {
		t.Fatalf("audio-only understand must carry the frames note: %+v", result)
	}

	// Direct URL: transcript with the honest framing note.
	result, err = session.MediaUnderstand("", site.URL+"/scene.webm", whisperURL, 3)
	if err != nil {
		t.Fatalf("understand url: %v", err)
	}
	if result.Transcript == nil || !speechSurvived(result.Transcript.Text) {
		t.Fatalf("url understand lost the transcript: %+v", result.Transcript)
	}
	if !strings.Contains(strings.Join(result.Notes, ";"), "cannot be framed") {
		t.Fatalf("url understand must carry the framing note: %+v", result.Notes)
	}
}
