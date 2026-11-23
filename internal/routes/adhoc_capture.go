package routes

import (
	"crypto/rand"
	"encoding/hex"
	"math"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
)

// #299/#308: authorized ad hoc capture path. When an operator-directed or
// authorized autonomous capture supplies no evidence scope, the capture may
// still be productive: viewable pixels are served through a private,
// short-lived, in-memory image route. Nothing is written to disk (scope-blocked
// packages remain metadata-only per #313), no public EPWA link is created, and
// formal project evidence still requires a complete scope — this path is not an
// evidence bypass and scoped requests keep their mandatory EPWA delivery.

const (
	adhocTokenTTL    = 10 * time.Minute
	adhocMaxCaptures = 256
	adhocTokenLen    = 32
)

type adhocCapture struct {
	data      []byte
	format    string
	createdAt time.Time
	expiresAt time.Time
}

var (
	adhocMu       sync.Mutex
	adhocCaptures = make(map[string]*adhocCapture)
)

func registerAdhocCapture(data []byte, format string) string {
	tokenBytes := make([]byte, adhocTokenLen/2)
	if _, err := rand.Read(tokenBytes); err != nil {
		// crypto/rand failure is not recoverable; fail closed without a link.
		return ""
	}
	token := hex.EncodeToString(tokenBytes)
	now := time.Now()
	adhocMu.Lock()
	defer adhocMu.Unlock()
	for id, entry := range adhocCaptures {
		if now.After(entry.expiresAt) {
			delete(adhocCaptures, id)
		}
	}
	for len(adhocCaptures) >= adhocMaxCaptures {
		oldest, oldestAt := "", time.Time{}
		for id, entry := range adhocCaptures {
			if oldest == "" || entry.createdAt.Before(oldestAt) {
				oldest, oldestAt = id, entry.createdAt
			}
		}
		if oldest == "" {
			break
		}
		delete(adhocCaptures, oldest)
	}
	adhocCaptures[token] = &adhocCapture{data: data, format: format, createdAt: now, expiresAt: now.Add(adhocTokenTTL)}
	return token
}

func adhocCaptureURL(token string) string {
	return "/api/screenshot/adhoc/" + token
}

var adhocMIME = map[string]string{"png": "image/png", "jpg": "image/jpeg", "jpeg": "image/jpeg", "webp": "image/webp"}

// adhocUnsealedResponse decorates a blocked-but-successful capture result with
// the truthful unsealed work contract: the action worked, the pixels are
// privately reachable for a short window, and no formal evidence was published.
func adhocUnsealedResponse(response map[string]any, data []byte, format string) {
	if response == nil {
		return
	}
	token := registerAdhocCapture(data, format)
	if token == "" {
		return
	}
	response["delivery_state"] = "unsealed"
	response["ok"] = true
	response["capture_posture"] = "adhoc_unsealed_not_formal_evidence"
	response["private_image_url"] = adhocCaptureURL(token)
	response["private_image_expires_in"] = "10m"
	response["private_image_hint"] = "GET the private_image_url on the same loopback engine; ad hoc pixels are never published as formal evidence"
}

func mountAdhocCaptureRoutes(r chi.Router) {
	r.Get("/adhoc/{token}", func(w http.ResponseWriter, req *http.Request) {
		token := strings.TrimSpace(chi.URLParam(req, "token"))
		if len(token) != adhocTokenLen {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "ad hoc capture not found", "reason_code": "adhoc_unknown"})
			return
		}
		adhocMu.Lock()
		entry, ok := adhocCaptures[token]
		valid := ok && !time.Now().After(entry.expiresAt)
		var data []byte
		var format string
		remaining := time.Duration(0)
		if valid {
			data, format = entry.data, entry.format
			remaining = time.Until(entry.expiresAt)
		}
		adhocMu.Unlock()
		if !valid {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "ad hoc capture not found or expired; re-capture for a fresh private link", "reason_code": "adhoc_expired_or_unknown"})
			return
		}
		mime := adhocMIME[strings.ToLower(format)]
		if mime == "" {
			mime = "application/octet-stream"
		}
		w.Header().Set("Content-Type", mime)
		w.Header().Set("Cache-Control", "private, no-store")
		w.Header().Set("X-UIAI-Capture-Posture", "adhoc_unsealed_not_formal_evidence")
		w.Header().Set("X-UIAI-Private-Link-Expires-In", strconv.Itoa(int(math.Ceil(remaining.Seconds())))+"s")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})
}
