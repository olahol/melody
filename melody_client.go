package melody

import (
	"github.com/gorilla/websocket"
	"net/url"
)

// MelodyClient implements a websocket connection.
type MelodyClient struct {
	Config                   *Config
	Upgrader                 *websocket.Upgrader
	messageHandler           handleClientMessageFunc
	messageHandlerBinary     handleClientMessageFunc
	messageSentHandler       handleClientMessageFunc
	messageSentHandlerBinary handleClientMessageFunc
	errorHandler             handleClientErrorFunc
	closeHandler             handleClientCloseFunc
	connectHandler           handleClientConnectionState
	disconnectHandler        handleClientConnectionState
	pongHandler              handleClientConnectionState
	hub                      *clientHub
}

type handleClientConnection func(conn *websocket.Conn)
type handleClientConnectionState func()
type handleClientMessageFunc func([]byte)
type handleClientErrorFunc func(error)
type handleClientCloseFunc func(int, string) error

// TODO document
func NewClient(url url.URL) *MelodyClient {

	c := &MelodyClient{
		Config:                   newConfig(),
		messageHandler:           func([]byte) {},
		messageHandlerBinary:     func([]byte) {},
		messageSentHandler:       func([]byte) {},
		messageSentHandlerBinary: func([]byte) {},
		errorHandler:             func(error) {},
		closeHandler:             func(i int, s string) error { return nil },
		connectHandler:           func() {},
		disconnectHandler:        func() {},
		pongHandler:              func() {},
	}
	c.hub = newClientHub(url, c)

	return c
}

func (m *MelodyClient) Connect() {
	m.hub.run()
}

// HandleConnect fires fn when a session connects.
func (m *MelodyClient) HandleConnect(fn func()) {
	m.connectHandler = fn
}

// HandleDisconnect fires fn when a session disconnects.
func (m *MelodyClient) HandleDisconnect(fn func()) {
	m.disconnectHandler = fn
}

// HandlePong fires fn when a pong is received from a session.
func (m *MelodyClient) HandlePong(fn func()) {
	m.pongHandler = fn
}

// HandleMessage fires fn when a text message comes in.
func (m *MelodyClient) HandleMessage(fn func([]byte)) {
	m.messageHandler = fn
}

// HandleMessageBinary fires fn when a binary message comes in.
func (m *MelodyClient) HandleMessageBinary(fn func([]byte)) {
	m.messageHandlerBinary = fn
}

// HandleSentMessage fires fn when a text message is successfully sent.
func (m *MelodyClient) HandleSentMessage(fn func([]byte)) {
	m.messageSentHandler = fn
}

// HandleSentMessageBinary fires fn when a binary message is successfully sent.
func (m *MelodyClient) HandleSentMessageBinary(fn func([]byte)) {
	m.messageSentHandlerBinary = fn
}

// HandleError fires fn when a session has an error.
func (m *MelodyClient) HandleError(fn func(error)) {
	m.errorHandler = fn
}

func (m *MelodyClient) Send(msg []byte) error {
	if m.hub.closed() {
		return ErrClosed
	}

	m.hub.sendToServer <- &envelope{t: websocket.TextMessage, msg: msg}
	return nil
}

func (m *MelodyClient) SendBinary(msg []byte) error {
	if m.hub.closed() {
		return ErrClosed
	}

	m.hub.sendToServer <- &envelope{t: websocket.BinaryMessage, msg: msg}
	return nil
}

// HandleClose sets the handler for close messages received from the session.
// The code argument to h is the received close code or CloseNoStatusReceived
// if the close message is empty. The default close handler sends a close frame
// back to the session.
//
// The application must read the connection to process close messages as
// described in the section on Control Frames above.
//
// The connection read methods return a CloseError when a close frame is
// received. Most applications should handle close messages as part of their
// normal error handling. Applications should only set a close handler when the
// application must perform some action before sending a close frame back to
// the session.
func (m *MelodyClient) HandleClose(fn func(int, string) error) {
	if fn != nil {
		m.closeHandler = fn
	}
}

// Close closes the melody instance and all connected sessions.
func (m *MelodyClient) Close() error {
	if m.hub.closed() {
		return ErrClosed
	}

	m.hub.exit <- &envelope{t: websocket.CloseMessage, msg: []byte{}}

	return nil
}

// CloseWithMsg closes the melody instance with the given close payload and all connected sessions.
// Use the FormatCloseMessage function to format a proper close message payload.
func (m *MelodyClient) CloseWithMsg(msg []byte) error {
	if m.hub.closed() {
		return ErrClosed
	}

	m.hub.exit <- &envelope{t: websocket.CloseMessage, msg: msg}

	return nil
}

// IsClosed returns the status of the melody instance.
func (m *MelodyClient) IsClosed() bool {
	return m.hub.closed()
}
