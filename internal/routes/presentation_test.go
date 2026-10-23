package routes

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/desktop"
	"github.com/WPUIAI/uiai-engine/internal/desktopcontract"
	"github.com/go-chi/chi/v5"
)

type routeLauncher struct{ calls int }

func (l *routeLauncher) Open(context.Context, string) error { l.calls++; return nil }

func presentationTestRouter(t *testing.T) (http.Handler, *routeLauncher) {
	t.Helper()
	launcher := &routeLauncher{}
	presenter := desktop.NewPresenter(func(id string) bool { return id == "session-route" }, launcher)
	r := chi.NewRouter()
	r.Route("/api/session", func(r chi.Router) { MountSessionPresentationRoute(r, presenter) })
	r.Route("/api/presentation", func(r chi.Router) { MountPresentationRoutes(r, presenter) })
	return r, launcher
}

func TestPresentationRoutesConvergeAndAcknowledge(t *testing.T) {
	router, launcher := presentationTestRouter(t)
	body := `{"schema":"uiai.desktop_presentation_request.v1","mode":"full","reason":"operator_request","requested_by":{"client_type":"pi","client_id":"pi-route"},"focus":true,"expires_in_ms":30000,"idempotency_key":"route-request"}`
	post := func() desktopcontract.DesktopPresentationReceipt {
		req := httptest.NewRequest(http.MethodPost, "/api/session/session-route/present", strings.NewReader(body))
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != http.StatusAccepted {
			t.Fatalf("present status=%d body=%s", res.Code, res.Body.String())
		}
		var receipt desktopcontract.DesktopPresentationReceipt
		if err := json.Unmarshal(res.Body.Bytes(), &receipt); err != nil {
			t.Fatal(err)
		}
		return receipt
	}
	first, second := post(), post()
	if first != second || launcher.calls != 1 {
		t.Fatalf("idempotency failed: calls=%d first=%#v second=%#v", launcher.calls, first, second)
	}

	ackBody := `{"schema":"uiai.desktop_presentation_ack.v1","status":"visible","cockpit_instance_id":"cockpit-route"}`
	ack := httptest.NewRequest(http.MethodPost, "/api/presentation/"+first.PresentationID+"/ack", strings.NewReader(ackBody))
	ackRes := httptest.NewRecorder()
	router.ServeHTTP(ackRes, ack)
	if ackRes.Code != http.StatusOK {
		t.Fatalf("ack status=%d body=%s", ackRes.Code, ackRes.Body.String())
	}

	statusReq := httptest.NewRequest(http.MethodGet, "/api/presentation/"+first.PresentationID, nil)
	statusRes := httptest.NewRecorder()
	router.ServeHTTP(statusRes, statusReq)
	if statusRes.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", statusRes.Code, statusRes.Body.String())
	}
	var status desktopcontract.DesktopPresentationStatus
	if err := json.Unmarshal(statusRes.Body.Bytes(), &status); err != nil {
		t.Fatal(err)
	}
	if status.Schema != desktopcontract.SchemaDesktopPresentationStatusV1 || status.Status != "visible" || status.SessionID != "session-route" {
		t.Fatalf("status=%#v", status)
	}
}

func TestPresentationRouteRejectsUnknownFieldsAndMissingSession(t *testing.T) {
	router, launcher := presentationTestRouter(t)
	invalid := httptest.NewRequest(http.MethodPost, "/api/session/session-route/present", strings.NewReader(`{"schema":"uiai.desktop_presentation_request.v1","unknown":true}`))
	invalidRes := httptest.NewRecorder()
	router.ServeHTTP(invalidRes, invalid)
	if invalidRes.Code != http.StatusBadRequest {
		t.Fatalf("invalid status=%d", invalidRes.Code)
	}

	body := `{"schema":"uiai.desktop_presentation_request.v1","mode":"full","reason":"workflow","requested_by":{"client_type":"api","client_id":"api-route"},"focus":false,"expires_in_ms":30000,"idempotency_key":"missing-route"}`
	missing := httptest.NewRequest(http.MethodPost, "/api/session/missing/present", strings.NewReader(body))
	missingRes := httptest.NewRecorder()
	router.ServeHTTP(missingRes, missing)
	if missingRes.Code != http.StatusOK {
		t.Fatalf("missing status=%d body=%s", missingRes.Code, missingRes.Body.String())
	}
	var receipt desktopcontract.DesktopPresentationReceipt
	if err := json.Unmarshal(missingRes.Body.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "session_missing" || launcher.calls != 0 {
		t.Fatalf("receipt=%#v calls=%d", receipt, launcher.calls)
	}
}
