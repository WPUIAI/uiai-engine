package evidenceregistry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFocusaTokenFileTransportInjectsWithoutMutatingCallerRequest(t *testing.T) {
	const token = "renewable-test-token"
	path := filepath.Join(t.TempDir(), "focusa-token")
	if err := os.WriteFile(path, []byte(token+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Authorization") != "Bearer "+token {
			t.Fatal("expected bearer token from credential file")
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	request, err := http.NewRequest(http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := syncHTTPClient(FocusaSyncConfig{TokenFile: path}).Do(request)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if request.Header.Get("Authorization") != "" {
		t.Fatal("credential transport mutated caller request")
	}
}

func TestRegistryEventHubPublishesRevisionAndDegradedTruth(t *testing.T) {
	manager, err := NewManager(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer manager.Close()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := manager.RegistryEvents(ctx)
	manager.eventHub().record(FocusaSyncResult{Projects: 1, ChangedProjects: 1, Items: 25, Results: []ProviderGraphResult{{ProjectRef: "project:focusa", Revision: 2, Changed: true}}}, "focusa_event", nil)
	select {
	case event := <-events:
		if event.Schema != RegistryEventSchemaV1 || event.Status != "completed" || event.Freshness != "live" || event.Trigger != "focusa_event" || event.ChangedProjects != 1 {
			t.Fatalf("unexpected live event: %#v", event)
		}
	case <-time.After(time.Second):
		t.Fatal("registry event was not delivered")
	}
	manager.eventHub().record(FocusaSyncResult{}, "reconciliation", ErrIndexUnavailable)
	status := manager.RegistrySyncStatus()
	if status.Status != "degraded" || status.Freshness != "stale" || status.LastSuccessAt.IsZero() {
		t.Fatalf("degraded sync truth missing: %#v", status)
	}
}

func TestRelevantFocusaEventRejectsNoiseAndAcceptsGraphChanges(t *testing.T) {
	if relevantFocusaEvent([]byte(`{"event_type":"IntuitionSignalObserved","invalidate":[],"scope":{"project_root":null}}`), []string{t.TempDir()}) {
		t.Fatal("unrelated cognition event triggered provider graph sync")
	}
	if !relevantFocusaEvent([]byte(`{"event_type":"WorkItemUpdated","invalidate":[],"scope":{"project_root":null}}`), []string{t.TempDir()}) {
		t.Fatal("work item event did not trigger provider graph sync")
	}
}

func TestLatestFocusaEventCursorUsesDurableEventID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if req.URL.Path != "/v1/events/recent" || req.URL.Query().Get("limit") != "1" {
			http.NotFound(w, req)
			return
		}
		_, _ = w.Write([]byte(`{"events":[{"id":"event-old"},{"id":"event-latest"}]}`))
	}))
	defer server.Close()
	cursor, err := latestFocusaEventCursor(context.Background(), FocusaSyncConfig{BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	if cursor != "event-latest" {
		t.Fatalf("cursor=%q", cursor)
	}
}
