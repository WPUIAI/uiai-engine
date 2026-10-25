package routes

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

const fpvFirstFrameTimeout = 3 * time.Second
const fpvStreamWriteTimeout = 2 * time.Second

func fpvRequestedFPS(r *http.Request) (int, error) {
	raw := r.URL.Query().Get("fps")
	if raw == "" {
		return 8, nil
	}
	fps, err := strconv.Atoi(raw)
	if err != nil || fps < 1 || fps > 30 {
		return 0, fmt.Errorf("fps must be between1 and30")
	}
	return fps, nil
}

// Delay HTTP success until the first actual JPEG. Bound both initial silence and
// socket writes; a dead reader cannot retain a viewer slot indefinitely.
func fpvWriteStream(w http.ResponseWriter, r *http.Request, frames <-chan []byte, fps int) {
	timer := time.NewTimer(fpvFirstFrameTimeout)
	defer timer.Stop()
	var first []byte
	select {
	case <-r.Context().Done():
		return
	case frame, ok := <-frames:
		if !ok || len(frame) == 0 {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "screencast unavailable", "error_class": "fpv_stream_unavailable"})
			return
		}
		first = frame
	case <-timer.C:
		writeJSON(w, http.StatusGatewayTimeout, map[string]string{"error": "screencast produced no frame within its deadline", "error_class": "fpv_stream_stalled"})
		return
	}
	controller := http.NewResponseController(w)
	if err := controller.SetWriteDeadline(time.Now().Add(fpvStreamWriteTimeout)); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "bounded streaming unsupported", "error_class": "fpv_stream_deadline_unavailable"})
		return
	}
	defer controller.SetWriteDeadline(time.Time{})
	markFPVOperationalMirror(w)
	w.Header().Set("Content-Type", "multipart/x-mixed-replace; boundary=frame")
	w.Header().Set("X-Accel-Buffering", "no")
	w.Header().Set("X-UIAI-Stream-FPS", strconv.Itoa(fps))
	if _, err := fmt.Fprint(w, "--frame\r\nContent-Type: image/jpeg\r\n\r\n"); err != nil {
		return
	}
	write := func(frame []byte) bool {
		if err := controller.SetWriteDeadline(time.Now().Add(fpvStreamWriteTimeout)); err != nil {
			return false
		}
		if _, err := w.Write(frame); err != nil {
			return false
		}
		if _, err := fmt.Fprint(w, "\r\n--frame\r\nContent-Type: image/jpeg\r\n\r\n"); err != nil {
			return false
		}
		return controller.Flush() == nil
	}
	if !write(first) {
		return
	}
	ticker := time.NewTicker(time.Second / time.Duration(fps))
	defer ticker.Stop()
	latest := first
	for {
		select {
		case <-r.Context().Done():
			return
		case frame, ok := <-frames:
			if !ok {
				return
			}
			latest = frame
		case <-ticker.C:
			if latest != nil {
				if !write(latest) {
					return
				}
			}
		}
	}
}
