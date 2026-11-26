package vision

import (
	"encoding/base64"
	"fmt"
	"hash/fnv"
)

// #294 watch slice: agents can sample bounded frames from a <video> element
// in a live session, so a page's video content becomes inspectable without
// leaving the session. Each frame is the seeked frame drawn into an in-page
// canvas and exported as a JPEG; the audio/hear path is a separate slice.

const (
	maxVideoFrameSamples = 12
	maxVideoFramesBytes  = 6 << 20
)

// VideoFrame is one sampled frame from a session video element.
type VideoFrame struct {
	TimestampMs int64  `json:"timestamp_ms"`
	JPEGB64     string `json:"jpeg_base64"`
	Bytes       int    `json:"bytes"`
	Hash        string `json:"hash"`
}

// SampleVideoFrames seeks the session's <video> element to count evenly
// spaced timestamps in [startMs, endMs] and returns JPEG frames. The result
// is truncated when the byte bound is hit.
func (s *Session) SampleVideoFrames(selector string, count int, startMs, endMs int64) ([]VideoFrame, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, false, fmt.Errorf("session closed")
	}
	if count <= 0 {
		count = 3
	}
	if count > maxVideoFrameSamples {
		count = maxVideoFrameSamples
	}
	element, err := actionElement(s.page, selector)
	if err != nil {
		return nil, false, fmt.Errorf("video element not found: %w", err)
	}
	probe, err := element.Eval(`() => this.tagName + '|' + (this.duration || 0) + '|' + (this.readyState || 0)`)
	if err != nil {
		return nil, false, fmt.Errorf("video probe failed: %w", err)
	}
	probeParts := splitProbe(probe.Value.Str())
	tag := probeParts[0]
	var dur float64
	var ready int
	_, _ = fmt.Sscanf(probeParts[1], "%g", &dur)
	_, _ = fmt.Sscanf(probeParts[2], "%d", &ready)
	if tag != "VIDEO" {
		return nil, false, fmt.Errorf("selector %s is not a video element (%s)", selector, tag)
	}
	if dur <= 0 || ready < 2 {
		return nil, false, fmt.Errorf("video has no loaded media (duration=%.2fs readyState=%d)", dur, ready)
	}

	if startMs < 0 {
		startMs = 0
	}
	if endMs <= 0 || endMs > int64(dur*1000) {
		endMs = int64(dur * 1000)
	}
	if startMs >= endMs {
		return nil, false, fmt.Errorf("empty sample range: start_ms=%d end_ms=%d", startMs, endMs)
	}

	if _, err := element.Eval(`() => { this.pause(); return 'paused'; }`); err != nil {
		return nil, false, fmt.Errorf("video pause failed: %w", err)
	}

	// The seeked frame is drawn into an in-page canvas and exported as a JPEG
	// data URL. This reads the decoded frame directly and never depends on
	// headless compositor video painting, which can stay frozen on the first
	// presented frame.
	captureJS := `(ts) => new Promise((resolve, reject) => {
		const v = this;
		const finish = () => {
			try {
				const scale = Math.min(1, 640 / Math.max(v.videoWidth, v.videoHeight));
				const cw = Math.max(2, Math.round(v.videoWidth * scale));
				const ch = Math.max(2, Math.round(v.videoHeight * scale));
				const canvas = document.createElement('canvas');
				canvas.width = cw; canvas.height = ch;
				canvas.getContext('2d').drawImage(v, 0, 0, cw, ch);
				const url = canvas.toDataURL('image/jpeg', 0.6);
				resolve(url.substring(url.indexOf(',') + 1));
			} catch (e) { reject(e); }
		};
		if (Math.abs(v.currentTime - ts) < 0.001) { finish(); return; }
		const timer = setTimeout(() => { v.removeEventListener('seeked', onseeked); reject(new Error('seek timeout')); }, 4000);
		const onseeked = () => { clearTimeout(timer); finish(); };
		v.addEventListener('seeked', onseeked, { once: true });
		try { v.currentTime = ts; } catch (e) { clearTimeout(timer); reject(e); }
	})`

	frames := make([]VideoFrame, 0, count)
	truncated := false
	var total int
	for i := 0; i < count; i++ {
		// Even spacing including both endpoints of the range.
		ts := startMs + (endMs-startMs)*int64(i)/(int64(count-1))
		if count == 1 {
			ts = startMs
		}
		result, err := element.Eval(captureJS, float64(ts)/1000.0)
		if err != nil {
			return frames, truncated, fmt.Errorf("frame capture at %dms failed: %w", ts, err)
		}
		if result == nil {
			return frames, truncated, fmt.Errorf("frame capture at %dms returned no result", ts)
		}
		dataURLB64 := result.Value.Str()
		jpeg, err := base64.StdEncoding.DecodeString(dataURLB64)
		if err != nil || len(jpeg) == 0 {
			return frames, truncated, fmt.Errorf("frame at %dms did not produce a decodable image", ts)
		}
		total += len(jpeg)
		if total > maxVideoFramesBytes {
			truncated = true
			break
		}
		h := fnv.New64a()
		h.Write(jpeg)
		frames = append(frames, VideoFrame{
			TimestampMs: ts,
			JPEGB64:     dataURLB64,
			Bytes:       len(jpeg),
			Hash:        fmt.Sprintf("%016x", h.Sum64()),
		})
	}
	s.touch()
	return frames, truncated, nil
}

// splitProbe splits a "tag|num|num" probe string; missing parts are empty.
func splitProbe(tag string) []string {
	out := make([]string, 3)
	idx, cur := 0, 0
	for i := 0; i < len(tag) && idx < 3; i++ {
		if tag[i] == '|' {
			out[idx], cur = tag[cur:i], i+1
			idx++
		}
	}
	if idx < 3 {
		out[idx] = tag[cur:]
	}
	return out
}
