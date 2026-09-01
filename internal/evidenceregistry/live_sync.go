package evidenceregistry

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const RegistryEventSchemaV1 = "uiai.evidence_registry_event.v1"

type RegistryEvent struct {
	Schema          string                `json:"schema"`
	EventID         string                `json:"event_id"`
	Kind            string                `json:"kind"`
	Status          string                `json:"status"`
	Trigger         string                `json:"trigger"`
	Projects        uint64                `json:"projects"`
	ChangedProjects uint64                `json:"changed_projects"`
	Items           uint64                `json:"items"`
	Results         []ProviderGraphResult `json:"results,omitempty"`
	Freshness       string                `json:"freshness"`
	ObservedAt      time.Time             `json:"observed_at"`
}

type RegistrySyncStatus struct {
	Schema          string    `json:"schema"`
	Status          string    `json:"status"`
	Freshness       string    `json:"freshness"`
	LastAttemptAt   time.Time `json:"last_attempt_at,omitempty"`
	LastSuccessAt   time.Time `json:"last_success_at,omitempty"`
	LastEventID     string    `json:"last_event_id,omitempty"`
	Projects        uint64    `json:"projects"`
	ChangedProjects uint64    `json:"changed_projects"`
	Items           uint64    `json:"items"`
}

type registryEventHub struct {
	mu          sync.Mutex
	next        uint64
	subscribers map[uint64]chan RegistryEvent
	status      RegistrySyncStatus
}

func newRegistryEventHub() *registryEventHub {
	return &registryEventHub{subscribers: make(map[uint64]chan RegistryEvent), status: RegistrySyncStatus{Schema: "uiai.evidence_registry_sync_status.v1", Status: "unavailable", Freshness: "unavailable"}}
}

func (h *registryEventHub) subscribe(ctx context.Context) <-chan RegistryEvent {
	h.mu.Lock()
	h.next++
	id := h.next
	ch := make(chan RegistryEvent, 8)
	h.subscribers[id] = ch
	status := h.status
	h.mu.Unlock()
	if status.LastEventID != "" {
		ch <- RegistryEvent{Schema: RegistryEventSchemaV1, EventID: status.LastEventID, Kind: "snapshot", Status: status.Status, Freshness: status.Freshness, Projects: status.Projects, ChangedProjects: status.ChangedProjects, Items: status.Items, ObservedAt: status.LastAttemptAt}
	}
	go func() {
		<-ctx.Done()
		h.mu.Lock()
		delete(h.subscribers, id)
		close(ch)
		h.mu.Unlock()
	}()
	return ch
}

func (h *registryEventHub) record(result FocusaSyncResult, trigger string, syncErr error) RegistryEvent {
	now := time.Now().UTC()
	h.mu.Lock()
	h.next++
	event := RegistryEvent{Schema: RegistryEventSchemaV1, EventID: fmt.Sprintf("registry-%d-%d", now.UnixNano(), h.next), Kind: "registry_revision", Trigger: trigger, Projects: result.Projects, ChangedProjects: result.ChangedProjects, Items: result.Items, Results: result.Results, ObservedAt: now}
	h.status.LastAttemptAt = now
	if syncErr != nil {
		event.Status, event.Freshness = "degraded", "stale"
		h.status.Status, h.status.Freshness = event.Status, event.Freshness
	} else {
		event.Status, event.Freshness = "completed", "live"
		h.status.Status, h.status.Freshness = event.Status, event.Freshness
		h.status.LastSuccessAt = now
		h.status.Projects, h.status.ChangedProjects, h.status.Items = result.Projects, result.ChangedProjects, result.Items
	}
	h.status.LastEventID = event.EventID
	for _, subscriber := range h.subscribers {
		select {
		case subscriber <- event:
		default:
		}
	}
	h.mu.Unlock()
	return event
}

func (h *registryEventHub) snapshot() RegistrySyncStatus {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.status
}

func (m *Manager) SyncAndPublish(ctx context.Context, cfg FocusaSyncConfig, trigger string) (FocusaSyncResult, error) {
	m.syncMu.Lock()
	defer m.syncMu.Unlock()
	result, err := m.SyncFocusa(ctx, cfg)
	m.eventHub().record(result, trigger, err)
	return result, err
}

func (m *Manager) RegistryEvents(ctx context.Context) <-chan RegistryEvent {
	return m.eventHub().subscribe(ctx)
}

func (m *Manager) RegistrySyncStatus() RegistrySyncStatus {
	return m.eventHub().snapshot()
}

func (m *Manager) eventHub() *registryEventHub {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.events == nil {
		m.events = newRegistryEventHub()
	}
	return m.events
}

type ContinuousSync struct {
	cancel context.CancelFunc
	done   chan struct{}
}

func StartContinuousSync(manager *Manager, cfg FocusaSyncConfig, interval time.Duration) (*ContinuousSync, error) {
	if manager == nil || interval < time.Second || interval > time.Hour {
		return nil, ErrConfig
	}
	ctx, cancel := context.WithCancel(context.Background())
	syncer := &ContinuousSync{cancel: cancel, done: make(chan struct{})}
	go syncer.run(ctx, manager, cfg, interval)
	return syncer, nil
}

func (s *ContinuousSync) Close() {
	if s == nil {
		return
	}
	s.cancel()
	<-s.done
}

func (s *ContinuousSync) run(ctx context.Context, manager *Manager, cfg FocusaSyncConfig, interval time.Duration) {
	defer close(s.done)
	triggers := make(chan string, 1)
	go tailFocusaEvents(ctx, cfg, triggers)
	queueTrigger(triggers, "startup")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			queueTrigger(triggers, "reconciliation")
		case trigger := <-triggers:
			syncCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			_, err := manager.SyncAndPublish(syncCtx, cfg, trigger)
			cancel()
			if err != nil && ctx.Err() == nil {
				log.Printf("[evidence-registry] continuous sync degraded trigger=%s: %v", trigger, err)
			}
		}
	}
}

func tailFocusaEvents(ctx context.Context, cfg FocusaSyncConfig, triggers chan<- string) {
	backoff := time.Second
	for ctx.Err() == nil {
		cursor, err := latestFocusaEventCursor(ctx, cfg)
		if err == nil && cursor != "" {
			err = consumeFocusaEvents(ctx, cfg, cursor, triggers)
		}
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			log.Printf("[evidence-registry] Focusa event stream degraded; bounded reconciliation remains active: %v", err)
		}
		timer := time.NewTimer(backoff)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
		if err != nil && backoff < 30*time.Second {
			backoff *= 2
		} else {
			backoff = time.Second
		}
	}
}

func latestFocusaEventCursor(ctx context.Context, cfg FocusaSyncConfig) (string, error) {
	client := syncHTTPClient(cfg)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(cfg.BaseURL, "/")+"/v1/events/recent?limit=1", nil)
	if err != nil {
		return "", err
	}
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Focusa recent events HTTP %d", response.StatusCode)
	}
	var payload struct {
		Events []struct {
			ID string `json:"id"`
		} `json:"events"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, 1<<20)).Decode(&payload); err != nil || len(payload.Events) == 0 {
		return "", fmt.Errorf("Focusa recent events unavailable")
	}
	return payload.Events[len(payload.Events)-1].ID, nil
}

func consumeFocusaEvents(ctx context.Context, cfg FocusaSyncConfig, cursor string, triggers chan<- string) error {
	streamURL := strings.TrimRight(cfg.BaseURL, "/") + "/v1/events/stream?cursor=" + url.QueryEscape(cursor)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return err
	}
	response, err := syncHTTPClient(cfg).Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		return fmt.Errorf("Focusa event stream unavailable")
	}
	scanner := bufio.NewScanner(response.Body)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	eventName, data := "", ""
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "event:"):
			eventName = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			data += strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		case line == "":
			if eventName == "focusa_event" && relevantFocusaEvent([]byte(data), cfg.AllowedRootPrefixes) {
				queueTrigger(triggers, "focusa_event")
			}
			eventName, data = "", ""
		}
	}
	return scanner.Err()
}

func relevantFocusaEvent(data []byte, allowedRoots []string) bool {
	var envelope struct {
		EventType  string `json:"event_type"`
		Invalidate []any  `json:"invalidate"`
		Scope      struct {
			ProjectRoot string `json:"project_root"`
		} `json:"scope"`
	}
	if json.Unmarshal(data, &envelope) != nil {
		return false
	}
	if envelope.Scope.ProjectRoot != "" {
		roots, err := canonicalAllowedRoots(allowedRoots)
		if err != nil {
			return false
		}
		if _, err := allowedProviderRoot(envelope.Scope.ProjectRoot, roots); err != nil {
			return false
		}
	}
	if len(envelope.Invalidate) > 0 {
		return true
	}
	kind := strings.ToLower(envelope.EventType)
	for _, token := range []string{"project", "task", "workitem", "work_item", "workpoint", "workset", "callgraph", "acceptance", "completion", "closure", "provider", "attachment"} {
		if strings.Contains(kind, token) {
			return true
		}
	}
	return false
}

func queueTrigger(triggers chan<- string, trigger string) {
	select {
	case triggers <- trigger:
	default:
	}
}

func syncHTTPClient(cfg FocusaSyncConfig) *http.Client {
	if cfg.HTTPClient != nil {
		return cfg.HTTPClient
	}
	return &http.Client{Timeout: 20 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}
}
