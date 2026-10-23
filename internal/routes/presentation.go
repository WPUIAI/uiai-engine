package routes

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/WPUIAI/uiai-engine/internal/desktop"
	"github.com/WPUIAI/uiai-engine/internal/desktopcontract"
	"github.com/go-chi/chi/v5"
)

const presentationAckSchemaV1 = "uiai.desktop_presentation_ack.v1"

func MountSessionPresentationRoute(r chi.Router, presenter desktop.DesktopPresenter) {
	if presenter == nil {
		return
	}
	r.Post("/{id}/present", func(w http.ResponseWriter, req *http.Request) {
		var body desktopcontract.DesktopPresentationRequest
		if err := decodeStrictJSON(req, &body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_presentation_request", "detail": err.Error()})
			return
		}
		receipt, err := presenter.EnsureVisible(req.Context(), chi.URLParam(req, "id"), body)
		if err != nil {
			status := http.StatusBadRequest
			code := "invalid_presentation_request"
			if errors.Is(err, desktop.ErrIdempotencyConflict) {
				status, code = http.StatusConflict, "idempotency_conflict"
			}
			writeJSON(w, status, map[string]string{"error": code, "detail": err.Error()})
			return
		}
		if receipt.Status == "desktop_unavailable" {
			w.Header().Set("Link", "</api/fpv>; rel=\"recovery\"")
		}
		status := http.StatusAccepted
		if receipt.Status != "launching" && receipt.Status != "requested" {
			status = http.StatusOK
		}
		writeJSON(w, status, receipt)
	})
}

func MountPresentationRoutes(r chi.Router, presenter desktop.DesktopPresenter) {
	if presenter == nil {
		return
	}
	r.Get("/{presentation_id}", func(w http.ResponseWriter, req *http.Request) {
		receipt, err := presenter.Status(req.Context(), chi.URLParam(req, "presentation_id"))
		if err != nil {
			writePresentationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, desktopcontract.DesktopPresentationStatus{
			Schema:         desktopcontract.SchemaDesktopPresentationStatusV1,
			PresentationID: receipt.PresentationID, SessionID: receipt.SessionID,
			Status: receipt.Status, ReasonCode: receipt.ReasonCode,
			ObservedAt: time.Now().UTC().Format(time.RFC3339Nano),
		})
	})

	r.Post("/{presentation_id}/ack", func(w http.ResponseWriter, req *http.Request) {
		var body struct {
			Schema            string `json:"schema"`
			Status            string `json:"status"`
			CockpitInstanceID string `json:"cockpit_instance_id,omitempty"`
			ReasonCode        string `json:"reason_code,omitempty"`
		}
		if err := decodeStrictJSON(req, &body); err != nil || body.Schema != presentationAckSchemaV1 {
			detail := "unexpected acknowledgement schema"
			if err != nil {
				detail = err.Error()
			}
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_presentation_ack", "detail": detail})
			return
		}
		receipt, err := presenter.Acknowledge(req.Context(), chi.URLParam(req, "presentation_id"), desktop.Acknowledgement{
			Status: body.Status, CockpitInstanceID: body.CockpitInstanceID, ReasonCode: body.ReasonCode,
		})
		if err != nil {
			writePresentationError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, receipt)
	})
}

func decodeStrictJSON(req *http.Request, target any) error {
	decoder := json.NewDecoder(http.MaxBytesReader(nilResponseWriter{}, req.Body, 64<<10))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("request must contain one JSON object")
	}
	return nil
}

// nilResponseWriter is used only to apply MaxBytesReader's bounded reader; it never writes a response.
type nilResponseWriter struct{}

func (nilResponseWriter) Header() http.Header       { return make(http.Header) }
func (nilResponseWriter) Write([]byte) (int, error) { return 0, nil }
func (nilResponseWriter) WriteHeader(int)           {}

func writePresentationError(w http.ResponseWriter, err error) {
	if errors.Is(err, desktop.ErrPresentationNotFound) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "presentation_not_found"})
		return
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_presentation", "detail": err.Error()})
}
