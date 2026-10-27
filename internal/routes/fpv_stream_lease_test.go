package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestFPVStreamLeaseFailsClosedBeforeFirstFrame(t *testing.T) {
	for _, check := range []func() bool{nil, func() bool { return false }} {
		response := httptest.NewRecorder()
		fpvWriteStream(response, httptest.NewRequest("GET", "/stream", nil), make(chan []byte), 8, check)
		if response.Code != http.StatusGone || !strings.Contains(response.Body.String(), "fpv_stream_lease_ended") {
			t.Fatal("missing or ended lease reported stream success")
		}
	}
}

func TestFPVStreamLeaseEndingDuringInitialSilenceStopsPromptly(t *testing.T) {
	var valid atomic.Bool
	valid.Store(true)
	response := httptest.NewRecorder()
	done := make(chan struct{})
	admitted := make(chan struct{})
	var once sync.Once
	check := func() bool { current := valid.Load(); once.Do(func() { close(admitted) }); return current }
	go func() {
		defer close(done)
		fpvWriteStream(response, httptest.NewRequest("GET", "/stream", nil), make(chan []byte), 8, check)
	}()
	<-admitted
	valid.Store(false)
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("invalid lease retained first-frame wait")
	}
	if response.Code != http.StatusGone || !strings.Contains(response.Body.String(), "fpv_stream_lease_ended") {
		t.Fatal("initial lease loss was not typed")
	}
}
