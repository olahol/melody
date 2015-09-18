package melody

type hub struct {
	sessions   map[*Session]bool
	broadcast  chan *envelope
	register   chan *Session
	unregister chan *Session
	exit       chan bool
	open       bool
}

func newHub() *hub {
	return &hub{
		sessions:   make(map[*Session]bool),
		broadcast:  make(chan *envelope),
		register:   make(chan *Session),
		unregister: make(chan *Session),
		exit:       make(chan bool),
		open:       true,
	}
}

func (h *hub) run() {
loop:
	for {
		select {
		case s := <-h.register:
			h.sessions[s] = true
		case s := <-h.unregister:
			if _, ok := h.sessions[s]; ok {
				delete(h.sessions, s)
				s.conn.Close()
				close(s.output)
			}
		case m := <-h.broadcast:
			for s := range h.sessions {
				if m.filter != nil {
					if m.filter(s) {
						s.writeMessage(m)
					}
				} else {
					s.writeMessage(m)
				}
			}
		case <-h.exit:
			for s := range h.sessions {
				delete(h.sessions, s)
				s.conn.Close()
				close(s.output)
			}
			h.open = false
			break loop
		}
	}
}
