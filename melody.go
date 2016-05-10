package melody

import (
	"github.com/gorilla/websocket"
	"net/http"
)

type handleMessageFunc func(*Session, []byte)
type handleErrorFunc func(*Session, error)
type handleSessionFunc func(*Session)
type filterFunc func(*Session) bool

// Melody implements a websocket manager.
type Melody struct {
	Config               *Config
	Upgrader             *websocket.Upgrader
	messageHandler       handleMessageFunc
	messageHandlerBinary handleMessageFunc
	errorHandler         handleErrorFunc
	connectHandler       handleSessionFunc
	disconnectHandler    handleSessionFunc
	pongHandler          handleSessionFunc
	hub                  *hub
}

// New creates a new melody instance with default Upgrader and Config.
func New() *Melody {
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	hub := newHub()

	go hub.run()

	return &Melody{
		Config:               newConfig(),
		Upgrader:             upgrader,
		messageHandler:       func(*Session, []byte) {},
		messageHandlerBinary: func(*Session, []byte) {},
		errorHandler:         func(*Session, error) {},
		connectHandler:       func(*Session) {},
		disconnectHandler:    func(*Session) {},
		pongHandler:          func(*Session) {},
		hub:                  hub,
	}
}

// HandleConnect fires fn when a session connects.
func (m *Melody) HandleConnect(fn func(*Session)) {
	m.connectHandler = fn
}

// HandleDisconnect fires fn when a session disconnects.
func (m *Melody) HandleDisconnect(fn func(*Session)) {
	m.disconnectHandler = fn
}

// HandlePong fires fn when a pong is received from a session.
func (m *Melody) HandlePong(fn func(*Session)) {
	m.pongHandler = fn
}

// HandleMessage fires fn when a text message comes in.
func (m *Melody) HandleMessage(fn func(*Session, []byte)) {
	m.messageHandler = fn
}

// HandleMessageBinary fires fn when a binary message comes in.
func (m *Melody) HandleMessageBinary(fn func(*Session, []byte)) {
	m.messageHandlerBinary = fn
}

// HandleError fires fn when a session has an error.
func (m *Melody) HandleError(fn func(*Session, error)) {
	m.errorHandler = fn
}

// HandleRequest upgrades http requests to websocket connections and dispatches them to be handled by the melody instance.
func (m *Melody) HandleRequest(w http.ResponseWriter, r *http.Request) {
	conn, err := m.Upgrader.Upgrade(w, r, nil)

	if err != nil {
		m.errorHandler(nil, err)
		return
	}

	session := &Session{
		Request: r,
		conn:    conn,
		output:  make(chan *envelope, m.Config.MessageBufferSize),
		melody:  m,
	}

	m.hub.register <- session

	go m.connectHandler(session)

	go session.writePump()

	session.readPump()

	if m.hub.open {
		m.hub.unregister <- session
	}

	go m.disconnectHandler(session)
}

// Broadcast broadcasts a text message to all sessions.
func (m *Melody) Broadcast(msg []byte) {
	message := &envelope{t: websocket.TextMessage, msg: msg}
	m.hub.broadcast <- message
}

// BroadcastFilter broadcasts a text message to all sessions that fn returns true for.
func (m *Melody) BroadcastFilter(msg []byte, fn func(*Session) bool) {
	message := &envelope{t: websocket.TextMessage, msg: msg, filter: fn}
	m.hub.broadcast <- message
}

// BroadcastOthers broadcasts a text message to all sessions except session s.
func (m *Melody) BroadcastOthers(msg []byte, s *Session) {
	m.BroadcastFilter(msg, func(q *Session) bool {
		return s != q
	})
}

// BroadcastBinary broadcasts a binary message to all sessions.
func (m *Melody) BroadcastBinary(msg []byte) {
	message := &envelope{t: websocket.BinaryMessage, msg: msg}
	m.hub.broadcast <- message
}

// BroadcastBinaryFilter broadcasts a binary message to all sessions that fn returns true for.
func (m *Melody) BroadcastBinaryFilter(msg []byte, fn func(*Session) bool) {
	message := &envelope{t: websocket.BinaryMessage, msg: msg, filter: fn}
	m.hub.broadcast <- message
}

// BroadcastBinaryOthers broadcasts a binary message to all sessions except session s.
func (m *Melody) BroadcastBinaryOthers(msg []byte, s *Session) {
	m.BroadcastBinaryFilter(msg, func(q *Session) bool {
		return s != q
	})
}

// Close closes the melody instance and all connected sessions.
func (m *Melody) Close() {
	m.hub.exit <- true
}
