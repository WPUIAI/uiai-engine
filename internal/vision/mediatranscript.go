package vision

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// #294 hear slice: a page's media element (or a direct media URL) can be
// transcribed through the configured Whisper endpoint, so agents can consume
// audio and video content without leaving the session. The provider is
// reused from the CAPTCHA audio path — no new secret surface.

const (
	maxTranscriptMediaBytes   = 50 << 20
	maxTranscriptSeconds      = 900
	transcriptDownloadTimeout = 60 * time.Second
	transcriptWhisperTimeout  = 120 * time.Second
)

// MediaTranscriptResult is one bounded transcription outcome.
type MediaTranscriptResult struct {
	Text       string  `json:"text"`
	Language   string  `json:"language"`
	DurationS  float64 `json:"duration_s"`
	LatencyMs  int64   `json:"latency_ms"`
	Source     string  `json:"source"`
	MediaBytes int64   `json:"media_bytes"`
}

// MediaTranscript resolves the media from a page element (selector) or a
// direct URL, downloads it bounded, extracts the audio track with ffmpeg,
// and transcribes it through the configured Whisper endpoint.
func (s *Session) MediaTranscript(selector, mediaURL, whisperURL string) (*MediaTranscriptResult, error) {
	if whisperURL == "" {
		return nil, fmt.Errorf("no transcript provider configured: set captcha.recaptcha.whisper_url")
	}
	source := "url"
	if selector != "" {
		s.mu.Lock()
		page := s.page
		s.mu.Unlock()
		if page == nil {
			return nil, fmt.Errorf("session closed")
		}
		element, err := actionElement(page, selector)
		if err != nil {
			return nil, fmt.Errorf("media element not found: %w", err)
		}
		probe, perr := element.Eval(`() => (this.currentSrc || this.src || this.getAttribute('src') || '')`)
		if perr != nil {
			return nil, fmt.Errorf("media probe failed: %w", perr)
		}
		mediaURL = probe.Value.Str()
		source = "selector"
	}
	if mediaURL == "" {
		return nil, fmt.Errorf("no media URL resolved from selector %q", selector)
	}
	parsed, err := url.Parse(mediaURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, fmt.Errorf("media URL is not http(s): %q", truncateFor(mediaURL, 80))
	}

	s.mu.Lock()
	dir, derr := s.sessionFilesDirLocked()
	s.mu.Unlock()
	if derr != nil {
		return nil, derr
	}
	input := filepath.Join(dir, "media-transcript-input.bin")
	audio := filepath.Join(dir, "media-transcript-audio.mp3")
	defer func() {
		_ = os.Remove(input)
		_ = os.Remove(audio)
	}()

	mediaBytes, err := downloadBounded(mediaURL, input, maxTranscriptMediaBytes)
	if err != nil {
		return nil, fmt.Errorf("media download failed: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), transcriptDownloadTimeout)
	defer cancel()
	extract := exec.CommandContext(ctx, "ffmpeg", "-i", input, "-vn", "-ac", "1", "-ar", "16000",
		"-t", fmt.Sprint(maxTranscriptSeconds), "-b:a", "64k", "-f", "mp3", "-y", audio)
	var extractErr bytes.Buffer
	extract.Stderr = &extractErr
	if err := extract.Run(); err != nil {
		return nil, fmt.Errorf("audio extraction failed (ffmpeg): %w", err)
	}
	audioInfo, err := os.Stat(audio)
	if err != nil || audioInfo.Size() == 0 {
		return nil, fmt.Errorf("audio extraction produced no audio track")
	}

	transcript, err := whisperTranscribe(audio, whisperURL)
	if err != nil {
		return nil, err
	}
	return &MediaTranscriptResult{
		Text:       transcript.Text,
		Language:   transcript.Language,
		DurationS:  transcript.Duration,
		LatencyMs:  transcript.LatencyMs,
		Source:     source,
		MediaBytes: mediaBytes,
	}, nil
}

type whisperResponse struct {
	Text      string  `json:"text"`
	Language  string  `json:"language"`
	Duration  float64 `json:"duration"`
	LatencyMs int64   `json:"latency_ms"`
}

// whisperTranscribe POSTs the extracted audio to the configured Whisper
// endpoint using the same audio_file multipart contract the CAPTCHA solver
// documented.
func whisperTranscribe(audioPath, whisperURL string) (*whisperResponse, error) {
	endpoint := strings.TrimSuffix(whisperURL, "/") + "/transcribe"
	file, err := os.Open(audioPath)
	if err != nil {
		return nil, fmt.Errorf("audio open failed: %w", err)
	}
	defer file.Close()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("audio_file", "audio.mp3")
	if err != nil {
		return nil, fmt.Errorf("transcript request build failed: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("transcript request build failed: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("transcript request build failed: %w", err)
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("transcript request build failed: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	client := &http.Client{Timeout: transcriptWhisperTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("transcript provider unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("transcript provider returned %s", resp.Status)
	}
	var out whisperResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(&out); err != nil {
		return nil, fmt.Errorf("transcript provider response unreadable: %w", err)
	}
	return &out, nil
}

// downloadBounded fetches a media URL into path with a hard byte bound.
func downloadBounded(mediaURL, path string, limit int64) (int64, error) {
	client := &http.Client{Timeout: transcriptDownloadTimeout}
	resp, err := client.Get(mediaURL)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("media source returned %s", resp.Status)
	}
	if resp.ContentLength > limit {
		return 0, fmt.Errorf("media exceeds the %d byte bound", limit)
	}
	out, err := os.Create(path)
	if err != nil {
		return 0, err
	}
	defer out.Close()
	written, err := io.Copy(out, io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return 0, err
	}
	if written > limit {
		return 0, fmt.Errorf("media exceeds the %d byte bound", limit)
	}
	return written, nil
}

// truncateFor bounds error text carrying caller-supplied URLs.
func truncateFor(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// truncateFor bounds error text carrying caller-supplied URLs.
