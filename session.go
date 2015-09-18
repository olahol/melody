package melody

import (
	"errors"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
)

// A melody session.
type Session struct {
	Request *http.Request
	conn    *websocket.Conn
	output  chan *envelope
	melody  *Melody
}

func (s *Session) writeMessage(message *envelope) {
	if len(s.output) < s.melody.Config.MessageBufferSize {
		s.output <- message
	} else {
		s.melody.errorHandler(s, errors.New("Message buffer full"))
	}
}

func (s *Session) writeRaw(message *envelope) error {
	s.conn.SetWriteDeadline(time.Now().Add(s.melody.Config.WriteWait))
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

func (s *Session) writePump() {
	defer s.conn.Close()

	ticker := time.NewTicker(s.melody.Config.PingPeriod)
	defer ticker.Stop()

loop:
	for {
		select {
		case msg, ok := <-s.output:
			if !ok {
				s.close()
				break loop
			}
			if err := s.writeRaw(msg); err != nil {
				s.melody.errorHandler(s, err)
				break loop
			}
		case <-ticker.C:
			s.ping()
		}
	}
}

func (s *Session) readPump() {
	defer s.conn.Close()

	s.conn.SetReadLimit(s.melody.Config.MaxMessageSize)
	s.conn.SetReadDeadline(time.Now().Add(s.melody.Config.PongWait))

	s.conn.SetPongHandler(func(string) error {
		s.conn.SetReadDeadline(time.Now().Add(s.melody.Config.PongWait))
		return nil
	})

	for {
		t, message, err := s.conn.ReadMessage()

		if err != nil {
			s.melody.errorHandler(s, err)
			break
		}

		if t == websocket.TextMessage {
			s.melody.messageHandler(s, message)
		}

		if t == websocket.BinaryMessage {
			s.melody.messageHandlerBinary(s, message)
		}
	}
}

// Write message to session.
func (s *Session) Write(msg []byte) {
	s.writeMessage(&envelope{t: websocket.TextMessage, msg: msg})
}

// Write binary message to session.
func (s *Session) WriteBinary(msg []byte) {
	s.writeMessage(&envelope{t: websocket.BinaryMessage, msg: msg})
}

// Close session.
func (s *Session) Close() {
	s.writeMessage(&envelope{t: websocket.CloseMessage, msg: []byte{}})
}
