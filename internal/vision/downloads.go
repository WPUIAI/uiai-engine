package vision

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/focusapacket"
	"github.com/go-rod/rod/lib/proto"
)

// #301 download slice: page-initiated downloads land in a governed
// per-session directory (config-rooted, private fallback), are listed with
// digests and progress events, and are served back bounded. The directory
// is removed on session close — downloads are session-scoped.

const (
	maxDownloadEvents     = 50
	MaxDownloadServeBytes = 25 << 20
)

// DownloadEvent records one download state change observed from the page.
type DownloadEvent struct {
	Seq      uint64 `json:"seq"`
	TS       string `json:"ts"`
	GUID     string `json:"guid,omitempty"`
	Name     string `json:"name,omitempty"`
	URL      string `json:"url,omitempty"`
	State    string `json:"state"`
	Received int64  `json:"received,omitempty"`
	Total    int64  `json:"total,omitempty"`
}

// DownloadFile describes one landed download in the session's governed dir.
type DownloadFile struct {
	Name      string `json:"name"`
	Bytes     int64  `json:"bytes"`
	SHA256    string `json:"sha256"`
	CreatedAt string `json:"created_at"`
}

// initDownloads points the page's download behavior at a governed
// per-session directory. Called during session open; non-fatal on failure
// (the session still opens, downloads degrade).
func (s *Session) initDownloads(root string) {
	if s.page == nil {
		return
	}
	base := root
	if base == "" {
		base = filepath.Join(os.TempDir(), "uiai-session-downloads")
	}
	dir := filepath.Join(base, "sessions", s.ID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		log.Printf("[vision] download dir setup failed for %s: %v", s.ID, err)
		return
	}
	if err := (proto.PageSetDownloadBehavior{
		Behavior: proto.PageSetDownloadBehaviorBehaviorAllow, DownloadPath: dir,
	}).Call(s.page); err != nil {
		log.Printf("[vision] download behavior setup failed for %s: %v", s.ID, err)
		_ = os.Remove(dir)
		return
	}
	s.mu.Lock()
	s.downloadsDir = dir
	s.mu.Unlock()
}

// recordDownloadEvent appends to the bounded download event ring.
func (s *Session) recordDownloadEvent(e DownloadEvent) {
	if s.diagnostics == nil {
		return
	}
	s.diagnostics.mu.Lock()
	defer s.diagnostics.mu.Unlock()
	s.diagnostics.seq++
	e.Seq = s.diagnostics.seq
	e.TS = time.Now().UTC().Format(time.RFC3339)
	s.downloadEvents = append(s.downloadEvents, e)
	if len(s.downloadEvents) > maxDownloadEvents {
		s.downloadEvents = s.downloadEvents[len(s.downloadEvents)-maxDownloadEvents:]
	}
}

// noteDownloadBegin is wired into the session event listener.
func (s *Session) noteDownloadBegin(e *proto.PageDownloadWillBegin) {
	s.recordDownloadEvent(DownloadEvent{
		GUID: e.GUID, Name: filepath.Base(e.SuggestedFilename),
		URL: focusapacket.SanitizeURL(e.URL), State: "started",
	})
}

// noteDownloadProgress is wired into the session event listener.
func (s *Session) noteDownloadProgress(e *proto.PageDownloadProgress) {
	s.recordDownloadEvent(DownloadEvent{
		GUID: e.GUID, State: string(e.State),
		Received: int64(e.ReceivedBytes), Total: int64(e.TotalBytes),
	})
}

// Downloads lists the files that landed in the session's governed download
// directory, with sizes and SHA-256 digests.
func (s *Session) Downloads() []DownloadFile {
	s.mu.Lock()
	dir := s.downloadsDir
	s.mu.Unlock()
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]DownloadFile, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, ierr := entry.Info()
		if ierr != nil {
			continue
		}
		file := DownloadFile{Name: entry.Name(), Bytes: info.Size(), CreatedAt: info.ModTime().UTC().Format(time.RFC3339)}
		if digest, derr := fileDigest(filepath.Join(dir, entry.Name()), MaxDownloadServeBytes); derr == nil {
			file.SHA256 = digest
		}
		out = append(out, file)
	}
	return out
}

// DownloadEvents returns a copy of the bounded download event ring.
func (s *Session) DownloadEvents() []DownloadEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]DownloadEvent, len(s.downloadEvents))
	copy(out, s.downloadEvents)
	return out
}

// DownloadFileBytes serves one landed download bounded to
// MaxDownloadServeBytes; names are sanitized to their base so no path can
// escape the governed directory.
func (s *Session) DownloadFileBytes(name string) ([]byte, error) {
	s.mu.Lock()
	dir := s.downloadsDir
	s.mu.Unlock()
	if dir == "" {
		return nil, fmt.Errorf("downloads not available for this session")
	}
	clean := filepath.Base(name)
	if clean == "." || clean == string(filepath.Separator) {
		return nil, fmt.Errorf("invalid download name")
	}
	path := filepath.Join(dir, clean)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return nil, fmt.Errorf("download not found: %s", clean)
	}
	if info.Size() > MaxDownloadServeBytes {
		return nil, fmt.Errorf("download exceeds the %d byte serve bound", MaxDownloadServeBytes)
	}
	return os.ReadFile(path)
}

// clearDownloads removes the session's governed download directory; called
// from closeSession. Caller must hold s.mu.
func (s *Session) clearDownloadsLocked() {
	if s.downloadsDir == "" {
		return
	}
	_ = os.RemoveAll(s.downloadsDir)
	s.downloadsDir = ""
}

func fileDigest(path string, capBytes int64) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, io.LimitReader(f, capBytes)); err != nil {
		return "", err
	}
	return hex.EncodeToString(digest.Sum(nil)), nil
}
