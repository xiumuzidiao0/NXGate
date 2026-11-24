package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"aimili-vpngate-go/pkg/stats"
)

const defaultSSEStatusInterval = 2 * time.Second

type SSEHub struct {
	server         *Server
	statusInterval time.Duration

	mu      sync.Mutex
	clients map[chan []byte]struct{}
	stopCh  chan struct{}
}

func NewSSEHub(s *Server) *SSEHub {
	return newSSEHubWithInterval(s, defaultSSEStatusInterval)
}

func newSSEHubWithInterval(s *Server, interval time.Duration) *SSEHub {
	if interval <= 0 {
		interval = defaultSSEStatusInterval
	}
	return &SSEHub{
		server:         s,
		statusInterval: interval,
		clients:        make(map[chan []byte]struct{}),
	}
}

func (hub *SSEHub) subscribe() chan []byte {
	ch := make(chan []byte, 64)

	hub.mu.Lock()
	hub.clients[ch] = struct{}{}
	if hub.stopCh == nil {
		stopCh := make(chan struct{})
		hub.stopCh = stopCh
		go hub.runBroadcaster(stopCh)
	}
	hub.mu.Unlock()

	return ch
}

func (hub *SSEHub) unsubscribe(ch chan []byte) {
	hub.mu.Lock()
	delete(hub.clients, ch)
	if len(hub.clients) == 0 && hub.stopCh != nil {
		close(hub.stopCh)
		hub.stopCh = nil
	}
	hub.mu.Unlock()
}

func (hub *SSEHub) broadcast(owner chan struct{}, frame []byte) {
	hub.mu.Lock()
	if hub.stopCh != owner {
		hub.mu.Unlock()
		return
	}
	for ch := range hub.clients {
		select {
		case ch <- frame:
		default:
		}
	}
	hub.mu.Unlock()
}

func (hub *SSEHub) runBroadcaster(stopCh chan struct{}) {
	logChan := stats.GetRingLog().Subscribe()
	defer stats.GetRingLog().Unsubscribe(logChan)

	ticker := time.NewTicker(hub.statusInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stopCh:
			return

		case entry, ok := <-logChan:
			if !ok {
				return
			}
			data, err := json.Marshal(entry)
			if err == nil {
				frame := fmt.Appendf(nil, "event: log\ndata: %s\n\n", data)
				hub.broadcast(stopCh, frame)
			}

		case <-ticker.C:
			state := hub.server.buildStatusResponse()
			data, err := json.Marshal(state)
			if err == nil {
				frame := fmt.Appendf(nil, "event: status\ndata: %s\n\n", data)
				hub.broadcast(stopCh, frame)
			}
		}
	}
}

func (hub *SSEHub) HandleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Initial ping
	if _, err := fmt.Fprintf(w, ": connected\n\n"); err != nil {
		return
	}
	flusher.Flush()

	clientCh := hub.subscribe()
	defer hub.unsubscribe(clientCh)

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return

		case frame := <-clientCh:
			if _, err := w.Write(frame); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}
