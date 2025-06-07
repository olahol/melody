package melody

import (
	"sync"
	"sync/atomic"
)

type hub struct {
	mu       sync.RWMutex
	sessions map[*Session]struct{}
	open     atomic.Bool
}

func newHub() *hub {
	hub := &hub{
		sessions: make(map[*Session]struct{}),
	}
	hub.open.Store(true)
	return hub
}

func (h *hub) closed() bool {
	return !h.open.Load()
}

func (h *hub) len() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return len(h.sessions)
}

func (h *hub) all() []*Session {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make([]*Session, 0, len(h.sessions))
	for s := range h.sessions {
		result = append(result, s)
	}
	return result
}

func (h *hub) register(s *Session) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.sessions[s] = struct{}{}
}

func (h *hub) unregister(s *Session) {
	h.mu.Lock()
	defer h.mu.Unlock()

	delete(h.sessions, s)
}

func (h *hub) exit(msg envelope) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for s := range h.sessions {
		s.writeMessage(msg)
		s.Close()
	}
	h.sessions = make(map[*Session]struct{})
	h.open.Store(false)
}

func (h *hub) broadcast(msg envelope) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for s := range h.sessions {
		if msg.filter == nil || msg.filter(s) {
			s.writeMessage(msg)
		}
	}
}
