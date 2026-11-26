package vision

import (
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-rod/rod/lib/proto"
)

// #301 upload slice: the agent supplies file bytes (never a raw host path);
// the engine writes them into the session's governed file area and sets them
// on the page's file input through CDP, so the upload is visible in the next
// snapshot (input shows the file name).

const maxUploadBytes = 25 << 20

// sessionFilesDirLocked returns the session's governed file directory,
// creating it if the download setup did not run. Caller must hold s.mu.
func (s *Session) sessionFilesDirLocked() (string, error) {
	if s.downloadsDir != "" {
		return s.downloadsDir, nil
	}
	if s.page == nil {
		return "", fmt.Errorf("session closed")
	}
	dir := filepath.Join(os.TempDir(), "uiai-session-downloads", "sessions", s.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	s.downloadsDir = dir
	return dir, nil
}

// UploadFile writes base64 content into the session's governed file area and
// sets it on the page's file input located by selector. Returns a fresh
// capture so the chosen file name is visible to the agent.
func (s *Session) UploadFile(selector, filename string, contentBase64 string) (*SnapResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.page == nil {
		return nil, fmt.Errorf("session closed")
	}
	start := time.Now()
	if selector == "" || filename == "" {
		return nil, fmt.Errorf("selector and filename are required")
	}
	content, err := base64.StdEncoding.DecodeString(contentBase64)
	if err != nil {
		return nil, fmt.Errorf("upload content is not valid base64: %w", err)
	}
	if len(content) == 0 {
		return nil, fmt.Errorf("upload content is empty")
	}
	if len(content) > maxUploadBytes {
		return nil, fmt.Errorf("upload exceeds the %d byte bound", maxUploadBytes)
	}
	clean := filepath.Base(filename)
	if clean == "." || clean == string(filepath.Separator) || clean != filename && filepath.Base(clean) != clean {
		return nil, fmt.Errorf("upload filename must be a plain file name")
	}
	if clean != filename {
		filename = clean
	}

	dir, err := s.sessionFilesDirLocked()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return nil, fmt.Errorf("upload write failed: %w", err)
	}

	element, err := actionElement(s.page, selector)
	if err != nil {
		return nil, fmt.Errorf("upload input not found: %w", err)
	}
	probe, perr := element.Eval(`() => this.tagName + ':' + (this.type || '')`)
	if perr != nil {
		return nil, fmt.Errorf("upload input probe failed: %w", perr)
	}
	if probe.Value.Str() != "INPUT:file" {
		return nil, fmt.Errorf("selector %s is not a file input (%s)", selector, probe.Value.Str())
	}
	if err := element.SetFiles([]string{path}); err != nil {
		return nil, fmt.Errorf("upload set failed: %w", err)
	}

	s.URL = safeEvalStr(s.page, `() => window.location.href`)
	s.Title = safePageTitle(s.page)
	s.touch()
	snapshot, err := s.snapWithPage(s.page, start)
	if err != nil {
		return nil, fmt.Errorf("upload snapshot failed: %w", err)
	}
	return snapshot, nil
}

var _ = proto.InputMouseButtonLeft // keep proto import aligned with the session action paths
