package melody

import (
	"math/rand"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
	"time"

	"github.com/gorilla/websocket"
)

type TestServer struct {
	m *Melody
}

func NewTestServerHandler(handler handleMessageFunc) *TestServer {
	m := New()
	m.HandleMessage(handler)
	return &TestServer{
		m: m,
	}
}

func NewTestServer() *TestServer {
	m := New()
	return &TestServer{
		m: m,
	}
}

func (s *TestServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.m.HandleRequest(w, r)
}

func NewDialer(url string) (*websocket.Conn, error) {
	dialer := &websocket.Dialer{}
	conn, _, err := dialer.Dial(strings.Replace(url, "http", "ws", 1), nil)
	return conn, err
}

func TestEcho(t *testing.T) {
	echo := NewTestServerHandler(func(session *Session, msg []byte) {
		session.Write(msg)
	})
	server := httptest.NewServer(echo)
	defer server.Close()

	fn := func(msg string) bool {
		conn, err := NewDialer(server.URL)
		defer conn.Close()

		if err != nil {
			t.Error(err)
			return false
		}

		conn.WriteMessage(websocket.TextMessage, []byte(msg))

		_, ret, err := conn.ReadMessage()

		if err != nil {
			t.Error(err)
			return false
		}

		if msg != string(ret) {
			t.Errorf("%s should equal %s", msg, string(ret))
			return false
		}

		return true
	}

	if err := quick.Check(fn, nil); err != nil {
		t.Error(err)
	}

	// we also wanna check manually for a string longer than 512 characters
	// because this was failing
	manualCheckValues := func(vs []reflect.Value, _ *rand.Rand) {
		vs[0] = reflect.ValueOf(string(make([]byte, 513)))
	}

	if err := quick.Check(fn, &quick.Config{Values: manualCheckValues}); err != nil {
		t.Error(err)
	}

}

func TestEchoBinary(t *testing.T) {
	echo := NewTestServer()
	echo.m.HandleMessageBinary(func(session *Session, msg []byte) {
		session.WriteBinary(msg)
	})
	server := httptest.NewServer(echo)
	defer server.Close()

	fn := func(msg string) bool {
		conn, err := NewDialer(server.URL)
		defer conn.Close()

		if err != nil {
			t.Error(err)
			return false
		}

		conn.WriteMessage(websocket.BinaryMessage, []byte(msg))

		_, ret, err := conn.ReadMessage()

		if err != nil {
			t.Error(err)
			return false
		}

		if msg != string(ret) {
			t.Errorf("%s should equal %s", msg, string(ret))
			return false
		}

		return true
	}

	if err := quick.Check(fn, nil); err != nil {
		t.Error(err)
	}
}

func TestHandlers(t *testing.T) {
	echo := NewTestServer()
	echo.m.HandleMessage(func(session *Session, msg []byte) {
		session.Write(msg)
	})
	server := httptest.NewServer(echo)
	defer server.Close()

	var q *Session

	echo.m.HandleConnect(func(session *Session) {
		q = session
		session.Close()
	})

	echo.m.HandleDisconnect(func(session *Session) {
		if q != session {
			t.Error("disconnecting session should be the same as connecting")
		}
	})

	NewDialer(server.URL)
}

func TestUpgrader(t *testing.T) {
	broadcast := NewTestServer()
	broadcast.m.HandleMessage(func(session *Session, msg []byte) {
		session.Write(msg)
	})
	server := httptest.NewServer(broadcast)
	defer server.Close()

	broadcast.m.Upgrader = &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return false },
	}

	broadcast.m.HandleError(func(session *Session, err error) {
		if err == nil || err.Error() != "websocket: origin not allowed" {
			t.Error("there should be a origin error")
		}
	})

	_, err := NewDialer(server.URL)

	if err == nil || err.Error() != "websocket: bad handshake" {
		t.Error("there should be a badhandshake error")
	}
}

func TestBroadcast(t *testing.T) {
	broadcast := NewTestServer()
	broadcast.m.HandleMessage(func(session *Session, msg []byte) {
		broadcast.m.Broadcast(msg)
	})
	server := httptest.NewServer(broadcast)
	defer server.Close()

	n := 10

	fn := func(msg string) bool {
		conn, _ := NewDialer(server.URL)
		defer conn.Close()

		listeners := make([]*websocket.Conn, n)
		for i := 0; i < n; i++ {
			listener, _ := NewDialer(server.URL)
			listeners[i] = listener
			defer listeners[i].Close()
		}

		conn.WriteMessage(websocket.TextMessage, []byte(msg))

		for i := 0; i < n; i++ {
			_, ret, err := listeners[i].ReadMessage()

			if err != nil {
				t.Error(err)
				return false
			}

			if msg != string(ret) {
				t.Errorf("%s should equal %s", msg, string(ret))
				return false
			}
		}

		return true
	}

	if !fn("test") {
		t.Errorf("should not be false")
	}
}

func TestBroadcastOthers(t *testing.T) {
	broadcast := NewTestServer()
	broadcast.m.HandleMessage(func(session *Session, msg []byte) {
		broadcast.m.BroadcastOthers(msg, session)
	})
	broadcast.m.Config.PongWait = time.Second
	broadcast.m.Config.PingPeriod = time.Second * 9 / 10
	server := httptest.NewServer(broadcast)
	defer server.Close()

	n := 10

	fn := func(msg string) bool {
		conn, _ := NewDialer(server.URL)
		defer conn.Close()

		listeners := make([]*websocket.Conn, n)
		for i := 0; i < n; i++ {
			listener, _ := NewDialer(server.URL)
			listeners[i] = listener
			defer listeners[i].Close()
		}

		conn.WriteMessage(websocket.TextMessage, []byte(msg))

		for i := 0; i < n; i++ {
			_, ret, err := listeners[i].ReadMessage()

			if err != nil {
				t.Error(err)
				return false
			}

			if msg != string(ret) {
				t.Errorf("%s should equal %s", msg, string(ret))
				return false
			}
		}

		return true
	}

	if !fn("test") {
		t.Errorf("should not be false")
	}
}

func TestPingPong(t *testing.T) {
	noecho := NewTestServer()
	noecho.m.Config.PongWait = time.Second
	noecho.m.Config.PingPeriod = time.Second * 9 / 10
	server := httptest.NewServer(noecho)
	defer server.Close()

	conn, err := NewDialer(server.URL)
	conn.SetPingHandler(func(string) error {
		return nil
	})
	defer conn.Close()

	if err != nil {
		t.Error(err)
	}

	conn.WriteMessage(websocket.TextMessage, []byte("test"))

	_, _, err = conn.ReadMessage()

	if err == nil {
		t.Error("there should be an error")
	}
}

func TestBroadcastFilter(t *testing.T) {
	broadcast := NewTestServer()
	broadcast.m.HandleMessage(func(session *Session, msg []byte) {
		broadcast.m.BroadcastFilter(msg, func(q *Session) bool {
			return session == q
		})
	})
	server := httptest.NewServer(broadcast)
	defer server.Close()

	fn := func(msg string) bool {
		conn, err := NewDialer(server.URL)
		defer conn.Close()

		if err != nil {
			t.Error(err)
			return false
		}

		conn.WriteMessage(websocket.TextMessage, []byte(msg))

		_, ret, err := conn.ReadMessage()

		if err != nil {
			t.Error(err)
			return false
		}

		if msg != string(ret) {
			t.Errorf("%s should equal %s", msg, string(ret))
			return false
		}

		return true
	}

	if !fn("test") {
		t.Errorf("should not be false")
	}
}
