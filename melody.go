package melody

import (
	"github.com/gorilla/websocket"
	"net/http"
)

type handleMessageFunc func(*Session, []byte)
type handleErrorFunc func(*Session, error)
type handleSessionFunc func(*Session)
type filterFunc func(*Session) bool

type Melody struct {
	Config               *Config
	Upgrader             *websocket.Upgrader
	messageHandler       handleMessageFunc
	messageHandlerBinary handleMessageFunc
	errorHandler         handleErrorFunc
	connectHandler       handleSessionFunc
	disconnectHandler    handleSessionFunc
	hub                  *hub
}

// Returns a new melody instance with default Upgrader and Config.
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
		hub:                  hub,
	}
}

// Fires fn when a session connects.
func (m *Melody) HandleConnect(fn func(*Session)) {
	m.connectHandler = fn
}

// Fires fn when a session disconnects.
func (m *Melody) HandleDisconnect(fn func(*Session)) {
	m.disconnectHandler = fn
}

// Callback when a text message comes in.
func (m *Melody) HandleMessage(fn func(*Session, []byte)) {
	m.messageHandler = fn
}

// Callback when a binary message comes in.
func (m *Melody) HandleMessageBinary(fn func(*Session, []byte)) {
	m.messageHandlerBinary = fn
}

// Fires when a session has an error.
func (m *Melody) HandleError(fn func(*Session, error)) {
	m.errorHandler = fn
}

// Handles http requests and upgrades them to websocket connections.
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

// Broadcasts a text message to all sessions.
func (m *Melody) Broadcast(msg []byte) {
	message := &envelope{t: websocket.TextMessage, msg: msg}
	m.hub.broadcast <- message
}

// Broadcasts a text message to all sessions that fn returns true for.
func (m *Melody) BroadcastFilter(msg []byte, fn func(*Session) bool) {
	message := &envelope{t: websocket.TextMessage, msg: msg, filter: fn}
	m.hub.broadcast <- message
}

// Broadcasts a text message to all sessions except session s.
func (m *Melody) BroadcastOthers(msg []byte, s *Session) {
	m.BroadcastFilter(msg, func(q *Session) bool {
		return s != q
	})
}

// Broadcasts a binary message to all sessions.
func (m *Melody) BroadcastBinary(msg []byte) {
	message := &envelope{t: websocket.BinaryMessage, msg: msg}
	m.hub.broadcast <- message
}

// Broadcasts a binary message to all sessions that fn returns true for.
func (m *Melody) BroadcastBinaryFilter(msg []byte, fn func(*Session) bool) {
	message := &envelope{t: websocket.BinaryMessage, msg: msg, filter: fn}
	m.hub.broadcast <- message
}

// Broadcasts a binary message to all sessions except session s.
func (m *Melody) BroadcastBinaryOthers(msg []byte, s *Session) {
	m.BroadcastBinaryFilter(msg, func(q *Session) bool {
		return s != q
	})
}

// Closes the melody instance and all connected sessions.
func (m *Melody) Close() {
	m.hub.exit <- true
}
