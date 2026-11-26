package vision

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ledongthuc/pdf"
)

// #301 PDF text slice: a PDF that landed in the session's governed file area
// (downloaded or uploaded) can be read as bounded plain text so agents can
// consume invoices, statements, and documents without leaving the session.

// MaxPDFTextBytes bounds extracted PDF text output.
const MaxPDFTextBytes = 256 << 10

// DownloadFileText extracts bounded plain text from one PDF in the session's
// governed file area. Only PDF files are supported; names are sanitized to
// their base component like the byte-serving route.
func (s *Session) DownloadFileText(name string) (string, error) {
	s.mu.Lock()
	dir := s.downloadsDir
	s.mu.Unlock()
	if dir == "" {
		return "", fmt.Errorf("downloads not available for this session")
	}
	clean := filepath.Base(name)
	if clean == "." || clean == "" || clean == string(filepath.Separator) {
		return "", fmt.Errorf("invalid download name")
	}
	if !strings.EqualFold(clean, filepath.Ext(clean)) && !strings.EqualFold(filepath.Ext(clean), ".pdf") {
		return "", fmt.Errorf("text extraction supports PDF downloads only")
	}
	if strings.EqualFold(filepath.Ext(clean), "") {
		return "", fmt.Errorf("text extraction supports PDF downloads only")
	}
	path := filepath.Join(dir, clean)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", fmt.Errorf("download not found: %s", clean)
	}
	if info.Size() > MaxDownloadServeBytes {
		return "", fmt.Errorf("PDF exceeds the %d byte read bound", MaxDownloadServeBytes)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("PDF read failed: %w", err)
	}
	reader, err := pdf.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("PDF parse failed: %w", err)
	}
	textReader, err := reader.GetPlainText()
	if err != nil {
		return "", fmt.Errorf("PDF text extraction failed: %w", err)
	}
	out, err := io.ReadAll(io.LimitReader(textReader, MaxPDFTextBytes))
	if err != nil {
		return "", fmt.Errorf("PDF text read failed: %w", err)
	}
	return string(out), nil
}
