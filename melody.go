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
	Config            *Config
	Upgrader          *websocket.Upgrader
	MessageHandler    handleMessageFunc
	ErrorHandler      handleErrorFunc
	ConnectHandler    handleSessionFunc
	DisconnectHandler handleSessionFunc
	hub               *hub
}

func Default() *Melody {
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
	}

	hub := newHub()

	go hub.run()

	return &Melody{
		Config:            newConfig(),
		Upgrader:          upgrader,
		MessageHandler:    func(*Session, []byte) {},
		ErrorHandler:      func(*Session, error) {},
		ConnectHandler:    func(*Session) {},
		DisconnectHandler: func(*Session) {},
		hub:               hub,
	}
}

func (m *Melody) HandleConnect(fn handleSessionFunc) {
	m.ConnectHandler = fn
}

func (m *Melody) HandleDisconnect(fn handleSessionFunc) {
	m.DisconnectHandler = fn
}

func (m *Melody) HandleMessage(fn handleMessageFunc) {
	m.MessageHandler = fn
}

func (m *Melody) HandleError(fn handleErrorFunc) {
	m.ErrorHandler = fn
}

func (m *Melody) HandleRequest(w http.ResponseWriter, r *http.Request) error {
	conn, err := m.Upgrader.Upgrade(w, r, nil)

	if err != nil {
		return err
	}

	session := newSession(m.Config, conn)

	m.hub.register <- session

	go m.ConnectHandler(session)

	go session.writePump(m.ErrorHandler)

	session.readPump(m.MessageHandler, m.ErrorHandler)

	m.hub.unregister <- session

	go m.DisconnectHandler(session)

	return nil
}

func (m *Melody) Broadcast(msg []byte) {
	message := &envelope{t: websocket.TextMessage, msg: msg}
	m.hub.broadcast <- message
}

func (m *Melody) BroadcastFilter(fn filterFunc, msg []byte) {
	message := &envelope{t: websocket.TextMessage, msg: msg, filter: fn}
	m.hub.broadcast <- message
}
