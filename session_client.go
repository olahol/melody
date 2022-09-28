package melody

import (
	"github.com/gorilla/websocket"
	"net"
	"net/http"
	"net/url"
	"time"
)

// ClientSession wrapper around websocket connections.
type ClientSession struct {
	Request    *http.Request
	Keys       map[string]interface{}
	conn       *websocket.Conn
	output     chan *envelope
	outputDone chan struct{}
	melody     *MelodyClient
	open       bool
}

func (s *ClientSession) connect(url url.URL) {
	var err error
	s.conn, _, err = websocket.DefaultDialer.Dial(url.String(), nil)
	if err != nil {
		s.melody.errorHandler(err)
		return
	}
	defer s.conn.Close()

	s.melody.connectHandler()

	s.conn.SetReadLimit(s.melody.Config.MaxMessageSize)
	s.conn.SetReadDeadline(time.Now().Add(s.melody.Config.PongWait))

	s.conn.SetPongHandler(func(string) error {
		s.conn.SetReadDeadline(time.Now().Add(s.melody.Config.PongWait))
		s.melody.pongHandler()
		return nil
	})

	if s.melody.closeHandler != nil {
		s.conn.SetCloseHandler(func(code int, text string) error {
			return s.melody.closeHandler(code, text)
		})
	}

	for {
		t, message, err := s.conn.ReadMessage()

		if err != nil {
			s.melody.errorHandler(err)
			break
		}

		if t == websocket.TextMessage {
			s.melody.messageHandler(message)
		}

		if t == websocket.BinaryMessage {
			s.melody.messageHandlerBinary(message)
		}
	}
}

func (s *ClientSession) writeMessage(message *envelope) {
	if s.closed() {
		s.melody.errorHandler(ErrWriteClosed)
		return
	}

	select {
	case s.output <- message:
	default:
		s.melody.errorHandler(ErrMessageBufferFull)
	}
}

func (s *ClientSession) writeRaw(message *envelope) error {
	if s.closed() {
		return ErrWriteClosed
	}

	s.conn.SetWriteDeadline(time.Now().Add(s.melody.Config.WriteWait))
	err := s.conn.WriteMessage(message.t, message.msg)

	if err != nil {
		return err
	}

	return nil
}

func (s *ClientSession) closed() bool {
	return !s.open
}

func (s *ClientSession) close() {
	open := s.open
	s.open = false
	if open {
		s.conn.Close()
		close(s.outputDone)
	}
}

func (s *ClientSession) ping() {
	s.writeRaw(&envelope{t: websocket.PingMessage, msg: []byte{}})
}

func (s *ClientSession) writePump() {
	ticker := time.NewTicker(s.melody.Config.PingPeriod)
	defer ticker.Stop()

loop:
	for {
		select {
		case msg := <-s.output:
			err := s.writeRaw(msg)

			if err != nil {
				s.melody.errorHandler(err)
				break loop
			}

			if msg.t == websocket.CloseMessage {
				break loop
			}

			if msg.t == websocket.TextMessage {
				s.melody.messageSentHandler(msg.msg)
			}

			if msg.t == websocket.BinaryMessage {
				s.melody.messageSentHandlerBinary(msg.msg)
			}
		case <-ticker.C:
			s.ping()
		case _, ok := <-s.outputDone:
			if !ok {
				break loop
			}
		}
	}
}

// Write writes message to session.
func (s *ClientSession) Write(msg []byte) error {
	if s.closed() {
		return ErrSessionClosed
	}

	s.writeMessage(&envelope{t: websocket.TextMessage, msg: msg})

	return nil
}

// WriteBinary writes a binary message to session.
func (s *ClientSession) WriteBinary(msg []byte) error {
	if s.closed() {
		return ErrSessionClosed
	}

	s.writeMessage(&envelope{t: websocket.BinaryMessage, msg: msg})

	return nil
}

// Close closes session.
func (s *ClientSession) Close() error {
	if s.closed() {
		return ErrSessionClosed
	}

	s.writeMessage(&envelope{t: websocket.CloseMessage, msg: []byte{}})

	return nil
}

// CloseWithMsg closes the session with the provided payload.
// Use the FormatCloseMessage function to format a proper close message payload.
func (s *ClientSession) CloseWithMsg(msg []byte) error {
	if s.closed() {
		return ErrSessionClosed
	}

	s.writeMessage(&envelope{t: websocket.CloseMessage, msg: msg})

	return nil
}

// Set is used to store a new key/value pair exclusively for this session.
// It also lazy initializes s.Keys if it was not used previously.
func (s *ClientSession) Set(key string, value interface{}) {
	if s.Keys == nil {
		s.Keys = make(map[string]interface{})
	}

	s.Keys[key] = value
}

// Get returns the value for the given key, ie: (value, true).
// If the value does not exists it returns (nil, false)
func (s *ClientSession) Get(key string) (value interface{}, exists bool) {
	if s.Keys != nil {
		value, exists = s.Keys[key]
	}

	return
}

// MustGet returns the value for the given key if it exists, otherwise it panics.
func (s *ClientSession) MustGet(key string) interface{} {
	if value, exists := s.Get(key); exists {
		return value
	}

	panic("Key \"" + key + "\" does not exist")
}

// IsClosed returns the status of the connection.
func (s *ClientSession) IsClosed() bool {
	return s.closed()
}

// LocalAddr returns the local addr of the connection.
func (s *ClientSession) LocalAddr() net.Addr {
	return s.conn.LocalAddr()
}

// RemoteAddr returns the remote addr of the connection.
func (s *ClientSession) RemoteAddr() net.Addr {
	return s.conn.RemoteAddr()
}
