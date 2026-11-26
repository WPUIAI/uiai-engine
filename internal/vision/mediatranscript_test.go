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
	"time"
)

// #294 hear slice: a page's media element (or a direct media URL) transcribes
// through the configured Whisper endpoint. The fixture is real speech
// synthesized with espeak-ng; the test skips when ffmpeg, espeak-ng, or the
// whisper provider is unavailable.
func TestMediaTranscriptFromElementAndURL(t *testing.T) {
	if os.Getenv("UIAI_INPUT_TESTS") == "" {
		t.Skip("set UIAI_INPUT_TESTS=1 for browser transcript regression")
	}
	espeak, err := exec.LookPath("espeak-ng")
	if err != nil {
		t.Skip("espeak-ng not available for the speech fixture")
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available for the audio extraction")
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
	mp3 := filepath.Join(dir, "speech.mp3")
	if out, err := exec.Command(espeak, "-v", "en-us", "-s", "140", "-w", wav,
		"The invoice total is forty two dollars.").CombinedOutput(); err != nil {
		t.Skipf("speech synthesis failed: %v: %s", err, string(out)[:120])
	}
	if out, err := exec.Command("ffmpeg", "-i", wav, "-ar", "16000", "-ac", "1",
		"-b:a", "64k", "-f", "mp3", "-y", mp3).CombinedOutput(); err != nil {
		t.Skipf("mp3 conversion failed: %v: %s", err, string(out)[:120])
	}

	site := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path == "/" {
			fmt.Fprintf(w, `<!doctype html><html><body><audio id="a" src="/speech.mp3" controls></audio></body></html>`)
			return
		}
		w.Header().Set("Content-Type", "audio/mpeg")
		http.ServeFile(w, req, mp3)
	}))
	defer site.Close()

	pool, err := NewPoolWithConfig(PoolConfig{MaxPages: 1, AllowPrivateURLs: true})
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	sm := NewSessionManager(pool)
	defer sm.CloseAll()

	session, _, err := sm.Open(site.URL, 640, 480)
	if err != nil {
		t.Fatalf("open: %v", err)
	}

	// Selector form: the page's audio element.
	result, err := session.MediaTranscript("#a", "", whisperURL)
	if err != nil {
		t.Fatalf("transcript by selector: %v", err)
	}
	if result.Source != "selector" {
		t.Fatalf("expected selector source, got %q", result.Source)
	}
	if !speechSurvived(result.Text) {
		t.Fatalf("transcript lost the fixture speech: %q", result.Text)
	}
	if result.DurationS < 1.0 {
		t.Fatalf("transcript duration suspiciously short: %.2f", result.DurationS)
	}

	// Direct URL form.
	result, err = session.MediaTranscript("", site.URL+"/speech.mp3", whisperURL)
	if err != nil {
		t.Fatalf("transcript by url: %v", err)
	}
	if result.Source != "url" {
		t.Fatalf("expected url source, got %q", result.Source)
	}
	if !speechSurvived(result.Text) {
		t.Fatalf("url transcript lost the fixture speech: %q", result.Text)
	}

	// Missing provider is refused with the typed configuration error.
	if _, err := session.MediaTranscript("#a", "", ""); err == nil {
		t.Fatal("transcription without a whisper endpoint must be refused")
	}
	// Unknown selectors are refused.
	if _, err := session.MediaTranscript("#missing", "", whisperURL); err == nil {
		t.Fatal("transcription of a missing element must be refused")
	}
}

// speechSurvived accepts the fixture phrase in either spelled-out or
// digit-normalized form — whisper's small model transcribes "forty two"
// faithfully but emits digits in its text.
func speechSurvived(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "forty") || strings.Contains(lower, "42") ||
		strings.Contains(lower, "invoice")
}

func whisperReachable(whisperURL string) bool {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(strings.TrimSuffix(whisperURL, "/") + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK
}
