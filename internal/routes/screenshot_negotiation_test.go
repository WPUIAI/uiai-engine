package routes

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/WPUIAI/uiai-engine/internal/config"
	"github.com/WPUIAI/uiai-engine/internal/evidencepwa"
	"github.com/WPUIAI/uiai-engine/internal/evidenceshare"
	"github.com/go-chi/chi/v5"
)

// TestShareRecordContentNegotiation verifies the canonical share URL serves
// the EPWA viewer webpage to browsers (Accept: text/html) and the durable
// JSON record to agents (Accept: application/json or default */*).
func TestShareRecordContentNegotiation(t *testing.T) {
	t.Setenv("UIAI_SCREENSHOT_DIR", t.TempDir())
	t.Setenv("UIAI_EVIDENCE_SHARE_DIR", "")
	cfg := &config.Config{Storage: config.StorageConfig{DataDir: t.TempDir()}}
	router := chi.NewRouter()
	MountShortShare(router, cfg)
	router.Route("/api/screenshot", func(r chi.Router) { MountScreenshotReal(r, cfg, screenshotSharePool{}, nil, nil) })

	requestBody, err := json.Marshal(map[string]any{
		"url": "https://example.com/", "width": 375, "height": 812, "format": "png",
		"evidence_scope": evidenceshare.Scope{
			ProjectRef: "project:test", WorkstreamRef: "workstream:epwa", WorksetRef: "workset:negotiation",
			CallGraphRef: "callgraph:test", WorkpointRef: "workpoint:negotiation", WorkItemRef: "work-item:negotiation",
			ContinuityRef: "continuity:negotiation",
			WorkItems: []evidencepwa.WorkItemProjection{{
				ProviderSurface: "screenshot.capture", WorkItemRef: "work-item:negotiation", ItemID: "negotiation-1", ItemType: "task",
				Title: "Content negotiation", Description: "Viewer for browsers, record for agents.",
				DescriptionState: "visible", Revision: "r1", Digest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
				RevisionState: "current", StatusAtCapture: "in_progress", ClosurePosture: "open",
			}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "https://engine.example/api/screenshot/", bytes.NewReader(requestBody)))
	if recorder.Code != http.StatusCreated && recorder.Code != http.StatusOK {
		t.Fatalf("publish: %d %s", recorder.Code, recorder.Body.String())
	}
	var published struct {
		EPWADelivery struct {
			EPWA struct {
				RecordURL string `json:"record_url"`
			} `json:"epwa"`
		} `json:"epwa_delivery"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &published); err != nil {
		t.Fatal(err)
	}
	recordURL := published.EPWADelivery.EPWA.RecordURL
	if recordURL == "" {
		t.Fatal("record_url empty")
	}
	// Canonical no-slash record path exercises the negotiated handler itself.
	recordPath := strings.TrimSuffix(mustPath(t, recordURL), "/")

	// Browser negotiation: Accept: text/html → 302 to the EPWA viewer webpage.
	browser := httptest.NewRecorder()
	browserReq := httptest.NewRequest(http.MethodGet, recordPath, nil)
	browserReq.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	router.ServeHTTP(browser, browserReq)
	if browser.Code != http.StatusFound || browser.Header().Get("Location") != recordPath+"/" {
		t.Fatalf("browser negotiation: %d %s", browser.Code, browser.Header().Get("Location"))
	}
	viewer := httptest.NewRecorder()
	router.ServeHTTP(viewer, httptest.NewRequest(http.MethodGet, recordPath+"/", nil))
	if viewer.Code != http.StatusOK || viewer.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("viewer webpage: %d %s", viewer.Code, viewer.Header().Get("Content-Type"))
	}

	// Agent negotiation: default */* → durable JSON record.
	agent := httptest.NewRecorder()
	router.ServeHTTP(agent, httptest.NewRequest(http.MethodGet, recordPath, nil))
	if agent.Code != http.StatusOK || agent.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("agent negotiation: %d %s", agent.Code, agent.Header().Get("Content-Type"))
	}

	// Explicit JSON wins even when HTML is also mentioned with lower q.
	jsonClient := httptest.NewRecorder()
	jsonReq := httptest.NewRequest(http.MethodGet, recordPath, nil)
	jsonReq.Header.Set("Accept", "text/html;q=0.4,application/json;q=1.0")
	router.ServeHTTP(jsonClient, jsonReq)
	if jsonClient.Header().Get("Content-Type") != "application/json; charset=utf-8" {
		t.Fatalf("explicit json: %s", jsonClient.Header().Get("Content-Type"))
	}
}

func TestAcceptPrefersHTML(t *testing.T) {
	cases := []struct {
		accept string
		want   bool
	}{
		{"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8", true},
		{"application/json", false},
		{"*/*", false},
		{"", false},
		{"text/html;q=0.4,application/json;q=1.0", false},
		{"application/json;q=0.2,text/html;q=0.9", true},
	}
	for _, tc := range cases {
		if got := acceptPrefersHTML(tc.accept); got != tc.want {
			t.Fatalf("acceptPrefersHTML(%q) = %v, want %v", tc.accept, got, tc.want)
		}
	}
}