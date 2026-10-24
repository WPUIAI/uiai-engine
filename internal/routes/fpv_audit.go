package routes

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

func fpvAuditEventsHandler(w http.ResponseWriter, req *http.Request) {
	since := uint64(0)
	if raw := strings.TrimSpace(req.URL.Query().Get("since_seq")); raw != "" {
		_, _ = fmt.Sscan(raw, &since)
	}
	limit := 50
	if raw := strings.TrimSpace(req.URL.Query().Get("limit")); raw != "" {
		var parsed int
		if _, err := fmt.Sscan(raw, &parsed); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}
	events, latest := fpvAuditEventsSince(since, limit)
	writeJSON(w, http.StatusOK, map[string]any{"events": events, "latest_seq": latest, "count": len(events)})
}

func fpvRecordAuditEvent(token, sessionID string, log fpvAuditLog) fpvAuditLog {
	// FPV capabilities use unreserved base64url characters, so the exact token
	// also occurs verbatim in encoded JSON. Clone/redact before storing metadata.
	redact := func(value string) string {
		if token == "" {
			return value
		}
		return strings.ReplaceAll(value, token, "[REDACTED]")
	}
	log.Action, log.Selector, log.Key = redact(log.Action), redact(log.Selector), redact(log.Key)
	log.Message, log.Error = redact(log.Message), redact(log.Error)
	var meta map[string]any
	data, err := json.Marshal(log.Meta)
	if err == nil && len(data) <= 10000 {
		if token != "" {
			data = bytes.ReplaceAll(data, []byte(token), []byte("[REDACTED]"))
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		err = decoder.Decode(&meta)
	}
	if err != nil || len(data) > 10000 {
		meta = map[string]any{"metadata_withheld": true}
	}
	if meta == nil {
		meta = map[string]any{}
	}
	meta["token"] = "[REDACTED]"
	meta["share_ref"] = fmt.Sprintf("fpv-share:sha256:%x", sha256.Sum256([]byte(token)))
	meta["session_id"] = sessionID
	log.Meta = meta
	fpvAuditMu.Lock()
	defer fpvAuditMu.Unlock()
	fpvAuditSeq++
	log.Seq = fpvAuditSeq
	fpvAuditEvents = append(fpvAuditEvents, log)
	if len(fpvAuditEvents) > 500 {
		fpvAuditEvents = fpvAuditEvents[len(fpvAuditEvents)-500:]
	}
	return log
}

func fpvAuditEventsSince(since uint64, limit int) ([]fpvAuditLog, uint64) {
	fpvAuditMu.Lock()
	defer fpvAuditMu.Unlock()
	latest := fpvAuditSeq
	items := []fpvAuditLog{}
	for _, event := range fpvAuditEvents {
		if event.Seq > since {
			items = append(items, event)
		}
	}
	if len(items) > limit {
		items = items[len(items)-limit:]
	}
	return items, latest
}
