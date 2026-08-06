// Package realtime fans out project-scoped events to connected WebSocket
// clients. Publishing goes through Redis pub/sub (see publisher.go /
// subscriber.go) rather than a plain in-memory broadcast, so this stays
// correct if the API ever runs as more than one replica behind Nginx.
package realtime

import "sync"

// Sender is the minimum a WebSocket connection needs to support to receive
// broadcasts. Kept separate from *websocket.Conn so Hub can be unit tested
// without a real socket.
type Sender interface {
	Send(data []byte) error
}

type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[Sender]bool // projectID -> connected senders
}

func NewHub() *Hub {
	return &Hub{clients: make(map[string]map[Sender]bool)}
}

func (h *Hub) Register(projectID string, s Sender) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[projectID] == nil {
		h.clients[projectID] = make(map[Sender]bool)
	}
	h.clients[projectID][s] = true
}

func (h *Hub) Unregister(projectID string, s Sender) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients[projectID], s)
	if len(h.clients[projectID]) == 0 {
		delete(h.clients, projectID)
	}
}

// BroadcastLocal delivers data to every sender registered for projectID on
// this process. Called by the Redis subscriber, not directly by callers
// that just want to publish (use Publisher.Publish for that, so the event
// also reaches other replicas).
func (h *Hub) BroadcastLocal(projectID string, data []byte) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for s := range h.clients[projectID] {
		// Best-effort: a dead connection's own read loop will notice the
		// error and unregister it, so a failed send here is not fatal.
		_ = s.Send(data)
	}
}

func (h *Hub) ConnectionCount(projectID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients[projectID])
}
