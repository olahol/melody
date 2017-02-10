package melody

import (
	"errors"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
)

type handleMessageFunc func(*Session, []byte)
type handleErrorFunc func(*Session, error)
type handleSessionFunc func(*Session)
type filterFunc func(*Session) bool

// Melody implements a websocket manager.
type Melody struct {
	Config                   *Config
	Upgrader                 *websocket.Upgrader
	messageHandler           handleMessageFunc
	messageHandlerBinary     handleMessageFunc
	messageSentHandler       handleMessageFunc
	messageSentHandlerBinary handleMessageFunc
	errorHandler             handleErrorFunc
	connectHandler           handleSessionFunc
	disconnectHandler        handleSessionFunc
	pongHandler              handleSessionFunc
	hub                      *hub
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
		Config:                   newConfig(),
		Upgrader:                 upgrader,
		messageHandler:           func(*Session, []byte) {},
		messageHandlerBinary:     func(*Session, []byte) {},
		messageSentHandler:       func(*Session, []byte) {},
		messageSentHandlerBinary: func(*Session, []byte) {},
		errorHandler:             func(*Session, error) {},
		connectHandler:           func(*Session) {},
		disconnectHandler:        func(*Session) {},
		pongHandler:              func(*Session) {},
		hub:                      hub,
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

// HandleSentMessage fires fn when a text message is successfully sent.
func (m *Melody) HandleSentMessage(fn func(*Session, []byte)) {
	m.messageSentHandler = fn
}

// HandleSentMessageBinary fires fn when a binary message is successfully sent.
func (m *Melody) HandleSentMessageBinary(fn func(*Session, []byte)) {
	m.messageSentHandler = fn
}

// HandleError fires fn when a session has an error.
func (m *Melody) HandleError(fn func(*Session, error)) {
	m.errorHandler = fn
}

// HandleRequest upgrades http requests to websocket connections and dispatches them to be handled by the melody instance.
func (m *Melody) HandleRequest(w http.ResponseWriter, r *http.Request) error {
	return m.HandleRequestWithKeys(w, r, nil)
}

// HandleRequestWithKeys does the same as HandleRequest but populates session.Keys with keys.
func (m *Melody) HandleRequestWithKeys(w http.ResponseWriter, r *http.Request, keys map[string]interface{}) error {
	if m.hub.closed() {
		return errors.New("Melody instance is closed.")
	}

	conn, err := m.Upgrader.Upgrade(w, r, nil)

	if err != nil {
		return err
	}

	session := &Session{
		Request: r,
		Keys:    keys,
		conn:    conn,
		output:  make(chan *envelope, m.Config.MessageBufferSize),
		melody:  m,
		open:    true,
		rwmutex: &sync.RWMutex{},
	}

	m.hub.register <- session

	m.connectHandler(session)

	go session.writePump()

	session.readPump()

	if !m.hub.closed() {
		m.hub.unregister <- session
	}

	session.close()

	m.disconnectHandler(session)

	return nil
}

// Broadcast broadcasts a text message to all sessions.
func (m *Melody) Broadcast(msg []byte) error {
	if m.hub.closed() {
		return errors.New("Melody instance is closed.")
	}

	message := &envelope{t: websocket.TextMessage, msg: msg}
	m.hub.broadcast <- message

	return nil
}

// BroadcastFilter broadcasts a text message to all sessions that fn returns true for.
func (m *Melody) BroadcastFilter(msg []byte, fn func(*Session) bool) error {
	if m.hub.closed() {
		return errors.New("Melody instance is closed.")
	}

	message := &envelope{t: websocket.TextMessage, msg: msg, filter: fn}
	m.hub.broadcast <- message

	return nil
}

// BroadcastOthers broadcasts a text message to all sessions except session s.
func (m *Melody) BroadcastOthers(msg []byte, s *Session) error {
	return m.BroadcastFilter(msg, func(q *Session) bool {
		return s != q
	})
}

// BroadcastBinary broadcasts a binary message to all sessions.
func (m *Melody) BroadcastBinary(msg []byte) error {
	if m.hub.closed() {
		return errors.New("Melody instance is closed.")
	}

	message := &envelope{t: websocket.BinaryMessage, msg: msg}
	m.hub.broadcast <- message

	return nil
}

// BroadcastBinaryFilter broadcasts a binary message to all sessions that fn returns true for.
func (m *Melody) BroadcastBinaryFilter(msg []byte, fn func(*Session) bool) error {
	if m.hub.closed() {
		return errors.New("Melody instance is closed.")
	}

	message := &envelope{t: websocket.BinaryMessage, msg: msg, filter: fn}
	m.hub.broadcast <- message

	return nil
}

// BroadcastBinaryOthers broadcasts a binary message to all sessions except session s.
func (m *Melody) BroadcastBinaryOthers(msg []byte, s *Session) error {
	return m.BroadcastBinaryFilter(msg, func(q *Session) bool {
		return s != q
	})
}

// Close closes the melody instance and all connected sessions.
func (m *Melody) Close() error {
	if m.hub.closed() {
		return errors.New("Melody instance is already closed.")
	}

	m.hub.exit <- true

	return nil
}

// Len return the number of connected sessions.
func (m *Melody) Len() int {
	return m.hub.len()
}
