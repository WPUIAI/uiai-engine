package routes

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestFPVStreamFirstFrameFailureIsTypedAndBounded(t *testing.T) {
	frames := make(chan []byte)
	response := httptest.NewRecorder()
	start := time.Now()
	fpvWriteStream(response, httptest.NewRequest("GET", "/stream", nil), frames, 8)
	if response.Code != http.StatusGatewayTimeout || !strings.Contains(response.Body.String(), "fpv_stream_stalled") || time.Since(start) > 4*time.Second {
		t.Fatal("initial silence became hanging or false-success stream")
	}
	close(frames)
	response = httptest.NewRecorder()
	fpvWriteStream(response, httptest.NewRequest("GET", "/stream", nil), frames, 8)
	if response.Code != http.StatusServiceUnavailable {
		t.Fatal("closed pump reported success")
	}
}

type stalledFPVWriter struct {
	header   http.Header
	deadline time.Time
}

func (w *stalledFPVWriter) Header() http.Header { return w.header }
func (w *stalledFPVWriter) WriteHeader(int)     {}
func (w *stalledFPVWriter) SetWriteDeadline(deadline time.Time) error {
	w.deadline = deadline
	return nil
}
func (w *stalledFPVWriter) FlushError() error { return nil }
func (w *stalledFPVWriter) Write([]byte) (int, error) {
	if w.deadline.IsZero() {
		return 0, errors.New("missing write deadline")
	}
	<-time.After(time.Until(w.deadline))
	return 0, os.ErrDeadlineExceeded
}
func TestFPVStreamDeadReaderCannotHoldWriter(t *testing.T) {
	frames := make(chan []byte, 1)
	frames <- []byte("owned-synthetic-jpeg-transport-fixture")
	writer := &stalledFPVWriter{header: make(http.Header)}
	start := time.Now()
	fpvWriteStream(writer, httptest.NewRequest("GET", "/", nil), frames, 8)
	if time.Since(start) > 3*time.Second {
		t.Fatal("dead reader retained stream writer")
	}
}
func TestFPVStreamFPSAndDisconnect(t *testing.T) {
	for _, raw := range []string{"0", "31", "nope"} {
		if _, err := fpvRequestedFPS(httptest.NewRequest("GET", "/?fps="+raw, nil)); err == nil {
			t.Fatal("unbounded frame rate admitted")
		}
	}
	if fps, err := fpvRequestedFPS(httptest.NewRequest("GET", "/?fps=8", nil)); err != nil || fps != 8 {
		t.Fatal("supported fps rejected")
	}
	frames := make(chan []byte, 1)
	frames <- []byte("synthetic-transport-only")
	done := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { defer close(done); fpvWriteStream(w, r, frames, 8) }))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, "GET", server.URL, nil)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != 200 || response.Header.Get("X-UIAI-Stream-FPS") != "8" {
		t.Fatal("stream frame-rate contract missing")
	}
	data := make([]byte, 100)
	_, _ = response.Body.Read(data)
	response.Body.Close()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("disconnected reader retained handler")
	}
}

var _ io.Writer = (*stalledFPVWriter)(nil)
