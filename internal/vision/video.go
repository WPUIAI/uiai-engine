package vision

import (
	"encoding/base64"
	"encoding/json"
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

	// Frame capture reads the decoded frame directly: the target frame is
	// drawn into an in-page canvas and exported as a JPEG data URL, never
	// depending on headless compositor video painting. Two strategies:
	// a confirmed seek (fast, and the only option for rewinding), with a
	// play-to-crossing fallback — Chrome headless variants exist that drop
	// seek requests on a paused pipeline while the decoder still advances
	// during playback, so capture-at-crossing reaches the same frames.
	captureJS := `(tsMs) => new Promise(async (resolve, reject) => {
		const v = this;
		const ts = tsMs / 1000;
		const draw = () => {
			const scale = Math.min(1, 640 / Math.max(v.videoWidth, v.videoHeight));
			const cw = Math.max(2, Math.round(v.videoWidth * scale));
			const ch = Math.max(2, Math.round(v.videoHeight * scale));
			const canvas = document.createElement('canvas');
			canvas.width = cw; canvas.height = ch;
			canvas.getContext('2d').drawImage(v, 0, 0, cw, ch);
			const url = canvas.toDataURL('image/jpeg', 0.6);
			resolve(JSON.stringify({ t: Math.round(v.currentTime * 1000), b64: url.substring(url.indexOf(',') + 1) }));
		};
		try {
			if (Math.abs(v.currentTime - ts) < 0.005) { draw(); return; }
			if (v.currentTime < ts - 0.25) {
				// Confirmed seek: retry because the initial-load seeked can
				// satisfy a listener before the requested seek lands.
				for (let attempt = 0; attempt < 2; attempt++) {
					await new Promise((resolve) => {
						const timer = setTimeout(() => { v.removeEventListener('seeked', onseeked); resolve(); }, 2000);
						const onseeked = () => { clearTimeout(timer); resolve(); };
						v.addEventListener('seeked', onseeked, { once: true });
						try { v.currentTime = ts; } catch (e) { clearTimeout(timer); resolve(); }
					});
					await new Promise(r => setTimeout(r, 60));
					if (Math.abs(v.currentTime - ts) < 0.05) { draw(); return; }
				}
				// Play-to-crossing fallback for pipelines that drop seeks.
				v.muted = true;
				try { await v.play(); } catch (e) { reject(new Error('play refused for crossing capture')); return; }
				const deadline = Date.now() + 8000;
				while (v.currentTime < ts && !v.ended && Date.now() < deadline) {
					await new Promise(r => setTimeout(r, 40));
				}
				v.pause();
				if (v.currentTime < ts) { reject(new Error('playback never reached the target')); return; }
			}
			draw();
		} catch (e) { reject(e); }
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
		result, err := element.Eval(captureJS, ts)
		if err != nil {
			return frames, truncated, fmt.Errorf("frame capture at %dms failed: %w", ts, err)
		}
		if result == nil {
			return frames, truncated, fmt.Errorf("frame capture at %dms returned no result", ts)
		}
		var captured struct {
			T   int64  `json:"t"`
			B64 string `json:"b64"`
		}
		if err := json.Unmarshal([]byte(result.Value.Str()), &captured); err != nil || captured.B64 == "" {
			return frames, truncated, fmt.Errorf("frame at %dms did not produce a decodable image", ts)
		}
		dataURLB64 := captured.B64
		jpeg, err := base64.StdEncoding.DecodeString(dataURLB64)
		if err != nil || len(jpeg) == 0 {
			return frames, truncated, fmt.Errorf("frame at %dms did not produce a decodable image", ts)
		}
		// Record where the frame actually landed (crossing captures land
		// slightly after the request).
		if captured.T > 0 {
			ts = captured.T
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
