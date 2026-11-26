package vision

import (
	"fmt"
	"strings"
)

// #294 understand slice: one call returning bounded frames plus the
// transcript of the same media, sized for a vision model, with explicit
// limits carried through from the watch and hear slices.

// MediaUnderstandResult combines sampled frames with the transcript.
type MediaUnderstandResult struct {
	Frames     []VideoFrame           `json:"frames"`
	Truncated  bool                   `json:"truncated"`
	Transcript *MediaTranscriptResult `json:"transcript"`
	Notes      []string               `json:"notes"`
}

// MediaUnderstand samples frames from a session <video> element and
// transcribes its audio in one call. Audio-only elements and direct URLs
// return the transcript with a note explaining why frames are absent —
// the agent sees exactly what was consumed and what was not.
func (s *Session) MediaUnderstand(selector, mediaURL, whisperURL string, frameCount int) (*MediaUnderstandResult, error) {
	result := &MediaUnderstandResult{Notes: []string{}}
	if selector == "" && mediaURL == "" {
		return nil, fmt.Errorf("selector or url is required")
	}

	s.mu.Lock()
	page := s.page
	s.mu.Unlock()
	if page == nil {
		return nil, fmt.Errorf("session closed")
	}

	isVideo := false
	if selector != "" {
		element, err := actionElement(page, selector)
		if err != nil {
			return nil, fmt.Errorf("media element not found: %w", err)
		}
		probe, perr := element.Eval(`() => this.tagName`)
		if perr != nil {
			return nil, fmt.Errorf("media probe failed: %w", perr)
		}
		isVideo = strings.EqualFold(probe.Value.Str(), "VIDEO")
		if isVideo {
			frames, truncated, ferr := s.SampleVideoFrames(selector, frameCount, 0, 0)
			if ferr != nil {
				result.Notes = append(result.Notes, fmt.Sprintf("frame sampling failed: %v", ferr))
			} else {
				result.Frames = frames
				result.Truncated = truncated
			}
		} else {
			result.Notes = append(result.Notes, "frames require a video element; this element is audio-only")
		}
	} else {
		result.Notes = append(result.Notes, "frames require an in-page video element selector; a direct url cannot be framed")
	}

	transcript, terr := s.MediaTranscript(selector, mediaURL, whisperURL)
	if terr != nil {
		result.Notes = append(result.Notes, fmt.Sprintf("transcription failed: %v", terr))
	} else {
		result.Transcript = transcript
	}
	if result.Transcript == nil && result.Frames == nil {
		return nil, fmt.Errorf("media understand produced neither frames nor a transcript")
	}
	return result, nil
}
