package vision

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/proto"
)

// Chrome owns one screencast per page. Viewers subscribe to one session-owned
// pump; ending one subscription must never stop the other viewers' capture.
type screencastHub struct {
	page           *rod.Page
	cancel         context.CancelFunc
	done           chan struct{}
	mu             sync.Mutex
	clients        map[chan []byte]struct{}
	closed         bool
	last           []byte
	quality, every int
}

func (h *screencastHub) broadcast(data []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.last = data
	for client := range h.clients {
		select {
		case client <- data:
		default:
			// One latest frame, not a queue of stale frames behind a slow reader.
			select {
			case <-client:
			default:
			}
			select {
			case client <- data:
			default:
			}
		}
	}
}

func closeScreencastClient(client chan []byte) {
	select {
	case <-client:
	default:
	}
	close(client)
}

func (h *screencastHub) closeClients() {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.closed {
		return
	}
	h.closed = true
	for client := range h.clients {
		closeScreencastClient(client)
		delete(h.clients, client)
	}
}

func (h *screencastHub) stop() {
	h.cancel()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := (proto.PageStopScreencast{}).Call(h.page.Context(ctx)); err != nil {
		log.Print("[vision] screencast stop incomplete: protocol_stop_failed")
	}
	select {
	case <-h.done:
	case <-time.After(2 * time.Second):
		log.Print("[vision] screencast listener stop incomplete: listener_timeout")
	}
	h.closeClients()
}

// Caller holds Session.mu; this also prevents a replacement pump starting while
// a previous Page.stopScreencast is still being sent to the same Chrome target.
func (s *Session) stopScreencastLocked() {
	if s.screencast != nil {
		s.screencast.stop()
		s.screencast = nil
	}
}

func (s *Session) CDPScreencast(caller context.Context, quality, everyNthFrame int) (<-chan []byte, func(), error) {
	if err := caller.Err(); err != nil {
		return nil, nil, err
	}
	if quality <= 0 || quality > 100 {
		quality = 60
	}
	if everyNthFrame <= 0 {
		everyNthFrame = 1
	}
	s.mu.Lock()
	if s.page == nil {
		s.mu.Unlock()
		return nil, nil, fmt.Errorf("session closed")
	}
	hub := s.screencast
	if hub != nil {
		hub.mu.Lock()
		closed := hub.closed
		hub.mu.Unlock()
		if closed {
			s.stopScreencastLocked()
			hub = nil
		}
	}
	if hub != nil && (hub.quality != quality || hub.every != everyNthFrame) {
		s.mu.Unlock()
		return nil, nil, fmt.Errorf("screencast profile differs from active pump")
	}
	client := make(chan []byte, 1)
	if hub == nil {
		ctx, cancel := context.WithCancel(context.Background())
		hub = &screencastHub{page: s.page, cancel: cancel, done: make(chan struct{}), clients: map[chan []byte]struct{}{client: {}}, quality: quality, every: everyNthFrame}
		wait := hub.page.Context(ctx).EachEvent(func(event *proto.PageScreencastFrame) {
			ackCtx, ackCancel := context.WithTimeout(ctx, time.Second)
			err := (proto.PageScreencastFrameAck{SessionID: event.SessionID}).Call(hub.page.Context(ackCtx))
			ackCancel()
			if err != nil {
				cancel()
				return
			}
			hub.broadcast(event.Data)
		})
		go func() { defer close(hub.done); defer hub.closeClients(); wait() }()
		startCtx, startCancel := context.WithTimeout(caller, 3*time.Second)
		err := (proto.PageStartScreencast{Format: proto.PageStartScreencastFormatJpeg, Quality: &quality, EveryNthFrame: &everyNthFrame}).Call(hub.page.Context(startCtx))
		startCancel()
		if err != nil {
			hub.stop()
			s.mu.Unlock()
			return nil, nil, fmt.Errorf("start screencast: protocol_start_failed")
		}
		s.screencast = hub
	} else {
		hub.mu.Lock()
		hub.clients[client] = struct{}{}
		if hub.last != nil {
			client <- hub.last
		}
		hub.mu.Unlock()
	}
	s.mu.Unlock()
	stopped := make(chan struct{})
	var once sync.Once
	stop := func() {
		once.Do(func() {
			close(stopped)
			s.mu.Lock()
			defer s.mu.Unlock()
			hub.mu.Lock()
			if _, ok := hub.clients[client]; ok {
				delete(hub.clients, client)
				closeScreencastClient(client)
			}
			remaining := len(hub.clients)
			hub.mu.Unlock()
			if remaining == 0 && s.screencast == hub {
				s.stopScreencastLocked()
			}
		})
	}
	go func() {
		select {
		case <-caller.Done():
			stop()
		case <-stopped:
		case <-hub.done:
			stop()
		}
	}()
	return client, stop, nil
}
