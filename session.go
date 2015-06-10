package melody

import (
	"github.com/gorilla/websocket"
	"net/http"
	"time"
)

// A melody session.
type Session struct {
	Request *http.Request
	conn    *websocket.Conn
	output  chan *envelope
	config  *Config
}

func newSession(config *Config, conn *websocket.Conn, req *http.Request) *Session {
	return &Session{
		Request: req,
		conn:    conn,
		output:  make(chan *envelope, config.MessageBufferSize),
		config:  config,
	}
}

func (s *Session) writeMessage(message *envelope) {
	s.output <- message
}

func (s *Session) writeRaw(message *envelope) error {
	s.conn.SetWriteDeadline(time.Now().Add(s.config.WriteWait))
	err := s.conn.WriteMessage(message.t, message.msg)

	if err != nil {
		return err
	}

	if message.t == websocket.CloseMessage {
		err := s.conn.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *Session) close() {
	s.writeRaw(&envelope{t: websocket.CloseMessage, msg: []byte{}})
}

func (s *Session) ping() {
	s.writeMessage(&envelope{t: websocket.PingMessage, msg: []byte{}})
}

func (s *Session) writePump(errorHandler handleErrorFunc) {
	defer s.conn.Close()

	ticker := time.NewTicker(s.config.PingPeriod)
	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-s.output:
			if !ok {
				s.close()
				return
			}
			if err := s.writeRaw(msg); err != nil {
				go errorHandler(s, err)
				return
			}
		case <-ticker.C:
			s.ping()
		}
	}
}

func (s *Session) readPump(messageHandler handleMessageFunc, messageHandlerBinary handleMessageFunc, errorHandler handleErrorFunc) {
	defer s.conn.Close()

	s.conn.SetReadLimit(s.config.MaxMessageSize)
	s.conn.SetReadDeadline(time.Now().Add(s.config.PongWait))

	s.conn.SetPongHandler(func(string) error {
		s.conn.SetReadDeadline(time.Now().Add(s.config.PongWait))
		return nil
	})

	for {
		t, message, err := s.conn.ReadMessage()

		if err != nil {
			go errorHandler(s, err)
			break
		}

		if t == websocket.TextMessage {
			go messageHandler(s, message)
		}

		if t == websocket.BinaryMessage {
			go messageHandlerBinary(s, message)
		}
	}
}

// Write message to session.
func (s *Session) WriteBinary(msg []byte) {
	s.writeMessage(&envelope{t: websocket.BinaryMessage, msg: msg})
}

// Write message to session.
func (s *Session) Write(msg []byte) {
	s.writeMessage(&envelope{t: websocket.TextMessage, msg: msg})
}

// Close session.
func (s *Session) Close() {
	s.writeMessage(&envelope{t: websocket.CloseMessage, msg: []byte{}})
}
