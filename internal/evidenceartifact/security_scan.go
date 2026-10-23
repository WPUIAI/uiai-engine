package evidenceartifact

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"
)

var secretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)-----BEGIN [A-Z0-9 ]*PRIVATE KEY-----`),
	regexp.MustCompile(`(?i)authorization\s*:\s*bearer\s+[a-z0-9._~+/=-]{12,}`),
	regexp.MustCompile(`(?i)(api[_-]?key|client[_-]?secret|access[_-]?token|session[_-]?token|password)\s*[:=]\s*["']?[a-z0-9_./+=-]{12,}`),
	regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
	regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{20,}\b`),
	regexp.MustCompile(`\bsk_live_[A-Za-z0-9]{16,}\b`),
	regexp.MustCompile(`(?i)(cookie|set-cookie)\s*:\s*[^\r\n]{12,}`),
}

var (
	emailPattern   = regexp.MustCompile(`(?i)\b[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,}\b`)
	phonePattern   = regexp.MustCompile(`\b(?:\+?1[-. ]?)?\(?[2-9][0-9]{2}\)?[-. ]?[0-9]{3}[-. ]?[0-9]{4}\b`)
	activePatterns = [][]byte{
		[]byte("<script"), []byte("javascript:"), []byte("<iframe"), []byte("<object"),
		[]byte("onerror="), []byte("onload="), []byte("ignore previous"), []byte("system prompt"),
	}
)

func (i *BuiltinInspector) Inspect(ctx context.Context, request InspectionRequest) (InspectionRecord, error) {
	if i == nil || i.MaxTextBytes <= 0 || i.MaxInspectBytes <= 0 {
		return InspectionRecord{}, ErrInspectionUnavailable
	}
	if request.Security.PolicyRef != StrictSecurityPolicyV1 {
		return InspectionRecord{}, ErrInspectionRequired
	}
	if err := ctx.Err(); err != nil {
		return InspectionRecord{}, ErrInspectionUnavailable
	}
	info, err := os.Stat(request.Path)
	if err != nil || info.Size() != request.Asset.ByteSize || info.Size() <= 0 {
		return InspectionRecord{}, ErrInspectionFailed
	}
	if info.Size() > i.MaxInspectBytes {
		return InspectionRecord{}, ErrInspectionUnavailable
	}
	data, err := os.ReadFile(request.Path) // #nosec G304 -- staged hash-derived path supplied by Store.
	if err != nil {
		return InspectionRecord{}, ErrInspectionFailed
	}
	digest := sha256.Sum256(data)
	if hex.EncodeToString(digest[:]) != request.Asset.SHA256 {
		return InspectionRecord{}, ErrInspectionFailed
	}
	if err := ctx.Err(); err != nil {
		return InspectionRecord{}, ErrInspectionUnavailable
	}
	observed, text, err := inspectMedia(request.Asset.MediaType, data, i.MaxTextBytes)
	if err != nil {
		return InspectionRecord{}, err
	}
	findings := make([]string, 0, 2)
	if text {
		for _, pattern := range secretPatterns {
			if pattern.Match(data) {
				return InspectionRecord{}, ErrSensitiveContent
			}
		}
		pii := emailPattern.Match(data) || phonePattern.Match(data)
		if pii {
			if request.Policy.AccessClass == AccessPublicSafe || request.Policy.RedactionState == RedactionPublicSafe {
				return InspectionRecord{}, ErrSensitiveContent
			}
			findings = append(findings, FindingPIIDetected)
		}
		lower := bytes.ToLower(data)
		for _, pattern := range activePatterns {
			if bytes.Contains(lower, pattern) {
				findings = append(findings, FindingActiveTextUntrusted)
				break
			}
		}
	}
	findings = normalizeFindingCodes(findings)
	status := InspectionPassed
	if len(findings) > 0 {
		status = InspectionPassedWithFindings
	}
	record := InspectionRecord{
		AssetID: request.Asset.AssetID, PolicyRef: request.Security.PolicyRef, Status: status,
		ObservedMediaType: observed, FindingCodes: findings,
	}
	record.InspectionSHA256 = inspectionDigest(record, request.Asset)
	return record, nil
}

func inspectMedia(declared string, data []byte, maxText int64) (string, bool, error) {
	switch declared {
	case "application/json":
		if int64(len(data)) > maxText || !utf8.Valid(data) || !json.Valid(data) {
			return "", false, ErrMediaTypeMismatch
		}
		return declared, true, nil
	case "text/plain", "text/markdown", "text/csv", "text/vtt":
		if int64(len(data)) > maxText || !utf8.Valid(data) || bytes.IndexByte(data, 0) >= 0 {
			return "", false, ErrMediaTypeMismatch
		}
		return declared, true, nil
	case "image/png":
		if !bytes.HasPrefix(data, []byte("\x89PNG\r\n\x1a\n")) {
			return "", false, ErrMediaTypeMismatch
		}
		if containsAny(data, []byte("eXIf"), []byte("tEXt"), []byte("zTXt"), []byte("iTXt")) {
			return "", false, ErrSensitiveContent
		}
	case "image/jpeg":
		if len(data) < 4 || data[0] != 0xff || data[1] != 0xd8 || data[len(data)-2] != 0xff || data[len(data)-1] != 0xd9 {
			return "", false, ErrMediaTypeMismatch
		}
		if containsAny(data, []byte("Exif\x00\x00"), []byte("http://ns.adobe.com/xap/1.0/")) {
			return "", false, ErrSensitiveContent
		}
	case "image/gif":
		if !(bytes.HasPrefix(data, []byte("GIF87a")) || bytes.HasPrefix(data, []byte("GIF89a"))) {
			return "", false, ErrMediaTypeMismatch
		}
	case "image/webp":
		if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
			return "", false, ErrMediaTypeMismatch
		}
		if containsAny(data, []byte("EXIF"), []byte("XMP ")) {
			return "", false, ErrSensitiveContent
		}
	case "video/mp4":
		if len(data) < 12 || string(data[4:8]) != "ftyp" {
			return "", false, ErrMediaTypeMismatch
		}
	case "video/webm":
		if len(data) < 4 || !bytes.Equal(data[:4], []byte{0x1a, 0x45, 0xdf, 0xa3}) {
			return "", false, ErrMediaTypeMismatch
		}
	case "audio/wav":
		if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WAVE" {
			return "", false, ErrMediaTypeMismatch
		}
	case "audio/ogg":
		if !bytes.HasPrefix(data, []byte("OggS")) {
			return "", false, ErrMediaTypeMismatch
		}
	case "audio/mpeg":
		if !(bytes.HasPrefix(data, []byte("ID3")) || (len(data) >= 2 && data[0] == 0xff && data[1]&0xe0 == 0xe0)) {
			return "", false, ErrMediaTypeMismatch
		}
	case "text/html", "application/javascript", "image/svg+xml", "application/pdf", "application/zip",
		"application/x-tar", "application/gzip", "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", "application/vnd.openxmlformats-officedocument.presentationml.presentation":
		return "", false, ErrSanitizerRequired
	default:
		return "", false, ErrUnsafeContent
	}
	return declared, false, nil
}

func containsAny(data []byte, values ...[]byte) bool {
	for _, value := range values {
		if bytes.Contains(data, value) {
			return true
		}
	}
	return false
}

func normalizeFindingCodes(in []string) []string {
	sort.Strings(in)
	out := in[:0]
	for _, value := range in {
		value = strings.TrimSpace(value)
		if value != "" && (len(out) == 0 || out[len(out)-1] != value) {
			out = append(out, value)
		}
	}
	return out
}
