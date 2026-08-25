// C-010-01 — Intent verbs: extract / act / task.
// Agents declare outcomes; the engine executes bounded mechanics and returns
// typed results with receipts. Values of typed inputs are never echoed back.
package routes

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/vision"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// PageOps is the minimal page surface the intent layer needs (fake-tested).
type PageOps interface {
	Text(selector string) (string, error)
	Click(selector string) error
	Input(selector, value string) error
	Press(key string) error
	Navigate(url string) error
}

// ── Extract ─────────────────────────────────────────────────────────────────

type ExtractField struct {
	Selector string `json:"selector"`
	Attr     string `json:"attr,omitempty"`
	Required bool   `json:"required,omitempty"`
}

type ExtractRequest struct {
	URL       string                  `json:"url,omitempty"`
	SessionID string                  `json:"session_id,omitempty"`
	Schema    map[string]ExtractField `json:"schema"`
	BudgetID  string                  `json:"budget_id,omitempty"`
	TimeoutMS int64                   `json:"timeout_ms,omitempty"`
}

type ExtractResult struct {
	Data       map[string]string `json:"data"`
	Missing    []string          `json:"missing"`
	Confidence float64           `json:"confidence"`
	Receipt    map[string]any    `json:"receipt"`
}

func RunExtract(page PageOps, req ExtractRequest) ExtractResult {
	res := ExtractResult{Data: map[string]string{}, Missing: []string{}}
	found := 0
	total := len(req.Schema)
	for field, spec := range req.Schema {
		val, err := page.Text(spec.Selector)
		if err != nil || strings.TrimSpace(val) == "" {
			res.Missing = append(res.Missing, field)
			if spec.Required {
				continue
			}
			res.Data[field] = ""
			continue
		}
		res.Data[field] = strings.TrimSpace(val)
		found++
	}
	if total > 0 {
		res.Confidence = float64(found) / float64(total)
	}
	res.Receipt = map[string]any{
		"receipt_id": "rcpt_" + uuid.NewString()[:13],
		"verb":       "extract",
		"at":         time.Now().UTC().Format(time.RFC3339),
	}
	return res
}

// ── Act ─────────────────────────────────────────────────────────────────────

type ActStep struct {
	Type     string `json:"type"` // click|type|press|navigate
	Selector string `json:"selector,omitempty"`
	Value    string `json:"value,omitempty"` // never echoed back
	Key      string `json:"key,omitempty"`
	URL      string `json:"url,omitempty"`
}

type AssertStep struct {
	Selector string `json:"selector"`
	Contains string `json:"contains,omitempty"`
}

type ActRequest struct {
	SessionID string       `json:"session_id"`
	Actions   []ActStep    `json:"actions"`
	Asserts   []AssertStep `json:"asserts,omitempty"`
	BudgetID  string       `json:"budget_id,omitempty"`
}

type ActResult struct {
	Steps   []map[string]any `json:"steps"`
	Asserts []map[string]any `json:"asserts"`
	Ok      bool             `json:"ok"`
	Receipt map[string]any   `json:"receipt"`
}

func RunAct(page PageOps, req ActRequest) ActResult {
	res := ActResult{Ok: true, Steps: []map[string]any{}, Asserts: []map[string]any{}}
	for i, a := range req.Actions {
		step := map[string]any{"index": i, "type": a.Type, "selector": a.Selector}
		var err error
		switch strings.ToLower(a.Type) {
		case "click":
			err = page.Click(a.Selector)
		case "type":
			err = page.Input(a.Selector, a.Value)
			step["value_len"] = len(a.Value) // redaction: length only
		case "press":
			err = page.Press(a.Key)
			step["key"] = a.Key
		case "navigate":
			err = page.Navigate(a.URL)
			step["url"] = a.URL
		default:
			err = errors.New("unsupported action type")
		}
		step["ok"] = err == nil
		if err != nil {
			step["error"] = err.Error()
			res.Ok = false
			res.Steps = append(res.Steps, step)
			break
		}
		res.Steps = append(res.Steps, step)
	}
	for _, as := range req.Asserts {
		got, err := page.Text(as.Selector)
		pass := err == nil && strings.Contains(got, as.Contains)
		res.Asserts = append(res.Asserts, map[string]any{
			"selector": as.Selector, "pass": pass,
		})
		if !pass {
			res.Ok = false
		}
	}
	res.Receipt = map[string]any{
		"receipt_id": "rcpt_" + uuid.NewString()[:13],
		"verb":       "act",
		"at":         time.Now().UTC().Format(time.RFC3339),
	}
	return res
}

// ── Task orchestrator (bounded) ─────────────────────────────────────────────

type TaskStep struct {
	Verb    string          `json:"verb"` // extract|act
	Payload ExtractRequest  `json:"-"`
	Act     *ActRequest     `json:"act,omitempty"`
	Ext     *ExtractRequest `json:"extract,omitempty"`
}

type TaskRequest struct {
	Caps  BudgetLimits `json:"caps"`
	Steps []TaskStep   `json:"steps"`
}

func MountIntentRoutes(r chi.Router, sm *vision.SessionManager) {
	r.Post("/extract", func(w http.ResponseWriter, r *http.Request) {
		var req ExtractRequest
		if err := jsonDecode(r, &req); err != nil || (req.URL == "" && req.SessionID == "") {
			writeJSON(w, 400, map[string]string{"error": "url or session_id required"})
			return
		}
		page, cleanup, err := intentPage(sm, req.SessionID, req.URL)
		if err != nil {
			writeJSON(w, 502, map[string]string{"error": err.Error()})
			return
		}
		defer cleanup()
		writeJSON(w, 200, RunExtract(page, req))
	})

	r.Post("/act", func(w http.ResponseWriter, r *http.Request) {
		var req ActRequest
		if err := jsonDecode(r, &req); err != nil || req.SessionID == "" {
			writeJSON(w, 400, map[string]string{"error": "session_id required"})
			return
		}
		page, cleanup, err := intentPage(sm, req.SessionID, "")
		if err != nil {
			writeJSON(w, 502, map[string]string{"error": err.Error()})
			return
		}
		defer cleanup()
		writeJSON(w, 200, RunAct(page, req))
	})

	r.Post("/task", func(w http.ResponseWriter, r *http.Request) {
		var req TaskRequest
		if err := jsonDecode(r, &req); err != nil || len(req.Steps) == 0 {
			writeJSON(w, 400, map[string]string{"error": "steps required"})
			return
		}
		b := CreateBudget(req.Caps, "intent-task")
		results := make([]map[string]any, 0, len(req.Steps))
		status := "completed"
		for i, step := range req.Steps {
			if _, okB := GuardBudget(w, r, b.ID, 250, 1, 0); !okB {
				return // GuardBudget wrote envelope (exceeded/paused)
			}
			entry := map[string]any{"step": i, "verb": step.Verb}
			var page PageOps
			var cleanup func()
			var err error
			sid := ""
			if step.Act != nil {
				sid = step.Act.SessionID
			} else if step.Ext != nil {
				sid = step.Ext.SessionID
			}
			page, cleanup, err = intentPage(sm, sid, stepURL(step))
			if err != nil {
				entry["error"] = err.Error()
				status = "failed"
				results = append(results, entry)
				break
			}
			switch step.Verb {
			case "extract":
				if step.Ext != nil {
					res := RunExtract(page, *step.Ext)
					entry["result"] = res
				}
			case "act":
				if step.Act != nil {
					res := RunAct(page, *step.Act)
					entry["result"] = res
					if !res.Ok {
						status = "failed"
					}
				}
			default:
				entry["error"] = "unknown verb"
				status = "failed"
			}
			cleanup()
			results = append(results, entry)
			if status == "failed" {
				break
			}
		}
		budgets.mu.RLock()
		used := b.Used
		budgets.mu.RUnlock()
		writeJSON(w, 200, map[string]any{
			"status": status, "results": results, "budget_id": b.ID, "used": used,
		})
	})
}

func stepURL(step TaskStep) string {
	if step.Ext != nil {
		return step.Ext.URL
	}
	return ""
}

// sessionPageOps adapts vision.Session to the intent PageOps surface.
type sessionPageOps struct{ s *vision.Session }

func (a sessionPageOps) Text(sel string) (string, error) { return a.s.TextIntent(sel) }
func (a sessionPageOps) Click(sel string) error          { _, err := a.s.Click(sel); return err }
func (a sessionPageOps) Input(sel, v string) error       { return a.s.InputIntent(sel, v) }
func (a sessionPageOps) Press(key string) error          { return a.s.PressKeyIntent(key) }
func (a sessionPageOps) Navigate(url string) error       { _, err := a.s.Navigate(url); return err }

// intentPage resolves an existing session or opens a fresh one.
func intentPage(sm *vision.SessionManager, sessionID, url string) (PageOps, func(), error) {
	if sm == nil {
		return nil, nil, errors.New("session manager unavailable")
	}
	if sessionID != "" {
		if s, ok := sm.Get(sessionID); ok {
			return sessionPageOps{s}, func() {}, nil
		}
		return nil, nil, errors.New("unknown session")
	}
	if url == "" {
		return nil, nil, errors.New("url required when no session")
	}
	s, _, err := sm.Open(url, 1280, 800)
	if err != nil {
		return nil, nil, err
	}
	return sessionPageOps{s}, func() { _ = sm.Close(s.ID) }, nil
}
